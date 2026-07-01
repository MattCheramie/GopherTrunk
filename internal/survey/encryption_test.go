package survey

import (
	"math/rand"
	"testing"
)

func TestEncryptionTriage_RandomLooksEncrypted(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5151))
	data := make([]byte, 512)
	rng.Read(data) // uniform random ≈ a strong cipher's keystream output
	enc, label := EncryptionTriage(data)
	if !enc || label == "" {
		t.Errorf("random payload: got (%v, %q), want encrypted", enc, label)
	}
}

func TestEncryptionTriage_StructuredNotFlagged(t *testing.T) {
	// Repetitive / low-entropy structured data (a scrambler over a constant, or
	// plaintext-ish bytes) must not read as encrypted.
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte('A' + (i % 8)) // 8 distinct values, highly structured
	}
	if enc, _ := EncryptionTriage(data); enc {
		t.Error("structured payload flagged as encrypted")
	}
}

func TestEncryptionTriage_ShortPayloadIgnored(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	data := make([]byte, 16) // below minEncryptBytes
	rng.Read(data)
	if enc, _ := EncryptionTriage(data); enc {
		t.Error("a payload too short to judge was flagged encrypted")
	}
}
