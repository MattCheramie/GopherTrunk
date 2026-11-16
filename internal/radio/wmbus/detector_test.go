package wmbus

import (
	"math"
	"math/rand"
	"testing"
)

// noisyCModeYield sends n C mode telegrams of random content, 50 kHz off
// centre as the issue #1256 tuner hears them, through one receiver at
// 1.6 MS/s, and returns how many came out CRC-valid with the content sent.
// snrDB is the signal-to-noise ratio in the 200 kHz the FSK occupies.
func noisyCModeYield(t *testing.T, n int, snrDB float64) int {
	t.Helper()
	const rate = 1_600_000
	// Synthesize's signal amplitude is 0.5 and its noise is per axis, so
	// the SNR in 200 kHz of the 1.6 MHz band is 0.25 / (2σ²) × 8.
	sigma := math.Sqrt(1 / math.Pow(10, snrDB/10))
	rng := rand.New(rand.NewSource(1256))
	sent := map[string]bool{}
	got := 0
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
		if f.CRCOK && !info.Inverted && sent[string(f.Data)] {
			sent[string(f.Data)] = false
			got++
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		data := make([]byte, 100)
		rng.Read(data)
		data[0] = byte(len(data) - 1)
		sent[string(data)] = true
		rx.Process(Synthesize(data, SynthOptions{
			Mode: ModeC1, SampleRateHz: rate, OffsetHz: -50_000, DeviationHz: 45_000,
			NoiseRMS: sigma, Seed: int64(i + 1),
		}))
	}
	rx.Flush()
	return got
}

// TestReceiverDecodesWeakTelegrams pins the chip detector's sensitivity
// (issue #1256). The FM discriminator receiver it replaced decoded 0 of 40
// at both of these levels (and 33 of 40 at 8 dB); the tone correlators
// measured 27 and 37.
func TestReceiverDecodesWeakTelegrams(t *testing.T) {
	for _, tc := range []struct {
		snrDB float64
		want  int
	}{
		{6, 20},
		{7, 32},
	} {
		got := noisyCModeYield(t, 40, tc.snrDB)
		t.Logf("%.0f dB in the channel: %d of 40 telegrams CRC-valid", tc.snrDB, got)
		if got < tc.want {
			t.Errorf("%.0f dB in the channel: %d of 40 telegrams CRC-valid, want at least %d", tc.snrDB, got, tc.want)
		}
	}
}

// TestReceiverRealAirTelegramsDiscriminatorMissed: three telegrams from the
// issue #1256 reporter's 8 Nov walk (S5 + RTL-SDR, the dongle 50 kHz low),
// each decoded by a fresh receiver with no bit repaired. The discriminator
// receiver failed the CRC of all three. Their signal-to-noise ratio in the
// channel is 11, 15 and 24 dB (that tuner's noise floor is about -44 dBFS
// in 200 kHz, so the levels are -34, -30 and -18 dBFS): the walk's
// "strong telegrams that still fail" were mostly not strong against the
// noise. Every one of these meters decodes elsewhere on the walk too.
func TestReceiverRealAirTelegramsDiscriminatorMissed(t *testing.T) {
	for _, tc := range []struct{ file, mfr, id string }{
		{"testdata/issue1256_eie_02816799_snr11db.wav", "EIE", "02816799"},
		{"testdata/issue1256_eie_02828183_snr15db.wav", "EIE", "02828183"},
		{"testdata/issue1256_qds_95951829_snr24db.wav", "QDS", "95951829"},
	} {
		iq, rate := loadIQWav(t, tc.file)
		type got struct {
			mfr, id   string
			corrected int
		}
		var ok []got
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
			if !f.CRCOK || info.Inverted {
				return
			}
			tg, err := ParseTelegram(f)
			if err != nil {
				t.Errorf("%s: %v", tc.file, err)
				return
			}
			ok = append(ok, got{tg.Manufacturer, tg.ID, f.CorrectedBits})
		}})
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < len(iq); j += 16384 {
			rx.Process(iq[j:min(j+16384, len(iq))])
		}
		rx.Flush()
		if len(ok) != 1 || ok[0] != (got{tc.mfr, tc.id, 0}) {
			t.Errorf("%s: CRC-valid telegrams %+v, want %s %s with no repair", tc.file, ok, tc.mfr, tc.id)
		}
	}
}
