package equalizer

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// genQPSK returns n unit-magnitude QPSK symbols deterministically
// drawn from {±1±j}/√2. Constant-modulus, perfect for the equaliser
// tests.
func genQPSK(n int, seed int64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	out := make([]complex64, n)
	const inv = 0.7071067811865475 // 1/sqrt(2)
	for i := range out {
		var re, im float32
		if r.Intn(2) == 0 {
			re = inv
		} else {
			re = -inv
		}
		if r.Intn(2) == 0 {
			im = inv
		} else {
			im = -inv
		}
		out[i] = complex(re, im)
	}
	return out
}

// passThroughChannel applies a 2-tap multipath model:
// y[n] = x[n] + alpha * x[n-1].
func passThroughChannel(x []complex64, alpha complex64) []complex64 {
	y := make([]complex64, len(x))
	for i := range x {
		y[i] = x[i]
		if i > 0 {
			y[i] += complex(real(alpha)*real(x[i-1])-imag(alpha)*imag(x[i-1]),
				real(alpha)*imag(x[i-1])+imag(alpha)*real(x[i-1]))
		}
	}
	return y
}

// meanSquaredErr reports E[|a-b|^2] over the supplied prefix.
func meanSquaredErr(a, b []complex64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return math.Inf(1)
	}
	var sum float64
	for i := range a {
		dr := float64(real(a[i]) - real(b[i]))
		di := float64(imag(a[i]) - imag(b[i]))
		sum += dr*dr + di*di
	}
	return sum / float64(len(a))
}

func TestLMSConvergesOnTwoTapChannel(t *testing.T) {
	const n = 4000
	tx := genQPSK(n, 42)
	rx := passThroughChannel(tx, complex(0.5, 0))

	// 11-tap symbol-spaced equaliser.
	eq := NewLMS(11, 0.05)

	// Train using known symbols; record |error|^2 over time.
	earlyMSE := 0.0
	const earlyWin = 200
	for i := 0; i < earlyWin; i++ {
		_, err := eq.Process(rx[i], tx[i])
		earlyMSE += float64(real(err)*real(err) + imag(err)*imag(err))
	}
	earlyMSE /= earlyWin

	for i := earlyWin; i < n-200; i++ {
		eq.Process(rx[i], tx[i])
	}

	// After convergence, run another window without updates and
	// measure the equalised MSE.
	lateMSE := 0.0
	const lateWin = 200
	for i := n - lateWin; i < n; i++ {
		y, _ := eq.Process(rx[i], tx[i])
		dr := float64(real(tx[i]) - real(y))
		di := float64(imag(tx[i]) - imag(y))
		lateMSE += dr*dr + di*di
	}
	lateMSE /= lateWin

	if lateMSE >= earlyMSE/4 {
		t.Errorf("LMS did not converge: early MSE = %g, late MSE = %g (want late < early/4)",
			earlyMSE, lateMSE)
	}
	if lateMSE > 0.05 {
		t.Errorf("post-convergence MSE = %g, want < 0.05", lateMSE)
	}
}

func TestLMSResetReturnsToPassThrough(t *testing.T) {
	eq := NewLMS(7, 0.05)
	// Drive the taps somewhere off-centre.
	tx := genQPSK(200, 7)
	rx := passThroughChannel(tx, complex(0.4, 0.1))
	for i := range tx {
		eq.Process(rx[i], tx[i])
	}
	// Reset and check the centre tap is 1+0j and the rest are 0.
	eq.Reset()
	taps := eq.Taps()
	for i, tap := range taps {
		want := complex64(0)
		if i == len(taps)/2 {
			want = complex(1, 0)
		}
		if tap != want {
			t.Errorf("tap[%d] = %v, want %v", i, tap, want)
		}
	}
}

func TestLMSPanicsOnBadConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for taps=0")
		}
	}()
	NewLMS(0, 0.01)
}

func TestCMAOpensConstantModulusConstellation(t *testing.T) {
	const n = 8000
	tx := genQPSK(n, 13)
	rx := passThroughChannel(tx, complex(0.5, 0))

	// Unit-modulus QPSK → R^2 = 1.
	eq := NewCMA(11, 0.005, 1.0)
	var lastErr float32
	const settle = 4000
	for i := 0; i < settle; i++ {
		_, e := eq.Process(rx[i])
		lastErr = e
	}

	// Measure the variance of |y|^2 around the target over a clean
	// post-settle window.
	const window = 2000
	var sumDev float64
	var sample float32
	for i := settle; i < settle+window; i++ {
		y, e := eq.Process(rx[i])
		mag2 := real(y)*real(y) + imag(y)*imag(y)
		dev := float64(mag2) - 1.0
		sumDev += dev * dev
		sample = e
	}
	rms := math.Sqrt(sumDev / float64(window))
	_ = lastErr
	_ = sample
	if rms > 0.4 {
		t.Errorf("CMA did not open the constellation: |y|^2 RMS deviation = %g, want < 0.4", rms)
	}
}

func TestCMAResetReturnsToPassThrough(t *testing.T) {
	eq := NewCMA(7, 0.005, 1.0)
	tx := genQPSK(500, 7)
	rx := passThroughChannel(tx, complex(0.3, 0.2))
	for _, s := range rx {
		eq.Process(s)
	}
	eq.Reset()
	taps := eq.Taps()
	for i, tap := range taps {
		want := complex64(0)
		if i == len(taps)/2 {
			want = complex(1, 0)
		}
		if cmplx.Abs(complex128(tap-want)) > 1e-9 {
			t.Errorf("tap[%d] = %v, want %v", i, tap, want)
		}
	}
}

func TestCMAPanicsOnBadConfig(t *testing.T) {
	for _, tc := range []struct {
		taps int
		tgt  float32
	}{
		{0, 1.0},
		{5, 0.0},
		{5, -1.0},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for taps=%d target=%v", tc.taps, tc.tgt)
				}
			}()
			NewCMA(tc.taps, 0.01, tc.tgt)
		}()
	}
}

func TestPassthroughIsBenignWithDelayAlignment(t *testing.T) {
	// With α = 0 the channel is identity. A centre-spike-initialised
	// equaliser produces a delayed copy of the input by construction
	// (y[n] = x[n - N/2]); after that delay alignment the output
	// should match the input essentially exactly when no training
	// updates are made.
	const taps = 7
	const delay = taps / 2
	tx := genQPSK(200, 99)
	rx := passThroughChannel(tx, 0)
	eq := NewLMS(taps, 0) // step size 0 → no updates
	out := make([]complex64, len(tx))
	for i := range tx {
		y, _ := eq.Process(rx[i], 0) // desired ignored when stepSize is 0
		out[i] = y
	}
	// Compare out[i] against tx[i-delay] for i >= delay; ignore the
	// initial transient where history hasn't filled.
	var mse float64
	count := 0
	for i := delay + 5; i < len(out); i++ {
		dr := float64(real(out[i]) - real(tx[i-delay]))
		di := float64(imag(out[i]) - imag(tx[i-delay]))
		mse += dr*dr + di*di
		count++
	}
	mse /= float64(count)
	if mse > 1e-9 {
		t.Errorf("identity channel passthrough MSE = %g, want ≈ 0", mse)
	}
}
