package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// huntLiveParams are the resolved inputs for a live (on-air) hunt.
type huntLiveParams struct {
	serial          string
	bands           []string // "low:high" in MHz
	candidatesMHz   string   // comma-separated MHz
	noSweep         bool
	sampleRateHz    float64
	protocol        trunking.Protocol
	fftSize         int
	sweepDwell      time.Duration
	peakThresholdDb float64
	minSpacingHz    uint32
	dwellSeconds    float64
	autoTune        bool
	gain            int
	ppm             int
	name            string
	state           string
	county          string
	location        string
	minConfidence   float64
}

// runHuntLive opens the SDR directly (standalone, not through the daemon pool),
// sweeps the requested band(s) — or probes an explicit candidate list — and
// returns the discovered system plus per-candidate reports for the shared
// export tail. The daemon-integrated live hunt (with spare-SDR-else-borrow
// acquisition and a REST/TUI/web cockpit) is a later phase; this is the
// one-shot CLI path.
func runHuntLive(rep *diag.Reporter, p huntLiveParams) (*hunt.DiscoveredSystem, []hunt.CaptureReport) {
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

	if len(bands) > 0 {
		fmt.Fprintf(os.Stderr, "hunt: live sweep on %s[%s] @ %g MS/s across %d band(s)…\n",
			info.Driver, info.Serial, p.sampleRateHz/1e6, len(bands))
	} else {
		fmt.Fprintf(os.Stderr, "hunt: live probe on %s[%s] of %d candidate(s)…\n",
			info.Driver, info.Serial, len(candidates))
	}

	sys, reports, err := hunt.RunLiveHunt(ctx, hunt.LiveHuntOptions{
		Source:        src,
		Bands:         bands,
		Candidates:    candidates,
		Protocol:      p.protocol,
		FFTSize:       p.fftSize,
		SweepDwell:    p.sweepDwell,
		PeakOpts:      hunt.PeakOptions{ThresholdDb: float32(p.peakThresholdDb), MinSpacingHz: p.minSpacingHz},
		DwellSeconds:  p.dwellSeconds,
		MinConfidence: p.minConfidence,
		AutoTune:      p.autoTune,
		Name:          p.name,
		State:         p.state,
		County:        p.county,
		Location:      p.location,
		OnProgress: func(pr hunt.LiveHuntProgress) {
			switch pr.Phase {
			case hunt.PhaseSweeping:
				fmt.Fprintf(os.Stderr, "hunt: sweeping %.4f MHz — %s\n", float64(pr.CenterHz)/1e6, pr.Detail)
			case hunt.PhaseIdentifying:
				fmt.Fprintf(os.Stderr, "hunt: probing candidate %d/%d @ %s\n", pr.CandidateN, pr.Candidates, pr.Detail)
			}
		},
	})
	if err != nil {
		rep.Fatal(1, fmt.Errorf("live hunt: %w", err))
	}
	return sys, reports
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

// deviceIQSource adapts a live sdr.Device to hunt.IQSource. A background
// goroutine drains the device's IQ stream into a channel; Capture pulls from a
// rolling buffer and Tune retunes the device, flushing the transient.
type deviceIQSource struct {
	dev     sdr.Device
	rate    uint32
	ch      <-chan []complex64
	pending []complex64
	settle  time.Duration
}

func newDeviceIQSource(ctx context.Context, dev sdr.Device, rate uint32) (*deviceIQSource, error) {
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	return &deviceIQSource{dev: dev, rate: rate, ch: ch, settle: 50 * time.Millisecond}, nil
}

func (s *deviceIQSource) SampleRateHz() uint32 { return s.rate }

func (s *deviceIQSource) Tune(centerHz uint32) error {
	if err := s.dev.SetCenterFreq(centerHz); err != nil {
		return err
	}
	// Discard buffered + in-flight samples captured before/at the retune so the
	// next Capture sees only settled IQ at the new center.
	s.pending = nil
	deadline := time.NewTimer(s.settle)
	defer deadline.Stop()
	for {
		select {
		case <-s.ch:
			// drain
		case <-deadline.C:
			return nil
		}
	}
}

func (s *deviceIQSource) Capture(ctx context.Context, n int) ([]complex64, error) {
	for len(s.pending) < n {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk, ok := <-s.ch:
			if !ok {
				out := s.pending
				s.pending = nil
				return out, nil // stream ended
			}
			s.pending = append(s.pending, chunk...)
		}
	}
	out := s.pending[:n:n]
	s.pending = s.pending[n:]
	return out, nil
}
