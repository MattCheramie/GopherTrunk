package metrics

import (
	"math"
	"math/rand"
	"testing"
)

// makeC4FMSoft builds n soft samples on the four ideal C4FM rails (outer,
// ±outer/3) with additive Gaussian noise of the given sigma, seeded.
func makeC4FMSoft(n int, outer, sigma float64, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	inner := outer * c4fmInnerRatio
	rails := [4]float64{-outer, -inner, inner, outer}
	out := make([]float32, n)
	for i := range out {
		r := rails[rng.Intn(4)]
		out[i] = float32(r + rng.NormFloat64()*sigma)
	}
	return out
}

func TestEVMC4FMCleanIsZero(t *testing.T) {
	soft := makeC4FMSoft(4000, 1.0, 0, 1)
	if evm := EVMC4FM(soft, 1.0); evm > 1e-6 {
		t.Errorf("noise-free EVM should be ~0, got %g%%", evm)
	}
}

func TestEVMC4FMTracksNoise(t *testing.T) {
	// With outer=1 and sigma=0.05, the RMS residual is ~0.05, so EVM ≈ 5%.
	soft := makeC4FMSoft(20000, 1.0, 0.05, 2)
	evm := EVMC4FM(soft, 1.0)
	approx(t, evm, 5.0, 0.5, "EVM at sigma=0.05")
	// More noise → more EVM.
	soft2 := makeC4FMSoft(20000, 1.0, 0.10, 3)
	if EVMC4FM(soft2, 1.0) <= evm {
		t.Error("EVM should grow with noise")
	}
}

func TestEstimateOuterRailC4FM(t *testing.T) {
	// Recover the outer rail from a moderately noisy stream.
	soft := makeC4FMSoft(20000, 2.5, 0.10, 4)
	got := EstimateOuterRailC4FM(soft)
	approx(t, got, 2.5, 0.1, "recovered outer rail")
}

// makeQPSK builds n complex symbols on the ideal π/4 positions with circular
// Gaussian noise of per-component sigma, scaled by an arbitrary gain.
func makeQPSK(n int, gain, sigma float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	const s = math.Sqrt2 / 2
	ideal := [4][2]float64{{s, s}, {-s, s}, {-s, -s}, {s, -s}}
	out := make([]complex64, n)
	for i := range out {
		id := ideal[rng.Intn(4)]
		out[i] = complex(
			float32(gain*id[0]+rng.NormFloat64()*sigma),
			float32(gain*id[1]+rng.NormFloat64()*sigma),
		)
	}
	return out
}

func TestEVMConstellationCleanIsZero(t *testing.T) {
	pts := makeQPSK(4000, 3.0, 0, 5) // arbitrary gain, no noise
	if evm := EVMConstellation(pts); evm > 1e-3 {
		t.Errorf("noise-free constellation EVM should be ~0, got %g%%", evm)
	}
}

func TestEVMConstellationGainInvariant(t *testing.T) {
	// Same noise relative to signal at two gains → same EVM (RMS-referenced).
	a := EVMConstellation(makeQPSK(20000, 1.0, 0.05, 6))
	b := EVMConstellation(makeQPSK(20000, 4.0, 0.20, 6)) // 4× gain, 4× sigma
	approx(t, a, b, 0.5, "EVM gain-invariance")
}

func TestSNRFromEVMRoundTrip(t *testing.T) {
	// EVM of 10% ⇒ SNR of 20 dB.
	approx(t, SNRFromEVM(10), 20, 1e-9, "SNR from 10% EVM")
	approx(t, SNRFromEVM(1), 40, 1e-9, "SNR from 1% EVM")
}

func TestSNRResidualC4FM(t *testing.T) {
	// Construct a stream with a known SNR and check the estimate.
	// Signal power of the rails {±1, ±1/3} is (1+1/9)/2 = 0.5556; with
	// sigma=0.05 noise power is 0.0025, so SNR ≈ 10log10(0.5556/0.0025) ≈ 23.5 dB.
	outer := 1.0
	sigma := 0.05
	soft := makeC4FMSoft(40000, outer, sigma, 7)
	want := 10 * math.Log10(0.5555556/(sigma*sigma))
	got := SNRResidualC4FM(soft, outer)
	approx(t, got, want, 0.6, "C4FM residual SNR")
}

func TestSNRM2M4AgreesWithResidual(t *testing.T) {
	// On a QPSK stream the moment estimator and the decision-directed
	// residual estimator should land close.
	pts := makeQPSK(40000, 1.0, 0.08, 8)
	m2m4 := SNRM2M4Constellation(pts)
	resid := SNRResidualConstellation(pts)
	approx(t, m2m4, resid, 1.5, "M2M4 vs residual SNR")
	// And both near the injected SNR: signal power per component sums to 1
	// (unit modulus), noise power = 2·sigma² = 0.0128, SNR ≈ 19 dB.
	want := 10 * math.Log10(1.0/(2*0.08*0.08))
	approx(t, resid, want, 1.5, "residual SNR vs injected")
}
