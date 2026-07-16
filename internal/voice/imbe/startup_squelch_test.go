package imbe

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// packInfo packs an 88-element bit slice into an 11-byte frame (MSB-first),
// the inverse of unpackInfoBits.
func packInfo(bits []byte) []byte {
	f := make([]byte, FrameBytes)
	for i := 0; i < InfoBits && i < len(bits); i++ {
		if bits[i]&1 == 1 {
			f[i/8] |= 1 << (7 - uint(i)%8)
		}
	}
	return f
}

// setB0 writes an 8-bit fundamental-frequency index into the info-bit slots
// b0FromInfo reads (info[0..5] high bits, info[85..86] low bits).
func setB0(bits []byte, b0 int) {
	bits[0] = byte((b0 >> 7) & 1)
	bits[1] = byte((b0 >> 6) & 1)
	bits[2] = byte((b0 >> 5) & 1)
	bits[3] = byte((b0 >> 4) & 1)
	bits[4] = byte((b0 >> 3) & 1)
	bits[5] = byte((b0 >> 2) & 1)
	bits[85] = byte((b0 >> 1) & 1)
	bits[86] = byte(b0 & 1)
}

// speechBits builds an info-bit pattern with a given (stable) b0 and a fixed
// non-b0 payload that decodes to a valid, non-idle, partly-voiced frame.
func speechBits(b0 int) []byte {
	bits := make([]byte, InfoBits)
	for i := range bits {
		bits[i] = byte((i * 7) % 3 & 1) // deterministic non-trivial payload
	}
	setB0(bits, b0)
	return bits
}

func firstNonZero(pcm []int16) int {
	for i, s := range pcm {
		if s != 0 {
			return i
		}
	}
	return -1
}

// TestStartupSquelchMutesUntilStableVoice verifies the call-startup acquisition
// squelch: with it enabled, jumping-pitch acquisition frames are muted and
// output only begins once a stable-pitch voiced run confirms real speech; with
// it disabled the decoder behaves as before (output from the first frame). The
// muted frames are zeroed, not dropped, so the length is unchanged.
func TestStartupSquelchMutesUntilStableVoice(t *testing.T) {
	// A "speech" frame reused for a stable-pitch run.
	speech := packInfo(speechBits(48))
	// Sanity: it must decode to a real (non-idle, non-silent), partly-voiced
	// frame, else the run can never confirm.
	probe := New()
	if s, err := probe.Decode(speech); err != nil || firstNonZero(s) < 0 {
		t.Fatalf("constructed speech frame did not synthesize (err=%v); adjust speechBits", err)
	}

	// Garbage: valid frames with wildly jumping pitch (acquisition signature).
	garbageB0 := []int{60, 190, 40, 175, 30, 200, 66}
	var stream [][]byte
	for _, b0 := range garbageB0 {
		stream = append(stream, packInfo(speechBits(b0)))
	}
	for i := 0; i < 10; i++ { // stable-pitch voiced run → releases the squelch
		stream = append(stream, speech)
	}

	decode := func(squelch bool) []int16 {
		d := New()
		if squelch {
			d.EnableStartupSquelch()
		}
		var out []int16
		for _, f := range stream {
			s, _ := d.Decode(f)
			out = append(out, s...)
		}
		return out
	}

	off := decode(false)
	on := decode(true)

	if len(on) != len(off) {
		t.Fatalf("squelch changed length: on=%d off=%d (must mute, not drop)", len(on), len(off))
	}
	garbageSamples := len(garbageB0) * mbe.SamplesPerFrame
	// Without the squelch, the garbage synthesizes within the garbage window —
	// this is the "startup scratch" the squelch removes.
	if off0 := firstNonZero(off); off0 < 0 || off0 >= garbageSamples {
		t.Fatalf("baseline (squelch off) produced no output in the garbage window (first=%d); "+
			"the fixture must exercise the scratch", off0)
	}
	// With the squelch, the jumping-pitch garbage stays muted: nothing in the
	// first len(garbageB0) frames.
	for i := 0; i < garbageSamples && i < len(on); i++ {
		if on[i] != 0 {
			t.Fatalf("squelch leaked garbage at sample %d (frame %d): got %d, want 0",
				i, i/mbe.SamplesPerFrame, on[i])
		}
	}
	// And it eventually releases on the stable-voiced run: the tail is non-zero.
	if firstNonZero(on) < 0 {
		t.Fatal("squelch never released on the stable-voiced run — whole call muted")
	}
	if firstNonZero(on) < garbageSamples {
		t.Fatalf("squelch released during the garbage (sample %d < %d)", firstNonZero(on), garbageSamples)
	}
}
