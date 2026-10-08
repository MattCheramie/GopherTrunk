package sync

// Does the Gardner loop actually move the sampling instant *toward* the eye?
//
// The existing Gardner tests shape their symbols with a triangular / raised
// window and then assert a quadrant match rate. A flat-topped pulse decodes to
// the right quadrant over most of the symbol period, so those tests pass
// whether the loop converges on the eye or runs away from it — which is how an
// inverted feedback sign survived in this loop: with `mu += sps + gain*err`
// the loop's stable point is the *transition* instant, the worst sampling
// phase there is, and it settles there and stays. On real P25 Phase 2 air that
// cost roughly 7x the recoverable signalling (issue #915).
//
// These tests use the repo's own RRC modulator, so the pulse the loop sees is
// the one the receiver actually gets, and they judge the loop by ground truth
// — the transmitted dibits — rather than by a metric a mistimed sample can
// still satisfy.

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

const (
	timingSPS      = 8
	timingSpan     = 8
	timingAlpha    = 0.20
	timingRotation = math.Pi / 8
)

// timingDibits builds a deterministic pseudorandom dibit stream.
func timingDibits(n int) []uint8 {
	out := make([]uint8, n)
	x := uint32(0x13579BDF)
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = uint8(x & 3)
	}
	return out
}

// recoverDibits runs the timing loop over src and differentially decodes what
// it emits, returning the recovered dibit stream.
func recoverDibits(g *Gardner, src []complex64) []uint8 {
	syms := g.Process(nil, src)
	d := demod.NewDQPSK()
	d.SetRotation(timingRotation)
	return d.Decode(nil, syms)
}

// dibitErrorRate aligns got against want over the trailing tail (the loop is
// given the head to converge in) and returns the best error rate found over a
// small range of stream shifts.
func dibitErrorRate(got, want []uint8) float64 {
	best := 1.0
	for shift := -4; shift <= 4; shift++ {
		start := len(got) * 3 / 4 // judge the trailing 25%, after the loop has settled
		errs, total := 0, 0
		for i := start; i < len(got); i++ {
			j := i + shift
			if j < 0 || j >= len(want) {
				continue
			}
			total++
			if got[i] != want[j] {
				errs++
			}
		}
		if total < 50 {
			continue
		}
		if r := float64(errs) / float64(total); r < best {
			best = r
		}
	}
	return best
}

// TestGardnerConvergesFromAnyTimingPhase is the sign test. A correct loop pulls
// the sampling instant onto the eye from any starting sub-sample phase; an
// inverted one pulls it onto the transition and parks there, so the recovered
// dibits stay wrong no matter how long the stream runs.
//
// The exact half-symbol offset is skipped: it is the S-curve's unstable
// equilibrium, where a *correct* loop sees zero error by symmetry on a
// noiseless signal and so cannot leave either. Real air's noise breaks that
// symmetry; a synthesized fixture's does not.
func TestGardnerConvergesFromAnyTimingPhase(t *testing.T) {
	// Long enough that the loop has settled well before the window judged
	// below: it walks the sampling instant at roughly the loop gain per symbol.
	dibits := timingDibits(8000)
	iq := demod.ModulatePiOver4DQPSK(dibits, timingSPS, timingSpan, timingAlpha, timingRotation)

	for off := 0; off < timingSPS; off++ {
		if off == timingSPS/2 {
			continue // unstable equilibrium; see the doc comment
		}
		g := NewGardner(timingSPS, 0.02)
		got := recoverDibits(g, iq[off:])
		rate := dibitErrorRate(got, dibits)
		if rate > 0.02 {
			t.Errorf("start offset %d/%d symbol: dibit error rate %.3f, want <= 0.02 "+
				"(the loop is not converging on the eye)", off, timingSPS, rate)
		}
	}
}

// TestGardnerHigherGainIsNotWorse guards the same property from the other
// side, and is the shape of the evidence that exposed the bug on air: Phase 2
// yield collapsed monotonically as the loop gain rose (31 distinct MAC PDUs at
// a frozen loop, 2 at gain 0.05), which is what a loop running *away* from the
// eye does — the gain is how fast it leaves.
//
// A correctly-signed loop has the eye as its steady state at every gain, so
// raising the gain changes only how quickly it gets there. The assertion is
// therefore comparative, not a threshold: it holds regardless of how long
// settling takes at the slowest gain.
func TestGardnerHigherGainIsNotWorse(t *testing.T) {
	dibits := timingDibits(8000)
	iq := demod.ModulatePiOver4DQPSK(dibits, timingSPS, timingSpan, timingAlpha, timingRotation)

	rates := map[float64]float64{}
	gains := []float64{0.005, 0.02, 0.05}
	for _, gain := range gains {
		g := NewGardner(timingSPS, gain)
		rates[gain] = dibitErrorRate(recoverDibits(g, iq[3:]), dibits) // start off the eye
	}
	slow, fast := rates[gains[0]], rates[gains[len(gains)-1]]
	if fast > slow+0.01 {
		t.Errorf("dibit error rate rises with loop gain (%.3f at %g -> %.3f at %g): "+
			"the loop is running away from the eye, not toward it",
			slow, gains[0], fast, gains[len(gains)-1])
	}
	for _, gain := range gains {
		if rates[gain] > 0.02 {
			t.Errorf("gain %g: dibit error rate %.3f, want <= 0.02", gain, rates[gain])
		}
	}
}

// TestGardnerTimingErrorSignIsNegativeFeedback pins the sign directly, so a
// future edit that flips it fails here with an unambiguous message rather than
// as a yield regression nobody can attribute.
//
// The detector's error is e = Re{(y[n] − y[n−1]) · conj(m[n])} with m[n] the
// midpoint between the two symbol instants. Sampling *late* makes e positive
// (the midpoint has moved past the zero crossing toward the next symbol, and
// the symbol difference has the opposite sign), so the correction must
// *subtract* it: sampling later still would be positive feedback.
func TestGardnerTimingErrorSignIsNegativeFeedback(t *testing.T) {
	dibits := timingDibits(4000)
	iq := demod.ModulatePiOver4DQPSK(dibits, timingSPS, timingSpan, timingAlpha, timingRotation)

	// Two loops from the same deliberately-late start: one free to correct,
	// one effectively frozen. The correcting loop must end up strictly better.
	gFree := NewGardner(timingSPS, 0.02)
	gFrozen := NewGardner(timingSPS, 1e-12)
	free := dibitErrorRate(recoverDibits(gFree, iq[3:]), dibits)
	frozen := dibitErrorRate(recoverDibits(gFrozen, iq[3:]), dibits)
	if free > frozen {
		t.Errorf("adapting loop (%.3f) is worse than a frozen one (%.3f): "+
			"the timing feedback has the wrong sign", free, frozen)
	}
}
