package rtlsdr

import "math"

// DCBlocker subtracts a slowly-tracked DC bias from a complex sample stream.
// Useful because RTL-SDR dongles park noticeable DC at the tuned center bin.
type DCBlocker struct {
	alpha float64
	avgI  float64
	avgQ  float64
}

func NewDCBlocker(alpha float64) *DCBlocker {
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.001
	}
	return &DCBlocker{alpha: alpha}
}

func (d *DCBlocker) Process(in []complex64) {
	for i, s := range in {
		ri := float64(real(s))
		qi := float64(imag(s))
		d.avgI += d.alpha * (ri - d.avgI)
		d.avgQ += d.alpha * (qi - d.avgQ)
		in[i] = complex(float32(ri-d.avgI), float32(qi-d.avgQ))
	}
}

// IQBalancer applies a single-coefficient IQ-imbalance correction
// (gain + phase) measured against a calibration tone. Coefficients are 1.0
// and 0.0 by default (passthrough).
type IQBalancer struct {
	GainQ float32 // amplitude scale for Q
	Phase float32 // radians, applied to Q via Q' = Q*cos - I*sin
}

func (b IQBalancer) Process(in []complex64) {
	cos := float32(math.Cos(float64(b.Phase)))
	sin := float32(math.Sin(float64(b.Phase)))
	gain := b.GainQ
	if gain == 0 {
		gain = 1
	}
	for i, s := range in {
		ri := real(s)
		qi := imag(s) * gain
		in[i] = complex(ri, qi*cos-ri*sin)
	}
}

// PPMToHz returns the absolute-Hz offset implied by a PPM value at the given
// center frequency. Useful when reporting calibration results.
func PPMToHz(ppm int, centerHz uint32) int64 {
	return int64(ppm) * int64(centerHz) / 1_000_000
}
