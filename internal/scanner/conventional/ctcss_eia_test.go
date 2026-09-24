package conventional

import (
	"math"
	"math/rand"
	"testing"
)

// genNoisyCTCSS is genFMModulatedTone plus complex white noise at snrDb
// (carrier-to-noise over the full sample rate), with a random start
// phase for the tone so no test depends on block alignment.
func genNoisyCTCSS(toneHz, devHz, sampleHz float64, n int, snrDb float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	sigma := math.Sqrt(math.Pow(10, -snrDb/10) / 2)
	out := make([]complex64, n)
	phase := 0.0
	t0 := rng.Float64()
	for i := range out {
		m := math.Sin(2*math.Pi*toneHz*float64(i)/sampleHz + 2*math.Pi*t0)
		phase += 2 * math.Pi * devHz * m / sampleHz
		out[i] = complex(
			float32(math.Cos(phase)+rng.NormFloat64()*sigma),
			float32(math.Sin(phase)+rng.NormFloat64()*sigma))
	}
	return out
}

// ctcssFraction feeds x in chunk-sized pieces and returns the fraction of
// chunks, after the first second's warm-up allowance of skip samples,
// that report the tone present.
func ctcssFraction(d *CTCSSDetector, x []complex64, chunk, skip int) float64 {
	var n, on int
	for s := 0; s+chunk <= len(x); s += chunk {
		got := d.Process(x[s : s+chunk])
		if s < skip {
			continue
		}
		n++
		if got {
			on++
		}
	}
	return float64(on) / float64(n)
}

// neighbours returns the EIA tones adjacent to EIACTCSSTones[i].
func eiaNeighbours(i int) []float64 {
	var out []float64
	if i > 0 {
		out = append(out, EIACTCSSTones[i-1])
	}
	if i+1 < len(EIACTCSSTones) {
		out = append(out, EIACTCSSTones[i+1])
	}
	return out
}

// TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation is the #1184 NFM
// regression: "with narrow FM TX the squelch does not open for any
// configured tone". An NFM radio puts ~300-400 Hz of deviation into its
// CTCSS tone (15% of 2.5 kHz); the old 5e-4 magnitude threshold needed
// ~540 Hz, so only wide-FM radios ever opened the gate.
func TestCTCSSOpensOnEveryEIAToneAtNarrowbandDeviation(t *testing.T) {
	const rate = 48_000
	for i, hz := range EIACTCSSTones {
		if hz == 150.0 || hz == 151.4 {
			continue // 1.4 Hz apart — see TestCTCSSCloseTonePair
		}
		d := NewCTCSSDetector(CTCSSConfig{SampleHz: rate, TargetHz: hz})
		x := genNoisyCTCSS(hz, 350, rate, 2*rate, 20, int64(i))
		if f := ctcssFraction(d, x, 4096, rate/2); f < 0.9 {
			t.Errorf("%.1f Hz at 350 Hz deviation: present in %.0f%% of chunks, want >= 90%%", hz, f*100)
		}
	}
}

// TestCTCSSRejectsAdjacentEIATones: a transmission on the next tone up or
// down the EIA table must never open the gate, even at full wideband
// deviation.
func TestCTCSSRejectsAdjacentEIATones(t *testing.T) {
	const rate = 48_000
	for i, hz := range EIACTCSSTones {
		for j, other := range eiaNeighbours(i) {
			if math.Abs(other-hz) < 2 {
				continue // 150.0 / 151.4 — see TestCTCSSCloseTonePair
			}
			d := NewCTCSSDetector(CTCSSConfig{SampleHz: rate, TargetHz: hz})
			x := genNoisyCTCSS(other, 750, rate, 2*rate, 20, int64(100*i+j))
			if f := ctcssFraction(d, x, 4096, 0); f > 0 {
				t.Errorf("%.1f Hz gate opened in %.0f%% of chunks on a %.1f Hz transmission", hz, f*100, other)
			}
		}
	}
}

// TestCTCSSReportedToneOffsetAtSDRRate reproduces the #1184 report at the
// production rate: 162.2 Hz and 192.8 Hz configured did not open on
// their own tone, but did open when the radio sent the next tone DOWN
// (159.8 / 189.9 Hz). The Goertzel bin was rounded to the 5 Hz grid, so
// the "162.2 Hz" bin was really 160.0 Hz.
func TestCTCSSReportedToneOffsetAtSDRRate(t *testing.T) {
	const rate = 2_400_000
	cases := []struct {
		target, sent float64
		dev          float64
		wantOpen     bool
	}{
		{162.2, 162.2, 350, true},
		{162.2, 162.2, 750, true},
		{192.8, 192.8, 350, true},
		{192.8, 192.8, 750, true},
		{162.2, 159.8, 750, false},
		{192.8, 189.9, 750, false},
	}
	for i, c := range cases {
		d := NewCTCSSDetector(CTCSSConfig{SampleHz: rate, TargetHz: c.target})
		x := genNoisyCTCSS(c.sent, c.dev, rate, rate*3/2, 30, int64(i))
		f := ctcssFraction(d, x, 16384, rate/2)
		if c.wantOpen && f < 0.9 {
			t.Errorf("target %.1f, sent %.1f at %.0f Hz dev: present %.0f%%, want >= 90%%", c.target, c.sent, c.dev, f*100)
		}
		if !c.wantOpen && f > 0 {
			t.Errorf("target %.1f, sent %.1f at %.0f Hz dev: present %.0f%%, want 0%%", c.target, c.sent, c.dev, f*100)
		}
	}
}
