package imbe

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// rms returns the root-mean-square of an int16 PCM window.
func rms(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// nonZeroCount counts non-zero samples — a muted frame is exactly zero
// in its non-overlap region, so this measures how much audio leaked.
func nonZeroCount(samples []int16) int {
	n := 0
	for _, s := range samples {
		if s != 0 {
			n++
		}
	}
	return n
}

// TestUnpackHeaderIdleToneFlag pins the b_0 → IdleTone classification:
// the degenerate low corner [0, IdleToneMaxB0] flags IdleTone; ordinary
// fundamentals do not; the silence window stays Silent (not IdleTone).
func TestUnpackHeaderIdleToneFlag(t *testing.T) {
	for b0 := uint(0); b0 <= IdleToneMaxB0; b0++ {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Fatalf("b0=%d: unexpected err %v", b0, err)
		}
		if !h.IdleTone {
			t.Errorf("b0=%d: IdleTone=false, want true", b0)
		}
		if h.Silent {
			t.Errorf("b0=%d: Silent=true, want false", b0)
		}
	}
	for _, b0 := range []uint{IdleToneMaxB0 + 1, 9, 50, 100, 207} {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Fatalf("b0=%d: unexpected err %v", b0, err)
		}
		if h.IdleTone {
			t.Errorf("b0=%d: IdleTone=true, want false", b0)
		}
	}
	for b0 := uint(216); b0 <= 219; b0++ {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Fatalf("b0=%d: unexpected err %v", b0, err)
		}
		if !h.Silent {
			t.Errorf("b0=%d: Silent=false, want true", b0)
		}
		if h.IdleTone {
			t.Errorf("b0=%d: silence frame flagged IdleTone, want false", b0)
		}
	}
}

// TestDecodeMutesSustainedToneRun: a run of identical degenerate-tone
// frames is muted from the IdleToneRunThreshold-th frame on, and the
// mute resets the synthesizer state exactly like the silence path.
func TestDecodeMutesSustainedToneRun(t *testing.T) {
	d := New()
	tone := frameB0(6) // b_0=6, the field-observed idle-tone frame
	// First frame leaks (run count 1 < threshold) — at most one 20 ms buzz.
	if _, err := d.Decode(tone); err != nil {
		t.Fatalf("frame 0: %v", err)
	}
	// Frame at the threshold (and beyond) is muted.
	out, err := d.Decode(tone)
	if err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	if d.lowB0RunCount != IdleToneRunThreshold {
		t.Errorf("lowB0RunCount = %d, want %d", d.lowB0RunCount, IdleToneRunThreshold)
	}
	for i := 96; i < mbe.SamplesPerFrame; i++ {
		if out[i] != 0 {
			t.Fatalf("muted tone frame sample[%d] = %d, want 0", i, out[i])
		}
	}
	if d.state.PrevW0 != 0 || d.state.PrevL != 0 {
		t.Errorf("after mute: PrevW0=%v PrevL=%d, want 0/0", d.state.PrevW0, d.state.PrevL)
	}
}

// TestDecodeIsolatedLowB0NotMuted: a lone low-b_0 frame surrounded by
// real voice is NOT muted — the run threshold protects a transient
// low-pitch speech frame. Pins the false-positive guard.
func TestDecodeIsolatedLowB0NotMuted(t *testing.T) {
	d := New()
	if _, err := d.Decode(goodVoiceFrame()); err != nil {
		t.Fatalf("good 0: %v", err)
	}
	out, err := d.Decode(frameB0(6)) // single low-b_0 frame
	if err != nil {
		t.Fatalf("low-b0: %v", err)
	}
	if agcPeak(out) == 0 {
		t.Fatal("isolated low-b_0 frame was muted; run threshold should protect it")
	}
}

// TestDecodeToneRunResetsOnRealFrame verifies the run counter is managed
// outside clearLastGood: after a mute, a genuine voice frame clears the
// run (so it isn't dragged into a never-ending mute) and synthesizes.
func TestDecodeToneRunResetsOnRealFrame(t *testing.T) {
	d := New()
	tone := frameB0(6)
	_, _ = d.Decode(tone) // count 1
	_, _ = d.Decode(tone) // count 2 — muted
	if d.lowB0RunCount != IdleToneRunThreshold {
		t.Fatalf("pre-reset lowB0RunCount = %d, want %d", d.lowB0RunCount, IdleToneRunThreshold)
	}
	out, err := d.Decode(goodVoiceFrame())
	if err != nil {
		t.Fatalf("real frame: %v", err)
	}
	if d.lowB0RunCount != 0 {
		t.Errorf("after real frame: lowB0RunCount = %d, want 0", d.lowB0RunCount)
	}
	if agcPeak(out) == 0 {
		t.Fatal("real voice frame after a tone run was muted; want synthesis")
	}
}

// TestDecodeMutesVaryingLowB0Run: a noisy idle carrier resolves to
// low-b_0 frames whose *parameter* bits differ frame-to-frame (only the
// b_0 corner is stable). The run must still be recognised and muted —
// the discriminator keys on b_0, not on frame-to-frame similarity.
func TestDecodeMutesVaryingLowB0Run(t *testing.T) {
	d := New()
	if _, err := d.Decode(frameB0(6)); err != nil { // count 1, leaks
		t.Fatalf("frame 0: %v", err)
	}
	// Same b_0=6 but with a block of parameter bits set — very different
	// bits from the all-zero frame, yet still an idle-tone frame.
	info := putB0(6)
	for i := 20; i < 40; i++ {
		info[i] = 1
	}
	out, err := d.Decode(packInfoBits(info))
	if err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	if d.lowB0RunCount != IdleToneRunThreshold {
		t.Errorf("lowB0RunCount = %d, want %d (run continues across param variation)",
			d.lowB0RunCount, IdleToneRunThreshold)
	}
	for i := 96; i < mbe.SamplesPerFrame; i++ {
		if out[i] != 0 {
			t.Fatalf("varying low-b_0 run frame sample[%d] = %d, want 0 (muted)", i, out[i])
		}
	}
}

// TestDecodeToneRunFirstFrameOfStream: when a fresh decoder's very first
// frame is a tone (no predecessor), it still synthesizes (run count 1)
// and the next near-identical tone frame trips the mute.
func TestDecodeToneRunFirstFrameOfStream(t *testing.T) {
	d := New()
	out0, err := d.Decode(frameB0(6))
	if err != nil {
		t.Fatalf("frame 0: %v", err)
	}
	if agcPeak(out0) == 0 {
		t.Fatal("first frame of stream muted; threshold should let one frame through")
	}
	out1, err := d.Decode(frameB0(6))
	if err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	for i := 96; i < mbe.SamplesPerFrame; i++ {
		if out1[i] != 0 {
			t.Fatalf("second tone frame sample[%d] = %d, want 0 (muted)", i, out1[i])
		}
	}
}

// decodeRawFixture decodes a flat 11-byte-per-frame IMBE sidecar through
// a fresh decoder, returning the concatenated PCM. Skips the test when
// the fixture isn't present (mirrors testdata/README.md convention).
func decodeRawFixture(t *testing.T, name string) []int16 {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Skipf("fixture %s not present: %v", name, err)
	}
	d := New()
	var pcm []int16
	for off := 0; off+FrameBytes <= len(b); off += FrameBytes {
		out, derr := d.Decode(b[off : off+FrameBytes])
		if derr != nil {
			t.Fatalf("decode frame at %d: %v", off, derr)
		}
		pcm = append(pcm, out...)
	}
	return pcm
}

// TestDecodeIdleToneRawFixtureMostlySilent: a real captured all-tone
// dead-key call (every frame the b_0=6 idle buzz) decodes to near-total
// silence — only the leading run-onset frame (plus its overlap-add tail)
// may carry audio.
func TestDecodeIdleToneRawFixtureMostlySilent(t *testing.T) {
	pcm := decodeRawFixture(t, "p25p1-idle-tone.raw")
	if len(pcm) == 0 {
		t.Fatal("no PCM decoded")
	}
	nz := nonZeroCount(pcm)
	// Allow up to ~3 frames of leak (onset frame + OA tail) out of 144.
	const maxLeak = 3 * mbe.SamplesPerFrame
	if nz > maxLeak {
		t.Errorf("all-tone fixture decoded %d non-zero samples of %d; want <= %d (near-silent)",
			nz, len(pcm), maxLeak)
	}
}

// TestDecodeBracketedRawFixtureHasVoiceMiddle: a real call whose voice is
// bracketed by idle-tone runs decodes with the leading buzz muted while
// the voice in the middle is preserved.
func TestDecodeBracketedRawFixtureHasVoiceMiddle(t *testing.T) {
	pcm := decodeRawFixture(t, "p25p1-tone-bracketed-voice.raw")
	fpf := mbe.SamplesPerFrame
	if len(pcm) < 200*fpf {
		t.Fatalf("fixture too short: %d samples", len(pcm))
	}
	// Head window inside the leading tone run (frames 1..4, past the one
	// leaked onset frame) must be near-silent.
	head := rms(pcm[1*fpf : 5*fpf])
	// Middle window sits in real voice.
	mid := rms(pcm[60*fpf : 180*fpf])
	if mid < 500 {
		t.Errorf("middle RMS = %.0f, want substantial voice (>500)", mid)
	}
	if head > mid/4 {
		t.Errorf("leading-tone head RMS = %.0f not muted relative to voice mid RMS = %.0f", head, mid)
	}
}
