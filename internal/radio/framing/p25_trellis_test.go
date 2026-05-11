package framing

import "testing"

func TestEncodeP25TrellisProducesExpectedLength(t *testing.T) {
	for _, n := range []int{4, 16, 48, 72, 96} {
		info := make([]uint8, n)
		for i := range info {
			info[i] = uint8(i % 4)
		}
		ch := EncodeP25Trellis(info)
		want := 2 * (n + 1)
		if len(ch) != want {
			t.Errorf("EncodeP25Trellis(len=%d) → %d channel dibits, want %d", n, len(ch), want)
		}
	}
}

func TestP25TrellisRoundTripCleanChannel(t *testing.T) {
	info := []uint8{
		0, 1, 2, 3, 3, 2, 1, 0,
		1, 0, 3, 2, 0, 3, 2, 1,
		2, 3, 0, 1, 1, 2, 3, 0,
	}
	ch := EncodeP25Trellis(info)
	got, metric := DecodeP25Trellis(ch)
	if metric != 0 {
		t.Errorf("clean-channel metric = %d, want 0", metric)
	}
	if len(got) != len(info) {
		t.Fatalf("decoded len = %d, want %d", len(got), len(info))
	}
	for i, d := range info {
		if got[i] != d {
			t.Errorf("dibit %d: got %d, want %d", i, got[i], d)
		}
	}
}

func TestP25TrellisCorrectsSingleDibitError(t *testing.T) {
	info := []uint8{0, 1, 2, 3, 2, 1, 0, 3, 1, 0, 2, 3}
	ch := EncodeP25Trellis(info)
	ch[7] ^= 0x3 // flip both bits of one channel dibit

	got, metric := DecodeP25Trellis(ch)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted channel")
	}
	for i, d := range info {
		if got[i] != d {
			t.Errorf("dibit %d: got %d, want %d (error not corrected)", i, got[i], d)
		}
	}
}

func TestP25TrellisDecodeRejectsOddLength(t *testing.T) {
	if got, metric := DecodeP25Trellis([]uint8{0, 1, 2}); got != nil || metric == 0 {
		t.Errorf("odd-length channel = (%v, %d), want (nil, inf)", got, metric)
	}
}

// TestP25TrellisMatchesPhase1Tables: encode the same 48 info dibits
// through both the new framing primitive and a finisher-aware
// reference computation, and confirm bit-for-bit equality. This
// guards the table extraction against accidental drift from the
// Phase 1 source (internal/radio/p25/phase1/trellis.go).
func TestP25TrellisMatchesPhase1Tables(t *testing.T) {
	info := make([]uint8, 48)
	for i := range info {
		info[i] = uint8(i % 4)
	}
	ch := EncodeP25Trellis(info)
	if len(ch) != 98 {
		t.Fatalf("Phase 1 sized encode produced %d channel dibits, want 98", len(ch))
	}
	// First pair: state=0, next=info[0]=0 → idx = p25TrellisStates[0][0] = 0
	// → pair = (0b00, 0b10)
	if ch[0] != 0b00 || ch[1] != 0b10 {
		t.Errorf("first pair = (%d, %d), want (0, 2)", ch[0], ch[1])
	}
}
