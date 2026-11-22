package conventional

import (
	"math/rand"
	"testing"
)

// Issue #1374: a `dpmr` scan-list channel is the only way an untrunked dPMR
// call (a dPMR446 handheld, say) reaches the composer's dPMR voice chain.
// These run the whole scanner on one channel at the SDR's 2.4 MS/s.

// A dPMR call opens exactly one call, as protocol "dpmr".
func TestScannerDPMRCallOpensOneCall(t *testing.T) {
	eng := runModeChannel(t, ModeDPMR, dmrAt2400k(dpmrCallIQ(71, 8, 400)))
	if eng.startCount() != 1 {
		t.Fatalf("a dPMR call opened %d call(s), want 1", eng.startCount())
	}
	eng.mu.Lock()
	defer eng.mu.Unlock()
	if p := eng.starts[0].Protocol; p != "dpmr" {
		t.Fatalf("protocol = %q, want dpmr (the composer picks its voice chain from it)", p)
	}
	if !eng.starts[0].Conventional {
		t.Fatal("the grant is not marked conventional")
	}
}

// An analog FM carrier, and the reporter's real NXDN48 call (dPMR's own
// physical layer), on a dpmr channel open no call.
func TestScannerDPMROtherCarriersOpenNoCall(t *testing.T) {
	rng := rand.New(rand.NewSource(72))
	tone := dmrAt2400k(fmToneCarrier(rng, 3, -1400, 1000, 2500))
	for i := range tone {
		tone[i] *= 0.4
	}
	for name, iq := range map[string][]complex64{
		"analog FM tone": tone,
		"NXDN48 voice":   dmrAt2400k(readCS16Fixture(t, nxdn48VoiceFixture)),
	} {
		if eng := runModeChannel(t, ModeDPMR, iq); eng.startCount() != 0 {
			t.Errorf("%s opened %d call(s) on a dpmr channel; the scanner must move on", name, eng.startCount())
		}
	}
}
