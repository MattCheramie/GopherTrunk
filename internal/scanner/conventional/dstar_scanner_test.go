package conventional

import (
	"math/rand"
	"testing"
)

// TestScannerDStarAnalogCarrierOpensNoCall: a dstar channel used to open a
// call on any carrier (issue #1261); with the gate an analog FM tone is
// skipped.
func TestScannerDStarAnalogCarrierOpensNoCall(t *testing.T) {
	rng := rand.New(rand.NewSource(71))
	iq := dmrAt2400k(fmToneCarrier(rng, 3, -1400, 1000, 2500))
	for i := range iq {
		iq[i] *= 0.4
	}
	if eng := runModeChannel(t, ModeDStar, iq); eng.startCount() != 0 {
		t.Fatalf("an analog FM tone opened %d dstar call(s); the scanner must move on", eng.startCount())
	}
}

// TestScannerDStarCallStillOpens: the gate must not cost a real call, joined
// mid-transmission (no header heard) or from its start.
func TestScannerDStarCallStillOpens(t *testing.T) {
	for _, withHeader := range []bool{false, true} {
		rng := rand.New(rand.NewSource(72))
		iq := dmrAt2400k(modulateDStar(rng, dstarTransmissionBits(rng, 150, withHeader), false))
		for i := range iq {
			iq[i] *= 0.4
		}
		if eng := runModeChannel(t, ModeDStar, iq); eng.startCount() != 1 {
			t.Fatalf("header=%v: a D-STAR call opened %d call(s), want 1", withHeader, eng.startCount())
		}
	}
}
