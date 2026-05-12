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
}

// Receiver is the composed IQ → dibit pipeline. Process is the only
// hot path; instantiate once per call chain and reuse.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	dibitSink ysf.DibitSink
	dibitBase int

	// Reusable scratch slices so Process doesn't allocate per call.
	disc    []float32
	matched []float32
	symbols []float32
	sliced  []int8
	dibits  []uint8
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

	return &Receiver{
		fm:        demod.NewFM(),
		mf:        demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale),
		clock:     sync.NewMuellerMuller(sps, gain),
		dibitSink: opts.DibitSink,
	}
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more dibit batches may be emitted to DibitSink during the
// call, matching the standard "data-driven, callback per available
// batch" pattern the rest of the radio packages use.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	r.disc = r.fm.Process(r.disc, iq)
	r.matched = r.mf.MatchedFilter(r.matched, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.matched)
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
