package framing

import (
	"math"
	"math/rand"
	"testing"
)

// bruteForceRM3014 is the original per-codeword re-encode search, kept as
// the reference the codebook decoders must match bit-for-bit.
func bruteForceRM3014(received []byte) ([]byte, int) {
	bestDist := 31
	var bestInfo [14]byte
	for v := 0; v < (1 << 14); v++ {
		var info [14]byte
		for i := 0; i < 14; i++ {
			info[i] = byte((v >> uint(13-i)) & 1)
		}
		cw := EncodeRM3014Tetra(info[:])
		dist := 0
		for i := 0; i < 30; i++ {
			if cw[i] != received[i]&1 {
				dist++
			}
		}
		if dist < bestDist {
			bestDist = dist
			bestInfo = info
			if dist == 0 {
				break
			}
		}
	}
	return bestInfo[:], bestDist
}

func bruteForceRM3014Soft(llr []float32) ([]byte, int, float64) {
	best, second := math.Inf(-1), math.Inf(-1)
	var bestInfo [14]byte
	for v := 0; v < (1 << 14); v++ {
		var cand [14]byte
		for i := 0; i < 14; i++ {
			cand[i] = byte((v >> uint(13-i)) & 1)
		}
		cw := EncodeRM3014Tetra(cand[:])
		var m float64
		for i := 0; i < 30; i++ {
			if cw[i] == 0 {
				m += float64(llr[i])
			} else {
				m -= float64(llr[i])
			}
		}
		switch {
		case m > best:
			second = best
			best = m
			bestInfo = cand
		case m > second:
			second = m
		}
	}
	cw := EncodeRM3014Tetra(bestInfo[:])
	dist := 0
	var energy float64
	for i := 0; i < 30; i++ {
		hb := byte(0)
		if llr[i] < 0 {
			hb = 1
		}
		if cw[i] != hb {
			dist++
		}
		energy += math.Abs(float64(llr[i]))
	}
	margin := 0.0
	if energy > 0 {
		margin = (best - second) / (2 * energy)
	}
	return bestInfo[:], dist, margin
}

// TestRM3014CodebookMatchesBruteForce pins the table-driven decoders against
// the original re-encode-every-codeword search on random words at every
// error weight, including uncorrectable ones where tie-breaking matters.
func TestRM3014CodebookMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for n := 0; n < 400; n++ {
		info := make([]byte, 14)
		for i := range info {
			info[i] = byte(rng.Intn(2))
		}
		cw := EncodeRM3014Tetra(info)
		rx := append([]byte(nil), cw...)
		for k := rng.Intn(9); k > 0; k-- {
			rx[rng.Intn(30)] ^= 1
		}
		if n%5 == 0 { // fully random words too
			for i := range rx {
				rx[i] = byte(rng.Intn(2))
			}
		}
		wantInfo, wantDist := bruteForceRM3014(rx)
		gotInfo, gotDist := DecodeRM3014Tetra(rx)
		if gotDist != wantDist || string(gotInfo) != string(wantInfo) {
			t.Fatalf("hard #%d: got %v/%d want %v/%d", n, gotInfo, gotDist, wantInfo, wantDist)
		}
		llr := make([]float32, 30)
		for i := range llr {
			llr[i] = float32(1-2*int(rx[i])) + float32(rng.NormFloat64()*0.8)
		}
		wi, wd, wm := bruteForceRM3014Soft(llr)
		gi, gd, gm := DecodeRM3014TetraSoft(llr)
		if string(gi) != string(wi) || gd != wd || math.Abs(gm-wm) > 1e-9 {
			t.Fatalf("soft #%d: got %v/%d/%.9f want %v/%d/%.9f", n, gi, gd, gm, wi, wd, wm)
		}
	}
}

// TestRM3014DecodeAllocations pins the allocation-free search path: the
// pre-codebook implementation allocated ~16 k slices per decode, which was
// the wideband TETRA pump's dominant GC load.
func TestRM3014DecodeAllocations(t *testing.T) {
	rx := make([]byte, 30)
	rx[3] = 1
	if a := testing.AllocsPerRun(20, func() { DecodeRM3014Tetra(rx) }); a > 1 {
		t.Fatalf("DecodeRM3014Tetra allocs/run = %v, want ≤ 1", a)
	}
	llr := make([]float32, 30)
	for i := range llr {
		llr[i] = 1
	}
	if a := testing.AllocsPerRun(20, func() { DecodeRM3014TetraSoft(llr) }); a > 1 {
		t.Fatalf("DecodeRM3014TetraSoft allocs/run = %v, want ≤ 1", a)
	}
}

func BenchmarkDecodeRM3014Tetra(b *testing.B) {
	rx := make([]byte, 30)
	rx[7], rx[20] = 1, 1
	for i := 0; i < b.N; i++ {
		DecodeRM3014Tetra(rx)
	}
}

func BenchmarkDecodeRM3014TetraSoft(b *testing.B) {
	llr := make([]float32, 30)
	for i := range llr {
		llr[i] = float32(1 - 2*(i%3&1))
	}
	for i := 0; i < b.N; i++ {
		DecodeRM3014TetraSoft(llr)
	}
}

func BenchmarkDecodeRM3014TetraBruteForce(b *testing.B) {
	rx := make([]byte, 30)
	rx[7], rx[20] = 1, 1
	for i := 0; i < b.N; i++ {
		bruteForceRM3014(rx)
	}
}
