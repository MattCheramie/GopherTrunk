package receiver

import (
	"math"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// carrierTrackSymbols is the tracker's time constant in symbol periods
// (~53 ms at NXDN48, ~27 ms at NXDN96): long enough that the data
// modulation averages out (the loop's own wander is a few tens of Hz, which
// the post-clock CoarseAFC removes), short enough to pull a 1 kHz offset in
// within the first few frames of a call.
const carrierTrackSymbols = 128

// carrierTrackMaxHz bounds the correction. A noise-only stretch has a
// zero-mean discriminator and cannot walk the loop far, but the bound keeps
// it from running off the channel whatever the input.
const carrierTrackMaxHz = 3000.0

// carrierTrackNoiseFraction is the discriminator variance, as a fraction of
// noise alone through the same channel filter, above which a chunk is not
// trusted to carry a carrier and the estimate holds. On the #1261 captures a
// carrier chunk sits at 0.07x (NXDN96) and 0.05x (NXDN48) of the noise
// variance, so a quarter leaves room for a weak signal. Without the gate the
// loop has no restoring force on noise — every chunk reads zero-mean in its
// own mixed frame — and it random-walked to ~1.1 kHz in 5 s of noise.
const carrierTrackNoiseFraction = 0.25

// carrierTracker is a first-order frequency-locked loop in the IQ domain. It
// mixes each chunk down by its current estimate, and the next estimate moves
// toward the mean of the discriminator output — the residual offset in
// rad/sample for a balanced symbol stream — with weight 1-exp(-n/tc), so the
// response does not depend on the chunk size.
type carrierTracker struct {
	nco      *dsp.NCO
	fs       float64
	tc       float64 // time constant in samples
	varLimit float64 // discriminator variance above which a chunk is noise
	block    int     // longest stretch processed per update (tc/4)
	offHz    float64
	mixed    []complex64
}

func newCarrierTracker(sampleRateHz, tcSamples, varLimit float64) *carrierTracker {
	return &carrierTracker{nco: dsp.NewNCO(0, sampleRateHz), fs: sampleRateHz, tc: tcSamples, varLimit: varLimit,
		block: max(1, int(tcSamples/4))}
}

func (t *carrierTracker) mix(iq []complex64) []complex64 {
	t.mixed = t.nco.Mix(t.mixed, iq)
	return t.mixed
}

// update folds the residual offset measured on disc (the discriminator
// output of the chunk just mixed) into the estimate.
func (t *carrierTracker) update(disc []float32) {
	if len(disc) == 0 {
		return
	}
	var sum, sum2 float64
	for _, x := range disc {
		sum += float64(x)
		sum2 += float64(x) * float64(x)
	}
	mean := sum / float64(len(disc))
	if sum2/float64(len(disc))-mean*mean > t.varLimit {
		return // no carrier in this chunk: hold
	}
	residualHz := mean * t.fs / (2 * math.Pi)
	t.offHz += (1 - math.Exp(-float64(len(disc))/t.tc)) * residualHz
	t.offHz = math.Max(-carrierTrackMaxHz, math.Min(carrierTrackMaxHz, t.offHz))
	t.nco.SetOffset(t.offHz, t.fs)
}

func (t *carrierTracker) reset() {
	t.offHz = 0
	t.nco.SetOffset(0, t.fs)
	t.nco.Reset()
}

// noiseDiscVariance is the FM-discriminator output variance of white noise
// after the channel filter taps (nil = no filter), measured on fixed-seed
// noise: the wrapped variance of narrowband noise's instantaneous frequency
// has no handy closed form (the DMR carrier gate measures it the same way).
func noiseDiscVariance(taps []float32) float64 {
	rng := rand.New(rand.NewSource(1))
	noise := make([]complex64, 16384)
	for i := range noise {
		noise[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	if taps != nil {
		noise = filter.NewFIR(taps).Process(nil, noise)
	}
	disc := demod.NewFM().Process(nil, noise)
	disc = disc[len(taps):] // skip the filter's start-up transient
	var sum, sum2 float64
	for _, x := range disc {
		sum += float64(x)
		sum2 += float64(x) * float64(x)
	}
	m := sum / float64(len(disc))
	return sum2/float64(len(disc)) - m*m
}
