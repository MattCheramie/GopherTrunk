package conventional

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// toneRefRateHz is the rate the tone detectors decimate their input to,
// and the rate their thresholds are calibrated at. The scanner feeds the
// detectors the SDR's full-rate IQ (2.4 MS/s on an RTL-SDR); the
// discriminator output is in radians per sample, so the same deviation
// reads 50x smaller at 2.4 MS/s than at 48 kHz. Run at the SDR rate, a
// real CTCSS tone sat ~3000x below its threshold and the gate never
// opened on any signal (issue #1184).
const toneRefRateHz = 48_000

// toneChannelCutoffHz is the one-sided bandwidth of the channel filter
// ahead of the discriminator: an FM channel (±2.5–5 kHz deviation plus
// Carson margin) passes, a 12.5/25 kHz neighbour — and the rest of the
// SDR's band, whose FM noise would otherwise swamp a sub-audible
// signalling tone — does not.
const toneChannelCutoffHz = 8_000

// toneFrontEnd is the IQ → discriminator chain the CTCSS and DCS
// detectors share: decimate to ~toneRefRateHz behind an anti-alias
// filter, channel-filter, FM-discriminate. Its output is in radians per
// sample at toneRefRateHz whatever the input rate, so a threshold means
// the same deviation at any SDR sample rate.
type toneFrontEnd struct {
	pre        *dsp.Resampler // nil when the input is already near toneRefRateHz
	chanFilter *filter.FIR    // nil when the rate is too low to need one
	scratch    []complex64
	chanBuf    []complex64
	demod      []float64
	// discScale converts radians per sample at the post-decimation rate
	// to radians per sample at toneRefRateHz.
	discScale float64
	last      complex64
	// rate is the post-decimation sample rate — the rate the detector
	// stages after the front end run at.
	rate float64
	// m is the integer decimation factor (1 = none).
	m int
}

func newToneFrontEnd(sampleHz float64) *toneFrontEnd {
	var pre *dsp.Resampler
	m := int(sampleHz / toneRefRateHz)
	if m < 2 {
		m = 1
	} else {
		pre = dsp.NewResampler(1, m, m*8+1, 8.6)
	}
	rate := sampleHz / float64(m)
	var chanFilter *filter.FIR
	if fc := toneChannelCutoffHz / rate; fc < 0.45 {
		chanFilter = filter.NewFIR(filter.LowpassKaiser(63, fc, 8.6))
	}
	return &toneFrontEnd{
		pre:        pre,
		chanFilter: chanFilter,
		discScale:  rate / toneRefRateHz,
		last:       complex(1, 0),
		rate:       rate,
		m:          m,
	}
}

// process returns the discriminator output for iq, one value per
// post-decimation sample. The returned slice is reused by the next call.
func (f *toneFrontEnd) process(iq []complex64) []float64 {
	if f.pre != nil {
		f.scratch = f.pre.Process(f.scratch, iq)
		iq = f.scratch
	}
	if f.chanFilter != nil {
		f.chanBuf = f.chanFilter.Process(f.chanBuf[:0], iq)
		iq = f.chanBuf
	}
	f.demod = f.demod[:0]
	for _, s := range iq {
		// arg(z[n] · conj(z[n-1])), normalised to toneRefRateHz.
		ar := real(s)*real(f.last) + imag(s)*imag(f.last)
		ai := imag(s)*real(f.last) - real(s)*imag(f.last)
		f.demod = append(f.demod, math.Atan2(float64(ai), float64(ar))*f.discScale)
		f.last = s
	}
	return f.demod
}

func (f *toneFrontEnd) reset() {
	if f.pre != nil {
		f.pre.Reset()
	}
	if f.chanFilter != nil {
		f.chanFilter.Reset()
	}
	f.last = complex(1, 0)
}

// onePoleAlpha is the per-sample coefficient of a single-pole IIR
// low-pass with cutoff fc at rate: alpha = dt / (RC + dt).
func onePoleAlpha(fc, rate float64) float64 {
	dt := 1.0 / rate
	rc := 1.0 / (2 * math.Pi * fc)
	return dt / (rc + dt)
}
