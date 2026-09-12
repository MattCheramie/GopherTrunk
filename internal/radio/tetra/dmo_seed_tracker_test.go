package tetra

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Literal on-air vectors from the 12 Sep #1003 capture (438.9 MHz DMO, three
// consecutive PTTs): each transmission's decoded DSB SCH/H (124 type-1 bits,
// hex, MSB-first) and the scramble seed its DNBs solved to exactly
// (TestTETRADMOSeedScan). The seed's low 24 bits sit at SCH/H bits 42..65 with
// the MNI (MCC 250 / MNC 1 — the operator's codeplug) at 66..89; the top 6
// bits were 000001 on all three. A field-layout pin against real air, not a
// round trip.
var dmoFieldSCHH = []struct {
	schh string
	seed uint32
}{
	{"00200000034a45700fa0005ae0000440", 0x012915c0},
	{"0020000003576701cfa0005ae0000440", 0x015d9c07},
	{"002000000359c4e10fa0005ae0000440", 0x01671384},
}

func TestDMSyncSCHHScrambleSeedFieldCapture(t *testing.T) {
	for _, v := range dmoFieldSCHH {
		raw, err := hex.DecodeString(v.schh)
		if err != nil {
			t.Fatal(err)
		}
		bits := framing.UnpackBitsMSB(raw, 124)
		got, ok := DMSyncSCHHScrambleSeed(bits)
		if !ok || got != v.seed {
			t.Errorf("SCH/H %s: seed %#010x ok=%v, want %#010x", v.schh, got, ok, v.seed)
		}
		// The MNI right after the field is the operator's network.
		if mcc, mnc := bitsToUint(bits, 66, 10), bitsToUint(bits, 76, 14); mcc != 250 || mnc != 1 {
			t.Errorf("SCH/H %s: MNI after the seed field = %d/%d, want 250/1", v.schh, mcc, mnc)
		}
	}
}

// dmoTrackerDNB builds one TCH/S DNB scrambled with seed (optionally with bit
// errors so it cannot solve exactly).
func dmoTrackerDNB(rng *rand.Rand, seed uint32, flips int) DMBurst {
	frameA, frameB := randomSpeechFrames(rng)
	type4 := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
	type5 := framing.ScrambleTetra(type4, seed)
	for i := 0; i < flips; i++ {
		type5[tchType5IndexOfCoded(rng.Intn(tchCodedBits))] ^= 1
	}
	dibits := TetraBitsToDibits(type5)
	return DMBurst{Kind: DMBurstNormal, BKN1: dibits[:dmBlockDibits], BKN2: dibits[dmBlockDibits:]}
}

// dmoTrackerDSB builds a DSB whose SCH/H announces seed's low 24 bits at the
// capture-pinned field.
func dmoTrackerDSB(seed uint32) DMBurst {
	type1 := make([]byte, 124)
	for i := 0; i < 124; i++ {
		type1[i] = byte((i * 7) % 2)
	}
	for i := 0; i < DMSCHHSeedFieldBits; i++ {
		type1[DMSCHHSeedFieldOffset+i] = byte(seed >> uint(DMSCHHSeedFieldBits-1-i) & 1)
	}
	return DMBurst{Kind: DMBurstSync, BKN2: TetraBitsToDibits(EncodeSCHHD(type1, 0))}
}

// A transmission whose seed changes at the next PTT (the on-air behaviour) is
// followed: the exact solve adopts each new seed on its first clean burst, and
// every burst decodes at its own transmission's seed. The old 64-colour brute
// force could not represent either seed (both > 63 with MNI 0).
func TestDMSeedTrackerFollowsPerTransmissionSeeds(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	tr := NewDMSeedTracker()
	for _, seed := range []uint32{0x012915c0, 0x015d9c07, 0x01671384} {
		for i := 0; i < 10; i++ {
			frames, got, _ := tr.ObserveDNB(dmoTrackerDNB(rng, seed, 0), true)
			if frames == nil || got != seed {
				t.Fatalf("seed %#x burst %d: frames=%v seed=%#x", seed, i, frames != nil, got)
			}
		}
		if !tr.Verified() {
			t.Fatalf("seed %#x not verified", seed)
		}
	}
	if tr.ExactAdopts != 3 {
		t.Errorf("exact adopts = %d, want 3", tr.ExactAdopts)
	}
}

// Weak bursts (bit errors, no exact solution) are decoded through the SCH/H
// hint once two of them CRC-confirm it; a wrong hint (another radio's DSB
// during the call) never latches, and the exact solve still wins when a clean
// burst arrives.
func TestDMSeedTrackerHintNeedsCRCConfirmation(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	const seed, wrong = 0x015d9c07, 0x012915c0
	tr := NewDMSeedTracker()
	if _, ok := tr.ObserveDSB(dmoTrackerDSB(wrong)); !ok {
		t.Fatal("DSB SCH/H did not decode")
	}
	// Errored bursts at the true seed cannot confirm the wrong hint.
	for i := 0; i < 8; i++ {
		if frames, _, adopted := tr.ObserveDNB(dmoTrackerDNB(rng, seed, 3), true); adopted || frames != nil {
			t.Fatalf("burst %d: wrong hint produced frames=%v adopted=%v", i, frames != nil, adopted)
		}
	}
	if _, known := tr.Seed(); known {
		t.Fatal("a seed was adopted from an unconfirmed hint")
	}
	// The transmitting radio's DSB: the right hint confirms on the third
	// CRC-valid decode even though no burst solves exactly.
	tr.ObserveDSB(dmoTrackerDSB(seed))
	decoded, adoptedAt := 0, -1
	for i := 0; i < 12; i++ {
		frames, got, adopted := tr.ObserveDNB(dmoTrackerDNB(rng, seed, 3), true)
		if frames != nil {
			decoded++
		}
		if adopted && adoptedAt < 0 {
			adoptedAt = i
			if got != seed {
				t.Fatalf("adopted %#x, want %#x", got, seed)
			}
		}
	}
	if adoptedAt < 0 || tr.HintAdopts != 1 || decoded < 6 {
		t.Fatalf("hint adoption at %d, hint_adopts=%d, decoded=%d", adoptedAt, tr.HintAdopts, decoded)
	}
	// A clean burst from a NEW transmission preempts the hint-adopted seed.
	if _, got, adopted := tr.ObserveDNB(dmoTrackerDNB(rng, 0x01671384, 0), true); !adopted || got != 0x01671384 {
		t.Fatalf("exact solve did not preempt: adopted=%v seed=%#x", adopted, got)
	}
}

// A pinned (configured) seed is used as is and never displaced; Reset keeps it.
func TestDMSeedTrackerPinnedSeed(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	tr := NewDMSeedTracker()
	tr.Pin(0x2c)
	if frames, got, adopted := tr.ObserveDNB(dmoTrackerDNB(rng, 0x012915c0, 0), true); adopted || got != 0x2c || frames != nil {
		t.Fatalf("pinned seed changed: frames=%v seed=%#x adopted=%v", frames != nil, got, adopted)
	}
	tr.Reset()
	if s, known := tr.Seed(); !known || s != 0x2c {
		t.Fatalf("pinned seed lost on Reset: %#x known=%v", s, known)
	}
}

// loadDMOFixtureDNB reads a literal on-air DNB from testdata: rotation (u8),
// BKN1 (108 dibits), BKN2 (108 dibits), then the 216 receiver differentials as
// little-endian float32 (re, im) pairs — exactly the DMBurst the production
// stream extractor emitted for that burst.
func loadDMOFixtureDNB(t *testing.T, name string) DMBurst {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	const n = 2 * dmBlockDibits
	if len(raw) != 1+n+n*8 {
		t.Fatalf("%s: %d bytes, want %d", name, len(raw), 1+n+n*8)
	}
	b := DMBurst{Kind: DMBurstNormal, Rotation: raw[0]}
	b.BKN1 = append([]uint8(nil), raw[1:1+dmBlockDibits]...)
	b.BKN2 = append([]uint8(nil), raw[1+dmBlockDibits:1+n]...)
	soft := make([]complex64, n)
	for i := range soft {
		off := 1 + n + i*8
		re := math.Float32frombits(binary.LittleEndian.Uint32(raw[off:]))
		im := math.Float32frombits(binary.LittleEndian.Uint32(raw[off+4:]))
		soft[i] = complex(re, im)
	}
	b.SoftBKN1, b.SoftBKN2 = soft[:dmBlockDibits], soft[dmBlockDibits:]
	return b
}

// Two DNBs from the operator's 13 Sep #1003 capture (438.9 MHz DMO, five
// PTTs) that the soft-assisted reliable-check solver "solves" to a seed at
// which they decode NOTHING: a slot-grid-qualified burst of the third
// transmission (seed 0x0001733855) returning 0x00028b4304 — which the live
// pipeline and voice chain both adopted and flipped back from a burst later
// ("composer: tetra DMO scramble seed changed" twice within 56 ms) — and a
// correlator false alarm off the idle channel returning 0x0003d69966. A solve
// that cannot decode its own burst is no proof of anything: it must neither
// displace a verified seed nor seed a fresh tracker.
func TestDMSeedTrackerRejectsSolveThatDoesNotDecode(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		bogus   uint32
		// wantFrames at the verified seed: the qualified burst is a real,
		// mildly errored burst of the transmission and decodes at its true
		// seed once the bogus solve stops displacing it (the gate does not
		// just avoid a flip, it rescues the burst's speech); the noise burst
		// decodes at nothing.
		wantFrames int
	}{
		{"dmo_13sep_false_solve_0x00028b43.dnb", 0x00028b4304, 2},
		{"dmo_13sep_false_solve_0x0003d699.dnb", 0x0003d69966, 0},
	} {
		b := loadDMOFixtureDNB(t, tc.fixture)
		// The solver itself still returns the bogus seed (the fixture pins
		// that the path exists); the tracker is what must not trust it.
		if s, ok := DMBurstScrambleSeed(b); !ok || s != tc.bogus {
			t.Fatalf("%s: solver returned %#010x ok=%v, fixture expects %#010x", tc.fixture, s, ok, tc.bogus)
		}
		if frames := dmDecodeDNB(b, tc.bogus); frames != nil {
			t.Fatalf("%s: decodes at the bogus seed — fixture no longer exercises the gate", tc.fixture)
		}

		const verified = 0x0001733855
		tr := NewDMSeedTracker()
		tr.Adopt(verified)
		frames, seed, adopted := tr.ObserveDNB(b, true)
		if adopted || seed != verified {
			t.Errorf("%s: displaced a verified seed: adopted=%v seed=%#010x", tc.fixture, adopted, seed)
		}
		if len(frames) != tc.wantFrames {
			t.Errorf("%s: decoded %d frames at the verified seed, want %d", tc.fixture, len(frames), tc.wantFrames)
		}
		if tr.SolveRejects != 1 {
			t.Errorf("%s: solve_rejects=%d, want 1", tc.fixture, tr.SolveRejects)
		}

		fresh := NewDMSeedTracker()
		if _, seed, adopted := fresh.ObserveDNB(b, false); adopted {
			t.Errorf("%s: a fresh tracker adopted %#010x from a burst that does not decode at it", tc.fixture, seed)
		}
		if _, known := fresh.Seed(); known {
			t.Errorf("%s: fresh tracker claims a seed", tc.fixture)
		}
	}
}
