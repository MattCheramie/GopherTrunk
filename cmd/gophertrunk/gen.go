package main

import (
	"flag"
	"fmt"
	"math"
	"strconv"
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
	modulation := fs.String("modulation", "", "override the fixture modulator (P25: \"c4fm\" default, or \"lsm\"/\"cqpsk\" for the linear waveform)")
	list := fs.Bool("list", false, "list the protocols with a synthesis fixture and exit")
	out := fs.String("out", "", "output capture path (required)")
	format := fs.String("format", "f32", "capture sample format: u8 | f32")
	metaOut := fs.String("meta", "", "metadata sidecar path (default: <out stem>.metadata.json)")
	snr := fs.Float64("snr", 0, "additive white Gaussian noise target SNR in dB (0 = clean)")
	freqOffset := fs.Float64("freq-offset", 0, "residual carrier frequency offset in Hz")
	freqDrift := fs.Float64("drift", 0, "linear carrier-frequency drift in Hz per second (added to -freq-offset)")
	multipath := fs.String("multipath", "", "multipath/simulcast channel: preset \"simulcast\" or comma-separated taps \"delay:mag:deg\" (e.g. \"0:1:0,5:0.95:70\")")
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
	mpTaps, err := parseMultipath(*multipath)
	if err != nil {
		fs.Usage()
		rep.Fatal(2, err)
	}

	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol:   proto,
		Format:     sampleFormat,
		Modulation: *modulation,
		Impairments: demod.Impairments{
			Multipath:         mpTaps,
			SNRdB:             *snr,
			FreqOffsetHz:      *freqOffset,
			FreqDriftHzPerSec: *freqDrift,
			DCOffset:          complex(float32(*dc), 0),
			IQGainImbalance:   *gainImb,
			IQPhaseSkewRad:    *phaseSkew,
			Seed:              *seed,
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

// parseMultipath builds a complex channel FIR from the -multipath flag. An
// empty string is no multipath. The preset "simulcast" is the near-spectral-
// null two-path channel issue #492 reproduces with (main path + a half-symbol
// 0.95∠70° echo at 48 kHz / 10 sps). Otherwise the value is comma-separated
// taps "delay:mag:deg" (delay in samples, magnitude, phase in degrees); the
// returned FIR is sized to the largest delay with the named taps filled in.
func parseMultipath(s string) ([]complex64, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "":
		return nil, nil
	case "simulcast":
		echo := complex(float32(0.95*math.Cos(70*math.Pi/180)), float32(0.95*math.Sin(70*math.Pi/180)))
		return []complex64{complex(1, 0), 0, 0, 0, 0, echo}, nil
	}
	type tap struct {
		delay int
		val   complex64
	}
	var taps []tap
	maxDelay := 0
	for _, field := range strings.Split(s, ",") {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("multipath tap %q: want delay:mag:deg", field)
		}
		delay, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || delay < 0 {
			return nil, fmt.Errorf("multipath tap %q: bad delay", field)
		}
		mag, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("multipath tap %q: bad magnitude", field)
		}
		deg, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return nil, fmt.Errorf("multipath tap %q: bad phase", field)
		}
		rad := deg * math.Pi / 180
		taps = append(taps, tap{delay: delay, val: complex(float32(mag*math.Cos(rad)), float32(mag*math.Sin(rad)))})
		if delay > maxDelay {
			maxDelay = delay
		}
	}
	fir := make([]complex64, maxDelay+1)
	for _, t := range taps {
		fir[t.delay] = t.val
	}
	return fir, nil
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
