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
	d := NewDownconverter(48_000, 48_000)
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

// TestDownconverterTuningOffset: with a tuning offset and no decimation
// (rate == target), a tone at +offsetHz is shifted to DC — it becomes a
// constant phasor — and the caller's input slice is never mutated.
func TestDownconverterTuningOffset(t *testing.T) {
	const fs, offset = 48_000.0, 750.0
	d := NewDownconverterWithOffset(fs, fs, offset)
	if d.nco == nil {
		t.Fatalf("expected an NCO for a non-zero offset")
	}
	in := complexTone(offset, fs, 4096, 0.8)
	cpy := append([]complex64(nil), in...)
	out := d.Process(nil, in)

	// raw must be untouched.
	for i := range in {
		if in[i] != cpy[i] {
			t.Fatalf("Process mutated raw at %d: %v != %v", i, in[i], cpy[i])
		}
	}
	// The shifted tone is DC: imaginary energy collapses relative to a
	// reference real axis. Measure residual frequency via mean phase step.
	if len(out) != len(in) {
		t.Fatalf("len(out) = %d, want %d", len(out), len(in))
	}
	var sumStep float64
	for i := 1; i < len(out); i++ {
		a := math.Atan2(float64(imag(out[i])), float64(real(out[i])))
		b := math.Atan2(float64(imag(out[i-1])), float64(real(out[i-1])))
		d := a - b
		if d > math.Pi {
			d -= 2 * math.Pi
		} else if d < -math.Pi {
			d += 2 * math.Pi
		}
		sumStep += d
	}
	meanStep := sumStep / float64(len(out)-1) // rad/sample residual
	if math.Abs(meanStep) > 1e-3 {
		t.Errorf("residual phase step %.2e rad/sample after tuning, want ~0", meanStep)
	}
}

// TestDownconverterDecimationRate: a 2.048 MHz SDR rate must land
// exactly on the 48 kHz target, and the output chunk length must
// follow the L/M ratio.
func TestDownconverterDecimationRate(t *testing.T) {
	d := NewDownconverter(2_048_000, 48_000)
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
	inband := NewDownconverter(sdrRate, 48_000).
		Process(nil, complexTone(5_000, sdrRate, n, 1.0))
	gotIn := rms(inband[100:]) // skip filter start-up transient

	// Out of band: +400 kHz. Without filtering this folds to
	// 400000 mod 48000 = 16 kHz — right in the passband — so the
	// anti-alias filter is the only thing that can reject it.
	interf := NewDownconverter(sdrRate, 48_000).
		Process(nil, complexTone(400_000, sdrRate, n, 1.0))
	gotInterf := rms(interf[100:])

	if gotIn < 0.7 {
		t.Errorf("in-channel tone attenuated: rms = %v, want ~1.0", gotIn)
	}
	if gotInterf >= 0.01 {
		t.Errorf("interferer not rejected: rms = %v, want < 0.01 (≥ 40 dB)", gotInterf)
	}
}

// TestDownconverterUpsampleBuildsResampler: a pre-channelized capture
// BELOW the target rate must be interpolated up to ~target, not passed
// through — the regression guard for the low-rate-replay bug where a
// 39062.5 Hz / 48000 Hz WAV reached the TETRA receiver at a fractional
// samples-per-symbol and decoded at the wrong baud. Before the fix
// NewDownconverter returned a nil resampler for in < target.
func TestDownconverterUpsampleBuildsResampler(t *testing.T) {
	const target = tetraDDCTargetRateHz // 144 kHz, 8 sps at 18000 baud

	// 48 kHz → 144 kHz is an exact 3× interpolation.
	d48 := NewDownconverter(48_000, target)
	if d48.resampler == nil {
		t.Errorf("48000 → %v: expected a resampler (was pass-through before fix)", target)
	}
	if math.Abs(d48.outRateHz-target) > 1e-6 {
		t.Errorf("48000 → %v: outRateHz = %v, want %v exactly", target, d48.outRateHz, target)
	}

	// 39062.5 Hz (2.5 MS/s ÷ 64, the SDR++ baseband rate) does not
	// reduce to small terms, so the ratio search snaps to within 1 %.
	dbb := NewDownconverter(39_062.5, target)
	if dbb.resampler == nil {
		t.Fatalf("39062.5 → %v: expected a resampler (was pass-through before fix)", target)
	}
	if relErr := math.Abs(dbb.outRateHz-target) / target; relErr > ddcUpTolerance {
		t.Errorf("39062.5 → %v: outRateHz = %v (rel err %.4f), want within %.2f%%",
			target, dbb.outRateHz, relErr, ddcUpTolerance*100)
	}
	if sps := dbb.outRateHz / 18_000; sps < 7.5 || sps > 8.5 {
		t.Errorf("39062.5 → %v: sps = %.3f, want ~8 (in [7.5, 8.5])", target, sps)
	}
}

// TestDownconverterUpsampleRateAndLength: the interpolated output has
// ~len(in)·L/M samples and reports the corresponding rate.
func TestDownconverterUpsampleRateAndLength(t *testing.T) {
	const target = tetraDDCTargetRateHz
	d := NewDownconverter(48_000, target)
	in := make([]complex64, 48_000) // 1 s at 48 kHz
	out := d.Process(nil, in)
	want := len(in) * 3 // exact 3× interpolation
	if diff := len(out) - want; diff < -8 || diff > 8 {
		t.Errorf("len(out) = %d, want %d ± 8", len(out), want)
	}
}

// TestDownconverterUpsamplePreservesInbandTone: interpolation must not
// attenuate an in-channel tone (gain restored) nor let a spectral image
// leak into the band. Mirror of TestDownconverterIsolatesChannel for
// the L>M path.
func TestDownconverterUpsamplePreservesInbandTone(t *testing.T) {
	const inRate, target = 48_000.0, tetraDDCTargetRateHz
	const n = 48_000
	// +5 kHz is well inside both the 48 kHz input band and the TETRA
	// channel; it must survive interpolation at ~unity gain.
	out := NewDownconverter(inRate, target).
		Process(nil, complexTone(5_000, inRate, n, 1.0))
	if got := rms(out[200:]); got < 0.7 || got > 1.3 {
		t.Errorf("in-band tone after upsample: rms = %v, want ~1.0", got)
	}
}

// TestDownconverterPassthroughPreserved: the fix must not disturb the
// rate==target no-op path — only in < target changed.
func TestDownconverterPassthroughPreserved(t *testing.T) {
	for _, rate := range []float64{48_000, tetraDDCTargetRateHz} {
		d := NewDownconverter(rate, rate)
		if d.resampler != nil {
			t.Errorf("rate==target %v: expected pass-through (nil resampler)", rate)
		}
		if d.outRateHz != rate {
			t.Errorf("rate==target %v: outRateHz = %v, want %v", rate, d.outRateHz, rate)
		}
	}
}

// TestDownconverterReset: after Reset the decimation filter history
// is cleared, so re-processing the same input reproduces the first
// run bit-for-bit.
func TestDownconverterReset(t *testing.T) {
	d := NewDownconverter(2_048_000, 48_000)
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
