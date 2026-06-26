package tuner

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/channelizer"
)

// ChannelizerBank is a Bank that amortises the wide-band anti-alias
// filter across all taps. A single M-channel polyphase channelizer
// splits the input into evenly-spaced bins of width InputRateHz/M; the
// tap whose requested offset is closest to a bin centre takes that
// bin's output and runs a small fine-tune DDC (NCO mixer + rational
// resampler) to remove the residual offset and decimate to OutputRateHz.
//
// CPU cost is roughly O(N_samples · log M) for the shared channelizer
// plus a small per-tap fine-tune; this wins over DDCBank once tap
// count is large (≥ 7-8 in practice).
//
// More than one tap may share a channelizer bin: each tap reads the same
// bin output and runs its own fine-tune DDC (NCO mixer keyed off its own
// residual offset + resampler), so the taps are independent downstream and
// their narrow per-protocol receivers reject the in-band neighbours. The
// caveat is purely a quality one: a critically-sampled bin rolls off toward
// its edges, so a tap whose residual sits near ±binRate/2 is attenuated.
// Callers that need every channel at full SNR should keep residuals well
// inside the bin (raise the bin count) or use DDCBank, which has no bin
// geometry at all. See widebandt2's strategy picker.
type ChannelizerBank struct {
	inRateHz    float64
	outRateHz   float64
	actualOutHz float64
	guardFrac   float64

	channels  int
	binRateHz float64

	ch   *channelizer.Polyphase
	bins [][]complex64 // reused channelizer output buffer

	taps []*channelizerTap
}

type channelizerTap struct {
	offsetHz   float64
	binIdx     int
	residualHz float64
	nco        *nco
	resampler  *dsp.Resampler
	mixBuf     []complex64
	outBuf     []complex64
	sink       SinkFunc
}

// NewChannelizerBank constructs a ChannelizerBank with the requested
// number of channelizer bins (must be ≥ 2). tapsPerBranch and kaiserBeta
// govern the channelizer's prototype filter; sensible defaults are 16
// and 9.0 respectively, matching the channelizer package's own tests.
func NewChannelizerBank(inRateHz, outRateHz, guardFrac float64, channels, tapsPerBranch int, kaiserBeta float64) *ChannelizerBank {
	if channels < 2 {
		panic("tuner: ChannelizerBank requires channels >= 2")
	}
	binRateHz := inRateHz / float64(channels)
	l, m := rationalRatio(outRateHz, binRateHz)
	return &ChannelizerBank{
		inRateHz:    inRateHz,
		outRateHz:   outRateHz,
		actualOutHz: binRateHz * float64(l) / float64(m),
		guardFrac:   guardFrac,
		channels:    channels,
		binRateHz:   binRateHz,
		ch:          channelizer.New(channels, tapsPerBranch, kaiserBeta),
	}
}

// AddTap registers a new tap. Several taps may share a channelizer bin;
// each gets its own fine-tune DDC keyed off its residual offset from the
// bin centre, so they decode independently (see the type doc for the
// edge-roll-off caveat). Returns ErrOffsetOutOfBand if the offset falls
// outside the usable band.
func (b *ChannelizerBank) AddTap(offsetHz float64, sink SinkFunc) error {
	if !b.offsetInBand(offsetHz) {
		return ErrOffsetOutOfBand
	}
	binIdx := b.binForOffset(offsetHz)
	binCenter := b.binCenterHz(binIdx)
	residual := offsetHz - binCenter

	l, m := rationalRatio(b.outRateHz, b.binRateHz)
	tapsPerBranch := (ddcStopbandTaps*m + l - 1) / l
	if tapsPerBranch < ddcMinTapsPerBranch {
		tapsPerBranch = ddcMinTapsPerBranch
	}
	tap := &channelizerTap{
		offsetHz:   offsetHz,
		binIdx:     binIdx,
		residualHz: residual,
		nco:        newNCO(residual, b.binRateHz),
		resampler:  dsp.NewResampler(l, m, tapsPerBranch, ddcKaiserBeta),
		sink:       sink,
	}
	b.taps = append(b.taps, tap)
	return nil
}

// MaxResidualFrac returns the largest |residual|/binRate over all taps added
// so far — a 0..0.5 measure of how close the worst-placed tap sits to its
// bin edge, where the critically-sampled prototype rolls off. Callers use it
// to decide whether the polyphase layout is clean enough or whether a
// per-tap DDC (no bin geometry) would decode the set better.
func (b *ChannelizerBank) MaxResidualFrac() float64 {
	worst := 0.0
	for _, t := range b.taps {
		if f := math.Abs(t.residualHz) / b.binRateHz; f > worst {
			worst = f
		}
	}
	return worst
}

func (b *ChannelizerBank) offsetInBand(offsetHz float64) bool {
	half := b.inRateHz * (0.5 - b.guardFrac)
	return offsetHz >= -half && offsetHz <= half
}

// binForOffset picks the channelizer bin whose centre is closest to
// offsetHz. Bins 0..M/2-1 cover positive frequencies; bins M/2..M-1
// alias negative frequencies (-Fs/2 .. 0), matching the standard FFT
// indexing convention.
func (b *ChannelizerBank) binForOffset(offsetHz float64) int {
	k := int(math.Round(offsetHz / b.binRateHz))
	k = ((k % b.channels) + b.channels) % b.channels
	return k
}

func (b *ChannelizerBank) binCenterHz(binIdx int) float64 {
	if binIdx < b.channels/2 {
		return float64(binIdx) * b.binRateHz
	}
	return float64(binIdx-b.channels) * b.binRateHz
}

// Process drives the shared channelizer once and dispatches each tap's
// bin into its fine-tune chain.
func (b *ChannelizerBank) Process(src []complex64) {
	if len(src) == 0 {
		for _, t := range b.taps {
			t.sink(nil)
		}
		return
	}
	b.bins = b.ch.Process(b.bins, src)
	for _, t := range b.taps {
		bin := b.bins[t.binIdx]
		if cap(t.mixBuf) < len(bin) {
			t.mixBuf = make([]complex64, len(bin))
		} else {
			t.mixBuf = t.mixBuf[:len(bin)]
		}
		mixInto(t.mixBuf, bin, t.nco)
		t.outBuf = t.resampler.Process(t.outBuf, t.mixBuf)
		t.sink(t.outBuf)
	}
}

// InputRateHz returns the wide-band sample rate.
func (b *ChannelizerBank) InputRateHz() float64 { return b.inRateHz }

// OutputRateHz returns the per-tap narrow-band sample rate the bank actually
// emits. The fine-tune resampler decimates the bin rate (InputRateHz/channels)
// to the construction target, but when that exact ratio exceeds the resampler
// caps the achieved rate differs by a fraction of a percent (issue #550);
// consumers should build their symbol clocks from this value.
func (b *ChannelizerBank) OutputRateHz() float64 { return b.actualOutHz }

// Reset clears the channelizer and every tap's fine-tune state. The
// channelizer.Polyphase has no Reset method of its own; we drive a
// flush of zero-input samples through it to clear the polyphase history
// so a restart doesn't replay stale samples.
func (b *ChannelizerBank) Reset() {
	flush := make([]complex64, b.channels*16)
	b.ch.Process(b.bins, flush)
	for _, t := range b.taps {
		t.nco.reset()
		t.resampler.Reset()
	}
}
