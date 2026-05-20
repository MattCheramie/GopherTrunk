package rc4

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// TestCipherKnownVectors checks the cipher against the canonical RC4
// test vectors (the widely-published "key / plaintext / ciphertext"
// triples). These exercise both the key-scheduling algorithm and the
// pseudo-random generation algorithm end to end.
func TestCipherKnownVectors(t *testing.T) {
	cases := []struct {
		key, plain, cipherHex string
	}{
		{"Key", "Plaintext", "bbf316e8d940af0ad3"},
		{"Wiki", "pedia", "1021bf0420"},
		{"Secret", "Attack at dawn", "45a01f645fc35b383552544b9bf5"},
	}
	for _, tc := range cases {
		c, err := NewCipher([]byte(tc.key))
		if err != nil {
			t.Fatalf("NewCipher(%q): %v", tc.key, err)
		}
		want, err := hex.DecodeString(tc.cipherHex)
		if err != nil {
			t.Fatalf("bad test vector hex: %v", err)
		}
		got := make([]byte, len(tc.plain))
		c.XORKeyStream(got, []byte(tc.plain))
		if !bytes.Equal(got, want) {
			t.Errorf("key=%q plain=%q: got %x, want %x", tc.key, tc.plain, got, want)
		}
	}
}

// TestCipherRFC6229 checks the keystream against RFC 6229's 40-bit-key
// vector — key 0x0102030405, keystream bytes 0..15. This covers a
// binary key and the KeyStream accessor.
func TestCipherRFC6229(t *testing.T) {
	c, err := NewCipher([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("b2396305f03dc027ccc3524a0a1118a8")
	got := c.KeyStream(16)
	if !bytes.Equal(got, want) {
		t.Errorf("RFC 6229 40-bit keystream[0:16]: got %x, want %x", got, want)
	}
}

// TestCipherRoundTrip confirms encrypt-then-decrypt with the same key
// recovers the plaintext — the property the DMR descrambler relies on.
func TestCipherRoundTrip(t *testing.T) {
	key := []byte("a-known-dmr-key")
	plain := []byte("eighteen AMBE frames of voice payload, descrambled")

	enc, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(plain))
	enc.XORKeyStream(ct, plain)
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext equals plaintext; cipher did nothing")
	}

	dec, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	pt := make([]byte, len(ct))
	dec.XORKeyStream(pt, ct)
	if !bytes.Equal(pt, plain) {
		t.Errorf("round trip: got %q, want %q", pt, plain)
	}
}

// TestKeyStreamContinuity confirms a Cipher produces one continuous
// keystream: two short pulls equal one long pull from a fresh Cipher.
func TestKeyStreamContinuity(t *testing.T) {
	key := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE}

	split, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	joined := append(split.KeyStream(7), split.KeyStream(25)...)

	whole, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	want := whole.KeyStream(32)

	if !bytes.Equal(joined, want) {
		t.Errorf("keystream not continuous: got %x, want %x", joined, want)
	}
}

func TestNewCipherBadKey(t *testing.T) {
	if _, err := NewCipher(nil); err == nil {
		t.Error("expected error for empty key")
	}
	if _, err := NewCipher([]byte{}); err == nil {
		t.Error("expected error for zero-length key")
	}
	if _, err := NewCipher(make([]byte, 257)); err == nil {
		t.Error("expected error for oversized key")
	}
}

func TestXORKeyStreamShortDst(t *testing.T) {
	c, err := NewCipher([]byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Error("expected panic when dst is shorter than src")
		}
	}()
	c.XORKeyStream(make([]byte, 2), make([]byte, 4))
}
