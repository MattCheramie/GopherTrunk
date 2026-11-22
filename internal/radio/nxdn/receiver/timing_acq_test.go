package receiver

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// The timing acquisition (issue #1374) is off unless armed, seeds the loop
// once on a 4FSK carrier, and gives up on noise so a caller can re-arm it.
func TestTimingAcquisitionArmSeedAndGiveUp(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	dibits := make([]uint8, 2400)
	for i := range dibits {
		dibits[i] = uint8(r.Intn(4))
	}
	sig := demod.ModulateC4FM(dibits, 20, 8, 0.20, 48_000, 1050)
	noise := make([]complex64, 48_000)
	for i := range noise {
		noise[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
	}
	newRx := func() *Receiver {
		return New(Options{SampleRateHz: 48_000, SymbolRateHz: SymbolRateNXDN48, DeviationHz: 1050,
			DibitSink: func([]uint8, int) {}})
	}

	rx := newRx()
	rx.Process(sig)
	if rx.TimingSeeds() != 0 || rx.TimingAcquisitionPending() {
		t.Fatalf("unarmed: %d seeds, pending %v", rx.TimingSeeds(), rx.TimingAcquisitionPending())
	}

	rx = newRx()
	rx.ArmTimingAcquisition()
	rx.Process(sig)
	if rx.TimingSeeds() != 1 || rx.TimingAcquisitionPending() {
		t.Fatalf("armed on a carrier: %d seeds, pending %v; want one seed", rx.TimingSeeds(), rx.TimingAcquisitionPending())
	}

	rx = newRx()
	rx.ArmTimingAcquisition()
	rx.Process(noise)
	if rx.TimingAcquisitionPending() {
		t.Fatal("armed on a second of noise: still pending, so a caller could never re-arm it")
	}
	rx.ArmTimingAcquisition()
	rx.Process(sig)
	if rx.TimingSeeds() != 1 {
		t.Fatalf("re-armed after noise: %d seeds on the carrier, want 1", rx.TimingSeeds())
	}
}
