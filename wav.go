package gowhisper

import (
	"encoding/binary"
	"fmt"
	"os"
)

// decodeWAV reads a 16-bit PCM WAV file and returns mono samples as float32 in
// [-1, 1], the form whisper.cpp expects. Multi-channel input is downmixed by
// averaging channels.
func decodeWAV(path string) ([]float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("gowhisper: not a WAV file")
	}

	var channels int
	var bitsPerSample int
	var pcm []byte

	for pos := 12; pos+8 <= len(data); {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		body := pos + 8
		if body+size > len(data) {
			size = len(data) - body
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return nil, fmt.Errorf("gowhisper: short fmt chunk")
			}
			channels = int(binary.LittleEndian.Uint16(data[body+2 : body+4]))
			bitsPerSample = int(binary.LittleEndian.Uint16(data[body+14 : body+16]))
		case "data":
			pcm = data[body : body+size]
		}
		pos = body + size
		if size%2 == 1 {
			pos++ // chunks are word-aligned
		}
	}

	if bitsPerSample != 16 {
		return nil, fmt.Errorf("gowhisper: expected 16-bit PCM, got %d-bit", bitsPerSample)
	}
	if channels < 1 {
		channels = 1
	}
	if pcm == nil {
		return nil, fmt.Errorf("gowhisper: no data chunk")
	}

	frames := len(pcm) / 2 / channels
	out := make([]float32, frames)
	for i := 0; i < frames; i++ {
		var sum int
		for c := 0; c < channels; c++ {
			off := (i*channels + c) * 2
			sum += int(int16(binary.LittleEndian.Uint16(pcm[off : off+2])))
		}
		out[i] = float32(sum) / float32(channels) / 32768
	}
	return out, nil
}
