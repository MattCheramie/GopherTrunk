package hunt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// surveyChannelRateHz is the rate each detected carrier is decimated to before
// classification and conventional decode. 48 kHz matches the narrowband rate
// the wideband-T2 engine and the single-channel ccdecoder target, gives the
// blind classifier fine spectral resolution, and band-limits the carrier so the
// FM-demod chain isn't swamped by out-of-channel noise. The trunking decode
// still runs on the full-rate capture (siglab channelises it itself).
const surveyChannelRateHz = 48_000

// RunLiveSurvey sweeps (or probes a candidate list) like RunLiveHunt, but
// instead of only mapping trunking control channels it classifies every
// detected carrier and routes it: trunking carriers fold into the discovered
// system (the existing identify→decode→accumulate path), paging and analog
// carriers run their conventional decoders, and the rest are recorded as
// classified-only. It returns the full SignalSurvey (which embeds the trunking
// map) plus the per-candidate trunking reports for the shared export tail.
func RunLiveSurvey(ctx context.Context, opts LiveHuntOptions) (*SignalSurvey, []CaptureReport, error) {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	if opts.Source == nil {
		return nil, nil, fmt.Errorf("hunt: live survey requires an IQSource")
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

	candidates, err := surveyCandidates(ctx, opts, log, progress)
	if err != nil {
		return nil, nil, err
	}

	sv := &SignalSurvey{
		StartedAt: time.Now(),
		System: &DiscoveredSystem{
			Name:     opts.Name,
			State:    opts.State,
			County:   opts.County,
			Location: opts.Location,
		},
	}
	reports := make([]CaptureReport, 0, len(candidates))
	nSamples := int(dwell * float64(rate))

	for i, cand := range candidates {
		if err := ctx.Err(); err != nil {
			return finishSurvey(sv), reports, err
		}
		progress(LiveHuntProgress{
			Phase: PhaseIdentifying, CenterHz: cand.FreqHz,
			CandidateN: i + 1, Candidates: len(candidates),
			Detail: fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6),
		})

		ds := DetectedSignal{FreqHz: cand.FreqHz, SNRDb: cand.SNRDb}
		if err := opts.Source.Tune(cand.FreqHz); err != nil {
			ds.Error = fmt.Sprintf("tune: %v", err)
			ds.Class = survey.ClassUnknown
			sv.Signals = append(sv.Signals, ds)
			continue
		}
		iq, err := captureN(ctx, opts.Source, nSamples)
		if err != nil {
			return finishSurvey(sv), reports, err
		}

		// Channelise to a narrow baseband stream for classification + the
		// conventional (analog/paging) decoders.
		chIQ, chRate := channelize(iq, rate, surveyChannelRateHz)
		cls := survey.Classify(chIQ, float64(chRate))
		ds.Class = cls.Class
		ds.Confidence = cls.Confidence
		ds.OccupiedBwHz = cls.OccupiedBwHz
		ds.BaudHz = cls.Features.BaudHz
		ds.Features = cls.Features

		rep := routeSignal(sv.System, &ds, routeInputs{
			fullIQ: iq, fullRate: float64(rate),
			chIQ: chIQ, chRate: chRate,
			cand: cand, opts: opts, log: log,
		})
		if rep != nil {
			reports = append(reports, *rep)
		}
		sv.Signals = append(sv.Signals, ds)
		if opts.OnSignal != nil {
			opts.OnSignal(ds)
		}
	}

	return finishSurvey(sv), reports, nil
}

// routeInputs bundles the per-candidate buffers and config the router needs.
type routeInputs struct {
	fullIQ   []complex64
	fullRate float64
	chIQ     []complex64
	chRate   uint32
	cand     Candidate
	opts     LiveHuntOptions
	log      *slog.Logger
}

// routeSignal decodes ds according to its class, mutating ds with the decode
// summary. For trunking-family carriers it runs the shared identify→decode→
// accumulate body (returning its CaptureReport); for paging/analog it runs the
// conventional decoders; everything else is left as classified-only. Returns a
// non-nil CaptureReport only when the trunking path ran.
func routeSignal(sys *DiscoveredSystem, ds *DetectedSignal, in routeInputs) *CaptureReport {
	source := fmt.Sprintf("%.4f MHz", float64(in.cand.FreqHz)/1e6)

	// Paging: a digital carrier at a POCSAG/FLEX baud — prove it by decoding.
	if survey.IsDigital(ds.Class) && survey.IsPagingBaud(ds.Features.BaudHz) {
		pages := survey.DecodePOCSAG(in.chIQ, in.chRate)
		if flex := survey.DecodeFLEX(in.chIQ, in.chRate); len(flex) > len(pages) {
			pages = flex
		}
		if len(pages) > 0 {
			ds.Pages = pages
			ds.Class = survey.ClassPaging
			return nil
		}
		// No pages — fall through to a trunking identify (could be NXDN/DMR).
	}

	// Trunking: hand digital carriers to the authoritative siglab identify on
	// the full-rate capture (siglab channelises and auto-tunes internally).
	if survey.IsDigital(ds.Class) {
		buf := siglab.EncodeCapture(in.fullIQ, siglab.FormatF32)
		rep := decodeAndAccumulate(sys, bytes.NewReader(buf), source, decodeParams{
			Protocol:      in.opts.Protocol,
			Format:        siglab.FormatF32,
			SampleRateHz:  in.fullRate,
			FrequencyHz:   in.cand.FreqHz,
			AutoTune:      in.opts.AutoTune,
			MinConfidence: in.opts.MinConfidence,
			Log:           in.log,
		})
		switch {
		case rep.Locked:
			ds.Class = survey.ClassTrunkControl
			ds.Trunking = &TrunkingRef{Protocol: rep.Protocol, Confidence: rep.Confidence, Locked: true, ControlHz: rep.ControlHz}
		case !rep.Skipped && rep.Error == "":
			ds.Class = survey.ClassTrunkVoice
			ds.Trunking = &TrunkingRef{Protocol: rep.Protocol, Confidence: rep.Confidence}
		}
		// Skipped/errored ⇒ keep the classifier's family label (generic FSK/etc.).
		return &rep
	}

	// Analog FM / AM: carrier activity + sub-audible squelch identification.
	switch ds.Class {
	case survey.ClassNBFM, survey.ClassWideFM, survey.ClassAM:
		ds.Analog = survey.AnalyzeAnalogFM(in.chIQ, in.chRate)
	}
	return nil
}

// surveyCandidates resolves the carriers to examine: an explicit candidate list
// or a spectrum sweep (shared with RunLiveHunt's front-end).
func surveyCandidates(ctx context.Context, opts LiveHuntOptions, log *slog.Logger, progress func(LiveHuntProgress)) ([]Candidate, error) {
	if len(opts.Candidates) > 0 {
		out := make([]Candidate, 0, len(opts.Candidates))
		for _, f := range opts.Candidates {
			out = append(out, Candidate{FreqHz: f})
		}
		return out, nil
	}
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
		return nil, err
	}
	progress(LiveHuntProgress{Phase: PhaseSweeping, Detail: "scanning bands"})
	return sw.Sweep(ctx, func(centerHz uint32, peaks []Peak) {
		progress(LiveHuntProgress{Phase: PhaseSweeping, CenterHz: centerHz,
			Detail: fmt.Sprintf("%d peak(s)", len(peaks))})
	})
}

// finishSurvey stamps the finish time, sorts the inventory, and clears an empty
// trunking map so callers can treat System==nil as "no system found".
func finishSurvey(sv *SignalSurvey) *SignalSurvey {
	sv.FinishedAt = time.Now()
	sv.sortSignals()
	if sv.System != nil {
		sv.System.sortAll()
		if len(sv.System.Sites) == 0 && len(sv.System.Talkgroups) == 0 {
			sv.System = nil
		}
	}
	return sv
}

// channelize decimates wideband IQ to ~targetHz by an integer factor, band-
// limiting the carrier at DC. It returns the decimated buffer and its actual
// rate. When the source is already at or below targetHz it is returned
// unchanged (the test FileIQSource runs at the channel rate directly).
func channelize(iq []complex64, rateHz, targetHz uint32) ([]complex64, uint32) {
	if rateHz <= targetHz || len(iq) == 0 {
		return iq, rateHz
	}
	m := int((float64(rateHz) / float64(targetHz)) + 0.5)
	if m < 2 {
		return iq, rateHz
	}
	out := dsp.NewResampler(1, m, 16, 7.0).Process(nil, iq)
	return out, rateHz / uint32(m)
}
