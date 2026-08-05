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
	"github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"
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

// Equalizer defaults (see Options.EnableEqualizer). Applied by New when the
// corresponding Options field is <= 0. Validated on the reporter's captures
// (soft-decision TCH/S yield 410→778, ~1.9×) and the synthetic multipath
// fixtures: snapEvery=200 converges within a few hundred symbols; mu=6e-3 tracks
// a same-carrier multipath channel without over-shooting the near-unit-modulus
// symbols the AFC hands the equalizer.
const (
	DefaultEqualizerTaps     = 11
	DefaultEqualizerMu       = 6e-3
	DefaultEqualizerSnapshot = 200
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
	// SoftSink, when non-nil, receives the complex π/4-DQPSK differential
	// (s·conj(last)) for each symbol, aligned 1:1 with the dibits emitted
	// to DibitSink and carrying the same baseIdx. It is the soft
	// information for soft-decision channel decoding (the two on-air bits'
	// LLRs are Im and Re of the differential). Emitted just before the
	// matching DibitSink call. nil ⇒ no soft emission, zero overhead.
	SoftSink func(diffs []complex64, baseIdx int)
	// SymbolSink, when non-nil, receives the RAW post-timing/AFC/equalizer
	// complex symbols (before the differential decode), aligned 1:1 with the
	// dibits emitted to DibitSink and carrying the same baseIdx. Unlike the
	// SoftSink differential (a nonlinear product s·conj(last), in which the
	// channel is no longer a clean convolution), the symbol stream is where a
	// linear channel IS a convolution — so it is the input a training-sequence
	// equalizer (tetra.TrafficExtractor.EnableLMSEqualizer / equalizer.SnapshotLMS)
	// must train on and equalize per burst. Emitted just before the matching
	// DibitSink call. nil ⇒ no symbol emission, zero overhead. See issue #1001.
	SymbolSink func(symbols []complex64, baseIdx int)
	// EnableDCBlock inserts a first-order complex DC-removal high-pass at the
	// very top of Process, ahead of the channel filter, to strip the front-end
	// DC spur that sits on a same-carrier TETRA voice channel (0 Hz offset) and
	// biases the π/4-DQPSK differential decode under heavy multislot load. Safe
	// for TETRA because π/4-DQPSK has a spectral null at DC; off by default and
	// intended for the VOICE receivers only — never the control-channel path,
	// which must not disturb steady-state CC decode. See dcblock.go.
	EnableDCBlock bool
	// EnableEqualizer inserts a blind CMA adaptive equalizer between symbol-
	// timing recovery and the differential decoder. It inverts the linear
	// channel (multipath / ISI / band-edge group delay) that smears the
	// π/4-DQPSK constellation on real captures — the demod-side gap that made
	// otherwise-clean concurrent-load recordings garble. On the reporter's
	// captures it roughly doubles CRC-valid TCH/S burst yield with no regression
	// on clean captures. Off by default (a near-noop on a clean single-carrier
	// synth, but non-zero cost); recommended for live / replayed captures. See
	// internal/dsp/equalizer.SnapshotCMA.
	EnableEqualizer bool
	// EqualizerTaps overrides the CMA tap count (forced odd; <=0 ⇒ default).
	EqualizerTaps int
	// EqualizerMu overrides the CMA step size (<=0 ⇒ default).
	EqualizerMu float64
	// EqualizerSnapshot overrides the symbols between frozen-tap snapshots
	// (<=0 ⇒ default). Trades convergence speed against phase continuity: the
	// applied filter is frozen between snapshots so each burst decodes with a
	// constant phase, and the one symbol straddling a snapshot is absorbed by
	// the FEC. See equalizer.go.
	EqualizerSnapshot int
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

	clockMode  ClockMode
	gardner    *sync.Gardner
	afc        *carrierAFC
	chanFilt   *filter.FIR
	softSink   func(diffs []complex64, baseIdx int)
	symbolSink func(symbols []complex64, baseIdx int)
	eq         *equalizer.SnapshotCMA
	equalized  []complex64
	dcBlk      *dcBlock
	dcBuf      []complex64

	matched   []complex64
	filtered  []complex64
	dibits    []uint8
	diffs     []complex64
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
		dq:         demod.NewPiOver4DQPSK(int(sps+0.5), span, alpha, Rotation),
		sps:        int(sps + 0.5),
		dibitSink:  opts.DibitSink,
		softSink:   opts.SoftSink,
		symbolSink: opts.SymbolSink,
		clockMode:  opts.ClockMode,
	}
	if r.clockMode == ClockGardner {
		gain := opts.GardnerGain
		if gain <= 0 {
			gain = 0.03
		}
		r.gardner = sync.NewGardner(float64(r.sps), gain)
	}
	if opts.EnableDCBlock {
		r.dcBlk = &dcBlock{}
	}
	if opts.EnableAFC {
		r.afc = newCarrierAFC(r.sps)
	}
	if opts.EnableChannelFilter {
		fc := ChannelCutoffHz / opts.SampleRateHz
		taps := 2*channelFilterSpanSymbols*r.sps + 1 // delay = span*sps = whole symbols
		r.chanFilt = filter.NewFIR(filter.LowpassKaiser(taps, fc, channelFilterBeta))
	}
	if opts.EnableEqualizer {
		taps := opts.EqualizerTaps
		if taps <= 0 {
			taps = DefaultEqualizerTaps
		}
		mu := opts.EqualizerMu
		if mu <= 0 {
			mu = DefaultEqualizerMu
		}
		snap := opts.EqualizerSnapshot
		if snap <= 0 {
			snap = DefaultEqualizerSnapshot
		}
		r.eq = equalizer.NewSnapshotCMA(taps, float32(mu), snap)
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
	if r.dcBlk != nil {
		// Strip the front-end DC spur before any filtering so it never enters the
		// matched filter / AFC / differential decode. TETRA-only (voice path); a
		// no-op numerically on a clean, DC-free synthetic fixture.
		r.dcBuf = r.dcBlk.process(r.dcBuf, iq)
		iq = r.dcBuf
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
		// The AFC's coarse (wide-range) stage reads the pre-matched-filter
		// stream, which is only free of adjacent carriers when the channel
		// filter ran; without it, pass nil so the AFC stays fine-only rather
		// than risk locking the coarse estimate onto a neighbour. When the
		// filter is on, iq still holds that channel-filtered stream here,
		// sample-aligned with r.matched.
		var wide []complex64
		if r.chanFilt != nil {
			wide = iq
		}
		r.derotated = r.afc.Process(r.derotated, r.matched, wide)
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
	if r.eq != nil {
		r.equalized = r.equalized[:0]
		for _, s := range r.symbols {
			r.equalized = append(r.equalized, r.eq.Process(s))
		}
		r.symbols = r.equalized
	}
	if r.softSink != nil {
		// Emit the complex differential (soft info) just before the
		// matching dibits, both keyed by r.dibitBase.
		r.dibits, r.diffs = r.dq.DecodeBoth(r.dibits, r.diffs, r.symbols)
		r.softSink(r.diffs, r.dibitBase)
	} else {
		r.dibits = r.dq.Decode(r.dibits, r.symbols)
	}
	if r.symbolSink != nil {
		// Emit the raw pre-differential symbols (the linear-channel domain a
		// training-sequence equalizer works in), aligned 1:1 with the dibits and
		// keyed by the same base. DecodeBoth/Decode do not mutate r.symbols, so it
		// is still the symbol stream that produced these dibits. Fired before the
		// DibitSink so the extractor's StashSymbols lands before Process consumes
		// the matching dibit block.
		r.symbolSink(r.symbols, r.dibitBase)
	}
	r.dibitSink(r.dibits, r.dibitBase)
	r.dibitBase += len(r.dibits)
}

// CarrierOffsetHz reports the residual carrier-frequency offset the AFC most
// recently estimated, in Hz. Zero when the AFC is disabled (EnableAFC off).
// Diagnostic only — surfaced in the connector's periodic decode-status line so
// an operator can see how far off-centre the tuned control channel sits.
func (r *Receiver) CarrierOffsetHz() float64 {
	if r.afc == nil {
		return 0
	}
	return r.afc.OffsetHz(SymbolRate)
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
	if r.dcBlk != nil {
		r.dcBlk.Reset()
	}
	if r.gardner != nil {
		r.gardner.Reset()
	}
	if r.afc != nil {
		r.afc.Reset()
	}
	if r.chanFilt != nil {
		r.chanFilt.Reset()
	}
	if r.eq != nil {
		r.eq.Reset()
	}
}
