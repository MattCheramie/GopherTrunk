package motorola

import "testing"

func TestLUTSize(t *testing.T) {
	if len(motorolaAliasLUT) != 256 {
		t.Errorf("motorolaAliasLUT has %d entries, want 256", len(motorolaAliasLUT))
	}
}

func TestDecodeAliasBytesDeterministic(t *testing.T) {
	enc := []byte{0x00, 0x41, 0x80, 0xFF, 0x12, 0x34, 0x56, 0x78}
	a := DecodeAliasBytes(enc)
	b := DecodeAliasBytes(enc)
	if len(a) != len(enc) {
		t.Fatalf("len = %d, want %d", len(a), len(enc))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic at %d: %02x vs %02x", i, a[i], b[i])
		}
	}
}

func TestDecodeAliasBytesEmpty(t *testing.T) {
	if got := DecodeAliasBytes(nil); len(got) != 0 {
		t.Errorf("DecodeAliasBytes(nil) = %v, want empty", got)
	}
}

func TestDecodeAlias(t *testing.T) {
	tests := []struct {
		name         string
		raw          []byte
		wantAlias    string
		wantReliable bool
	}{
		{
			// UTF-16 BE "Hi" = 00 48 00 69 — clean printable ASCII.
			name:         "clean ascii",
			raw:          []byte{0x00, 0x48, 0x00, 0x69},
			wantAlias:    "Hi",
			wantReliable: true,
		},
		{
			// Trailing NUL padding is expected and stays reliable.
			name:         "trailing nul padding",
			raw:          []byte{0x00, 0x48, 0x00, 0x69, 0x00, 0x00},
			wantAlias:    "Hi",
			wantReliable: true,
		},
		{
			// U+017F (ſ) is non-ASCII — best-effort "Hi" but unreliable.
			name:         "non-ascii latin codepoint",
			raw:          []byte{0x00, 0x48, 0x00, 0x69, 0x01, 0x7F},
			wantAlias:    "Hi",
			wantReliable: false,
		},
		{
			// U+4E2D (中) — the CJK mojibake the reporter saw (#711).
			name:         "cjk codepoint",
			raw:          []byte{0x4E, 0x2D},
			wantAlias:    "",
			wantReliable: false,
		},
		{
			// A control low byte (0x01) is corruption.
			name:         "control low byte",
			raw:          []byte{0x00, 0x48, 0x00, 0x01},
			wantAlias:    "H",
			wantReliable: false,
		},
		{
			// An odd trailing byte can't form a UTF-16 unit.
			name:         "odd length",
			raw:          []byte{0x00, 0x48, 0x00},
			wantAlias:    "H",
			wantReliable: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alias, reliable := DecodeAlias(tt.raw)
			if alias != tt.wantAlias {
				t.Errorf("alias = %q, want %q", alias, tt.wantAlias)
			}
			if reliable != tt.wantReliable {
				t.Errorf("reliable = %v, want %v", reliable, tt.wantReliable)
			}
		})
	}
}

// TestDecodeMessageSUID asserts the SUID prefix (WACN / System / Radio
// ID) is read from the front of a reassembled (packed) message. The
// values are the ones SDRTrunk reports inline for the #376 Phase 2
// alias call (RADIO:ISSI 781824.356.200062): WACN 0xBEE00, System
// 0x164, RID 200062. The message framing is identical across all
// Motorola alias carriers, so this also covers the Phase 1 SUID read.
func TestDecodeMessageSUID(t *testing.T) {
	// 7-byte SUID: WACN(20)=0xBEE00, Sys(12)=0x164, RID(24)=0x030D7E
	// (200062). These are exactly the leading 7 bytes of the SDRTrunk
	// Phase 2 fragment "BEE00164030D7E24" — confirming this framing
	// matches the reporter's inline RADIO:ISSI 781824.356.200062.
	suid := []byte{0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E}
	// One alias char (UTF-16 BE 'A' = 00 41) + 16-bit CRC placeholder.
	msgBytes := append(append([]byte{}, suid...), 0x00, 0x41, 0x00, 0x00)
	msg, ok := DecodeMessage(msgBytes)
	if !ok {
		t.Fatal("DecodeMessage returned ok=false")
	}
	if msg.WACN != 0xBEE00 {
		t.Errorf("WACN = %#x, want 0xBEE00", msg.WACN)
	}
	if msg.SystemID != 0x164 {
		t.Errorf("SystemID = %#x, want 0x164", msg.SystemID)
	}
	if msg.RadioID != 200062 {
		t.Errorf("RadioID = %d, want 200062", msg.RadioID)
	}
}

// TestDecodeMessagePopulatesAliasReliable guards that DecodeMessage
// actually computes and assigns AliasReliable from the decoded alias,
// rather than leaving it at its zero value.
func TestDecodeMessagePopulatesAliasReliable(t *testing.T) {
	suid := []byte{0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E}
	encoded := []byte{0x12, 0x34, 0x56, 0x78} // arbitrary cipher bytes
	full := append(append([]byte{}, suid...), encoded...)
	full = append(full, 0x00, 0x00) // CRC placeholder
	msg, ok := DecodeMessage(full)
	if !ok {
		t.Fatal("DecodeMessage returned ok=false")
	}
	wantAlias, wantReliable := DecodeAlias(DecodeAliasBytes(encoded))
	if msg.Alias != wantAlias {
		t.Errorf("Alias = %q, want %q", msg.Alias, wantAlias)
	}
	if msg.AliasReliable != wantReliable {
		t.Errorf("AliasReliable = %v, want %v", msg.AliasReliable, wantReliable)
	}
}

func TestDecodeMessageTooShort(t *testing.T) {
	if _, ok := DecodeMessage([]byte{0x00, 0x01}); ok {
		t.Error("expected ok=false for too-short input")
	}
}

func TestCRC16GSMMatchesRoundTrip(t *testing.T) {
	// Build a message whose trailing 16 bits are the CRC we compute,
	// then confirm DecodeMessage reports CRCOK.
	body := []byte{0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x3E, 0x00, 0x41}
	crc := crc16GSM(body)
	full := append(append([]byte{}, body...), byte(crc>>8), byte(crc))
	msg, ok := DecodeMessage(full)
	if !ok {
		t.Fatal("ok=false")
	}
	if !msg.CRCOK {
		t.Error("CRCOK = false, want true for a self-consistent CRC")
	}
}
