package phase1

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// dibitsFor builds a length-n dibit slice with values copied
// from `pattern` (cycling). Used by the synthetic-stream tests
// below.
func dibitsFor(n int, pattern ...uint8) []uint8 {
	out := make([]uint8, n)
	if len(pattern) == 0 {
		return out
	}
	for i := range out {
		out[i] = pattern[i%len(pattern)]
	}
	return out
}

// makeLDUDibits builds an 864-dibit slice that begins with the
// canonical FrameSyncWord and is padded with the supplied filler
// dibits (cycling) for the remaining 840 positions. The result
// can be fed to LDUAssembler.Process to test FSW detection +
// trailing-collection together.
func makeLDUDibits(filler ...uint8) []uint8 {
	out := make([]uint8, LDUDibitCount)
	copy(out, FrameSyncWord[:])
	if len(filler) > 0 {
		for i := 24; i < LDUDibitCount; i++ {
			out[i] = filler[(i-24)%len(filler)]
		}
	}
	return out
}

// TestLDUAssemblerEmitsOnClean864: a clean FSW + 840 trailing
// dibits triggers exactly one sink invocation with a 1728-bit
// LDU. The first 24 dibits' bit-representation (48 bits) must
// match the canonical frame-sync bits.
func TestLDUAssemblerEmitsOnClean864(t *testing.T) {
	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(makeLDUDibits(0, 1, 2, 3))

	if len(got) != 1 {
		t.Fatalf("sink invoked %d times, want 1", len(got))
	}
	if len(got[0]) != LDUTotalBits {
		t.Errorf("LDU length = %d bits, want %d", len(got[0]), LDUTotalBits)
	}
	// First 48 bits must match the canonical FrameSyncBits.
	wantFS := FrameSyncBits()
	for i, b := range wantFS {
		if got[0][i] != b {
			t.Fatalf("bit %d of LDU = %d, want %d (frame sync mismatch)", i, got[0][i], b)
		}
	}
	if a.Buffered() != 0 {
		t.Errorf("Buffered() = %d after emission, want 0 (compaction failed)", a.Buffered())
	}
}

// TestLDUAssemblerSkipsLeadingGarbage: feed 100 dibits of
// non-FSW junk, then an FSW + 840 trailing dibits. The sink
// must fire exactly once with the LDU that begins at the FSW —
// the leading garbage is dropped, not prepended.
func TestLDUAssemblerSkipsLeadingGarbage(t *testing.T) {
	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(dibitsFor(100, 2, 0, 3, 1)) // 100 random dibits, never matches FSW
	a.Process(makeLDUDibits(0))

	if len(got) != 1 {
		t.Fatalf("sink invoked %d times, want 1", len(got))
	}
	// First dibit of the emitted LDU's bits must be the first
	// bit of the FSW (which is 0x55 high nibble's bit = 0).
	wantFirst := framing.DibitsToBits(FrameSyncWord[:1])[0]
	if got[0][0] != wantFirst {
		t.Errorf("LDU starts at bit %d, want %d (FSW first bit)", got[0][0], wantFirst)
	}
}

// TestLDUAssemblerHandlesMultipleLDUs: back-to-back FSW + 840
// dibits twice → sink called twice. Validates that the buffer
// compaction correctly resets state between LDUs.
func TestLDUAssemblerHandlesMultipleLDUs(t *testing.T) {
	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(makeLDUDibits(0, 1))
	a.Process(makeLDUDibits(2, 3))

	if len(got) != 2 {
		t.Fatalf("sink invoked %d times, want 2", len(got))
	}
	if len(got[0]) != LDUTotalBits || len(got[1]) != LDUTotalBits {
		t.Errorf("LDU lengths = %d / %d, want %d each",
			len(got[0]), len(got[1]), LDUTotalBits)
	}
	if a.Buffered() != 0 {
		t.Errorf("Buffered() = %d after two emissions, want 0", a.Buffered())
	}
}

// TestLDUAssemblerNoEmitOnPartialLDU: FSW + only 100 trailing
// dibits (incomplete) → sink not called yet; assembler is still
// collecting. After feeding the remaining 740 dibits, the
// trailing one completes the LDU and fires the sink exactly once.
func TestLDUAssemblerNoEmitOnPartialLDU(t *testing.T) {
	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(FrameSyncWord[:])
	a.Process(dibitsFor(100, 0))
	if len(got) != 0 {
		t.Fatalf("sink invoked %d times after partial input, want 0", len(got))
	}
	a.Process(dibitsFor(LDUDibitCount-24-100, 0))
	if len(got) != 1 {
		t.Fatalf("sink invoked %d times after completing the LDU, want 1", len(got))
	}
}

// TestLDUAssemblerToleratesNoisyFSW: corrupt the FSW with 3
// dibit-position errors and feed with tolerance=4. The
// assembler must still detect + emit. With tolerance=2, the
// same noisy FSW is rejected.
func TestLDUAssemblerToleratesNoisyFSW(t *testing.T) {
	noisy := makeLDUDibits(0)
	// Flip 3 FSW dibits. Using positions inside the 24-dibit
	// FSW span (0..23) so we corrupt the sync pattern, not the
	// payload.
	noisy[1] = (noisy[1] + 1) & 0x3
	noisy[7] = (noisy[7] + 1) & 0x3
	noisy[15] = (noisy[15] + 1) & 0x3

	// tolerance=4: 3 errors fall within the budget — emission fires.
	var got1 [][]byte
	a1 := NewLDUAssembler(func(ldu []byte) { got1 = append(got1, ldu) }, 4)
	a1.Process(noisy)
	if len(got1) != 1 {
		t.Errorf("tolerance=4 with 3 FSW errors: sink invoked %d times, want 1", len(got1))
	}

	// tolerance=2: 3 errors exceed budget — no emission.
	var got2 [][]byte
	a2 := NewLDUAssembler(func(ldu []byte) { got2 = append(got2, ldu) }, 2)
	a2.Process(noisy)
	if len(got2) != 0 {
		t.Errorf("tolerance=2 with 3 FSW errors: sink invoked %d times, want 0", len(got2))
	}
}

// TestLDUAssemblerResetClearsState: after collecting partial
// dibits + a pending FSW, Reset returns the assembler to a
// clean state. Subsequent dibits then build up a fresh LDU
// from scratch.
func TestLDUAssemblerResetClearsState(t *testing.T) {
	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(FrameSyncWord[:])           // pending FSW set
	a.Process(dibitsFor(100, 0))          // collecting
	if a.Buffered() == 0 || a.pending < 0 {
		t.Fatal("setup: assembler not in collecting state")
	}
	a.Reset()
	if a.Buffered() != 0 || a.pending != -1 {
		t.Errorf("after Reset: buffered=%d pending=%d, want 0/-1",
			a.Buffered(), a.pending)
	}
	if len(got) != 0 {
		t.Errorf("Reset shouldn't trigger emission; got %d", len(got))
	}
	// Feed a fresh LDU; emission fires.
	a.Process(makeLDUDibits(0))
	if len(got) != 1 {
		t.Errorf("after Reset + fresh LDU: sink invoked %d times, want 1", len(got))
	}
}

// TestLDUAssemblerEmitsLDUConsumableByExtractVoiceFrames:
// end-to-end stack test. Build a synthetic LDU containing 9
// properly-encoded IMBE channel-coded subframes at the
// documented voice offsets, include status symbols, run the
// bytes through the assembler as a dibit stream, and confirm
// the emitted LDU is accepted cleanly by ExtractVoiceFrames
// (totalErrs == 0, no error). Closes the loop between the
// assembler and the existing voice-extraction primitives.
func TestLDUAssemblerEmitsLDUConsumableByExtractVoiceFrames(t *testing.T) {
	payload := make([]byte, LDUPayloadBits)
	// First 48 bits of the payload must be the canonical FSW bits
	// so the assembler latches.
	copy(payload, FrameSyncBits())
	// Encode 9 synthetic IMBE subframes through the full channel
	// path: info → EncodeChannel → Scramble → place at the voice
	// offsets in the payload.
	for i := 0; i < LDUVoiceSubframeCount; i++ {
		info := make([]byte, 88)
		for k := range info {
			info[k] = byte((i*7 + k*3) % 2)
		}
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
	dibits := framing.BitsToDibits(ldu)

	var got [][]byte
	a := NewLDUAssembler(func(ldu []byte) { got = append(got, ldu) }, 0)
	a.Process(dibits)
	if len(got) != 1 {
		t.Fatalf("sink invoked %d times, want 1", len(got))
	}
	frames, errs, err := ExtractVoiceFrames(got[0])
	if err != nil {
		t.Errorf("ExtractVoiceFrames: %v", err)
	}
	if errs != 0 {
		t.Errorf("ExtractVoiceFrames errs = %d, want 0 on a clean encoded LDU", errs)
	}
	for i, frame := range frames {
		if len(frame) != 11 {
			t.Errorf("frame %d length = %d, want 11", i, len(frame))
		}
	}
}
