package phase1

import (
	"bytes"
	"errors"
	"testing"
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

// TestExtractVoiceFramesIsStubbed: until the LDU voice-frame
// bit positions are sourced from TIA-102.BAAA-A § 8, callers
// must receive ErrLDUVoicePositionsUnknown. Pin the contract so
// the future implementation lands as a clear before/after
// behaviour change.
func TestExtractVoiceFramesIsStubbed(t *testing.T) {
	_, err := ExtractVoiceFrames(make([]byte, LDUTotalBits))
	if !errors.Is(err, ErrLDUVoicePositionsUnknown) {
		t.Errorf("err = %v, want ErrLDUVoicePositionsUnknown", err)
	}

	// Wrong-length input still surfaces ErrLDULength (the length
	// check runs before the voice-position TODO).
	_, err = ExtractVoiceFrames(make([]byte, 100))
	if !errors.Is(err, ErrLDULength) {
		t.Errorf("err = %v, want ErrLDULength for short input", err)
	}
}
