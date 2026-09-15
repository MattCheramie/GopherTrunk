package voice

import (
	"path/filepath"
	"testing"
)

// TestFlacWriterOddRateUsesStreamInfoRate: the mono voice twin shares the
// frame-header rate policy with the IQ encode core — a rate the frame header
// cannot carry (above 65535 Hz and not a multiple of 10) must defer to
// STREAMINFO rather than fail the first flushed block. Voice runs at 8 kHz in
// practice; this pins the shared rule so the two writers cannot drift.
func TestFlacWriterOddRateUsesStreamInfoRate(t *testing.T) {
	const rate = 88_001
	path := filepath.Join(t.TempDir(), "odd.flac")
	w, err := NewFlacFile(path, rate)
	if err != nil {
		t.Fatalf("NewFlacFile: %v", err)
	}
	pcm := speechLikeSamples(10_000, 0.3)
	if err := w.WriteSamples(pcm); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	got, gotRate, err := ReadFLACSamples(path)
	if err != nil {
		t.Fatalf("ReadFLACSamples: %v", err)
	}
	if gotRate != rate || len(got) != len(pcm) {
		t.Fatalf("decoded rate %d / %d samples, want %d / %d", gotRate, len(got), rate, len(pcm))
	}
	for i := range pcm {
		if got[i] != pcm[i] {
			t.Fatalf("sample %d = %d, want %d", i, got[i], pcm[i])
		}
	}
}
