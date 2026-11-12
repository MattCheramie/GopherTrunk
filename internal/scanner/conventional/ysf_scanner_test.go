package conventional

import (
	"math/rand"
	"testing"
)

// TestScannerYSFAnalogCarrierOpensNoCall is the issue #1261 field report of
// 11 Nov: a `ysf` scan-list channel opened a call on a carrier that was an
// analog FM 1 kHz test tone, and the chain recorded nothing ("No actual
// sound was detected"). On power alone any carrier opened a "ysf" call;
// with the frame-sync gate it must not.
func TestScannerYSFAnalogCarrierOpensNoCall(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	iq := dmrAt2400k(fmToneCarrier(rng, 3, -1400, 1000, 2500))
	for i := range iq {
		iq[i] *= 0.4
	}
	if eng := runModeChannel(t, ModeYSF, iq); eng.startCount() != 0 {
		t.Fatalf("an analog FM tone opened %d ysf call(s); the scanner must move on", eng.startCount())
	}
}

// TestScannerYSFCallStillOpens: the gate must not cost a real call.
func TestScannerYSFCallStillOpens(t *testing.T) {
	rng := rand.New(rand.NewSource(32))
	iq := dmrAt2400k(modulateYSF(rng, ysfCallDibits(t, rng, 20)))
	for i := range iq {
		iq[i] *= 0.4
	}
	if eng := runModeChannel(t, ModeYSF, iq); eng.startCount() != 1 {
		t.Fatalf("a System Fusion call opened %d call(s), want 1", eng.startCount())
	}
}
