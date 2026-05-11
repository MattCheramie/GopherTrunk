package demod

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// FFSK is a Fast-Frequency-Shift-Keying audio-band demodulator. Used by
// MPT 1327 (CCIR FFSK: mark = 1200 Hz, space = 1800 Hz, 1200 baud) and
// other audio-FSK trunked-radio signalling layers carried inside a
// narrowband-FM voice channel.
//
// Pipeline: IQ → demod.FM → real audio → demod.FFSK.Discriminate →
//
//	clock recovery (e.g. sync.MuellerMuller) → demod.FFSK.Slice / SliceMany.
//
// Internally Discriminate complex-mixes the audio down to baseband by the
// midpoint of the two tones, low-pass filters the result to keep just the
// tone deviation, then FM-discriminates the complex baseband. The output
// sign is arranged so the slicer returns 1 for the mark tone (CCIR FFSK
// convention: mark = binary 1) regardless of which tone is the higher
// of the two.
type FFSK struct {
	radPerSample float64
	phase        float64

	lpf      *filter.FIR
	lpfDelay int // group delay of the LPF in samples ((N − 1) / 2)

	// Scratch buffers reused across calls so Discriminate does not
	// allocate on the hot path.
	mixed    []complex64
	filtered []complex64

	// FM-discriminator state (last complex-baseband sample).
	last complex64

	// invertSlice is true when markHz < spaceHz: after mixing by
	// the midpoint, mark lands on the negative side, so the
	// discriminator output must be negated to make positive →
	// mark for the slicer.
	invertSlice bool
}

// NewFFSK constructs an FFSK demod for the given audio sample rate
// and mark / space tone frequencies. Mark conventionally encodes binary
// 1 (CCIR FFSK / MPT 1327: markHz = 1200, spaceHz = 1800). Panics if
// any argument is non-positive or markHz == spaceHz.
func NewFFSK(sampleRate, markHz, spaceHz float64) *FFSK {
	if sampleRate <= 0 || markHz <= 0 || spaceHz <= 0 {
		panic("demod: NewFFSK requires positive sampleRate, markHz, spaceHz")
	}
	if markHz == spaceHz {
		panic("demod: NewFFSK requires markHz != spaceHz")
	}
	centerHz := (markHz + spaceHz) / 2
	deltaHz := math.Abs(spaceHz-markHz) / 2 // half tone spacing

	// LPF cutoff: passband ends at 1.5 × the half-tone spacing so
	// the desired tone (at ±deltaHz) sits cleanly inside the
	// passband, yet the cutoff is well below the mirror tone at
	// 2·centerHz - cutoff.
	cutoff := deltaHz * 1.5 / sampleRate
	if cutoff > 0.45 {
		cutoff = 0.45
	}

	// Length: pick from the Kaiser design rule
	//   N = (A_dB − 7.95) / (2.285 · Δω)
	// with Δω = the absolute transition width in rad/sample. We
	// need ≥ 60 dB rejection by the mirror at 2·centerHz to keep
	// the inter-tone beat from contaminating the discriminator;
	// pick A = 60 dB and let the transition span from `cutoff` up
	// to (2·centerHz − cutoff) with a 100 Hz buffer.
	const stopbandDB = 60.0
	const beta = 0.1102 * (stopbandDB - 8.7) // Kaiser β for that A
	transitionHz := 2*centerHz - 2*deltaHz*1.5 - 100
	if transitionHz < 100 {
		transitionHz = 100
	}
	dOmega := 2 * math.Pi * transitionHz / sampleRate
	n := int((stopbandDB-7.95)/(2.285*dOmega)) + 1
	if n < 21 {
		n = 21
	}
	if n%2 == 0 {
		n++
	}

	return &FFSK{
		radPerSample: -2 * math.Pi * centerHz / sampleRate,
		lpf:          filter.NewFIR(filter.LowpassKaiser(n, cutoff, beta)),
		lpfDelay:     (n - 1) / 2,
		last:         complex(1, 0),
		invertSlice:  markHz < spaceHz,
	}
}

// Discriminate processes one chunk of real audio samples and writes
// the complex-mixed-and-discriminated tone-difference signal to dst.
// Output is sign-aligned: positive when the mark tone is present.
//
// Internal state carries across calls so chunk boundaries do not
// corrupt the stream.
func (f *FFSK) Discriminate(dst, src []float32) []float32 {
	if cap(f.mixed) < len(src) {
		f.mixed = make([]complex64, len(src))
	} else {
		f.mixed = f.mixed[:len(src)]
	}
	for i, x := range src {
		c, s := math.Cos(f.phase), math.Sin(f.phase)
		f.mixed[i] = complex(float32(float64(x)*c), float32(float64(x)*s))
		f.phase += f.radPerSample
		// Wrap the phase so float64 precision stays well-conditioned
		// over very long streams.
		if f.phase < -2*math.Pi || f.phase > 2*math.Pi {
			f.phase = math.Mod(f.phase, 2*math.Pi)
		}
	}
	f.filtered = f.lpf.Process(f.filtered, f.mixed)

	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, z := range f.filtered {
		// FM discriminator: arg(z * conj(last)).
		ar := real(z)*real(f.last) + imag(z)*imag(f.last)
		ai := imag(z)*real(f.last) - real(z)*imag(f.last)
		soft := float32(math.Atan2(float64(ai), float64(ar)))
		if f.invertSlice {
			soft = -soft
		}
		dst[i] = soft
		f.last = z
	}
	return dst
}

// Slice maps a soft sample to a binary symbol: mark tone present
// (positive) → 1, space tone present (non-positive) → 0. CCIR FFSK
// convention: mark = binary 1, space = binary 0.
func (f *FFSK) Slice(soft float32) int {
	if soft > 0 {
		return 1
	}
	return 0
}

// SliceMany applies Slice to a slice of soft samples.
func (f *FFSK) SliceMany(dst []int8, src []float32) []int8 {
	if cap(dst) < len(src) {
		dst = make([]int8, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		dst[i] = int8(f.Slice(s))
	}
	return dst
}

// Reset clears the mixer phase, LPF history, and discriminator
// memory. Call on stream re-sync (control-channel hunt success,
// IQ underrun recovery) so stale state doesn't bleed.
func (f *FFSK) Reset() {
	f.phase = 0
	f.lpf.Reset()
	f.last = complex(1, 0)
}

// Delay returns the LPF group delay (in samples). Symbol-rate
// clock recovery downstream of Discriminate self-aligns and does
// not need this — it's exposed mainly for unit tests that
// synthesise a known signal and need to look up the corresponding
// output sample at a fixed offset.
func (f *FFSK) Delay() int { return f.lpfDelay }
