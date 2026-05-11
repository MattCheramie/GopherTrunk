// Package receiver wires the IQ → π/4-DQPSK dibit chain that feeds
// the TETRA TMO control-channel state machine.
//
//	IQ samples
//	  → RRC matched filter (internal/dsp/demod.PiOver4DQPSK, α = 0.35)
//	  → naive decimation to one sample per symbol
//	  → π/4-rotated differential decode → 0..3 dibit
//	  → tetra.DibitSink
//
// TETRA TMO is true π/4-DQPSK (rotation = π/4) at 18000 sym/s with
// α = 0.35 RRC pulse shaping per ETSI EN 300 392-2. The receiver is
// deliberately minimal: matched filter + decoder + naive symbol-
// time decimation. Symbol-time clock recovery on complex IQ
// (Gardner / Mueller-Müller on |y|² envelope) is a follow-up; the
// connector that lands later wraps a proper timing-recovery loop
// around this once a real-air capture is available.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TETRA TMO on-air parameters (ETSI EN 300 392-2).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries
	// one dibit (2 bits) for a total channel capacity of 36 kbps
	// before TDMA slot multiplexing (4 slots / 56.67 ms frame).
	SymbolRate = 18000.0
	// RolloffAlpha is the RRC roll-off the matched filter is
	// designed around. 0.35 is the standard TETRA pulse shape.
	RolloffAlpha = 0.35
	// PulseSpanSymbols is the half-span of the RRC pulse on each
	// side of the symbol time.
	PulseSpanSymbols = 8
	// Rotation is the constellation offset for true π/4-DQPSK
	// (TETRA / IS-136). The PiOver4DQPSK helper subtracts this from
	// each phase delta before quadrant classification, so a clean
	// +π/4 phase delta lands squarely in the 0b00 quadrant.
	Rotation = math.Pi / 4
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate (36 kHz).
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. Required.
	DibitSink tetra.DibitSink
	// PulseSpanSymbols overrides the RRC half-span. <= 0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <= 0 uses RolloffAlpha.
	Alpha float64
}

// Receiver is the composed IQ → dibit pipeline.
type Receiver struct {
	dq        *demod.PiOver4DQPSK
	sps       int
	dibitSink tetra.DibitSink
	dibitBase int
	rxOffset  int

	matched []complex64
	dibits  []uint8
	pending []complex64
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
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (36000 Hz)")
	}
	span := opts.PulseSpanSymbols
	if span <= 0 {
		span = PulseSpanSymbols
	}
	alpha := opts.Alpha
	if alpha <= 0 {
		alpha = RolloffAlpha
	}
	return &Receiver{
		dq:        demod.NewPiOver4DQPSK(int(sps+0.5), span, alpha, Rotation),
		sps:       int(sps + 0.5),
		dibitSink: opts.DibitSink,
	}
}

// Process pushes one chunk of complex64 IQ samples through the
// matched filter, decimates to symbol time, and emits dibits via
// DibitSink.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	r.matched = r.dq.MatchedFilter(r.matched, iq)
	r.pending = append(r.pending, r.matched...)

	r.dibits = r.dibits[:0]
	var symbols []complex64
	for r.rxOffset < len(r.pending) {
		symbols = append(symbols, r.pending[r.rxOffset])
		r.rxOffset += r.sps
	}
	if len(symbols) == 0 {
		return
	}
	r.dibits = r.dq.Decode(r.dibits, symbols)
	r.dibitSink(r.dibits, r.dibitBase)
	r.dibitBase += len(r.dibits)

	drop := r.rxOffset - r.sps
	if drop < 0 {
		drop = 0
	}
	if drop > len(r.pending) {
		drop = len(r.pending)
	}
	if drop > 0 {
		copy(r.pending, r.pending[drop:])
		r.pending = r.pending[:len(r.pending)-drop]
		r.rxOffset -= drop
		if r.rxOffset < 0 {
			r.rxOffset = 0
		}
	}
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the matched filter + differential decoder shed their history and
// the DibitSink baseIdx restarts at 0.
func (r *Receiver) Reset() {
	r.dibitBase = 0
	r.dq.Reset()
	r.pending = r.pending[:0]
	r.rxOffset = 0
}
