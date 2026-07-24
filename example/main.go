// Command example records a few seconds of microphone audio and prints the
// transcript. It is a manual check that ffmpeg and whisper.cpp are wired up.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	gowhisper "github.com/diegosalasmartinez/go-whisper"
)

func main() {
	model := flag.String("model", "", "path to a whisper.cpp ggml model")
	seconds := flag.Int("seconds", 5, "recording duration in seconds")
	flag.Parse()

	if *model == "" {
		log.Fatal("--model is required")
	}

	recorder := gowhisper.NewFFmpegRecorder()
	transcriber := gowhisper.NewCLITranscriber(*model)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*seconds)*time.Second)
	defer cancel()

	fmt.Printf("recording for %ds...\n", *seconds)
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
}
