package conventional

import (
	"math/rand"
	"testing"
)

// Issue #1261: an nxdn / nxdn48 scan-list channel opened a call on ANY
// carrier above its power squelch, as ysf channels did until the YSF gate.
// These run the whole scanner on one channel at the SDR's 2.4 MS/s.

// An analog FM carrier on an nxdn channel opens no call.
func TestScannerNXDNAnalogCarrierOpensNoCall(t *testing.T) {
	for _, mode := range []string{ModeNXDN, ModeNXDN48} {
		rng := rand.New(rand.NewSource(51))
		iq := dmrAt2400k(fmToneCarrier(rng, 3, -1400, 1000, 2500))
		for i := range iq {
			iq[i] *= 0.4
		}
		if eng := runModeChannel(t, mode, iq); eng.startCount() != 0 {
			t.Errorf("%s: an analog FM tone opened %d call(s); the scanner must move on", mode, eng.startCount())
		}
	}
}

// An NXDN control channel (the reporter's real 453.775 MHz RCCH) opens no
// call: it is NXDN, but nobody is talking on it.
func TestScannerNXDNControlChannelOpensNoCall(t *testing.T) {
	iq := dmrAt2400k(readCS16Fixture(t, nxdnCCFixture))
	if eng := runModeChannel(t, ModeNXDN, iq); eng.startCount() != 0 {
		t.Fatalf("an NXDN control channel opened %d call(s)", eng.startCount())
	}
}

// The gate must not cost a real call: the reporter's NXDN96 and NXDN48
// voice captures each still open one.
func TestScannerNXDNRealAirCallsStillOpen(t *testing.T) {
	for _, tc := range []struct{ mode, file string }{
		{ModeNXDN, nxdn96VoiceFixture},
		{ModeNXDN48, nxdn48VoiceFixture},
	} {
		iq := dmrAt2400k(readCS16Fixture(t, tc.file))
		if eng := runModeChannel(t, tc.mode, iq); eng.startCount() != 1 {
			t.Errorf("%s: the reporter's call opened %d call(s), want 1", tc.mode, eng.startCount())
		}
	}
}
