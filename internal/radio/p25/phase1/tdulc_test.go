package phase1

import "testing"

// buildTDULCFrame synthesises a 1728-bit on-air TDULC frame whose
// status-stripped payload carries the given 9 LC content octets in the
// TDULC link-control region (FS+NID then 288-bit Golay/RS region).
func buildTDULCFrame(t *testing.T, content [lcContentOctets]byte) []byte {
	t.Helper()
	payload := make([]byte, LDUPayloadBits)
	// FS + NID region: a real DUID so LDUDuid would report TDULC. The
	// extractor itself doesn't depend on these bits, but keep them
	// faithful so the frame is well-formed.
	nid := EncodeNIDBits(0x123, DUIDTerminatorWithLC)
	copy(payload[lduNIDOffset:lduNIDOffset+LDUNIDBits], nid)

	region := encodeTDULCLCRegion(content)
	copy(payload[tdulcLCOffsetBits:tdulcLCOffsetBits+tdulcLCRegionBits], region)

	var status [LDUStatusSymbolCount]uint8 // all 0b00
	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}
	return ldu
}

// TestTDULCRoundTrip confirms the inner Golay + outer RS encode/decode
// pair recovers the 9 LC octets exactly through a full
// status-symbol-bearing TDULC frame.
func TestTDULCRoundTrip(t *testing.T) {
	want := [lcContentOctets]byte{0x15, 0x90, 0x77, 0x77, 0x07, 0x01, 0x00, 0x99, 0x02}
	ldu := buildTDULCFrame(t, want)

	lcf, got, errs, err := ExtractTDULCLinkControl(ldu)
	if err != nil {
		t.Fatalf("ExtractTDULCLinkControl: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean frame reported %d corrected errors, want 0", errs)
	}
	if lcf != want[0] {
		t.Errorf("lcf = %#x, want %#x", lcf, want[0])
	}
	if got != want {
		t.Errorf("content = % x, want % x", got, want)
	}
}

// TestTDULCRoundTripCorrectsErrors confirms the inner Golay layer
// repairs a single flipped bit per codeword.
func TestTDULCRoundTripCorrectsErrors(t *testing.T) {
	want := [lcContentOctets]byte{0x17, 0x90, 0x01, 0x9B, 0xEE, 0x00, 0x16, 0x43, 0x10}
	ldu := buildTDULCFrame(t, want)

	// Flip one payload bit inside the first Golay codeword (post-status
	// position). The status-symbol stride keeps payload bit 112 at
	// on-air bit 112 + 2*(112/70) = 114.
	onAir := 114
	ldu[onAir] ^= 1

	_, got, errs, err := ExtractTDULCLinkControl(ldu)
	if err != nil {
		t.Fatalf("ExtractTDULCLinkControl: %v", err)
	}
	if got != want {
		t.Errorf("content = % x, want % x (errs=%d)", got, want, errs)
	}
	if errs == 0 {
		t.Errorf("expected a non-zero corrected-error count after a bit flip")
	}
}

// TestTDULCFeedsMotorolaAliasBuf is the end-to-end parse check: the
// real SDRTrunk TDULC alias bytes from issue #376
// (header 159077770701009902, data 1790019BEE00164310...) drive the
// existing carriage-independent MotorolaTalkerAliasBuf exactly as the
// LDU1 path does. Independent of the TDULC FEC: it asserts the header
// fields and that the buffer accepts the data block.
func TestTDULCFeedsMotorolaAliasBuf(t *testing.T) {
	buf := NewMotorolaTalkerAliasBuf(nil)

	// HEADER 0x15: MFID 0x90, TG 0x7777=30583, blocks 7, format 1,
	// reserved 0, seq nibble 9 + checksum.
	header := [lcContentOctets]byte{0x15, 0x90, 0x77, 0x77, 0x07, 0x01, 0x00, 0x99, 0x02}
	if _, ok := buf.AddFragment(LCOTalkerAliasHeader, header); ok {
		t.Fatal("header alone should not complete an alias")
	}
	if buf.blockCount != 7 {
		t.Errorf("blockCount = %d, want 7", buf.blockCount)
	}
	if buf.sequence != 9 {
		t.Errorf("sequence = %d, want 9", buf.sequence)
	}
	if buf.talkgroupID != 30583 {
		t.Errorf("talkgroupID = %d, want 30583", buf.talkgroupID)
	}

	// DATA BLOCK 1 of seq 9: 17 90 01 9B EE 00 16 43 10. Octet 2 =
	// block number 1; octet 3 high nibble = seq 9.
	data := [lcContentOctets]byte{0x17, 0x90, 0x01, 0x9B, 0xEE, 0x00, 0x16, 0x43, 0x10}
	if _, ok := buf.AddFragment(LCOTalkerAliasBlock2, data); ok {
		t.Fatal("only 1 of 7 blocks present; alias must not complete")
	}
	if buf.blocks[0] == nil {
		t.Error("data block 1 was not stored")
	}
}
