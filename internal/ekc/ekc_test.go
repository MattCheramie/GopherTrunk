package ekc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

// referencePassword is the password testdata/kfdtool_reference.ekc was
// written with. The file was produced by KFDtool's own, unmodified
// container code (sw/control/KFDtool.Container/ContainerUtilities.cs at
// omahacommsys/KFDtool fe4dcaf, built with .NET 8 and
// System.Security.Cryptography.Xml), and read back by KFDtool's
// DecryptOuterContainer, so this pins the real format, not this package's
// idea of it. Its ciphertext ends in ISO 10126 padding (random filler), which
// a strict PKCS#7 unpad would reject.
const referencePassword = "gophertrunk-1305"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readReference(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/kfdtool_reference.ekc")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOpenKFDtoolReferenceContainer(t *testing.T) {
	c, err := Open(readReference(t), referencePassword)
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{
		{Name: "County AES", KeysetID: 1, ActiveKeyset: true, SLN: 1, KeyID: 1, AlgorithmID: 0x84, TEK: true,
			Material: mustHex(t, "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F")},
		{Name: "Fire DES", KeysetID: 1, ActiveKeyset: true, SLN: 2, KeyID: 0x2A, AlgorithmID: 0x81, TEK: true,
			Material: mustHex(t, "0123456789ABCDEF")},
		{Name: "Ops ADP", KeysetID: 1, ActiveKeyset: true, SLN: 3, KeyID: 3, AlgorithmID: 0xAA, TEK: true,
			Material: mustHex(t, "1234567890")},
		{Name: "Master KEK", KeysetID: 1, ActiveKeyset: true, SLN: 4, KeyID: 4, AlgorithmID: 0x84, KEK: true,
			Material: mustHex(t, "FFEEDDCCBBAA99887766554433221100FFEEDDCCBBAA99887766554433221100")},
		{Name: "County AES next", KeysetID: 2, SLN: 1, KeyID: 1, AlgorithmID: 0x85, TEK: true,
			Material: mustHex(t, "A0A1A2A3A4A5A6A7A8A9AAABACADAEAF")},
		{Name: "Legacy DES-XL", KeysetID: 1, ActiveKeyset: true, SLN: 6, KeyID: 6, AlgorithmID: 0x9F, TEK: true,
			Material: mustHex(t, "0011223344556677")},
		{Name: "Ops TDES", KeysetID: 1, ActiveKeyset: true, SLN: 7, KeyID: 0x1234, AlgorithmID: 0x83, TEK: true,
			Material: mustHex(t, "00112233445566778899AABBCCDDEEFF0011223344556677")},
	}
	if len(c.Keys) != len(want) {
		t.Fatalf("%d keys, want %d", len(c.Keys), len(want))
	}
	for i, w := range want {
		g := c.Keys[i]
		if g.Name != w.Name || g.KeysetID != w.KeysetID || g.ActiveKeyset != w.ActiveKeyset || g.SLN != w.SLN ||
			g.KeyID != w.KeyID || g.AlgorithmID != w.AlgorithmID || g.TEK != w.TEK || g.KEK != w.KEK ||
			!bytes.Equal(g.Material, w.Material) {
			t.Errorf("key %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestOpenRejectsWrongPassword(t *testing.T) {
	for _, pw := range []string{"", "gophertrunk-1306", referencePassword + " "} {
		if _, err := Open(readReference(t), pw); !errors.Is(err, ErrWrongPassword) {
			t.Errorf("password %q: err = %v, want ErrWrongPassword", pw, err)
		}
	}
}

func TestOpenRejectsNonContainers(t *testing.T) {
	ref := readReference(t)
	cases := map[string][]byte{
		"empty":     nil,
		"yaml":      []byte("encryption_keys:\n  - key_id: 1\n"),
		"truncated": ref[:len(ref)/2],
	}
	for name, b := range cases {
		if _, err := Open(b, referencePassword); err == nil || errors.Is(err, ErrWrongPassword) {
			t.Errorf("%s: err = %v, want a format error", name, err)
		}
	}
}
