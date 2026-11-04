package tetra

import (
	"math/rand"
	"reflect"
	"testing"
)

// refSyncDetector is the pre-#1321 SyncDetector, verbatim: a byte ring with
// a modulo per compared dibit. The packed detector must report exactly the
// same indices.
type refSyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

func (s *refSyncDetector) Process(dst []int, src []uint8, baseIndex int) ([]int, int) {
	n := len(s.pattern)
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % n
		if s.primed < n {
			s.primed++
			continue
		}
		mismatch := 0
		idx := s.pos
		for k := 0; k < n; k++ {
			if s.hist[idx] != s.pattern[k] {
				mismatch++
				if mismatch > s.tolerance {
					break
				}
			}
			idx = (idx + 1) % n
		}
		if mismatch <= s.tolerance {
			dst = append(dst, baseIndex+i)
		}
	}
	return dst, baseIndex + len(src)
}

func TestSyncDetectorPackedMatchesByteCompare(t *testing.T) {
	rng := rand.New(rand.NewSource(1321))
	patterns := map[string][]uint8{
		"normal-11":   NormalSyncDibits(),
		"extended-15": ExtendedSyncDibits(),
		"sync-19":     SyncTrainingDibits(),
		"rot-sync-19": rotateDibits(SyncTrainingDibits(), 2),
	}
	for _, n := range []int{1, 31, 32, 33, 40} { // 33 and 40 take the byte path
		p := make([]uint8, n)
		for i := range p {
			p[i] = uint8(rng.Intn(4))
		}
		patterns["random"+string(rune('A'+n%26))+"-"+itoa(n)] = p
	}
	for name, pat := range patterns {
		for tol := 0; tol <= 4; tol++ {
			// Random dibits with the pattern planted, with errors, often
			// enough to produce matches, plus rare out-of-range values.
			src := make([]uint8, 30000)
			for i := range src {
				src[i] = uint8(rng.Intn(4))
			}
			for at := 50; at+len(pat) < len(src); at += 50 + rng.Intn(200) {
				copy(src[at:], pat)
				for e := rng.Intn(tol + 2); e > 0; e-- {
					src[at+rng.Intn(len(pat))] = uint8(rng.Intn(4))
				}
			}
			for k := 0; k < 20; k++ {
				src[rng.Intn(len(src))] = uint8(4 + rng.Intn(250))
			}
			ref := &refSyncDetector{pattern: pat, tolerance: tol, hist: make([]uint8, len(pat))}
			got := NewSyncDetector(pat, tol)
			var a, b []int
			base := 0
			for off := 0; off < len(src); {
				m := 1 + rng.Intn(900)
				if off+m > len(src) {
					m = len(src) - off
				}
				a, _ = ref.Process(a, src[off:off+m], base)
				b, base = got.Process(b, src[off:off+m], base)
				off += m
			}
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("%s tol %d: packed detector reported %d matches, byte compare %d", name, tol, len(b), len(a))
			}
			if tol >= 1 && len(a) == 0 && len(pat) > 4 {
				t.Fatalf("%s tol %d: fixture produced no matches; the test proves nothing", name, tol)
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func BenchmarkSyncDetectorSTS19(b *testing.B) {
	d := NewSyncDetector(SyncTrainingDibits(), 3)
	src := make([]uint8, 18000) // one second of TETRA dibits
	rng := rand.New(rand.NewSource(1))
	for i := range src {
		src[i] = uint8(rng.Intn(4))
	}
	var dst []int
	b.SetBytes(int64(len(src)))
	for i := 0; i < b.N; i++ {
		dst, _ = d.Process(dst[:0], src, 0)
	}
}

func BenchmarkSyncDetectorSTS19ByteCompare(b *testing.B) {
	pat := SyncTrainingDibits()
	d := &refSyncDetector{pattern: pat, tolerance: 3, hist: make([]uint8, len(pat))}
	src := make([]uint8, 18000)
	rng := rand.New(rand.NewSource(1))
	for i := range src {
		src[i] = uint8(rng.Intn(4))
	}
	var dst []int
	b.SetBytes(int64(len(src)))
	for i := 0; i < b.N; i++ {
		dst, _ = d.Process(dst[:0], src, 0)
	}
}
