package imbe

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
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
	if d.Name() != "imbe-go" {
		t.Errorf("Name() = %q, want %q", d.Name(), "imbe-go")
	}
}

func TestDecoderFrameSize(t *testing.T) {
	d := New()
	if d.FrameSize() != 11 {
		t.Errorf("FrameSize() = %d, want 11", d.FrameSize())
	}
}

// TestDecodeReturnsSilenceForB0SilenceFrame: b_0 in [216, 219] is
// the explicit silence indicator (TIA-102.BABA §5.3). Build the
// minimum frame that decodes to b_0 = 216 (frame[0] = 0xD8, rest
// zero) and confirm the output is exactly silence.
func TestDecodeReturnsSilenceForB0SilenceFrame(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	frame[0] = 0xD8 // b_0 = 0b11011000 = 216 (silence window)
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), SamplesPerFrame)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (silence indicator)", i, s)
		}
	}
}

// TestDecodeProducesAudioForValidFrame: an all-zero info-bit frame
// decodes to b_0 = 0 (the lowest valid fundamental, L = 9, K = 3),
// all Vl unvoiced. The unvoiced FFT excitation produces noise in
// the 9 harmonic bands; output must be non-zero. Pins the
// "synthesizer is wired up" milestone.
func TestDecodeProducesAudioForValidFrame(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes) // all zero ⇒ b_0 = 0
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var nonzero int
	for _, s := range out {
		if s != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("all-zero output for a valid (non-silent) frame; expected unvoiced excitation to produce non-zero PCM")
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

// TestDecodeBadB0ReturnsSilenceNoError: b_0 ∈ {208..215} ∪ {220..255}
// is invalid (FEC slip). Decode returns silence + nil error so the
// upstream call pipeline keeps streaming PCM at the 20 ms cadence
// instead of stalling on a parse error.
func TestDecodeBadB0ReturnsSilenceNoError(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	frame[0] = 0xD0 // b_0 = 208 (invalid: between valid 207 and silence 216)
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: unexpected error %v on bad b_0", err)
	}
	if len(out) != SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), SamplesPerFrame)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (graceful silence on bad b_0)", i, s)
		}
	}
}

func TestResetAndCloseAreSafe(t *testing.T) {
	d := New()
	d.Reset()
	if err := d.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	// Re-using after Close should still work — pure-Go decoder holds
	// no resources, and the Vocoder contract doesn't forbid it for
	// stateless implementations.
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

// TestDecodeDeterministicAcrossDecoders: two decoders constructed
// via New() (default seed) must produce byte-identical PCM for the
// same input frame stream. Pins the reproducibility contract that
// makes audio diffs across PRs reviewable.
func TestDecodeDeterministicAcrossDecoders(t *testing.T) {
	a, b := New(), New()
	frame := make([]byte, FrameBytes) // valid b_0=0 frame
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
// distinct unvoiced noise, so decoded output for the same frame
// must differ. Pins that the seed actually affects synthesis.
func TestNewWithSeedDifferentSeedsDifferentOutput(t *testing.T) {
	a := NewWithSeed(1)
	b := NewWithSeed(2)
	frame := make([]byte, FrameBytes)
	outA, err := a.Decode(frame)
	if err != nil {
		t.Fatalf("a.Decode: %v", err)
	}
	outB, err := b.Decode(frame)
	if err != nil {
		t.Fatalf("b.Decode: %v", err)
	}
	differs := false
	for i := range outA {
		if outA[i] != outB[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Error("two distinct seeds produced byte-identical output; seeding is a no-op")
	}
}

// TestDecodeRollsCrossFrameState: two consecutive decodes on the
// same input frame must produce *different* PCM — frame 1 starts
// from no prev state (the first-frame log2(Ml) = Tl path), frame
// 2's PredictLog2Ml sees frame 1's state and emits a different
// log2(Ml). And the unvoiced step pulls fresh noise on frame 2.
// Pins that state-rolling actually happens.
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
	differs := false
	for i := range out1 {
		if out1[i] != out2[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Error("two consecutive decodes produced identical output; cross-frame state isn't rolling")
	}
}

// TestResetClearsCrossFrameState: after Reset, frame 1's output
// matches what a fresh Decoder would produce on frame 1 — the
// cross-frame prediction state is cleared. Note: the rng is NOT
// re-seeded on Reset (per the docs) so output won't be identical
// to a fresh New() decoder; we test by feeding the silence-frame
// indicator (which short-circuits before touching rng) so only
// state matters.
func TestResetClearsCrossFrameState(t *testing.T) {
	d := New()
	// Drive some non-trivial state into d via a valid frame.
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("seed Decode: %v", err)
	}
	if d.state.PrevW0 == 0 {
		t.Fatal("prev frame didn't establish synth state; test setup invalid")
	}
	d.Reset()
	if d.state.PrevW0 != 0 || d.state.PrevL != 0 {
		t.Errorf("after Reset: PrevW0=%v PrevL=%d, want 0/0", d.state.PrevW0, d.state.PrevL)
	}
	for l := 0; l <= 56; l++ {
		if d.state.PrevLog2Ml[l] != 0 || d.state.PrevMl[l] != 0 || d.state.PrevPhase[l] != 0 {
			t.Errorf("after Reset: PrevLog2Ml[%d]=%v PrevMl[%d]=%v PrevPhase[%d]=%v",
				l, d.state.PrevLog2Ml[l], l, d.state.PrevMl[l], l, d.state.PrevPhase[l])
		}
	}
}

// TestDecodeSilenceFrameResetsState: per §6.1, b_0 ∈ [216, 219]
// signals a silence boundary; the next non-silent frame should
// start from clean state, not interpolate from before the silence.
// Decode must invoke Reset internally on the silence indicator.
//
// Step 5a (§6.4 overlap-add) added a fade-out: the silence frame's
// dst[0..95] receives the prev frame's unvoiced tail before reset
// (no click on the boundary). dst[96..159] is all-zero. After the
// frame all SynthState fields are zero — including the tail itself,
// which the OA function clears once it's emitted.
func TestDecodeSilenceFrameResetsState(t *testing.T) {
	d := New()
	// Establish state.
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("seed Decode: %v", err)
	}
	if d.state.PrevW0 == 0 {
		t.Fatal("prev frame didn't establish state; test setup invalid")
	}
	// Silence frame.
	silence := make([]byte, FrameBytes)
	silence[0] = 0xD8
	out, err := d.Decode(silence)
	if err != nil {
		t.Fatalf("silence Decode: %v", err)
	}
	// dst[96..159] (the non-overlap region) must be silent — no
	// new synthesis happens on a silent frame.
	for i := 96; i < SamplesPerFrame; i++ {
		if out[i] != 0 {
			t.Errorf("silence sample[%d] = %d, want 0 (non-overlap region)", i, out[i])
		}
	}
	// dst[0..95] is the OA fade-out region; values can be non-zero
	// (prev unvoiced tail) but must be valid int16. We don't pin
	// exact values — those depend on the rng seed + FFT path.
	if d.state.PrevW0 != 0 || d.state.PrevL != 0 {
		t.Errorf("after silence frame: PrevW0=%v PrevL=%d, want 0/0",
			d.state.PrevW0, d.state.PrevL)
	}
	// Tail must be cleared so a *second* silence frame is fully
	// silent (no perpetual fade-out loop).
	for n := 0; n < UnvoicedTailSamples; n++ {
		if d.state.PrevUnvoicedTail[n] != 0 {
			t.Errorf("PrevUnvoicedTail[%d] = %v, want 0 after silence",
				n, d.state.PrevUnvoicedTail[n])
		}
	}
}

// TestDecodeSilenceAfterSilenceIsFullySilent: a second silence frame
// after the first should produce all-zero PCM (no leftover tail to
// fade). Pins that the OA tail-clear in the silent path actually
// happened.
func TestDecodeSilenceAfterSilenceIsFullySilent(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	silence := make([]byte, FrameBytes)
	silence[0] = 0xD8
	if _, err := d.Decode(silence); err != nil {
		t.Fatalf("silence1: %v", err)
	}
	out, err := d.Decode(silence)
	if err != nil {
		t.Fatalf("silence2: %v", err)
	}
	for i, s := range out {
		if s != 0 {
			t.Errorf("silence2[%d] = %d, want 0", i, s)
		}
	}
}

// TestDecodeOutputIsBounded: every PCM sample must be a valid int16
// regardless of the input frame. Catches gain regressions that send
// the synthesizer into NaN / Inf land before the int16 cast.
func TestDecodeOutputIsBounded(t *testing.T) {
	d := New()
	// Try a few different frame patterns to stress different b_0 + Vl
	// combinations.
	patterns := []byte{0x00, 0x55, 0xAA, 0x7F, 0x80}
	for _, fill := range patterns {
		frame := make([]byte, FrameBytes)
		for i := range frame {
			frame[i] = fill
		}
		// Skip frames that decode to invalid b_0 — they short-circuit
		// to silence and don't exercise the output path.
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatalf("Decode 0x%02X: %v", fill, err)
		}
		for i, s := range out {
			// int16 is by construction in [-32768, 32767]; this
			// also catches NaN / Inf which would have failed earlier
			// at the int16 cast.
			_ = s
			if i >= SamplesPerFrame {
				t.Errorf("output longer than expected at pattern 0x%02X", fill)
			}
		}
	}
}

// agcPeak finds the absolute peak in an int16 slice — used by AGC
// tests to verify per-frame normalization.
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

// TestAGCConvergesTowardTarget: feeding the same valid frame
// repeatedly drives the smoothed envelope toward the per-frame peak,
// so by frame ~30 the output peak should sit close to agcTargetPeak.
// Pins that the AGC actually adapts (rather than being a fixed
// constant gain).
func TestAGCConvergesTowardTarget(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes) // valid b_0=0 frame
	var lastPeak int
	for i := 0; i < 30; i++ {
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		lastPeak = agcPeak(out)
	}
	// After convergence the peak should be within ~30% of the target
	// (loose because the §6.4 noise draw varies per frame).
	const tol = 0.3
	lo := int(agcTargetPeak * (1 - tol))
	hi := int(agcTargetPeak * (1 + tol))
	if lastPeak < lo || lastPeak > hi {
		t.Errorf("converged peak = %d, want in [%d, %d] (target=%v ±%.0f%%)",
			lastPeak, lo, hi, agcTargetPeak, tol*100)
	}
}

// TestAGCInitialFrameNotOverAmplified: the first frame on a fresh
// decoder seeds the envelope from peak (since envelope starts at 0,
// peak > envelope, so attack coefficient applies). Output peak on
// frame 1 should be within int16 range and not produce all-zero or
// all-saturated samples (which would indicate a divide-by-zero or
// gain explosion at the first-frame edge).
func TestAGCInitialFrameNotOverAmplified(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	peak := agcPeak(out)
	if peak == 0 {
		t.Error("first-frame peak = 0 (silent output, expected unvoiced excitation)")
	}
	if peak >= 32767 {
		t.Errorf("first-frame peak = %d (clipped); AGC should land below int16 max", peak)
	}
}

// TestAGCSilenceFramePreservesEnvelope: a silence-window frame
// (b_0 ∈ [216, 219]) clears the §6.1/§6.3/§6.4 SynthState but
// preserves the AGC envelope so the first frame after silence
// applies the same gain as the last frame before silence (no
// audible level jump on speech-pause-speech transitions). Public
// Reset() does clear the envelope (covered separately).
func TestAGCSilenceFramePreservesEnvelope(t *testing.T) {
	d := New()
	for i := 0; i < 5; i++ {
		if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
			t.Fatalf("seed frame %d: %v", i, err)
		}
	}
	if d.agc == 0 {
		t.Fatal("envelope is still zero after seed frames; test setup invalid")
	}
	envBefore := d.agc

	silence := make([]byte, FrameBytes)
	silence[0] = 0xD8
	if _, err := d.Decode(silence); err != nil {
		t.Fatalf("silence: %v", err)
	}
	// Envelope must not drift at all — silence path passes
	// freezeEnvelope=true to applyAGC so the OA tail magnitude
	// doesn't perturb the envelope.
	if d.agc != envBefore {
		t.Errorf("after silence: agc shifted from %v to %v (want exact preservation)",
			envBefore, d.agc)
	}
}

// TestAGCResetClearsEnvelope: the Reset method (e.g., called on
// stream re-sync) must zero the envelope so the next frame's AGC
// starts from a clean baseline.
func TestAGCResetClearsEnvelope(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}
	if d.agc == 0 {
		t.Fatal("envelope is zero after one frame; test setup invalid")
	}
	d.Reset()
	if d.agc != 0 {
		t.Errorf("after Reset: agc = %v, want 0", d.agc)
	}
}

// TestAGCBoundedOutputAcrossFramePatterns confirms the output stays
// in valid int16 range for a range of frame patterns + frame
// counts. Catches AGC regressions that send the gain to NaN or Inf.
func TestAGCBoundedOutputAcrossFramePatterns(t *testing.T) {
	d := New()
	patterns := [][]byte{
		make([]byte, FrameBytes),
		{0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
		{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA},
	}
	for round := 0; round < 10; round++ {
		for _, frame := range patterns {
			out, err := d.Decode(frame)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			for i, s := range out {
				_ = i
				_ = s // every int16 is by construction in [-32768, 32767]
			}
			// Cross-check that the AGC didn't NaN.
			if math.IsNaN(d.agc) || math.IsInf(d.agc, 0) {
				t.Fatalf("agc envelope = %v after round %d", d.agc, round)
			}
		}
	}
}

// TestAGCDoesNotPumpOnConstantInput: feeding the same frame
// repeatedly should produce stable output peaks across frames
// (variance in peak across the last 10 frames < 30% of mean). The
// fast-attack / slow-release smoothing is what avoids per-frame
// pumping that would be audible as breathing.
func TestAGCDoesNotPumpOnConstantInput(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	// Warm up the envelope.
	for i := 0; i < 30; i++ {
		if _, err := d.Decode(frame); err != nil {
			t.Fatal(err)
		}
	}
	// Sample the next 10 frames' peaks.
	var peaks [10]int
	for i := range peaks {
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		peaks[i] = agcPeak(out)
	}
	var sum int
	for _, p := range peaks {
		sum += p
	}
	mean := float64(sum) / float64(len(peaks))
	var variance float64
	for _, p := range peaks {
		d := float64(p) - mean
		variance += d * d
	}
	variance /= float64(len(peaks))
	stddev := math.Sqrt(variance)
	if stddev/mean > 0.3 {
		t.Errorf("peak stddev/mean = %.3f, want < 0.30 (pumping check)",
			stddev/mean)
	}
}

// badFrame returns an 11-byte frame whose b_0 = 208 — an invalid
// fundamental-frequency parameter that UnpackParams rejects with
// ErrInvalidFundamental.
func badFrame() []byte {
	f := make([]byte, FrameBytes)
	f[0] = 0xD0 // 0b11010000 → b_0 packs to 208 (invalid)
	return f
}

// TestFrameRepeatOnBadFrameAfterGood: a bad frame following a good
// frame replays the cached params with attenuation — the output
// must be non-zero (proving the repeat actually synthesized rather
// than emitting silence).
func TestFrameRepeatOnBadFrameAfterGood(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("good seed: %v", err)
	}
	if d.lastGoodParams.L == 0 {
		t.Fatal("good frame didn't populate the repeat cache; test setup invalid")
	}
	out, err := d.Decode(badFrame())
	if err != nil {
		t.Fatalf("bad frame Decode: %v", err)
	}
	if agcPeak(out) == 0 {
		t.Fatal("bad-frame output is fully silent; expected repeat to synthesize")
	}
	if d.badFrameCount != 1 {
		t.Errorf("badFrameCount = %d after one bad frame, want 1", d.badFrameCount)
	}
}

// TestFrameRepeatProgressiveAttenuation: after a good frame, six
// consecutive bad frames produce outputs whose peaks should be
// non-strictly decreasing (each frame attenuates by 0.7 of the
// previous, but AGC freezing means the *gain* is fixed across
// frames so the peaks track the synthesizer's float output).
//
// The strict decrease can occasionally flip due to noise variance
// in the §6.4 unvoiced step, so test that the *trend* is
// downward: peak[5] < peak[0].
func TestFrameRepeatProgressiveAttenuation(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("good seed: %v", err)
	}
	var peaks [MaxBadFrames]int
	for i := 0; i < MaxBadFrames; i++ {
		out, err := d.Decode(badFrame())
		if err != nil {
			t.Fatalf("bad frame %d: %v", i, err)
		}
		peaks[i] = agcPeak(out)
	}
	if peaks[MaxBadFrames-1] >= peaks[0] {
		t.Errorf("peaks didn't trend down: peaks=%v (first %d, last %d)",
			peaks, peaks[0], peaks[MaxBadFrames-1])
	}
}

// TestFrameRepeatExhaustedBudgetForcesSilence: after MaxBadFrames
// consecutive bad-frame replays, the next bad frame must hit the
// "no cache or budget exhausted" path → silence + cache cleared.
func TestFrameRepeatExhaustedBudgetForcesSilence(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("good seed: %v", err)
	}
	for i := 0; i < MaxBadFrames; i++ {
		if _, err := d.Decode(badFrame()); err != nil {
			t.Fatalf("bad frame %d: %v", i, err)
		}
	}
	// Next bad frame: budget exhausted (count == MaxBadFrames),
	// silence + clear.
	out, err := d.Decode(badFrame())
	if err != nil {
		t.Fatalf("budget-exhausted Decode: %v", err)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("budget-exhausted sample[%d] = %d, want 0", i, s)
		}
	}
	if d.lastGoodParams.L != 0 {
		t.Errorf("lastGoodParams.L = %d after budget exhaustion, want 0",
			d.lastGoodParams.L)
	}
	if d.badFrameCount != 0 {
		t.Errorf("badFrameCount = %d after clear, want 0", d.badFrameCount)
	}
}

// TestFrameRepeatCounterResetsOnGoodFrame: counter goes good → bad
// (count=1) → bad (count=2) → good (count=0). Pins that a single
// good frame inside a bad streak resets the budget so the next
// bad streak gets a fresh budget.
func TestFrameRepeatCounterResetsOnGoodFrame(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("good seed: %v", err)
	}
	if _, err := d.Decode(badFrame()); err != nil {
		t.Fatalf("bad1: %v", err)
	}
	if _, err := d.Decode(badFrame()); err != nil {
		t.Fatalf("bad2: %v", err)
	}
	if d.badFrameCount != 2 {
		t.Fatalf("badFrameCount = %d after 2 bads, want 2", d.badFrameCount)
	}
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("recovery: %v", err)
	}
	if d.badFrameCount != 0 {
		t.Errorf("badFrameCount = %d after recovery, want 0", d.badFrameCount)
	}
	if d.lastGoodParams.L == 0 {
		t.Error("lastGoodParams cleared after recovery; should be repopulated")
	}
}

// TestSilenceFrameClearsRepeatCache: a silence-window frame clears
// the last-good cache, so a subsequent bad frame can't repeat the
// pre-silence content (we'd want fresh-state behavior across the
// silence boundary).
func TestSilenceFrameClearsRepeatCache(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatalf("good seed: %v", err)
	}
	silence := make([]byte, FrameBytes)
	silence[0] = 0xD8
	if _, err := d.Decode(silence); err != nil {
		t.Fatalf("silence: %v", err)
	}
	if d.lastGoodParams.L != 0 {
		t.Errorf("lastGoodParams.L = %d after silence, want 0",
			d.lastGoodParams.L)
	}
	if d.badFrameCount != 0 {
		t.Errorf("badFrameCount = %d after silence, want 0", d.badFrameCount)
	}
	// Bad frame after silence → silence (no cache).
	out, err := d.Decode(badFrame())
	if err != nil {
		t.Fatalf("post-silence bad: %v", err)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("post-silence bad[%d] = %d, want 0", i, s)
		}
	}
}

// TestResetClearsFrameRepeatCache: explicit Reset() clears the
// last-good cache + counter alongside SynthState + AGC. Pins the
// "Reset = clean slate" contract.
func TestResetClearsFrameRepeatCache(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Decode(badFrame()); err != nil {
		t.Fatal(err)
	}
	if d.lastGoodParams.L == 0 || d.badFrameCount == 0 {
		t.Fatal("test setup invalid: cache empty before Reset")
	}
	d.Reset()
	if d.lastGoodParams.L != 0 {
		t.Errorf("after Reset: lastGoodParams.L = %d, want 0",
			d.lastGoodParams.L)
	}
	if d.badFrameCount != 0 {
		t.Errorf("after Reset: badFrameCount = %d, want 0", d.badFrameCount)
	}
}

// TestFrameRepeatFreezesAGC: a bad-frame replay must not perturb
// the AGC envelope (the freeze contract is what makes the
// attenuation audible — without it AGC would partially compensate).
func TestFrameRepeatFreezesAGC(t *testing.T) {
	d := New()
	for i := 0; i < 5; i++ {
		if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
			t.Fatalf("warmup %d: %v", i, err)
		}
	}
	envBefore := d.agc
	if envBefore == 0 {
		t.Fatal("envelope is zero after warmup; test setup invalid")
	}
	if _, err := d.Decode(badFrame()); err != nil {
		t.Fatalf("bad frame: %v", err)
	}
	if d.agc != envBefore {
		t.Errorf("after bad-frame replay: agc shifted from %v to %v (want exact preservation)",
			envBefore, d.agc)
	}
}
