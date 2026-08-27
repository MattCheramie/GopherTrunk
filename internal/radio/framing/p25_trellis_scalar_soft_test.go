package framing

import (
	"math/rand"
	"testing"
)

// dibitToBitLLRs maps a canonical channel dibit to its ideal (MSB, LSB) LLR
// pair under the soft_tetra convention (LLR > 0 ⇒ bit 0).
func dibitToBitLLRs(v uint8) (msb, lsb float32) {
	msb, lsb = 1, 1
	if (v>>1)&1 == 1 {
		msb = -1
	}
	if v&1 == 1 {
		lsb = -1
	}
	return msb, lsb
}

// TestDecodeP25TrellisSoftClean anchors the scalar soft form against the
// hard encoder: ideal LLRs recover exactly the transmitted info dibits.
func TestDecodeP25TrellisSoftClean(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 300; trial++ {
		n := 8 + rng.Intn(40)
		info := make([]uint8, n)
		for i := range info {
			info[i] = uint8(rng.Intn(4))
		}
		ch := EncodeP25Trellis(info)
		llr := make([]float32, 2*len(ch))
		for i, v := range ch {
			llr[2*i], llr[2*i+1] = dibitToBitLLRs(v)
		}
		got, _ := DecodeP25TrellisSoft(llr)
		if len(got) != len(info) {
			t.Fatalf("len %d != %d", len(got), len(info))
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("trial %d: scalar-soft decode wrong at %d (got %d want %d)", trial, i, got[i], info[i])
			}
		}
	}
}

// TestDecodeP25TrellisSoftBeatsHard: over an AWGN per-bit-LLR channel the
// scalar soft Viterbi has a strictly lower info-dibit error rate than the
// hard decoder fed the sliced dibits of the same channel — the coding-gain
// claim behind p25_phase1_soft_decision.
func TestDecodeP25TrellisSoftBeatsHard(t *testing.T) {
	const infoLen = 48 // the TSBK's info-dibit count
	for _, sigma := range []float64{0.6, 0.7} {
		rng := rand.New(rand.NewSource(9))
		var hardErr, softErr, total int
		for trial := 0; trial < 400; trial++ {
			info := make([]uint8, infoLen)
			for i := range info {
				info[i] = uint8(rng.Intn(4))
			}
			ch := EncodeP25Trellis(info)
			hardCh := make([]uint8, len(ch))
			llr := make([]float32, 2*len(ch))
			for i, v := range ch {
				m, l := dibitToBitLLRs(v)
				vm := float64(m) + rng.NormFloat64()*sigma
				vl := float64(l) + rng.NormFloat64()*sigma
				llr[2*i] = float32(vm)
				llr[2*i+1] = float32(vl)
				var msb, lsb uint8
				if vm < 0 {
					msb = 1
				}
				if vl < 0 {
					lsb = 1
				}
				hardCh[i] = msb<<1 | lsb
			}
			hard, _ := DecodeP25Trellis(hardCh)
			sft, _ := DecodeP25TrellisSoft(llr)
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
