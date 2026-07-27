//go:build whisper_cgo

package gowhisper

/*
#cgo CFLAGS: -I/opt/homebrew/include
#cgo LDFLAGS: -L/opt/homebrew/lib -lwhisper -lggml -lggml-base
#include <stdlib.h>
#include <whisper.h>
#include <ggml-backend.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"sync"
	"unsafe"
)

// NativeTranscriber transcribes in-process through CGO bindings to whisper.cpp.
// The model is loaded once at construction and reused across calls, so repeated
// transcriptions avoid the per-call model load of the CLI backend. Build with
// the "whisper_cgo" tag and link against an installed whisper.cpp.
type NativeTranscriber struct {
	Language string

	mu  sync.Mutex
	ctx *C.struct_whisper_context
}

// NewNativeTranscriber loads the model at modelPath. Call Close to free it.
func NewNativeTranscriber(modelPath string) (*NativeTranscriber, error) {
	// Register the available compute backends (CPU, Metal) before init; without
	// this the device registry is empty and model loading aborts.
	C.ggml_backend_load_all()

	cModel := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cModel))

	params := C.whisper_context_default_params()
	ctx := C.whisper_init_from_file_with_params(cModel, params)
	if ctx == nil {
		return nil, errors.New("gowhisper: failed to load whisper model")
	}
	return &NativeTranscriber{Language: "auto", ctx: ctx}, nil
}

// Close releases the model. The transcriber cannot be used afterward.
func (t *NativeTranscriber) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ctx != nil {
		C.whisper_free(t.ctx)
		t.ctx = nil
	}
}

func (t *NativeTranscriber) Transcribe(_ context.Context, wavPath string) (string, error) {
	samples, err := decodeWAV(wavPath)
	if err != nil {
		return "", err
	}
	if len(samples) == 0 {
		return "", nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ctx == nil {
		return "", errors.New("gowhisper: transcriber is closed")
	}

	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)
	params.print_progress = C.bool(false)
	params.print_realtime = C.bool(false)
	params.print_timestamps = C.bool(false)

	lang := t.Language
	if lang == "" {
		lang = "auto"
	}
	cLang := C.CString(lang)
	defer C.free(unsafe.Pointer(cLang))
	params.language = cLang

	if C.whisper_full(t.ctx, params, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples))) != 0 {
		return "", errors.New("gowhisper: whisper_full failed")
	}

	var b strings.Builder
	for i := 0; i < int(C.whisper_full_n_segments(t.ctx)); i++ {
		b.WriteString(C.GoString(C.whisper_full_get_segment_text(t.ctx, C.int(i))))
	}
	return normalizeTranscript(b.String()), nil
}
