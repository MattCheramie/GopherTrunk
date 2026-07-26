package framing

import (
	"math"
	"math/rand"
	"testing"
)

// dibitToDiag maps a canonical channel dibit to its ideal soft point in the
// diagonal frame DecodeP25TrellisSoftC expects: b0 = Re<0, b1 = Im<0.
func dibitToDiag(v uint8) complex64 {
	b0 := float64(v & 1)
	b1 := float64((v >> 1) & 1)
	return complex(float32((1-2*b0)/math.Sqrt2), float32((1-2*b1)/math.Sqrt2))
}

// TestDecodeP25TrellisSoftCClean: on ideal (noiseless) soft points the soft
// Viterbi recovers exactly the transmitted info dibits — anchors the bit
// convention against the hard encoder/decoder.
func TestDecodeP25TrellisSoftCClean(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 300; trial++ {
		n := 8 + rng.Intn(40)
		info := make([]uint8, n)
		for i := range info {
			info[i] = uint8(rng.Intn(4))
		}
		ch := EncodeP25Trellis(info)
		soft := make([]complex64, len(ch))
		for i, v := range ch {
			soft[i] = dibitToDiag(v)
		}
		got, _ := DecodeP25TrellisSoftC(soft)
		if len(got) != len(info) {
			t.Fatalf("len %d != %d", len(got), len(info))
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("trial %d: soft-C decode wrong at %d (got %d want %d)", trial, i, got[i], info[i])
			}
		}
	}
}

// TestDecodeP25TrellisSoftCBeatsHard: over an AWGN diagonal channel the true
// soft Viterbi has a strictly lower info-dibit error rate than the hard decoder.
func TestDecodeP25TrellisSoftCBeatsHard(t *testing.T) {
	const infoLen = 72
	for _, sigma := range []float64{0.5, 0.6, 0.7} {
		rng := rand.New(rand.NewSource(9))
		var hardErr, softErr, total int
		for trial := 0; trial < 400; trial++ {
			info := make([]uint8, infoLen)
			for i := range info {
				info[i] = uint8(rng.Intn(4))
			}
			ch := EncodeP25Trellis(info)
			hardCh := make([]uint8, len(ch))
			soft := make([]complex64, len(ch))
			for i, v := range ch {
				ideal := dibitToDiag(v)
				re := float64(real(ideal)) + rng.NormFloat64()*sigma
				im := float64(imag(ideal)) + rng.NormFloat64()*sigma
				soft[i] = complex(float32(re), float32(im))
				var b0, b1 uint8
				if re < 0 {
					b0 = 1
				}
				if im < 0 {
					b1 = 1
				}
				hardCh[i] = 2*b1 + b0
			}
			hard, _ := DecodeP25Trellis(hardCh)
			sft, _ := DecodeP25TrellisSoftC(soft)
			for i := 0; i < infoLen; i++ {
				if hard[i] != info[i] {
					hardErr++
				}
				if sft[i] != info[i] {
					softErr++
				}
				total++
			}
		}
		hBER := float64(hardErr) / float64(total)
		sBER := float64(softErr) / float64(total)
		t.Logf("sigma=%.2f  hard BER=%.4f  soft BER=%.4f  (%.1f%% reduction)",
			sigma, hBER, sBER, 100*(hBER-sBER)/hBER)
		if sBER >= hBER {
			t.Errorf("sigma=%.2f: soft BER %.4f not better than hard %.4f", sigma, sBER, hBER)
		}
	}
}
