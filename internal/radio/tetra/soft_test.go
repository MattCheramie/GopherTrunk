package tetra

import (
	"math"
	"math/rand"
	"testing"
)

// dibitIdealDiff returns the ideal π/4-DQPSK differential the demod sees
// for a given dibit value: angle = rotation(π/4) + the dibit's phase
// increment. Matches demod.dibitToDQPSKPhase + the receiver rotation.
func dibitIdealDiff(v uint8) complex64 {
	theta := map[uint8]float64{0: math.Pi / 4, 1: 3 * math.Pi / 4, 2: -3 * math.Pi / 4, 3: -math.Pi / 4}[v&3]
	return complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
}

// TestSoftType5Polarity guards the load-bearing invariant: the soft
// type-5 LLRs (softType5FromDiffs) hard-slice to exactly the hard bits
// TetraDibitsToBits(rotateDibits(...)) produces, for every dibit value
// and every constellation rotation. A polarity or rotation error here
// would make the soft decoder silently disagree with the hard one.
func TestSoftType5Polarity(t *testing.T) {
	for v := uint8(0); v < 4; v++ {
		d := []complex64{dibitIdealDiff(v)}
		for rot := uint8(0); rot < 4; rot++ {
			soft := softType5FromDiffs(d, rot)
			hard := TetraDibitsToBits(rotateDibits([]uint8{v}, rot))
			for i := range hard {
				softBit := byte(0)
				if soft[i] < 0 { // LLR < 0 ⇒ bit 1
					softBit = 1
				}
				if softBit != hard[i] {
					t.Errorf("dibit %d rot %d bit %d: soft slices to %d (llr %.2f), hard %d",
						v, rot, i, softBit, soft[i], hard[i])
				}
			}
		}
	}
}

// schhdInfo builds a deterministic 124-bit SCH/HD type-1 block.
func schhdInfo() []byte {
	info := make([]byte, 124)
	for i := range info {
		info[i] = byte((i*7 + 3) & 1)
	}
	return info
}

// bitsToLLR maps hard type-5 bits to LLRs (bit 0 → +1, bit 1 → -1) plus
// Gaussian noise of the given sigma. Returns the LLRs and the hard-sliced
// bits (sign decision) so the hard and soft decoders see the same channel.
func bitsToLLR(bits []byte, sigma float64, rng *rand.Rand) ([]float32, []byte) {
	llr := make([]float32, len(bits))
	hard := make([]byte, len(bits))
	for i, b := range bits {
		mean := 1.0
		if b&1 == 1 {
			mean = -1.0
		}
		v := mean + rng.NormFloat64()*sigma
		llr[i] = float32(v)
		if v < 0 {
			hard[i] = 1
		}
	}
	return llr, hard
}

// TestSoftSCHHDRoundTripCleanMatchesHard: with no noise, soft decode
// recovers the info and passes CRC, identical to hard.
func TestSoftSCHHDRoundTripClean(t *testing.T) {
	info := schhdInfo()
	const colour = 0x12345
	type5 := EncodeSCHHD(info, colour)
	llr, hard := bitsToLLR(type5, 0, rand.New(rand.NewSource(1)))

	got, ok := DecodeSCHHDSoft(llr, colour)
	if !ok {
		t.Fatal("soft SCH/HD failed CRC on a clean channel")
	}
	for i := range info {
		if got[i] != info[i] {
			t.Fatalf("soft SCH/HD recovered bit %d = %d, want %d", i, got[i], info[i])
		}
	}
	if _, ok := DecodeSCHHD(hard, colour); !ok {
		t.Fatal("hard SCH/HD failed CRC on a clean channel (sanity)")
	}
}

// TestSoftSCHHDOutperformsHardUnderNoise: at a noise level where the
// hard decoder mostly fails, the soft decoder mostly succeeds —
// demonstrating the ~2 dB soft-decision coding gain through the full
// descramble → deinterleave → depuncture → Viterbi chain.
func TestSoftSCHHDOutperformsHardUnderNoise(t *testing.T) {
	info := schhdInfo()
	const colour = 0x12345
	type5 := EncodeSCHHD(info, colour)
	rng := rand.New(rand.NewSource(42))
	const (
		sigma  = 0.7
		trials = 200
	)
	hardPass, softPass := 0, 0
	for tr := 0; tr < trials; tr++ {
		llr, hard := bitsToLLR(type5, sigma, rng)
		if _, ok := DecodeSCHHD(hard, colour); ok {
			hardPass++
		}
		if got, ok := DecodeSCHHDSoft(llr, colour); ok {
			softPass++
			// A CRC pass must mean correct recovery.
			for i := range info {
				if got[i] != info[i] {
					t.Fatalf("soft CRC passed but bit %d wrong", i)
				}
			}
		}
	}
	t.Logf("sigma=%.2f: hard %d/%d, soft %d/%d", sigma, hardPass, trials, softPass, trials)
	// Soft must be a clear win: at this noise the hard decoder mostly
	// fails while soft mostly succeeds.
	if softPass < 60 || softPass < 2*hardPass {
		t.Errorf("soft gain insufficient: hard %d, soft %d of %d (want soft>=60 and >=2x hard)", hardPass, softPass, trials)
	}
}

// TestSoftBSCHOutperformsHardUnderNoise: same gain on the colour-0 BSCH
// (the cold-start lock chain).
func TestSoftBSCHOutperformsHardUnderNoise(t *testing.T) {
	info := make([]byte, 60)
	for i := range info {
		info[i] = byte((i*5 + 1) & 1)
	}
	type5 := EncodeBSCH(info)
	rng := rand.New(rand.NewSource(7))
	const (
		sigma  = 0.7
		trials = 200
	)
	hardPass, softPass := 0, 0
	for tr := 0; tr < trials; tr++ {
		llr, hard := bitsToLLR(type5, sigma, rng)
		if _, ok := DecodeBSCH(hard); ok {
			hardPass++
		}
		if _, ok := DecodeBSCHSoft(llr); ok {
			softPass++
		}
	}
	t.Logf("BSCH sigma=%.2f: hard %d/%d, soft %d/%d", sigma, hardPass, trials, softPass, trials)
	if softPass < 60 || softPass < 2*hardPass {
		t.Errorf("soft BSCH gain insufficient: hard %d, soft %d of %d", hardPass, softPass, trials)
	}
}
