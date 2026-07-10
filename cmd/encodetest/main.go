// Command encodetest encodes a 16-bit PCM mono WAV file to MP3 using the same
// internal/voice/mp3 path the broadcast backends use, writing the result to a
// file. It exists to reproduce and verify call-audio encoding issues offline —
// e.g. piping its output through ffmpeg exactly as Rdio Scanner does:
//
//	go run ./cmd/encodetest in.wav out.mp3
//	ffmpeg -i out.mp3 -f null -        # must succeed
//	cat out.mp3 | ffmpeg -i - -f null - # must succeed (stdin, like Rdio Scanner)
package main

import (
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mp3"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <in.wav> <out.mp3>\n", os.Args[0])
		os.Exit(2)
	}
	data, rate, err := mp3.EncodeWAVFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[2], data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}
	fmt.Printf("sample rate=%d bitrate=%d bytes=%d\n", rate, mp3.BitrateFor(rate), len(data))
}
