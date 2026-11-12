// Package receiver wires the IQ → C4FM dibit chain that feeds the
// YSF control-channel state machine. It composes primitives that
// already live in internal/dsp + internal/radio/ysf:
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter + 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → 4-level symbol → 0..3 dibit (SymbolToDibit, local)
//	  → DibitSink → ysf.ControlChannel.Process
//
// YSF runs the same 4800-baud C4FM as P25 Phase 1, so the matched
// filter parameters (RRC α = 0.20, 4800 sym/s) are identical. Only
// the downstream framing (480-dibit frames, 20-dibit FSW, 100-dibit
// FICH) differs and that lives in the parent ysf package.
//
// The receiver is stateful and not safe for concurrent Process calls.
// Instantiate one per tuned frequency / per call chain. All primitives
// it composes own their own internal history so chunk boundaries do
// not corrupt the stream.
package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
)

// YSF on-air parameters (per Yaesu specification).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries one
	// dibit (2 bits) on the wire.
	SymbolRate = 4800.0
	// RolloffAlpha is the RRC roll-off the matched filter is designed
	// around. 0.20 matches the standard YSF / P25 Phase 1 receiver
	// pulse shape.
	RolloffAlpha = 0.20
	// PulseSpanSymbols is the half-span of the RRC pulse on each side
	// of the symbol time. 8 symbols (16 total) is the standard
	// receiver-side compromise between truncation noise and CPU cost.
	PulseSpanSymbols = 8
)

// Options configures a Receiver. Zero-valued fields fall back to
// sensible YSF defaults so the typical caller can write
//
//	r := receiver.New(receiver.Options{
//	    SampleRateHz: 48_000,
//	    DibitSink: func(d []uint8, base int) {
//	        cc.Process(d, base)
//	    },
//	})
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate.
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. Wire it into ysf.ControlChannel.Process to drive the
	// CC state machine. Required.
	DibitSink ysf.DibitSink
	// PulseSpanSymbols overrides the RRC half-span. <= 0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <= 0 uses RolloffAlpha.
	Alpha float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05,
	// which is appropriate for clean signals; raise for noisy /
	// drifting transmitters.
	ClockGain float64
	// DeviationHz is the peak frequency deviation of the C4FM
	// signal at symbol ±3 (1800 Hz on YSF, same as P25 P1 / NXDN).
	// Used to calibrate the slicer thresholds against the
	// FM-discriminator output level (slicer scale = 2π ·
	// DeviationHz / SampleRateHz). <= 0 falls back to the legacy
	// slicerScale = 1.0 — back-compat with fixtures that pre-scale
	// their FM levels.
	DeviationHz float64

	// EnableAFC subtracts the carrier offset's DC bias from the FM
	// discriminator. The 4-level eye's inner levels sit ±600 Hz from
	// centre, so an uncorrected offset of ~200 Hz already slices
	// wrongly (measured on a synthetic stream) — and a real radio is
	// routinely a few hundred hertz off. Off by default so existing
	// fixtures stay byte-identical; the voice chain turns it on.
	EnableAFC bool
	// EnableChannelFilter inserts a ±ChannelCutoffHz low-pass ahead of
	// the discriminator. The voice chain's front end barely band-limits
	// at an SDR rate of a few MS/s, so without it the discriminator sees
	// the whole ±24 kHz intermediate stream (the NXDN / DMR lesson). Off
	// by default; the voice chain turns it on.
	EnableChannelFilter bool
}

// ChannelCutoffHz is the optional channel filter's one-sided cutoff: half
// the 12.5 kHz channel spacing, as for DMR and NXDN96 (the same 4-FSK
// occupancy).
const ChannelCutoffHz = 6250.0

// channelFilterSpanSymbols sizes the channel FIR at 2·span·sps+1 taps
// (Kaiser β 8.6): 181 taps at 48 kHz, a ~1.5 kHz skirt.
const channelFilterSpanSymbols = 9

// Receiver is the composed IQ → dibit pipeline. Process is the only
// hot path; instantiate once per call chain and reuse.
type Receiver struct {
	chanFilt  *filter.FIR
	filtered  []complex64
	afc       *demod.CoarseAFC
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	dibitSink ysf.DibitSink
	dibitBase int

	// Reusable scratch slices so Process doesn't allocate per call.
	disc     []float32
	matched  []float32
	symbols  []float32
	symbols2 []float32
	sliced   []int8
	dibits   []uint8

	// Feed-forward symbol-timing acquisition (ArmTimingAcquisition).
	sps         float64
	sampleBase  int // matched-filter samples processed so far
	acqPending  bool
	acqSkip     int
	acqBuf      []float32
	acqStart    int
	acqHaveCand bool
	acqCandInst float64
	acqWindows  int
	timingSeeds int
}

// New constructs a Receiver from opts. Panics if SampleRateHz or
// DibitSink are unset, or the resulting samples-per-symbol is below 2
// (the Mueller-Müller loop's minimum).
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.DibitSink == nil {
		panic("receiver: DibitSink is required")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (9600 Hz)")
	}
	span := opts.PulseSpanSymbols
	if span <= 0 {
		span = PulseSpanSymbols
	}
	alpha := opts.Alpha
	if alpha <= 0 {
		alpha = RolloffAlpha
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	// Slicer thresholds: calibrate against the physical FM level
	// when DeviationHz is set (same fix as the P25 P1 / NXDN /
	// DMR / dPMR receivers). Legacy fixtures that pre-scale their
	// FM output to ±1 stay green via the fallback.
	slicerScale := 1.0
	if opts.DeviationHz > 0 {
		slicerScale = 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
	}

	var chanFilt *filter.FIR
	if opts.EnableChannelFilter {
		taps := 2*channelFilterSpanSymbols*int(sps+0.5) + 1
		chanFilt = filter.NewFIR(filter.LowpassKaiser(taps, ChannelCutoffHz/opts.SampleRateHz, 8.6))
	}
	// The AFC runs on the discriminator, ahead of the matched filter and
	// the timing loop: measured on a synthetic stream it decodes cleanly
	// from 0 to 1.2 kHz of offset there, at least as well as on the
	// recovered symbols (where NXDN and DMR run theirs).
	var afc *demod.CoarseAFC
	if opts.EnableAFC {
		afc = demod.NewCoarseAFC(sps)
	}

	return &Receiver{
		chanFilt:  chanFilt,
		afc:       afc,
		fm:        demod.NewFM(),
		mf:        demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale),
		clock:     sync.NewMuellerMuller(sps, gain),
		dibitSink: opts.DibitSink,
		sps:       sps,
	}
}

// Feed-forward symbol-timing acquisition, the DMR receiver's (issue #836)
// on a continuous carrier. The Mueller-Müller loop pulls a bad start phase
// in at gain·error per symbol, and from near a half symbol (its unstable
// equilibrium) that takes a long time: the scanner's YSF call gate, started
// on a transmission at ten sub-sample phases, found the first frame sync
// after 0.11 s at the best phases and 0.6–1.2 s at three of them (issue
// #1261) — longer than the scan window it has to open in. A caller that
// starts the receiver on a carrier that may already be up (the gate, the
// voice chain at a grant) arms an acquisition: the next
// timingAcqSkipSymbols symbols are skipped (the matched filter's
// transient), then windows of timingAcqSymbols symbols of matched-filter
// output are measured with sync.EstimateSymbolPhase, and the loop is seeded
// (MuellerMuller.SetPhase) once two consecutive windows agree within
// timingAcqAgreeSamples (one window is not evidence: a keyup transient can
// pass the estimator's gates a couple of samples off). After
// timingAcqMaxWindows disagreeing windows it gives up and the loop pulls in
// on its own, as before the acquisition existed. Off unless armed, so every
// existing stream is byte-identical.
const (
	timingAcqSkipSymbols  = 10
	timingAcqSymbols      = 96
	timingAcqAgreeSamples = 1.0
	timingAcqMaxWindows   = 4
)

// ArmTimingAcquisition schedules a feed-forward symbol-timing acquisition
// on the samples that follow. Call after New and after each Reset when the
// stream may already carry a transmission.
func (r *Receiver) ArmTimingAcquisition() {
	r.acqPending = true
	r.acqSkip = int(timingAcqSkipSymbols * r.sps)
	r.acqBuf = r.acqBuf[:0]
	r.acqHaveCand = false
	r.acqWindows = 0
}

// TimingSeeds is the number of feed-forward timing seeds applied so far.
func (r *Receiver) TimingSeeds() int { return r.timingSeeds }

// observeTimingAcq fills the armed acquisition window from this chunk's
// matched-filter output. When a window completes and its estimate is
// confirmed it returns the chunk index k at which the clock must be seeded
// (samples [0,k) are timed as before, [k,…) after the seed) and the seed
// value mu: the loop's next symbol instant, in samples past chunk sample
// k−1, on the estimated phase's grid.
func (r *Receiver) observeTimingAcq() (k int, mu float64, seed bool) {
	if !r.acqPending {
		return 0, 0, false
	}
	window := int(timingAcqSymbols * r.sps)
	for i := range r.matched {
		if r.acqSkip > 0 {
			r.acqSkip--
			continue
		}
		if len(r.acqBuf) == 0 {
			r.acqStart = r.sampleBase + i
		}
		r.acqBuf = append(r.acqBuf, r.matched[i])
		if len(r.acqBuf) < window {
			continue
		}
		tau, ok := sync.EstimateSymbolPhase(r.acqBuf, r.sps)
		start := r.acqStart
		r.acqBuf = r.acqBuf[:0]
		r.acqWindows++
		if !ok {
			if r.acqWindows >= timingAcqMaxWindows {
				r.acqPending = false
				r.acqHaveCand = false
				return 0, 0, false
			}
			continue
		}
		// The symbol instants sit at start + tau + n·sps. Confirm the
		// estimate against the previous window's before touching the loop.
		inst := float64(start) + tau
		if !r.acqHaveCand {
			r.acqHaveCand = true
			r.acqCandInst = inst
			continue
		}
		d := math.Mod(inst-r.acqCandInst, r.sps)
		if d > r.sps/2 {
			d -= r.sps
		} else if d < -r.sps/2 {
			d += r.sps
		}
		if math.Abs(d) > timingAcqAgreeSamples {
			r.acqCandInst = inst
			if r.acqWindows >= timingAcqMaxWindows {
				r.acqPending = false
				r.acqHaveCand = false
				return 0, 0, false
			}
			continue
		}
		r.acqPending = false
		r.acqHaveCand = false
		// The loop's next instant must be the first of those past the last
		// sample it will have processed before the seed, chunk sample i.
		last := float64(r.sampleBase + i)
		if inst <= last {
			inst += math.Ceil((last-inst)/r.sps) * r.sps
			if inst <= last {
				inst += r.sps
			}
		}
		r.timingSeeds++
		return i + 1, inst - last, true
	}
	return 0, 0, false
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more dibit batches may be emitted to DibitSink during the
// call, matching the standard "data-driven, callback per available
// batch" pattern the rest of the radio packages use.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	if r.chanFilt != nil {
		r.filtered = r.chanFilt.Process(r.filtered, iq)
		iq = r.filtered
	}
	r.disc = r.fm.Process(r.disc, iq)
	if r.afc != nil {
		r.afc.Process(r.disc)
	}
	r.matched = r.mf.MatchedFilter(r.matched, r.disc)
	if k, mu, seed := r.observeTimingAcq(); seed {
		// Seed the clock between the two halves of this chunk so every
		// symbol after the acquisition window is timed from the estimate.
		r.symbols = r.clock.Process(r.symbols, r.matched[:k])
		r.clock.SetPhase(mu)
		r.symbols2 = r.clock.Process(r.symbols2, r.matched[k:])
		r.symbols = append(r.symbols, r.symbols2...)
	} else {
		r.symbols = r.clock.Process(r.symbols, r.matched)
	}
	r.sampleBase += len(r.matched)
	if len(r.symbols) == 0 {
		return
	}
	r.sliced = r.mf.SliceMany(r.sliced, r.symbols)
	if cap(r.dibits) < len(r.sliced) {
		r.dibits = make([]uint8, len(r.sliced))
	} else {
		r.dibits = r.dibits[:len(r.sliced)]
	}
	for i, sym := range r.sliced {
		r.dibits[i] = SymbolToDibit(sym)
	}
	r.dibitSink(r.dibits, r.dibitBase)
	r.dibitBase += len(r.dibits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so the
// DibitSink baseIdx restarts at 0 for downstream consumers that track
// absolute dibit positions.
func (r *Receiver) Reset() {
	r.dibitBase = 0
	r.sampleBase = 0
	r.acqPending = false
	r.acqBuf = r.acqBuf[:0]
	r.acqHaveCand = false
	if r.afc != nil {
		r.afc.Reset()
	}
	if r.chanFilt != nil {
		r.chanFilt.Reset()
	}
	// FM discriminator's `last` is harmless to leave alone — the next
	// sample it processes will produce one slightly-wrong derivative,
	// which the matched filter smooths out.
}

// SymbolToDibit maps a C4FM slicer output ({-3, -1, +1, +3}) to a
// dibit value (0..3). Uses the same Gray-coded convention as
// P25 Phase 1 (TIA-102.BAAA): +3→01, +1→00, -1→10, -3→11. Other
// 4FSK protocols (DMR, NXDN) may map differently; YSF tracks the
// P25 / DSDcc convention pending real-air capture validation
// against the FSWPattern in the parent ysf package.
func SymbolToDibit(sym int8) uint8 {
	switch sym {
	case 1:
		return 0
	case 3:
		return 1
	case -1:
		return 2
	case -3:
		return 3
	}
	return 0
}
