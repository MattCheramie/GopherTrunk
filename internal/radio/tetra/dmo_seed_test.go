package tetra

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// The seed solver rests on two linearity claims about code GT already ships;
// pin both numerically rather than by argument.
func TestTCHSeedLinearStructure(t *testing.T) {
	lin := tchSeedLinearInstance()
	if got := len(lin.checks); got != tchCheckBits {
		t.Fatalf("parity checks = %d, want %d (330 coded bits − 172 information bits)", got, tchCheckBits)
	}
	// Every check annihilates every codeword: random speech, no scrambling.
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 20; trial++ {
		info := make([]byte, tchInfoBits)
		for i := range info {
			info[i] = byte(rng.Intn(2))
		}
		coded := tchCodedBitsFor(info)
		var type5 [tchType3Bits]byte
		for k, b := range coded {
			type5[tchType5IndexOfCoded(k)] = b
		}
		w := packBits432(type5[:])
		for ri := range lin.checks {
			if parityWords(&lin.checks[ri], &w) != 0 {
				t.Fatalf("trial %d: parity check %d does not annihilate a valid codeword", trial, ri)
			}
		}
	}
	// PN(seed) is affine in the seed: PN(s1^s2) = PN(s1) ^ PN(s2) ^ PN(0).
	for trial := 0; trial < 20; trial++ {
		s1, s2 := uint32(rng.Intn(1<<30)), uint32(rng.Intn(1<<30))
		p0 := framing.NewScramblerTetra(0).Generate(tchType3Bits)
		p1 := framing.NewScramblerTetra(s1).Generate(tchType3Bits)
		p2 := framing.NewScramblerTetra(s2).Generate(tchType3Bits)
		p12 := framing.NewScramblerTetra(s1 ^ s2).Generate(tchType3Bits)
		for k := range p0 {
			if p12[k] != p1[k]^p2[k]^p0[k] {
				t.Fatalf("scrambler is not affine in its seed at bit %d", k)
			}
		}
	}
	// H'·P has full column rank: every seed bit is observable from one burst.
	rows := append([]uint32(nil), lin.a...)
	rank := 0
	for col := 0; col < tchSeedBits; col++ {
		sel := -1
		for i := rank; i < len(rows); i++ {
			if rows[i]&(1<<uint(col)) != 0 {
				sel = i
				break
			}
		}
		if sel < 0 {
			continue
		}
		rows[rank], rows[sel] = rows[sel], rows[rank]
		for i := range rows {
			if i != rank && rows[i]&(1<<uint(col)) != 0 {
				rows[i] ^= rows[rank]
			}
		}
		rank++
	}
	if rank != tchSeedBits {
		t.Fatalf("H'·P rank = %d, want %d", rank, tchSeedBits)
	}
}

// A scrambled TCH/S block solves to exactly the seed it was scrambled with —
// for arbitrary 30-bit seeds, not just the 64 colours the brute force tried —
// and a single bit error is rejected rather than mis-solved.
func TestSolveTCHScrambleSeedRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 200; trial++ {
		frameA, frameB := randomSpeechFrames(rng)
		seed := uint32(rng.Intn(1 << 30))
		switch trial % 4 { // include the structured shapes an operator meets
		case 1:
			seed = uint32(rng.Intn(64)) // MNI 0, bare colour
		case 2:
			seed = ExtendedColourCode(250, 1, uint8(rng.Intn(64)))
		}
		type4 := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
		type5 := framing.ScrambleTetra(type4, seed)
		got, ok := SolveTCHScrambleSeed(type5)
		if !ok || got != seed {
			t.Fatalf("trial %d: seed %#x solved to %#x ok=%v", trial, seed, got, ok)
		}
		// Flip one bit inside the coded region: no seed explains it.
		flipped := append([]byte(nil), type5...)
		k := tchType5IndexOfCoded(rng.Intn(tchCodedBits))
		flipped[k] ^= 1
		if _, ok := SolveTCHScrambleSeed(flipped); ok {
			t.Fatalf("trial %d: a corrupted block solved to a seed", trial)
		}
		// Flipping an UNCODED class-0 bit must not disturb the solve.
		flipped = append([]byte(nil), type5...)
		t3 := rng.Intn(tchClass0Bits)
		flipped[(t3%18)*24+t3/18] ^= 1
		if got, ok := SolveTCHScrambleSeed(flipped); !ok || got != seed {
			t.Fatalf("trial %d: class-0 bit flip broke the solve (got %#x ok=%v)", trial, got, ok)
		}
	}
}

// RecoverDMScrambleSeed on a synthetic DNB train: bursts scrambled with one
// arbitrary seed (a non-zero MNI and colour, as on air) vote it in, and the
// CRC cross-check confirms it. Noise bursts (random dibits) neither vote nor
// confuse it, and a train with no agreeing solutions is not trusted.
func TestRecoverDMScrambleSeed(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	seed := ExtendedColourCode(250, 13, 39)
	var bursts []DMBurst
	for i := 0; i < 12; i++ {
		frameA, frameB := randomSpeechFrames(rng)
		type4 := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
		type5 := framing.ScrambleTetra(type4, seed)
		dibits := TetraBitsToDibits(type5)
		bursts = append(bursts, DMBurst{Kind: DMBurstNormal, BKN1: dibits[:dmBlockDibits], BKN2: dibits[dmBlockDibits:]})
		// Interleave a noise burst.
		noise := make([]uint8, 2*dmBlockDibits)
		for k := range noise {
			noise[k] = uint8(rng.Intn(4))
		}
		bursts = append(bursts, DMBurst{Kind: DMBurstNormal, BKN1: noise[:dmBlockDibits], BKN2: noise[dmBlockDibits:]})
	}
	got, votes, ok := RecoverDMScrambleSeed(bursts)
	if !ok || got != seed || votes != 12 {
		t.Fatalf("recovered %#x votes=%d ok=%v, want %#x votes=12 ok=true", got, votes, ok, seed)
	}
	if _, _, ok := RecoverDMScrambleSeed(bursts[1:2]); ok {
		t.Fatal("a single noise burst must not yield a confident seed")
	}
}

func randomSpeechFrames(rng *rand.Rand) (frameA, frameB []byte) {
	frameA = make([]byte, tchSpeechFrameBits)
	frameB = make([]byte, tchSpeechFrameBits)
	for i := range frameA {
		frameA[i] = byte(rng.Intn(2))
		frameB[i] = byte(rng.Intn(2))
	}
	return frameA, frameB
}
