package gowhisper

import "context"

// FakeRecorder returns a fixed path, for tests that don't touch a microphone.
type FakeRecorder struct {
	Path string
	Err  error
}

func (f FakeRecorder) Record(context.Context) (string, error) {
	return f.Path, f.Err
}

// FakeTranscriber returns fixed text and records the WAV paths it was given.
type FakeTranscriber struct {
	Text string
	Err  error
	Seen []string
}

func (f *FakeTranscriber) Transcribe(_ context.Context, wavPath string) (string, error) {
	f.Seen = append(f.Seen, wavPath)
	return f.Text, f.Err
}
