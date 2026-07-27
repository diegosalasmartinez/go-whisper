# go-whisper

A small Go library for on-device speech-to-text: record from the microphone and
transcribe with [whisper.cpp](https://github.com/ggerganov/whisper.cpp). No audio
leaves the machine and there is no API key.

Recording and transcription are separate interfaces (`Recorder`, `Transcriber`)
so they can run in different processes — for example, capture in a foreground
app that holds the microphone permission, transcribe in a long-lived daemon that
keeps the model loaded.

The current backends shell out to `ffmpeg` and `whisper-cli`. A CGO-linked
whisper backend can be added behind the `Transcriber` interface without changing
callers.

## Requirements

- macOS (the recorder uses ffmpeg's `avfoundation` input)
- `ffmpeg` — `brew install ffmpeg`
- `whisper.cpp` — `brew install whisper-cpp` (provides `whisper-cli`)
- A ggml model, e.g. `ggml-base.bin` from
  [huggingface.co/ggerganov/whisper.cpp](https://huggingface.co/ggerganov/whisper.cpp)

## Install

```
go get github.com/diegosalasmartinez/go-whisper
```

## Usage

```go
recorder := gowhisper.NewFFmpegRecorder()
transcriber := gowhisper.NewCLITranscriber("/path/to/ggml-base.bin")

// Record until ctx is cancelled (or the recorder's cap is hit).
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

wav, err := recorder.Record(ctx)
if err != nil {
	log.Fatal(err)
}
defer os.Remove(wav)

text, err := transcriber.Transcribe(context.Background(), wav)
if err != nil {
	log.Fatal(err)
}
fmt.Println(text)
```

Both types expose their configuration as fields (binary path, audio device,
sample rate, language, threads) with sensible defaults from the constructors.

Recording stops when the context is cancelled, so an interactive caller can
start recording on one keypress and stop on the next by cancelling the context.

## Native backend (CGO)

The default `CLITranscriber` shells out to `whisper-cli`. An optional in-process
backend links whisper.cpp directly, loading the model once and reusing it across
calls. It is behind the `whisper_cgo` build tag, so the default build stays
CGO-free.

```go
tr, err := gowhisper.NewNativeTranscriber("/path/to/ggml-base.bin")
if err != nil {
	log.Fatal(err)
}
defer tr.Close()
text, err := tr.Transcribe(ctx, wav)
```

Build and test it with the tag:

```
go build -tags whisper_cgo ./...
go test  -tags whisper_cgo ./...
```

Requirements: `whisper-cpp` installed with its headers and libraries (Homebrew
provides them under `/opt/homebrew`), and CGO enabled. The `#cgo` directives in
`transcriber_native.go` point at the Homebrew paths; adjust them for other
prefixes. On Apple Silicon this requires a native arm64 Go toolchain — an
amd64 (Rosetta) toolchain cannot link the arm64 whisper libraries.

## Example

```
go run ./example --model /path/to/ggml-base.bin --seconds 5
```

Records for five seconds and prints the transcript. Use it to confirm ffmpeg and
whisper.cpp are installed and the microphone works.

## Finding the audio device

List avfoundation inputs and set `FFmpegRecorder.Device` accordingly:

```
ffmpeg -f avfoundation -list_devices true -i ""
```

The default is `:0` (first audio input).

## License

MIT. See [LICENSE](LICENSE).
