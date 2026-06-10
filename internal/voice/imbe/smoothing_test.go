package imbe

import "testing"

// TestSetFrameErrorsZeroIsInert: feeding 0 corrected bits before each
// Decode produces output byte-identical to never calling SetFrameErrors —
// adaptive smoothing must not perturb clean audio.
func TestSetFrameErrorsZeroIsInert(t *testing.T) {
	frame := goodVoiceFrame()
	a := New()
	b := New()
	for i := 0; i < 30; i++ {
		oa, err := a.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		b.SetFrameErrors(0)
		ob, err := b.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		for j := range oa {
			if oa[j] != ob[j] {
				t.Fatalf("frame %d sample %d differs: %d vs %d (smoothing not inert at 0 errors)", i, j, oa[j], ob[j])
			}
		}
	}
}

// TestSustainedErrorsMute: feeding the max corrected-bit count before every
// Decode eventually drives the error rate past the mute threshold, so the
// output decays to silence.
func TestSustainedErrorsMute(t *testing.T) {
	d := New()
	frame := goodVoiceFrame()
	var lastPeak int
	for i := 0; i < 300; i++ {
		d.SetFrameErrors(15)
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		lastPeak = agcPeak(out)
	}
	if lastPeak != 0 {
		t.Errorf("after sustained max errors, output peak = %d, want 0 (muted)", lastPeak)
	}
}
