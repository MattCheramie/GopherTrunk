package sync

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// shapedPAM renders random 4-level symbols through a root-raised-cosine
// pulse at sps samples/symbol, so the symbol instants sit at
// index = delay + k·sps where delay is the filter's group delay.
func shapedPAM(rng *rand.Rand, symbols, sps, span int, alpha float64) (sig []float32, delay int) {
	taps := filter.RootRaisedCosine(sps, span, alpha)
	up := make([]float32, symbols*sps)
	levels := []float32{-3, -1, 1, 3}
	for k := 0; k < symbols; k++ {
		up[k*sps] = levels[rng.Intn(4)]
	}
	sig = make([]float32, len(up))
	for n := range sig {
		var acc float32
		for t, h := range taps {
			if i := n - t; i >= 0 {
				acc += h * up[i]
			}
		}
		sig[n] = acc
	}
	return sig, (len(taps) - 1) / 2
}

// matchedPAM is shapedPAM after the receive root-raised-cosine (the
// matched-filter output a receiver estimates timing on), with optional AWGN
// added before the receive filter. instant is the index of the first symbol
// decision point.
func matchedPAM(rng *rand.Rand, symbols, sps int, noise float64) (mf []float32, instant int) {
	sig, delay := shapedPAM(rng, symbols, sps, 8, 0.20)
	if noise > 0 {
		var e float64
		for _, v := range sig {
			e += float64(v * v)
		}
		rms := math.Sqrt(e / float64(len(sig)))
		for i := range sig {
			sig[i] += float32(rng.NormFloat64() * noise * rms)
		}
	}
	taps := filter.RootRaisedCosine(sps, 8, 0.20)
	mf = make([]float32, len(sig))
	for n := range mf {
		var acc float32
		for t, h := range taps {
			if i := n - t; i >= 0 {
				acc += h * sig[i]
			}
		}
		mf[n] = acc
	}
	return mf, 2 * delay
}

// TestEstimateSymbolPhaseRecoversKnownInstants: over hundreds of 96-symbol
// windows at every sub-symbol offset, clean and at ~10 dB SNR, the estimate
// must point at the true symbol instants to within one sample.
func TestEstimateSymbolPhaseRecoversKnownInstants(t *testing.T) {
	const sps = 10
	for _, noise := range []float64{0, 0.3} {
		rng := rand.New(rand.NewSource(836))
		mf, instant := matchedPAM(rng, 3000, sps, noise)
		const L = 96 * sps
		var within, total, declined int
		for start := 200; start+L < len(mf); start += 37 {
			tau, ok := EstimateSymbolPhase(mf[start:start+L], sps)
			total++
			if !ok {
				// A declined window costs the receiver one burst (it retries
				// on the next); it must stay rare.
				declined++
				continue
			}
			want := math.Mod(float64(instant-start)+10*sps, sps)
			d := math.Abs(tau - want)
			if d > sps/2 {
				d = sps - d
			}
			if d <= 1.0 {
				within++
			} else if d > 2.5 {
				t.Errorf("noise=%.1f start=%d: tau=%.2f want %.2f (err %.2f samples)", noise, start, tau, want, d)
			}
		}
		// Clean: ≥ 95 % within a sample. At ~10 dB SNR a few windows land
		// 1–2 samples off (the loop pulls those in within a burst); ≥ 90 %.
		need := 95
		if noise > 0 {
			need = 90
		}
		t.Logf("noise=%.1f: %d of %d windows within 1 sample, %d declined", noise, within, total, declined)
		if within*100 < total*need {
			t.Errorf("noise=%.1f: only %d of %d windows within 1 sample", noise, within, total)
		}
		if declined*100 > total*3 {
			t.Errorf("noise=%.1f: %d of %d windows declined", noise, declined, total)
		}
	}
}

// TestEstimateSymbolPhaseRejectsNoise: Gaussian noise has no eye — kurtosis
// ≈ 3 at every phase — so the estimator must decline rather than return a
// random phase; so must a window too short to trust.
func TestEstimateSymbolPhaseRejectsNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	taps := filter.RootRaisedCosine(10, 8, 0.20)
	rejected := 0
	const trials = 200
	for trial := 0; trial < trials; trial++ {
		raw := make([]float32, 960+len(taps))
		for i := range raw {
			raw[i] = float32(rng.NormFloat64())
		}
		// Band-limit like the receiver's matched filter would.
		nb := make([]float32, 960)
		for n := range nb {
			var acc float32
			for t, h := range taps {
				acc += h * raw[n+len(taps)-1-t]
			}
			nb[n] = acc
		}
		if _, ok := EstimateSymbolPhase(nb, 10); !ok {
			rejected++
		}
	}
	if rejected < trials*90/100 {
		t.Fatalf("only %d of %d noise windows rejected", rejected, trials)
	}
	if _, ok := EstimateSymbolPhase(make([]float32, 300), 10); ok {
		t.Fatal("short window accepted")
	}
}

// TestSetPhaseSeedsNextInstant: after SetPhase(mu) the loop emits its next
// symbol exactly mu samples after the last processed sample.
func TestSetPhaseSeedsNextInstant(t *testing.T) {
	m := NewMuellerMuller(10, 0.015)
	m.Process(nil, make([]float32, 25))
	m.SetPhase(3.5)
	if got := m.Mu(); got != 3.5 {
		t.Fatalf("mu=%v", got)
	}
	ramp := make([]float32, 30)
	for i := range ramp {
		ramp[i] = float32(i)
	}
	out := m.Process(nil, ramp)
	if len(out) == 0 {
		t.Fatal("no symbol")
	}
	// Instant 3.5 samples past the previous chunk's last sample: between
	// ramp[2] (=2) and ramp[3] (=3) at frac 0.5 → 2.5.
	if math.Abs(float64(out[0])-2.5) > 1e-5 {
		t.Fatalf("first symbol = %v, want 2.5", out[0])
	}
	m.SetPhase(-4)
	if got := m.Mu(); got != 6 {
		t.Fatalf("wrapped mu=%v want 6", got)
	}
	m.SetPhase(0)
	if got := m.Mu(); got != 10 {
		t.Fatalf("zero mu=%v want 10", got)
	}
}
