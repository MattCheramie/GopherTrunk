package receiver

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// decodeMMRSignalling runs the voice chain's receiver over iq and decodes
// every signalling burst, framing with reliabilities (soft) or without. It
// returns the bursts' decoded PDUs and how many of them needed the soft
// retry.
func decodeMMRSignalling(iq []complex64, soft bool) (pdus [][]byte, erased int, dibits []uint8) {
	masks := phase2.NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164))
	fr := phase2.NewAirFramer()
	handle := func(bs []phase2.AirBurst) {
		for _, b := range bs {
			if !b.DUID.IsSignalling() {
				continue
			}
			if m, ok := phase2.DecodeAirMAC(b, masks); ok {
				pdus = append(pdus, m.Bytes)
				if m.Erased > 0 {
					erased++
				}
			}
		}
	}
	var rel []float32
	rx := New(Options{
		SampleRateHz: 48_000, ClockMode: ClockGardner, GardnerGain: 0.005,
		SoftDecision: true,
		SoftSink: func(d []uint8, s []complex64, base int) {
			dibits = append(dibits, d...)
			if soft {
				rel = phase2.SoftReliability(s, rel)
				handle(fr.ProcessSoft(d, rel, base))
			} else {
				handle(fr.Process(d, base))
			}
		},
	})
	rx.ArmTimingAcquisition()
	for i := 0; i < len(iq); i += 655 {
		rx.Process(iq[i:min(i+655, len(iq))])
	}
	return pdus, erased, dibits
}

// On a weak Phase 2 signal the signalling decode retries a burst its hard
// Reed-Solomon decode cannot correct with the burst's least reliable symbols
// erased (issue #1364). Measured on the real MMR carrier with white noise
// added so the wideband SNR (48 kHz) is -1 dB, over 30 noise draws: the hard
// decode recovers 50 of 161 signalling bursts per 30 draws, the soft retry 76
// (and, at 0 dB, 125 against 137). Every soft-decoded PDU must also be one
// the clean capture decodes: the idle carrier repeats a handful of PDUs, so a
// wrong decode would show up as bytes the clean capture never carries.
//
// The soft path also slices exactly the hard path's dibits (checked here on
// the clean capture), which is what lets every production receiver run it.
func TestSoftMACRetryRecoversWeakRealAirBursts(t *testing.T) {
	iq := loadCS16(t, mmrPhase2Fixture)

	clean, _, softDibits := decodeMMRSignalling(iq, true)
	var hardDibits []uint8
	rx := New(Options{SampleRateHz: 48_000, ClockMode: ClockGardner, GardnerGain: 0.005,
		DibitSink: func(d []uint8, _ int) { hardDibits = append(hardDibits, d...) }})
	rx.ArmTimingAcquisition()
	for i := 0; i < len(iq); i += 655 {
		rx.Process(iq[i:min(i+655, len(iq))])
	}
	if len(hardDibits) != len(softDibits) {
		t.Fatalf("hard path %d dibits, soft path %d", len(hardDibits), len(softDibits))
	}
	for i := range hardDibits {
		if hardDibits[i] != softDibits[i] {
			t.Fatalf("dibit %d: hard path %d, soft path %d", i, hardDibits[i], softDibits[i])
		}
	}
	known := map[string]bool{}
	for _, p := range clean {
		known[string(p)] = true
	}
	if len(clean) == 0 {
		t.Fatal("the clean capture decodes no signalling")
	}

	var p float64
	for _, x := range iq {
		p += float64(real(x)*real(x) + imag(x)*imag(x))
	}
	p /= float64(len(iq))
	const snrDB = -1
	sd := math.Sqrt(p / math.Pow(10, snrDB/10.0) / 2)
	var hard, soft, erased int
	for seed := int64(0); seed < 30; seed++ {
		r := rand.New(rand.NewSource(seed))
		noisy := make([]complex64, len(iq))
		for i, x := range iq {
			noisy[i] = x + complex(float32(r.NormFloat64()*sd), float32(r.NormFloat64()*sd))
		}
		h, _, _ := decodeMMRSignalling(noisy, false)
		s, e, _ := decodeMMRSignalling(noisy, true)
		hard += len(h)
		soft += len(s)
		erased += e
		for _, pdu := range s {
			if !known[string(pdu)] {
				t.Fatalf("noise draw %d: soft decode produced a PDU the clean capture never carries: % x", seed, pdu)
			}
		}
	}
	t.Logf("-1 dB: hard %d, soft %d (%d by the erasure retry)", hard, soft, erased)
	if soft < hard+hard/5 || erased == 0 {
		t.Fatalf("-1 dB: soft decoded %d signalling bursts (%d by the retry) against the hard decode's %d; want at least 20 %% more",
			soft, erased, hard)
	}
}
