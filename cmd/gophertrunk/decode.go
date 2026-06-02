package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// runDecode is the entry point for `gophertrunk decode`. It reads
// a captured .raw vocoder-frame sidecar and writes a playable WAV
// using one of the registered vocoder backends. Operators use this
// to decode frames after a session — the daemon stores raw frames
// alongside each call so post-processing can defer the
// vocoder-choice decision.
//
// Usage:
//
//	gophertrunk decode -in <path> -out <path> [-vocoder <name>]
//	cat session.raw | gophertrunk decode -vocoder imbe -out out.wav
//	gophertrunk decode -in session.raw -out out.wav -vocoder ambe2
//
// The vocoder defaults to "imbe" since P25 Phase 1 is the most
// common digital protocol on US public-safety bands. The
// available names match the keys registered in
// voice.DefaultRegistry — typically "imbe", "ambe2", and "null".
func runDecode(args []string) {
	fs := flag.NewFlagSet("decode", flag.ExitOnError)
	in := fs.String("in", "-", "raw vocoder-frame input path; '-' reads stdin")
	out := fs.String("out", "", "WAV output path (required; must be a regular file for the WAV header to be patched)")
	vocoderName := fs.String("vocoder", "imbe", "vocoder name from the registry (imbe, ambe2, null, ...)")
	listVocoders := fs.Bool("list-vocoders", false, "print the registered vocoder names and exit")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk decode — decode a captured raw vocoder-frame stream into a WAV.

USAGE:
  gophertrunk decode -in <path> -out <path> [-vocoder <name>]

EXAMPLES:
  # Decode an IMBE .raw sidecar from a P25 Phase 1 capture
  gophertrunk decode -in call.raw -out call.wav -vocoder imbe

  # Decode AMBE+2 frames piped from a fixture script
  cat dmr.raw | gophertrunk decode -vocoder ambe2 -out dmr.wav

  # List the vocoder names this binary knows about
  gophertrunk decode -list-vocoders

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("decode")

	if *listVocoders {
		fmt.Println(strings.Join(voice.DefaultRegistry.Names(), "\n"))
		return
	}

	if *out == "" {
		fs.Usage()
		rep.Fatalf(2, "-out is required")
	}

	reader, closer, err := openInput(*in)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("open input: %w", err))
	}
	defer closer()

	outFile, err := os.Create(*out)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("create output: %w", err))
	}
	defer outFile.Close()

	frames, decodeErr := voice.DecodeStream(reader, *vocoderName, outFile)
	if errors.Is(decodeErr, voice.ErrPartialFrame) {
		fmt.Fprintf(os.Stderr, "decode: input ended mid-frame after %d complete frames; WAV truncated to that point\n", frames)
		// Don't exit non-zero — the WAV is playable up to the
		// partial trailer, which is the operator-friendly outcome.
	} else if decodeErr != nil {
		rep.Fatal(1, decodeErr)
	}
	fmt.Printf("decoded %d frame(s) → %s (%d ms of audio)\n",
		frames, *out, frames*20)
}

// openInput returns a Reader for the chosen path. "-" maps to
// stdin; everything else is opened as a regular file. The closer
// is a no-op for stdin and a file-close otherwise.
func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}
