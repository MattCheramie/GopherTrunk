package p25

import "testing"

func TestAlgorithmName(t *testing.T) {
	cases := []struct {
		id   uint8
		want string
	}{
		{0x80, "CLEAR"},
		{0x81, "DES-OFB"},
		{0x84, "AES-256"},
		{0x85, "AES-128"},
		{0xAA, "ADP/RC4"},
		{0x00, "unknown"},
		{0xFF, "unknown"},
	}
	for _, c := range cases {
		if got := AlgorithmName(c.id); got != c.want {
			t.Errorf("AlgorithmName(0x%02X) = %q, want %q", c.id, got, c.want)
		}
	}
}

func TestAlgorithmKnown(t *testing.T) {
	cases := []struct {
		id   uint8
		want bool
	}{
		// Registered TIA-102 algorithms (clear included) are known.
		{0x80, true}, // CLEAR
		{0x81, true}, // DES-OFB
		{0x83, true}, // TDES-2
		{0x84, true}, // AES-256
		{0x85, true}, // AES-128
		{0x89, true}, // AES-256-OFB
		{0x9F, true}, // DES-XL
		{0xAA, true}, // ADP/RC4
		// Bit-error smear values that a mis-decoded Encryption Sync emits
		// must be rejected so they are never surfaced as a real key (#924).
		{0x00, false},
		{0x01, false},
		{0x37, false},
		{0x82, false},
		{0xFF, false},
	}
	for _, c := range cases {
		if got := AlgorithmKnown(c.id); got != c.want {
			t.Errorf("AlgorithmKnown(0x%02X) = %v, want %v", c.id, got, c.want)
		}
	}
}

func TestFormatAlgorithm(t *testing.T) {
	cases := []struct {
		id   uint8
		want string
	}{
		{0x84, "0x84 (AES-256)"},
		{0x80, "0x80 (CLEAR)"},
		{0x42, "0x42 (unknown)"},
	}
	for _, c := range cases {
		if got := FormatAlgorithm(c.id); got != c.want {
			t.Errorf("FormatAlgorithm(0x%02X) = %q, want %q", c.id, got, c.want)
		}
	}
}
