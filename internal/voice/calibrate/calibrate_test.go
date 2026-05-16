package calibrate

import (
	"math"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"

	// Blank-import the in-tree vocoders so Compare can resolve
	// them by name through voice.DefaultRegistry.
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

func TestRMSZeroOnEmpty(t *testing.T) {
	if got := rms(nil); got != 0 {
		t.Errorf("rms(nil) = %f, want 0", got)
	}
}

func TestRMSConstantSignal(t *testing.T) {
	// Constant signal of amplitude A has RMS == A.
	const A = 1000
	samples := make([]int16, 800)
	for i := range samples {
		samples[i] = A
	}
	got := rms(samples)
	if math.Abs(got-A) > 1e-6 {
		t.Errorf("rms(constant %d) = %f, want %d", A, got, A)
	}
}

func TestRMSSineWaveHasExpectedAmplitude(t *testing.T) {
	// A sine of peak amplitude A has RMS = A / sqrt(2).
	const A = 16000
	const N = 8000 // one second of 8 kHz audio at 1 cycle/8000 samples
	samples := make([]int16, N)
	for i := range samples {
		samples[i] = int16(float64(A) * math.Sin(2*math.Pi*float64(i)/N))
	}
	want := float64(A) / math.Sqrt2
	got := rms(samples)
	if math.Abs(got-want)/want > 0.01 {
		t.Errorf("rms(sine A=%d) = %f, want %f (±1%%)", A, got, want)
	}
}

// TestBestXcorrFindsKnownDelay: build a signal plus a copy shifted
// by a known number of samples; bestXcorr must report that exact
// lag and a correlation magnitude close to 1.
func TestBestXcorrFindsKnownDelay(t *testing.T) {
	const N = 4000
	const knownLag = 37
	x := make([]int16, N)
	y := make([]int16, N)

	// Build a non-trivial signal (sum of two tones) so the
	// correlation peak is sharp.
	for i := 0; i < N; i++ {
		v := 8000 * (math.Sin(2*math.Pi*100*float64(i)/8000) +
			0.5*math.Sin(2*math.Pi*240*float64(i)/8000))
		x[i] = int16(v)
		if i+knownLag < N {
			y[i+knownLag] = x[i]
		}
	}

	// We search the lag that aligns x[i] with y[i+lag]. Built y so
	// y[i+knownLag] = x[i] ⇒ bestXcorr should peak at lag = +knownLag.
	peak, lag := bestXcorr(x, y, 200)
	if peak < 0.99 {
		t.Errorf("peak xcorr = %f, want > 0.99", peak)
	}
	if lag != knownLag {
		t.Errorf("lag = %d, want %d", lag, knownLag)
	}
}

func TestBestXcorrZeroEnergyInputs(t *testing.T) {
	x := make([]int16, 1000)
	y := make([]int16, 1000)
	if peak, lag := bestXcorr(x, y, 100); peak != 0 || lag != 0 {
		t.Errorf("bestXcorr(zeros) = (%f, %d), want (0, 0)", peak, lag)
	}
}

// seekableBuffer wraps a bytes.Buffer with the io.Seeker bits that
// WavWriter needs to patch length fields. The buffer grows on write
// so Seek calls past the end zero-pad the gap.
type seekableBuffer struct {
	data []byte
	pos  int64
}

func (b *seekableBuffer) Write(p []byte) (int, error) {
	end := b.pos + int64(len(p))
	if int64(cap(b.data)) < end {
		grown := make([]byte, end)
		copy(grown, b.data)
		b.data = grown
	} else if int64(len(b.data)) < end {
		b.data = b.data[:end]
	}
	copy(b.data[b.pos:end], p)
	b.pos = end
	return len(p), nil
}

func (b *seekableBuffer) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		b.pos = offset
	case 1:
		b.pos += offset
	case 2:
		b.pos = int64(len(b.data)) + offset
	}
	return b.pos, nil
}

// TestReadWavRoundtripsViaWavWriter: pipe int16 samples through
// voice.WavWriter, then read them back via readWav. The recovered
// samples and sample rate must match what we wrote.
func TestReadWavRoundtripsViaWavWriter(t *testing.T) {
	const want = 8000
	in := []int16{0, 1, 2, 100, 200, -100, -200, 32000, -32000}

	buf := &seekableBuffer{}
	wav, err := voice.NewWavWriter(buf, want)
	if err != nil {
		t.Fatalf("NewWavWriter: %v", err)
	}
	if err := wav.WriteSamples(in); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if err := wav.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Spill to a temp file so readWav can os.Open it.
	tf, err := os.CreateTemp("", "calibrate-wav-*.wav")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(tf.Name())
	if _, err := tf.Write(buf.data); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tf.Close()

	got, rate, err := readWav(tf.Name())
	if err != nil {
		t.Fatalf("readWav: %v", err)
	}
	if rate != want {
		t.Errorf("sample rate = %d, want %d", rate, want)
	}
	if len(got) != len(in) {
		t.Fatalf("sample count = %d, want %d", len(got), len(in))
	}
	for i := range in {
		if got[i] != in[i] {
			t.Errorf("samples[%d] = %d, want %d", i, got[i], in[i])
		}
	}
}

// TestCompareIMBESkipsWithoutFixtures: the calibration test for the
// IMBE decoder needs `internal/voice/imbe/testdata/p25-p1-voice.raw`
// plus a DSD-FME / OP25 reference WAV alongside it. Until those are
// checked in the test skips and CI stays green. Once fixtures land,
// remove the skip guard and the assertions enforce |RMS offset| <
// 3 dB and peak xcorr > 0.85.
func TestCompareIMBESkipsWithoutFixtures(t *testing.T) {
	const raw = "../imbe/testdata/p25-p1-voice.raw"
	const refWav = "../imbe/testdata/p25-p1-voice-dsdfme.wav"

	if _, err := os.Stat(raw); os.IsNotExist(err) {
		t.Skipf("no IMBE testdata at %s — capture and check in per README", raw)
	}
	if _, err := os.Stat(refWav); os.IsNotExist(err) {
		t.Skipf("no IMBE reference WAV at %s", refWav)
	}

	res, err := Compare(raw, refWav, "imbe")
	if err != nil {
		t.Fatalf("Compare(imbe): %v", err)
	}
	t.Logf("IMBE calibration: rmsRatio=%+.2f dB peakXcorr=%.3f lag=%d in=%d ref=%d",
		res.RMSRatioDb, res.PeakXcorr, res.LagSamples,
		res.InTreeSampleCount, res.RefSampleCount)

	if math.Abs(res.RMSRatioDb) > 3.0 {
		t.Errorf("IMBE RMS offset %.2f dB exceeds ±3 dB — adjust mbe.DefaultAGCConfig.TargetPeak in internal/voice/mbe/agc.go",
			res.RMSRatioDb)
	}
	if res.PeakXcorr < 0.85 {
		t.Errorf("IMBE peak xcorr %.3f < 0.85 (lag=%d) — verify §6.2 spectral enhancement and AGC attack/release in internal/voice/mbe/agc.go:DefaultAGCConfig",
			res.PeakXcorr, res.LagSamples)
	}
}

// TestCompareAMBE2SkipsWithoutFixtures: same idea for the AMBE+2
// decoder. Fixture path:
// `internal/voice/ambe2/testdata/dmr-voice.raw` plus reference WAV.
func TestCompareAMBE2SkipsWithoutFixtures(t *testing.T) {
	const raw = "../ambe2/testdata/dmr-voice.raw"
	const refWav = "../ambe2/testdata/dmr-voice-dsdfme.wav"

	if _, err := os.Stat(raw); os.IsNotExist(err) {
		t.Skipf("no AMBE+2 testdata at %s — capture and check in per README", raw)
	}
	if _, err := os.Stat(refWav); os.IsNotExist(err) {
		t.Skipf("no AMBE+2 reference WAV at %s", refWav)
	}

	res, err := Compare(raw, refWav, "ambe2")
	if err != nil {
		t.Fatalf("Compare(ambe2): %v", err)
	}
	t.Logf("AMBE+2 calibration: rmsRatio=%+.2f dB peakXcorr=%.3f lag=%d in=%d ref=%d",
		res.RMSRatioDb, res.PeakXcorr, res.LagSamples,
		res.InTreeSampleCount, res.RefSampleCount)

	if math.Abs(res.RMSRatioDb) > 3.0 {
		t.Errorf("AMBE+2 RMS offset %.2f dB exceeds ±3 dB — adjust mbe.DefaultAGCConfig.TargetPeak",
			res.RMSRatioDb)
	}
	if res.PeakXcorr < 0.85 {
		t.Errorf("AMBE+2 peak xcorr %.3f < 0.85 (lag=%d) — verify the dual-tone fallback in internal/voice/ambe2/decoder.go (b1 ∈ [128,163] routes through silence; see README) is not corrupting the in-tree signal",
			res.PeakXcorr, res.LagSamples)
	}
}

// TestCompareSamplesSyntheticGainOffset validates the calibrate
// math without external fixtures. Generates a 1 s sine through the
// "in-tree" stream + a +3 dB-louder copy as the "reference", then
// asserts CompareSamples reports the expected loudness offset and
// perfect waveform alignment. Keeps the math under test even when
// the real-air reference WAVs in internal/voice/{imbe,ambe2}/testdata
// haven't been checked in yet.
func TestCompareSamplesSyntheticGainOffset(t *testing.T) {
	const (
		sampleRateHz = 8000
		durationSec  = 1.0
		toneHz       = 440.0
		inAmplitude  = 5000.0 // int16 RMS of a sine at amplitude A is A/√2.
	)
	// gainLinear = 10^(3/20) ≈ 1.4125 → the reference is +3 dB louder
	// than the in-tree stream. Compare reports RMSRatioDb as
	// 20·log10(rmsIn / rmsRef), so a louder reference produces a
	// negative dB value.
	const wantDb = -3.0
	gainLinear := math.Pow(10, 3.0/20.0)

	n := int(sampleRateHz * durationSec)
	inTree := make([]int16, n)
	refSamples := make([]int16, n)
	for i := 0; i < n; i++ {
		v := inAmplitude * math.Sin(2*math.Pi*toneHz*float64(i)/sampleRateHz)
		inTree[i] = int16(v)
		refSamples[i] = int16(v * gainLinear)
	}

	res := CompareSamples(inTree, refSamples)

	if math.Abs(res.RMSRatioDb-wantDb) > 0.5 {
		t.Errorf("RMSRatioDb = %.3f dB, want %.3f ± 0.5 dB", res.RMSRatioDb, wantDb)
	}
	if res.PeakXcorr < 0.99 {
		t.Errorf("PeakXcorr = %.4f, want >= 0.99 (identical waveforms up to scale)", res.PeakXcorr)
	}
	if res.LagSamples != 0 {
		t.Errorf("LagSamples = %d, want 0 (no delay between identical streams)", res.LagSamples)
	}
	if res.InTreeSampleCount != n || res.RefSampleCount != n {
		t.Errorf("sample counts = (%d, %d), want (%d, %d)",
			res.InTreeSampleCount, res.RefSampleCount, n, n)
	}
}

// TestCompareSamplesSilencePath: one silent stream produces a
// well-defined zero RMS ratio (vs. NaN / −∞ from a log of zero).
func TestCompareSamplesSilencePath(t *testing.T) {
	const n = 800
	silent := make([]int16, n)
	noisy := make([]int16, n)
	for i := range noisy {
		noisy[i] = int16(1000 * math.Sin(2*math.Pi*440*float64(i)/8000))
	}

	res := CompareSamples(silent, noisy)
	if res.RMSRatioDb != 0 {
		t.Errorf("RMSRatioDb with silent in-tree = %v, want 0", res.RMSRatioDb)
	}
}

// TestCompareSamplesInvertedPolarity verifies the |xcorr| comment at
// calibrate.go:177-179: a 180°-phase-inverted reference produces
// PeakXcorr ≈ 1 (the polarity sign is squashed by math.Abs in
// bestXcorr) and RMSRatioDb ≈ 0. The polarity flag itself surfaces
// elsewhere (the test harness comparing decoded audio to a reference
// catches sign inversions via the audible result, not this metric).
func TestCompareSamplesInvertedPolarity(t *testing.T) {
	const (
		sampleRateHz = 8000
		n            = 8000 // 1 s
		amp          = 6000.0
		toneHz       = 440.0
	)
	inTree := make([]int16, n)
	inverted := make([]int16, n)
	for i := 0; i < n; i++ {
		v := amp * math.Sin(2*math.Pi*toneHz*float64(i)/sampleRateHz)
		inTree[i] = int16(v)
		inverted[i] = int16(-v)
	}

	res := CompareSamples(inTree, inverted)
	if res.PeakXcorr < 0.99 {
		t.Errorf("PeakXcorr for inverted polarity = %.4f, want >= 0.99 (|xcorr| collapses sign)", res.PeakXcorr)
	}
	if math.Abs(res.RMSRatioDb) > 0.5 {
		t.Errorf("RMSRatioDb for inverted polarity = %.3f dB, want |Δ| <= 0.5", res.RMSRatioDb)
	}
}

// TestCompareSamplesShortInputReturnsZeroXcorr covers the early-
// return guard at calibrate.go:185 — when both streams' usable length
// is at most 2*MaxLagSamples, the search window can't fit and the
// function must return PeakXcorr=0, LagSamples=0 instead of looping
// over an empty range.
func TestCompareSamplesShortInputReturnsZeroXcorr(t *testing.T) {
	// MaxLagSamples is 200; both streams length 400 yields n ≤
	// 2*maxLag = 400 → early-return branch.
	const n = 2 * MaxLagSamples
	x := make([]int16, n)
	y := make([]int16, n)
	for i := 0; i < n; i++ {
		v := 4000 * math.Sin(2*math.Pi*220*float64(i)/8000)
		x[i] = int16(v)
		y[i] = int16(v) // identical → would yield xcorr=1 if measured
	}

	res := CompareSamples(x, y)
	if res.PeakXcorr != 0 {
		t.Errorf("PeakXcorr on short input = %.4f, want 0 (guard at calibrate.go:185)", res.PeakXcorr)
	}
	if res.LagSamples != 0 {
		t.Errorf("LagSamples on short input = %d, want 0", res.LagSamples)
	}
}

// TestCompareSamplesLagAtBoundary verifies the search loop's outer
// bounds: a reference shifted by exactly MaxLagSamples samples must
// resolve to LagSamples=MaxLagSamples (an off-by-one in the bound
// check at calibrate.go:201 would land on MaxLagSamples-1 with a
// lower peak). Uses a deterministic LCG-generated noise sequence
// so the autocorrelation drops to near-zero at non-zero lags — a
// periodic test signal can land at a multiple of MaxLagSamples and
// produce a tie at lag=0.
func TestCompareSamplesLagAtBoundary(t *testing.T) {
	const (
		n   = 8000
		lag = MaxLagSamples
	)
	x := make([]int16, n)
	// Linear-congruential generator → uniform pseudo-random int16s.
	// Same trivial LCG used in Go's runtime for fast hashing; far
	// from cryptographic but its autocorrelation drops below 0.05
	// for any non-zero offset on this length.
	seed := uint32(0xC0FFEE)
	for i := 0; i < n; i++ {
		seed = seed*1103515245 + 12345
		x[i] = int16(seed >> 16)
	}
	// y is x shifted right by `lag` samples: y[i+lag] = x[i]. The
	// search reports the lag j where y[i+j] best matches x[i] — so
	// it should resolve to +lag.
	y := make([]int16, n)
	for i := 0; i+lag < n; i++ {
		y[i+lag] = x[i]
	}

	res := CompareSamples(x, y)
	if res.LagSamples != lag {
		t.Errorf("LagSamples = %d, want %d (search bound)", res.LagSamples, lag)
	}
	if res.PeakXcorr < 0.95 {
		t.Errorf("PeakXcorr at boundary lag = %.4f, want >= 0.95", res.PeakXcorr)
	}
}
