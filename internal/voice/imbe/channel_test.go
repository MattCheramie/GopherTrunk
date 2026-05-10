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
