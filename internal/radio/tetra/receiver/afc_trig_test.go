package receiver

import (
	"math"
	"math/rand"
	"testing"
)

// The pre-#1321 derotate: math.Cos / math.Sin per sample, a float32 phasor.
func derotateReference(block []complex64, perSample float64) {
	theta := 0.0
	for i := range block {
		rot := complex(float32(math.Cos(-theta)), float32(math.Sin(-theta)))
		block[i] *= rot
		theta += perSample
	}
}

// The pre-#1321 estimateOmega: atan2 then cos/sin of 4·arg per symbol.
func estimateOmegaReference(block []complex64, sps int) float64 {
	var sr, si float64
	for k := sps; k < len(block); k += sps {
		d := block[k] * conjf(block[k-sps])
		p := 4 * float64(anglef(d))
		sr += math.Cos(p)
		si += math.Sin(p)
	}
	if sr == 0 && si == 0 {
		return 0
	}
	off := (math.Atan2(si, sr) - math.Pi) / 4
	for off > math.Pi/4 {
		off -= math.Pi / 2
	}
	for off <= -math.Pi/4 {
		off += math.Pi / 2
	}
	return off
}

func TestDerotateMatchesTrigReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1321))
	for _, omega := range []float64{0, 0.01, -0.2, math.Pi / 4, -math.Pi / 4, 0.7853} {
		for _, n := range []int{1, 63, 64, 65, 1024, 8 * 255} {
			a := make([]complex64, n)
			for i := range a {
				a[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
			}
			b := append([]complex64(nil), a...)
			derotateReference(a, omega/8)
			derotateBlock(b, omega/8)
			for i := range a {
				d := a[i] - b[i]
				if e := math.Hypot(float64(real(d)), float64(imag(d))); e > 2e-5*(1+math.Hypot(float64(real(a[i])), float64(imag(a[i])))) {
					t.Fatalf("omega %g n %d sample %d: %v vs %v", omega, n, i, a[i], b[i])
				}
			}
		}
	}
}

func TestEstimateOmegaMatchesTrigReference(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	const sps = 8
	for trial := 0; trial < 200; trial++ {
		off := (rng.Float64() - 0.5) * math.Pi / 2.2
		block := make([]complex64, 64*sps)
		phase := 0.0
		for s := 0; s < 64; s++ {
			phase += math.Pi/4 + float64(rng.Intn(4))*math.Pi/2 + off
			for k := 0; k < sps; k++ {
				i := s*sps + k
				nz := complex(rng.NormFloat64()*0.05, rng.NormFloat64()*0.05)
				v := complex(math.Cos(phase), math.Sin(phase)) + nz
				block[i] = complex64(v)
			}
		}
		if trial%50 == 0 {
			block[sps] = 0 // a zero product: arg 0 in both
		}
		want := estimateOmegaReference(block, sps)
		got := estimateOmega(block, sps)
		if math.Abs(want-got) > 1e-5 {
			t.Fatalf("trial %d: estimateOmega %g, reference %g", trial, got, want)
		}
	}
}

func TestTETRAReceiverFiltersAreFolded(t *testing.T) {
	r := New(Options{SampleRateHz: 144000, EnableChannelFilter: true, DibitSink: func([]uint8, int) {}})
	if r.chanFilt == nil || !r.chanFilt.Folded() {
		t.Error("channel filter is not on the folded path")
	}
}
