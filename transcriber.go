package gowhisper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Transcriber converts a WAV file into text.
type Transcriber interface {
	Transcribe(ctx context.Context, wavPath string) (string, error)
}

// CLITranscriber shells out to the whisper.cpp command-line tool. It is the
// quick-to-run backend; a CGO-linked backend can replace it behind this same
// interface without changing callers.
type CLITranscriber struct {
	Binary   string // whisper.cpp CLI; defaults to "whisper-cli"
	Model    string // path to a ggml model; required
	Language string // language code or "auto"; defaults to "auto"
	Threads  int    // worker threads; 0 leaves the whisper.cpp default
}

func NewCLITranscriber(model string) *CLITranscriber {
	return &CLITranscriber{Binary: "whisper-cli", Model: model, Language: "auto"}
}

func (t *CLITranscriber) Transcribe(ctx context.Context, wavPath string) (string, error) {
	if t.Model == "" {
		return "", errors.New("gowhisper: model path is required")
	}
	// whisper.cpp writes the transcript to <outBase>.txt; reading that file is
	// more stable across versions than parsing stdout.
	outBase := wavPath + ".out"
	txtPath := outBase + ".txt"

	cmd := exec.CommandContext(ctx, t.binary(), t.args(wavPath, outBase)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gowhisper: whisper-cli failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	data, err := os.ReadFile(txtPath)
	if err != nil {
		return "", fmt.Errorf("gowhisper: read transcript: %w", err)
	}
	os.Remove(txtPath)
	return normalizeTranscript(string(data)), nil
}

func (t *CLITranscriber) args(wavPath, outBase string) []string {
	args := []string{"-m", t.Model, "-f", wavPath, "-nt", "-otxt", "-of", outBase}
	if t.Language != "" {
		args = append(args, "-l", t.Language)
	}
	if t.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(t.Threads))
	}
	return args
}

func (t *CLITranscriber) binary() string {
	if t.Binary != "" {
		return t.Binary
	}
	return "whisper-cli"
}

// normalizeTranscript collapses whisper.cpp's line-wrapped output into a single
// clean line of text.
func normalizeTranscript(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
