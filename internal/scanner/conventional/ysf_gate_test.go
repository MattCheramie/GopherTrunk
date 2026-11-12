package conventional

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
)

// Fixtures for the YSF call gate (issue #1261): a System Fusion transmission
// (header frame, V/D mode 2 voice frames, terminator) modulated as C4FM at
// 48 kHz / 10 sps, and the analog carriers the gate must ignore.

// On-air FICH vectors from MMDVMHost's encoder (the composer's
// ysf_voice_test.go has the same three): header, V/D mode 2, terminator.
const (
	ysfGateFICHHeader = "0037b4ad9d002d9f9071003fe1e3ac0025d16449000b81cc5f"
	ysfGateFICHVD2    = "3127cca4bacfbe90326f4b9d70efea4fd5207d0684c823065b"
	ysfGateFICHTerm   = "c397b4716241cd9ca0ea4c0fe395858a65d1e502c3cb8161af"
)

func ysfGateFrame(t *testing.T, fichHex string, rng *rand.Rand) []uint8 {
	t.Helper()
	b, err := hex.DecodeString(fichHex)
	if err != nil {
		t.Fatal(err)
	}
	out := append([]uint8(nil), ysf.FSWPattern...)
	for _, x := range b {
		for s := 6; s >= 0; s -= 2 {
			out = append(out, (x>>uint(s))&3)
		}
	}
	for i := 0; i < ysf.PayloadDibits; i++ {
		out = append(out, uint8(rng.Intn(4)))
	}
	return out
}

// ysfCallDibits is one transmission: a header, frames V/D mode 2 voice
// frames and a terminator.
func ysfCallDibits(t *testing.T, rng *rand.Rand, frames int) []uint8 {
	t.Helper()
	d := ysfGateFrame(t, ysfGateFICHHeader, rng)
	for i := 0; i < frames; i++ {
		d = append(d, ysfGateFrame(t, ysfGateFICHVD2, rng)...)
	}
	return append(d, ysfGateFrame(t, ysfGateFICHTerm, rng)...)
}

func modulateYSF(rng *rand.Rand, dibits []uint8) []complex64 {
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, dmrGateTestRate, ysfGateDeviationHz)
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

// fmToneCarrier is an analog FM carrier at offHz, modulated by a toneHz sine
// at devHz peak deviation (devHz 0 = unmodulated), for seconds at 48 kHz. The
// reporter's first "YSF" capture was exactly this: a 1 kHz tone at ±2.5 kHz
// on the channel, 1.4 kHz off.
func fmToneCarrier(rng *rand.Rand, seconds, offHz, toneHz, devHz float64) []complex64 {
	n := int(seconds * dmrGateTestRate)
	iq := make([]complex64, n)
	phase := 0.0
	for i := range iq {
		f := offHz + devHz*math.Sin(2*math.Pi*toneHz*float64(i)/dmrGateTestRate)
		phase += 2 * math.Pi * f / dmrGateTestRate
		iq[i] = complex(float32(0.5*math.Cos(phase)), float32(0.5*math.Sin(phase))) +
			complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

// noiseIQ is receiver noise between transmissions.
func noiseIQ(rng *rand.Rand, seconds float64) []complex64 {
	iq := make([]complex64, int(seconds*dmrGateTestRate))
	for i := range iq {
		iq[i] = complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

func ysfGateTrace(d *YSFCallDetector, iq []complex64) []bool {
	var out []bool
	for i := 0; i < len(iq); i += 1024 {
		out = append(out, d.Process(iq[i:min(i+1024, len(iq))]))
	}
	return out
}

// An analog carrier on the channel (a test tone, an unmodulated carrier, an
// NFM voice-like tone) is not a System Fusion transmission: the gate must
// never open on it.
func TestYSFCallDetectorIgnoresAnalogCarrier(t *testing.T) {
	for _, tc := range []struct {
		name               string
		offHz, tone, devHz float64
	}{
		{"1 kHz tone, ±2.5 kHz, 1.4 kHz off (the #1261 capture)", -1400, 1000, 2500},
		{"1 kHz tone, ±1.8 kHz, on frequency", 0, 1000, 1800},
		{"unmodulated carrier", 300, 0, 0},
		{"300 Hz tone, ±4 kHz", 0, 300, 4000},
	} {
		rng := rand.New(rand.NewSource(21))
		d := NewYSFCallDetector(dmrGateTestRate)
		if anyOpen(ysfGateTrace(d, fmToneCarrier(rng, 3, tc.offHz, tc.tone, tc.devHz))) {
			t.Errorf("%s: YSF gate opened (%d sync hits)", tc.name, d.SyncHits())
		}
	}
}

// The reporter's own on-air capture (issue #1261, 11 Nov): 448.900 MHz,
// 1 s channelized to 48 kHz. It is an FM carrier with a 1 kHz tone, every
// sample at the ADC rail, and no YSF frame sync anywhere in it. On power
// alone the scanner opened a "ysf" call on it; the gate must not.
func TestYSFCallDetectorIgnoresRealAirAnalogTone(t *testing.T) {
	raw, err := os.ReadFile("testdata/ysf_issue1261_analog_tone_448900k_48k.cs16")
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	d := NewYSFCallDetector(dmrGateTestRate)
	if anyOpen(ysfGateTrace(d, iq)) {
		t.Fatalf("YSF gate opened on the reporter's analog-tone capture (%d sync hits)", d.SyncHits())
	}
}

// A System Fusion transmission opens the gate for its whole length, and the
// gate closes again once the transmission is over.
func TestYSFCallDetectorOpensOnCallAndClosesAfter(t *testing.T) {
	rng := rand.New(rand.NewSource(22))
	lead := noiseIQ(rng, 0.5)
	call := modulateYSF(rng, ysfCallDibits(t, rng, 30)) // 3.2 s
	tail := noiseIQ(rng, 1.5)
	iq := append(append(append([]complex64{}, lead...), call...), tail...)
	d := NewYSFCallDetector(dmrGateTestRate)
	tr := ysfGateTrace(d, iq)

	chunkOf := func(sample int) int { return sample / 1024 }
	leadEnd, callEnd := chunkOf(len(lead)), chunkOf(len(lead)+len(call))
	if anyOpen(tr[:leadEnd]) {
		t.Error("gate open on noise before the call")
	}
	open := 0
	for _, v := range tr[leadEnd:callEnd] {
		if v {
			open++
		}
	}
	if frac := float64(open) / float64(callEnd-leadEnd); frac < 0.8 {
		t.Errorf("gate open for %.0f%% of the call, want >= 80%%", 100*frac)
	}
	// ysfCallHold (0.5 s) after the terminator, noise must not hold it.
	closeBy := callEnd + chunkOf(int(0.7*dmrGateTestRate))
	if anyOpen(tr[closeBy:]) {
		t.Error("gate still open on noise after the call ended")
	}
	if d.SyncHits() < 25 {
		t.Errorf("heard %d of 32 frame syncs", d.SyncHits())
	}
}

// The scanner tunes onto a carrier that is already up, at an arbitrary
// sub-symbol phase; a call must open the gate whatever the phase, within
// the gate's minimum dwell.
func TestYSFCallDetectorAcquiresAtAnySymbolPhase(t *testing.T) {
	rng := rand.New(rand.NewSource(23))
	base := modulateYSF(rng, ysfCallDibits(t, rng, 8)) // 1 s, no lead-in
	window := int(ysfGateMinDwell.Seconds() * dmrGateTestRate)
	for phase := 0; phase < 10; phase++ {
		iq := append(make([]complex64, phase), base...)
		d := NewYSFCallDetector(dmrGateTestRate)
		tr := ysfGateTrace(d, iq[:window])
		if !anyOpen(tr) {
			t.Errorf("symbol phase %d/10: a YSF call did not open the gate within %v (%d sync hits)", phase, ysfGateMinDwell, d.SyncHits())
		}
	}
}
