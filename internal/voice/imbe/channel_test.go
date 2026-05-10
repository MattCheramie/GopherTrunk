package imbe

import (
	"errors"
	"math/rand"
	"testing"
)

func TestEncodeChannelLength(t *testing.T) {
	info := make([]byte, InfoBits)
	channel, err := EncodeChannel(info)
	if err != nil {
		t.Fatalf("EncodeChannel: %v", err)
	}
	if len(channel) != ChannelBits {
		t.Errorf("len(channel) = %d, want %d", len(channel), ChannelBits)
	}
}

func TestEncodeChannelRejectsWrongLength(t *testing.T) {
	if _, err := EncodeChannel(make([]byte, InfoBits-1)); err == nil {
		t.Error("EncodeChannel accepted short info")
	}
	if _, err := EncodeChannel(make([]byte, InfoBits+1)); err == nil {
		t.Error("EncodeChannel accepted long info")
	}
}

func TestDecodeChannelRejectsWrongLength(t *testing.T) {
	if _, _, err := DecodeChannel(make([]byte, ChannelBits-1)); err == nil {
		t.Error("DecodeChannel accepted short channel")
	}
	if _, _, err := DecodeChannel(make([]byte, ChannelBits+1)); err == nil {
		t.Error("DecodeChannel accepted long channel")
	}
}

func TestChannelRoundTripCleanFrame(t *testing.T) {
	// Random 88-bit info → encode → decode round-trips with zero
	// corrected errors. Run a handful of seeds to broaden coverage.
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 16; trial++ {
		info := make([]byte, InfoBits)
		for i := range info {
			info[i] = byte(rng.Intn(2))
		}
		channel, err := EncodeChannel(info)
		if err != nil {
			t.Fatalf("trial %d: encode: %v", trial, err)
		}
		got, errs, err := DecodeChannel(channel)
		if err != nil {
			t.Fatalf("trial %d: decode: %v", trial, err)
		}
		if errs != 0 {
			t.Errorf("trial %d: errs = %d, want 0", trial, errs)
		}
		for i, b := range got {
			if b != info[i] {
				t.Fatalf("trial %d: bit %d round-trip mismatch", trial, i)
			}
		}
	}
}

func TestChannelCorrectsSingleBitErrorPerFECVector(t *testing.T) {
	// One bit flipped inside any of the seven FEC-protected vectors
	// (u_0..u_6) is well within the per-vector correction radius.
	// The recovered info should still match.
	info := make([]byte, InfoBits)
	for i := range info {
		info[i] = byte(i & 1)
	}
	clean, err := EncodeChannel(info)
	if err != nil {
		t.Fatal(err)
	}

	flipPositions := []int{
		u0Offset + 5,
		u1Offset + 11,
		u2Offset + 0,
		u3Offset + 22,
		u4Offset + 7,
		u5Offset + 14,
		u6Offset + 3,
	}
	for _, pos := range flipPositions {
		corrupt := append([]byte(nil), clean...)
		corrupt[pos] ^= 1
		got, errs, err := DecodeChannel(corrupt)
		if err != nil {
			t.Errorf("flip @ %d: decode err = %v", pos, err)
			continue
		}
		if errs <= 0 {
			t.Errorf("flip @ %d: errs = %d, want > 0", pos, errs)
		}
		for i, b := range got {
			if b != info[i] {
				t.Fatalf("flip @ %d: bit %d differs after correction", pos, i)
				break
			}
		}
	}
}

func TestChannelU7HasNoFEC(t *testing.T) {
	// u_7 is unprotected — flipping any bit there must propagate
	// straight through to the recovered info.
	info := make([]byte, InfoBits)
	clean, _ := EncodeChannel(info)
	const u7InfoStart = 48 + 33 // 81
	for offset := 0; offset < u7Bits; offset++ {
		corrupt := append([]byte(nil), clean...)
		corrupt[u7Offset+offset] ^= 1
		got, _, _ := DecodeChannel(corrupt)
		if got[u7InfoStart+offset] != 1 {
			t.Errorf("u_7 bit %d: corruption did not propagate (got %d)", offset, got[u7InfoStart+offset])
		}
	}
}

func TestChannelFlagsUncorrectableVector(t *testing.T) {
	// Heavy random corruption inside u_0 pushes past Golay's 3-error
	// correction radius. Some patterns happen to land inside another
	// codeword's t-ball and silently mis-decode (a property of any
	// minimum-distance decoder, not unique to this implementation),
	// so we sample a handful of seeds and assert that at least one
	// trips ErrUncorrectable. The test guards the shape of the API
	// — that the error is surfaceable — not the per-pattern
	// behaviour.
	info := make([]byte, InfoBits)
	for i := range info {
		info[i] = byte(i % 3 & 1)
	}
	clean, _ := EncodeChannel(info)
	saw := false
	for seed := int64(1); seed < 32 && !saw; seed++ {
		rng := rand.New(rand.NewSource(seed))
		corrupt := append([]byte(nil), clean...)
		// Flip 12 bits scattered across u_0 (way past t = 3).
		flipped := map[int]bool{}
		for len(flipped) < 12 {
			pos := u0Offset + rng.Intn(u0Bits)
			if !flipped[pos] {
				flipped[pos] = true
				corrupt[pos] ^= 1
			}
		}
		_, _, err := DecodeChannel(corrupt)
		if errors.Is(err, ErrUncorrectable) {
			saw = true
		}
	}
	if !saw {
		t.Error("no random 12-error pattern across 32 seeds tripped ErrUncorrectable")
	}
}

func TestChannelInfoBitsConstantMatchesPackageConstant(t *testing.T) {
	if InfoBitsTotal != InfoBits {
		t.Errorf("InfoBitsTotal alias drifted: %d != %d", InfoBitsTotal, InfoBits)
	}
	// Vector geometry must add up.
	const wantChannel = u0Bits + u1Bits + u2Bits + u3Bits + u4Bits + u5Bits + u6Bits + u7Bits
	const wantInfo = u0InfoBits + u1InfoBits + u2InfoBits + u3InfoBits + u4InfoBits + u5InfoBits + u6InfoBits + u7InfoBits
	if wantChannel != ChannelBits {
		t.Errorf("vector channel-bit sum = %d, want %d", wantChannel, ChannelBits)
	}
	if wantInfo != InfoBits {
		t.Errorf("vector info-bit sum = %d, want %d", wantInfo, InfoBits)
	}
}

// TestPackInfoBitsToFrameRoundTrip: 88 info bits → 11-byte frame
// → unpack via the same MSB-first scheme as Decoder.Decode → must
// recover the original 88 bits. Pins the frame-packing wire
// format against the implicit unpacking the Decoder + DecodeStream
// path performs.
func TestPackInfoBitsToFrameRoundTrip(t *testing.T) {
	info := make([]byte, InfoBits)
	// Mix of bit patterns: every 7th bit set so the byte boundaries
	// see both 0 and 1 across the whole frame.
	for i := range info {
		if i%7 == 0 {
			info[i] = 1
		}
	}
	frame, err := PackInfoBitsToFrame(info)
	if err != nil {
		t.Fatalf("PackInfoBitsToFrame: %v", err)
	}
	if len(frame) != FrameBytes {
		t.Errorf("frame length = %d, want %d", len(frame), FrameBytes)
	}
	// Manually unpack the same way Decoder.Decode does and confirm
	// every bit matches.
	for i := 0; i < InfoBits; i++ {
		got := (frame[i/8] >> (7 - uint(i)%8)) & 1
		if got != info[i] {
			t.Fatalf("bit %d: packed/unpacked = %d, original = %d", i, got, info[i])
		}
	}
}

// TestPackInfoBitsToFrameRejectsWrongLength: any input that isn't
// exactly InfoBits long surfaces ErrChannelLength so callers don't
// silently produce truncated/over-long frames.
func TestPackInfoBitsToFrameRejectsWrongLength(t *testing.T) {
	for _, n := range []int{0, InfoBits - 1, InfoBits + 1, 87, 89, 144} {
		_, err := PackInfoBitsToFrame(make([]byte, n))
		if err == nil {
			t.Errorf("PackInfoBitsToFrame(len=%d) returned no error", n)
		}
	}
}

// TestDecodeChannelToFrameRoundTrip: the full channel-decode
// pipeline must recover the original 88 information bits when
// fed clean (zero-error) channel bits. The on-air shape is:
//
//	info → EncodeChannel → 144 channel bits
//	     → Scramble       → 144 scrambled channel bits  (transmitted)
//	     → Descramble     → 144 channel bits             (after RX)
//	     → DecodeChannel  → 88 info bits
//	     → PackInfoBitsToFrame → 11-byte frame
//
// DecodeChannelToFrame collapses the last three steps, so feeding
// it post-Scramble bits should reproduce the original info inside
// the packed frame.
func TestDecodeChannelToFrameRoundTrip(t *testing.T) {
	original := make([]byte, InfoBits)
	for i := range original {
		// Pseudo-random 0/1 pattern that exercises every vector.
		original[i] = byte((i*13 + 7) % 2)
	}
	encoded, err := EncodeChannel(original)
	if err != nil {
		t.Fatalf("EncodeChannel: %v", err)
	}
	scrambled, err := Scramble(encoded)
	if err != nil {
		t.Fatalf("Scramble: %v", err)
	}

	frame, errs, err := DecodeChannelToFrame(scrambled)
	if err != nil {
		t.Fatalf("DecodeChannelToFrame: %v", err)
	}
	if errs != 0 {
		t.Errorf("errs = %d, want 0 on a clean channel", errs)
	}
	if len(frame) != FrameBytes {
		t.Errorf("frame length = %d, want %d", len(frame), FrameBytes)
	}

	// Unpack the frame and compare to the original.
	for i := 0; i < InfoBits; i++ {
		got := (frame[i/8] >> (7 - uint(i)%8)) & 1
		if got != original[i] {
			t.Fatalf("bit %d: round-trip = %d, original = %d", i, got, original[i])
		}
	}
}

// TestDecodeChannelToFrameWiresIntoDecoder: the recorder-ready
// frame produced by DecodeChannelToFrame must be consumable by a
// fresh imbe.Decoder.Decode() — that's the whole point of the
// helper: bridge a protocol-layer 144-bit burst into the same
// frame format the Decoder ingests. Pin the wire shape end-to-end.
func TestDecodeChannelToFrameWiresIntoDecoder(t *testing.T) {
	// Use the canonical b₀=216 silence frame as the reference.
	// info bits 0..5 = (1,1,0,1,1,0) ⇒ b₀ MSBs = 0xD8 = 216.
	// bits 85, 86 (the two LSBs of b₀) stay 0; remaining info bits
	// are 0. UnpackHeader then flags Silent=true.
	info := make([]byte, InfoBits)
	info[0] = 1
	info[1] = 1
	info[2] = 0
	info[3] = 1
	info[4] = 1
	info[5] = 0

	encoded, err := EncodeChannel(info)
	if err != nil {
		t.Fatalf("EncodeChannel: %v", err)
	}
	scrambled, err := Scramble(encoded)
	if err != nil {
		t.Fatalf("Scramble: %v", err)
	}
	frame, errs, err := DecodeChannelToFrame(scrambled)
	if err != nil {
		t.Fatalf("DecodeChannelToFrame: %v", err)
	}
	if errs != 0 {
		t.Errorf("errs = %d, want 0 on clean channel", errs)
	}

	// Hand the packed frame to a Decoder. A silence frame produces
	// 160 zero PCM samples.
	d := New()
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (silence frame round-tripped through DecodeChannelToFrame)", i, s)
		}
	}
}

// TestDecodeChannelToFrameSurvivesRecoverableError: a single
// flipped bit in a Golay(23,12) vector is correctable; the
// round-trip recovers the original info, and errs > 0 reports the
// correction count. Pins that the helper preserves the
// "frame still usable, just with non-zero errs" contract.
func TestDecodeChannelToFrameSurvivesRecoverableError(t *testing.T) {
	original := make([]byte, InfoBits)
	for i := range original {
		original[i] = byte((i*5 + 1) % 2)
	}
	encoded, err := EncodeChannel(original)
	if err != nil {
		t.Fatalf("EncodeChannel: %v", err)
	}
	scrambled, err := Scramble(encoded)
	if err != nil {
		t.Fatalf("Scramble: %v", err)
	}
	// Flip a single bit inside u_1 (a Golay(23,12) vector,
	// correction radius 3). Bits 0..11 of the channel double as
	// the seed for the §7.4 PRBS scrambler — a flip in that range
	// would cascade through descrambling of u_1..u_6 and isn't
	// straightforwardly recoverable. u_1 spans bits 23..45, so
	// bit 25 sits comfortably past the seed region.
	scrambled[25] ^= 1
	frame, errs, err := DecodeChannelToFrame(scrambled)
	if err != nil {
		t.Fatalf("DecodeChannelToFrame: %v", err)
	}
	if errs == 0 {
		t.Errorf("errs = 0 after a 1-bit flip; expected the FEC to report the correction")
	}
	for i := 0; i < InfoBits; i++ {
		got := (frame[i/8] >> (7 - uint(i)%8)) & 1
		if got != original[i] {
			t.Fatalf("bit %d: round-trip = %d, original = %d (single-bit error should be correctable)", i, got, original[i])
		}
	}
}
