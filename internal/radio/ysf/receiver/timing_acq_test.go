package receiver

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
)

// ysfFrameStream is n YSF-shaped frames (FSW + 460 random dibits) modulated
// as C4FM at 48 kHz / 10 sps.
func ysfFrameStream(rng *rand.Rand, n int) []complex64 {
	var d []uint8
	for i := 0; i < n; i++ {
		d = append(d, ysf.FSWPattern...)
		for k := 0; k < ysf.FrameDibits-ysf.FSWDibits; k++ {
			d = append(d, uint8(rng.Intn(4)))
		}
	}
	iq := demod.ModulateC4FM(d, 10, 8, 0.20, 48_000, 1800)
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

// firstSyncSeconds runs a receiver over iq offset by phase samples and
// returns when its first frame sync was found, or -1.
func firstSyncSeconds(iq []complex64, phase int, arm bool) (float64, int) {
	iq = append(make([]complex64, phase), iq...)
	det := ysf.NewSyncDetector(2)
	first := -1
	var hits []int
	r := New(Options{
		SampleRateHz: 48_000, DeviationHz: 1800, EnableAFC: true, EnableChannelFilter: true,
		DibitSink: func(d []uint8, base int) {
			hits, _ = det.Process(hits[:0], d, base)
			if first < 0 && len(hits) > 0 {
				first = hits[0]
			}
		},
	})
	if arm {
		r.ArmTimingAcquisition()
	}
	for i := 0; i < len(iq); i += 1024 {
		r.Process(iq[i:min(i+1024, len(iq))])
	}
	if first < 0 {
		return -1, r.TimingSeeds()
	}
	return float64(first) / 4800, r.TimingSeeds()
}

// TestReceiverTimingAcquisitionSeedsAnyPhase: a receiver started on a
// running transmission at an arbitrary sub-symbol phase must find a frame
// sync within a few frames whatever the phase. Without the acquisition the
// loop pulled in from near a half symbol only after 0.6–1.2 s (issue
// #1261, the scanner's YSF call gate); with it the first sync comes in the
// second frame at every phase.
func TestReceiverTimingAcquisitionSeedsAnyPhase(t *testing.T) {
	rng := rand.New(rand.NewSource(41))
	iq := ysfFrameStream(rng, 20) // 2 s
	slowWithout := 0
	for phase := 0; phase < 10; phase++ {
		at, seeds := firstSyncSeconds(iq, phase, true)
		if at < 0 || at > 0.35 {
			t.Errorf("phase %d/10: armed receiver found its first frame sync at %.2f s (seeds %d), want within 0.35 s", phase, at, seeds)
		}
		if seeds != 1 {
			t.Errorf("phase %d/10: %d timing seeds, want exactly 1", phase, seeds)
		}
		if at, _ := firstSyncSeconds(iq, phase, false); at < 0 || at > 0.35 {
			slowWithout++
		}
	}
	if slowWithout == 0 {
		t.Log("note: the unarmed loop locked within 0.35 s at every phase on this fixture; the acquisition still seeds, but the fixture no longer shows the slow pull-in")
	}
}

// TestReceiverUnarmedIsUnchanged: without ArmTimingAcquisition the receiver
// never seeds, so every existing stream decodes exactly as before.
func TestReceiverUnarmedIsUnchanged(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	iq := ysfFrameStream(rng, 5)
	if _, seeds := firstSyncSeconds(iq, 3, false); seeds != 0 {
		t.Fatalf("unarmed receiver applied %d timing seeds", seeds)
	}
}
