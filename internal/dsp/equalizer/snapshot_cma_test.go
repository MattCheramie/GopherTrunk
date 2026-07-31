package equalizer

import (
	"math"
	"math/cmplx"
	"testing"
)

// π/4-DQPSK differential phases keyed by dibit (arbitrary but self-consistent
// encode/decode pair for the test).
var snapDiffPhase = [4]float64{math.Pi / 4, 3 * math.Pi / 4, -math.Pi / 4, -3 * math.Pi / 4}

func snapEncodeDQPSK(dibits []uint8) []complex128 {
	s := make([]complex128, len(dibits)+1)
	s[0] = 1
	for i, d := range dibits {
		s[i+1] = s[i] * cmplx.Rect(1, snapDiffPhase[d&3])
	}
	return s
}

func snapDecodeDQPSK(s []complex128) []uint8 {
	out := make([]uint8, len(s)-1)
	for i := 1; i < len(s); i++ {
		d := s[i] * cmplx.Conj(s[i-1])
		phi := math.Atan2(imag(d), real(d))
		best, bd := math.Inf(1), uint8(0)
		for k := 0; k < 4; k++ {
			e := math.Abs(snapAngleDiff(phi, snapDiffPhase[k]))
			if e < best {
				best, bd = e, uint8(k)
			}
		}
		out[i-1] = bd
	}
	return out
}

func snapAngleDiff(a, b float64) float64 {
	d := a - b
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d < -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

// snapDibitErr reports the fraction of dibits that differ from want, searching a
// small delay window to absorb the equalizer's group delay (the differential
// decode is invariant to a constant delay and phase).
func snapDibitErr(got, want []uint8, maxShift int) float64 {
	best := 1.0
	n := len(want)
	for sh := -maxShift; sh <= maxShift; sh++ {
		var errs, cnt int
		for i := 0; i < n; i++ {
			j := i + sh
			if j < 0 || j >= len(got) {
				continue
			}
			cnt++
			if got[j] != want[i] {
				errs++
			}
		}
		if cnt == 0 {
			continue
		}
		if r := float64(errs) / float64(cnt); r < best {
			best = r
		}
	}
	return best
}

func runSnapshotCMA(e *SnapshotCMA, src []complex64) []complex128 {
	out := make([]complex128, len(src))
	for i, x := range src {
		out[i] = complex128(e.Process(x))
	}
	return out
}

// TestSnapshotCMARecoversISIChannel is the regression test for the garbled
// TETRA recordings: a linear multipath/ISI channel smears the π/4-DQPSK
// differentials so hard decode fails; SnapshotCMA inverts it and recovery
// succeeds. Fails without the equalizer (the raw-channel error is high), passes
// with it — pinning the fix.
func TestSnapshotCMARecoversISIChannel(t *testing.T) {
	const n = 40000
	dibits := make([]uint8, n)
	x := uint32(0x1234567)
	for i := range dibits {
		x = x*1664525 + 1013904223
		dibits[i] = uint8(x >> 30)
	}
	sym := snapEncodeDQPSK(dibits)

	// A 3-tap complex multipath channel (strong ISI) + light noise.
	h := []complex128{1, complex(0.65, 0.25), complex(-0.35, 0.15)}
	chan64 := make([]complex64, len(sym))
	nz := uint32(0x9e3779b9)
	for i := range sym {
		var y complex128
		for k := range h {
			if i-k >= 0 {
				y += h[k] * sym[i-k]
			}
		}
		nz = nz*1664525 + 1013904223
		ni := (float64(nz>>16)/65535 - 0.5) * 0.06
		nz = nz*1664525 + 1013904223
		nq := (float64(nz>>16)/65535 - 0.5) * 0.06
		chan64[i] = complex(float32(real(y))+float32(ni), float32(imag(y))+float32(nq))
	}

	chan128 := make([]complex128, len(chan64))
	for i, v := range chan64 {
		chan128[i] = complex128(v)
	}
	rawErr := snapDibitErr(snapDecodeDQPSK(chan128), dibits, len(h)+2)

	out128 := runSnapshotCMA(NewSnapshotCMA(15, 6e-3, 200), chan64)
	tail := len(out128) * 3 / 4 // after convergence
	eqErr := snapDibitErr(snapDecodeDQPSK(out128[tail:]), dibits[tail:], 15)

	t.Logf("raw-channel dibit err=%.1f%%  equalized dibit err=%.1f%%", rawErr*100, eqErr*100)
	if rawErr < 0.10 {
		t.Fatalf("test channel too benign: raw err %.1f%% (want >10%% so the fix is meaningful)", rawErr*100)
	}
	if eqErr > 0.02 {
		t.Fatalf("equalizer failed to recover ISI channel: err %.1f%% (want <2%%)", eqErr*100)
	}
}

// TestSnapshotCMAHarmlessOnCleanSignal checks the equalizer leaves an
// already-clean signal essentially intact (center-spike init + convergence to
// near-identity), so enabling it does not regress good captures.
func TestSnapshotCMAHarmlessOnCleanSignal(t *testing.T) {
	const n = 20000
	dibits := make([]uint8, n)
	x := uint32(0xabcdef)
	for i := range dibits {
		x = x*1664525 + 1013904223
		dibits[i] = uint8(x >> 30)
	}
	sym := snapEncodeDQPSK(dibits)
	clean := make([]complex64, len(sym))
	for i, v := range sym {
		clean[i] = complex64(v)
	}
	out128 := runSnapshotCMA(NewSnapshotCMA(11, 6e-3, 200), clean)
	tail := len(out128) / 2
	eqErr := snapDibitErr(snapDecodeDQPSK(out128[tail:]), dibits[tail:], 11)
	t.Logf("clean-signal equalized dibit err=%.2f%%", eqErr*100)
	if eqErr > 0.01 {
		t.Fatalf("equalizer corrupted a clean signal: err %.2f%% (want <1%%)", eqErr*100)
	}
}

// TestNewSnapshotCMAPanicsOnBadConfig matches the package's constructor-
// validation convention.
func TestNewSnapshotCMAPanicsOnBadConfig(t *testing.T) {
	for _, tc := range []struct {
		name            string
		taps, snapEvery int
	}{
		{"taps<=0", 0, 200},
		{"snapEvery<=0", 11, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for %s", tc.name)
				}
			}()
			NewSnapshotCMA(tc.taps, 6e-3, tc.snapEvery)
		})
	}
}
