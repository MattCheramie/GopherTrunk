package conventional

import (
	"encoding/binary"
	"math/rand"
	"os"
	"testing"
	"time"

	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// Real-air fixtures for the NXDN call gate (issue #1261), shared with the
// nxdn package's own tests: 48 kS/s slices of the reporter's Kenwood
// captures. Two conventional voice calls on 447.1 MHz (NXDN96 and NXDN48,
// ~400 Hz off as captured) and an idle NXDN96 control channel on 453.775 MHz.
const (
	nxdn96VoiceFixture = "../../radio/nxdn/testdata/nxdn96_rdch_voice_447100_15oct_48k.cs16"
	nxdn48VoiceFixture = "../../radio/nxdn/testdata/nxdn48_rdch_voice_447100_16oct_48k.cs16"
	nxdnCCFixture      = "../../radio/nxdn/testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16"
	analogToneFixture  = "testdata/ysf_issue1261_analog_tone_448900k_48k.cs16"
)

func readCS16Fixture(t *testing.T, path string) []complex64 {
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

// nxdnGateFirstOpen feeds iq in 10 ms chunks and returns when the gate first
// opened (-1 = never) and whether it was open at the end.
func nxdnGateFirstOpen(d *NXDNCallDetector, iq []complex64, rate float64) (first time.Duration, last bool) {
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

// A real NXDN call opens the gate within the mode's scan window, at the
// scanner's own SDR rate, from any starting point in the transmission.
func TestNXDNCallDetectorOpensOnRealAirVoice(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		symRate    float64
	}{
		{"NXDN96", nxdn96VoiceFixture, nxdnrx.SymbolRate},
		{"NXDN48", nxdn48VoiceFixture, nxdnrx.SymbolRateNXDN48},
	} {
		base := readCS16Fixture(t, tc.file)
		for off := 0; off < 5; off++ {
			// Start 20 ms further in each time; only the scan window
			// (+100 ms) is upsampled and fed, since a later open fails anyway.
			d := NewNXDNCallDetector(2_400_000, tc.symRate)
			n := int((d.minDwell() + 100*time.Millisecond).Seconds() * dmrGateTestRate)
			iq := dmrAt2400k(base[off*960 : min(off*960+n, len(base))])
			first, _ := nxdnGateFirstOpen(d, iq, 2_400_000)
			if first < 0 || first > d.minDwell() {
				t.Errorf("%s offset %d ms: gate opened after %v, want within the %v scan window (%d frames)",
					tc.name, off*20, first, d.minDwell(), d.Frames())
			}
		}
	}
}

// An NXDN CONTROL channel is NXDN, but not a call: the gate must not open on
// it, or a scan list that holds a trunked site's control frequency would sit
// on it recording nothing.
func TestNXDNCallDetectorIgnoresRealAirControlChannel(t *testing.T) {
	iq := dmrAt2400k(readCS16Fixture(t, nxdnCCFixture))
	d := NewNXDNCallDetector(2_400_000, nxdnrx.SymbolRate)
	if first, _ := nxdnGateFirstOpen(d, iq, 2_400_000); first >= 0 {
		t.Fatalf("NXDN gate opened on a control channel after %v (%d frames)", first, d.Frames())
	}
}

// Analog carriers are not NXDN: the reporter's own analog-tone capture and
// synthetic FM carriers, for both NXDN modes.
func TestNXDNCallDetectorIgnoresAnalogCarriers(t *testing.T) {
	for _, sym := range []float64{nxdnrx.SymbolRate, nxdnrx.SymbolRateNXDN48} {
		d := NewNXDNCallDetector(dmrGateTestRate, sym)
		if first, _ := nxdnGateFirstOpen(d, readCS16Fixture(t, analogToneFixture), dmrGateTestRate); first >= 0 {
			t.Errorf("sym %v: gate opened on the reporter's analog-tone capture after %v", sym, first)
		}
		for _, tc := range []struct {
			name               string
			offHz, tone, devHz float64
		}{
			{"1 kHz tone, ±2.5 kHz", -1400, 1000, 2500},
			{"unmodulated carrier", 300, 0, 0},
			{"300 Hz tone, ±4 kHz", 0, 300, 4000},
		} {
			rng := rand.New(rand.NewSource(41))
			d := NewNXDNCallDetector(dmrGateTestRate, sym)
			if first, _ := nxdnGateFirstOpen(d, fmToneCarrier(rng, 5, tc.offHz, tc.tone, tc.devHz), dmrGateTestRate); first >= 0 {
				t.Errorf("sym %v, %s: gate opened after %v", sym, tc.name, first)
			}
		}
	}
}

// Receiver noise alone never opens the gate: the frame-sync PAIR is what
// keeps chance matches out (a lone sync at the search tolerance turns up
// in noise every few seconds).
func TestNXDNCallDetectorIgnoresNoise(t *testing.T) {
	for _, sym := range []float64{nxdnrx.SymbolRate, nxdnrx.SymbolRateNXDN48} {
		rng := rand.New(rand.NewSource(42))
		d := NewNXDNCallDetector(dmrGateTestRate, sym)
		if first, _ := nxdnGateFirstOpen(d, noiseIQ(rng, 30), dmrGateTestRate); first >= 0 {
			t.Errorf("sym %v: gate opened on 30 s of noise after %v", sym, first)
		}
	}
}

// Once the transmission ends the gate closes within its hold.
func TestNXDNCallDetectorClosesAfterCall(t *testing.T) {
	rng := rand.New(rand.NewSource(43))
	iq := append(readCS16Fixture(t, nxdn96VoiceFixture), noiseIQ(rng, 1)...)
	d := NewNXDNCallDetector(dmrGateTestRate, nxdnrx.SymbolRate)
	first, last := nxdnGateFirstOpen(d, iq, dmrGateTestRate)
	if first < 0 {
		t.Fatal("gate never opened on the call")
	}
	if last {
		t.Fatal("gate still open a second after the call ended")
	}
	d.Reset()
	if d.Present() || d.Frames() != 0 {
		t.Fatal("Reset left the gate open")
	}
}
