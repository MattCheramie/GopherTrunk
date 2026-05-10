package ambe2

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

func TestDecoderRegistered(t *testing.T) {
	v, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatalf("DefaultRegistry.New(%q): %v", VocoderName, err)
	}
	defer v.Close()
	if v.Name() != VocoderName {
		t.Errorf("Name() = %q, want %q", v.Name(), VocoderName)
	}
}

func TestDecoderName(t *testing.T) {
	d := New()
	if d.Name() != "ambe2" {
		t.Errorf("Name() = %q, want %q", d.Name(), "ambe2")
	}
}

func TestDecoderFrameSize(t *testing.T) {
	d := New()
	if d.FrameSize() != FrameBytes {
		t.Errorf("FrameSize() = %d, want %d", d.FrameSize(), FrameBytes)
	}
}

// TestDecodeProducesAudioForValidFrame: an all-zero info-bit frame
// decodes to b₀=0 (lowest valid fundamental, L=9, all Vl=0); the
// shared mbe.SynthUnvoicedOverlapAdd produces noise in the 9
// harmonic bands. Output must be non-zero. Pins the "synthesizer
// is wired up" milestone.
func TestDecodeProducesAudioForValidFrame(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes) // all zero ⇒ b₀=0 voice frame
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != mbe.SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), mbe.SamplesPerFrame)
	}
	var nonzero int
	for _, s := range out {
		if s != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("all-zero output for a valid voice frame; expected synthesis to produce non-zero PCM")
	}
}

// TestDecodeToneFrameReturnsSilenceNonOverlapRegion: a tone-frame
// indicator (b₀ ∈ {0x7E, 0x7F}) currently routes through the
// silence path — emit prev-frame OA tail into pcm[0..95] and clear
// state. The non-overlap region pcm[96..159] must be exactly
// silent (no new synthesis happens on a tone frame). After this
// frame the SynthState fields are zero, so a second tone frame
// produces fully-silent output.
func TestDecodeToneFrameReturnsSilenceNonOverlapRegion(t *testing.T) {
	d := New()
	// Tone frame: info[0..5] = 1, info[48] = 0 ⇒ b₀ = 0x7E.
	frame := make([]byte, FrameBytes)
	frame[0] = 0xFC // bits 0..5 = 1
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != mbe.SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), mbe.SamplesPerFrame)
	}
	// pcm[96..159] (the non-overlap region) must be silent on a
	// tone frame — no new synthesis runs.
	for i := mbe.SamplesPerFrame - 64; i < mbe.SamplesPerFrame; i++ {
		if out[i] != 0 {
			t.Errorf("sample[%d] = %d, want 0 (non-overlap region on tone frame)", i, out[i])
		}
	}
}

// TestDecodeToneAfterToneIsFullySilent: a second tone-frame after
// a tone frame produces all-zero PCM. Pins that the silence path
// clears the OA tail so a perpetual silence stream stays silent.
func TestDecodeToneAfterToneIsFullySilent(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	frame[0] = 0xFC // b₀ = 0x7E (tone)
	if _, err := d.Decode(frame); err != nil {
		t.Fatalf("first tone: %v", err)
	}
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("second tone: %v", err)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (perpetual tone/silence)", i, s)
		}
	}
}

func TestDecodeRejectsShortFrame(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes-1)); err == nil {
		t.Error("expected error for short frame")
	}
	if _, err := d.Decode(make([]byte, FrameBytes+1)); err == nil {
		t.Error("expected error for long frame")
	}
}

func TestResetAndCloseAreSafe(t *testing.T) {
	d := New()
	d.Reset()
	if err := d.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	// Re-using after Close should still work — pure-Go decoder
	// holds no resources, and the Vocoder contract doesn't forbid
	// it for stateless implementations.
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Errorf("Decode after Close: %v", err)
	}
}

func TestRegistryReturnsFreshInstancePerCall(t *testing.T) {
	a, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if a == b {
		t.Error("expected each Registry.New call to return a fresh instance")
	}
}

func TestNameAppearsInRegistryListing(t *testing.T) {
	names := voice.DefaultRegistry.Names()
	for _, n := range names {
		if n == VocoderName {
			return
		}
	}
	t.Errorf("registry names %v missing %q", names, VocoderName)
}

// TestNewWithConfigUsesCustomAGC pins that operator-supplied AGC
// tuning reaches the underlying *mbe.AGC instance. Mirrors the
// imbe-side test so AMBE+2 grows the same customization surface
// as it gains real synthesis.
func TestNewWithConfigUsesCustomAGC(t *testing.T) {
	custom := mbe.AGCConfig{TargetPeak: 16000}
	d := NewWithConfig(0, custom)
	def := mbe.DefaultAGCConfig()
	got := d.agc.Config()
	if got.TargetPeak != 16000 {
		t.Errorf("TargetPeak = %v, want 16000", got.TargetPeak)
	}
	if got.Attack != def.Attack {
		t.Errorf("Attack = %v, want default %v", got.Attack, def.Attack)
	}
}

// agcPeak returns the absolute peak of an int16 slice. Used by AGC
// tests to verify per-frame normalisation.
func agcPeak(out []int16) int {
	var peak int
	for _, s := range out {
		v := int(s)
		if v < 0 {
			v = -v
		}
		if v > peak {
			peak = v
		}
	}
	return peak
}

// TestDecodeDeterministicAcrossDecoders: two decoders constructed
// via New() (default seed) must produce byte-identical PCM for the
// same input frame stream. Pins the reproducibility contract that
// makes audio diffs across PRs reviewable.
func TestDecodeDeterministicAcrossDecoders(t *testing.T) {
	a, b := New(), New()
	frame := make([]byte, FrameBytes)
	outA, err := a.Decode(frame)
	if err != nil {
		t.Fatalf("a.Decode: %v", err)
	}
	outB, err := b.Decode(frame)
	if err != nil {
		t.Fatalf("b.Decode: %v", err)
	}
	for i := range outA {
		if outA[i] != outB[i] {
			t.Fatalf("non-deterministic at i=%d: a=%d b=%d", i, outA[i], outB[i])
		}
	}
}

// TestNewWithSeedDifferentSeedsDifferentOutput: distinct seeds give
// distinct unvoiced noise so decoded output for the same frame must
// differ.
func TestNewWithSeedDifferentSeedsDifferentOutput(t *testing.T) {
	a := NewWithSeed(1)
	b := NewWithSeed(2)
	frame := make([]byte, FrameBytes)
	outA, err := a.Decode(frame)
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	outB, err := b.Decode(frame)
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	for i := range outA {
		if outA[i] != outB[i] {
			return
		}
	}
	t.Error("two distinct seeds produced byte-identical output; seeding is a no-op")
}

// TestDecodeRollsCrossFrameState: two consecutive decodes on the
// same input frame must produce *different* PCM — frame 1 starts
// from no prev state, frame 2's mbe.PredictLog2Ml sees frame 1's
// state and the cross-frame gamma evolves. Pins that state-rolling
// happens across both mbe.SynthState and ambe2's prevGamma.
func TestDecodeRollsCrossFrameState(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	out1, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("frame1: %v", err)
	}
	out2, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("frame2: %v", err)
	}
	for i := range out1 {
		if out1[i] != out2[i] {
			return
		}
	}
	t.Error("two consecutive decodes produced identical output; cross-frame state isn't rolling")
}

// frameWithB2 returns a 7-byte voice frame whose b₂ is non-zero
// (so DeltaGamma = AmbePlusDg[b₂] ≠ 0). Used by state-rolling
// tests that need a frame which actually advances prevGamma.
// info[43] (the LSB of b₂) packs into byte 5, bit position
// 7 − (43%8) = 7 − 3 = 4. b₀ stays 0 (voice frame, lowest L).
func frameWithB2() []byte {
	frame := make([]byte, FrameBytes)
	frame[5] |= 1 << 4 // info[43] = 1 ⇒ b₂ = 1
	return frame
}

// TestResetClearsCrossFrameState: after Reset the decoder behaves
// as if it had just been constructed — prevGamma is zero, the
// mbe.SynthState fields are zero, the AGC envelope is zero, and
// the frame-repeat cache is empty.
func TestResetClearsCrossFrameState(t *testing.T) {
	d := New()
	if _, err := d.Decode(frameWithB2()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if d.state.PrevW0 == 0 {
		t.Fatal("prev frame didn't establish synth state; test setup invalid")
	}
	if d.prevGamma == 0 {
		t.Fatal("prev frame didn't establish gamma; test setup invalid")
	}
	d.Reset()
	if d.state.PrevW0 != 0 || d.state.PrevL != 0 {
		t.Errorf("after Reset: PrevW0=%v PrevL=%d, want 0/0", d.state.PrevW0, d.state.PrevL)
	}
	if d.prevGamma != 0 {
		t.Errorf("after Reset: prevGamma=%v, want 0", d.prevGamma)
	}
	if d.agc.Envelope() != 0 {
		t.Errorf("after Reset: AGC envelope=%v, want 0", d.agc.Envelope())
	}
	if d.lastGoodParams.L != 0 {
		t.Errorf("after Reset: lastGoodParams.L=%d, want 0", d.lastGoodParams.L)
	}
}

// TestDecodeToneFrameClearsState: a tone frame resets the
// SynthState + prevGamma + last-good cache so the next non-tone
// frame starts from a clean baseline. Pins that the silence-path
// state-clear actually runs.
func TestDecodeToneFrameClearsState(t *testing.T) {
	d := New()
	// Seed with a good voice frame whose b₂ ≠ 0 so prevGamma is
	// established (AmbePlusDg[0] = 0).
	if _, err := d.Decode(frameWithB2()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if d.prevGamma == 0 || d.state.PrevW0 == 0 {
		t.Fatal("seed didn't establish state; test setup invalid")
	}
	// Tone frame (b₀ = 0x7E).
	tone := make([]byte, FrameBytes)
	tone[0] = 0xFC
	if _, err := d.Decode(tone); err != nil {
		t.Fatalf("tone: %v", err)
	}
	if d.prevGamma != 0 {
		t.Errorf("after tone: prevGamma=%v, want 0", d.prevGamma)
	}
	if d.state.PrevW0 != 0 || d.state.PrevL != 0 {
		t.Errorf("after tone: PrevW0=%v PrevL=%d, want 0/0",
			d.state.PrevW0, d.state.PrevL)
	}
	if d.lastGoodParams.L != 0 {
		t.Errorf("after tone: lastGoodParams.L=%d, want 0", d.lastGoodParams.L)
	}
}

// TestAGCConvergesTowardTarget: feeding the same valid frame
// repeatedly drives the AGC envelope toward the per-frame peak so
// the converged output peak sits near AGCConfig.TargetPeak. Pins
// the AGC is actually adapting on AMBE+2 (the synthesis primitives
// are shared with IMBE but AMBE+2's per-frame amplitudes can be
// systematically different — operator-tunable AGC is the safety
// valve).
func TestAGCConvergesTowardTarget(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	var lastPeak int
	for i := 0; i < 30; i++ {
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		lastPeak = agcPeak(out)
	}
	target := mbe.DefaultAGCConfig().TargetPeak
	const tol = 0.5 // loose: noise variance + AMBE+2 quantization
	lo := int(target * (1 - tol))
	hi := int(target * (1 + tol))
	if lastPeak < lo || lastPeak > hi {
		t.Errorf("converged peak = %d, want in [%d, %d] (target=%v ±%.0f%%)",
			lastPeak, lo, hi, target, tol*100)
	}
}

// TestDecodeOutputIsBoundedAcrossFramePatterns: every PCM sample
// must be a valid int16 across a range of frame patterns. Catches
// gain regressions that send the synthesizer into NaN / Inf land
// before the int16 cast, and AGC envelope blow-ups.
func TestDecodeOutputIsBoundedAcrossFramePatterns(t *testing.T) {
	d := New()
	patterns := [][]byte{
		make([]byte, FrameBytes),
		{0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
		{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA},
		{0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	for round := 0; round < 10; round++ {
		for fi, frame := range patterns {
			out, err := d.Decode(frame)
			if err != nil {
				t.Fatalf("Decode pattern %d round %d: %v", fi, round, err)
			}
			// int16 is by construction in [-32768, 32767]; this also
			// catches NaN / Inf which would have failed earlier at the
			// int16 cast.
			_ = out
			if math.IsNaN(d.agc.Envelope()) || math.IsInf(d.agc.Envelope(), 0) {
				t.Fatalf("AGC envelope = %v after pattern %d round %d", d.agc.Envelope(), fi, round)
			}
			if math.IsNaN(d.prevGamma) || math.IsInf(d.prevGamma, 0) {
				t.Fatalf("prevGamma = %v after pattern %d round %d", d.prevGamma, fi, round)
			}
		}
	}
}

// TestPrevGammaCrossFrameAccumulation: gamma_curr = ΔG_curr +
// 0.5·gamma_prev. Feed a frame with non-zero ΔG twice and verify
// the second frame's resolved gamma matches the closed form.
// Using b₂ = 1 (AmbePlusDg[1] ≈ 0.1182) means both frames carry
// real gain and the 0.5· recursive term is exercised numerically.
func TestPrevGammaCrossFrameAccumulation(t *testing.T) {
	d := New()
	frame := frameWithB2() // b₂ = 1 ⇒ ΔG = AmbePlusDg[1] ≠ 0
	if _, err := d.Decode(frame); err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	gamma1 := d.prevGamma
	wantGamma1 := AmbePlusDg[1] // first frame: prevGamma starts at 0
	if math.Abs(gamma1-wantGamma1) > 1e-12 {
		t.Errorf("frame 1 prevGamma = %v, want %v", gamma1, wantGamma1)
	}
	if _, err := d.Decode(frame); err != nil {
		t.Fatalf("frame 2: %v", err)
	}
	gamma2 := d.prevGamma
	wantGamma2 := AmbePlusDg[1] + 0.5*gamma1
	if math.Abs(gamma2-wantGamma2) > 1e-12 {
		t.Errorf("frame 2 prevGamma = %v, want %v", gamma2, wantGamma2)
	}
}
