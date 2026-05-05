// Package channelizer implements an M-channel critically-sampled polyphase
// channelizer. One IQ stream at sample rate Fs is split into M evenly-spaced
// baseband streams each at Fs/M.
//
// The implementation follows the standard textbook structure (e.g., Harris,
// "Multirate Signal Processing for Communication Systems"): a prototype
// lowpass filter is decomposed into M polyphase branches; for every M input
// samples, each branch produces one sample, the M branch outputs are
// FFT-rotated, and the result is M output samples — one per channel.
package channelizer

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

type Polyphase struct {
	m         int
	taps      int          // taps per branch
	branches  [][]float32  // [m][taps]
	hist      [][]complex64 // [m][taps] ring buffer per branch
	histPos   []int
	commIdx   int          // next branch to receive an input sample
	plan      fft.Plan
	scratchIn []complex128
}

// New builds an M-channel channelizer using a Kaiser-window prototype LPF
// with cutoff 1/(2M) and the given taps-per-branch and shape parameter.
func New(channels, tapsPerBranch int, kaiserBeta float64) *Polyphase {
	if channels < 2 {
		panic("channelizer: channels must be >= 2")
	}
	N := channels * tapsPerBranch
	if N%2 == 0 {
		N++
	}
	proto := filter.LowpassKaiser(N, 0.5/float64(channels), kaiserBeta)
	return NewWithPrototype(channels, proto)
}

// NewWithPrototype lets callers supply a custom prototype filter.
func NewWithPrototype(channels int, proto []float32) *Polyphase {
	branches := make([][]float32, channels)
	tapsPer := (len(proto) + channels - 1) / channels
	for b := 0; b < channels; b++ {
		row := make([]float32, tapsPer)
		for k := 0; k < tapsPer; k++ {
			j := k*channels + b
			if j < len(proto) {
				row[k] = proto[j]
			}
		}
		branches[b] = row
	}
	hist := make([][]complex64, channels)
	histPos := make([]int, channels)
	for i := range hist {
		hist[i] = make([]complex64, tapsPer)
	}
	return &Polyphase{
		m:         channels,
		taps:      tapsPer,
		branches:  branches,
		hist:      hist,
		histPos:   histPos,
		plan:      fft.New(channels),
		scratchIn: make([]complex128, channels),
	}
}

// Channels returns the number of output channels M.
func (p *Polyphase) Channels() int { return p.m }

// Process consumes len(src) IQ samples. For every M input samples it emits
// one output sample per channel; the result is dst[ch][k] for each channel.
// dst is reused if it has the right shape.
func (p *Polyphase) Process(dst [][]complex64, src []complex64) [][]complex64 {
	if dst == nil || len(dst) != p.m {
		dst = make([][]complex64, p.m)
	}
	for c := 0; c < p.m; c++ {
		dst[c] = dst[c][:0]
	}
	for _, x := range src {
		// Feed the incoming sample into branch p.commIdx. With a forward
		// FFT (e^{-j2π m n / M}), input sample n going to branch n means
		// that a positive tone at +m·Fs/M lands in output channel m.
		b := p.commIdx
		hpos := p.histPos[b]
		p.hist[b][hpos] = x
		p.histPos[b] = (hpos + 1) % p.taps

		p.commIdx++
		if p.commIdx == p.m {
			p.commIdx = 0
			// Compute one polyphase output per branch, then FFT.
			for k := 0; k < p.m; k++ {
				row := p.branches[k]
				idx := p.histPos[k] - 1
				if idx < 0 {
					idx += p.taps
				}
				var accI, accQ float32
				for n := 0; n < p.taps; n++ {
					s := p.hist[k][idx]
					h := row[n]
					accI += h * real(s)
					accQ += h * imag(s)
					idx--
					if idx < 0 {
						idx = p.taps - 1
					}
				}
				p.scratchIn[k] = complex(float64(accI), float64(accQ))
			}
			// FFT across branches → per-channel output sample.
			freq := p.plan.Forward(nil, p.scratchIn)
			for c := 0; c < p.m; c++ {
				dst[c] = append(dst[c], complex(float32(real(freq[c])), float32(imag(freq[c]))))
			}
		}
	}
	return dst
}
