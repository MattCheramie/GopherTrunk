package framing

import (
	"math/rand"
	"testing"
)

// facchErasures is how a P25 Phase 2 FACCH-S burst arrives: information
// symbols 0..8 are shortened away (known zero), 9..34 are received, parity
// 35..53 is received, and parity 54..62 is punctured — not transmitted, so
// unknown at a known position. SACCH-S shortens 0..4 and punctures 57..62.
var facchErasures = []int{54, 55, 56, 57, 58, 59, 60, 61, 62}
var sacchErasures = []int{57, 58, 59, 60, 61, 62}

func randRS63Info(rng *rand.Rand, shortened int) [rs63K]byte {
	var info [rs63K]byte
	for i := shortened; i < rs63K; i++ {
		info[i] = byte(rng.Intn(64))
	}
	return info
}

func TestEncodeRS63_35RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		info := randRS63Info(rng, 0)
		cw := EncodeRS63_35(info)
		for _, s := range syndromesGF64(cw[:], rs63N, rs63N-rs63K) {
			if s != 0 {
				t.Fatalf("trial %d: encoded codeword has a non-zero syndrome", trial)
			}
		}
	}
}

// TestDecodeRS63_35FillsPuncturedParity is the case every ACCH burst hits even
// when the air is clean: nine parity symbols were never transmitted, so the
// block does not syndrome to zero until the decoder reconstructs them.
func TestDecodeRS63_35FillsPuncturedParity(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for trial := 0; trial < 200; trial++ {
		info := randRS63Info(rng, 9)
		cw := EncodeRS63_35(info)
		recv := append([]byte(nil), cw[:]...)
		for _, e := range facchErasures {
			recv[e] = 0
		}
		got, nErr, err := DecodeRS63_35(recv, facchErasures)
		if err != nil {
			t.Fatalf("trial %d: %v", trial, err)
		}
		if nErr != 0 {
			t.Errorf("trial %d: reported %d errors on a clean burst", trial, nErr)
		}
		for i := range cw {
			if got[i] != cw[i] {
				t.Fatalf("trial %d: symbol %d = %d, want %d", trial, i, got[i], cw[i])
			}
		}
	}
}

// TestDecodeRS63_35CorrectsToBudget checks the full 2·errors + erasures ≤ 28
// budget: 9 punctured parity symbols leave room for 9 symbol errors, which is
// what erasure decoding buys over declaring the shortened symbols erased too.
func TestDecodeRS63_35CorrectsToBudget(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, nErrWant := range []int{1, 4, 9} {
		for trial := 0; trial < 100; trial++ {
			info := randRS63Info(rng, 9)
			cw := EncodeRS63_35(info)
			recv := append([]byte(nil), cw[:]...)
			for _, e := range facchErasures {
				recv[e] = 0
			}
			// Corrupt distinct received positions only.
			hit := map[int]bool{}
			for len(hit) < nErrWant {
				p := 9 + rng.Intn(54-9)
				if hit[p] {
					continue
				}
				hit[p] = true
				for {
					v := byte(rng.Intn(64))
					if v != recv[p] {
						recv[p] = v
						break
					}
				}
			}
			got, nErr, err := DecodeRS63_35(recv, facchErasures)
			if err != nil {
				t.Fatalf("%d errors, trial %d: %v", nErrWant, trial, err)
			}
			if nErr != nErrWant {
				t.Errorf("%d errors, trial %d: reported %d", nErrWant, trial, nErr)
			}
			for i := range cw {
				if got[i] != cw[i] {
					t.Fatalf("%d errors, trial %d: symbol %d not repaired", nErrWant, trial, i)
				}
			}
		}
	}
}

// TestDecodeRS63_35RejectsBeyondBudget: past the radius the decoder must fail,
// not return a plausible-looking wrong codeword. That matters more here than
// the correction itself — a silently miscorrected MAC PDU is a fabricated
// grant, and the CRC-12 behind it only catches 4095 of every 4096.
func TestDecodeRS63_35RejectsBeyondBudget(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	var miscorrected int
	for trial := 0; trial < 300; trial++ {
		info := randRS63Info(rng, 9)
		cw := EncodeRS63_35(info)
		recv := append([]byte(nil), cw[:]...)
		for _, e := range facchErasures {
			recv[e] = 0
		}
		hit := map[int]bool{}
		for len(hit) < 14 {
			p := 9 + rng.Intn(54-9)
			if hit[p] {
				continue
			}
			hit[p] = true
			recv[p] = byte(rng.Intn(64))
		}
		got, _, err := DecodeRS63_35(recv, facchErasures)
		if err != nil {
			continue
		}
		for i := range cw {
			if got[i] != cw[i] {
				miscorrected++
				break
			}
		}
	}
	if miscorrected > 0 {
		t.Errorf("%d of 300 over-budget words decoded to the wrong codeword", miscorrected)
	}
}

func TestDecodeRS63_35SacchGeometry(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for trial := 0; trial < 100; trial++ {
		info := randRS63Info(rng, 5)
		cw := EncodeRS63_35(info)
		recv := append([]byte(nil), cw[:]...)
		for _, e := range sacchErasures {
			recv[e] = 0
		}
		recv[20] ^= 0x2A
		got, nErr, err := DecodeRS63_35(recv, sacchErasures)
		if err != nil {
			t.Fatalf("trial %d: %v", trial, err)
		}
		if nErr != 1 {
			t.Errorf("trial %d: reported %d errors, want 1", trial, nErr)
		}
		for i := range cw {
			if got[i] != cw[i] {
				t.Fatalf("trial %d: symbol %d not repaired", trial, i)
			}
		}
	}
}

func TestDecodeRS63_35RejectsMalformedInput(t *testing.T) {
	cw := make([]byte, rs63N)
	if _, _, err := DecodeRS63_35(cw[:10], facchErasures); err == nil {
		t.Error("short block accepted")
	}
	bad := append([]byte(nil), cw...)
	bad[3] = 0x40 // not a 6-bit symbol
	if _, _, err := DecodeRS63_35(bad, facchErasures); err == nil {
		t.Error("out-of-field symbol accepted")
	}
	if _, _, err := DecodeRS63_35(cw, []int{5, 5}); err == nil {
		t.Error("duplicate erasure accepted")
	}
	if _, _, err := DecodeRS63_35(cw, []int{63}); err == nil {
		t.Error("out-of-range erasure accepted")
	}
}
