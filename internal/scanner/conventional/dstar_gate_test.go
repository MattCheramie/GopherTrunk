package conventional

import (
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
)

// dstarTransmissionBits is one D-STAR transmission in the on-air format
// (preamble, header, nFrames DV frames of random voice, end pattern).
// Random voice bits are fine here: the gate keys on the syncs and the
// header, not the AMBE content.
func dstarTransmissionBits(r *rand.Rand, nFrames int, withHeader bool) []uint8 {
	var out []uint8
	if withHeader {
		out = append(out, dstar.BitSyncBits(320)...)
		out = append(out, dstar.EncodeHeaderBurst(dstar.SealHeader(dstar.Header{
			RPT2: "KD0AAA G", RPT1: "KD0AAA B", UR: "CQCQCQ  ", MY1: "W1AW    ", MY2: "ID51",
		}))...)
	}
	voice := make([]uint8, dstar.DVVoiceBits)
	for n := 0; n < nFrames; n++ {
		for i := range voice {
			voice[i] = uint8(r.Intn(2))
		}
		out = append(out, dstar.EncodeDVFrame(voice, n, [3]byte{byte(r.Intn(256)), 'G', 'T'})...)
	}
	return append(out, dstar.EndSyncBitsSlice()...)
}

// modulateDStar is GMSK at 4800 bit/s, BT 0.5, ±1.2 kHz, at 48 kHz, plus a
// little noise. invert flips the deviation sense, as a radio of the other
// polarity would.
func modulateDStar(r *rand.Rand, bits []uint8, invert bool) []complex64 {
	if invert {
		b := make([]uint8, len(bits))
		for i, x := range bits {
			b[i] = x ^ 1
		}
		bits = b
	}
	iq := demod.ModulateGFSK(bits, 10, 4, 0.5, dmrGateTestRate, 1200)
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(r.NormFloat64()*0.01), float32(r.NormFloat64()*0.01))
	}
	return iq
}

func dstarGateFirstOpen(d *DStarCallDetector, iq []complex64, rate float64) (first time.Duration, last bool) {
	first = -1
	step := int(rate / 100)
	for i := 0; i < len(iq); i += step {
		open := d.Process(iq[i:min(i+step, len(iq))])
		if open && first < 0 {
			first = time.Duration(float64(i) / rate * float64(time.Second))
		}
		last = open
	}
	return first, last
}

// Joining a transmission already under way (no header heard) opens the
// gate within its scan window, at the scanner's 2.4 MS/s, in either
// polarity, from any starting point.
func TestDStarCallDetectorOpensMidTransmission(t *testing.T) {
	r := rand.New(rand.NewSource(61))
	bits := dstarTransmissionBits(r, 200, false) // 4 s
	for _, inv := range []bool{false, true} {
		base := modulateDStar(r, bits, inv)
		worst := time.Duration(0)
		for off := 0; off < 20; off++ {
			// ~21 ms further in each time, at a new sub-symbol phase. Only
			// the scan window (+100 ms) is upsampled and fed: an open after
			// it fails the test anyway, and the 4 s transmission at 2.4 MS/s
			// through both receiver lanes, 40 times, was the package's
			// slowest test under -race.
			start := off * 1013
			iq := dmrAt2400k(base[start:min(start+int((dstarGateMinDwell+100*time.Millisecond).Seconds()*dmrGateTestRate), len(base))])
			d := NewDStarCallDetector(2_400_000)
			first, _ := dstarGateFirstOpen(d, iq, 2_400_000)
			if first < 0 || first > d.minDwell() {
				t.Errorf("inverted=%v offset %d: gate opened after %v, want within %v", inv, off, first, d.minDwell())
			}
			worst = max(worst, first)
		}
		t.Logf("inverted=%v: slowest open %v", inv, worst)
	}
}

// The start of a transmission opens the gate on its header, before the
// first slow-data sync pair could.
func TestDStarCallDetectorOpensOnHeader(t *testing.T) {
	r := rand.New(rand.NewSource(62))
	for _, inv := range []bool{false, true} {
		bits := dstarTransmissionBits(r, 10, true) // header + 200 ms: one slow-data sync
		iq := append(noiseIQ(r, 0.2), modulateDStar(r, bits, inv)...)
		d := NewDStarCallDetector(dmrGateTestRate)
		if first, _ := dstarGateFirstOpen(d, iq, dmrGateTestRate); first < 0 {
			t.Errorf("inverted=%v: header did not open the gate", inv)
		}
		// Only one slow-data sync follows, so the header is the only
		// possible evidence (each receiver lane may confirm it).
		if d.Evidence() < 1 {
			t.Errorf("inverted=%v: no evidence", inv)
		}
	}
}

// Analog carriers, a 2-level data carrier that is not D-STAR, and receiver
// noise never open the gate.
func TestDStarCallDetectorIgnoresOtherSignals(t *testing.T) {
	d := NewDStarCallDetector(dmrGateTestRate)
	if first, _ := dstarGateFirstOpen(d, readCS16Fixture(t, analogToneFixture), dmrGateTestRate); first >= 0 {
		t.Errorf("gate opened on the reporter's analog-tone capture after %v", first)
	}
	rng := rand.New(rand.NewSource(63))
	for _, tc := range []struct {
		name string
		iq   []complex64
	}{
		{"1 kHz tone, ±2.5 kHz", fmToneCarrier(rng, 5, -1400, 1000, 2500)},
		{"unmodulated carrier", fmToneCarrier(rng, 5, 300, 0, 0)},
		{"random GMSK data", func() []complex64 {
			b := make([]uint8, 5*4800)
			for i := range b {
				b[i] = uint8(rng.Intn(2))
			}
			return modulateDStar(rng, b, false)
		}()},
		{"noise", noiseIQ(rng, 30)},
	} {
		d := NewDStarCallDetector(dmrGateTestRate)
		if first, _ := dstarGateFirstOpen(d, tc.iq, dmrGateTestRate); first >= 0 {
			t.Errorf("%s: gate opened after %v", tc.name, first)
		}
	}
}

// The end pattern closes the gate at once, and Reset clears it.
func TestDStarCallDetectorClosesOnEnd(t *testing.T) {
	r := rand.New(rand.NewSource(64))
	bits := dstarTransmissionBits(r, 100, true)
	iq := append(modulateDStar(r, bits, false), noiseIQ(r, 0.2)...)
	d := NewDStarCallDetector(dmrGateTestRate)
	first, last := dstarGateFirstOpen(d, iq, dmrGateTestRate)
	if first < 0 {
		t.Fatal("gate never opened")
	}
	if last {
		t.Fatal("gate still open 200 ms after the end pattern (its hold is 1 s)")
	}
	d.Reset()
	if d.Present() || d.Evidence() != 0 {
		t.Fatal("Reset left the gate open")
	}
}
