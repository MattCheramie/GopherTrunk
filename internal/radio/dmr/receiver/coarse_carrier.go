package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// coarseCarrierAcquirer performs a one-shot, frozen coarse carrier-offset
// acquisition in the COMPLEX-IQ domain, ahead of the FM discriminator — the
// "before-clock/freeze stage" the receiver's post-clock CoarseAFC doc names as
// the follow-up for grossly-mistuned dongles (issue #836).
//
// # Why the post-clock CoarseAFC is not enough on its own
//
// The receiver's CoarseAFC removes a carrier offset's constant DC bias from the
// recovered SYMBOL stream (post-clock, pre-slicer). That recentres the 4-level
// eye for the slicer, but the Mueller-Müller timing loop and the RRC matched
// filter still see the uncorrected, frequency-shifted signal — so its pull-in
// range is bounded to a few hundred Hz. Measured on the synthetic decode path,
// decode is clean to ~800 Hz, collapses by ~1.5 kHz, and floors at the
// mis-slice level for any offset past ~3 kHz. At 446 MHz that ceiling is ≈1.8
// ppm — far short of a typical RTL-SDR's tens-of-ppm tuner error (issue #836:
// the reporter's dongle never decoded 446.500 simplex, and kalibrate-rtl was
// unavailable to hand-set sdr.ppm).
//
// # What this stage does
//
// A constant carrier offset Δf appears at the FM discriminator output as a
// constant DC bias of 2π·Δf/Fs rad/sample; for a spectrally-balanced 4-level
// C4FM stream the data modulation averages to zero, so the MEAN of the
// discriminator output over a long-enough window is an unbiased estimate of Δf.
// The acquirer accumulates that mean over a fixed acquisition window, and once
// filled applies it ONCE to a complex NCO that de-rotates the IQ before the
// discriminator — recentring the signal in the channel so the timing loop and
// matched filter see a centred eye. The estimate is then FROZEN: a constant
// de-rotation is a single step the timing loop rides out, whereas a
// continuously-adapting IQ-domain correction would wander the discriminator
// output and destabilise symbol timing (the exact reason the residual AFC runs
// post-clock, and the same frozen-snapshot discipline the TETRA equalizers use).
//
// It is deliberately coarse and one-shot. The residual it leaves (tens of Hz)
// is well inside the post-clock CoarseAFC's comfortable range, which continues
// to track slow drift. A deadband keeps the stage from touching signals the
// post-clock AFC already handles, so a well-tuned dongle stays byte-identical.
//
// # Idle-channel robustness
//
// A conventional / simplex channel is silent before its first transmission, and
// the discriminator output on noise is ~zero-mean but not zero-variance — one
// window of noise can post a mean of a few hundred Hz by chance. So the stage
// does NOT freeze on a single window: it requires TWO consecutive windows whose
// estimates both clear the deadband AND agree with each other (a real tuner
// offset is constant window-to-window; noise is not). Windows that fall below
// the deadband — silence, or a well-tuned signal — simply re-arm, so the stage
// waits through idle noise and engages on the first ~two windows of a genuinely
// offset transmission rather than latching onto a noise fluctuation. This
// replaces an earlier single-window design that locked on the leading silence of
// a bursty channel and never corrected the transmission (found replaying a real
// simplex capture, issue #836). It uses no absolute-power gate — agreement, not
// dBFS, is the signal-vs-noise discriminator.
type coarseCarrierAcquirer struct {
	nco        *dsp.NCO
	sampleRate float64
	acqSamples int     // acquisition window length, in IQ samples
	deadbandHz float64 // minimum |estimate| worth correcting
	agreeHz    float64 // max window-to-window disagreement to accept a lock

	seen    int
	sumRad  float64 // running sum of discriminator output over the window (rad/sample)
	haveCan bool    // a prior above-deadband window is pending confirmation
	canHz   float64 // that window's estimate, awaiting an agreeing second window
	locked  bool
	offHz   float64 // frozen correction applied to the NCO (0 until/unless it engages)
}

// coarseAcqSymbols is the acquisition window length in symbol periods. ~512
// symbols (~107 ms at 4800 baud) is long enough that ordinary DMR traffic /
// idle / sync content averages spectrally flat — so the mean tracks the carrier
// offset, not a transient symbol-distribution bias — yet short enough that the
// two-window confirmation still locks early in a transmission (~2 windows,
// ~214 ms). Only that leading span of the first transmission after a Reset
// decodes uncorrected; every burst after the freeze is centred.
const coarseAcqSymbols = 512

// coarseAcqAgreeHz is the maximum window-to-window disagreement accepted when
// confirming a lock. A real tuner offset is constant to well within this;
// two independent noise windows agreeing this closely at a >deadband magnitude
// is vanishingly unlikely, which is what keeps the stage from latching on idle
// noise. 250 Hz is well below the deadband so a confirmed pair is unambiguous.
const coarseAcqAgreeHz = 250.0

// coarseAcqDeadbandHz is the smallest estimated offset the stage will correct.
// The post-clock CoarseAFC already decodes cleanly below ~800 Hz, so a 500 Hz
// deadband leaves that regime untouched (a well-tuned dongle never engages this
// stage and stays byte-identical) while still catching everything past the
// ~1.5 kHz decode cliff. It also guards against a mildly-unbalanced acquisition
// window: faking a 500 Hz mean needs a sustained ~0.77-of-full-scale symbol DC
// bias over the whole window, which real DMR content does not carry.
const coarseAcqDeadbandHz = 500.0

// newCoarseCarrierAcquirer builds a one-shot coarse acquirer for an IQ stream
// sampled at sampleRateHz with sps samples per symbol. Panics on non-positive
// arguments.
func newCoarseCarrierAcquirer(sampleRateHz, sps float64) *coarseCarrierAcquirer {
	if sampleRateHz <= 0 || sps <= 0 {
		panic("receiver: coarse carrier acquirer requires positive sample rate and sps")
	}
	return &coarseCarrierAcquirer{
		nco:        dsp.NewNCO(0, sampleRateHz), // identity until it engages
		sampleRate: sampleRateHz,
		acqSamples: int(math.Round(coarseAcqSymbols * sps)),
		deadbandHz: coarseAcqDeadbandHz,
		agreeHz:    coarseAcqAgreeHz,
	}
}

// Mix de-rotates iq by the frozen offset and returns the result (into dst when
// it has capacity). Until the stage engages the NCO is an identity mix, so the
// output is bit-identical to the input and downstream decode is unchanged.
func (c *coarseCarrierAcquirer) Mix(dst, iq []complex64) []complex64 {
	return c.nco.Mix(dst, iq)
}

// Observe accumulates the discriminator output toward the acquisition mean and,
// once the window is full, applies the one-shot frozen correction. disc is the
// FM-discriminator output for the (already-mixed) IQ of the same Process call.
// It returns engaged==true on the single call where a correction is applied, so
// the receiver can reset the downstream DSP to re-lock cleanly on the newly
// centred signal. A no-op (returns false) after the stage has locked.
func (c *coarseCarrierAcquirer) Observe(disc []float32) (engaged bool) {
	if c.locked {
		return false
	}
	for _, x := range disc {
		c.sumRad += float64(x)
		c.seen++
	}
	if c.seen < c.acqSamples {
		return false
	}
	// One window complete: estimate its offset and reset the accumulator for the
	// next window.
	meanRad := c.sumRad / float64(c.seen)
	offHz := meanRad * c.sampleRate / (2 * math.Pi)
	c.seen = 0
	c.sumRad = 0

	// Below the deadband — silence, or a signal the post-clock AFC already
	// handles. Drop any pending candidate and re-arm; never freeze here, so an
	// idle channel keeps waiting instead of latching on a noise fluctuation.
	if math.Abs(offHz) < c.deadbandHz {
		c.haveCan = false
		return false
	}
	// Above the deadband but not yet confirmed: hold it as a candidate and wait
	// for the next window. A real tuner offset repeats; a noise window won't.
	if !c.haveCan || math.Abs(offHz-c.canHz) > c.agreeHz {
		c.haveCan = true
		c.canHz = offHz
		return false
	}
	// Two consecutive windows agree above the deadband — a genuine, constant
	// carrier offset. Freeze the average and apply it once. The NCO shifts a
	// component at +offHz down to DC; the estimate is measured on the current
	// (identity, pre-lock) stream, so the full offset folds in at once. The
	// frequency step this applies mid-stream would otherwise leave the
	// Mueller-Müller timing loop parked at a bad phase (measured: a narrow band
	// of offsets re-locks to a wrong symbol instant), so the receiver resets the
	// downstream chain when engaged is true.
	c.locked = true
	c.offHz += 0.5 * (offHz + c.canHz)
	c.nco.SetOffset(c.offHz, c.sampleRate)
	return true
}

// OffsetHz reports the frozen coarse correction in hertz (0 until/unless the
// stage engages). Exposed for daemon logging so an operator sees the tuner
// offset the receiver pulled out.
func (c *coarseCarrierAcquirer) OffsetHz() float64 { return c.offHz }

// Reset clears the acquisition so the next transmission re-acquires from
// scratch. Call on stream re-tune / re-sync alongside the rest of the receiver.
func (c *coarseCarrierAcquirer) Reset() {
	c.nco.Reset()
	c.nco.SetOffset(0, c.sampleRate)
	c.seen = 0
	c.sumRad = 0
	c.haveCan = false
	c.canHz = 0
	c.locked = false
	c.offHz = 0
}
