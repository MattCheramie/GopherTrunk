package fleetsync

import "testing"

// pushBitsMSB feeds v's low n bits, most-significant first.
func pushBitsMSB(f *Framer, v uint64, n int, invert bool) {
	for i := n - 1; i >= 0; i-- {
		b := byte((v >> uint(i)) & 1)
		if invert {
			b ^= 1
		}
		f.Push(b)
	}
}

// pushSlice feeds a one-byte-per-bit slice, optionally inverted.
func pushSlice(f *Framer, bitsSlice []byte, invert bool) {
	for _, b := range bitsSlice {
		if invert {
			b ^= 1
		}
		f.Push(b)
	}
}

// feedBurst drives preamble + sync + payload through a framer, optionally
// with inverted tone sense.
func feedBurst(f *Framer, payload []byte, invert bool) {
	// 32 alternating preamble bits (1,0,1,0,…) — more than the 24 the
	// framer checks, so the register is clean by the time sync arrives.
	pushBitsMSB(f, 0xAAAAAAAA, 32, invert)
	pushBitsMSB(f, uint64(SyncWord), SyncBits, invert)
	pushSlice(f, payload, invert)
}

func TestFramerDecodesFS1Burst(t *testing.T) {
	word1 := uint32(0x0000_337D) // fleet 150, unit high 0x7D
	payload, _ := buildFS1Frame(t, word1, 0x2F00)

	var got []Message
	f := NewFramer(func(m Message) { got = append(got, m) })
	feedBurst(f, payload, false)

	if len(got) != 1 {
		t.Fatalf("emitted %d messages, want 1", len(got))
	}
	m := got[0]
	if !m.CRCOK || m.Fleet != 150 || m.Unit != 3001 {
		t.Errorf("decoded %+v, want CRCOK fleet 150 unit 3001", m)
	}
	if s := f.Stats(); s.BurstsIn != 1 || s.BurstsEmitted != 1 || s.BurstsBadCRC != 0 {
		t.Errorf("stats = %+v, want 1 in / 1 emitted / 0 bad-CRC", s)
	}
}

// TestFramerDecodesInvertedPolarity: an FM discriminator can present the
// burst with the opposite tone sense, complementing every bit including
// the sync word. The framer must lock on the complemented sync and
// de-invert the payload to recover the same ANI.
func TestFramerDecodesInvertedPolarity(t *testing.T) {
	word1 := uint32(0x0000_337D)
	payload, _ := buildFS1Frame(t, word1, 0x2F00)

	var got []Message
	f := NewFramer(func(m Message) { got = append(got, m) })
	feedBurst(f, payload, true) // inverted

	if len(got) != 1 {
		t.Fatalf("emitted %d messages, want 1", len(got))
	}
	if m := got[0]; !m.CRCOK || m.Fleet != 150 || m.Unit != 3001 {
		t.Errorf("inverted-polarity decode = %+v, want CRCOK fleet 150 unit 3001", m)
	}
}

// TestFramerIgnoresIdleAndPreambleOnly: neither a long idle (all zeros)
// nor a pure alternating preamble with no sync word may trigger a lock —
// the 16-bit sync gates capture.
func TestFramerIgnoresIdleAndPreambleOnly(t *testing.T) {
	var got []Message
	f := NewFramer(func(m Message) { got = append(got, m) })

	for i := 0; i < 2000; i++ { // idle
		f.Push(0)
	}
	pushBitsMSB(f, 0xAAAAAAAAAAAAAAAA, 64, false) // preamble, no sync
	pushBitsMSB(f, 0x5555555555555555, 64, false) // opposite phase, no sync

	if len(got) != 0 {
		t.Fatalf("emitted %d messages on idle/preamble-only input, want 0", len(got))
	}
	if s := f.Stats(); s.BurstsIn != 0 {
		t.Errorf("BurstsIn = %d, want 0 (no sync word was present)", s.BurstsIn)
	}
}

// TestFramerReframesAfterBurst: two back-to-back bursts each decode, so a
// decoded frame returns the framer cleanly to the sync hunt.
func TestFramerReframesAfterBurst(t *testing.T) {
	p1, _ := buildFS1Frame(t, 0x0000_337D, 0x2F00) // fleet 150 unit 3001
	p2, _ := buildFS1Frame(t, 0x0000_647D, 0x2F00) // byte2=0x64=100 → fleet 199

	var got []Message
	f := NewFramer(func(m Message) { got = append(got, m) })
	feedBurst(f, p1, false)
	feedBurst(f, p2, false)

	if len(got) != 2 {
		t.Fatalf("emitted %d messages, want 2", len(got))
	}
	if got[0].Fleet != 150 || got[1].Fleet != 199 {
		t.Errorf("fleets = %d, %d, want 150, 199", got[0].Fleet, got[1].Fleet)
	}
}
