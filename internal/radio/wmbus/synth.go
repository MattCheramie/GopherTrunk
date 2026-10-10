package wmbus

import (
	"math"
	"math/rand"
)

// SynthOptions configures Synthesize.
type SynthOptions struct {
	Mode   Mode   // ModeT1 (default) or ModeC1
	Format Format // FormatA (default) or FormatB (C mode only)
	// SampleRateHz is the IQ rate. Required.
	SampleRateHz float64
	// OffsetHz places the carrier off the IQ centre.
	OffsetHz float64
	// ChipRateHz defaults to ChipRateHz; T mode meters may be a few
	// percent off it.
	ChipRateHz float64
	// DeviationHz defaults to 50 kHz (T mode nominal).
	DeviationHz float64
	// NoiseRMS is the per-axis noise level against a 0.5 signal
	// amplitude; 0 is noiseless.
	NoiseRMS float64
	Seed     int64
}

// Synthesize modulates one telegram as on-air 2-FSK IQ, with a millisecond
// of silence before and after. data is link-layer data without CRCs, its
// L-field counting the bytes after it (Frame.Data's convention); the CRCs
// are added here. For tests and offline checks of the decoder.
func Synthesize(data []byte, o SynthOptions) []complex64 {
	if o.Mode == 0 {
		o.Mode = ModeT1
	}
	if o.Format == 0 || o.Mode == ModeT1 {
		o.Format = FormatA
	}
	if o.ChipRateHz == 0 {
		o.ChipRateHz = ChipRateHz
	}
	if o.DeviationHz == 0 {
		o.DeviationHz = 50_000
	}
	r := rand.New(rand.NewSource(o.Seed))
	chips := encodeFrameChips(o.Mode, o.Format, buildFrame(data, o.Format), 48)
	rate := o.SampleRateHz
	spc := rate / o.ChipRateHz
	n := int(float64(len(chips)) * spc)
	pad := int(rate / 1000)
	iq := make([]complex64, 0, n+2*pad)
	ph, f := 0.0, 0.0
	for i := 0; i < n+2*pad; i++ {
		target, amp := 0.0, 0.0
		if k := i - pad; k >= 0 && k < n {
			if c := int(float64(k) / spc); c < len(chips) {
				amp, target = 1, -o.DeviationHz
				if chips[c] == 1 {
					target = o.DeviationHz
				}
			}
		}
		// A one-pole frequency smoothing stands in for the
		// transmitter's filtering.
		f += (target - f) * 0.35
		ph += 2 * math.Pi * (f + o.OffsetHz) / rate
		iq = append(iq, complex(
			float32(amp*0.5*math.Cos(ph)+r.NormFloat64()*o.NoiseRMS),
			float32(amp*0.5*math.Sin(ph)+r.NormFloat64()*o.NoiseRMS)))
	}
	return iq
}
