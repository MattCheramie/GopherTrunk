package dmr

import (
	"encoding/hex"
	"errors"
	"testing"
)

// TestParsePIHeaderReferenceLiteral pins the PI header layout and CRC
// convention against a LITERAL 12-octet vector whose CRC was computed by an
// independent implementation of DSD-FME's bit-serial ComputeCrcCCITT
// (poly 0x1021, init 0, output inverted) with the ETSI 0x6969 PI mask — not
// by this package. A round-trip alone would pass with a wrong mask on both
// sides (the #764/#771 self-consistent trap); this literal cannot.
func TestParsePIHeaderReferenceLiteral(t *testing.T) {
	raw, err := hex.DecodeString("211005deadbeef000abc3f1d")
	if err != nil {
		t.Fatal(err)
	}
	h, err := ParsePIHeader(raw)
	if err != nil {
		t.Fatalf("ParsePIHeader: %v", err)
	}
	if h.AlgID != PIAlgRC4 || h.FID != PIFIDDMRA || h.KeyID != 5 {
		t.Fatalf("alg/fid/kid = %#x/%#x/%d, want 0x21/0x10/5", h.AlgID, h.FID, h.KeyID)
	}
	if h.MI32() != 0xDEADBEEF {
		t.Fatalf("MI = %#x, want 0xDEADBEEF", h.MI32())
	}
	if h.DstAddr != 0x000ABC {
		t.Fatalf("dst = %#x, want 0xABC", h.DstAddr)
	}
	if !h.IsRC4() || h.AlgName() != "rc4" {
		t.Fatalf("IsRC4/AlgName = %v/%q", h.IsRC4(), h.AlgName())
	}
	if got := AssemblePIHeader(h); hex.EncodeToString(got) != "211005deadbeef000abc3f1d" {
		t.Fatalf("AssemblePIHeader = %x", got)
	}
}

func TestParsePIHeaderRejectsBadCRC(t *testing.T) {
	raw, _ := hex.DecodeString("211005deadbeef000abc3f1c")
	if _, err := ParsePIHeader(raw); !errors.Is(err, ErrPIHeaderCRC) {
		t.Fatalf("err = %v, want ErrPIHeaderCRC", err)
	}
	// The ETSI-convention mask (0x6969) applied to the non-inverted CRC is
	// the wrong convention and must NOT verify — the two differ by ^0xFFFF.
	raw[11] ^= 0x01 // restore the literal's CRC, then invert all 16 bits
	raw[10] ^= 0xFF
	raw[11] ^= 0xFF
	if _, err := ParsePIHeader(raw); !errors.Is(err, ErrPIHeaderCRC) {
		t.Fatalf("inverted-mask header verified: %v", err)
	}
	if _, err := ParsePIHeader(raw[:11]); !errors.Is(err, ErrPIHeaderLength) {
		t.Fatalf("short header err = %v", err)
	}
}

func TestPIHeaderAlgNames(t *testing.T) {
	cases := []struct {
		fid, alg uint8
		want     string
		rc4      bool
	}{
		{PIFIDDMRA, PIAlgRC4, "rc4", true},
		{PIFIDDMRA, 0x01, "rc4", true}, // "DMRA compatible" (DSD-FME's low-3-bit rule)
		{PIFIDDMRA, PIAlgDES, "des", false},
		{PIFIDDMRA, PIAlgAES128, "aes-128", false},
		{PIFIDDMRA, PIAlgAES256, "aes-256", false},
		{PIFIDHytera, 0x02, "hytera-0x02", false},
		{PIFIDKirisun, 0x36, "kirisun-0x36", false},
		{0x55, 0x21, "fid-0x55-alg-0x21", false},
	}
	for _, c := range cases {
		h := PIHeader{FID: c.fid, AlgID: c.alg}
		if got := h.AlgName(); got != c.want {
			t.Errorf("fid %#x alg %#x: AlgName = %q, want %q", c.fid, c.alg, got, c.want)
		}
		if h.IsRC4() != c.rc4 {
			t.Errorf("fid %#x alg %#x: IsRC4 = %v, want %v", c.fid, c.alg, h.IsRC4(), c.rc4)
		}
	}
}
