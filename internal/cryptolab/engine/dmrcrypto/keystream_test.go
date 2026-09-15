package dmrcrypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func xor(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func TestKeystreamRoundTrip(t *testing.T) {
	t.Parallel()
	iv := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	pt := []byte("DMR ENHANCED PRIVACY TEST TRAFFIC 0123456789")
	cases := []struct {
		alg uint8
		key []byte
	}{
		{AlgRC4, []byte{0x11, 0x22, 0x33, 0x44, 0x55}},
		{AlgDESOFB, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{AlgTDES, bytes.Repeat([]byte{0x3C}, 24)},
		{AlgAES128, bytes.Repeat([]byte{0xAB}, 16)},
		{AlgAES256, bytes.Repeat([]byte{0xCD}, 32)},
	}
	for _, c := range cases {
		ks, err := Keystream(c.alg, c.key, iv, len(pt))
		if err != nil {
			t.Fatalf("%s: %v", AlgName(c.alg), err)
		}
		ct := xor(pt, ks)
		if bytes.Equal(ct, pt) {
			t.Fatalf("%s: ciphertext equals plaintext", AlgName(c.alg))
		}
		ks2, _ := Keystream(c.alg, c.key, iv, len(pt))
		if !bytes.Equal(ks, ks2) {
			t.Fatalf("%s: keystream not deterministic", AlgName(c.alg))
		}
		if got := xor(ct, ks2); !bytes.Equal(got, pt) {
			t.Fatalf("%s: decrypt mismatch: %q", AlgName(c.alg), got)
		}
	}
}

// TestRC4ReferenceVector pins the Enhanced Privacy RC4 construction against
// a literal produced by an independent RC4 (KSA/PRGA written from the
// algorithm): key‖MI, keystream bytes 256.. (issue #1187).
func TestRC4ReferenceVector(t *testing.T) {
	t.Parallel()
	ks, err := Keystream(AlgRC4, []byte{1, 2, 3, 4, 5}, []byte{0xDE, 0xAD, 0xBE, 0xEF}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(ks); got != "28e71dc649ae90d466864165d1c5" {
		t.Fatalf("keystream = %s (a stream starting 4fa6162c… skipped the 256-byte discard)", got)
	}
}

func TestRC4AppendsIV(t *testing.T) {
	t.Parallel()
	key := []byte{1, 2, 3, 4, 5}
	a, _ := Keystream(AlgRC4, key, []byte{0xAA}, 16)
	b, _ := Keystream(AlgRC4, key, []byte{0xBB}, 16)
	if bytes.Equal(a, b) {
		t.Fatal("different IVs should change the RC4 keystream (IV is appended to the key)")
	}
}

func TestUnsupported(t *testing.T) {
	t.Parallel()
	if Supported(0x99) {
		t.Fatal("0x99 should be unsupported")
	}
	if _, err := Keystream(0x99, []byte{1}, nil, 4); err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
}
