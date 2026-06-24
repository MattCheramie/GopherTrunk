package filter

import "math"

// Biquad is a second-order IIR section (the RBJ "Audio EQ Cookbook"
// biquad) operating on float64 PCM in place. It backs the band-limiting
// high-pass / low-pass stages of the optional voice enhancement chain
// (internal/voice/mbe.VoiceEnhancer) — the rival decoders OP25 and
// Trunk Recorder band-limit their output to the telephone band to trim
// rumble and quantization hiss, and this is the primitive that lets the
// enhancer do the same.
//
// The transfer function is the bilinear-transformed analog prototype
//
//	H(s) = 1 / (1 + s/Q + s²)
//
// realized as the difference equation
//
//	y[n] = b0·x[n] + b1·x[n−1] + b2·x[n−2] − a1·y[n−1] − a2·y[n−2]
//
// with the coefficients normalized so a0 = 1. The section runs as a
// transposed direct-form II, which keeps the two state words
// (z1, z2) numerically well-behaved at the low corner frequencies the
// voice chain uses (a 250 Hz high-pass at 8 kHz).
//
// A Biquad is continuous across Process calls — the caller keeps one
// per stream and Resets it only on stream re-sync, exactly like
// mbe.DCBlock. It is NOT safe for concurrent Process calls.
type Biquad struct {
	b0, b1, b2 float64
	a1, a2     float64
	z1, z2     float64 // transposed-DF2 state
}

// newBiquadRBJ builds a normalized biquad from un-normalized RBJ
// coefficients (a0 divides everything out so the difference equation
// can run without a per-sample divide).
func newBiquadRBJ(b0, b1, b2, a0, a1, a2 float64) *Biquad {
	return &Biquad{
		b0: b0 / a0, b1: b1 / a0, b2: b2 / a0,
		a1: a1 / a0, a2: a2 / a0,
	}
}

// butterworthQ is the Q of a maximally-flat (Butterworth) second-order
// section: 1/√2. A single such section gives a gentle 12 dB/oct roll-off
// with no passband ripple — the right shape for unobtrusive band
// limiting of voice.
const butterworthQ = math.Sqrt2 / 2

// NewLowPass returns a Butterworth low-pass biquad with its −3 dB corner
// at cutoffHz for a stream sampled at sampleRate Hz. cutoffHz is clamped
// below Nyquist; a non-positive sampleRate or cutoffHz yields a
// unity pass-through.
func NewLowPass(sampleRate, cutoffHz float64) *Biquad {
	if sampleRate <= 0 || cutoffHz <= 0 {
		return passThroughBiquad()
	}
	if cutoffHz > 0.49*sampleRate {
		cutoffHz = 0.49 * sampleRate
	}
	w0 := 2 * math.Pi * cutoffHz / sampleRate
	cosw0, sinw0 := math.Cos(w0), math.Sin(w0)
	alpha := sinw0 / (2 * butterworthQ)
	b1 := 1 - cosw0
	b0 := b1 / 2
	b2 := b0
	a0 := 1 + alpha
	a1 := -2 * cosw0
	a2 := 1 - alpha
	return newBiquadRBJ(b0, b1, b2, a0, a1, a2)
}

// NewHighPass returns a Butterworth high-pass biquad with its −3 dB
// corner at cutoffHz for a stream sampled at sampleRate Hz. cutoffHz is
// clamped below Nyquist; a non-positive sampleRate or cutoffHz yields a
// unity pass-through.
func NewHighPass(sampleRate, cutoffHz float64) *Biquad {
	if sampleRate <= 0 || cutoffHz <= 0 {
		return passThroughBiquad()
	}
	if cutoffHz > 0.49*sampleRate {
		cutoffHz = 0.49 * sampleRate
	}
	w0 := 2 * math.Pi * cutoffHz / sampleRate
	cosw0, sinw0 := math.Cos(w0), math.Sin(w0)
	alpha := sinw0 / (2 * butterworthQ)
	b0 := (1 + cosw0) / 2
	b1 := -(1 + cosw0)
	b2 := b0
	a0 := 1 + alpha
	a1 := -2 * cosw0
	a2 := 1 - alpha
	return newBiquadRBJ(b0, b1, b2, a0, a1, a2)
}

// passThroughBiquad returns a biquad whose output equals its input
// (b0 = 1, all other coefficients zero) so a degenerate configuration
// degrades to a no-op rather than silence.
func passThroughBiquad() *Biquad {
	return &Biquad{b0: 1}
}

// Process applies the section in place over pcm, carrying state across
// calls so successive frames filter continuously. A nil receiver is a
// no-op so callers can hold an optional *Biquad and invoke it
// unconditionally.
func (f *Biquad) Process(pcm []float64) {
	if f == nil {
		return
	}
	z1, z2 := f.z1, f.z2
	for i, x := range pcm {
		y := f.b0*x + z1
		z1 = f.b1*x - f.a1*y + z2
		z2 = f.b2*x - f.a2*y
		pcm[i] = y
	}
	f.z1, f.z2 = z1, z2
}

// Reset clears the section's state words so the next sample starts from
// a quiescent filter. Call on stream re-sync alongside the rest of the
// decode state. A nil receiver is a no-op.
func (f *Biquad) Reset() {
	if f == nil {
		return
	}
	f.z1, f.z2 = 0, 0
}
