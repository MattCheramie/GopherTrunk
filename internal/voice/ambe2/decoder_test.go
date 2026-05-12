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

// singleToneFrame builds a 7-byte AMBE+2 tone frame with a chosen
// single-tone b1 ∈ [5, 122] and 8-bit volume b2. The tone-frame
// b1 packing uses the t5/t6/t7 table lookup for bits 5..7 — to
// keep the test self-contained we force idx = 1 (info[6,7,8] =
// 0,0,1) so those tables emit (0,0,0), letting bits 0..4 of b1
// be set directly via info[9, 42, 43, 10, 11].
func singleToneFrame(b1, b2 int) []byte {
	if b1 < 0 || b1 > 31 {
		panic("singleToneFrame test helper only emits b1 in [0, 31] (high bits via t-table idx=1 are forced 0)")
	}
	info := make([]byte, InfoBits)
	// b0 = 0x7E (tone frame): info[0..5] = 1, info[48] = 0.
	for i := 0; i <= 5; i++ {
		info[i] = 1
	}
	// t-table idx = 1 → bits 5..7 of b1 = 0.
	info[8] = 1
	// b1 low 5 bits via info[9, 42, 43, 10, 11] (MSB-first: bit4 → 16).
	info[9] = byte((b1 >> 4) & 1)
	info[42] = byte((b1 >> 3) & 1)
	info[43] = byte((b1 >> 2) & 1)
	info[10] = byte((b1 >> 1) & 1)
	info[11] = byte(b1 & 1)
	// b2 (8 bits) via info[12, 13, 14, 15, 16, 44, 45, 17].
	info[12] = byte((b2 >> 7) & 1)
	info[13] = byte((b2 >> 6) & 1)
	info[14] = byte((b2 >> 5) & 1)
	info[15] = byte((b2 >> 4) & 1)
	info[16] = byte((b2 >> 3) & 1)
	info[44] = byte((b2 >> 2) & 1)
	info[45] = byte((b2 >> 1) & 1)
	info[17] = byte(b2 & 1)
	// Pack info bits into 7 bytes MSB-first to mirror the wire format.
	frame := make([]byte, FrameBytes)
	for i := 0; i < InfoBits; i++ {
		frame[i/8] |= info[i] << (7 - uint(i)%8)
	}
	return frame
}

// dualToneFrame builds an AMBE+2 tone frame with b1 ∈ [128, 143]
// (the DTMF dual-tone range) and 8-bit volume b2. The bit layout
// matches the single-tone helper above; the difference is the t-
// table dispatch: for the dual-tone range the upper-3-bit pattern
// is (t7, t6, t5) = (1, 0, 0), which corresponds to t-table
// idx = 0 (info[6,7,8] = 0, 0, 0).
func dualToneFrame(b1, b2 int) []byte {
	if b1 < 128 || b1 > 143 {
		panic("dualToneFrame test helper only emits b1 in [128, 143]")
	}
	info := make([]byte, InfoBits)
	for i := 0; i <= 5; i++ {
		info[i] = 1
	}
	// t-table idx = 0 → info[6,7,8] all zero (default). Skip.
	// Low 4 bits of b1 via the same info positions as single-tone.
	low := b1 - 128
	info[9] = byte((low >> 4) & 1) // bit 4 of low (always 0 for [128,143])
	info[42] = byte((low >> 3) & 1)
	info[43] = byte((low >> 2) & 1)
	info[10] = byte((low >> 1) & 1)
	info[11] = byte(low & 1)
	// b2 packing — identical to single-tone.
	info[12] = byte((b2 >> 7) & 1)
	info[13] = byte((b2 >> 6) & 1)
	info[14] = byte((b2 >> 5) & 1)
	info[15] = byte((b2 >> 4) & 1)
	info[16] = byte((b2 >> 3) & 1)
	info[44] = byte((b2 >> 2) & 1)
	info[45] = byte((b2 >> 1) & 1)
	info[17] = byte(b2 & 1)
	frame := make([]byte, FrameBytes)
	for i := 0; i < InfoBits; i++ {
		frame[i/8] |= info[i] << (7 - uint(i)%8)
	}
	return frame
}

// TestDecodeDualToneDTMFAudible: a valid DTMF dual-tone frame
// (b1 ∈ [128, 143]) synthesises audio. Pins that the dual-tone
// path actually produces sound rather than the legacy
// route-to-silence behaviour.
func TestDecodeDualToneDTMFAudible(t *testing.T) {
	d := New()
	// b1 = 128 (DTMF "1": 697, 1209 Hz); b2 = 200 (loud).
	out, err := d.Decode(dualToneFrame(128, 200))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != mbe.SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), mbe.SamplesPerFrame)
	}
	if agcPeak(out) == 0 {
		t.Fatal("DTMF dual-tone frame output is fully silent; synthesis didn't run")
	}
}

// TestDecodeDualToneDTMFFrequencyContent: a DTMF "1" tone should
// have most of its energy split between 697 Hz and 1209 Hz. We do
// a coarse Goertzel at each expected frequency and at a quiet
// neighbour (300 Hz) and verify both expected bins dominate.
func TestDecodeDualToneDTMFFrequencyContent(t *testing.T) {
	d := New()
	// Decode 5 frames so the AGC has settled and we have enough
	// samples for stable Goertzel magnitudes.
	frame := dualToneFrame(128, 200) // DTMF "1": 697 + 1209 Hz
	var pcm []int16
	for i := 0; i < 5; i++ {
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		pcm = append(pcm, out...)
	}
	mag697 := goertzelMag(pcm, 697, mbe.PCMSampleRate)
	mag1209 := goertzelMag(pcm, 1209, mbe.PCMSampleRate)
	magOff := goertzelMag(pcm, 300, mbe.PCMSampleRate) // unrelated bin
	if mag697 < 10*magOff {
		t.Errorf("697 Hz bin (%.0f) not dominant vs 300 Hz (%.0f)", mag697, magOff)
	}
	if mag1209 < 10*magOff {
		t.Errorf("1209 Hz bin (%.0f) not dominant vs 300 Hz (%.0f)", mag1209, magOff)
	}
}

// TestDecodeDualToneKnoxIsSilent: knox / call-alert dual-tone
// indices (b1 ∈ [144, 163]) fall through to the silence branch
// because the public AMBE+2 spec doesn't document their
// frequencies. Pins that contract — operators who want them
// configurable will need to extend ambeDualToneTable.
func TestDecodeDualToneKnoxIsSilent(t *testing.T) {
	d := New()
	// Build a knox-range tone frame directly. b1 high bits come
	// from t-table idx = 0 with bit-4 of low set (b1 = 128 | 16 =
	// 144), matching the boundary of the knox range.
	info := make([]byte, InfoBits)
	for i := 0; i <= 5; i++ {
		info[i] = 1
	}
	info[9] = 1 // sets bit 4 of low, so b1 = 128 + 16 = 144
	frame := make([]byte, FrameBytes)
	for i := 0; i < InfoBits; i++ {
		frame[i/8] |= info[i] << (7 - uint(i)%8)
	}
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// SynthUnvoicedOverlapAdd path fills pcm[0..95] with the §6.4
	// tail; for a fresh decoder state that tail is all-zero.
	for i := 96; i < len(out); i++ {
		if out[i] != 0 {
			t.Errorf("sample[%d] = %d, want 0 (knox dual-tone routes to silence)", i, out[i])
		}
	}
}

// goertzelMag computes the squared magnitude of `targetHz` in the
// supplied int16 PCM stream. Cheap stand-alone implementation
// (not the production toneout one) to keep the test self-contained.
func goertzelMag(samples []int16, targetHz, sampleHz int) float64 {
	if len(samples) == 0 {
		return 0
	}
	k := math.Round(float64(len(samples)) * float64(targetHz) / float64(sampleHz))
	omega := 2 * math.Pi * k / float64(len(samples))
	coeff := 2 * math.Cos(omega)
	var s1, s2 float64
	for _, x := range samples {
		s0 := float64(x)/32768.0 + coeff*s1 - s2
		s2 = s1
		s1 = s0
	}
	return s1*s1 + s2*s2 - coeff*s1*s2
}

// TestDecodeSingleToneFrameAudible: a valid single-tone frame
// (b1 ∈ [5, 122]) synthesises a sinewave at b1·31.25 Hz scaled by
// b2. Pins that the tone path actually produces audio rather than
// routing through silence.
func TestDecodeSingleToneFrameAudible(t *testing.T) {
	d := New()
	// b1 = 12 ⇒ 375 Hz; b2 = 128 ⇒ ~half-scale pre-AGC.
	out, err := d.Decode(singleToneFrame(12, 128))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != mbe.SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), mbe.SamplesPerFrame)
	}
	if agcPeak(out) == 0 {
		t.Fatal("single-tone frame output is fully silent; tone synthesis didn't run")
	}
}

// TestDecodeSingleToneFrameApproximateFrequency: count zero
// crossings in a frame of a known-frequency tone and confirm the
// count matches 2·f·(N/fs) ± 1 (zero crossings per cycle ×
// number of cycles in the frame). For b1 = 16 ⇒ f = 500 Hz and
// N = 160 samples at fs = 8000, expect 2·500·(160/8000) = 20
// zero crossings. Loose tolerance handles the AGC's per-frame
// gain (which doesn't change zero-crossing count anyway) plus
// first-frame phase seeding.
func TestDecodeSingleToneFrameApproximateFrequency(t *testing.T) {
	d := New()
	out, err := d.Decode(singleToneFrame(16, 200))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var crossings int
	for i := 1; i < len(out); i++ {
		if (out[i-1] < 0 && out[i] >= 0) || (out[i-1] >= 0 && out[i] < 0) {
			crossings++
		}
	}
	const wantCrossings = 20
	if crossings < wantCrossings-2 || crossings > wantCrossings+2 {
		t.Errorf("zero crossings = %d, want %d ±2 (b1=16 ⇒ 500 Hz, 160 samples @ 8 kHz)",
			crossings, wantCrossings)
	}
}

// TestDecodeSingleToneFramePhaseContinuity: two consecutive tone
// frames at the same b1 should not click at the frame boundary.
// The maximum absolute difference between consecutive samples
// across the join (last sample of frame 1, first sample of frame
// 2) should be on the same order as the maximum intra-frame
// sample-to-sample difference — a click would manifest as a 2×+
// jump.
func TestDecodeSingleToneFramePhaseContinuity(t *testing.T) {
	d := New()
	frame := singleToneFrame(20, 200) // 625 Hz, mid volume
	out1, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	out2, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("frame 2: %v", err)
	}
	// Largest intra-frame sample-to-sample diff in frame 2 (after
	// AGC has warmed up enough to give us a clean reference).
	var maxIntra int
	for i := 1; i < len(out2); i++ {
		d := int(out2[i]) - int(out2[i-1])
		if d < 0 {
			d = -d
		}
		if d > maxIntra {
			maxIntra = d
		}
	}
	// Boundary diff.
	boundary := int(out2[0]) - int(out1[len(out1)-1])
	if boundary < 0 {
		boundary = -boundary
	}
	// A click would be much larger than the intra-frame max. Allow
	// 3× headroom for AGC level adjustment between the two frames.
	if boundary > 3*maxIntra && boundary > 2000 {
		t.Errorf("boundary diff = %d, intra-frame max = %d (likely click on frame boundary)",
			boundary, maxIntra)
	}
}

// TestDecodeSingleToneFrameVolumeScaling: feed two streams of
// tones at the same frequency but different b2 (volume) values.
// Before the AGC fully converges, the louder b2 should produce a
// higher peak. Take the first frame's peak before the AGC
// normalises everything to TargetPeak.
func TestDecodeSingleToneFrameVolumeScaling(t *testing.T) {
	loud := New()
	quiet := New()
	loudOut, err := loud.Decode(singleToneFrame(12, 255))
	if err != nil {
		t.Fatalf("loud: %v", err)
	}
	quietOut, err := quiet.Decode(singleToneFrame(12, 8))
	if err != nil {
		t.Fatalf("quiet: %v", err)
	}
	loudPeak := agcPeak(loudOut)
	quietPeak := agcPeak(quietOut)
	// First-frame AGC seed lands both at ~TargetPeak (24000), so
	// the peaks themselves are close. What differs is the AGC
	// envelope: loud frame's envelope is ≈ peakAmpScale, quiet
	// frame's is much lower. Confirm both produced non-zero audio
	// (silence-out wouldn't have).
	if loudPeak == 0 || quietPeak == 0 {
		t.Fatalf("expected non-zero peaks: loud=%d quiet=%d", loudPeak, quietPeak)
	}
	loudEnv := loud.agc.Envelope()
	quietEnv := quiet.agc.Envelope()
	if loudEnv <= quietEnv {
		t.Errorf("loud envelope %v should exceed quiet envelope %v", loudEnv, quietEnv)
	}
}

// TestDecodeToneToVoiceTransition: a tone followed by a voice
// frame must produce voice audio (synth pipeline runs) and reset
// the tone-phase accumulator. Pins that the orthogonal voice /
// tone state machines don't interfere.
func TestDecodeToneToVoiceTransition(t *testing.T) {
	d := New()
	if _, err := d.Decode(singleToneFrame(12, 200)); err != nil {
		t.Fatalf("tone: %v", err)
	}
	if d.tonePhase == 0 {
		t.Fatal("tone frame didn't advance tonePhase; test setup invalid")
	}
	// All-zero info ⇒ voice frame (b0 = 0).
	out, err := d.Decode(make([]byte, FrameBytes))
	if err != nil {
		t.Fatalf("voice: %v", err)
	}
	if agcPeak(out) == 0 {
		t.Fatal("voice frame after tone produced silence")
	}
	if d.tonePhase != 0 {
		t.Errorf("voice frame didn't clear tonePhase: got %v", d.tonePhase)
	}
}

// TestResetClearsTonePhase: Reset must zero d.tonePhase so a
// stream re-sync doesn't carry the oscillator state into the
// next call's tone frames.
func TestResetClearsTonePhase(t *testing.T) {
	d := New()
	if _, err := d.Decode(singleToneFrame(12, 200)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if d.tonePhase == 0 {
		t.Fatal("seed didn't advance tonePhase; test setup invalid")
	}
	d.Reset()
	if d.tonePhase != 0 {
		t.Errorf("tonePhase = %v after Reset, want 0", d.tonePhase)
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
