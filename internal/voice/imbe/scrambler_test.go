package imbe

import (
	"math/rand"
	"testing"
)

func TestPRBSDeterministicForSeed(t *testing.T) {
	// LCG with seed 0: pr[0] = 0, pr[1] = (173*0+13849) mod 65536 =
	// 13849 (high bit 0), pr[2] = (173*13849+13849) mod 65536 =
	// 50430 (high bit 1), pr[3] = (173*50430+13849) mod 65536.
	bits := PRBS(0)
	wantPrefix := []byte{0, 1} // pr[1] high bit, pr[2] high bit
	for i, w := range wantPrefix {
		if bits[i] != w {
			t.Errorf("seed=0: bits[%d] = %d, want %d", i, bits[i], w)
		}
	}
}

func TestPRBSDifferentSeedsDifferentSequences(t *testing.T) {
	a := PRBS(0)
	b := PRBS(0x1234) // arbitrary non-zero seed
	if a == b {
		t.Error("different seeds produced identical PRBS sequences")
	}
}

func TestPRBSSeedFromU0ReadsFirstTwelveBits(t *testing.T) {
	channel := make([]byte, ChannelBits)
	// Stamp a recognisable 12-bit pattern into u_0's info bits.
	want := uint16(0x9AB) // 0b100110101011
	for i := 0; i < u0InfoBits; i++ {
		channel[u0Offset+i] = byte((want >> uint(u0InfoBits-1-i)) & 1)
	}
	got := PRBSSeedFromU0(channel)
	if got != want<<4 {
		t.Errorf("seed = %#x, want %#x (info bits << 4)", got, want<<4)
	}
}

func TestScrambleRoundTripIsIdentity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 8; trial++ {
		original := make([]byte, ChannelBits)
		for i := range original {
			original[i] = byte(rng.Intn(2))
		}
		work := append([]byte(nil), original...)
		if _, err := Scramble(work); err != nil {
			t.Fatalf("trial %d: Scramble: %v", trial, err)
		}
		if _, err := Descramble(work); err != nil {
			t.Fatalf("trial %d: Descramble: %v", trial, err)
		}
		for i, b := range work {
			if b != original[i] {
				t.Fatalf("trial %d: bit %d = %d after round trip, want %d",
					trial, i, b, original[i])
			}
		}
	}
}

func TestScrambleLeavesU0AndU7Untouched(t *testing.T) {
	// u_0 carries the seed (must be unscrambled) and u_7 is the
	// unprotected least-sensitive bits (also unscrambled per
	// TIA-102.BABA §7.4). Stamp recognisable patterns into both
	// regions and verify Scramble doesn't disturb them.
	channel := make([]byte, ChannelBits)
	for i := 0; i < u0Bits; i++ {
		channel[u0Offset+i] = byte((i * 7) & 1)
	}
	for i := 0; i < u7Bits; i++ {
		channel[u7Offset+i] = byte((i * 5 & 1) ^ 1)
	}
	original := append([]byte(nil), channel...)

	if _, err := Scramble(channel); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < u0Bits; i++ {
		if channel[u0Offset+i] != original[u0Offset+i] {
			t.Errorf("u_0 bit %d mutated by Scramble: %d → %d",
				i, original[u0Offset+i], channel[u0Offset+i])
		}
	}
	for i := 0; i < u7Bits; i++ {
		if channel[u7Offset+i] != original[u7Offset+i] {
			t.Errorf("u_7 bit %d mutated by Scramble: %d → %d",
				i, original[u7Offset+i], channel[u7Offset+i])
		}
	}
}

func TestScrambleAffectsU1ThroughU6(t *testing.T) {
	// Conversely, Scramble must touch *some* bits in u_1..u_6 for
	// non-trivial seeds. Using a seed of all-1s in u_0 gives a
	// non-zero PRBS; verify at least one bit in each scrambled
	// region flips.
	channel := make([]byte, ChannelBits)
	for i := 0; i < u0InfoBits; i++ {
		channel[u0Offset+i] = 1
	}
	original := append([]byte(nil), channel...)
	if _, err := Scramble(channel); err != nil {
		t.Fatal(err)
	}
	for _, region := range [][2]int{
		{u1Offset, u1Bits},
		{u2Offset, u2Bits},
		{u3Offset, u3Bits},
		{u4Offset, u4Bits},
		{u5Offset, u5Bits},
		{u6Offset, u6Bits},
	} {
		off, n := region[0], region[1]
		flipped := false
		for i := 0; i < n; i++ {
			if channel[off+i] != original[off+i] {
				flipped = true
				break
			}
		}
		if !flipped {
			t.Errorf("region @ %d (%d bits) untouched by Scramble", off, n)
		}
	}
}

func TestScrambleRejectsWrongLength(t *testing.T) {
	if _, err := Scramble(make([]byte, ChannelBits-1)); err == nil {
		t.Error("Scramble accepted short channel")
	}
	if _, err := Descramble(make([]byte, ChannelBits+1)); err == nil {
		t.Error("Descramble accepted long channel")
	}
}

func TestScrambleThenDecodeChannelRecoversInfo(t *testing.T) {
	// End-to-end self-consistency: encode → scramble (transmit) →
	// descramble → decode recovers the original 88 info bits.
	rng := rand.New(rand.NewSource(7))
	info := make([]byte, InfoBits)
	for i := range info {
		info[i] = byte(rng.Intn(2))
	}
	channel, err := EncodeChannel(info)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Scramble(channel); err != nil {
		t.Fatal(err)
	}
	if _, err := Descramble(channel); err != nil {
		t.Fatal(err)
	}
	got, errs, err := DecodeChannel(channel)
	if err != nil {
		t.Fatalf("DecodeChannel: %v", err)
	}
	if errs != 0 {
		t.Errorf("errs = %d, want 0", errs)
	}
	for i, b := range got {
		if b != info[i] {
			t.Fatalf("info bit %d differs after scramble/descramble round trip", i)
		}
	}
}
