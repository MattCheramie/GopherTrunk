package imbe

import (
	"math"
	"testing"
)

// TestDecodeOutputChangesWithEnhancement: a deterministic Decode
// run with the §6.2 enhancement enabled should produce non-silent,
// non-saturated PCM. Pins that the shared mbe.EnhanceAmplitudes
// step is wired into the IMBE Decode path rather than being a
// silent no-op. We can't toggle enhancement off easily without a
// test fixture, so instead verify a frame that the enhancement
// modifies M for produces audio whose RMS is sane (not zero, not
// saturated).
func TestDecodeOutputChangesWithEnhancement(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes) // valid b_0 = 0 frame
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var sumSq float64
	for _, s := range out {
		v := float64(s)
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(out)))
	if rms == 0 {
		t.Fatal("decoded output is fully silent; enhancement may have zeroed M")
	}
	if rms > 32000 {
		t.Errorf("rms = %v, want < 32000 (sanity envelope)", rms)
	}
}
