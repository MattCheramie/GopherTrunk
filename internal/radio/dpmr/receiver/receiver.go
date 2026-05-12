// Package receiver wires the IQ → C4FM dibit chain that feeds the
// dPMR Mode 3 control-channel state machine.
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter + 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → 4-level symbol → 0..3 dibit (SymbolToDibit, local)
//	  → dpmr.DibitSink
//
// dPMR Mode 3 runs 4-level C4FM at 2400 sym/s — half the symbol
// rate of P25 Phase 1 / DMR / YSF — with α = 0.20 RRC pulse
// shaping. The downstream framing (FS1 / FS2 / FS3 24-dibit sync,
// 80-bit CSBK signalling) lives in the parent dpmr package.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
)

// dPMR Mode 3 on-air parameters (per ETSI TS 102 658).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries
	// one dibit (2 bits) for a total channel capacity of 4800 bps.
	// Half the P25 P1 / DMR / YSF rate, matching the 6.25 kHz
	// channel spacing dPMR targets.
	SymbolRate = 2400.0
	// RolloffAlpha matches the standard dPMR receiver pulse shape.
	RolloffAlpha = 0.20
	// PulseSpanSymbols is the half-span of the RRC pulse on each
	// side of the symbol time.
	PulseSpanSymbols = 8
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate.
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. Required.
	DibitSink dpmr.DibitSink
	// PulseSpanSymbols overrides the RRC half-span. <= 0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <= 0 uses RolloffAlpha.
	Alpha float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
	// DeviationHz is the peak frequency deviation of the C4FM
	// signal at symbol ±3 (900 Hz on dPMR Mode 3 — half of P25 /
	// DMR / YSF, matching the 6.25 kHz channel spacing). Used to
	// calibrate the slicer thresholds against the FM-discriminator
	// output level (slicer scale = 2π · DeviationHz / SampleRateHz).
	// <= 0 falls back to the legacy slicerScale = 1.0 for fixtures
	// that pre-scale their FM levels.
	DeviationHz float64
}

// Receiver is the composed IQ → dibit pipeline.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	dibitSink dpmr.DibitSink
	dibitBase int

	disc    []float32
	matched []float32
	symbols []float32
	sliced  []int8
	dibits  []uint8
}

// New constructs a Receiver. Panics if SampleRateHz or DibitSink are
// unset, or the resulting samples-per-symbol is below 2.
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.DibitSink == nil {
		panic("receiver: DibitSink is required")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (4800 Hz)")
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
	// Slicer thresholds: calibrate against physical FM level when
	// DeviationHz is set (same fix as the P25 P1 / NXDN / DMR
	// receivers). Legacy "FM-output-normalised-to-±1" fixtures
	// stay green via the fallback.
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

// Reset returns the receiver to its initial state.
func (r *Receiver) Reset() {
	r.dibitBase = 0
}

// SymbolToDibit maps a C4FM slicer output ({-3, -1, +1, +3}) to a
// dibit value (0..3). Uses the same Gray-coded convention as the
// P25 Phase 1 / DMR / NXDN / YSF receivers: +3 → 01, +1 → 00,
// -1 → 10, -3 → 11. The dPMR symbol-to-dibit mapping in ETSI
// TS 102 658 matches this convention; the mapping is pinned by
// unit test so a future spec re-read doesn't silently desync from
// the FS1 / FS2 / FS3 patterns in the parent dpmr package.
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
