// Command voice-calibrate compares an in-tree Vocoder's PCM output
// against a reference WAV (DSD-FME, OP25, or similar) decoded from
// the same compressed-frame source. Wraps internal/voice/calibrate's
// Compare so operators can run a one-off check without writing a
// test.
//
// Usage:
//
//	voice-calibrate -raw <file.raw> -ref-wav <file.wav> -vocoder <name>
//
// See docs/voice-calibration.md for the full capture-and-validate
// recipe.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/calibrate"

	// Import every standard vocoder so DefaultRegistry sees them.
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

const (
	// Acceptance thresholds — match the in-tree calibrate test
	// criteria documented at internal/voice/calibrate/calibrate.go.
	rmsToleranceDb = 3.0
	xcorrFloor     = 0.85
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "voice-calibrate: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		rawPath   = flag.String("raw", "", "path to raw vocoder-frame file (.raw)")
		refPath   = flag.String("ref-wav", "", "path to reference WAV from DSD-FME / OP25 (8 kHz, 16-bit, mono PCM)")
		vocoder   = flag.String("vocoder", "", "vocoder name (one of: "+strings.Join(voice.DefaultRegistry.Names(), ", ")+")")
		listNames = flag.Bool("list-vocoders", false, "list available vocoders and exit")
		strict    = flag.Bool("strict", false, "exit non-zero when |RMSRatioDb| >= 3 dB or PeakXcorr < 0.85")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `voice-calibrate compares an in-tree Vocoder's output against a
reference WAV produced from the same raw vocoder frames.

Capture recipe: see docs/voice-calibration.md.

Usage:
  %s -raw <file.raw> -ref-wav <file.wav> -vocoder <name>

Flags:
`, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *listNames {
		for _, n := range voice.DefaultRegistry.Names() {
			fmt.Println(n)
		}
		return nil
	}

	if *rawPath == "" || *refPath == "" || *vocoder == "" {
		flag.Usage()
		return fmt.Errorf("missing required flags")
	}

	res, err := calibrate.Compare(*rawPath, *refPath, *vocoder)
	if err != nil {
		return err
	}

	fmt.Printf("vocoder           : %s\n", *vocoder)
	fmt.Printf("raw file          : %s\n", *rawPath)
	fmt.Printf("reference WAV     : %s\n", *refPath)
	fmt.Printf("in-tree samples   : %d\n", res.InTreeSampleCount)
	fmt.Printf("reference samples : %d\n", res.RefSampleCount)
	fmt.Printf("RMS ratio (dB)    : %+8.4f  (positive = in-tree is louder than reference)\n", res.RMSRatioDb)
	fmt.Printf("peak xcorr        : %8.4f  (1.0 = identical, 0 = uncorrelated)\n", res.PeakXcorr)
	fmt.Printf("best-alignment lag: %+d samples (%+.2f ms at 8 kHz)\n",
		res.LagSamples, float64(res.LagSamples)/8.0)

	pass := math.Abs(res.RMSRatioDb) < rmsToleranceDb && res.PeakXcorr > xcorrFloor
	if pass {
		fmt.Println("\nresult            : PASS (within calibration thresholds)")
	} else {
		fmt.Printf("\nresult            : FAIL (|RMSRatioDb| %.4f >= %.1f dB or PeakXcorr %.4f <= %.2f)\n",
			math.Abs(res.RMSRatioDb), rmsToleranceDb, res.PeakXcorr, xcorrFloor)
		if *strict {
			os.Exit(2)
		}
	}
	return nil
}
