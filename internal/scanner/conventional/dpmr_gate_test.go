package conventional

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
)

// The dPMR call gate (issue #1374). No real dPMR capture exists yet, so the
// transmissions here are built in the air format of
// internal/radio/dpmr/airframe.go (pinned to ETSI TS 102 658 and to DSD-FME's
// own decode of GopherTrunk's control channels) and modulated as the standard
// specifies: 4FSK at 2400 symbols a second, +-1050 Hz at the outer symbols,
// root raised cosine 0.2. The rejection tests use real air where it exists:
// the reporter's NXDN48 capture is the same physical layer and must not open
// a dpmr channel.

// dpmrGateCall returns the dibits of one dPMR voice transmission: the
// preamble, the header frame, superframes of voice and the end frame.
func dpmrGateCall(r *rand.Rand, superframes int) []uint8 {
	var d []uint8
	random := func(n int) {
		for i := 0; i < n; i++ {
			d = append(d, uint8(r.Intn(4)))
		}
	}
	code := []uint8{1, 1, 1, 3, 1, 1, 3, 3, 1, 3, 1, 3} // channel code 57 5F 77
	for i := 0; i < 18; i++ {                           // preamble: 5F 5F ...
		d = append(d, 1, 1, 3, 3)
	}
	d = append(d, dpmr.AirFS1Dibits()...)
	random(60)
	d = append(d, code...)
	random(60)
	half := [4]uint16{0x123, 0x456, 0xABC, 0xDEF}
	for s := 0; s < superframes; s++ {
		for f := 0; f < 4; f++ {
			if f%2 == 0 {
				d = append(d, dpmr.AirFS2Dibits()...)
			} else {
				d = append(d, code...)
			}
			d = append(d, dpmr.EncodeAirCCH(dpmr.AirCCH{
				FrameNumber: uint8(f), IDHalf: half[f], Mode: dpmr.AirModeVoice, Format: 1,
			})...)
			for v := 0; v < 4; v++ {
				info := make([]byte, 49)
				for i := range info {
					info[i] = byte(r.Intn(2))
				}
				bits, err := dpmr.EncodeTCHFrame(info)
				if err != nil {
					panic(err)
				}
				for i := 0; i < 36; i++ {
					d = append(d, bits[2*i]<<1|bits[2*i+1])
				}
			}
		}
	}
	d = append(d, dpmr.AirFS3Dibits()...)
	random(36)
	return d
}

// dpmrGateAir modulates dibits at 48 kHz, offHz off frequency, at amplitude
// 0.5 with the noise floor noiseIQ uses.
func dpmrGateAir(r *rand.Rand, dibits []uint8, offHz float64) []complex64 {
	iq := demod.ModulateC4FM(dibits, 20, 8, 0.20, dmrGateTestRate, 1050)
	out := make([]complex64, len(iq))
	for i := range iq {
		p := 2 * math.Pi * offHz * float64(i) / dmrGateTestRate
		v := 0.5 * complex128(iq[i]) * complex(math.Cos(p), math.Sin(p))
		v += complex(r.NormFloat64()*0.01, r.NormFloat64()*0.01)
		out[i] = complex64(v)
	}
	return out
}

// dpmrCallIQ is noise, one dPMR call of the given length, noise.
func dpmrCallIQ(seed int64, superframes int, offHz float64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	iq := noiseIQ(r, 0.3)
	iq = append(iq, dpmrGateAir(r, dpmrGateCall(r, superframes), offHz)...)
	return append(iq, noiseIQ(r, 1)...)
}

// dpmrGateFirstOpen feeds iq in 10 ms chunks and returns when the gate first
// opened (-1 = never) and whether it was open at the end.
func dpmrGateFirstOpen(d *DPMRCallDetector, iq []complex64, rate float64) (first time.Duration, last bool) {
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

// A dPMR call opens the gate within its scan window, at the scanner's own
// 2.4 MS/s, on frequency and a dongle's typical error off it, and joined at
// several points of the transmission. The window is measured from the start
// of the IQ, so a late join still has to lock within it.
func TestDPMRCallDetectorOpensOnACall(t *testing.T) {
	for _, offHz := range []float64{0, 400, -900} {
		for join := 0; join < 4; join++ {
			call := dpmrCallIQ(61, 8, offHz)
			// Skip the lead-in noise and a further 70 ms per step into the call.
			iq := dmrAt2400k(call[int(0.3*dmrGateTestRate)+join*3360:])
			d := NewDPMRCallDetector(2_400_000)
			first, last := dpmrGateFirstOpen(d, iq, 2_400_000)
			if first < 0 || first > d.minDwell() {
				t.Errorf("%+.0f Hz, joined %d ms in: gate opened after %v, want within the %v scan window (%d frames)",
					offHz, join*70, first, d.minDwell(), d.Frames())
			}
			t.Logf("%+5.0f Hz, joined %3d ms in: open after %v", offHz, join*70, first)
			if last {
				t.Errorf("%+.0f Hz, joined %d ms in: gate still open a second after the end frame", offHz, join*70)
			}
			if d.EndFrames() != 1 {
				t.Errorf("%+.0f Hz, joined %d ms in: %d end frames, want 1", offHz, join*70, d.EndFrames())
			}
		}
	}
}

// The end frame closes the gate at once, not a hold later: the scanner's
// hangtime starts when the radio releases the channel.
func TestDPMRCallDetectorClosesOnTheEndFrame(t *testing.T) {
	call := dpmrCallIQ(62, 4, 0)
	d := NewDPMRCallDetector(dmrGateTestRate)
	if _, last := dpmrGateFirstOpen(d, call, dmrGateTestRate); last {
		t.Fatalf("gate open after the end frame (%d frames, %d end frames)", d.Frames(), d.EndFrames())
	}
	if d.Frames() == 0 || d.EndFrames() != 1 {
		t.Fatalf("%d frames, %d end frames: the call was not framed", d.Frames(), d.EndFrames())
	}
	d.Reset()
	if d.Present() || d.Frames() != 0 || d.EndFrames() != 0 {
		t.Fatal("Reset left the gate's state behind")
	}
}

// NXDN48 is dPMR's physical layer exactly (4FSK, 2400 symbols a second, the
// same deviation and filter), so the reporter's real NXDN48 call is the
// hardest real-air carrier a dpmr channel will meet. It has the wrong sync
// words and must not open the gate; nor may the NXDN96 voice and control
// channel captures, the analog-tone capture, synthetic FM carriers or noise.
func TestDPMRCallDetectorIgnoresOtherCarriers(t *testing.T) {
	for _, file := range []string{nxdn48VoiceFixture, nxdn96VoiceFixture, nxdnCCFixture, analogToneFixture} {
		d := NewDPMRCallDetector(dmrGateTestRate)
		if first, _ := dpmrGateFirstOpen(d, readCS16Fixture(t, file), dmrGateTestRate); first >= 0 {
			t.Errorf("%s: gate opened after %v (%d frames)", file, first, d.Frames())
		}
	}
	for _, tc := range []struct {
		name               string
		offHz, tone, devHz float64
	}{
		{"1 kHz tone, ±2.5 kHz", -1400, 1000, 2500},
		{"unmodulated carrier", 300, 0, 0},
		{"300 Hz tone, ±4 kHz", 0, 300, 4000},
	} {
		d := NewDPMRCallDetector(dmrGateTestRate)
		rng := rand.New(rand.NewSource(63))
		if first, _ := dpmrGateFirstOpen(d, fmToneCarrier(rng, 5, tc.offHz, tc.tone, tc.devHz), dmrGateTestRate); first >= 0 {
			t.Errorf("%s: gate opened after %v", tc.name, first)
		}
	}
	d := NewDPMRCallDetector(dmrGateTestRate)
	if first, _ := dpmrGateFirstOpen(d, noiseIQ(rand.New(rand.NewSource(64)), 60), dmrGateTestRate); first >= 0 {
		t.Errorf("gate opened on 60 s of noise after %v", first)
	}
}

// Random 4FSK at dPMR's rate and deviation, which carries no sync word
// structure at all, opens nothing either: a lone FS2 match must be
// corroborated by the frames behind it.
func TestDPMRCallDetectorIgnoresRandom4FSK(t *testing.T) {
	r := rand.New(rand.NewSource(65))
	dibits := make([]uint8, 2400*30)
	for i := range dibits {
		dibits[i] = uint8(r.Intn(4))
	}
	d := NewDPMRCallDetector(dmrGateTestRate)
	if first, _ := dpmrGateFirstOpen(d, dpmrGateAir(r, dibits, 0), dmrGateTestRate); first >= 0 {
		t.Fatalf("gate opened on 30 s of random 4FSK after %v (%d frames)", first, d.Frames())
	}
}

// A dPMR call opens the gate from every sub-symbol timing phase. Without the
// receiver's feed-forward timing acquisition, the call started at twenty
// phases of a 48 kHz stream never framed from about one in four: the timing
// loop sat at its unstable half-symbol point for the whole transmission. The
// call follows noise, as it does when a radio keys up while the scanner
// dwells, so the acquisition has to be re-armed after giving up on the noise.
func TestDPMRCallDetectorOpensAtEveryTimingPhase(t *testing.T) {
	call := dpmrCallIQ(66, 6, 300)
	for delay := 0; delay < 20; delay++ { // 20 samples = one symbol at 48 kHz
		iq := append(make([]complex64, delay), call...)
		d := NewDPMRCallDetector(dmrGateTestRate)
		first, _ := dpmrGateFirstOpen(d, iq, dmrGateTestRate)
		// Measured from the end of the 0.3 s of lead-in noise.
		if first < 0 || first-300*time.Millisecond > d.minDwell() {
			t.Errorf("start %d samples later: gate opened after %v, want within %v of the call's start (%d frames)",
				delay, first, d.minDwell(), d.Frames())
		}
	}
}
