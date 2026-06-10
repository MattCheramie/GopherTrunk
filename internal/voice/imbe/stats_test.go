package imbe

import "testing"

// TestVoiceStatsAccumulates checks the per-call telemetry the recorder
// and `gophertrunk decode` surface for voice-quality triage.
func TestVoiceStatsAccumulates(t *testing.T) {
	d := New()
	if _, ok := d.VoiceStats(); ok {
		t.Fatal("VoiceStats ok == true before any Decode")
	}
	const n = 40
	frame := goodVoiceFrame()
	for i := 0; i < n; i++ {
		if _, err := d.Decode(frame); err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
	}
	vs, ok := d.VoiceStats()
	if !ok {
		t.Fatal("VoiceStats ok == false after decoding")
	}
	if vs.Frames != n {
		t.Errorf("Frames = %d, want %d", vs.Frames, n)
	}
	if vs.TotalSamples != n*160 {
		t.Errorf("TotalSamples = %d, want %d", vs.TotalSamples, n*160)
	}
	if vs.OutputRMS <= 0 {
		t.Errorf("OutputRMS = %v, want > 0", vs.OutputRMS)
	}
	if vs.CrestFactor <= 0 {
		t.Errorf("CrestFactor = %v, want > 0", vs.CrestFactor)
	}
	// A healthy AGC keeps clipping negligible on ordinary speech-like
	// frames; a regression to the over-driven gain would spike this.
	if vs.ClipPct() > 5 {
		t.Errorf("ClipPct = %.2f%%, want low (over-driven AGC regression?)", vs.ClipPct())
	}
}

// TestResetStatsClears confirms ResetStats zeroes the accumulator while
// Reset (synthesis re-sync) does NOT — a mid-call re-sync must keep the
// call's running totals.
func TestResetStatsClears(t *testing.T) {
	d := New()
	frame := goodVoiceFrame()
	for i := 0; i < 5; i++ {
		d.Decode(frame)
	}
	d.Reset() // synthesis re-sync — stats must survive
	if vs, ok := d.VoiceStats(); !ok || vs.Frames != 5 {
		t.Errorf("after Reset: Frames = %d ok=%v, want 5 true (Reset must not drop stats)", vs.Frames, ok)
	}
	d.ResetStats()
	if _, ok := d.VoiceStats(); ok {
		t.Error("after ResetStats: ok == true, want false")
	}
}
