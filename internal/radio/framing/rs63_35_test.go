package framing

import (
	"bytes"
	"math/rand"
	"testing"
)

// ezpwdRS6335 is a literal codeword from ezpwd::RS<63,35> (the decoder OP25's
// Phase 2 path uses: poly 0x43, FCR 1, PRIM 1), encoding data symbol i =
// (37i+11) mod 64 with the first nine zeroed, as FACCH shortening does.
var ezpwdRS6335 = []byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 24, 61, 34, 7, 44, 17, 54, 27, 0, 37, 10, 47,
	20, 57, 30, 3, 40, 13, 50, 23, 60, 33, 6, 43, 16, 53, 30, 28, 6, 47, 6,
	44, 10, 53, 6, 2, 42, 7, 13, 16, 63, 13, 50, 1, 54, 6, 22, 2, 13, 10, 33,
	36, 21, 43,
}

func TestEncodeRS63_35MatchesEzpwd(t *testing.T) {
	got, err := EncodeRS63_35(ezpwdRS6335[:RS6335K])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, ezpwdRS6335) {
		t.Fatalf("parity differs from ezpwd:\n got %v\nwant %v", got[RS6335K:], ezpwdRS6335[RS6335K:])
	}
}

// The case ezpwd itself decodes (5 errors + OP25's 18 FACCH erasures).
func TestDecodeRS63_35EzpwdCase(t *testing.T) {
	r := append([]byte(nil), ezpwdRS6335...)
	for _, p := range []int{10, 20, 30, 40, 50} {
		r[p] ^= 0x15
	}
	for i := 54; i < 63; i++ {
		r[i] = 0
	}
	er := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 54, 55, 56, 57, 58, 59, 60, 61, 62}
	got, n, err := DecodeRS63_35(r, er)
	if err != nil || !bytes.Equal(got, ezpwdRS6335) {
		t.Fatalf("err=%v corrected=%d match=%v", err, n, bytes.Equal(got, ezpwdRS6335))
	}
}

func TestDecodeRS63_35ErrorsAndErasures(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for trial := 0; trial < 400; trial++ {
		data := make([]byte, RS6335K)
		for i := range data {
			data[i] = byte(rng.Intn(64))
		}
		cw, _ := EncodeRS63_35(data)
		e := rng.Intn(20)
		tErr := rng.Intn((28-e)/2 + 1)
		perm := rng.Perm(RS6335N)
		r := append([]byte(nil), cw...)
		er := perm[:e]
		for _, p := range er {
			r[p] = byte(rng.Intn(64))
		}
		for _, p := range perm[e : e+tErr] {
			r[p] ^= byte(1 + rng.Intn(63))
		}
		got, _, err := DecodeRS63_35(r, er)
		if err != nil || !bytes.Equal(got, cw) {
			t.Fatalf("trial %d: %d erasures + %d errors: err=%v", trial, e, tErr, err)
		}
	}
}

func TestDecodeRS63_35RejectsBeyondRadius(t *testing.T) {
	cw, _ := EncodeRS63_35(make([]byte, RS6335K))
	r := append([]byte(nil), cw...)
	for p := 0; p < 15; p++ { // 15 errors > t = 14
		r[p*4] ^= 0x2A
	}
	if got, _, err := DecodeRS63_35(r, nil); err == nil && bytes.Equal(got, cw) {
		t.Fatal("decoded 15 errors with no erasures")
	}
}
