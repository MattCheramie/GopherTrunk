package dsp

import "math"

// AGC is a feedback automatic-gain-control loop that drives the average
// magnitude of a complex IQ stream toward Reference.
type AGC struct {
	Reference float32 // target output magnitude (default 1.0)
	Rate      float32 // adaptation step (typical 1e-3 to 1e-2)
	MaxGain   float32 // ceiling to avoid runaway on dead-air
	gain      float32
}

func NewAGC(reference, rate, maxGain float32) *AGC {
	if reference <= 0 {
		reference = 1
	}
	if rate <= 0 {
		rate = 1e-3
	}
	if maxGain <= 0 {
		maxGain = 65536
	}
	return &AGC{Reference: reference, Rate: rate, MaxGain: maxGain, gain: 1}
}

func (a *AGC) Gain() float32 { return a.gain }

func (a *AGC) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		out := complex(real(s)*a.gain, imag(s)*a.gain)
		mag := float32(math.Hypot(float64(real(out)), float64(imag(out))))
		// Update gain toward reference.
		err := a.Reference - mag
		a.gain += a.Rate * err / max32(mag, 1e-6)
		if a.gain > a.MaxGain {
			a.gain = a.MaxGain
		}
		if a.gain < 1e-6 {
			a.gain = 1e-6
		}
		dst[i] = out
	}
	return dst
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
