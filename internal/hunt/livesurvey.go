package hunt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
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
	maxSamples := nSamples
	if opts.MaxDwellSeconds > dwell {
		maxSamples = int(opts.MaxDwellSeconds * float64(rate))
	}

	for i, cand := range candidates {
		if err := ctx.Err(); err != nil {
			return finishSurvey(sv), reports, err
		}
		progress(LiveHuntProgress{
			Phase: PhaseIdentifying, CenterHz: cand.FreqHz,
			CandidateN: i + 1, Candidates: len(candidates),
			Detail: fmt.Sprintf("%.4f MHz", float64(cand.FreqHz)/1e6),
		})

		if err := opts.Source.Tune(cand.FreqHz); err != nil {
			ds := DetectedSignal{FreqHz: cand.FreqHz, SNRDb: cand.SNRDb,
				Class: survey.ClassUnknown, Error: fmt.Sprintf("tune: %v", err)}
			sv.Signals = append(sv.Signals, ds)
			if opts.OnSignal != nil {
				opts.OnSignal(ds)
			}
			continue
		}
		iq, err := captureCandidate(ctx, opts.Source, nSamples, maxSamples, rate)
		if err != nil {
			return finishSurvey(sv), reports, err
		}

		ds, rep := classifyAndRoute(sv.System, iq, rate, cand, nearestNeighborHz(candidates, i), opts, log)
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

// RunOfflineSurvey classifies and routes a set of capture files without an SDR
// — the offline sibling of RunLiveSurvey, mirroring how Discover is the offline
// sibling of RunLiveHunt. Each capture is loaded, treated as one baseband
// candidate (at its CaptureInput.FrequencyHz), classified, and routed through
// the same body the live survey uses, so an operator can survey recorded IQ
// (e.g. a wideband grab) the same way they survey on the air.
func RunOfflineSurvey(captures []CaptureInput, opts LiveHuntOptions) (*SignalSurvey, []CaptureReport, error) {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	sv := &SignalSurvey{
		StartedAt: time.Now(),
		System: &DiscoveredSystem{
			Name: opts.Name, State: opts.State, County: opts.County, Location: opts.Location,
		},
	}
	reports := make([]CaptureReport, 0, len(captures))
	for _, ci := range captures {
		iq, rate, err := loadCapture(ci)
		if err != nil {
			sv.Signals = append(sv.Signals, DetectedSignal{
				FreqHz: ci.FrequencyHz, Class: survey.ClassUnknown,
				Error: fmt.Sprintf("load %s: %v", ci.Path, err),
			})
			continue
		}
		cand := Candidate{FreqHz: ci.FrequencyHz}
		// Each offline capture is a lone baseband grab: no neighbour to bound
		// against, so the occupied-bandwidth walk stays unbounded.
		ds, rep := classifyAndRoute(sv.System, iq, rate, cand, 0, opts, log)
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

// classifyAndRoute is the shared per-candidate body of the live and offline
// surveys: measure occupied bandwidth on the full-rate capture, channelise to a
// narrow baseband stream, classify, and route to the matching decoder.
func classifyAndRoute(sys *DiscoveredSystem, fullIQ []complex64, fullRate uint32, cand Candidate, neighborHz uint32, opts LiveHuntOptions, log *slog.Logger) (DetectedSignal, *CaptureReport) {
	ds := DetectedSignal{FreqHz: cand.FreqHz, SNRDb: cand.SNRDb}

	// Occupied bandwidth from the full-rate capture (a wideband signal isn't
	// truncated by the channel decimation), but bounded by the nearest-neighbour
	// spacing so a dense band (DMR on a 12.5 kHz grid) doesn't bridge its
	// neighbours into one giant span. Modulation features + conventional decoders
	// run on the carrier isolated to its own channel by the same bound, so an
	// adjacent carrier 12.5 kHz away can't leak into the discriminator and defeat
	// the digital/baud detection (which is what mislabels DMR as am/wfm).
	wideBw, wideSnr := survey.OccupiedBandwidth(fullIQ, float64(fullRate), neighborHz, opts.ClassifyConfig)
	chIQ, chRate := channelize(fullIQ, fullRate, classifyChannelRate(neighborHz))
	cls := survey.ClassifyWith(chIQ, float64(chRate), wideBw, wideSnr, opts.ClassifyConfig)
	ds.Class = cls.Class
	ds.Confidence = cls.Confidence
	ds.OccupiedBwHz = cls.OccupiedBwHz
	ds.BaudHz = cls.Features.BaudHz
	ds.Features = cls.Features

	rep := routeSignal(sys, &ds, routeInputs{
		fullIQ: fullIQ, fullRate: float64(fullRate),
		chIQ: chIQ, chRate: chRate,
		cand: cand, opts: opts, log: log,
	})
	return ds, rep
}

// loadCapture reads an IQ capture file into complex64 using the format's shared
// decoder (siglab.SampleFormat.Decoder).
func loadCapture(ci CaptureInput) ([]complex64, uint32, error) {
	raw, err := os.ReadFile(ci.Path)
	if err != nil {
		return nil, 0, err
	}
	dec, bytesPerPair := ci.Format.Decoder()
	n := len(raw) / bytesPerPair
	if n == 0 {
		return nil, 0, fmt.Errorf("capture too short (%d bytes)", len(raw))
	}
	iq := make([]complex64, n)
	dec(raw[:n*bytesPerPair], iq)
	return iq, uint32(ci.SampleRateHz), nil
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

	// Classify-only: record the verdict, decode nothing.
	if in.opts.ClassifyOnly {
		return nil
	}

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
	// Skip the (expensive) identify for a low-confidence digital carrier when an
	// operator opts into the gate — but never for a paging-baud carrier (already
	// handled above) and never when the gate is 0 (default), so a real control
	// channel is never dropped.
	if survey.IsDigital(ds.Class) {
		if in.opts.IdentifyMinConfidence > 0 && ds.Confidence < in.opts.IdentifyMinConfidence {
			return nil
		}
		return identifyTrunking(sys, ds, in, source, true)
	}

	// Deep survey: a narrowband carrier the blind classifier called analog can
	// still be a trunked DMR/TETRA/MPT control channel whose fragile baud line
	// was lost (issue #648). Hand it to the authoritative siglab identify first.
	// If it locks (or decodes) the class is upgraded to trunking and we're done;
	// otherwise it really is analog, so we still run the tone scan — but return
	// the identify report so the attempt is recorded — at the cost of one
	// identify per borderline carrier.
	if in.opts.SurveyDeep && isAnalogClass(ds.Class) && ds.OccupiedBwHz <= surveyDeepMaxBwHz {
		// Lock-only: relabel as trunking solely on a genuine control-channel lock,
		// never on a non-locked best-guess identify — a real analog carrier must
		// not be promoted to trunk-voice by siglab's weak runner-up protocol.
		rep := identifyTrunking(sys, ds, in, source, false)
		if ds.Class == survey.ClassTrunkControl {
			return rep
		}
		analyzeAnalog(ds, in)
		return rep
	}

	// Analog FM / AM: carrier activity + sub-audible squelch identification,
	// plus an optional WAV clip when -survey-audio is set.
	analyzeAnalog(ds, in)
	return nil
}

// analyzeAnalog runs the analog-FM activity + sub-audible squelch scan on the
// channelised carrier (and writes a WAV clip when -survey-audio is set), for the
// analog families. A no-op for any other class.
func analyzeAnalog(ds *DetectedSignal, in routeInputs) {
	if !isAnalogClass(ds.Class) {
		return
	}
	ds.Analog = survey.AnalyzeAnalogFM(in.chIQ, in.chRate)
	if in.opts.SurveyAudioDir != "" && ds.Analog.Active {
		path := filepath.Join(in.opts.SurveyAudioDir,
			fmt.Sprintf("%.4fMHz.wav", float64(in.cand.FreqHz)/1e6))
		if err := survey.WriteAnalogClip(path, in.chIQ, in.chRate); err != nil {
			ds.Error = fmt.Sprintf("audio clip: %v", err)
		} else {
			ds.Analog.AudioClipPath = path
		}
	}
}

// surveyDeepMaxBwHz bounds the deep-survey identify to narrowband carriers — a
// trunked control channel is narrowband, and it keeps the (expensive) identify
// off wideband broadcast FM. Matches the NBFM/WideFM split.
const surveyDeepMaxBwHz = 25_000

// isAnalogClass reports whether the blind classifier put the carrier in an
// analog family (the classes deep survey reconsiders via siglab).
func isAnalogClass(c survey.SignalClass) bool {
	switch c {
	case survey.ClassNBFM, survey.ClassWideFM, survey.ClassAM:
		return true
	default:
		return false
	}
}

// identifyTrunking runs the authoritative siglab identify→decode on the full-rate
// capture and, when it locks (or decodes without skipping), upgrades ds.Class to
// trunk-control / trunk-voice and attaches the trunking ref. A skipped/errored
// identify leaves ds.Class as the classifier's family label. Returns the decode
// report for the shared export tail.
// voiceUpgrade ⇒ a non-locked but conclusive identify upgrades the carrier to
// trunk-voice (used for carriers the blind classifier already called digital);
// when false, only a genuine control-channel lock relabels the carrier, so a
// blind-analog carrier is never promoted by a weak runner-up identify.
func identifyTrunking(sys *DiscoveredSystem, ds *DetectedSignal, in routeInputs, source string, voiceUpgrade bool) *CaptureReport {
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
	case voiceUpgrade && !rep.Skipped && rep.Error == "":
		ds.Class = survey.ClassTrunkVoice
		ds.Trunking = &TrunkingRef{Protocol: rep.Protocol, Confidence: rep.Confidence}
	}
	// Skipped/errored (or non-locked under lock-only) ⇒ keep the classifier label.
	return &rep
}

// surveyCandidates resolves the carriers to examine: an explicit candidate list
// or a spectrum sweep (shared with RunLiveHunt's front-end).
func surveyCandidates(ctx context.Context, opts LiveHuntOptions, log *slog.Logger, progress func(LiveHuntProgress)) ([]Candidate, error) {
	if len(opts.Candidates) > 0 {
		out := make([]Candidate, 0, len(opts.Candidates))
		for _, f := range opts.Candidates {
			out = append(out, Candidate{FreqHz: f})
		}
		return skipSurveyed(out, opts.SkipFreqs), nil
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
	cands, err := sw.Sweep(ctx, func(centerHz uint32, peaks []Peak) {
		progress(LiveHuntProgress{Phase: PhaseSweeping, CenterHz: centerHz,
			Detail: fmt.Sprintf("%d peak(s)", len(peaks))})
	})
	if err != nil {
		return nil, err
	}
	return skipSurveyed(cands, opts.SkipFreqs), nil
}

// skipSurveyed drops candidates whose frequency is already in skip (a resumed
// survey), preserving order. A nil/empty skip set is a no-op.
func skipSurveyed(cands []Candidate, skip map[uint32]struct{}) []Candidate {
	if len(skip) == 0 {
		return cands
	}
	out := cands[:0]
	for _, c := range cands {
		if _, done := skip[c.FreqHz]; done {
			continue
		}
		out = append(out, c)
	}
	return out
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

// surveyActivityDbFS is the carrier-present threshold for the activity-dwell
// loop: a chunk whose channelised power clears it is treated as containing
// traffic and ends the dwell early.
const surveyActivityDbFS = -30

// captureCandidate gathers IQ for one candidate. With maxSamples == chunk it is
// a single fixed-dwell grab. When maxSamples is larger (MaxDwellSeconds set), it
// captures in chunk-sized windows up to maxSamples, returning as soon as a
// window shows carrier activity (so bursty paging/voice isn't missed in a
// blind window), else the strongest window seen.
func captureCandidate(ctx context.Context, src IQSource, chunk, maxSamples int, rate uint32) ([]complex64, error) {
	if maxSamples <= chunk {
		return captureN(ctx, src, chunk)
	}
	var best []complex64
	bestPwr := math.Inf(-1)
	for got := 0; got < maxSamples; got += chunk {
		iq, err := captureN(ctx, src, chunk)
		if err != nil {
			return nil, err
		}
		if len(iq) == 0 {
			break
		}
		chIQ, _ := channelize(iq, rate, surveyChannelRateHz)
		if pwr := survey.CarrierPowerDbFS(chIQ); pwr >= surveyActivityDbFS {
			return iq, nil // activity — use this window
		} else if pwr > bestPwr {
			bestPwr, best = pwr, iq
		}
	}
	if best == nil {
		return captureN(ctx, src, chunk)
	}
	return best, nil
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

// nearestNeighborHz returns the smallest absolute frequency distance from
// candidates[i] to any other candidate, or 0 when there is no other candidate
// (a lone carrier — nothing to isolate against).
func nearestNeighborHz(candidates []Candidate, i int) uint32 {
	if i < 0 || i >= len(candidates) {
		return 0
	}
	fi := candidates[i].FreqHz
	var best uint32
	for j, c := range candidates {
		if j == i || c.FreqHz == fi {
			continue
		}
		d := c.FreqHz - fi
		if c.FreqHz < fi {
			d = fi - c.FreqHz
		}
		if best == 0 || d < best {
			best = d
		}
	}
	return best
}

// classifyChannelRate picks the channel rate to isolate a carrier for
// classification. The default surveyChannelRateHz (48 kHz, ±24 kHz passband) is
// wide enough that on a dense grid (DMR at 12.5 kHz spacing) several neighbours
// leak in and beat together — the discriminator then carries no clean baud line
// and the carrier is mis-read as analog am/wfm (issue #648). When a neighbour
// sits inside that passband we halve the channel to 24 kHz (±12 kHz): the gentle
// decimation pushes the neighbour out to / past the passband edge, leaving the
// centre carrier dominant, while staying wide enough to keep the carrier's own
// energy intact (unlike a sharp brick-wall filter, which clips the constant-
// envelope carrier and spikes the discriminator). A lone carrier, or one whose
// nearest neighbour is already outside the wide passband, keeps the full channel
// so a genuinely wideband signal isn't truncated.
func classifyChannelRate(neighborHz uint32) uint32 {
	const narrow = surveyChannelRateHz / 2 // 24 kHz
	if neighborHz == 0 || neighborHz >= narrow {
		return surveyChannelRateHz
	}
	return narrow
}
