package gowhisper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Recorder captures microphone audio to a WAV file.
type Recorder interface {
	// Record captures audio until ctx is cancelled (or an internal cap is
	// reached), then returns the path to a 16 kHz mono WAV file. The caller owns
	// the file and is responsible for removing it.
	Record(ctx context.Context) (string, error)
}

// FFmpegRecorder records from a macOS avfoundation input device using ffmpeg.
type FFmpegRecorder struct {
	Binary     string // ffmpeg executable; defaults to "ffmpeg"
	Device     string // avfoundation audio device; defaults to ":0"
	SampleRate int    // output sample rate in Hz; defaults to 16000
	MaxSeconds int    // hard cap on recording length; defaults to 60
}

func NewFFmpegRecorder() *FFmpegRecorder {
	return &FFmpegRecorder{Binary: "ffmpeg", Device: ":0", SampleRate: 16000, MaxSeconds: 60}
}

func (r *FFmpegRecorder) Record(ctx context.Context) (string, error) {
	f, err := os.CreateTemp("", "gowhisper-*.wav")
	if err != nil {
		return "", fmt.Errorf("gowhisper: create temp file: %w", err)
	}
	out := f.Name()
	f.Close()

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(r.maxSeconds())*time.Second)
	defer cancel()

	cmd := exec.Command(r.binary(), r.args(out)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.Remove(out)
		return "", err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		os.Remove(out)
		return "", fmt.Errorf("gowhisper: start ffmpeg: %w", err)
	}

	// Tell ffmpeg to quit gracefully on stop so it writes the WAV trailer; a
	// killed process leaves a truncated, unreadable file.
	go func() {
		<-runCtx.Done()
		io.WriteString(stdin, "q")
		stdin.Close()
	}()

	if err := cmd.Wait(); err != nil {
		if info, statErr := os.Stat(out); statErr != nil || info.Size() == 0 {
			os.Remove(out)
			return "", fmt.Errorf("gowhisper: ffmpeg failed: %v: %s", err, strings.TrimSpace(stderr.String()))
		}
	}
	return out, nil
}

func (r *FFmpegRecorder) args(out string) []string {
	return []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "avfoundation",
		"-i", r.device(),
		"-ac", "1",
		"-ar", strconv.Itoa(r.sampleRate()),
		"-y", out,
	}
}

func (r *FFmpegRecorder) binary() string {
	if r.Binary != "" {
		return r.Binary
	}
	return "ffmpeg"
}

func (r *FFmpegRecorder) device() string {
	if r.Device != "" {
		return r.Device
	}
	return ":0"
}

func (r *FFmpegRecorder) sampleRate() int {
	if r.SampleRate > 0 {
		return r.SampleRate
	}
	return 16000
}

func (r *FFmpegRecorder) maxSeconds() int {
	if r.MaxSeconds > 0 {
		return r.MaxSeconds
	}
	return 60
}
