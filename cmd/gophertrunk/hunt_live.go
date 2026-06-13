package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// huntLiveParams are the resolved inputs for a live (on-air) hunt.
type huntLiveParams struct {
	serial           string
	survey           bool // classify+decode every carrier, not just trunking CCs
	classifyOnly     bool
	surveyDeep       bool   // hand analog-classed narrowband carriers to siglab identify
	surveyNDJSON     string // append each classified carrier here (crash-safe, resumable)
	resume           bool   // skip carriers already recorded in surveyNDJSON
	autoGain         bool   // after the run, sweep gains per locked CC and recommend one
	autoGainSet      string // comma-separated gains in dB to try (empty ⇒ default spread)
	outDir           string // explicit -out dir, for the gain-recommendation sidecar
	surveyAudioDir   string
	maxDwellSeconds  float64
	identifyMinConf  float64
	classSNRGate     float64
	classDigitalProm float64
	classAMCV        float64
	bands            []string // "low:high" in MHz
	candidatesMHz    string   // comma-separated MHz
	noSweep          bool
	sampleRateHz     float64
	protocol         trunking.Protocol
	fftSize          int
	sweepDwell       time.Duration
	peakThresholdDb  float64
	minSpacingHz     uint32
	dwellSeconds     float64
	autoTune         bool
	gain             int
	ppm              int
	name             string
	state            string
	county           string
	location         string
	minConfidence    float64
}

// runHuntLive opens the SDR directly (standalone, not through the daemon pool),
// sweeps the requested band(s) — or probes an explicit candidate list — and
// returns the discovered system plus per-candidate reports for the shared
// export tail. The daemon-integrated live hunt (with spare-SDR-else-borrow
// acquisition and a REST/TUI/web cockpit) is a later phase; this is the
// one-shot CLI path.
func runHuntLive(rep *diag.Reporter, p huntLiveParams) (*hunt.DiscoveredSystem, *hunt.SignalSurvey, []hunt.CaptureReport) {
	candidates := parseFreqListMHz(rep, p.candidatesMHz)
	bands := parseBandsMHz(rep, p.bands)
	if len(candidates) == 0 && len(bands) == 0 {
		rep.Fatalf(2, "live hunt needs -band low:high (to sweep) or -candidates f,f (to probe)")
	}
	if p.noSweep && len(candidates) == 0 {
		rep.Fatalf(2, "-no-sweep requires -candidates")
	}
	if p.noSweep {
		bands = nil // probe only the listed candidates
	}

	dev, info, err := openCaptureDevice(p.serial)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()
	if err := dev.SetSampleRate(uint32(p.sampleRateHz)); err != nil {
		rep.Fatal(1, fmt.Errorf("set sample rate: %w", err))
	}
	if p.ppm != 0 {
		if err := dev.SetPPM(p.ppm); err != nil {
			rep.Fatal(1, fmt.Errorf("set ppm: %w", err))
		}
	}
	if err := dev.SetGain(p.gain); err != nil {
		rep.Fatal(1, fmt.Errorf("set gain: %w", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	src, err := newDeviceIQSource(ctx, dev, uint32(p.sampleRateHz))
	if err != nil {
		rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
	}

	mode := "hunt"
	if p.survey {
		mode = "survey"
	}

	// Survey NDJSON stream + resume: append each classified carrier to disk
	// (fsync per record) and, on -resume, skip carriers already recorded.
	var onSignal func(hunt.DetectedSignal)
	var skipFreqs map[uint32]struct{}
	if p.surveyNDJSON != "" {
		if p.resume {
			skipFreqs, err = hunt.LoadSurveyedFreqs(p.surveyNDJSON)
			if err != nil {
				rep.Fatal(1, fmt.Errorf("resume %s: %w", p.surveyNDJSON, err))
			}
			if len(skipFreqs) > 0 {
				fmt.Fprintf(os.Stderr, "%s: resuming — skipping %d already-surveyed carrier(s)\n", mode, len(skipFreqs))
			}
		}
		sink, serr := hunt.OpenNDJSONSink(p.surveyNDJSON)
		if serr != nil {
			rep.Fatal(1, fmt.Errorf("open survey ndjson: %w", serr))
		}
		defer sink.Close()
		onSignal = func(ds hunt.DetectedSignal) {
			if werr := sink.Write(ds); werr != nil {
				fmt.Fprintf(os.Stderr, "%s: ndjson write: %v\n", mode, werr)
			}
		}
	}
	if len(bands) > 0 {
		fmt.Fprintf(os.Stderr, "%s: live sweep on %s[%s] @ %g MS/s across %d band(s)…\n",
			mode, info.Driver, info.Serial, p.sampleRateHz/1e6, len(bands))
	} else {
		fmt.Fprintf(os.Stderr, "%s: live probe on %s[%s] of %d candidate(s)…\n",
			mode, info.Driver, info.Serial, len(candidates))
	}

	opts := hunt.LiveHuntOptions{
		Source:                src,
		Bands:                 bands,
		Candidates:            candidates,
		Protocol:              p.protocol,
		FFTSize:               p.fftSize,
		SweepDwell:            p.sweepDwell,
		PeakOpts:              hunt.PeakOptions{ThresholdDb: float32(p.peakThresholdDb), MinSpacingHz: p.minSpacingHz},
		DwellSeconds:          p.dwellSeconds,
		MaxDwellSeconds:       p.maxDwellSeconds,
		MinConfidence:         p.minConfidence,
		AutoTune:              p.autoTune,
		Name:                  p.name,
		State:                 p.state,
		County:                p.county,
		Location:              p.location,
		ClassifyOnly:          p.classifyOnly,
		SurveyDeep:            p.surveyDeep,
		SurveyAudioDir:        p.surveyAudioDir,
		SkipFreqs:             skipFreqs,
		OnSignal:              onSignal,
		IdentifyMinConfidence: p.identifyMinConf,
		ClassifyConfig: survey.ClassifyConfig{
			SNRGateDb:         p.classSNRGate,
			DigitalProminence: p.classDigitalProm,
			AMEnvelopeCV:      p.classAMCV,
		},
		OnProgress: func(pr hunt.LiveHuntProgress) {
			switch pr.Phase {
			case hunt.PhaseSweeping:
				fmt.Fprintf(os.Stderr, "%s: sweeping %.4f MHz — %s\n", mode, float64(pr.CenterHz)/1e6, pr.Detail)
			case hunt.PhaseIdentifying:
				fmt.Fprintf(os.Stderr, "%s: probing candidate %d/%d @ %s\n", mode, pr.CandidateN, pr.Candidates, pr.Detail)
			}
		},
	}

	if p.survey {
		sv, reports, err := hunt.RunLiveSurvey(ctx, opts)
		if err != nil {
			rep.Fatal(1, fmt.Errorf("live survey: %w", err))
		}
		printSurvey(sv)
		maybeAutoGain(ctx, src, p, lockedControlChannels(sv.System, sv))
		return sv.System, sv, reports
	}

	sys, reports, err := hunt.RunLiveHunt(ctx, opts)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("live hunt: %w", err))
	}
	maybeAutoGain(ctx, src, p, lockedControlChannels(sys, nil))
	return sys, nil, reports
}

// lockedControlChannels collects the control-channel frequencies worth a gain
// sweep: every control channel of a discovered system, plus any survey carrier
// that locked as a trunk control channel.
func lockedControlChannels(sys *hunt.DiscoveredSystem, sv *hunt.SignalSurvey) []uint32 {
	seen := map[uint32]struct{}{}
	var out []uint32
	add := func(hz uint32) {
		if hz == 0 {
			return
		}
		if _, ok := seen[hz]; !ok {
			seen[hz] = struct{}{}
			out = append(out, hz)
		}
	}
	if sys != nil {
		for _, st := range sys.Sites {
			for _, ch := range st.ControlChannels {
				if ch.IsControl {
					add(ch.FrequencyHz)
				}
			}
		}
	}
	if sv != nil {
		for _, s := range sv.Signals {
			if s.Class == survey.ClassTrunkControl {
				add(s.FreqHz)
			}
		}
	}
	return out
}

// maybeAutoGain runs the optional gain sweep on the live SDR after a hunt/survey
// and prints the per-channel recommendation. It restores the run's configured
// gain afterwards and never applies a recommendation automatically.
func maybeAutoGain(ctx context.Context, src *streamIQSource, p huntLiveParams, ccs []uint32) {
	if !p.autoGain {
		return
	}
	if len(ccs) == 0 {
		fmt.Fprintln(os.Stderr, "auto-gain: no locked control channel to sweep")
		return
	}
	gains, err := parseGainSetDB(p.autoGainSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-gain: %v\n", err)
		return
	}
	if len(gains) == 0 {
		fmt.Fprintf(os.Stderr, "auto-gain: sweeping the default gain spread across %d control channel(s)…\n", len(ccs))
	} else {
		fmt.Fprintf(os.Stderr, "auto-gain: sweeping %d gain(s) across %d control channel(s)…\n", len(gains), len(ccs))
	}
	recs, err := hunt.AutoGainSweep(ctx, src, src, ccs, hunt.AutoGainOptions{
		Gains:        gains,
		Protocol:     p.protocol,
		SampleRateHz: p.sampleRateHz,
		AutoTune:     p.autoTune,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-gain: %v\n", err)
	}
	// Restore the operator's configured gain (the sweep left the SDR on the last
	// tried gain).
	_ = src.SetGain(p.gain)

	for _, rec := range recs {
		if rec.BestGainTenthDB < 0 {
			fmt.Fprintf(os.Stderr, "auto-gain: %.4f MHz — no usable decode at any gain\n", float64(rec.FreqHz)/1e6)
			continue
		}
		lock := ""
		if !rec.Locked {
			lock = " (no lock — low confidence)"
		}
		fmt.Fprintf(os.Stderr, "auto-gain: %.4f MHz — recommend %.1f dB (err %.2f/ksym)%s\n",
			float64(rec.FreqHz)/1e6, float64(rec.BestGainTenthDB)/10, rec.BestErrorRate, lock)
	}
	if p.outDir != "" && len(recs) > 0 {
		writeGainRecommendation(p.outDir, recs)
	}
}

// parseGainSetDB parses a comma-separated list of gains in dB into tenths of dB.
// An empty string yields nil (the sweep uses its default spread).
func parseGainSetDB(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []int
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		db, err := strconv.ParseFloat(tok, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gain %q in -auto-gain-set", tok)
		}
		out = append(out, int(db*10))
	}
	return out, nil
}

// writeGainRecommendation writes the sweep result as JSON into the out dir.
func writeGainRecommendation(outDir string, recs []hunt.GainRecommendation) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "auto-gain: %v\n", err)
		return
	}
	path := filepath.Join(outDir, "gain-recommendation.json")
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-gain: %v\n", err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(recs); err != nil {
		fmt.Fprintf(os.Stderr, "auto-gain: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "auto-gain: wrote %s\n", path)
}

// printSurvey writes the classified signal inventory to stderr — the survey's
// primary deliverable. The trunking export tail (finishHunt) runs afterwards on
// sv.System when a trunked system was found.
func printSurvey(sv *hunt.SignalSurvey) {
	trunking, analog, paging, other := sv.Counts()
	fmt.Fprintf(os.Stderr, "survey: %d signal(s) — %d trunking, %d analog, %d paging, %d other\n",
		len(sv.Signals), trunking, analog, paging, other)
	for _, s := range sv.Signals {
		line := fmt.Sprintf("  %10.4f MHz  %-13s  bw %5.1f kHz  snr %4.1f dB",
			float64(s.FreqHz)/1e6, s.Class, float64(s.OccupiedBwHz)/1e3, s.SNRDb)
		switch {
		case s.Trunking != nil:
			line += fmt.Sprintf("  [%s", s.Trunking.Protocol)
			if s.Trunking.Locked {
				line += " locked"
			}
			line += "]"
		case len(s.Pages) > 0:
			line += fmt.Sprintf("  [%d page(s), %s]", len(s.Pages), s.Pages[0].Protocol)
		case s.Analog != nil && s.Analog.Active:
			line += "  [active"
			if s.Analog.CTCSSHz > 0 {
				line += fmt.Sprintf(", CTCSS %.1f Hz", s.Analog.CTCSSHz)
			}
			if s.Analog.DCSCode != "" {
				line += fmt.Sprintf(", DCS %s", s.Analog.DCSCode)
			}
			line += "]"
		}
		if s.Error != "" {
			line += "  ERROR: " + s.Error
		}
		fmt.Fprintln(os.Stderr, line)
	}
}

// parseFreqListMHz parses a comma-separated MHz list into Hz. Empty ⇒ nil.
func parseFreqListMHz(rep *diag.Reporter, s string) []uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []uint32
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mhz, err := strconv.ParseFloat(part, 64)
		if err != nil {
			rep.Fatalf(2, "-candidates: invalid frequency %q", part)
		}
		out = append(out, uint32(mhz*1e6+0.5))
	}
	return out
}

// parseBandsMHz parses "low:high" MHz band specs into hunt.Band (Hz).
func parseBandsMHz(rep *diag.Reporter, specs []string) []hunt.Band {
	var out []hunt.Band
	for _, sp := range specs {
		lo, hi, ok := strings.Cut(sp, ":")
		if !ok {
			rep.Fatalf(2, "-band %q: want low:high in MHz", sp)
		}
		loMHz, e1 := strconv.ParseFloat(strings.TrimSpace(lo), 64)
		hiMHz, e2 := strconv.ParseFloat(strings.TrimSpace(hi), 64)
		if e1 != nil || e2 != nil || hiMHz <= loMHz {
			rep.Fatalf(2, "-band %q: invalid range", sp)
		}
		out = append(out, hunt.Band{LowHz: uint32(loMHz*1e6 + 0.5), HighHz: uint32(hiMHz*1e6 + 0.5)})
	}
	return out
}
