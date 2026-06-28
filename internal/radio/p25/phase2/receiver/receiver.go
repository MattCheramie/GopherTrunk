// Package receiver wires the IQ → H-DQPSK dibit chain that feeds the
// P25 Phase 2 control-channel state machine.
//
//	IQ samples
//	  → RRC matched filter (internal/dsp/demod.PiOver4DQPSK)
//	  → naive decimation to one sample per symbol (offset by the
//	    RRC group delay so the centre tap lands on the eye)
//	  → π/8-rotated differential decode → 0..3 dibit
//	  → phase2.DibitSink
//
// P25 Phase 2 uses H-DQPSK at 6000 sym/s with α = 0.20 RRC pulse
// shaping. The constellation is rotated by π/8 from the standard
// π/4-DQPSK reference, which is what the `rotation` argument on the
// PiOver4DQPSK helper compensates for.
//
// The receiver is intentionally minimal: it composes the matched
// filter + decoder and emits one dibit per `sps` complex samples
// from the matched-filter output. Symbol-time clock recovery
// (Gardner-style on complex IQ, or eye-tracking on |y|² envelope)
// is a follow-up — the connector that lands later wraps a proper
// timing-recovery loop around this when a real-air capture is
// available to calibrate against.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"math"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// P25 Phase 2 H-DQPSK on-air parameters (TIA-102.BBAB).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries
	// one dibit (2 bits) for a total channel capacity of 12 kbps
	// before TDMA slot multiplexing.
	SymbolRate = 6000.0
	// RolloffAlpha is the RRC pulse-shape roll-off.
	RolloffAlpha = 0.20
	// PulseSpanSymbols is the half-span of the RRC pulse on each
	// side of the symbol time.
	PulseSpanSymbols = 8
	// Rotation is the constellation offset for H-DQPSK. The
	// PiOver4DQPSK helper subtracts this from each phase delta
	// before quadrant classification, so a clean +π/8 phase delta
	// lands squarely in the 0b00 quadrant.
	Rotation = math.Pi / 8
)

// Carrier-frequency recovery for the ClockGardner path (issue #813). The
// differential decoder removes a constant carrier *phase* but not the
// per-symbol *rotation* (2π·Δf/SymbolRate) a real (non-TCXO) tuner's
// residual offset Δf imposes; at 6000 baud ~750 Hz already rotates every
// symbol a full π/4 into the wrong quadrant, so the 20-dibit outbound sync
// never correlates and superframes never lock. This mirrors the carrier
// recovery the Phase 1 CQPSK path gained in issue #492 (the C4FM path has
// CoarseAFC for the same reason): a one-shot coarse seed removes the bulk
// offset with the NCO, then a fine Costas loop tracks the residual + drift.
//
// On a clean, zero-offset stream the seed estimates ~0 Hz, the NCO is an
// identity mix, the AGC is a unit scalar and Costas drives to zero — so the
// path is a no-op and the synthesized ClockGardner fixtures are unaffected.
const (
	// seedMinSamples is the raw-IQ count accumulated before the one-shot
	// coarse carrier estimate fires. Production feeds ~160-200 samples per
	// Process call, so the estimate must accumulate across calls rather than
	// key off one chunk; 4096 samples (~512 symbols) gives the
	// autocorrelation estimate a low-variance average.
	seedMinSamples = 4096
	// seedMaxLag is the largest sub-symbol autocorrelation lag the coarse
	// estimator fits a phase-ramp slope through. Lags must stay well below
	// sps (8) so the pairs are within-symbol; 4 matches the CQPSK path.
	seedMaxLag = 4
	// seedMinCoherence is the minimum lag-1 autocorrelation coherence for the
	// coarse estimate to be trusted; below it the stream is too weak/noisy to
	// seed from and the NCO is left identity for the Costas loop to acquire.
	seedMinCoherence = 0.30
	// seedClampHz bounds the applied coarse seed to ±6 kHz of DC — the
	// channel passband; a larger residual is a mis-tuned / un-channelised
	// stream the upstream DDC + PPM correction should have handled.
	seedClampHz = 6000.0
	// agc* normalise the matched-filter output ahead of the amplitude-
	// sensitive Gardner timing-error detector (the issue #275 lesson on the
	// CQPSK path). Same values as the CQPSK path — gain normalisation is
	// baud-independent.
	agcReference = 0.95
	agcRate      = 1e-3
	agcMaxGain   = 1e4
	// costas* tune the fine carrier-tracking loop. The CQPSK path used
	// 120 Hz at 4800 baud (~2.5% of baud); 6000·0.025 = 150 Hz keeps the
	// same normalised loop bandwidth. The loop's pull-in is only
	// ±SymbolRate/8 = ±750 Hz, which is why the coarse seed must precede it.
	costasLoopBWHz = 150.0
	costasDamping  = 0.707
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate (12 kHz).
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. Required.
	DibitSink phase2.DibitSink
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
}

// ClockMode selects how the receiver decimates the matched-filter
// output to one sample per symbol.
//
//   - ClockNaive (default): picks every sps-th sample starting at
//     an offset advanced by sps each emission. Works on
//     synthesized signals whose symbol-time peaks fall at fixed
//     sample positions; matches the receiver's pre-Gardner
//     behaviour exactly.
//
//   - ClockGardner: routes the matched-filter output through the
//     Gardner symbol-timing-recovery loop in internal/dsp/sync —
//     non-data-aided, complex-valued, with cross-call state for
//     chunked streams. Useful for noisier on-air captures where
//     the symbol clock isn't perfectly aligned with the SDR
//     sample clock.
//
// Wiring Gardner into the pipeline is the follow-up the Gardner
// primitive PR (#128) called out; the choice is per-receiver and
// runtime-configurable so the existing test fixtures (which
// depend on fixed sample alignment) keep passing under
// ClockNaive while ClockGardner becomes the recommended default
// for live SDR captures.
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
	dibitSink phase2.DibitSink
	dibitBase int
	// rxOffset is the absolute sample index where the next symbol
	// centre should be picked from the matched-filter output. It
	// advances by sps each time we emit a dibit, and wraps when the
	// pending matched-filter buffer is consumed.
	rxOffset int

	clockMode ClockMode
	gardner   *sync.Gardner

	// Carrier-frequency recovery (ClockGardner only; nil under ClockNaive).
	// nco removes the coarse seed from the raw IQ; costas tracks the residual
	// on the symbol stream; agc normalises ahead of Gardner. See the
	// carrier-recovery const block and issue #813.
	nco     *dsp.NCO
	agc     *dsp.AGC
	costas  *sync.QPSKCostas
	fs      float64 // IQ sample rate, for the NCO seed and Hz reporting
	seeded  bool    // coarse carrier seed applied
	seedHz  float64 // coarse carrier offset the NCO removes (Hz)
	seedBuf []complex64

	// Scratch buffers reused across calls.
	rotated []complex64 // NCO-de-rotated IQ, fed to the matched filter
	matched []complex64
	dibits  []uint8
	symbols []complex64
	// pending holds matched-filter samples from prior Process calls
	// that didn't align with a symbol centre and are needed for the
	// next decimation step (ClockNaive). Under ClockGardner the
	// Gardner loop manages its own cross-call tail internally.
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
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (12000 Hz)")
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
		r.fs = opts.SampleRateHz
		r.nco = dsp.NewNCO(0, opts.SampleRateHz)
		r.agc = dsp.NewAGC(agcReference, agcRate, agcMaxGain)
		// Rotation-aware: H-DQPSK's π/8 constellation locks the detector to
		// 4·(π/8) = π/2, not the π/4 family's π. The wrong constant would
		// settle a π/8 per-symbol bias that halves the decision margin.
		r.costas = sync.NewQPSKCostasForRotation(SymbolRate, costasLoopBWHz, costasDamping, Rotation)
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
	r.dibits = r.dibits[:0]
	r.symbols = r.symbols[:0]

	if r.clockMode == ClockGardner {
		// Coarse carrier seed (issue #813): a real tuner's residual offset
		// spins the differential constellation so the sync never correlates.
		// Estimate it once from accumulated raw IQ and tune it out with the
		// NCO before the matched filter; the Costas loop below then tracks the
		// residual and drift. Production delivers ~160-200 samples per call,
		// so the estimate accumulates across calls and fires once. Until it
		// fires the NCO is an identity mix, so a centred/zero-offset stream is
		// unaffected.
		if !r.seeded {
			r.seedBuf = append(r.seedBuf, iq...)
			if len(r.seedBuf) >= seedMinSamples {
				hz := estimateCarrierSeedHz(r.seedBuf, r.fs, seedMaxLag)
				if hz > seedClampHz {
					hz = seedClampHz
				} else if hz < -seedClampHz {
					hz = -seedClampHz
				}
				r.seedHz = hz
				r.nco.SetOffset(hz, r.fs)
				r.seeded = true
				r.seedBuf = nil
				// The pre-seed samples ran through an identity NCO, so the
				// Costas frequency integrator wound toward the uncorrected
				// offset; reset it to re-acquire on the now-centred signal.
				// Gardner is left running — it re-locks within a few symbols
				// and resetting it would discard symbol timing.
				r.costas.Reset()
			}
		}
		r.rotated = r.nco.Mix(r.rotated, iq)
		r.matched = r.dq.MatchedFilter(r.matched, r.rotated)
		// Normalise amplitude ahead of the gain-sensitive Gardner TED.
		r.matched = r.agc.Process(r.matched, r.matched)
		// Gardner manages its own cross-call tail state, so the
		// receiver hands each matched-filter chunk straight in.
		r.symbols = r.gardner.Process(r.symbols, r.matched)
		// Fine carrier tracking: de-rotate each symbol so the residual
		// per-symbol rotation the coarse seed left behind is removed before
		// the differential decode. The Costas error detector is four-fold
		// differential, so the π/8 constellation offset cancels in it; the
		// downstream Decode applies the static π/8 subtraction.
		for i, y := range r.symbols {
			r.symbols[i] = r.costas.Update(y)
		}
	} else {
		r.matched = r.dq.MatchedFilter(r.matched, iq)
		// Naive decimation: pick every sps-th sample starting at
		// rxOffset, preserving cross-call state in r.pending.
		r.pending = append(r.pending, r.matched...)
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
	if r.nco != nil {
		r.nco.Reset()
		r.nco.SetOffset(0, r.fs) // identity until the seed re-fires
	}
	if r.agc != nil {
		r.agc.Reset()
	}
	if r.costas != nil {
		r.costas.Reset()
	}
	r.seeded = false
	r.seedHz = 0
	r.seedBuf = nil
}

// CarrierOffsetHz reports the carrier-recovery loop's current estimate of
// the residual carrier-frequency offset in Hz: the coarse NCO seed plus the
// fine Costas loop's tracked residual. Returns 0 on the ClockNaive path
// (no carrier recovery). Field diagnostic for issue #813 — a stable small
// value with sync still not locking points away from carrier offset (e.g.
// spectral inversion) and toward a different root cause.
func (r *Receiver) CarrierOffsetHz() float64 {
	if r.costas == nil {
		return 0
	}
	return r.seedHz + r.costas.OffsetHz()
}

// estimateCarrierSeedHz estimates the residual carrier offset (Hz) of an
// oversampled H-DQPSK stream from its sub-symbol-lag autocorrelation. A pure
// carrier offset Δf rotates the stream by a constant k·Δω at autocorrelation
// lag k (in samples), so angle(R_k) = k·Δω is linear in k; a least-squares
// slope through lags 1..maxLag is a lower-variance estimate than a single lag.
// The lags are sub-symbol (< sps), so most sample pairs fall within one
// RRC-real-pulse symbol where the only per-sample phase difference is the
// carrier rotation: the per-symbol π/8 constellation rotation and the
// zero-mean random data contribute only at the 1-in-sps symbol boundaries and
// largely average out. This is the estimator the CQPSK path uses (issue #492);
// a periodogram reads the rotation-shifted spectral centroid of a modulated
// signal instead and mis-estimates it. Returns 0 on empty / incoherent input,
// leaving the NCO identity for the Costas loop to acquire the in-range offset.
func estimateCarrierSeedHz(x []complex64, fs float64, maxLag int) float64 {
	if len(x) <= maxLag || fs <= 0 || maxLag < 1 {
		return 0
	}
	var energy float64
	for _, s := range x {
		energy += float64(real(s))*float64(real(s)) + float64(imag(s))*float64(imag(s))
	}
	if energy == 0 {
		return 0
	}
	ang := make([]float64, maxLag+1)
	var coh float64
	for k := 1; k <= maxLag; k++ {
		var accR, accI float64
		for n := k; n < len(x); n++ {
			ar, ai := float64(real(x[n])), float64(imag(x[n]))
			br, bi := float64(real(x[n-k])), float64(imag(x[n-k]))
			accR += ar*br + ai*bi
			accI += ai*br - ar*bi
		}
		if k == 1 {
			coh = math.Hypot(accR, accI) / energy
		}
		ang[k] = math.Atan2(accI, accR)
	}
	if coh < seedMinCoherence {
		return 0
	}
	// Unwrap each lag's angle around k·(lag-1 angle), then least-squares fit a
	// slope θ (rad/sample) through (k, angle_k) constrained through the origin.
	a1 := ang[1]
	var num, den float64
	for k := 1; k <= maxLag; k++ {
		ak := float64(k)*a1 + math.Remainder(ang[k]-float64(k)*a1, 2*math.Pi)
		num += float64(k) * ak
		den += float64(k * k)
	}
	slope := num / den
	return slope * fs / (2 * math.Pi)
}
