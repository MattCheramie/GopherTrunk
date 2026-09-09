package ccdecoder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

func tone(n int) []complex64 {
	s := make([]complex64, n)
	for i := range s {
		s[i] = complex(0.3, -0.2)
	}
	return s
}

// TestDDCRecorderWritesWav confirms the narrowband recorder opens a WAV at the
// pipeline rate and captures the channelized samples it is handed.
func TestDDCRecorderWritesWav(t *testing.T) {
	dir := t.TempDir()
	r := newDDCRecorder(dir, "", nil)
	if r == nil {
		t.Fatal("newDDCRecorder returned nil for a non-empty dir")
	}
	r.write("Main_Site_1", 144000, tone(1000))
	r.write("Main_Site_1", 144000, tone(500))
	r.close()

	matches, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 recording, got %d (%v)", len(matches), matches)
	}
	info, err := baseband.ReadIQWavInfo(matches[0])
	if err != nil {
		t.Fatalf("ReadIQWavInfo: %v", err)
	}
	if info.SampleRate != 144000 {
		t.Errorf("recording rate = %d, want 144000", info.SampleRate)
	}
	if info.Samples != 1500 {
		t.Errorf("recording samples = %d, want 1500", info.Samples)
	}
}

// TestDDCRecorderRollsOnRateChange confirms a change of system or pipeline rate
// starts a fresh file, so each recording holds one contiguous rate/system.
func TestDDCRecorderRollsOnRateChange(t *testing.T) {
	dir := t.TempDir()
	r := newDDCRecorder(dir, "", nil)
	r.write("Site_A", 144000, tone(100))
	r.write("Site_A", 48000, tone(100)) // rate change → roll
	r.write("Site_B", 48000, tone(100)) // system change → roll
	r.close()

	matches, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
	if len(matches) != 3 {
		t.Fatalf("expected 3 recordings after rolls, got %d (%v)", len(matches), matches)
	}
}

// TestDDCRecorderDisabled confirms an empty dir yields a nil recorder whose
// methods are safe no-ops (the zero-cost disabled path).
func TestDDCRecorderDisabled(t *testing.T) {
	var r *ddcRecorder = newDDCRecorder("", "", nil)
	if r != nil {
		t.Fatal("newDDCRecorder(\"\") should return nil")
	}
	// nil-safe
	r.write("x", 144000, tone(10))
	r.close()
}

// TestDDCRecorderIgnoresEmptyAndZeroRate guards the write fast-paths.
func TestDDCRecorderIgnoresEmptyAndZeroRate(t *testing.T) {
	dir := t.TempDir()
	r := newDDCRecorder(dir, "", nil)
	r.write("S", 144000, nil) // empty samples → no file
	r.write("S", 0, tone(10)) // zero rate → no file
	r.close()
	matches, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
	if len(matches) != 0 {
		t.Fatalf("expected no recordings, got %v", matches)
	}
	_ = os.RemoveAll(dir)
}

// TestDDCRecorderWritesFLAC confirms the format option produces a real FLAC
// recording at the pipeline rate with a .flac extension.
func TestDDCRecorderWritesFLAC(t *testing.T) {
	dir := t.TempDir()
	r := newDDCRecorder(dir, "flac", nil)
	if r == nil {
		t.Fatal("newDDCRecorder returned nil for a non-empty dir")
	}
	r.write("Main_Site_1", 144000, tone(5000))
	r.close()

	matches, _ := filepath.Glob(filepath.Join(dir, "*.flac"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 .flac recording, got %d (%v)", len(matches), matches)
	}
	info, err := baseband.ReadIQRecordingInfo(matches[0])
	if err != nil {
		t.Fatalf("ReadIQRecordingInfo: %v", err)
	}
	if info.SampleRate != 144000 {
		t.Errorf("recording rate = %d, want 144000", info.SampleRate)
	}
	if info.Samples != 5000 {
		t.Errorf("recording samples = %d, want 5000", info.Samples)
	}
}
