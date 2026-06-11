package voice

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
)

const nrmRate = 8000

// writeToneWAV writes a 16-bit mono WAV of a sine at the given amplitude
// (0..1 of full scale) and returns its path.
func writeToneWAV(t *testing.T, dir string, amp float64, durSec int) string {
	t.Helper()
	path := filepath.Join(dir, "tone.wav")
	w, err := NewWavFile(path, nrmRate)
	if err != nil {
		t.Fatalf("NewWavFile: %v", err)
	}
	n := nrmRate * durSec
	samples := make([]int16, n)
	for i := range samples {
		v := amp * math.Sin(2*math.Pi*1000*float64(i)/nrmRate)
		samples[i] = int16(math.Round(v * 32767))
	}
	if err := w.WriteSamples(samples); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return path
}

func measureWAV(t *testing.T, path string) (lufs float64, peak float64, rate uint32, ok bool) {
	t.Helper()
	samples, rate, err := ReadWAVSamples(path)
	if err != nil {
		t.Fatalf("ReadWAVSamples: %v", err)
	}
	fs := make([]float64, len(samples))
	for i, s := range samples {
		fs[i] = float64(s) / 32768.0
	}
	lufs, ok = loudness.IntegratedLUFS(fs, int(rate))
	peak = loudness.TruePeak(fs, int(rate))
	return lufs, peak, rate, ok
}

func TestNormalizeWAVFile_HitsTarget(t *testing.T) {
	dir := t.TempDir()
	// A quiet tone (~-26 LUFS) that needs a few dB of boost, well within
	// the ±12 dB cap.
	path := writeToneWAV(t, dir, 0.1, 3)

	cfg := NormalizeConfig{Enabled: true}.withDefaults()
	if err := normalizeWAVFile(path, cfg); err != nil {
		t.Fatalf("normalizeWAVFile: %v", err)
	}

	lufs, peak, rate, ok := measureWAV(t, path)
	if !ok {
		t.Fatal("expected measurable audio after normalize")
	}
	if rate != nrmRate {
		t.Fatalf("sample rate changed: %d", rate)
	}
	if math.Abs(lufs-cfg.TargetLUFS) > 0.5 {
		t.Fatalf("post-normalize loudness %.2f LUFS not near target %.2f", lufs, cfg.TargetLUFS)
	}
	ceiling := loudness.DBToLinear(cfg.TruePeakDBTP)
	if peak > ceiling+1e-3 {
		t.Fatalf("true peak %.4f exceeds ceiling %.4f", peak, ceiling)
	}
}

func TestNormalizeWAVFile_TruePeakLimited(t *testing.T) {
	dir := t.TempDir()
	// A loud tone already near full scale: boosting to -16 LUFS would
	// clip, so the true-peak limiter must hold it under the ceiling.
	path := writeToneWAV(t, dir, 0.95, 3)

	cfg := NormalizeConfig{Enabled: true}.withDefaults()
	if err := normalizeWAVFile(path, cfg); err != nil {
		t.Fatalf("normalizeWAVFile: %v", err)
	}
	_, peak, _, ok := measureWAV(t, path)
	if !ok {
		t.Fatal("expected measurable audio")
	}
	ceiling := loudness.DBToLinear(cfg.TruePeakDBTP)
	if peak > ceiling+1e-3 {
		t.Fatalf("true peak %.4f exceeds ceiling %.4f after limiting", peak, ceiling)
	}
}

func TestNormalizeWAVFile_BoostClamped(t *testing.T) {
	dir := t.TempDir()
	// Extremely quiet tone (~-46 LUFS): reaching -16 would need +30 dB,
	// but the cap is +12 dB, so it should fall well short of target.
	path := writeToneWAV(t, dir, 0.005, 3)

	cfg := NormalizeConfig{Enabled: true}.withDefaults()
	before, _, _, ok := measureWAV(t, path)
	if !ok {
		t.Fatal("baseline measure failed")
	}
	if err := normalizeWAVFile(path, cfg); err != nil {
		t.Fatalf("normalizeWAVFile: %v", err)
	}
	after, _, _, ok := measureWAV(t, path)
	if !ok {
		t.Fatal("post measure failed")
	}
	gained := after - before
	if math.Abs(gained-cfg.MaxBoostDB) > 0.3 {
		t.Fatalf("expected boost clamped to %.1f dB, got %.2f dB", cfg.MaxBoostDB, gained)
	}
	if after > cfg.TargetLUFS {
		t.Fatalf("clamped result %.2f should stay below target %.2f", after, cfg.TargetLUFS)
	}
}

func TestNormalizeWAVFile_SilenceSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "silent.wav")
	w, err := NewWavFile(path, nrmRate)
	if err != nil {
		t.Fatalf("NewWavFile: %v", err)
	}
	if err := w.WriteSamples(make([]int16, nrmRate*3)); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	w.Close()

	info1, _ := os.Stat(path)
	if err := normalizeWAVFile(path, NormalizeConfig{Enabled: true}.withDefaults()); err != nil {
		t.Fatalf("normalizeWAVFile: %v", err)
	}
	info2, _ := os.Stat(path)
	// Skipped: file untouched (same size, no temp left behind).
	if info1.Size() != info2.Size() {
		t.Fatalf("silent file should be left untouched (sizes %d != %d)", info1.Size(), info2.Size())
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should not remain")
	}
}

func TestNormalizeWAVFile_NoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	path := writeToneWAV(t, dir, 0.1, 3)
	if err := normalizeWAVFile(path, NormalizeConfig{Enabled: true}.withDefaults()); err != nil {
		t.Fatalf("normalizeWAVFile: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should not remain after atomic rewrite")
	}
}
