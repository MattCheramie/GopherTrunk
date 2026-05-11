package filter

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestFIRImpulseResponse(t *testing.T) {
	taps := []float32{0.1, 0.2, 0.3, 0.2, 0.1}
	f := NewFIR(taps)
	in := make([]complex64, 16)
	in[0] = 1 + 0i
	out := f.Process(nil, in)
	for i, want := range taps {
		got := real(out[i])
		if math.Abs(float64(got-want)) > 1e-7 {
			t.Errorf("out[%d] = %f, want %f", i, got, want)
		}
	}
	for i := len(taps); i < len(out); i++ {
		if cmplx.Abs(complex128(out[i])) > 1e-7 {
			t.Errorf("out[%d] should be zero, got %v", i, out[i])
		}
	}
}

func TestLowpassKaiserPasses(t *testing.T) {
	// Generate a 0.05-cycles/sample tone (well below cutoff 0.1) and verify
	// it survives. Generate a 0.4-cycles/sample tone (above cutoff) and
	// verify it's strongly attenuated.
	taps := LowpassKaiser(101, 0.1, 8.6)
	f := NewFIR(taps)

	gen := func(freq float64, n int) []complex64 {
		out := make([]complex64, n)
		for i := 0; i < n; i++ {
			theta := 2 * math.Pi * freq * float64(i)
			out[i] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
		}
		return out
	}
	measure := func(samples []complex64, skip int) float64 {
		var p float64
		for i := skip; i < len(samples); i++ {
			a := math.Hypot(float64(real(samples[i])), float64(imag(samples[i])))
			p += a * a
		}
		return p / float64(len(samples)-skip)
	}

	pass := f.Process(nil, gen(0.05, 4096))
	f.Reset()
	stop := f.Process(nil, gen(0.4, 4096))

	pPass := measure(pass, 200)
	pStop := measure(stop, 200)
	if pPass < 0.9 {
		t.Errorf("passband power = %f, want > 0.9", pPass)
	}
	if pStop > 0.01 {
		t.Errorf("stopband power = %f, want < 0.01", pStop)
	}
}

func TestRRCUnitEnergy(t *testing.T) {
	taps := RootRaisedCosine(8, 8, 0.2)
	var e float64
	for _, c := range taps {
		e += float64(c) * float64(c)
	}
	if math.Abs(e-1) > 1e-6 {
		t.Errorf("RRC energy = %f, want 1", e)
	}
}

func TestGaussianPeakSymmetryAndUnitDCGain(t *testing.T) {
	taps := Gaussian(10, 4, 0.3)
	mid := len(taps) / 2

	// Peak at the centre.
	for i, v := range taps {
		if v > taps[mid] {
			t.Errorf("Gaussian: taps[%d]=%f > centre taps[%d]=%f",
				i, v, mid, taps[mid])
		}
	}

	// Symmetric about the centre.
	for i := 0; i < mid; i++ {
		if math.Abs(float64(taps[i]-taps[len(taps)-1-i])) > 1e-7 {
			t.Errorf("Gaussian asymmetric at i=%d: %f vs %f",
				i, taps[i], taps[len(taps)-1-i])
		}
	}

	// Unit DC gain so a sustained NRZ level passes through
	// unchanged at the symbol centre.
	var sum float64
	for _, v := range taps {
		sum += float64(v)
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("Gaussian DC gain = %f, want 1", sum)
	}
}

// TestGaussianBTSetsBandwidth: the Gaussian impulse drops to
// e^(-0.5) at ±σ; the σ implied by BT = sps · sqrt(ln 2) / (2π · BT)
// must match the impulse shape within 1 %.
func TestGaussianBTSetsBandwidth(t *testing.T) {
	const sps = 10
	const bt = 0.3
	taps := Gaussian(sps, 8, bt)
	mid := len(taps) / 2
	peak := float64(taps[mid])
	const target = 0.6065306597 // e^(-1/2)

	for i := mid; i < len(taps)-1; i++ {
		hi := float64(taps[i]) / peak
		lo := float64(taps[i+1]) / peak
		if hi >= target && lo <= target {
			frac := (hi - target) / (hi - lo)
			gotSigma := float64(i-mid) + frac
			wantSigma := float64(sps) * math.Sqrt(math.Ln2) / (2 * math.Pi * bt)
			if math.Abs(gotSigma-wantSigma)/wantSigma > 0.01 {
				t.Errorf("σ from impulse = %f, want %f", gotSigma, wantSigma)
			}
			return
		}
	}
	t.Errorf("could not locate σ in impulse — filter span too short?")
}

func TestRRCMatchedFilterPeakAtCenter(t *testing.T) {
	// RRC * RRC = RC; symbol-rate convolution should peak at the center.
	rrc := RootRaisedCosine(4, 8, 0.3)
	N := len(rrc)
	conv := make([]float32, 2*N-1)
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			conv[i+j] += rrc[i] * rrc[j]
		}
	}
	mid := N - 1
	for i := 0; i < len(conv); i++ {
		if i == mid {
			continue
		}
		if math.Abs(float64(conv[i])) > math.Abs(float64(conv[mid])) {
			t.Errorf("peak not at center: %d > %d", i, mid)
		}
	}
}
