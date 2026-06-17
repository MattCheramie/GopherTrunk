package trunking

import "testing"

func TestEncryptedModeString(t *testing.T) {
	cases := map[EncryptedMode]string{
		EncryptedFollow:   "follow",
		EncryptedMetadata: "metadata",
		EncryptedIgnore:   "ignore",
		EncryptedMode(99): "follow", // unknown falls back to the safe default
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("EncryptedMode(%d).String() = %q, want %q", m, got, want)
		}
	}
}

func TestParseEncryptedMode(t *testing.T) {
	cases := map[string]EncryptedMode{
		"":          EncryptedFollow,
		"follow":    EncryptedFollow,
		"FOLLOW":    EncryptedFollow,
		" metadata": EncryptedMetadata,
		"metadata":  EncryptedMetadata,
		"ignore":    EncryptedIgnore,
		"Ignore ":   EncryptedIgnore,
		"bogus":     EncryptedFollow, // unknown maps to follow, never silently stops following
	}
	for in, want := range cases {
		if got := ParseEncryptedMode(in); got != want {
			t.Errorf("ParseEncryptedMode(%q) = %v, want %v", in, got, want)
		}
	}
	// Round-trip: String of every defined mode parses back to itself.
	for _, m := range []EncryptedMode{EncryptedFollow, EncryptedMetadata, EncryptedIgnore} {
		if got := ParseEncryptedMode(m.String()); got != m {
			t.Errorf("round-trip ParseEncryptedMode(%q) = %v, want %v", m.String(), got, m)
		}
	}
}
