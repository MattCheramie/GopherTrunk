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
	wantAlias, clean := DecodeAlias(DecodeAliasBytes(encoded))
	// AliasReliable is (clean decode AND CipherVerified). The cipher is
	// verified, so this tracks the clean-ASCII flag directly (#773).
	wantReliable := clean && CipherVerified
	if msg.Alias != wantAlias {
		t.Errorf("Alias = %q, want %q", msg.Alias, wantAlias)
	}
	if msg.AliasReliable != wantReliable {
		t.Errorf("AliasReliable = %v, want %v", msg.AliasReliable, wantReliable)
	}
}

// TestAliasReliableTracksCleanDecode pins the reliability gate now that the
// cipher is verified: DecodeMessage reports an alias reliable exactly when its
// decoded bytes are clean printable ASCII (no #711 corruption hallmarks), and
// never otherwise. This is what keeps a garbled-but-CRC-surviving frame from
// surfacing a fabricated name.
func TestAliasReliableTracksCleanDecode(t *testing.T) {
	suid := []byte{0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E}
	for n := 0; n < 4096; n++ {
		enc := []byte{byte(n), byte(n >> 4), byte(n >> 8), byte(n >> 2)}
		msg := append(append(append([]byte{}, suid...), enc...), 0x00, 0x00)
		m, ok := DecodeMessage(msg)
		if !ok {
			t.Fatalf("DecodeMessage ok=false for enc=% X", enc)
		}
		_, clean := DecodeAlias(DecodeAliasBytes(m.Encoded))
		if m.AliasReliable != clean {
			t.Fatalf("AliasReliable=%v but clean-decode=%v for enc=% X",
				m.AliasReliable, clean, enc)
		}
	}
}

// TestRealSampleDecodesAliasWithValidCRC is the on-air ground-truth check:
// the real reassembled Phase-2 FACCH-S stream for RID 200062 (the #376
// SDRTrunk FRAGMENT dump, pinned byte-for-byte in
// phase2.TestMotorolaAliasReassemblesSDRTrunkFragmentStream) must decode to
// the radio's actual alias "CRIO 0062" with a matching CRC. This validates the
// recovered cipher AND the self-delimiting CRC framing against genuine air
// data, not just the synthetic oracle: the 18-byte cipher region decodes to
// clean ASCII whose "0062" is the tail of RID 200062, and CRC-16/GSM over
// SUID+cipher reproduces the on-air 0x6A96 — after which the frame is FACCH
// zero-pad. Feeding that pad in as ciphertext (the pre-fix framing) corrupted
// the cipher's length-seed and decoded to noise.
func TestRealSampleDecodesAliasWithValidCRC(t *testing.T) {
	msg := []byte{
		0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E, // SUID (RID 200062)
		0x24, 0x44, 0xF6, 0xFF, 0x2F, 0xA9, 0xAC, 0x3E, 0xC3, 0x44,
		0x32, 0xFA, 0x63, 0xCC, 0x81, 0xC3, 0xC5, 0xD9, // 18-byte cipher region
		0x6A, 0x96, // CRC-16/GSM over SUID+cipher
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // FACCH block zero-pad
	}
	m, ok := DecodeMessage(msg)
	if !ok {
		t.Fatal("DecodeMessage returned ok=false")
	}
	if m.RadioID != 200062 {
		t.Errorf("RadioID = %d, want 200062", m.RadioID)
	}
	if m.Alias != "CRIO 0062" {
		t.Errorf("Alias = %q, want %q", m.Alias, "CRIO 0062")
	}
	if !m.AliasReliable {
		t.Error("AliasReliable = false, want true for a clean real-air decode")
	}
	if !m.CRCOK {
		t.Error("CRCOK = false, want true (crc16GSM(SUID+cipher) = 0x6A96)")
	}
	// The cipher region is the 18 pre-CRC bytes — the pad is stripped.
	if len(m.Encoded) != 18 {
		t.Errorf("Encoded len = %d, want 18 (pad stripped)", len(m.Encoded))
	}
}

// TestDecodeMessageExposesEncodedCiphertext pins the cipher region on the
// decoded message (#773): Message.Encoded is exactly the 2n encoded-alias
// bytes — the SUID prefix, the trailing CRC-16, and any FACCH zero-pad
// stripped by the self-delimiting CRC. Uses the real #376 stream for RID
// 200062, whose cipher region is the 18 bytes before the 0x6A96 CRC.
func TestDecodeMessageExposesEncodedCiphertext(t *testing.T) {
	msg := []byte{
		0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E, // SUID (RID 200062)
		0x24, 0x44, 0xF6, 0xFF, 0x2F, 0xA9, 0xAC, 0x3E, 0xC3, 0x44,
		0x32, 0xFA, 0x63, 0xCC, 0x81, 0xC3, 0xC5, 0xD9, // 18-byte cipher region
		0x6A, 0x96, // CRC-16/GSM
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // FACCH zero-pad
	}
	m, ok := DecodeMessage(msg)
	if !ok {
		t.Fatal("DecodeMessage returned ok=false")
	}
	want := msg[7:25] // the 18 cipher bytes between the SUID and the CRC
	if len(m.Encoded) != len(want) {
		t.Fatalf("Encoded len = %d, want %d (pad + CRC stripped)", len(m.Encoded), len(want))
	}
	for i := range want {
		if m.Encoded[i] != want[i] {
			t.Fatalf("Encoded[%d] = %02X, want %02X\n got % X\nwant % X",
				i, m.Encoded[i], want[i], m.Encoded, want)
		}
	}
	// Must be an independent copy, not an alias into the caller's buffer.
	if len(m.Encoded) > 0 {
		m.Encoded[0] ^= 0xFF
		if msg[7] == m.Encoded[0] {
			t.Error("Encoded aliases the input slice; want an independent copy")
		}
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
