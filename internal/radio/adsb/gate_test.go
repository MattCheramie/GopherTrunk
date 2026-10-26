package adsb

import (
	"testing"
	"time"
)

// frameWithParity builds a 7- or 14-byte Mode-S frame whose trailing 24
// bits are CRC XOR overlay (overlay 0 = a plain CRC, as DF 11/17/18 send;
// an ICAO = an address/parity frame, as DF 0/4/5/16/20/21 send).
func frameWithParity(body []byte, overlay uint32) []byte {
	frame := append([]byte{}, body...)
	frame = append(frame, 0, 0, 0)
	crc := crc24(frame[:len(body)]) ^ overlay
	n := len(body)
	frame[n] = byte(crc >> 16)
	frame[n+1] = byte(crc >> 8)
	frame[n+2] = byte(crc)
	return frame
}

// TestProcessFrameDropsUnseenAddressParityFrames is the #1316
// regression: a DF 4 altitude reply whose overlay "decodes" to an
// address nobody has announced is noise until proven otherwise, and must
// not become an aircraft. The old code accepted every such frame
// (CRCValid = true "by construction"): 78 801 reports in three minutes on
// the reporter's unfiltered dongle, ICAOs in unallocated blocks.
func TestProcessFrameDropsUnseenAddressParityFrames(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tr := NewTracker()
	// DF 4 (0b00100 → first byte 0x20), altitude code bits arbitrary.
	df4 := frameWithParity([]byte{0x20, 0x00, 0x1B, 0x7C}, 0xF7E4DC)
	m := Decode(df4)
	if m.DF != DFAltitudeReply || m.ICAO != 0xF7E4DC || !m.ICAOInferred || m.CRCValid {
		t.Fatalf("Decode(DF4) = DF %d ICAO %06X inferred=%v crc=%v; want DF 4, F7E4DC, inferred, not CRC-valid", m.DF, m.ICAO, m.ICAOInferred, m.CRCValid)
	}
	if rep, ok := ProcessFrame(df4, tr, now); ok {
		t.Fatalf("unseen DF4 accepted as aircraft %s", rep.ICAOHex)
	}
	// No tracker at all: nothing to check against, so still dropped.
	if _, ok := ProcessFrame(df4, nil, now); ok {
		t.Fatalf("DF4 accepted with no tracker")
	}

	// Now the aircraft announces itself with a CRC-checked DF 17
	// (identification squitter for ICAO F7E4DC), and the same DF 4 is
	// accepted.
	df17 := frameWithParity([]byte{0x8D, 0xF7, 0xE4, 0xDC, 0x20, 0x2C, 0xC3, 0x71, 0xC3, 0x2C, 0xE0}, 0)
	if rep, ok := ProcessFrame(df17, tr, now); !ok || rep.ICAOHex != "F7E4DC" || !rep.CRCValid {
		t.Fatalf("DF17 not accepted: ok=%v rep=%+v", ok, rep)
	}
	rep, ok := ProcessFrame(df4, tr, now.Add(time.Second))
	if !ok {
		t.Fatalf("DF4 for a just-announced ICAO dropped")
	}
	if rep.ICAOHex != "F7E4DC" || !rep.CRCValid {
		t.Errorf("accepted DF4 report = %+v; want ICAO F7E4DC, crc_valid", rep)
	}
	// ...but not once the announcement is older than the window.
	if _, ok := ProcessFrame(df4, tr, now.Add(ICAORecentWindow+time.Second)); ok {
		t.Errorf("DF4 accepted %v after the last CRC-checked frame", ICAORecentWindow+time.Second)
	}
	// Prune drops the stale announcement.
	tr.Prune(now.Add(ICAORecentWindow + time.Second).UnixNano())
	if tr.KnownICAOs() != 0 {
		t.Errorf("KnownICAOs after prune = %d, want 0", tr.KnownICAOs())
	}
}

// TestProcessFrameLongAddressParityFramesGatedToo covers the 112-bit
// family (DF 16 / 20 / 21): same rule.
func TestProcessFrameLongAddressParityFramesGatedToo(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tr := NewTracker()
	body := []byte{0xA8, 0x00, 0x1B, 0x7C, 0x73, 0xE5, 0x02, 0xC2, 0x6C, 0xDC, 0xB5} // DF 21
	df21 := frameWithParity(body, 0xDBE0B0)
	if m := Decode(df21); m.DF != DFCommBIdentityReply || m.ICAO != 0xDBE0B0 || !m.ICAOInferred {
		t.Fatalf("Decode(DF21) = %+v", m)
	}
	if _, ok := ProcessFrame(df21, tr, now); ok {
		t.Fatalf("unseen DF21 accepted")
	}
	tr.NoteICAO(0xDBE0B0, now.UnixNano())
	if _, ok := ProcessFrame(df21, tr, now); !ok {
		t.Fatalf("DF21 for a known ICAO dropped")
	}
}

// TestDecodeAllCallReplyWithIID: a DF 11 reply to a selective
// interrogation XORs the interrogator's 7-bit IID into its parity. It is
// still a CRC-checked frame and proves its address; the old straight
// compare dropped it.
func TestDecodeAllCallReplyWithIID(t *testing.T) {
	icao := uint32(0xAABBCC)
	body := []byte{0x58, byte(icao >> 16), byte(icao >> 8), byte(icao)}
	m := Decode(frameWithParity(body, 0x25)) // IID 37
	if !m.CRCValid || m.ICAO != icao || m.Kind != KindAllCall {
		t.Fatalf("DF11 with IID 37: crc=%v icao=%06X kind=%v", m.CRCValid, m.ICAO, m.Kind)
	}
	if m.IID != 0x25 {
		t.Errorf("IID = %d, want 37", m.IID)
	}
	// Anything beyond the 7 IID bits is a real CRC failure.
	if m := Decode(frameWithParity(body, 0x80)); m.CRCValid {
		t.Errorf("DF11 with bit 7 of the parity wrong accepted as CRC-valid")
	}
	// And a CRC-checked all-call proves the address for later DF 4/5.
	tr := NewTracker()
	now := time.Unix(1_700_000_000, 0)
	if _, ok := ProcessFrame(frameWithParity(body, 0x25), tr, now); !ok {
		t.Fatalf("DF11 with IID not accepted by ProcessFrame")
	}
	if !tr.KnownICAO(icao, now.UnixNano()) {
		t.Errorf("DF11 did not register its ICAO as known")
	}
}
