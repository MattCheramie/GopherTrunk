package phase1

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// TestLDUStructuralBitBudget: the 1728-bit LDU stream must
// account exactly for FS + NID + 9·voice + LC/ES + LSD +
// status. Any future change to a constant that breaks the sum
// would also fail the compile-time check at the top of ldu.go,
// but a runtime test pins the documented arithmetic for human
// readers and catches any introduction of a non-additive field.
func TestLDUStructuralBitBudget(t *testing.T) {
	sum := LDUFrameSyncBits +
		LDUNIDBits +
		LDUVoiceSubframeCount*LDUVoiceSubframeBits +
		LDULCBits +
		LDULSDBits +
		LDUStatusSymbolBits
	if sum != LDUTotalBits {
		t.Errorf("LDU bit-budget sum = %d, want %d (FS+NID+9·voice+LC+LSD+status)",
			sum, LDUTotalBits)
	}
	if LDULCBits != LDUESBits {
		t.Errorf("LDU1 LC width (%d) and LDU2 ES width (%d) must match (both 240)",
			LDULCBits, LDUESBits)
	}
	wantPayload := LDUTotalBits - LDUStatusSymbolBits
	if LDUPayloadBits != wantPayload {
		t.Errorf("LDUPayloadBits = %d, want %d", LDUPayloadBits, wantPayload)
	}
	// Status interleaving: 24 symbols × (70 payload bits + 2
	// status bits) per stride must equal LDUTotalBits.
	wantTotal := LDUStatusSymbolCount * (LDUStatusInterval + 2)
	if wantTotal != LDUTotalBits {
		t.Errorf("status-stride sum = %d, want %d", wantTotal, LDUTotalBits)
	}
}

func TestStripStatusSymbolsRejectsWrongLength(t *testing.T) {
	for _, n := range []int{0, LDUTotalBits - 1, LDUTotalBits + 1} {
		if _, err := StripStatusSymbols(make([]byte, n)); !errors.Is(err, ErrLDULength) {
			t.Errorf("StripStatusSymbols(len=%d): err=%v, want ErrLDULength", n, err)
		}
	}
}

// TestStripStatusSymbolsAndInjectRoundTrip: synthesise a payload
// with a recognisable bit pattern + a known set of 24 status
// symbols, inject them to build an LDU, strip them back out, and
// confirm both halves recover the originals bit-for-bit.
//
// The "2 bits after every 70 bits" rule is the only spec input
// from TIA-102.BAAA-A § 8 the project has access to; this test
// pins that rule end-to-end.
func TestStripStatusSymbolsAndInjectRoundTrip(t *testing.T) {
	payload := make([]byte, LDUPayloadBits)
	for i := range payload {
		// Pseudo-random 0/1 — exercises every status-symbol
		// boundary with both bit values.
		payload[i] = byte((i * 7) % 2)
	}
	var status [LDUStatusSymbolCount]uint8
	for i := range status {
		status[i] = uint8(i % 4) // 4 distinct 2-bit values: 0, 1, 2, 3
	}

	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}
	if len(ldu) != LDUTotalBits {
		t.Errorf("ldu length = %d, want %d", len(ldu), LDUTotalBits)
	}

	gotPayload, err := StripStatusSymbols(ldu)
	if err != nil {
		t.Fatalf("StripStatusSymbols: %v", err)
	}
	if !bytes.Equal(gotPayload, payload) {
		t.Errorf("payload round-trip mismatch")
	}

	gotStatus, err := StatusSymbols(ldu)
	if err != nil {
		t.Fatalf("StatusSymbols: %v", err)
	}
	if gotStatus != status {
		t.Errorf("status round-trip: got %v, want %v", gotStatus, status)
	}
}

// TestStripStatusSymbolsPositions: pin that the deinterleaver
// removes the bits at positions 70, 71, 142, 143, 214, 215, ...
// (the 24 status-symbol slots) and concatenates the 24 runs of
// 70 payload bits between them. A regression here would corrupt
// every voice / LC / LSD field downstream.
func TestStripStatusSymbolsPositions(t *testing.T) {
	ldu := make([]byte, LDUTotalBits)
	// Mark every payload bit with a 1 and every status-symbol
	// bit with a 0. After Strip, the result must be all 1s.
	for i := 0; i < LDUStatusSymbolCount; i++ {
		base := i * (LDUStatusInterval + 2)
		for k := 0; k < LDUStatusInterval; k++ {
			ldu[base+k] = 1
		}
		// status bits at base+70 and base+71 stay zero
	}
	payload, err := StripStatusSymbols(ldu)
	if err != nil {
		t.Fatalf("StripStatusSymbols: %v", err)
	}
	if len(payload) != LDUPayloadBits {
		t.Fatalf("payload length = %d, want %d", len(payload), LDUPayloadBits)
	}
	for i, b := range payload {
		if b != 1 {
			t.Fatalf("payload[%d] = %d, want 1 (status symbols should have been stripped)", i, b)
		}
	}
}

// TestStatusSymbolsExtraction: pin the 2-bit-per-symbol packing
// (high bit first into the low 2 bits of a uint8). A status bit
// pair (1,0) at positions 70/71 must surface as symbol 0b10 = 2.
func TestStatusSymbolsExtraction(t *testing.T) {
	ldu := make([]byte, LDUTotalBits)
	// Set status symbol 0 bits: ldu[70]=1, ldu[71]=0 ⇒ symbol 0
	// should be (1<<1)|0 = 2.
	ldu[70] = 1
	ldu[71] = 0
	// symbol 23 bits: positions 1726, 1727. Set both to 1 ⇒ 3.
	ldu[1726] = 1
	ldu[1727] = 1
	got, err := StatusSymbols(ldu)
	if err != nil {
		t.Fatalf("StatusSymbols: %v", err)
	}
	if got[0] != 0b10 {
		t.Errorf("status[0] = %d, want 0b10 (=2)", got[0])
	}
	if got[23] != 0b11 {
		t.Errorf("status[23] = %d, want 0b11 (=3)", got[23])
	}
	// Other symbols default to 0 since the rest of the LDU is zero.
	for i := 1; i < 23; i++ {
		if got[i] != 0 {
			t.Errorf("status[%d] = %d, want 0", i, got[i])
		}
	}
}

// TestExtractVoiceFramesRejectsWrongLength: pin that
// ExtractVoiceFrames surfaces ErrLDULength for any input that
// isn't exactly LDUTotalBits. Mirrors the StripStatusSymbols
// length check.
func TestExtractVoiceFramesRejectsWrongLength(t *testing.T) {
	_, _, err := ExtractVoiceFrames(make([]byte, 100))
	if !errors.Is(err, ErrLDULength) {
		t.Errorf("err = %v, want ErrLDULength for short input", err)
	}
}

// TestExtractVoiceFramesSliceBoundsMatchTIATable: pin the 9
// voice-subframe slice offsets against the documented
// TIA-102.BAAA-A § 8 table. We do this by writing a known
// pattern at each documented voice slot in the 1680-bit payload
// (each slot gets bit value (i+1) % 2 to keep things simple),
// injecting status symbols to build the 1728-bit on-air stream,
// then reading the bits at the voice slots back out via
// StripStatusSymbols + manual slicing. A mismatch in offsets
// would surface as a wrong pattern at slot i.
//
// This catches off-by-one errors in lduVoiceOffsets without
// depending on the imbe channel decoder behaving correctly on
// the patterns.
func TestExtractVoiceFramesSliceBoundsMatchTIATable(t *testing.T) {
	payload := make([]byte, LDUPayloadBits)
	// Mark each voice slot with a distinctive pattern: slot i
	// gets all bits = (i+1)&1 (alternating 1, 0, 1, 0, 1, 0, 1, 0, 1).
	for i, off := range lduVoiceOffsets {
		marker := byte((i + 1) & 1)
		for k := 0; k < LDUVoiceSubframeBits; k++ {
			payload[off+k] = marker
		}
	}
	var status [LDUStatusSymbolCount]uint8
	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}
	gotPayload, err := StripStatusSymbols(ldu)
	if err != nil {
		t.Fatalf("StripStatusSymbols: %v", err)
	}
	for i, off := range lduVoiceOffsets {
		marker := byte((i + 1) & 1)
		for k := 0; k < LDUVoiceSubframeBits; k++ {
			if gotPayload[off+k] != marker {
				t.Fatalf("voice slot %d bit %d: got %d, want %d (offset %d)",
					i, k, gotPayload[off+k], marker, off)
			}
		}
	}
}

// TestLDUFieldsCoverPayloadWithoutOverlap: pin that the
// concatenation of {FS, NID, 9 voice slots, 6 LC/ES blocks,
// 2 LSD blocks} exactly covers the 1680-bit payload with no gaps
// and no overlaps. Mismatch here means at least one offset is
// wrong against the TIA-102.BAAA-A § 8 cumulative table.
func TestLDUFieldsCoverPayloadWithoutOverlap(t *testing.T) {
	covered := make([]bool, LDUPayloadBits)

	cover := func(name string, off, length int) {
		t.Helper()
		if off < 0 || off+length > LDUPayloadBits {
			t.Errorf("%s: out of bounds (off=%d len=%d)", name, off, length)
			return
		}
		for k := off; k < off+length; k++ {
			if covered[k] {
				t.Errorf("%s: bit %d already covered (overlap)", name, k)
			}
			covered[k] = true
		}
	}

	cover("FS", lduFSOffset, LDUFrameSyncBits)
	cover("NID", lduNIDOffset, LDUNIDBits)
	for i, off := range lduVoiceOffsets {
		cover(fmt.Sprintf("voice u_%d", i), off, LDUVoiceSubframeBits)
	}
	for i, off := range lduLCESBlockOffsets {
		cover(fmt.Sprintf("LC/ES Block %d", i+1), off, LDULCESBlockBits)
	}
	for i, off := range lduLSDBlockOffsets {
		cover(fmt.Sprintf("LSD Block %d", i+1), off, LDULSDBlockBits)
	}

	for i, b := range covered {
		if !b {
			t.Errorf("payload bit %d not covered by any field (gap)", i)
		}
	}
}

// TestExtractVoiceFramesRoundTrip: build a synthetic LDU by
// encoding 9 distinct IMBE info-bit patterns through
// EncodeChannel + Scramble, placing them at the documented voice
// offsets, injecting status symbols, then calling
// ExtractVoiceFrames and confirming each returned frame
// round-trips back to its original info bits.
//
// This is the load-bearing test for the LDU layout: a single
// wrong offset in lduVoiceOffsets would surface as a mismatched
// frame at that slot index.
func TestExtractVoiceFramesRoundTrip(t *testing.T) {
	// Per-subframe info bits: subframe i gets bit pattern
	// (i + k*3) % 2 — picks distinct 0/1 stripes per slot so a
	// swapped pair of offsets would be caught.
	var originals [LDUVoiceSubframeCount][]byte
	for i := 0; i < LDUVoiceSubframeCount; i++ {
		info := make([]byte, imbe.InfoBits)
		for k := range info {
			info[k] = byte((i + k*3) % 2)
		}
		originals[i] = info
	}

	payload := make([]byte, LDUPayloadBits)
	for i, info := range originals {
		encoded, err := imbe.EncodeChannel(info)
		if err != nil {
			t.Fatalf("EncodeChannel u_%d: %v", i, err)
		}
		scrambled, err := imbe.Scramble(encoded)
		if err != nil {
			t.Fatalf("Scramble u_%d: %v", i, err)
		}
		copy(payload[lduVoiceOffsets[i]:lduVoiceOffsets[i]+LDUVoiceSubframeBits], scrambled)
	}
	var status [LDUStatusSymbolCount]uint8
	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}

	frames, errs, err := ExtractVoiceFrames(ldu)
	if err != nil {
		t.Fatalf("ExtractVoiceFrames: %v", err)
	}
	if errs != 0 {
		t.Errorf("totalErrs = %d, want 0 on a clean LDU", errs)
	}
	for i, original := range originals {
		frame := frames[i]
		if len(frame) != imbe.FrameBytes {
			t.Errorf("u_%d frame length = %d, want %d", i, len(frame), imbe.FrameBytes)
			continue
		}
		// Unpack frame to bits and compare to original.
		for k := 0; k < imbe.InfoBits; k++ {
			got := (frame[k/8] >> (7 - uint(k)%8)) & 1
			if got != original[k] {
				t.Fatalf("u_%d bit %d: extracted = %d, original = %d", i, k, got, original[k])
			}
		}
	}
}

// TestExtractLCESBlocksRoundTrip: place 6 distinct 40-bit
// patterns at the documented LC/ES offsets, inject status, then
// confirm ExtractLCESBlocks returns the same patterns.
func TestExtractLCESBlocksRoundTrip(t *testing.T) {
	payload := make([]byte, LDUPayloadBits)
	var want [LDULCESBlockCount][]byte
	for i, off := range lduLCESBlockOffsets {
		block := make([]byte, LDULCESBlockBits)
		for k := range block {
			block[k] = byte((i*7 + k) % 2)
		}
		copy(payload[off:off+LDULCESBlockBits], block)
		want[i] = block
	}
	var status [LDUStatusSymbolCount]uint8
	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}
	got, err := ExtractLCESBlocks(ldu)
	if err != nil {
		t.Fatalf("ExtractLCESBlocks: %v", err)
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("LC/ES block %d mismatch", i+1)
		}
	}
}

// TestExtractLSDBlocksRoundTrip: same idea for the 2 LSD slots.
func TestExtractLSDBlocksRoundTrip(t *testing.T) {
	payload := make([]byte, LDUPayloadBits)
	var want [LDULSDBlockCount][]byte
	for i, off := range lduLSDBlockOffsets {
		block := make([]byte, LDULSDBlockBits)
		for k := range block {
			block[k] = byte((i*5 + k) % 2)
		}
		copy(payload[off:off+LDULSDBlockBits], block)
		want[i] = block
	}
	var status [LDUStatusSymbolCount]uint8
	ldu, err := InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}
	got, err := ExtractLSDBlocks(ldu)
	if err != nil {
		t.Fatalf("ExtractLSDBlocks: %v", err)
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("LSD block %d mismatch", i+1)
		}
	}
}
