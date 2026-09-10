package channelizer

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// Oversampled is a 2x-oversampled M-channel polyphase channelizer: the bins
// are spaced Fs/M apart exactly like Polyphase, but each bin is emitted at
// 2·Fs/M and its prototype lowpass keeps the WHOLE bin — edges included — in
// its flat passband.
//
// Why it exists: the critically-sampled Polyphase emits each bin at Fs/M with
// a prototype whose −6 dB point sits ON the bin edge, so a carrier a fraction
// of a bin off-centre is both tilted by the roll-off and, past ±Fs/(2M),
// folded back onto itself. A DMR repeater 0.48 bins off-centre on a 6.25 MS/s
// / 32-bin plan (the 10 Sep IPSC report: sync_hits=0 for minutes on a 30 dB
// SNR carrier the DDC decoded continuously) is the failure mode. With 2x
// oversampling the bin's Nyquist edge moves to ±Fs/M and the prototype's
// cutoff to 0.75·Fs/M: everything within ±0.5 bin of the centre passes flat,
// the transition band is content the tap's own resampler rejects, and only
// the ≥ 0.93·Fs/M stopband can alias, onto |f| ≥ 0.93 bins — never onto a
// tap's channel.
//
// Structure (Harris, "Multirate Signal Processing", M/2 polyphase): every
// D = M/2 input samples one output sample per bin is computed as
//
//	y_k[s] = e^{-j2πk·t_s/M} · Σ_r e^{+j2πkr/M} · u_r,   u_r = Σ_p h[r+pM]·x[t_s−r−pM]
//
// i.e. the polyphase partial sums over the last N input samples, an M-point
// inverse DFT, and the per-bin rotation that the output-time advance of M/2
// (not M) samples per step leaves behind. That rotation is what a critically-
// sampled channelizer never needs — and what an oversampled one silently gets
// wrong without.
type Oversampled struct {
	m     int
	n     int         // prototype length padded to a multiple of m
	proto []float32   // padded prototype
	hist  []complex64 // 2n mirrored history: hist[pos:pos+n] is the last n samples oldest-first
	pos   int
	since int // input samples since the last output step
	tmod  int // (index of the newest input sample) mod m
	plan  fft.Plan
	u     []complex128
	y     []complex128
	rot   []complex64 // rot[q] = e^{-j2πq/M}
}

// NewOversampled builds a 2x-oversampled M-channel channelizer (M even, ≥ 2)
// with a Kaiser-window prototype of tapsPerBranch·M taps, cutoff 0.75·Fs/M
// and the given shape parameter.
func NewOversampled(channels, tapsPerBranch int, kaiserBeta float64) *Oversampled {
	if channels < 2 || channels%2 != 0 {
		panic("channelizer: oversampled channelizer needs an even channel count >= 2")
	}
	N := channels * tapsPerBranch
	if N%2 == 0 {
		N++
	}
	proto := filter.LowpassKaiser(N, 0.75/float64(channels), kaiserBeta)
	return NewOversampledWithPrototype(channels, proto)
}

// NewOversampledWithPrototype lets callers supply the prototype filter.
func NewOversampledWithPrototype(channels int, proto []float32) *Oversampled {
	if channels < 2 || channels%2 != 0 {
		panic("channelizer: oversampled channelizer needs an even channel count >= 2")
	}
	n := ((len(proto) + channels - 1) / channels) * channels
	padded := make([]float32, n)
	copy(padded, proto)
	rot := make([]complex64, channels)
	for q := range rot {
		th := -2 * math.Pi * float64(q) / float64(channels)
		rot[q] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
	}
	return &Oversampled{
		m:     channels,
		n:     n,
		proto: padded,
		hist:  make([]complex64, 2*n),
		tmod:  channels - 1, // so the first input sample is t = 0
		plan:  fft.New(channels),
		u:     make([]complex128, channels),
		rot:   rot,
	}
}

// Channels returns the number of bins M.
func (p *Oversampled) Channels() int { return p.m }

// OversamplingFactor is the ratio of a bin's output rate to the bin spacing.
func (p *Oversampled) OversamplingFactor() int { return 2 }

// Process consumes len(src) IQ samples and emits one output sample per bin
// for every M/2 input samples; dst[ch][k] is bin ch's k-th output. dst is
// reused when it has the right shape.
func (p *Oversampled) Process(dst [][]complex64, src []complex64) [][]complex64 {
	if dst == nil || len(dst) != p.m {
		dst = make([][]complex64, p.m)
	}
	for c := 0; c < p.m; c++ {
		dst[c] = dst[c][:0]
	}
	m, n, step := p.m, p.n, p.m/2
	for _, x := range src {
		p.hist[p.pos] = x
		p.hist[p.pos+n] = x
		p.pos++
		if p.pos == n {
			p.pos = 0
		}
		p.tmod++
		if p.tmod == m {
			p.tmod = 0
		}
		p.since++
		if p.since < step {
			continue
		}
		p.since = 0
		// w is the last n samples oldest-first: x[t_s − i] = w[n−1−i].
		w := p.hist[p.pos : p.pos+n]
		for r := 0; r < m; r++ {
			var accI, accQ float32
			// h[r+pM] pairs with w[n−1−r−pM]; walk both from the newest tap.
			for i := r; i < n; i += m {
				s := w[n-1-i]
				h := p.proto[i]
				accI += h * real(s)
				accQ += h * imag(s)
			}
			p.u[r] = complex(float64(accI), float64(accQ))
		}
		p.y = p.plan.Inverse(p.y, p.u)
		// The plan's inverse is 1/M-normalised; the channelizer wants the
		// plain sum so a unit tone at a bin centre leaves at unit amplitude
		// (DC-normalised prototype), matching Polyphase.
		scale := float32(m)
		for k := 0; k < m; k++ {
			v := p.y[k]
			r := p.rot[(k*p.tmod)%m]
			re, im := float32(real(v))*scale, float32(imag(v))*scale
			dst[k] = append(dst[k], complex(re*real(r)-im*imag(r), re*imag(r)+im*real(r)))
		}
	}
	return dst
}

// Reset clears the history so a restarted stream does not see stale samples.
func (p *Oversampled) Reset() {
	for i := range p.hist {
		p.hist[i] = 0
	}
	p.pos, p.since, p.tmod = 0, 0, p.m-1
}
