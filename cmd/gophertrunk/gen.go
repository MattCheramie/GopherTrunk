package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runGen is the entry point for `gophertrunk gen`. It synthesizes a
// known-good (optionally impaired) IQ capture for a protocol plus a metadata
// sidecar describing how to decode and grade it — the synthesize half of the
// synthesize→decode→grade loop the `test` harness closes. Synthesis reuses
// the production modulators (internal/dsp/demod) and the same locking symbol
// streams the integration tests drive, so a generated capture is a faithful
// stand-in for real-air traffic for everything above the RF front end.
func runGen(args []string) {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	protocolFlag := fs.String("protocol", "", "protocol to synthesize (required; see -list)")
	list := fs.Bool("list", false, "list the protocols with a synthesis fixture and exit")
	out := fs.String("out", "", "output capture path (required)")
	format := fs.String("format", "f32", "capture sample format: u8 | f32")
	metaOut := fs.String("meta", "", "metadata sidecar path (default: <out stem>.metadata.json)")
	snr := fs.Float64("snr", 0, "additive white Gaussian noise target SNR in dB (0 = clean)")
	freqOffset := fs.Float64("freq-offset", 0, "residual carrier frequency offset in Hz")
	dc := fs.Float64("dc", 0, "DC-offset magnitude relative to unit-scale output (0.1 ≈ -20 dBFS)")
	gainImb := fs.Float64("iq-gain", 0, "I/Q gain imbalance (Q gain relative to I; 0 = none)")
	phaseSkew := fs.Float64("iq-phase", 0, "I/Q quadrature phase skew in radians (0 = none)")
	seed := fs.Int64("seed", 1, "RNG seed for reproducible AWGN")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk gen — synthesize a test IQ capture + metadata for a protocol.

USAGE:
  gophertrunk gen -protocol <p> -out <path> [-format u8|f32] [impairment flags]

EXAMPLES:
  # Clean P25 Phase 1 capture + sidecar metadata
  gophertrunk gen -protocol p25p1 -out p25.cfile -format f32

  # DMR Tier III capture degraded with 20 dB SNR and a 300 Hz carrier offset
  gophertrunk gen -protocol dmr -out dmr.cfile -snr 20 -freq-offset 300

  # List the protocols a fixture exists for
  gophertrunk gen -list

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("gen")

	if *list {
		names := make([]string, 0)
		for _, p := range siglab.Fixtures() {
			names = append(names, p.String())
		}
		fmt.Println(strings.Join(names, "\n"))
		return
	}
	if *protocolFlag == "" {
		fs.Usage()
		rep.Fatalf(2, "-protocol is required")
	}
	if *out == "" {
		fs.Usage()
		rep.Fatalf(2, "-out is required")
	}
	proto, err := siglab.ParseProtocolCLI(*protocolFlag)
	if err != nil {
		rep.Fatal(2, err)
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}

	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: proto,
		Format:   sampleFormat,
		Impairments: demod.Impairments{
			SNRdB:           *snr,
			FreqOffsetHz:    *freqOffset,
			DCOffset:        complex(float32(*dc), 0),
			IQGainImbalance: *gainImb,
			IQPhaseSkewRad:  *phaseSkew,
			Seed:            *seed,
		},
	})
	if err != nil {
		rep.Fatal(2, err)
	}

	if err := siglab.WriteCapture(*out, iq, sampleFormat); err != nil {
		rep.Fatal(1, fmt.Errorf("write capture: %w", err))
	}
	metaPath := *metaOut
	if metaPath == "" {
		metaPath = strings.TrimSuffix(*out, ext(*out)) + ".metadata.json"
	}
	if err := siglab.WriteMetadata(metaPath, meta); err != nil {
		rep.Fatal(1, fmt.Errorf("write metadata: %w", err))
	}
	fmt.Printf("gen: wrote %d samples → %s  (metadata → %s)\n", len(iq), *out, metaPath)
}

// ext returns the file extension including the dot, or "" if none.
func ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}
