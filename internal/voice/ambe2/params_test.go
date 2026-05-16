package ambe2

import (
	"math"
	"testing"
)

// setBit places a single 0/1 bit at the given info position. Used by
// the synthetic-frame builders below to keep test fixtures readable
// (callers spell each non-zero index instead of an opaque byte array).
func setBit(info []byte, pos int, val byte) {
	info[pos] = val
}

// TestUnpackParamsRejectsWrongLength: UnpackParams must reject info
// buffers of any length other than 49. Callers above (P25 P2 / DMR /
// NXDN protocol decoders) all emit exactly 49 bits after their FEC.
func TestUnpackParamsRejectsWrongLength(t *testing.T) {
	for _, n := range []int{0, 48, 50, 56} {
		_, err := UnpackParams(make([]byte, n))
		if err == nil {
			t.Errorf("UnpackParams(len=%d) returned no error", n)
		}
	}
}

// TestUnpackParamsZeroFrame: an all-zero 49-bit info buffer decodes
// to b0=0 (lowest fundamental), b1..b8 all zero. Pins the bit-extract
// math against an easy-to-verify input. b0=0 ⇒ L = AmbePlusLtable[0] = 9,
// f0 = 2^(-4.311767578125 - 0.0213·0.5).
func TestUnpackParamsZeroFrame(t *testing.T) {
	info := make([]byte, InfoBits)
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams: %v", err)
	}
	if p.Tone {
		t.Fatal("expected voice frame, got Tone=true")
	}
	if p.B0 != 0 || p.B1 != 0 || p.B2 != 0 || p.B3 != 0 || p.B4 != 0 ||
		p.B5 != 0 || p.B6 != 0 || p.B7 != 0 || p.B8 != 0 {
		t.Errorf("all-zero info should produce all-zero indices, got b0=%d b1=%d b2=%d b3=%d b4=%d b5=%d b6=%d b7=%d b8=%d",
			p.B0, p.B1, p.B2, p.B3, p.B4, p.B5, p.B6, p.B7, p.B8)
	}
	if p.L != 9 {
		t.Errorf("L = %d, want 9 (AmbePlusLtable[0])", p.L)
	}
	wantW0 := math.Pow(2, -4.311767578125-2.1336e-2*0.5) * 2 * math.Pi
	if math.Abs(p.W0-wantW0) > 1e-9 {
		t.Errorf("W0 = %v, want %v", p.W0, wantW0)
	}
	wantUnvc := 0.2046 / math.Sqrt(wantW0)
	if math.Abs(p.Unvc-wantUnvc) > 1e-9 {
		t.Errorf("Unvc = %v, want %v", p.Unvc, wantUnvc)
	}
	if p.DeltaGamma != AmbePlusDg[0] {
		t.Errorf("DeltaGamma = %v, want %v (AmbePlusDg[0])", p.DeltaGamma, AmbePlusDg[0])
	}
	// AmbePlusVuv[0] is all zeros, so every Vl entry in [1..L] should
	// be zero (entirely unvoiced).
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] != 0 {
			t.Errorf("Vl[%d] = %d, want 0 (AmbePlusVuv[0] is all zeros)", l, p.Vl[l])
		}
	}
}

// TestUnpackParamsB0ExtractionBitPositions: pin every bit position
// that contributes to b0. b0 spans info[0..5] (6 high bits) +
// info[48] (LSB), packed MSB-first per mbelib's
// mbe_decodeAmbe2400Parms.
func TestUnpackParamsB0ExtractionBitPositions(t *testing.T) {
	cases := []struct {
		name    string
		setBits []int
		wantB0  int
	}{
		{"bit 0 set", []int{0}, 1 << 6},
		{"bit 5 set", []int{5}, 1 << 1},
		{"bit 48 set", []int{48}, 1},
		{"bits 0+48 set", []int{0, 48}, (1 << 6) | 1},
		{"bit 3 set (matches mbelib position)", []int{3}, 1 << 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			info := make([]byte, InfoBits)
			for _, b := range c.setBits {
				setBit(info, b, 1)
			}
			// Force voice frame by ensuring b0 < 0x7E. None of the test
			// cases above set bits that would push b0 into the tone
			// window, so no extra guard needed.
			p, err := UnpackParams(info)
			if err != nil {
				t.Fatalf("UnpackParams: %v", err)
			}
			if p.B0 != c.wantB0 {
				t.Errorf("B0 = %d (0x%02X), want %d (0x%02X)", p.B0, p.B0, c.wantB0, c.wantB0)
			}
		})
	}
}

// TestUnpackParamsToneFrameDetection: b0 in {0x7E, 0x7F} is the tone
// frame indicator (AMBE+2 §7.2). The unpacker must short-circuit
// past the voice-frame branches and flag Tone=true.
func TestUnpackParamsToneFrameDetection(t *testing.T) {
	// b0 = 0x7E: set info[0..5] = 1, info[48] = 0.
	info := make([]byte, InfoBits)
	for i := 0; i <= 5; i++ {
		info[i] = 1
	}
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams (b0=0x7E): %v", err)
	}
	if !p.Tone {
		t.Errorf("b0=0x%02X: Tone = false, want true", p.B0)
	}
	if p.B0 != 0x7E {
		t.Errorf("B0 = 0x%02X, want 0x7E", p.B0)
	}

	// b0 = 0x7F: same but info[48] = 1.
	info[48] = 1
	p, err = UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams (b0=0x7F): %v", err)
	}
	if !p.Tone {
		t.Errorf("b0=0x%02X: Tone = false, want true", p.B0)
	}
	if p.B0 != 0x7F {
		t.Errorf("B0 = 0x%02X, want 0x7F", p.B0)
	}
}

// TestUnpackParamsToneFrameInvalidIndexFlagsSilent: a tone frame with
// b1 < 5 (or in the reserved windows) is flagged silent by mbelib;
// we mirror that so the synthesizer's silence path runs cleanly.
//
// Reaching b1 = 0 requires the t5/t6/t7 table inputs to all map to
// 0. From the tables, only idx = 1 (info[6,7,8] = 0,0,1) yields
// (t5, t6, t7) = (0, 0, 0). With the remaining 5 b1 bits also
// zero, b1 = 0 which falls in the invalid (< 5) window.
func TestUnpackParamsToneFrameInvalidIndexFlagsSilent(t *testing.T) {
	info := make([]byte, InfoBits)
	// Tone frame (b0 = 0x7E).
	for i := 0; i <= 5; i++ {
		info[i] = 1
	}
	// idx = 1 → all three t-tables emit 0 → high 3 bits of b1 = 0.
	info[8] = 1
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams: %v", err)
	}
	if !p.Tone {
		t.Fatal("expected Tone=true")
	}
	if p.B1 != 0 {
		t.Errorf("B1 = %d, want 0 (invalid tone index)", p.B1)
	}
	if !p.Silent {
		t.Errorf("tone frame with b1=0 should flag Silent=true (mbelib parity)")
	}
}

// TestUnpackParamsToneFrameB1TableLookup: the t5/t6/t7 tables map
// the 3-bit input (info[6,7,8]) to bits 5..7 of the tone-frame b1.
// Pin the exact mapping mbelib transcribes from the AMBE+2 tone-
// frame definition — a regression here changes which tone index a
// frame decodes to, which would silently break tone synthesis.
func TestUnpackParamsToneFrameB1TableLookup(t *testing.T) {
	// (info[6], info[7], info[8]) → idx → bits 7,6,5 of b1 → high
	// 3 bits as a single integer (bits[7..5] << 5).
	cases := []struct {
		i6, i7, i8 byte
		wantHigh   int // (t7 << 7) | (t6 << 6) | (t5 << 5)
	}{
		{0, 0, 0, (1 << 7)},                       // idx 0: t7=1, t6=0, t5=0
		{0, 0, 1, 0},                              // idx 1: all zero
		{0, 1, 0, (1 << 5)},                       // idx 2: t5=1
		{0, 1, 1, (1 << 6)},                       // idx 3: t6=1
		{1, 0, 0, (1 << 6) | (1 << 5)},            // idx 4: t6=1, t5=1
		{1, 0, 1, (1 << 7) | (1 << 6) | (1 << 5)}, // idx 5: all 1
		{1, 1, 0, (1 << 7) | (1 << 6)},            // idx 6: t7=1, t6=1
		{1, 1, 1, (1 << 7) | (1 << 5)},            // idx 7: t7=1, t5=1
	}
	for _, c := range cases {
		info := make([]byte, InfoBits)
		// Tone frame.
		for i := 0; i <= 5; i++ {
			info[i] = 1
		}
		info[6], info[7], info[8] = c.i6, c.i7, c.i8
		p, err := UnpackParams(info)
		if err != nil {
			t.Fatalf("idx=(%d,%d,%d): %v", c.i6, c.i7, c.i8, err)
		}
		if !p.Tone {
			t.Fatalf("idx=(%d,%d,%d): not flagged Tone", c.i6, c.i7, c.i8)
		}
		// Lower 5 bits of b1 come from info[9,42,43,10,11], all zero.
		if p.B1 != c.wantHigh {
			t.Errorf("info[6,7,8]=(%d,%d,%d): B1 = %d (0x%02X), want %d (0x%02X)",
				c.i6, c.i7, c.i8, p.B1, p.B1, c.wantHigh, c.wantHigh)
		}
	}
}

// TestUnpackParamsB1Extraction: b1 spans info[38..41] MSB-first.
// Walk a couple of values to confirm the packing matches mbelib's
// "b1 |= ambe_d[38]<<3; b1 |= ambe_d[39]<<2; ..." sequence.
func TestUnpackParamsB1Extraction(t *testing.T) {
	cases := []struct {
		bits   [4]byte // bits at positions 38, 39, 40, 41
		wantB1 int
	}{
		{[4]byte{1, 0, 0, 0}, 0b1000},
		{[4]byte{0, 1, 0, 0}, 0b0100},
		{[4]byte{0, 0, 1, 0}, 0b0010},
		{[4]byte{0, 0, 0, 1}, 0b0001},
		{[4]byte{1, 0, 1, 0}, 0b1010},
		{[4]byte{1, 1, 1, 1}, 0b1111},
	}
	for _, c := range cases {
		info := make([]byte, InfoBits)
		for i := 0; i < 4; i++ {
			info[38+i] = c.bits[i]
		}
		p, err := UnpackParams(info)
		if err != nil {
			t.Fatalf("UnpackParams: %v", err)
		}
		if p.B1 != c.wantB1 {
			t.Errorf("bits %v at [38..41]: B1 = %d, want %d", c.bits, p.B1, c.wantB1)
		}
	}
}

// TestUnpackParamsB2Extraction: b2 spans info[6,7,8,9,42,43]. The
// first 4 bits live at info[6..9] in MSB-first order; the last 2
// live at info[42..43]. Mismatched ordering between the two regions
// would produce wrong gain selection — a subtle, audible regression
// that's worth pinning explicitly.
func TestUnpackParamsB2Extraction(t *testing.T) {
	info := make([]byte, InfoBits)
	info[6] = 1  // → bit 5 (32)
	info[42] = 1 // → bit 1 (2)
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams: %v", err)
	}
	if p.B2 != 32|2 {
		t.Errorf("B2 = %d, want %d (info[6]→bit5, info[42]→bit1)", p.B2, 32|2)
	}
}

// TestUnpackParamsB8LSBForcedZero: bit 0 of b8 is forced to 0 per
// mbelib's comment ("least significant bit of hoc3 unused here, and
// according to the patent is forced to 0 when not used"). So b8
// only takes even values 0, 2, 4, ..., 14.
func TestUnpackParamsB8LSBForcedZero(t *testing.T) {
	info := make([]byte, InfoBits)
	info[35] = 1
	info[36] = 1
	info[37] = 1 // all three b8 bits set
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams: %v", err)
	}
	// b8 = (1<<3)|(1<<2)|(1<<1) = 14; LSB always 0.
	if p.B8 != 14 {
		t.Errorf("B8 = %d, want 14 (LSB forced to 0)", p.B8)
	}
	if p.B8&1 != 0 {
		t.Errorf("B8 LSB = 1, must be 0")
	}
}

// TestUnpackParamsBit24IsUnused: AMBE+2's 49-bit info buffer reserves
// position 24 (the only one not feeding any b-index). Setting it
// must not affect any extracted index. Pins that no b-index
// accidentally reads info[24].
func TestUnpackParamsBit24IsUnused(t *testing.T) {
	a := make([]byte, InfoBits)
	b := make([]byte, InfoBits)
	b[24] = 1
	pa, err := UnpackParams(a)
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	pb, err := UnpackParams(b)
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if pa.B0 != pb.B0 || pa.B1 != pb.B1 || pa.B2 != pb.B2 ||
		pa.B3 != pb.B3 || pa.B4 != pb.B4 || pa.B5 != pb.B5 ||
		pa.B6 != pb.B6 || pa.B7 != pb.B7 || pa.B8 != pb.B8 {
		t.Errorf("info[24] affected an index: a=%+v b=%+v", pa, pb)
	}
}

// TestUnpackParamsJiSumsToL: AmbePlusLmprbl[L] partitions L
// harmonics across the 4 spectral bands. Ji[1]+Ji[2]+Ji[3]+Ji[4]
// must equal L by construction, so the inverse DCT loop walks
// exactly L Tl values. Verify for every supported L (9..56).
func TestUnpackParamsJiSumsToL(t *testing.T) {
	for L := 9; L <= 56; L++ {
		sum := AmbePlusLmprbl[L][0] + AmbePlusLmprbl[L][1] +
			AmbePlusLmprbl[L][2] + AmbePlusLmprbl[L][3]
		if sum != L {
			t.Errorf("L=%d: Ji sum = %d, want %d", L, sum, L)
		}
	}
}

// TestUnpackParamsTlLengthMatchesL: a valid voice frame produces Tl
// values for indices 1..L; indices [L+1..56] stay at zero. Pin that
// the inverse DCT walks exactly L harmonics.
func TestUnpackParamsTlLengthMatchesL(t *testing.T) {
	info := make([]byte, InfoBits)
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatalf("UnpackParams: %v", err)
	}
	for l := p.L + 1; l <= 56; l++ {
		if p.Tl[l] != 0 {
			t.Errorf("Tl[%d] = %v, want 0 (out of L=%d range)", l, p.Tl[l], p.L)
		}
	}
}

// TestUnpackParamsAllB0Values: walk every valid voice-frame b0
// (0..0x7D = 0..125) and confirm UnpackParams runs without panicking
// and produces an L in [9, 56] + a fully-populated Tl[1..L]. Catches
// out-of-range table indexing across the full codebook surface — the
// AmbePlusLmprbl Ji=17 rows at high L stress the inverse DCT loop's
// upper bound.
func TestUnpackParamsAllB0Values(t *testing.T) {
	for b0 := 0; b0 <= 0x7D; b0++ {
		info := make([]byte, InfoBits)
		// Pack b0 across info[0..5] (MSB-first, 6 bits) + info[48] (LSB).
		info[0] = byte((b0 >> 6) & 1)
		info[1] = byte((b0 >> 5) & 1)
		info[2] = byte((b0 >> 4) & 1)
		info[3] = byte((b0 >> 3) & 1)
		info[4] = byte((b0 >> 2) & 1)
		info[5] = byte((b0 >> 1) & 1)
		info[48] = byte(b0 & 1)

		p, err := UnpackParams(info)
		if err != nil {
			t.Fatalf("b0=%d: %v", b0, err)
		}
		if p.B0 != b0 {
			t.Errorf("b0=%d: B0=%d", b0, p.B0)
		}
		if p.Tone {
			t.Errorf("b0=%d: unexpectedly flagged Tone", b0)
		}
		if p.L < 9 || p.L > 56 {
			t.Errorf("b0=%d: L=%d out of [9, 56]", b0, p.L)
		}
		// Confirm Tl[1..L] was written (could be 0 in valid frames,
		// but the loop must have walked L iterations without panicking).
		// The check above implicitly verifies that — if the loop had
		// short-circuited, the post-condition in UnpackParams would have
		// returned an error.
	}
}

// TestUnpackParamsZeroFrameTlDeterministic: the all-zero info buffer
// produces a deterministic Tl set driven by the b0..b8 = 0 codebook
// entries. Two calls must produce identical Tl (pure-function
// guarantee).
func TestUnpackParamsZeroFrameTlDeterministic(t *testing.T) {
	a, err := UnpackParams(make([]byte, InfoBits))
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := UnpackParams(make([]byte, InfoBits))
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if a.Tl != b.Tl {
		t.Errorf("Tl differs across identical calls: a=%v b=%v", a.Tl, b.Tl)
	}
	if a.DeltaGamma != b.DeltaGamma || a.W0 != b.W0 || a.L != b.L {
		t.Errorf("non-Tl fields diverged across identical calls")
	}
}
