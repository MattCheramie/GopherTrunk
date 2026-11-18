package receiver

import (
	"encoding/binary"
	"math/bits"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// mmrPhase2Fixture is 1.09 s of a real P25 Phase 2 TDMA carrier: the MMR
// (Victoria) idle voice channel at 420.650 MHz in samples/mmr/mmr-s9-1.cfile
// (RTL-SDR, 2.4 MS/s centred on 420.0125 MHz, the carrier at +637.5 kHz),
// mixed to 0 Hz and decimated to 48 kHz by the composer's Phase 2 voice front
// end, stored as little-endian int16 I/Q. The site's control channel in the
// same capture is WACN BEE00 / system 164 / NAC 164.
const mmrPhase2Fixture = "testdata/mmr-s9-420650k-p2-48k.cs16"

func loadCS16(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

// firstSISCHSeconds runs a ClockGardner receiver (the voice chain's options)
// over iq in 655-sample chunks, the chunk size the voice chain hands it at
// 2.4 MS/s, and returns the stream time of the first S-ISCH (canonical
// rotation, at most 4 bit errors) in its dibits, or -1.
func firstSISCHSeconds(iq []complex64, arm bool) float64 {
	var dibits []uint8
	rx := New(Options{
		SampleRateHz: 48_000,
		ClockMode:    ClockGardner,
		GardnerGain:  0.005,
		DibitSink:    func(d []uint8, _ int) { dibits = append(dibits, d...) },
	})
	if arm {
		rx.ArmTimingAcquisition()
	}
	for i := 0; i < len(iq); i += 655 {
		rx.Process(iq[i:min(i+655, len(iq))])
	}
	var w uint64
	for i, d := range dibits {
		w = (w<<2 | uint64(d&3)) & (1<<40 - 1)
		if i >= phase2.AirISCHDibits-1 && bits.OnesCount64(w^phase2.SISCHSync) <= 4 {
			return float64(i-(phase2.AirISCHDibits-1)) / SymbolRate
		}
	}
	return -1
}

// TestRealAirTimingAcquisitionEveryStartPhase is the issue #1364 regression:
// a voice chain starts at the grant, at an arbitrary sub-symbol phase of a
// carrier that is already up, and MAC_PTT (the call's source radio) is sent
// at key-up. On this real carrier the Gardner loop alone found the first
// S-ISCH 32 ms in at six of the eight start phases but 0.27 s and 0.99 s in
// at the other two; armed, the feed-forward acquisition seeds it and every
// phase syncs within the first superframe and a half.
func TestRealAirTimingAcquisitionEveryStartPhase(t *testing.T) {
	iq := loadCS16(t, mmrPhase2Fixture)
	const maxAcq = 0.15 // seconds
	worstUnarmed := 0.0
	for k := 0; k < 8; k++ {
		armed := firstSISCHSeconds(iq[k:], true)
		unarmed := firstSISCHSeconds(iq[k:], false)
		t.Logf("start phase %d: first S-ISCH armed %.3f s, loop alone %.3f s", k, armed, unarmed)
		if armed < 0 || armed > maxAcq {
			t.Errorf("start phase %d: armed receiver's first S-ISCH at %.3f s, want within %.2f s", k, armed, maxAcq)
		}
		if unarmed < 0 || unarmed > worstUnarmed {
			worstUnarmed = unarmed
			if unarmed < 0 {
				worstUnarmed = 10
			}
		}
	}
	// The fixture must keep exercising a slow start phase, or the bound
	// above proves nothing.
	if worstUnarmed < 0.5 {
		t.Errorf("loop alone synced within %.3f s at every phase: the fixture no longer shows the slow start this test guards", worstUnarmed)
	}
}

// TestArmTimingAcquisitionNoOpUnarmed pins that an unarmed receiver is
// untouched: the acquisition is opt-in, so every existing stream decodes
// byte-identically.
func TestArmTimingAcquisitionNoOpUnarmed(t *testing.T) {
	iq := loadCS16(t, mmrPhase2Fixture)
	run := func() []uint8 {
		var out []uint8
		rx := New(Options{SampleRateHz: 48_000, ClockMode: ClockGardner, GardnerGain: 0.005,
			DibitSink: func(d []uint8, _ int) { out = append(out, d...) }})
		for i := 0; i < len(iq); i += 655 {
			rx.Process(iq[i:min(i+655, len(iq))])
		}
		if rx.TimingSeeds() != 0 {
			t.Fatalf("unarmed receiver applied %d timing seeds", rx.TimingSeeds())
		}
		return out
	}
	if a, b := run(), run(); string(a) != string(b) {
		t.Fatal("unarmed receiver is not deterministic")
	}
}

// TestArmTimingAcquisitionSeedsOnceAndResetDisarms checks the seed fires once
// on the real carrier, and that Reset cancels a pending acquisition.
func TestArmTimingAcquisitionSeedsOnceAndResetDisarms(t *testing.T) {
	iq := loadCS16(t, mmrPhase2Fixture)
	rx := New(Options{SampleRateHz: 48_000, ClockMode: ClockGardner, GardnerGain: 0.005,
		DibitSink: func([]uint8, int) {}})
	rx.ArmTimingAcquisition()
	rx.Process(iq)
	if got := rx.TimingSeeds(); got != 1 {
		t.Fatalf("TimingSeeds = %d after one armed pass, want 1", got)
	}
	rx.ArmTimingAcquisition()
	rx.Reset()
	rx.Process(iq)
	if got := rx.TimingSeeds(); got != 1 {
		t.Fatalf("TimingSeeds = %d: Reset did not disarm the pending acquisition", got)
	}
}
