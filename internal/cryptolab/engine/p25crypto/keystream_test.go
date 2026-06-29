package p25crypto

import (
	"bytes"
	"testing"
)

func xor(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// TestKeystreamRoundTrip confirms each supported algorithm produces a
// deterministic keystream that decrypts what it encrypts.
func TestKeystreamRoundTrip(t *testing.T) {
	t.Parallel()
	mi := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	pt := []byte("THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG 0123456789")
	cases := []struct {
		alg uint8
		key []byte
	}{
		{AlgADP, []byte{0x11, 0x22, 0x33, 0x44, 0x55}},
		{AlgDESOFB, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{AlgAES128, bytes.Repeat([]byte{0xAB}, 16)},
		{AlgAES256, bytes.Repeat([]byte{0xCD}, 32)},
	}
	for _, c := range cases {
		ks, err := Keystream(c.alg, c.key, mi, len(pt))
		if err != nil {
			t.Fatalf("%s: %v", AlgName(c.alg), err)
		}
		if len(ks) != len(pt) {
			t.Fatalf("%s: keystream len %d, want %d", AlgName(c.alg), len(ks), len(pt))
		}
		ct := xor(pt, ks)
		if bytes.Equal(ct, pt) {
			t.Fatalf("%s: ciphertext equals plaintext (zero keystream?)", AlgName(c.alg))
		}
		// Same key+MI must reproduce the keystream and decrypt.
		ks2, _ := Keystream(c.alg, c.key, mi, len(pt))
		if !bytes.Equal(ks, ks2) {
			t.Fatalf("%s: keystream not deterministic", AlgName(c.alg))
		}
		if got := xor(ct, ks2); !bytes.Equal(got, pt) {
			t.Fatalf("%s: decrypt mismatch:\n got %q\nwant %q", AlgName(c.alg), got, pt)
		}
	}
}

func TestKeystreamRejectsBadKeyLen(t *testing.T) {
	t.Parallel()
	if _, err := Keystream(AlgDESOFB, []byte{1, 2, 3}, []byte{1, 2, 3, 4, 5, 6, 7, 8}, 16); err == nil {
		t.Fatal("expected error for wrong DES key length")
	}
	if _, err := Keystream(0x00, []byte{1}, nil, 8); err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
}

func TestDifferentMIDifferentKeystream(t *testing.T) {
	t.Parallel()
	key := []byte{0x11, 0x22, 0x33, 0x44, 0x55}
	a, _ := Keystream(AlgADP, key, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, 32)
	b, _ := Keystream(AlgADP, key, []byte{9, 8, 7, 6, 5, 4, 3, 2, 1}, 32)
	if bytes.Equal(a, b) {
		t.Fatal("distinct MIs produced the same keystream")
	}
}

func TestDefaultKeysSized(t *testing.T) {
	t.Parallel()
	for _, alg := range []uint8{AlgADP, AlgDESOFB, AlgAES128, AlgAES256} {
		keys := DefaultKeys(alg)
		if len(keys) == 0 {
			t.Fatalf("%s: no default keys", AlgName(alg))
		}
		for _, k := range keys {
			if len(k) != KeySize(alg) {
				t.Fatalf("%s: default key size %d, want %d", AlgName(alg), len(k), KeySize(alg))
			}
		}
	}
}
