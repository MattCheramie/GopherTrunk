package hunt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
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
	// Survey selects the signal-survey pipeline (RunLiveSurvey): classify and
	// decode every detected carrier, not just trunking control channels. The
	// daemon Manager reads this to dispatch the right run.
	Survey bool
	// ClassifyConfig tunes the survey classifier thresholds. The zero value
	// uses survey.DefaultClassifyConfig.
	ClassifyConfig survey.ClassifyConfig
	// IdentifyMinConfidence, when > 0, skips the (expensive) trunking identify
	// for a digital carrier whose classifier confidence is below it — unless it
	// is at a paging baud. 0 ⇒ always identify (never drop a possible CC).
	IdentifyMinConfidence float64
	// ClassifyOnly records the classifier verdict for every carrier and skips
	// all decoding (trunking identify, paging, analog tone scan) for a fast
	// inventory.
	ClassifyOnly bool
	// SkipFreqs, when non-empty, drops candidate carriers whose center
	// frequency is already present (a resumed survey reads these from the
	// existing NDJSON output so it doesn't re-survey carriers it already has).
	SkipFreqs map[uint32]struct{}
	// PersistSurvey requests the daemon Manager stream this survey run's
	// classified carriers to a crash-safe NDJSON file (in ManagerOptions.SurveyDir),
	// so it can be resumed. Ignored for a plain (non-survey) hunt or when no
	// SurveyDir is configured. The CLI persists via its own -survey-ndjson flag.
	PersistSurvey bool
	// Resume, with PersistSurvey, preloads the frequencies already in the run's
	// NDJSON file into SkipFreqs so an interrupted survey continues.
	Resume bool
	// AutoGain requests a post-run gain sweep: after the hunt/survey, each locked
	// control channel is decoded across a set of front-end gains and the gain
	// minimising the decode error rate is recommended (advisory; nothing is
	// applied). Requires a gain-controllable, exclusively-held SDR — a no-op with
	// a clear note when the SDR is shared (the daemon's borrowed control SDR).
	AutoGain bool
	// AutoGainSet is the gain ladder to sweep (tenths of dB); empty uses a default.
	AutoGainSet []int
	// SurveyDeep hands narrowband carriers the blind classifier called analog
	// (am/nbfm/wfm ≤ NBFM bandwidth) to the authoritative siglab identify before
	// the analog tone scan, so a trunked DMR/TETRA/MPT control channel the blind
	// classifier missed is still discovered. Slower (an extra identify per
	// borderline carrier); the accurate-survey opt-in for digital-dense bands.
	SurveyDeep bool
	// Serial requests a specific SDR for the run. Empty ⇒ the daemon
	// auto-selects (spare SDR, else borrow the control SDR). Consumed by the
	// daemon Acquirer; RunLiveHunt itself ignores it (the IQSource is already
	// resolved by the time RunLiveHunt runs).
	Serial string

	FFTSize    int
	SweepDwell time.Duration
	GuardFrac  float64
	PeakOpts   PeakOptions
	// DetectCarriers expands a wideband OFFLINE capture into every carrier it
	// contains (FFT peak-detect) instead of treating the whole file as one
	// channel — so an off-centre / non-dominant control channel is found and
	// the rest of the band is inventoried. Live sweeps already do this via the
	// SDR; this is the offline equivalent. Ignored for live runs.
	DetectCarriers bool

	// DwellSeconds is how much IQ to capture per candidate for identify+decode.
	// 0 ⇒ 3 s.
	DwellSeconds float64
	// MaxDwellSeconds, when > DwellSeconds, lets a survey candidate extend its
	// capture (in DwellSeconds chunks) until carrier activity is seen or this
	// ceiling is reached — useful for bursty paging. 0 ⇒ a single fixed dwell.
	MaxDwellSeconds float64
	// MonitorSeconds, when > 0, switches each candidate from the buffered
	// fixed-dwell capture to a streaming long-dwell control-channel monitor:
	// live IQ is decoded in real time (bounded memory, no multi-GB capture) for
	// up to this many seconds, stopping early once identity + neighbors + band
	// plan settle. The short DwellSeconds prefix is still captured first to
	// identify the protocol. Opt-in: 0 ⇒ today's buffered behavior.
	MonitorSeconds float64
	MinConfidence  float64
	AutoTune       bool

	// SurveyAudioDir, when set, makes a survey write a WAV clip per active
	// analog-FM carrier into this directory.
	SurveyAudioDir string

	// System metadata for the resulting map.
	Name, State, County, Location string

	OnProgress func(LiveHuntProgress)
	// OnSignal, when set, is called once per classified carrier during a survey
	// run (RunLiveSurvey), after its decode summary is filled in. The daemon
	// uses it to publish per-signal events and to persist decoded pages; the CLI
	// leaves it nil. Ignored by RunLiveHunt.
	OnSignal func(DetectedSignal)
	Log      *slog.Logger
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
			Detail: fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6),
		})

		params := decodeParams{
			Protocol:      opts.Protocol,
			Format:        siglab.FormatF32,
			SampleRateHz:  float64(rate),
			FrequencyHz:   cand.FreqHz,
			AutoTune:      opts.AutoTune,
			MinConfidence: opts.MinConfidence,
			Log:           log,
		}

		// Streaming long-dwell monitor: decode the live source in real time
		// instead of buffering the whole dwell (the only way to watch a CC for
		// minutes without recording gigabytes of IQ).
		if opts.MonitorSeconds > 0 {
			rep := monitorCandidate(ctx, sys, opts.Source, cand.FreqHz, nSamples, opts.MonitorSeconds, params)
			reports = append(reports, rep)
			continue
		}

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
			fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6), params)
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
