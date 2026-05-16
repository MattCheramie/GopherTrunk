package framing

import (
	"testing"
)

// TestPN44SeedFromIdentity verifies the seed-calculation formula
// per TIA-102.BBAC-1 §7.2.5 equation (5).
func TestPN44SeedFromIdentity(t *testing.T) {
	cases := []struct {
		name      string
		wacn      uint32
		sysID     uint16
		colorCode uint16
		wantSeed  uint64
	}{
		{
			name:      "all-zero maps to all-ones per spec",
			wacn:      0,
			sysID:     0,
			colorCode: 0,
			wantSeed:  (1 << 44) - 1,
		},
		{
			name:      "color code only",
			wacn:      0,
			sysID:     0,
			colorCode: 0xABC,
			wantSeed:  0xABC,
		},
		{
			name:      "system ID only",
			wacn:      0,
			sysID:     0x123,
			colorCode: 0,
			wantSeed:  uint64(0x123) << 12,
		},
		{
			name:      "WACN only",
			wacn:      0x00012,
			sysID:     0,
			colorCode: 0,
			wantSeed:  uint64(0x00012) << 24,
		},
		{
			name:      "all three combined",
			wacn:      0xABCDE,
			sysID:     0x123,
			colorCode: 0xF7,
			wantSeed:  (uint64(0xABCDE) << 24) | (uint64(0x123) << 12) | 0xF7,
		},
		{
			name:      "oversize WACN masked to 20 bits",
			wacn:      0xFFFFFFFF,
			sysID:     0,
			colorCode: 0,
			wantSeed:  uint64(0x000FFFFF) << 24,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PN44SeedFromIdentity(c.wacn, c.sysID, c.colorCode)
			if got != c.wantSeed {
				t.Errorf("PN44SeedFromIdentity(%#x, %#x, %#x) = %#x, want %#x",
					c.wacn, c.sysID, c.colorCode, got, c.wantSeed)
			}
			if got&^pn44Mask != 0 {
				t.Errorf("seed %#x has bits outside the 44-bit field", got)
			}
		})
	}
}

// TestPN44ScramblerRoundTrip verifies the fundamental scrambler
// property: applying the scrambler twice (with the same seed)
// recovers the original bit stream.
func TestPN44ScramblerRoundTrip(t *testing.T) {
	const seed = uint64(0x1234567890A)
	const n = 4320 // one full superframe per spec
	original := make([]byte, n)
	for i := range original {
		original[i] = byte((i*7 + 3) & 1)
	}
	scrambled := append([]byte(nil), original...)
	NewPN44Scrambler(seed).Apply(scrambled)
	if equalByteSlice(scrambled, original) {
		t.Fatalf("scrambling did not change the bit stream")
	}
	NewPN44Scrambler(seed).Apply(scrambled)
	if !equalByteSlice(scrambled, original) {
		t.Fatalf("descrambling did not recover the original bit stream")
	}
}

// TestPN44ScramblerZeroSeedMapsToAllOnes confirms the seed = 0 edge
// case per spec equation (5).
func TestPN44ScramblerZeroSeedMapsToAllOnes(t *testing.T) {
	zero := NewPN44Scrambler(0)
	allOnes := NewPN44Scrambler(pn44Mask)
	for i := 0; i < 100; i++ {
		if zero.Next() != allOnes.Next() {
			t.Fatalf("seed=0 sequence diverged from seed=2^44-1 at bit %d", i)
		}
	}
}

// TestPN44ScramblerSequenceIsDeterministic verifies the same seed
// yields identical sequences across instances.
func TestPN44ScramblerSequenceIsDeterministic(t *testing.T) {
	seeds := []uint64{
		1,
		0xCAFE,
		(uint64(0xABCDE) << 24) | (uint64(0x123) << 12) | 0xF7,
		pn44Mask,
	}
	for _, seed := range seeds {
		a := NewPN44Scrambler(seed)
		b := NewPN44Scrambler(seed)
		for i := 0; i < 1000; i++ {
			if a.Next() != b.Next() {
				t.Fatalf("seed=%#x: scrambler diverged at bit %d", seed, i)
			}
		}
	}
}

// TestPN44ScramblerOutputDiffersAcrossSeeds confirms the sequence is
// genuinely seed-dependent — two different seeds produce two
// different sequences in the first few hundred bits.
func TestPN44ScramblerOutputDiffersAcrossSeeds(t *testing.T) {
	a := NewPN44Scrambler(0x12345678901)
	b := NewPN44Scrambler(0xFEDCBA98765)
	differences := 0
	for i := 0; i < 1000; i++ {
		if a.Next() != b.Next() {
			differences++
		}
	}
	// For a well-conditioned LFSR pair, we expect ~50% of the bits
	// to differ. Anywhere in the 300..700 range proves the seeds
	// are driving distinct sequences without overfitting the test.
	if differences < 300 || differences > 700 {
		t.Errorf("expected ~50%% bit differences across seeds; got %d/1000", differences)
	}
}

// TestPN44SeedInboundMatchesAdvance243 verifies that
// PN44SeedInbound(seed) returns the state of an LFSR initialised
// at seed and clocked 243 times — the spec's matrix multiplication
// in Figure 7-4 is equivalent to this direct iteration.
func TestPN44SeedInboundMatchesAdvance243(t *testing.T) {
	seed := PN44SeedFromIdentity(0xABCDE, 0x123, 0xF7)
	want := NewPN44Scrambler(seed)
	for i := 0; i < 243; i++ {
		want.Next()
	}
	got := PN44SeedInbound(seed)
	if got != want.state {
		t.Errorf("PN44SeedInbound(%#x) = %#x; want %#x", seed, got, want.state)
	}
}

// TestPN44ScramblerAdvanceAdvancesState confirms Advance(n) is
// equivalent to calling Next() n times.
func TestPN44ScramblerAdvanceAdvancesState(t *testing.T) {
	seed := uint64(0x12345)
	a := NewPN44Scrambler(seed)
	a.Advance(50)
	b := NewPN44Scrambler(seed)
	for i := 0; i < 50; i++ {
		b.Next()
	}
	if a.State() != b.State() {
		t.Errorf("Advance(50) state %#x != Next()×50 state %#x", a.State(), b.State())
	}
}

// TestPN44LFSRPeriodLowerBound clocks the scrambler enough cycles to
// confirm it doesn't return to the initial state too early — the
// PN44 generator polynomial yields a maximum-length sequence with
// period 2^44 - 1, so we expect the state never to revisit the seed
// within a smaller number of cycles.
//
// Clocking 2^44 cycles would take too long for a unit test; the
// 10000-cycle bound below is enough to catch a trivial cycling bug
// (e.g. if the LFSR self-loops after a single shift).
func TestPN44LFSRPeriodLowerBound(t *testing.T) {
	seed := PN44SeedFromIdentity(0xABCDE, 0x123, 0xF7)
	s := NewPN44Scrambler(seed)
	initial := s.State()
	for i := 1; i <= 10000; i++ {
		s.Next()
		if s.State() == initial {
			t.Fatalf("scrambler returned to initial state %#x after %d shifts", initial, i)
		}
	}
}

func equalByteSlice(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
