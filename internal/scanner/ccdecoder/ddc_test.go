package ccdecoder

import (
	"math"
	"testing"
)

// complexTone synthesises n samples of a complex exponential at
// freqHz given a sample rate of rateHz.
func complexTone(freqHz, rateHz float64, n int, amp float64) []complex64 {
	out := make([]complex64, n)
	w := 2 * math.Pi * freqHz / rateHz
	for i := range out {
		out[i] = complex(
			float32(amp*math.Cos(w*float64(i))),
			float32(amp*math.Sin(w*float64(i))),
		)
	}
	return out
}

func rms(s []complex64) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, c := range s {
		r, i := float64(real(c)), float64(imag(c))
		sum += r*r + i*i
	}
	return math.Sqrt(sum / float64(len(s)))
}

// TestDownconverterPassthrough: when the SDR already streams at the
// narrowband target the down-converter builds no resampler and
// forwards the chunk unchanged.
func TestDownconverterPassthrough(t *testing.T) {
	d := newDownconverter(48_000, 48_000)
	if d.resampler != nil {
		t.Errorf("expected pass-through (nil resampler) for rate == target")
	}
	if d.outRateHz != 48_000 {
		t.Errorf("outRateHz = %v, want 48000", d.outRateHz)
	}
	in := make([]complex64, 256)
	out := d.Process(nil, in)
	if len(out) != 256 {
		t.Errorf("pass-through len(out) = %d, want 256", len(out))
	}
}

// TestDownconverterDecimationRate: a 2.048 MHz SDR rate must land
// exactly on the 48 kHz target, and the output chunk length must
// follow the L/M ratio.
func TestDownconverterDecimationRate(t *testing.T) {
	d := newDownconverter(2_048_000, 48_000)
	if d.resampler == nil {
		t.Fatalf("expected a resampler for 2.048 MHz → 48 kHz")
	}
	if math.Abs(d.outRateHz-48_000) > 1e-6 {
		t.Errorf("outRateHz = %v, want 48000 exactly", d.outRateHz)
	}
	in := make([]complex64, 204_800) // 0.1 s at 2.048 MHz
	out := d.Process(nil, in)
	want := len(in) * 48_000 / 2_048_000
	if diff := len(out) - want; diff < -8 || diff > 8 {
		t.Errorf("len(out) = %d, want %d ± 8", len(out), want)
	}
}

// TestDownconverterIsolatesChannel: an in-channel tone survives the
// decimation at ~unity gain while an out-of-band interferer (which
// would otherwise alias straight into the passband) is rejected by
// well over 40 dB. This is the core anti-alias guarantee the fix
// depends on — and a check that dsp.Resampler's stopband is adequate.
func TestDownconverterIsolatesChannel(t *testing.T) {
	const sdrRate = 2_048_000.0
	const n = 262_144

	// In-channel: +5 kHz, comfortably inside the 48 kHz output band.
	inband := newDownconverter(sdrRate, 48_000).
		Process(nil, complexTone(5_000, sdrRate, n, 1.0))
	gotIn := rms(inband[100:]) // skip filter start-up transient

	// Out of band: +400 kHz. Without filtering this folds to
	// 400000 mod 48000 = 16 kHz — right in the passband — so the
	// anti-alias filter is the only thing that can reject it.
	interf := newDownconverter(sdrRate, 48_000).
		Process(nil, complexTone(400_000, sdrRate, n, 1.0))
	gotInterf := rms(interf[100:])

	if gotIn < 0.7 {
		t.Errorf("in-channel tone attenuated: rms = %v, want ~1.0", gotIn)
	}
	if gotInterf >= 0.01 {
		t.Errorf("interferer not rejected: rms = %v, want < 0.01 (≥ 40 dB)", gotInterf)
	}
}

// TestDownconverterReset: after Reset the decimation filter history
// is cleared, so re-processing the same input reproduces the first
// run bit-for-bit.
func TestDownconverterReset(t *testing.T) {
	d := newDownconverter(2_048_000, 48_000)
	in := complexTone(5_000, 2_048_000, 200_000, 1.0)

	first := append([]complex64(nil), d.Process(nil, in)...)
	d.Reset()
	second := d.Process(nil, in)

	if len(first) != len(second) {
		t.Fatalf("len after reset = %d, want %d", len(second), len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("sample %d after reset = %v, want %v", i, second[i], first[i])
		}
	}
}
