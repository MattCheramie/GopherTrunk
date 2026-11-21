package composer

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
)

// dPMR voice from IQ: a transmission in the air format of
// internal/radio/dpmr/airframe.go, modulated as the standard specifies
// (4FSK, 2400 symbols a second, +-1050 Hz at the outer symbols, root raised
// cosine 0.2), received by the voice chain's receiver and framer.

// dpmrAirTransmission builds the dibits of one voice call: noise, the
// preamble and header frame, superframes, the end frame, noise. It returns
// the dibits and the vocoder payloads sent, in order.
func dpmrAirTransmission(r *rand.Rand, superframes int, silence bool) ([]uint8, [][]byte) {
	var d []uint8
	random := func(n int) {
		for i := 0; i < n; i++ {
			d = append(d, uint8(r.Intn(4)))
		}
	}
	code := []uint8{1, 1, 1, 3, 1, 1, 3, 3, 1, 3, 1, 3} // channel code 57 5F 77
	random(600)                                         // the receiver's loops settle on noise
	for i := 0; i < 18; i++ {                           // preamble: 5F 5F ...
		d = append(d, 1, 1, 3, 3)
	}
	d = append(d, dpmr.AirFS1Dibits()...)
	random(60)
	d = append(d, code...)
	random(60)

	var sent [][]byte
	// A voice call from 0xABCDEF to 0x123456.
	half := [4]uint16{0x123, 0x456, 0xABC, 0xDEF}
	silent := []byte("1111100000000001101010011001111110001100111000001")
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
					if silence {
						info[i] = silent[i] - '0'
					} else {
						info[i] = byte(r.Intn(2))
					}
				}
				bits, err := dpmr.EncodeTCHFrame(info)
				if err != nil {
					panic(err)
				}
				for i := 0; i < 36; i++ {
					d = append(d, bits[2*i]<<1|bits[2*i+1])
				}
				sent = append(sent, info)
			}
		}
	}
	d = append(d, dpmr.AirFS3Dibits()...)
	random(36)
	random(600)
	return d, sent
}

// dpmrAir modulates dibits at 48 kHz, puts the carrier offHz off frequency,
// adds a same-power dPMR neighbour one 6.25 kHz channel up when asked, and
// white noise snrDB below the signal.
func dpmrAir(r *rand.Rand, dibits []uint8, offHz float64, neighbour bool, snrDB float64) []complex64 {
	const fs = 48_000.0
	iq := demod.ModulateC4FM(dibits, 20, 8, 0.20, fs, 1050)
	var adj []complex64
	if neighbour {
		other := make([]uint8, len(dibits))
		for i := range other {
			other[i] = uint8(r.Intn(4))
		}
		adj = demod.ModulateC4FM(other, 20, 8, 0.20, fs, 1050)
	}
	sigma := math.Pow(10, -snrDB/20) / math.Sqrt2
	out := make([]complex64, len(iq))
	for i := range iq {
		p := 2 * math.Pi * offHz * float64(i) / fs
		v := complex128(iq[i]) * complex(math.Cos(p), math.Sin(p))
		if neighbour {
			q := 2 * math.Pi * (offHz + 6250) * float64(i) / fs
			v += complex128(adj[i]) * complex(math.Cos(q), math.Sin(q))
		}
		v += complex(r.NormFloat64()*sigma, r.NormFloat64()*sigma)
		out[i] = complex64(v)
	}
	return out
}

// dpmrReceive runs IQ through the voice chain's receiver and framer and
// returns how many of the vocoder frames sent came out bit-exact, and the
// framer's counters.
func dpmrReceive(iq []complex64, sent [][]byte, legacyReceiver bool) (int, dpmr.TrafficStats) {
	want := map[string]int{}
	for _, s := range sent {
		want[string(s)]++
	}
	exact := 0
	tc := dpmr.NewTrafficChannel(func(frames []dpmr.TCHFrame) {
		for _, f := range frames {
			if want[string(f.Payload)] > 0 {
				want[string(f.Payload)]--
				exact++
			}
		}
	})
	sink := func(d []uint8, base int) { tc.Process(d, base) }
	process := newDPMRVoiceReceiver(48_000, sink).Process
	if legacyReceiver {
		// What the chain ran before: no carrier correction, no channel
		// filter, slicer set for 900 Hz.
		process = dpmrrx.New(dpmrrx.Options{SampleRateHz: 48_000, DeviationHz: 900, DibitSink: sink}).Process
	}
	for i := 0; i < len(iq); i += 4096 {
		j := i + 4096
		if j > len(iq) {
			j = len(iq)
		}
		process(iq[i:j])
	}
	return exact, tc.Stats()
}

// TestDPMRVoiceChainReceivesATransmission: a clean dPMR voice call, on
// frequency, comes out whole with its addresses, channel code and end
// frame.
func TestDPMRVoiceChainReceivesATransmission(t *testing.T) {
	r := rand.New(rand.NewSource(21))
	dibits, sent := dpmrAirTransmission(r, 6, false)
	iq := dpmrAir(r, dibits, 0, false, 30)
	exact, st := dpmrReceive(iq, sent, false)
	if exact != len(sent) {
		t.Fatalf("%d of %d vocoder frames received", exact, len(sent))
	}
	// On frequency and alone, the receiver the chain used before decodes
	// it too: what it lacked shows in the next test.
	if old, _ := dpmrReceive(iq, sent, true); old < len(sent)-16 {
		t.Errorf("the dPMR receiver decoded %d of %d frames of a clean on-frequency call", old, len(sent))
	}
	if st.Locks != 1 || st.EndFrames != 1 || st.CCHValid != 24 {
		t.Errorf("locks %d end frames %d control channels %d/24", st.Locks, st.EndFrames, st.CCHValid)
	}
	if st.CalledID != 0x123456 || st.CallingID != 0xABCDEF || st.ChannelCode != 0x575F77 {
		t.Errorf("called %06X calling %06X channel code %06X", st.CalledID, st.CallingID, st.ChannelCode)
	}
}

// TestDPMRVoiceReceiverToleratesOffsetAndNeighbour: a dongle is routinely a
// few hundred hertz off, and dPMR's inner symbols are only 350 Hz from the
// carrier. The receiver the chain used had no carrier correction and no
// channel filter; it decodes nothing of a call 400 Hz off frequency with a
// neighbour one channel up.
func TestDPMRVoiceReceiverToleratesOffsetAndNeighbour(t *testing.T) {
	for _, offHz := range []float64{400, -400, 1200, -1500} {
		r := rand.New(rand.NewSource(22))
		dibits, sent := dpmrAirTransmission(r, 8, false)
		iq := dpmrAir(r, dibits, offHz, true, 25)

		exact, st := dpmrReceive(iq, sent, false)
		// The first superframe may go to the carrier tracker settling.
		if exact < len(sent)-16 {
			t.Errorf("%+.0f Hz: %d of %d vocoder frames received (locks %d, drops %d)",
				offHz, exact, len(sent), st.Locks, st.Drops)
		}
		old, _ := dpmrReceive(iq, sent, true)
		if old > len(sent)/10 {
			t.Errorf("%+.0f Hz: the receiver without carrier correction decoded %d of %d frames; the test no longer shows why it was replaced",
				offHz, old, len(sent))
		}
		t.Logf("%+5.0f Hz off, neighbour at +6.25 kHz: %d/%d vocoder frames (receiver without carrier correction: %d)",
			offHz, exact, len(sent), old)
	}
}

// TestDPMRVoiceReceiverHoldsThroughSilenceFrames: dPMR does not scramble
// its voice payload, and a radio fills the end of a superframe with the
// vocoder's silence frame, whose symbols do not average to zero. A carrier
// tracker that follows the symbol mean must not walk off on a long run of
// them.
func TestDPMRVoiceReceiverHoldsThroughSilenceFrames(t *testing.T) {
	r := rand.New(rand.NewSource(23))
	dibits, sent := dpmrAirTransmission(r, 25, true) // 8 s of silence frames
	exact, st := dpmrReceive(dpmrAir(r, dibits, 300, false, 25), sent, false)
	if exact < len(sent)-16 {
		t.Fatalf("%d of %d silence frames received (locks %d, drops %d)", exact, len(sent), st.Locks, st.Drops)
	}
}

// TestDPMRVoiceChainIgnoresOtherCarriers: noise, and an FM carrier with a
// tone on it, give the framer nothing to lock to.
func TestDPMRVoiceChainIgnoresOtherCarriers(t *testing.T) {
	const fs = 48_000.0
	r := rand.New(rand.NewSource(24))
	n := int(20 * fs)
	noise := make([]complex64, n)
	tone := make([]complex64, n)
	phase := 0.0
	for i := range noise {
		noise[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
		// 1 kHz tone at 2.5 kHz deviation.
		phase += 2 * math.Pi * 2500 * math.Sin(2*math.Pi*1000*float64(i)/fs) / fs
		tone[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	for name, iq := range map[string][]complex64{"noise": noise, "FM tone": tone} {
		_, st := dpmrReceive(iq, nil, false)
		if st.Locks != 0 || st.VoiceFrames != 0 {
			t.Errorf("%s: %d locks, %d vocoder frames", name, st.Locks, st.VoiceFrames)
		}
	}
}
