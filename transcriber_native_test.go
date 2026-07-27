//go:build whisper_cgo

package gowhisper

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeTranscriberIntegration exercises the real CGO path end-to-end: it
// synthesizes speech with macOS `say`, then transcribes it. Skipped when the
// model or `say` is unavailable.
func TestNativeTranscriberIntegration(t *testing.T) {
	model := filepath.Join(os.Getenv("HOME"), ".whatui", "ggml-base.bin")
	if _, err := os.Stat(model); err != nil {
		t.Skip("model not found at ~/.whatui/ggml-base.bin")
	}

	wav := filepath.Join(t.TempDir(), "speech.wav")
	if err := exec.Command("say", "-o", wav, "--data-format=LEI16@16000", "the quick brown fox").Run(); err != nil {
		t.Skipf("say unavailable: %v", err)
	}

	tr, err := NewNativeTranscriber(model)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	tr.Language = "en"

	text, err := tr.Transcribe(context.Background(), wav)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("transcript: %q", text)
	if !strings.Contains(strings.ToLower(text), "fox") {
		t.Errorf("expected transcript to contain 'fox', got %q", text)
	}
}
