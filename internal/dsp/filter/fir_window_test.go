package filter

import (
	"math/rand"
	"testing"
)

// ringFIR is the original modulo-indexed ring-buffer implementation, kept as
// the bit-exact reference for the mirrored-window Process.
type ringFIR struct {
	taps    []float32
	hist    []complex64
	histPos int
}

func (f *ringFIR) process(src []complex64) []complex64 {
	dst := make([]complex64, len(src))
	N := len(f.taps)
	for i, x := range src {
		f.hist[f.histPos] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		var accI, accQ float32
		idx := f.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			s := f.hist[idx]
			h := f.taps[k]
			accI += h * real(s)
			accQ += h * imag(s)
			idx--
			if idx < 0 {
				idx = N - 1
			}
		}
		dst[i] = complex(accI, accQ)
	}
	return dst
}

// TestFIRMirroredWindowIsBitExact pins the mirrored-history Process against
// the original ring-buffer loop: same taps, same chunking, identical float32
// output (the summation order is preserved on purpose so every golden that
// runs through a FIR stays valid).
func TestFIRMirroredWindowIsBitExact(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, n := range []int{1, 2, 7, 33, 128, 257} {
		taps := make([]float32, n)
		for i := range taps {
			taps[i] = float32(rng.NormFloat64())
		}
		fast := NewFIR(taps)
		ref := &ringFIR{taps: taps, hist: make([]complex64, n)}
		var dst []complex64
		for chunk := 0; chunk < 20; chunk++ {
			src := make([]complex64, rng.Intn(300)+1)
			for i := range src {
				src[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
			}
			dst = fast.Process(dst, src)
			want := ref.process(src)
			for i := range want {
				if dst[i] != want[i] {
					t.Fatalf("taps=%d chunk=%d sample=%d: got %v want %v", n, chunk, i, dst[i], want[i])
				}
			}
		}
		fast.Reset()
		ref = &ringFIR{taps: taps, hist: make([]complex64, n)}
		src := make([]complex64, 50)
		src[0] = 1
		dst = fast.Process(dst, src)
		want := ref.process(src)
		for i := range want {
			if dst[i] != want[i] {
				t.Fatalf("after Reset taps=%d sample=%d: got %v want %v", n, i, dst[i], want[i])
			}
		}
	}
}

func BenchmarkFIR129(b *testing.B) {
	taps := make([]float32, 129)
	for i := range taps {
		taps[i] = float32(i%7) * 0.01
	}
	f := NewFIR(taps)
	src := make([]complex64, 4096)
	var dst []complex64
	b.SetBytes(int64(len(src) * 8))
	for i := 0; i < b.N; i++ {
		dst = f.Process(dst, src)
	}
}
