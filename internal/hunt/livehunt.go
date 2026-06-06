package hunt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LiveHuntPhase labels the stage a live hunt is in, for progress reporting.
type LiveHuntPhase string

const (
	PhaseSweeping    LiveHuntPhase = "sweeping"
	PhaseIdentifying LiveHuntPhase = "identifying"
	PhaseDone        LiveHuntPhase = "done"
)

// LiveHuntProgress is one progress notification emitted during a live hunt.
type LiveHuntProgress struct {
	Phase      LiveHuntPhase `json:"phase"`
	CenterHz   uint32        `json:"center_hz,omitempty"`
	CandidateN int           `json:"candidate_n,omitempty"`
	Candidates int           `json:"candidates,omitempty"`
	Detail     string        `json:"detail,omitempty"`
}

// LiveHuntOptions configure a live hunt.
type LiveHuntOptions struct {
	Source IQSource
	// Bands to sweep. Ignored when Candidates is non-empty.
	Bands []Band
	// Candidates, when set, are explicit control-channel frequencies to probe
	// directly (the sweep is skipped). Mirrors the offline -no-sweep path.
	Candidates []uint32
	// Protocol forces a decoder; trunking.ProtocolUnknown auto-identifies.
	Protocol trunking.Protocol

	FFTSize    int
	SweepDwell time.Duration
	GuardFrac  float64
	PeakOpts   PeakOptions

	// DwellSeconds is how much IQ to capture per candidate for identify+decode.
	// 0 ⇒ 3 s.
	DwellSeconds  float64
	MinConfidence float64
	AutoTune      bool

	// System metadata for the resulting map.
	Name, State, County, Location string

	OnProgress func(LiveHuntProgress)
	Log        *slog.Logger
}

// RunLiveHunt sweeps (or probes a candidate list) on a live IQSource, then
// identifies, decodes, and maps each candidate into a DiscoveredSystem. It is
// the on-air sibling of Discover: the sweep replaces operator-supplied
// captures, but the per-candidate identify→decode→accumulate body is shared
// (decodeAndAccumulate). Returns the discovered system, a per-candidate report,
// and an error only for fatal setup/sweep failures (per-candidate failures are
// recorded in the reports).
func RunLiveHunt(ctx context.Context, opts LiveHuntOptions) (*DiscoveredSystem, []CaptureReport, error) {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	if opts.Source == nil {
		return nil, nil, fmt.Errorf("hunt: live hunt requires an IQSource")
	}
	rate := opts.Source.SampleRateHz()
	if rate == 0 {
		return nil, nil, fmt.Errorf("hunt: IQSource has zero sample rate")
	}
	dwell := opts.DwellSeconds
	if dwell <= 0 {
		dwell = 3
	}
	progress := func(p LiveHuntProgress) {
		if opts.OnProgress != nil {
			opts.OnProgress(p)
		}
	}

	// Phase 1: candidates — explicit list, or a spectrum sweep.
	var candidates []Candidate
	if len(opts.Candidates) > 0 {
		for _, f := range opts.Candidates {
			candidates = append(candidates, Candidate{FreqHz: f})
		}
	} else {
		sw, err := NewSweeper(SweepOptions{
			Source:     opts.Source,
			Bands:      opts.Bands,
			FFTSize:    opts.FFTSize,
			SweepDwell: opts.SweepDwell,
			GuardFrac:  opts.GuardFrac,
			PeakOpts:   opts.PeakOpts,
			Log:        log,
		})
		if err != nil {
			return nil, nil, err
		}
		progress(LiveHuntProgress{Phase: PhaseSweeping, Detail: "scanning bands"})
		candidates, err = sw.Sweep(ctx, func(centerHz uint32, peaks []Peak) {
			progress(LiveHuntProgress{Phase: PhaseSweeping, CenterHz: centerHz,
				Detail: fmt.Sprintf("%d peak(s)", len(peaks))})
		})
		if err != nil {
			return nil, nil, err
		}
	}

	sys := &DiscoveredSystem{
		Name:     opts.Name,
		State:    opts.State,
		County:   opts.County,
		Location: opts.Location,
	}
	reports := make([]CaptureReport, 0, len(candidates))
	nSamples := int(dwell * float64(rate))

	for i, cand := range candidates {
		if err := ctx.Err(); err != nil {
			return sys, reports, err
		}
		progress(LiveHuntProgress{
			Phase: PhaseIdentifying, CenterHz: cand.FreqHz,
			CandidateN: i + 1, Candidates: len(candidates),
			Detail: fmt.Sprintf("%.3f MHz", float64(cand.FreqHz)/1e6),
		})

		if err := opts.Source.Tune(cand.FreqHz); err != nil {
			reports = append(reports, CaptureReport{
				Path:  fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6),
				Error: fmt.Sprintf("tune: %v", err),
			})
			continue
		}
		iq, err := captureN(ctx, opts.Source, nSamples)
		if err != nil {
			return sys, reports, err
		}

		// The candidate carrier is at DC of the tuned capture; decode at
		// baseband. Encode once (F32, lossless from complex64) and feed the
		// shared identify→decode→accumulate body over a seekable reader.
		buf := siglab.EncodeCapture(iq, siglab.FormatF32)
		rep := decodeAndAccumulate(sys, bytes.NewReader(buf),
			fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6), decodeParams{
				Protocol:      opts.Protocol,
				Format:        siglab.FormatF32,
				SampleRateHz:  float64(rate),
				FrequencyHz:   cand.FreqHz,
				AutoTune:      opts.AutoTune,
				MinConfidence: opts.MinConfidence,
				Log:           log,
			})
		reports = append(reports, rep)
	}

	sys.sortAll()
	progress(LiveHuntProgress{Phase: PhaseDone,
		Detail: fmt.Sprintf("%d site(s), %d talkgroup(s)", len(sys.Sites), len(sys.Talkgroups))})
	return sys, reports, nil
}

// captureN gathers exactly n samples from src (Capture may return fewer per
// call). It stops early at end-of-stream, returning what it has.
func captureN(ctx context.Context, src IQSource, n int) ([]complex64, error) {
	out := make([]complex64, 0, n)
	for len(out) < n {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		chunk, err := src.Capture(ctx, n-len(out))
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break // end of stream
		}
		out = append(out, chunk...)
	}
	return out, nil
}

// SortedCandidates returns candidates sorted by descending SNR (test helper).
func SortedCandidates(c []Candidate) []Candidate {
	out := append([]Candidate(nil), c...)
	sort.Slice(out, func(i, j int) bool { return out[i].SNRDb > out[j].SNRDb })
	return out
}
