package dsp

import "math"

// AGC normalises the average magnitude of a complex IQ stream toward
// Reference. It tracks signal power with an exponential moving average
// and applies that as a feed-forward scale, so an occasional near-zero
// sample — a linear-modulation symbol stream passes through the origin
// on π phase transitions — cannot spike the gain the way a per-sample
// feedback loop would.
type AGC struct {
	Reference float32 // target output magnitude (default 1.0)
	Rate      float32 // power-EMA coefficient (typical 1e-3 to 1e-2)
	MaxGain   float32 // ceiling that bounds the gain on near-silent input
	power     float32 // EMA of input power
	seeded    bool
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
	return &AGC{Reference: reference, Rate: rate, MaxGain: maxGain}
}

// Gain reports the scale Process would currently apply.
func (a *AGC) Gain() float32 { return a.gainFor(a.power) }

func (a *AGC) gainFor(power float32) float32 {
	if !a.seeded || power <= 1e-20 {
		return 1
	}
	g := a.Reference / float32(math.Sqrt(float64(power)))
	if g > a.MaxGain {
		g = a.MaxGain
	}
	return g
}

// Reset clears the tracked power so a stream re-sync or re-tune does
// not carry a stale gain into the new signal.
func (a *AGC) Reset() {
	a.power = 0
	a.seeded = false
}

func (a *AGC) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	if len(src) == 0 {
		return dst
	}
	// Seed the power EMA from the first batch's mean power so the gain
	// is correct immediately rather than ramping up from an arbitrary
	// start — important when batches are short and the caller cannot
	// afford a long settling transient. A silent batch is not a usable
	// seed (it would spike the gain when signal arrives), so hold off
	// and pass through at unity gain until a batch carries energy.
	if !a.seeded {
		var sum float32
		for _, s := range src {
			sum += real(s)*real(s) + imag(s)*imag(s)
		}
		if p := sum / float32(len(src)); p > 1e-20 {
			a.power = p
			a.seeded = true
		}
	}
	for i, s := range src {
		p := real(s)*real(s) + imag(s)*imag(s)
		a.power += a.Rate * (p - a.power)
		g := a.gainFor(a.power)
		dst[i] = complex(real(s)*g, imag(s)*g)
	}
	return dst
}
