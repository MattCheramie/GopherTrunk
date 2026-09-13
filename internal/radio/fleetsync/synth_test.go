package fleetsync

import "testing"

// TestANIWordsInvertsNewMessage: the synthesiser's field packing must be
// the exact inverse of the decoder's extraction, across the whole ANI
// range — otherwise a harness could assert the wrong ground truth.
func TestANIWordsInvertsNewMessage(t *testing.T) {
	cases := []struct{ fleet, unit int }{
		{MinFleet, MinUnit}, {MaxFleet, MaxUnit}, {107, 1772}, {150, 3001}, {200, 999 + 0x0F0},
	}
	for _, c := range cases {
		w1, hi16, err := ANIWords(c.fleet, c.unit)
		if err != nil {
			t.Fatalf("ANIWords(%d,%d): %v", c.fleet, c.unit, err)
		}
		m := newMessage(w1, uint32(hi16)<<16, false, true)
		if m.Fleet != c.fleet || m.Unit != c.unit {
			t.Errorf("ANIWords(%d,%d) decodes as fleet %d unit %d", c.fleet, c.unit, m.Fleet, m.Unit)
		}
	}
	if _, _, err := ANIWords(MinFleet-1, MinUnit); err == nil {
		t.Error("fleet below range accepted")
	}
	if _, _, err := ANIWords(MinFleet, MaxUnit+1); err == nil {
		t.Error("unit above range accepted")
	}
}

// TestSynthBurstRoundTripsThroughFramer: both FS-I and FS-II synthesised
// bursts must frame and decode to the ANI they were built from, CRC-valid,
// with the right path flag — the reporter's own ground truth (#1184) as
// the vector.
func TestSynthBurstRoundTripsThroughFramer(t *testing.T) {
	for _, fs2 := range []bool{false, true} {
		bits, err := SynthBurst(107, 1772, fs2)
		if err != nil {
			t.Fatalf("SynthBurst(fs2=%v): %v", fs2, err)
		}
		if want := SynthPreambleBits + SyncBits + FrameBits; len(bits) != want {
			t.Fatalf("SynthBurst(fs2=%v) len = %d, want %d", fs2, len(bits), want)
		}
		var got []Message
		f := NewFramer(func(m Message) { got = append(got, m) })
		for _, b := range bits {
			f.Push(b)
		}
		if len(got) != 1 {
			t.Fatalf("fs2=%v: framer emitted %d messages, want 1", fs2, len(got))
		}
		m := got[0]
		if !m.CRCOK || m.Fleet != 107 || m.Unit != 1772 || m.IsFS2 != fs2 {
			t.Errorf("fs2=%v: decoded %+v, want CRCOK fleet 107 unit 1772 IsFS2=%v", fs2, m, fs2)
		}
	}
}

// TestFS2PayloadSurvivesOneErrorPerHalfWord: the FS-II layout must place
// each nibble inside its own ECC code word so that one sliced-bit error
// in EVERY half-word is still corrected — the property the on-air format
// exists for, and what makes the synthetic FS-II vector meaningful for
// the DSP tests.
func TestFS2PayloadSurvivesOneErrorPerHalfWord(t *testing.T) {
	w1, hi16, _ := ANIWords(107, 1772)
	w2, ok := SolveBlockCheck(w1, hi16)
	if !ok {
		t.Fatal("no block check")
	}
	raw := FS2Payload(w1, w2)
	for blk := 0; blk < 4; blk++ {
		for half := 0; half < 4; half++ {
			// Flip a bit inside code-bit range (bit 14..0 of the half-word);
			// half-word h of block b occupies raw[frameOffset+blk*64+half*16 ...].
			raw[frameOffset+blk*64+half*16+3+half] ^= 1
		}
	}
	m, ok := DecodeFrame(raw)
	if !ok || !m.IsFS2 || m.Fleet != 107 || m.Unit != 1772 {
		t.Fatalf("FS2 with one error per half-word: ok=%v msg=%+v, want corrected fleet 107 unit 1772", ok, m)
	}
}
