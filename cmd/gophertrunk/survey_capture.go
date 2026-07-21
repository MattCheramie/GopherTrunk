package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/rfscope"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// surveyCaptureParams drive `gophertrunk hunt -survey-capture`: select one row
// from a survey inventory, record raw IQ of it, and hand the capture to the
// offline workbench (SigLab) or the crypto-research toolkit (CryptoLab).
type surveyCaptureParams struct {
	selector     string // a frequency in MHz, or "#INDEX" into the inventory
	from         string // survey JSON file (a prior -survey run's output)
	to           string // "siglab" | "cryptolab"
	serial       string
	seconds      float64
	sampleRateHz float64
	gain         int
	ppm          int
	out          string // capture path (default: survey-<MHz>.cfile)
	bundle       string // optional GopherTrunk Bundle (.gtb.tar.gz) to package into
}

// runSurveyCapture records the selected signal and routes the capture. It is the
// list-driven bridge from the full-spectrum survey to deeper analysis: pick a
// row, get a `.cfile` + metadata sidecar, and a ready next step in SigLab or
// CryptoLab.
func runSurveyCapture(rep *diag.Reporter, p surveyCaptureParams) {
	to := strings.ToLower(strings.TrimSpace(p.to))
	if to == "" {
		to = "siglab"
	}
	if to != "siglab" && to != "cryptolab" {
		rep.Fatalf(2, "-to must be siglab or cryptolab (got %q)", p.to)
	}
	if p.from == "" {
		rep.Fatalf(2, "-survey-capture needs -from <survey.json> (a prior `hunt -survey … -out-format json` run)")
	}
	sv, err := loadSurveyJSON(p.from)
	if err != nil {
		rep.Fatal(1, err)
	}
	sig, err := selectSurveySignal(sv, p.selector)
	if err != nil {
		rep.Fatal(2, err)
	}

	rate := p.sampleRateHz
	if rate <= 0 {
		rate = 2_400_000
	}
	truncated := sig.OccupiedBwHz > uint32(rate)

	out := p.out
	if out == "" {
		out = fmt.Sprintf("survey-%.4fMHz.cfile", float64(sig.FreqHz)/1e6)
	}

	dev, info, err := openCaptureDevice(p.serial)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()

	fmt.Printf("survey-capture: %s @ %.4f MHz", signalLabel(sig), float64(sig.FreqHz)/1e6)
	if sig.Service != "" {
		fmt.Printf(" (%s)", sig.Service)
	}
	fmt.Printf(" — recording %.0fs at %g MS/s on %s[%s] → %s\n",
		p.seconds, rate/1e6, info.Driver, info.Serial, out)
	if truncated {
		fmt.Fprintf(os.Stderr, "survey-capture: note — the signal is ~%.1f MHz wide but the capture is %g MHz; "+
			"this is a slice for analysis, not a full-bandwidth demod (raise -sample-rate to widen, device permitting).\n",
			float64(sig.OccupiedBwHz)/1e6, rate/1e6)
	}

	written, err := captureSignalToFile(dev, sig.FreqHz, uint32(rate), p.gain, p.ppm, out, p.seconds)
	if err != nil && !errors.Is(err, context.Canceled) {
		rep.Fatal(1, fmt.Errorf("capture: %w (wrote %d samples)", err, written))
	}

	metaPath := strings.TrimSuffix(out, ext(out)) + ".metadata.json"
	meta := &siglab.Metadata{
		Protocol:     surveyCaptureProtocol(sig),
		Source:       fmt.Sprintf("survey-capture %s @ %.4f MHz", signalLabel(sig), float64(sig.FreqHz)/1e6),
		SampleRateHz: rate,
		CenterFreqHz: sig.FreqHz,
		Format:       siglab.FormatF32.String(),
	}
	if err := siglab.WriteMetadata(metaPath, meta); err != nil {
		rep.Fatal(1, fmt.Errorf("write metadata: %w", err))
	}
	fmt.Printf("survey-capture: wrote %d samples → %s  (metadata → %s)\n", written, out, metaPath)

	var framesPath string
	switch to {
	case "siglab":
		routeToSigLab(out, rate)
	case "cryptolab":
		framesPath = routeToCryptoLab(rep, out, sig.FreqHz, rate)
	}

	if p.bundle != "" {
		if err := packSurveyCaptureBundle(surveyBundleInputs{
			bundlePath:  p.bundle,
			to:          to,
			capturePath: out,
			meta:        meta,
			sampleRate:  rate,
			protocol:    meta.Protocol,
			framesPath:  framesPath,
		}); err != nil {
			rep.Fatal(1, fmt.Errorf("write bundle: %w", err))
		}
		fmt.Printf("survey-capture: packaged → %s\n", p.bundle)
	}
}

// surveyBundleInputs carries what packSurveyCaptureBundle needs to assemble a
// bundle from a survey-capture run.
type surveyBundleInputs struct {
	bundlePath  string
	to          string
	capturePath string
	meta        *siglab.Metadata
	sampleRate  float64
	protocol    string
	framesPath  string // cryptolab frames.jsonl, when -to cryptolab
}

// packSurveyCaptureBundle packages a survey-capture into a GopherTrunk Bundle:
// the raw IQ + metadata + a carved narrowband slice, plus the CryptoLab frames
// when the capture was routed there. Intent follows the routing target (crypto
// vs cc-map) so the slice length matches the workflow.
func packSurveyCaptureBundle(in surveyBundleInputs) error {
	intent := gtbundle.IntentCCMap
	if in.to == "cryptolab" {
		intent = gtbundle.IntentCrypto
	}
	name := strings.TrimSuffix(baseName(in.bundlePath), gtbundle.Ext)
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".tar.gz"), ".gtb")

	// Reuse the capture-bundle assembler for IQ + meta + slice.
	if err := writeCaptureBundle(captureBundleParams{
		bundlePath:   in.bundlePath,
		name:         name,
		intent:       intent,
		capturePath:  in.capturePath,
		format:       siglab.FormatF32,
		sampleRateHz: in.sampleRate,
		centerHz:     in.meta.CenterFreqHz,
		tuneHz:       in.meta.TuneHz,
		protocol:     in.protocol,
		source:       in.meta.Source,
		meta:         in.meta,
	}); err != nil {
		return err
	}
	// Fold the CryptoLab frames in as a second step (bundle add semantics).
	if in.framesPath != "" {
		if err := bundleAddFile(in.bundlePath, gtbundle.RoleCryptolabFrames, in.framesPath, "cryptolab"); err != nil {
			return err
		}
	}
	return nil
}

// captureSignalToFile tunes the device to a signal and records seconds of raw
// f32 IQ, reusing the shared capture packer. Split out so it is exercisable
// against a mock device.
func captureSignalToFile(dev sdr.Device, freqHz, rateHz uint32, gain, ppm int, out string, seconds float64) (int64, error) {
	if err := dev.SetSampleRate(rateHz); err != nil {
		return 0, fmt.Errorf("set sample rate: %w", err)
	}
	if err := dev.SetCenterFreq(freqHz); err != nil {
		return 0, fmt.Errorf("set centre frequency: %w", err)
	}
	if ppm != 0 {
		if err := dev.SetPPM(ppm); err != nil {
			return 0, fmt.Errorf("set ppm: %w", err)
		}
	}
	if err := dev.SetGain(gain); err != nil {
		return 0, fmt.Errorf("set gain: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	stream, err := dev.StreamIQ(ctx)
	if err != nil {
		return 0, fmt.Errorf("start IQ stream: %w", err)
	}
	written, _, err := captureToFile(ctx, out, siglab.FormatF32, stream, rateHz, seconds, nil)
	return written, err
}

// routeToSigLab runs a quick in-process identify on the capture and prints the
// command to open it in the full SigLab workbench.
func routeToSigLab(path string, rateHz float64) {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		if idr, ierr := siglab.IdentifyReader(f, path, siglab.IdentifyConfig{
			SampleRateHz: rateHz, Format: siglab.FormatF32, AutoTune: true,
		}); ierr == nil {
			if idr.Inconclusive || idr.Winner == "" {
				fmt.Println("siglab: no protocol locked (inconclusive) — open it in the workbench to dig in")
			} else {
				fmt.Printf("siglab: identifies as %s (confidence %.2f)\n", idr.Winner, idr.Confidence)
			}
		}
	}
	fmt.Printf("siglab: analyze it →  gophertrunk siglab -in %s -sample-rate %g -format f32 -auto-tune\n", path, rateHz)
	fmt.Println("siglab: or open the browser console →  gophertrunk siglab serve -open")
}

// routeToCryptoLab runs rfscope over the capture to emit a cryptolab `ks` frames
// file from any unidentified payloads, then prints the cryptolab next step. It
// returns the frames-file path so a caller can fold it into a bundle.
func routeToCryptoLab(rep *diag.Reporter, path string, centerHz uint32, rateHz float64) string {
	framesOut, n, err := emitCryptolabFramesFromFile(path, centerHz, rateHz)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("rfscope frames: %w", err))
	}
	fmt.Printf("cryptolab: wrote %d frame(s) → %s\n", n, framesOut)
	fmt.Printf("cryptolab: triage them →  gophertrunk cryptolab classify auto -in %s\n", framesOut)
	fmt.Println("cryptolab: (needs a -tags cryptolab build; see docs/cryptolab.md)")
	return framesOut
}

// defaultRFScopeSegConfig is the carrier-discovery configuration the capture
// routes use when running rfscope over a recorded signal — the same defaults as
// the rfscope CLI.
func defaultRFScopeSegConfig() rfscope.SegmentConfig {
	return rfscope.SegmentConfig{
		FFTSize:             4096,
		ChannelTargetRateHz: 50_000,
		PeakThresholdDb:     10,
		MinSpacingHz:        12_500,
	}
}

// emitCryptolabFramesFromFile runs an rfscope pass over a recorded capture and
// writes the recovered payloads as a cryptolab `ks` frames file next to it.
// Returns the frames path and the number of frames written. Shared by the CLI
// route and the daemon cockpit so both behave identically.
func emitCryptolabFramesFromFile(path string, centerHz uint32, rateHz float64) (string, int, error) {
	src, err := rfscope.OpenFile(path, siglab.FormatF32, centerHz, rateHz)
	if err != nil {
		return "", 0, fmt.Errorf("rfscope open: %w", err)
	}
	defer src.Close()
	ctx := context.Background()
	scene, in, err := rfscope.Segment(ctx, src, defaultRFScopeSegConfig())
	if err != nil {
		return "", 0, err
	}
	scene.Source = path
	if err := rfscope.Run(ctx, scene, in); err != nil {
		return "", 0, err
	}
	framesOut := path + ".frames.jsonl"
	f, err := os.Create(framesOut)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", framesOut, err)
	}
	n, werr := rfscope.EmitFrames(scene, f)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return "", n, werr
	}
	return framesOut, n, nil
}

// identifySummary runs a quick SigLab identify over an in-memory f32 capture and
// returns a short label ("p25 (0.82)" / "inconclusive"), or "" on error.
func identifySummary(buf []byte, rateHz float64) string {
	idr, err := siglab.IdentifyReader(bytes.NewReader(buf), "survey-capture", siglab.IdentifyConfig{
		SampleRateHz: rateHz, Format: siglab.FormatF32, AutoTune: true,
	})
	if err != nil || idr == nil {
		return ""
	}
	if idr.Inconclusive || idr.Winner == "" {
		return "inconclusive"
	}
	return fmt.Sprintf("%s (%.2f)", idr.Winner, idr.Confidence)
}

// loadSurveyJSON reads a SignalSurvey from a `hunt -survey` JSON export.
func loadSurveyJSON(path string) (*hunt.SignalSurvey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read survey %s: %w", path, err)
	}
	var sv hunt.SignalSurvey
	if err := json.Unmarshal(raw, &sv); err != nil {
		return nil, fmt.Errorf("parse survey %s: %w", path, err)
	}
	if len(sv.Signals) == 0 {
		return nil, fmt.Errorf("survey %s has no signals", path)
	}
	return &sv, nil
}

// selectSurveySignal resolves a selector to one inventory row. The selector is
// either "#INDEX" (0-based, into the inventory order) or a frequency in MHz,
// matched to the nearest signal within a tolerance that scales with the signal's
// own bandwidth (so a wide signal a little off-centre still matches).
func selectSurveySignal(sv *hunt.SignalSurvey, selector string) (hunt.DetectedSignal, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return hunt.DetectedSignal{}, errors.New("-survey-capture needs a frequency in MHz or #INDEX")
	}
	if strings.HasPrefix(selector, "#") {
		idx, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
		if err != nil || idx < 0 || idx >= len(sv.Signals) {
			return hunt.DetectedSignal{}, fmt.Errorf("index %q out of range [0,%d)", selector, len(sv.Signals))
		}
		return sv.Signals[idx], nil
	}
	mhz, err := strconv.ParseFloat(selector, 64)
	if err != nil {
		return hunt.DetectedSignal{}, fmt.Errorf("selector %q is not a frequency in MHz or #INDEX", selector)
	}
	target := uint32(mhz*1e6 + 0.5)
	best := -1
	var bestDiff uint32 = ^uint32(0)
	for i, s := range sv.Signals {
		if d := absU32(s.FreqHz, target); d < bestDiff {
			bestDiff, best = d, i
		}
	}
	tol := uint32(100_000)
	if half := sv.Signals[best].OccupiedBwHz / 2; half > tol {
		tol = half
	}
	if best < 0 || bestDiff > tol {
		return hunt.DetectedSignal{}, fmt.Errorf("no signal within %.3f MHz of %.4f MHz", float64(tol)/1e6, mhz)
	}
	return sv.Signals[best], nil
}

// absU32 is |a-b| for unsigned frequencies.
func absU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// signalLabel is the human name for a captured row, falling back to the class.
func signalLabel(s hunt.DetectedSignal) string {
	if s.Name != "" {
		return s.Name
	}
	return string(s.Class)
}

// surveyCaptureProtocol returns the decoded protocol for the metadata sidecar
// (so `test`/`replay` can use it), or empty for an undecoded/wideband row.
func surveyCaptureProtocol(s hunt.DetectedSignal) string {
	if s.Trunking != nil {
		return s.Trunking.Protocol
	}
	return ""
}
