package gowhisper

import (
	"context"
	"slices"
	"testing"
)

func TestNormalizeTranscript(t *testing.T) {
	cases := map[string]string{
		"  hello   world \n": "hello world",
		"line one\nline two": "line one line two",
		"":                   "",
		"   ":                "",
	}
	for in, want := range cases {
		if got := normalizeTranscript(in); got != want {
			t.Errorf("normalizeTranscript(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCLITranscriberArgs(t *testing.T) {
	tr := NewCLITranscriber("/models/base.bin")
	tr.Language = "es"
	tr.Threads = 4
	got := tr.args("/tmp/a.wav", "/tmp/a.wav.out")
	want := []string{
		"-m", "/models/base.bin", "-f", "/tmp/a.wav",
		"-nt", "-otxt", "-of", "/tmp/a.wav.out", "-l", "es", "-t", "4",
	}
	if !slices.Equal(got, want) {
		t.Errorf("args = %v, want %v", got, want)
	}
}

func TestCLITranscriberRequiresModel(t *testing.T) {
	tr := &CLITranscriber{}
	if _, err := tr.Transcribe(context.Background(), "/tmp/a.wav"); err == nil {
		t.Fatal("expected an error when no model is set")
	}
}

func TestFFmpegRecorderArgs(t *testing.T) {
	r := NewFFmpegRecorder()
	got := r.args("/tmp/out.wav")
	want := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "avfoundation", "-i", ":0",
		"-ac", "1", "-ar", "16000", "-y", "/tmp/out.wav",
	}
	if !slices.Equal(got, want) {
		t.Errorf("args = %v, want %v", got, want)
	}
}

func TestFFmpegRecorderZeroValueDefaults(t *testing.T) {
	r := &FFmpegRecorder{}
	if r.binary() != "ffmpeg" || r.device() != ":0" || r.sampleRate() != 16000 || r.maxSeconds() != 60 {
		t.Errorf("zero-value defaults wrong: %q %q %d %d", r.binary(), r.device(), r.sampleRate(), r.maxSeconds())
	}
}
