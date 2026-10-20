package composer

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// nxdnRealAirIQ loads one of the #1261 reporter's 48 kHz cs16 slices.
func nxdnRealAirIQ(t *testing.T, file string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		iq[i] = complex(float32(int16(binary.LittleEndian.Uint16(raw[4*i:])))/32768,
			float32(int16(binary.LittleEndian.Uint16(raw[4*i+2:])))/32768)
	}
	return iq
}

// shiftIQ moves iq up by hz at sample rate fs.
func shiftIQ(iq []complex64, hz, fs float64) []complex64 {
	out := make([]complex64, len(iq))
	for i := range iq {
		ph := 2 * math.Pi * hz * float64(i) / fs
		out[i] = iq[i] * complex64(complex(math.Cos(ph), math.Sin(ph)))
	}
	return out
}

func nxdnVoiceFrames(in []complex64, symbolRate float64) (int, *nxdnrx.Receiver) {
	var n int
	trf := nxdn.NewTrafficChannel(func(f []nxdn.VCHFrame) { n += len(f) })
	rx := newNXDNVoiceReceiver(nxdnVoiceIntermediateHz, symbolRate, func(d []uint8, b int) { trf.Process(d, b) })
	for i := 0; i < len(in); i += 4096 {
		rx.Process(in[i:min(i+4096, len(in))])
	}
	return n, rx
}

// TestNXDNVoiceReceiverToleratesTunerDrift pins #1261's "decoding works for
// 5-10 minutes, then fails until I change ppm by 1 and restart". The voice
// receiver removed the carrier offset only AFTER its timing loop, which still
// saw the raw DC bias: on the reporter's NXDN48 capture (already +401 Hz
// off) decoding collapsed for an absolute offset of about 750-1250 Hz either
// side (0-12 of 56 voice frames) and came back further out — a band that a
// dongle 1-2 ppm off at 447 MHz drifts into as it warms. The carrier tracker
// de-rotates the IQ ahead of the filter, discriminator and timing loop.
func TestNXDNVoiceReceiverToleratesTunerDrift(t *testing.T) {
	const native = 401 // the capture's own offset, measured from its discriminator mean
	iq := nxdnRealAirIQ(t, "../../radio/nxdn/testdata/nxdn48_rdch_voice_447100_16oct_48k.cs16")
	for _, abs := range []float64{-1250, -1000, -750, 750, 1000, 1250} {
		n, rx := nxdnVoiceFrames(shiftIQ(iq, abs-native, nxdnVoiceIntermediateHz), nxdnrx.SymbolRateNXDN48)
		t.Logf("offset %+5.0f Hz: %d voice frames, tracked %.0f Hz", abs, n, rx.CarrierOffsetHz())
		if n < 48 {
			t.Errorf("offset %+.0f Hz: %d voice frames, want ≥ 48 of the capture's 56", abs, n)
		}
		if math.Abs(rx.CarrierOffsetHz()-abs) > 150 {
			t.Errorf("offset %+.0f Hz: tracker settled at %.0f Hz", abs, rx.CarrierOffsetHz())
		}
	}
}

// TestNXDNVoiceCarrierTrackerHoldsOnNoise: a scan-list dwell can hand the
// chain receiver noise (before the first frame, between overs). The tracker
// must not walk off the channel on it.
func TestNXDNVoiceCarrierTrackerHoldsOnNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	noise := make([]complex64, 5*nxdnVoiceIntermediateHz)
	for i := range noise {
		noise[i] = complex(float32(rng.NormFloat64()*0.1), float32(rng.NormFloat64()*0.1))
	}
	for _, sr := range []float64{nxdnrx.SymbolRate, nxdnrx.SymbolRateNXDN48} {
		_, rx := nxdnVoiceFrames(noise, sr)
		if off := rx.CarrierOffsetHz(); math.Abs(off) > 300 {
			t.Errorf("symbol rate %.0f: tracker at %.0f Hz after 5 s of noise", sr, off)
		}
	}
}
