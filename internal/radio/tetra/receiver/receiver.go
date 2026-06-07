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
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
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
	// ChannelCutoffHz is the one-sided cutoff of the optional channel-
	// select filter. A TETRA channel is 25 kHz wide (occupied ≈ ±12 kHz
	// at α = 0.35), but the channelised stream is much wider (the live
	// DDC decimates to 144 kHz, a ±72 kHz passband), so adjacent
	// carriers leak in and the RRC matched filter alone does not reject
	// them. A ≈±12.5 kHz channel filter ahead of the matched filter
	// removes them — measured to cut the on-air symbol error rate by an
	// order of magnitude (issue #553). 15 kHz keeps the passband flat
	// across the wanted signal's ±12.15 kHz occupied band (so it is a
	// noop on a clean single-carrier capture) while its sharp skirt
	// rejects a neighbour ≥~20 kHz away; the on-air win is flat from
	// ~12.5–16.5 kHz, so the exact cutoff is not critical.
	ChannelCutoffHz = 15_000.0
)

// channelFilterSpanSymbols sets the channel-select FIR length to
// 2*span*sps+1 taps so its group delay (span*sps samples) is a whole
// number of symbols — the same trick the RRC matched filter uses. A
// fractional-symbol delay would shift the naive decimator off the
// symbol centres and disrupt Gardner acquisition. 9 symbols gives a
// ~145-tap filter at the 8-sps production rate: a sharp enough skirt to
// reject a neighbour ~20 kHz away.
const channelFilterSpanSymbols = 9

// channelFilterBeta is the Kaiser shape for the channel-select FIR
// (matches the halfband design's ~70 dB stopband).
const channelFilterBeta = 8.6

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
	// ClockMode selects the symbol-time recovery strategy. See
	// the ClockMode type doc for the trade-offs. Zero value is
	// ClockNaive (matches the receiver's pre-Gardner behaviour).
	ClockMode ClockMode
	// GardnerGain overrides the Gardner loop step (default 0.03,
	// applied only when ClockMode is ClockGardner).
	GardnerGain float64
	// EnableAFC turns on the residual-carrier AFC (carrierAFC) between
	// symbol-timing recovery and differential decode. The live DDC has
	// no AFC, so a channel that is not perfectly centred leaves a
	// constant per-symbol phase offset that biases every dibit; the AFC
	// removes it. Off by default so sample-aligned synthesized fixtures
	// (zero offset) are byte-unchanged. Recommended for live / replayed
	// captures.
	EnableAFC bool
	// EnableChannelFilter inserts a ≈±ChannelCutoffHz channel-select
	// low-pass ahead of the matched filter, rejecting adjacent carriers
	// that the wide channelised passband admits. Off by default (a
	// near-noop on a clean single-carrier synth); recommended for live /
	// replayed captures. See ChannelCutoffHz.
	EnableChannelFilter bool
}

// ClockMode selects how the receiver decimates the matched-filter
// output to one sample per symbol. Same enum / semantics as the
// P25 Phase 2 receiver's ClockMode:
//
//   - ClockNaive (default): every sps-th sample. Matches the
//     receiver's pre-Gardner behaviour exactly.
//   - ClockGardner: routes through the Gardner symbol-timing-
//     recovery loop in internal/dsp/sync. Recommended for noisier
//     on-air captures.
type ClockMode uint8

const (
	ClockNaive ClockMode = iota
	ClockGardner
)

// ParseClockMode maps a config / user-facing string into a
// ClockMode. Recognised values (case-insensitive): "" / "gardner" /
// "on" / "true" / "1" → ClockGardner (the new default — Gardner
// timing-recovery loop, recommended for live SDR captures); "naive"
// / "off" / "false" / "0" → ClockNaive (pre-Gardner behaviour,
// preserved for tests using sample-aligned synthesized IQ
// fixtures). Unknown strings return ClockGardner with `ok = false`.
func ParseClockMode(s string) (ClockMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ClockGardner, true
	case "gardner", "on", "true", "1":
		return ClockGardner, true
	case "naive", "off", "false", "0":
		return ClockNaive, true
	default:
		return ClockGardner, false
	}
}

// Receiver is the composed IQ → dibit pipeline.
type Receiver struct {
	dq        *demod.PiOver4DQPSK
	sps       int
	dibitSink tetra.DibitSink
	dibitBase int
	rxOffset  int

	clockMode ClockMode
	gardner   *sync.Gardner
	afc       *carrierAFC
	chanFilt  *filter.FIR

	matched   []complex64
	filtered  []complex64
	dibits    []uint8
	symbols   []complex64
	derotated []complex64
	pending   []complex64
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
	r := &Receiver{
		dq:        demod.NewPiOver4DQPSK(int(sps+0.5), span, alpha, Rotation),
		sps:       int(sps + 0.5),
		dibitSink: opts.DibitSink,
		clockMode: opts.ClockMode,
	}
	if r.clockMode == ClockGardner {
		gain := opts.GardnerGain
		if gain <= 0 {
			gain = 0.03
		}
		r.gardner = sync.NewGardner(float64(r.sps), gain)
	}
	if opts.EnableAFC {
		r.afc = newCarrierAFC(r.sps)
	}
	if opts.EnableChannelFilter {
		fc := ChannelCutoffHz / opts.SampleRateHz
		taps := 2*channelFilterSpanSymbols*r.sps + 1 // delay = span*sps = whole symbols
		r.chanFilt = filter.NewFIR(filter.LowpassKaiser(taps, fc, channelFilterBeta))
	}
	return r
}

// Process pushes one chunk of complex64 IQ samples through the
// matched filter, decimates to symbol time, and emits dibits via
// DibitSink.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	if r.chanFilt != nil {
		// Reject adjacent carriers in the wide channelised passband
		// before matched filtering (issue #553).
		r.filtered = r.chanFilt.Process(r.filtered, iq)
		iq = r.filtered
	}
	r.matched = r.dq.MatchedFilter(r.matched, iq)
	r.dibits = r.dibits[:0]
	r.symbols = r.symbols[:0]

	// Remove the residual carrier offset BEFORE timing recovery: a
	// spinning constellation corrupts the Gardner timing metric (issue
	// #553). The AFC buffers into fixed blocks, so it may emit fewer
	// matched samples than it consumed.
	matched := r.matched
	if r.afc != nil {
		r.derotated = r.afc.Process(r.derotated, r.matched)
		matched = r.derotated
		if len(matched) == 0 {
			return
		}
	}

	if r.clockMode == ClockGardner {
		r.symbols = r.gardner.Process(r.symbols, matched)
	} else {
		r.pending = append(r.pending, matched...)
		for r.rxOffset < len(r.pending) {
			r.symbols = append(r.symbols, r.pending[r.rxOffset])
			r.rxOffset += r.sps
		}
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
	if len(r.symbols) == 0 {
		return
	}
	r.dibits = r.dq.Decode(r.dibits, r.symbols)
	r.dibitSink(r.dibits, r.dibitBase)
	r.dibitBase += len(r.dibits)
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
	if r.gardner != nil {
		r.gardner.Reset()
	}
	if r.afc != nil {
		r.afc.Reset()
	}
	if r.chanFilt != nil {
		r.chanFilt.Reset()
	}
}
