package phase1

import "testing"

func TestHamming10_6RoundTrip(t *testing.T) {
	for v := 0; v < 64; v++ {
		data := make([]byte, 6)
		for i := range data {
			data[i] = byte(v>>uint(5-i)) & 1
		}
		cw := encodeHamming10_6(data)
		if len(cw) != 10 {
			t.Fatalf("codeword len = %d, want 10", len(cw))
		}
		got, errs := decodeHamming10_6(cw)
		if errs != 0 {
			t.Errorf("clean codeword for %d corrected %d errors", v, errs)
		}
		for i := range data {
			if got[i] != data[i] {
				t.Errorf("v=%d: data bit %d = %d, want %d", v, i, got[i], data[i])
			}
		}
	}
}

func TestHamming10_6CorrectsSingleError(t *testing.T) {
	data := []byte{1, 0, 1, 1, 0, 1}
	cw := encodeHamming10_6(data)
	for pos := 0; pos < 10; pos++ {
		bad := append([]byte(nil), cw...)
		bad[pos] ^= 1
		got, errs := decodeHamming10_6(bad)
		if errs != 1 {
			t.Errorf("flip at %d: corrected %d errors, want 1", pos, errs)
		}
		for i := range data {
			if got[i] != data[i] {
				t.Errorf("flip at %d: data bit %d not recovered", pos, i)
			}
		}
	}
}

func TestLinkControlRoundTrip(t *testing.T) {
	in := LinkControl{
		LCFormat:       0x44,
		MFID:           0x90,
		ServiceOptions: 0xC0,
		TalkgroupID:    0x1234,
		SourceID:       0x00ABCD,
	}
	got, errs, err := ParseLinkControl(AssembleLinkControl(in))
	if err != nil {
		t.Fatalf("ParseLinkControl: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean LC corrected %d errors", errs)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

// TestLinkControlOctetLayout pins the absolute TIA-102.AABF Group Voice
// Channel User (LCO 0x00) octet positions: TG at octets 4-5, source at
// octets 6-8, service options at octet 2, MFID at octet 1. It builds the
// 9 content octets DIRECTLY (not via AssembleLinkControl) so a symmetric
// encoder/decoder swap can't mask a wrong layout — exactly the failure
// that fragmented field recordings, where the on-air TG (e.g. 0x086A =
// 2154) was misread out of octets 4-5 and the decoded talkgroup came back
// as the constant service-options byte (0x0400 = 1024).
func TestLinkControlOctetLayout(t *testing.T) {
	oct := make([]byte, lcContentOctets)
	oct[0] = LCOGroupVoiceChannelUser         // LCF
	oct[1] = 0x90                             // MFID
	oct[2] = 0x04                             // service options
	oct[3] = 0x00                             // reserved
	oct[4], oct[5] = 0x08, 0x6A               // talkgroup 0x086A = 2154
	oct[6], oct[7], oct[8] = 0x12, 0x34, 0x56 // source 0x123456

	// Encode the content bits through the layout-agnostic outer RS +
	// inner Hamming layers to produce valid on-wire blocks.
	blocks := lcInnerEncode(rsEncodeContentBits(octetsToBits(oct), 12))

	got, _, err := ParseLinkControl(blocks)
	if err != nil {
		t.Fatalf("ParseLinkControl: %v", err)
	}
	want := LinkControl{
		LCFormat:       LCOGroupVoiceChannelUser,
		MFID:           0x90,
		ServiceOptions: 0x04,
		TalkgroupID:    2154,
		SourceID:       0x123456,
	}
	if got != want {
		t.Errorf("parsed %+v, want %+v", got, want)
	}
}

func TestLinkControlCorrectsBitError(t *testing.T) {
	in := LinkControl{LCFormat: 0x00, ServiceOptions: 0x00, TalkgroupID: 0x5678, SourceID: 0x00BEEF}
	blocks := AssembleLinkControl(in)
	// One bit error per 40-bit block is within the per-codeword
	// Hamming(10,6,3) correction radius.
	blocks[0][3] ^= 1
	blocks[3][27] ^= 1
	got, errs, err := ParseLinkControl(blocks)
	if err != nil {
		t.Fatalf("ParseLinkControl: %v", err)
	}
	if errs < 2 {
		t.Errorf("corrected %d errors, want >= 2", errs)
	}
	if got != in {
		t.Errorf("LC after correction = %+v, want %+v", got, in)
	}
}

func TestParseLinkControlLengthError(t *testing.T) {
	var blocks [LDULCESBlockCount][]byte
	for i := range blocks {
		blocks[i] = make([]byte, LDULCESBlockBits)
	}
	blocks[2] = make([]byte, 39) // wrong length
	if _, _, err := ParseLinkControl(blocks); err == nil {
		t.Error("ParseLinkControl accepted a wrong-length block")
	}
}

func TestIsTalkerAliasLCO(t *testing.T) {
	// The three standard TIA-102.AABF talker-alias opcodes report
	// true; the common LCO 0x00 does not.
	for _, lcf := range []uint8{LCOTalkerAliasHeader, LCOTalkerAliasBlock1, LCOTalkerAliasBlock2} {
		if !IsTalkerAliasLCO(lcf) {
			t.Errorf("IsTalkerAliasLCO(0x%02x) = false, want true", lcf)
		}
	}
	for _, lcf := range []uint8{LCOGroupVoiceChannelUser, 0x14, 0x18, 0xFF} {
		if IsTalkerAliasLCO(lcf) {
			t.Errorf("IsTalkerAliasLCO(0x%02x) = true, want false", lcf)
		}
	}
}

// injectRSSymbolErrors replaces whole inner Hamming(10,6,3) codewords in
// the assembled LC/ES field with valid codewords carrying a *different*
// 6-bit value, so the inner layer decodes cleanly (0 inner errors) while
// exactly len(positions) outer RS symbols are wrong. This isolates the
// outer RS layer's correction behaviour from the inner Hamming layer.
func injectRSSymbolErrors(blocks [LDULCESBlockCount][]byte, positions []int) [LDULCESBlockCount][]byte {
	var field []byte
	for _, b := range blocks {
		field = append(field, b...)
	}
	for _, p := range positions {
		cur, _ := decodeHamming10_6(field[p*10 : p*10+10])
		var v byte
		for _, bit := range cur {
			v = v<<1 | bit&1
		}
		v = (v + 1) & 0x3F // a guaranteed-different 6-bit value
		nd := make([]byte, 6)
		for j := 0; j < 6; j++ {
			nd[j] = (v >> uint(5-j)) & 1
		}
		copy(field[p*10:p*10+10], encodeHamming10_6(nd))
	}
	var out [LDULCESBlockCount][]byte
	for j := range out {
		out[j] = append([]byte(nil), field[j*LDULCESBlockBits:(j+1)*LDULCESBlockBits]...)
	}
	return out
}

// TestLinkControlRSCorrection confirms the outer RS(24,12,13) layer
// recovers the correct talkgroup/source after up to t=6 symbol errors —
// the residual corruption that, before the RS layer, produced bogus
// talkgroups and false foreign-TG gating in the field capture.
func TestLinkControlRSCorrection(t *testing.T) {
	lc := LinkControl{
		LCFormat:       LCOGroupVoiceChannelUser,
		ServiceOptions: 0x00,
		TalkgroupID:    1234,
		SourceID:       567890,
	}
	clean := AssembleLinkControl(lc)
	for nerr := 0; nerr <= 6; nerr++ {
		positions := make([]int, nerr)
		for i := range positions {
			positions[i] = i * 3 // spread across distinct symbols
		}
		got, _, err := ParseLinkControl(injectRSSymbolErrors(clean, positions))
		if err != nil {
			t.Fatalf("nerr %d: unexpected error %v", nerr, err)
		}
		if got.TalkgroupID != lc.TalkgroupID || got.SourceID != lc.SourceID || got.LCFormat != lc.LCFormat {
			t.Fatalf("nerr %d: recovered %+v, want %+v", nerr, got, lc)
		}
	}
}

// TestLinkControlRSUncorrectable confirms a >t error burst is flagged
// rather than yielding a plausible-but-wrong talkgroup.
func TestLinkControlRSUncorrectable(t *testing.T) {
	lc := LinkControl{LCFormat: LCOGroupVoiceChannelUser, TalkgroupID: 4321, SourceID: 99}
	clean := AssembleLinkControl(lc)
	// 7 distinct symbol errors > t=6.
	got, _, err := ParseLinkControl(injectRSSymbolErrors(clean, []int{0, 2, 4, 6, 8, 10, 12}))
	if err == nil {
		// Aliasing onto another codeword is acceptable only if it does not
		// masquerade as the original.
		if got.TalkgroupID == lc.TalkgroupID && got.SourceID == lc.SourceID {
			t.Fatalf("7 symbol errors 'recovered' the original — impossible within t=6")
		}
		return
	}
	if err != ErrLinkControlUncorrectable {
		t.Fatalf("unexpected error %v, want ErrLinkControlUncorrectable", err)
	}
}
