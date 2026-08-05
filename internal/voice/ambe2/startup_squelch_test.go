package ambe2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// energy sums the squared samples over a slice (a cheap loudness proxy).
func energy(pcm []int16) float64 {
	var e float64
	for _, s := range pcm {
		e += float64(s) * float64(s)
	}
	return e
}

// TestStartupSquelchMutesOnsetThenReleases pins the call-startup acquisition
// squelch ported from the IMBE decoder. It decodes the committed real DMR
// fixture (testdata/dmr-voice.raw) twice — squelch off vs on — and asserts:
//  1. the two decodes have identical length (the squelch zeroes, never drops);
//  2. the guaranteed-mute window (first 2 frames, well inside the ~3-frame
//     ramp before `acquired` can latch) is all-zero WITH squelch but has real
//     audio WITHOUT it — this fails before the port (no squelch on AMBE2);
//  3. the squelch RELEASES on real speech well before the failsafe: every
//     sample from frame 4 onward is byte-identical to the un-squelched decode
//     (the mute runs after the AGC, so once released the paths converge). A
//     mis-tuned acqMaxW0Jump that only released via the 100-frame failsafe
//     would zero the fixture's short voiced burst and diverge here.
func TestStartupSquelchMutesOnsetThenReleases(t *testing.T) {
	const fr = mbe.SamplesPerFrame // 160 samples / 20 ms

	off := decodeSampleWith(t, NewDMR())

	sq := NewDMR()
	sq.EnableStartupSquelch()
	on := decodeSampleWith(t, sq)

	if len(on) != len(off) {
		t.Fatalf("squelch changed length: on=%d off=%d (must zero, not drop)", len(on), len(off))
	}
	if len(off) < 5*fr {
		t.Fatalf("fixture too short: %d samples", len(off))
	}

	// (2) Guaranteed-mute window: first 2 frames muted with squelch on...
	muteWin := on[:2*fr]
	if e := energy(muteWin); e != 0 {
		t.Errorf("first 2 frames not muted with squelch on: energy=%.0f", e)
	}
	// ...but real audio there without it (otherwise the assertion is vacuous).
	if energy(off[:2*fr]) == 0 {
		t.Fatal("fixture has no onset energy to squelch — test would be vacuous")
	}

	// (3) Early release: identical from frame 4 onward (proves the squelch
	// un-muted on real speech, not via the failsafe, and left audio untouched).
	for i := 4 * fr; i < len(on); i++ {
		if on[i] != off[i] {
			t.Fatalf("post-release sample %d differs (frame %d): on=%d off=%d — "+
				"squelch did not release cleanly (mis-tuned acqMaxW0Jump?)",
				i, i/fr, on[i], off[i])
		}
	}
	// And the released tail must actually carry audio (not silence).
	if energy(on[4*fr:]) == 0 {
		t.Fatal("no audio after release — squelch over-muted the call")
	}
}
