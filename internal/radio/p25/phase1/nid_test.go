package phase1

import (
	"errors"
	"testing"
)

func TestParseNID(t *testing.T) {
	const wantNAC = uint16(0x293)
	const wantDUID = DUIDTrunkingSignaling
	bits := EncodeNIDBits(wantNAC, wantDUID)
	nid, errs, err := ParseNID(bits)
	if err != nil {
		t.Fatalf("ParseNID: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean codeword reported %d errors", errs)
	}
	if nid.NAC != wantNAC {
		t.Errorf("NAC = %X, want %X", nid.NAC, wantNAC)
	}
	if nid.DUID != wantDUID {
		t.Errorf("DUID = %s, want %s", nid.DUID, wantDUID)
	}
}

func TestDUIDString(t *testing.T) {
	cases := map[DUID]string{
		DUIDHeader: "HDU", DUIDLogicalLink1: "LDU1", DUIDTrunkingSignaling: "TSDU",
		DUIDLogicalLink2: "LDU2", DUIDTerminatorWithLC: "TDULC",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("DUID(%X).String() = %s, want %s", uint8(d), got, want)
		}
	}
}

func TestNIDFromDibits(t *testing.T) {
	bits := EncodeNIDBits(0x456, DUIDLogicalLink1)
	dibits := make([]uint8, 32)
	for i := 0; i < 32; i++ {
		dibits[i] = (bits[2*i] << 1) | bits[2*i+1]
	}
	nid, errs, err := NIDFromDibits(dibits)
	if err != nil {
		t.Fatalf("NIDFromDibits: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean codeword reported %d errors", errs)
	}
	if nid.NAC != 0x456 || nid.DUID != DUIDLogicalLink1 {
		t.Errorf("NID = %+v", nid)
	}
}

func TestParseNIDCorrectsErrors(t *testing.T) {
	// 5 bit-flips spread across the codeword should still recover.
	bits := EncodeNIDBits(0x7AB, DUIDTrunkingSignaling)
	for _, idx := range []int{1, 9, 25, 40, 55} {
		bits[idx] ^= 1
	}
	nid, errs, err := ParseNID(bits)
	if err != nil {
		t.Fatalf("ParseNID: %v", err)
	}
	if errs != 5 {
		t.Errorf("errs = %d, want 5", errs)
	}
	if nid.NAC != 0x7AB || nid.DUID != DUIDTrunkingSignaling {
		t.Errorf("NID = %+v", nid)
	}
}

func TestParseNIDRejectsUncorrectable(t *testing.T) {
	bits := EncodeNIDBits(0x000, DUIDTrunkingSignaling)
	// Flip 12 bits (one beyond t=11) plus the parity bit. The decoder
	// must surface the failure rather than returning bogus data.
	for i := 0; i < 12; i++ {
		bits[i*5] ^= 1
	}
	bits[63] ^= 1
	_, _, err := ParseNID(bits)
	if err == nil {
		t.Fatal("expected error on >11-bit corruption")
	}
	if !errors.Is(err, ErrNIDUncorrectable) && !errors.Is(err, ErrNIDParity) {
		t.Errorf("error = %v, want ErrNIDUncorrectable or ErrNIDParity", err)
	}
}

func TestEncodeNIDBitsRoundTrip(t *testing.T) {
	// Validate a handful of NACs across the full 12-bit space, all DUIDs.
	for _, nac := range []uint16{0x000, 0x001, 0x293, 0x7AB, 0xFFF} {
		for _, duid := range []DUID{DUIDHeader, DUIDTrunkingSignaling, DUIDLogicalLink1, DUIDTerminatorWithLC} {
			bits := EncodeNIDBits(nac, duid)
			nid, errs, err := ParseNID(bits)
			if err != nil {
				t.Errorf("nac=%03x duid=%X: %v", nac, uint8(duid), err)
				continue
			}
			if errs != 0 || nid.NAC != nac || nid.DUID != duid {
				t.Errorf("nac=%03x duid=%X: round-trip = %+v errs=%d", nac, uint8(duid), nid, errs)
			}
		}
	}
}

// TestEncodeNIDBitsMtAnakieVector locks down the spec BCH(63,16,11)
// generator polynomial and the per-DUID parity-bit convention against
// the only ground-truth NID we have: the 32-dibit pattern observed at
// every FSW on the Mt Anakie capture (issue #275).
//
// The site transmits NAC=0x164, DUID=7 (TSDU). After stripping the
// status symbol at on-air position 35 (NID dibit position 11), the
// on-air NID is 32 dibits long. If the encoder produces ANY other
// byte sequence here, either the BCH polynomial is wrong (pre-fix this
// disagreed in 26 of the 47 parity bits) or the per-DUID parity-bit
// table is wrong (pre-fix this set the wrong final bit). Both bugs
// silently passed the round-trip tests because the encoder and decoder
// shared the same wrong assumption.
func TestEncodeNIDBitsMtAnakieVector(t *testing.T) {
	want := []uint8{0, 1, 1, 2, 1, 0, 1, 3, 2, 1, 2, 1, 3, 0, 1, 0, 1, 1, 1, 0, 3, 2, 2, 0, 3, 3, 1, 1, 3, 0, 3, 0}
	bits := EncodeNIDBits(0x164, DUIDTrunkingSignaling)
	got := make([]uint8, 32)
	for i := 0; i < 32; i++ {
		got[i] = (bits[2*i] << 1) | bits[2*i+1]
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("dibit[%d] = %d, want %d (full got=%v, want=%v)", i, got[i], w, got, want)
			return
		}
	}
}

func TestParseNIDRejectsReservedDUID(t *testing.T) {
	for _, duid := range []DUID{0x1, 0x2, 0x4, 0x6, 0x8, 0x9, 0xB, 0xD, 0xE} {
		bits := EncodeNIDBits(0x293, duid)
		if _, _, err := ParseNID(bits); err != ErrNIDReservedDUID {
			t.Errorf("DUID %X: err = %v, want ErrNIDReservedDUID", uint8(duid), err)
		}
		dibits := make([]uint8, 32)
		for i := range dibits {
			dibits[i] = bits[2*i]<<1 | bits[2*i+1]
		}
		if _, _, _, err := NIDFromDibitsWithErrors(dibits); err != ErrNIDReservedDUID {
			t.Errorf("DUID %X via dibits: err = %v, want ErrNIDReservedDUID", uint8(duid), err)
		}
	}
	for _, duid := range []DUID{DUIDHeader, DUIDTerminator, DUIDLogicalLink1, DUIDTrunkingSignaling,
		DUIDLogicalLink2, DUIDPacketDataUnit, DUIDTerminatorWithLC} {
		if !duid.Valid() {
			t.Errorf("DUID %v reported invalid", duid)
		}
		if _, _, err := ParseNID(EncodeNIDBits(0x293, duid)); err != nil {
			t.Errorf("DUID %v: unexpected error %v", duid, err)
		}
	}
}
