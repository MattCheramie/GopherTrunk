package filter

import (
	"math"
	"time"
)

// DeEmphasis is the single-pole IIR low-pass that recovers the
// pre-emphasized treble curve broadcast FM transmitters apply for SNR.
// The transmitter pre-emphasizes (boosts highs) with a single-pole
// high-shelf characterized by time constant τ; the receiver inverts
// with the matching low-pass:
//
//	H(s) = 1 / (1 + sτ)
//
// Discretized impulse-invariant at sample rate fs, the difference
// equation is:
//
//	α   = exp(-1 / (τ × fs))
//	y[n] = (1 - α) × x[n] + α × y[n-1]
//
// The DC gain is unity, the −3 dB cutoff is fc = 1 / (2π τ).
//
// Use NewDeEmphasisUS for the 75 µs constant standard in NA, or
// NewDeEmphasisEU for the 50 µs constant used in most of the world.
//
// DeEmphasis is *not* safe for concurrent Process calls — pin it to
// a single demod goroutine and Reset between calls.
type DeEmphasis struct {
	alpha    float32 // feedback weight (close to 1 for treble suppression)
	oneMinus float32 // input weight (1 - alpha)
	prev     float32 // last output sample (state)
}

// Common pre-emphasis time constants. Pass to NewDeEmphasis with the
// PCM sample rate the filter will see.
const (
	DeEmphasis75us = 75 * time.Microsecond // FM broadcast in North America
	DeEmphasis50us = 50 * time.Microsecond // FM broadcast in Europe / most other regions
)

// NewDeEmphasis builds a de-emphasis filter tuned to time constant τ
// at the given sample rate. Both must be positive; the constructor
// panics otherwise so misconfiguration trips at startup rather than
// silently producing wrong audio.
func NewDeEmphasis(tau time.Duration, sampleRate float64) *DeEmphasis {
	if tau <= 0 {
		panic("filter: NewDeEmphasis requires a positive time constant")
	}
	if sampleRate <= 0 {
		panic("filter: NewDeEmphasis requires a positive sample rate")
	}
	tauSec := tau.Seconds()
	alpha := math.Exp(-1.0 / (tauSec * sampleRate))
	return &DeEmphasis{
		alpha:    float32(alpha),
		oneMinus: float32(1.0 - alpha),
	}
}

// NewDeEmphasisUS is shorthand for NewDeEmphasis(DeEmphasis75us, sampleRate).
func NewDeEmphasisUS(sampleRate float64) *DeEmphasis {
	return NewDeEmphasis(DeEmphasis75us, sampleRate)
}

// Reset clears the filter's running state. Call between calls so
// stale audio from one transmission doesn't bleed into the next.
func (d *DeEmphasis) Reset() {
	d.prev = 0
}

// Process applies the filter to src and writes the result to dst (or
// appends to it). dst is reused if it has enough capacity. In-place
// operation (dst == src) is supported.
func (d *DeEmphasis) Process(dst, src []float32) []float32 {
	if cap(dst) >= len(src) {
		dst = dst[:len(src)]
	} else {
		dst = make([]float32, len(src))
	}
	prev := d.prev
	for i, x := range src {
		y := d.oneMinus*x + d.alpha*prev
		dst[i] = y
		prev = y
	}
	d.prev = prev
	return dst
}
