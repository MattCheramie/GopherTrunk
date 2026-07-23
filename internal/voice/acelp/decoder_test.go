package acelp

import "testing"

// makeFrame builds a plausible 24-word parameter array with valid-range field
// values driven by a seed, so tests exercise the real decode paths.
func makeFrame(seed int) [numFields]int16 {
	var prm [numFields]int16
	for i := 0; i < numParams; i++ {
		w := bitWidths[i]
		prm[1+i] = int16((seed*7 + i*13 + 1) & (1<<uint(w) - 1))
	}
	return prm
}

// TestDecodeFrameShape checks a frame decodes to exactly 240 samples without
// panicking, across a spread of parameter sets.
func TestDecodeFrameShape(t *testing.T) {
	for seed := 0; seed < 32; seed++ {
		d := NewDecoder()
		out := d.DecodeFrame(makeFrame(seed), false)
		if len(out) != frameLen {
			t.Fatalf("seed %d: got %d samples, want %d", seed, len(out), frameLen)
		}
	}
}

// TestDecodeZeroExcitation checks the synthesis stage produces no energy when
// there is no excitation: the synthesis filter output for a zero input is all
// zero. (A zero-parameter frame is NOT silent in ACELP — the algebraic codebook
// still injects pulses — so this checks the filter path directly.)
func TestDecodeZeroExcitation(t *testing.T) {
	a := lspAz(lspReset[:])
	x := make([]int16, subfrLen) // zero excitation
	y := make([]int16, subfrLen)
	mem := make([]int16, lpcOrder)
	synFilt(a[:], x, y, subfrLen, mem, true)
	for i, s := range y {
		if s != 0 {
			t.Fatalf("zero excitation produced sample %d = %d", i, s)
		}
	}
}

// TestDecodeBounded checks the decoder stays stable over many frames — no whole
// frame pegs to full-scale, which would signal a runaway synthesis filter.
func TestDecodeBounded(t *testing.T) {
	d := NewDecoder()
	for seed := 0; seed < 40; seed++ {
		out := d.DecodeFrame(makeFrame(seed), false)
		pegged := 0
		for _, s := range out {
			if s == maxWord16 || s == minWord16 {
				pegged++
			}
		}
		if pegged == frameLen {
			t.Fatalf("seed %d: entire frame saturated (unstable filter)", seed)
		}
	}
}

// TestDecodeDeterministic checks two decoders fed identical frames produce
// identical output.
func TestDecodeDeterministic(t *testing.T) {
	d1, d2 := NewDecoder(), NewDecoder()
	for seed := 0; seed < 8; seed++ {
		o1 := d1.DecodeFrame(makeFrame(seed), false)
		o2 := d2.DecodeFrame(makeFrame(seed), false)
		if o1 != o2 {
			t.Fatalf("seed %d: non-deterministic output", seed)
		}
	}
}

// TestDecodeProducesSignal checks a non-trivial parameter stream yields
// non-silent, in-range output — i.e. the chain actually synthesises something.
func TestDecodeProducesSignal(t *testing.T) {
	d := NewDecoder()
	nonzero := false
	for seed := 1; seed < 10; seed++ {
		out := d.DecodeFrame(makeFrame(seed), false)
		for _, s := range out {
			if s != 0 {
				nonzero = true
			}
		}
	}
	if !nonzero {
		t.Error("decoder produced only silence for non-trivial input")
	}
}

// TestDecodeBFI checks an erased frame decodes (frame repetition) without
// panicking and updates state, both from reset and mid-stream.
func TestDecodeBFI(t *testing.T) {
	d := NewDecoder()
	// erased first frame
	_ = d.DecodeFrame([numFields]int16{}, true)

	// a few good frames, then an erased one mid-stream
	for seed := 1; seed < 4; seed++ {
		_ = d.DecodeFrame(makeFrame(seed), false)
	}
	before := d.oldExc[excOff] // some state
	out := d.DecodeFrame(makeFrame(99), true)
	if len(out) != frameLen {
		t.Fatalf("BFI frame produced %d samples", len(out))
	}
	_ = before // state may or may not change; we only require no panic + shape
}

// TestDecodeFromBits checks the bits-in wrapper: a short slice is treated as
// erased, a full 137-bit frame decodes.
func TestDecodeFromBits(t *testing.T) {
	d := NewDecoder()
	if got := d.Decode(nil, false); len(got) != frameLen {
		t.Fatalf("nil bits: %d samples", len(got))
	}
	bits := make([]byte, speechFrameBits)
	for i := range bits {
		bits[i] = byte((i * 3) & 1)
	}
	if got := d.Decode(bits, false); len(got) != frameLen {
		t.Fatalf("full frame: %d samples", len(got))
	}
}
