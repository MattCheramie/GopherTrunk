package composer

import (
	"math"
	"testing"
)

// fmToneIQ is 2.4 MS/s IQ of a constant-envelope carrier frequency-modulated
// by a 1 kHz tone at devHz peak deviation.
func fmToneIQ(n int, devHz float64) []complex64 {
	const rate = 2_400_000.0
	out := make([]complex64, n)
	for i := range out {
		t := float64(i) / rate
		ph := devHz / 1000 * math.Sin(2*math.Pi*1000*t)
		out[i] = complex(float32(0.2*math.Cos(ph)), float32(0.2*math.Sin(ph)))
	}
	return out
}

// TestComposerFMChainAppliesAudioGain pins Options.AudioGainDB: the analog FM
// chain scales its demodulated audio by the configured gain before it reaches
// any sink (recorder, live stream, host player), the default (0) is unity so
// existing callers are byte-for-byte unchanged, and the PCM clamps instead of
// wrapping when the gain would overflow 16 bits.
func TestComposerFMChainAppliesAudioGain(t *testing.T) {
	iq := fmToneIQ(2_400_000, 3000)

	unity := toneAmp(recordAnalog(t, "fm-conv", iq), 1000)
	boosted := toneAmp(recordAnalogOpts(t, "fm-conv", iq, func(o *Options) { o.AudioGainDB = 10 }), 1000)
	t.Logf("1 kHz tone: unity %.0f, +10 dB %.0f", unity, boosted)

	if unity < 500 {
		t.Fatalf("fixture: unity tone at %.0f, the FM chain recovered nothing", unity)
	}
	want := math.Pow(10, 10.0/20)
	if r := boosted / unity; r < want*0.95 || r > want*1.05 {
		t.Errorf("+10 dB gain scaled the tone %.2fx, want %.2fx", r, want)
	}

	// +60 dB cannot fit in int16: the output must saturate, not wrap.
	pcm := recordAnalogOpts(t, "fm-conv", iq, func(o *Options) { o.AudioGainDB = 60 })
	var peak int16
	for _, v := range pcm {
		if v > peak {
			peak = v
		}
	}
	if peak != math.MaxInt16 {
		t.Errorf("+60 dB peak = %d, want saturation at %d", peak, math.MaxInt16)
	}
}
