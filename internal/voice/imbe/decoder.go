package imbe

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// Frame parameters per TIA-102.BABA. Every IMBE 4400 frame carries
// 88 information bits over 20 ms of audio at 8 kHz mono. The
// 20 ms / 8 kHz / 160 PCM cadence + the bad-frame replay constants
// + the AGC config live on the shared internal/voice/mbe package
// so AMBE+2 can consume the same primitives.
const (
	// InfoBits is the per-frame information-bit count after channel
	// FEC has been applied + verified.
	InfoBits = 88

	// FrameBytes is InfoBits packed MSB-first into octets, rounded up.
	FrameBytes = 11

	// VocoderName is the registry key the daemon resolves at startup.
	// This is the canonical "imbe" name; the pure-Go decoder is the
	// sole IMBE backend in default builds.
	VocoderName = "imbe"
)

// recoveryRampFactors scales the synthesised amplitudes M[1..L] for
// the first len(recoveryRampFactors) good frames after a bad-streak
// state clear. The ramp 0.4 → 0.7 → 1.0 across 60 ms (3 frames at
// 20 ms each) eases the listener back in after silence rather than
// jumping straight back to full amplitude — the §6.3 voiced
// harmonic generator's per-frame amp interpolation provides 20 ms
// of natural fade, but a 60 ms envelope ramp on top is more
// musically natural after extended bit loss. The AGC is frozen
// during recovery so the attenuation is audible (and not
// compensated by the envelope tracker).
//
// Phase-aware: PrevPhase is zeroed by the state clear, so the
// voiced harmonic generator starts every harmonic at phase 0.
// The amplitude-tilt 0 → factor·M[l] keeps sample 0 at exactly 0
// regardless of phase coherence, so there's no zero-crossing
// click. Subsequent samples mix harmonics at phases that diverge
// quickly per their ω₀·l·n term, decorrelating the energy
// naturally.
var recoveryRampFactors = [3]float64{0.4, 0.7, 1.0}

// IdleToneRunThreshold drives idle-tone suppression. An unmodulated/idle
// voice-channel carrier (or the demod warm-up/decay transient at the very
// start/end of a transmission) produces a dibit stream the per-vector FEC
// resolves to a degenerate low-b_0 frame (see UnpackHeader /
// Header.IdleTone) — one the synthesizer would otherwise render as an
// audible ~350 Hz buzz. Field captures show this as runs of low-b_0
// frames bracketing real speech, and whole dead-key "calls" that are 100%
// this frame; real speech never sustains the b_0 ≤ IdleToneMaxB0 corner
// across frames (a 344–405 Hz fundamental held flat is not speech), so a
// *run* of them is an unambiguous idle signature.
//
// IdleToneRunThreshold is how many consecutive low-b_0 frames must be
// seen before muting kicks in. 2 means a lone low-b_0 frame inside speech
// is never muted (its run is broken by the surrounding ordinary frames);
// at most one 20 ms buzz leaks at a tone-run onset before the mute
// engages. The run is broken only by a genuine non-idle frame (an
// ordinary b_0, a silence-window frame, or a hard FEC error) — NOT by
// intra-run variation in the idle frame's parameter bits, since a noisy
// idle carrier can resolve to differing b_0 ≤ IdleToneMaxB0 frames.
const IdleToneRunThreshold = 2

// phaseRngSeedOffset offsets the §6.3 voiced-phase-dispersion source from
// the unvoiced-noise source so the two deterministic streams don't march
// in lockstep. Any fixed non-zero value works; this one is arbitrary.
const phaseRngSeedOffset = 0x5DEECE66D

// Decoder is the pure-Go IMBE 4400 decoder. It owns one
// mbe.SynthState (cross-frame log2(Ml) prediction memory + voiced
// phase + amp memory + §6.4 OA tail), one math/rand source for
// the unvoiced excitation noise, one *mbe.AGC, and a one-frame
// cache of the last-good params for the frame-repeat path — all
// per-call so concurrent calls on different decoders don't share
// state.
//
// Decode() runs the full TIA-102.BABA pipeline:
//   - bytes → 88 info bits
//   - UnpackParams: §5.3 / §5.4 / Annex E (this package)
//   - p.MBE(): port to the shared mbe.Params shape
//   - mbe.PredictLog2Ml: §6.1 cross-frame prediction
//   - mbe.AmplitudesFromLog2Ml: log2(Ml) → linear Ml
//   - mbe.EnhanceAmplitudes: §6.2 spectral-amplitude enhancement
//   - recovery ramp (if applicable): scale M by recoveryRampFactors
//   - mbe.SynthVoicedDispersed: §6.3 voiced harmonic generator with
//     voicing-scaled upper-harmonic phase regeneration
//   - mbe.SynthUnvoicedOverlapAdd: §6.4 unvoiced FFT excitation + OA
//   - mbe.SynthState.Update{Log2Ml,VoicedState}: roll state forward
//     (the voiced phase memory stays coherent — see synthFrame)
//   - mbe.AGC.Apply: per-frame fast-attack / slow-release peak
//     tracker scaling to AGCConfig.TargetPeak, then int16 clip
//
// On a bad frame (UnpackParams returns ErrInvalidFundamental, etc.),
// Decode replays the last good frame's amplitudes scaled by
// mbe.BadFrameAttenuation^badFrameCount. After mbe.MaxBadFrames
// consecutive bad frames the cache is cleared and Decode returns
// silence so an extended bad streak fades naturally instead of
// looping the same envelope forever. When good frames return after
// such a clear, the recovery ramp eases the synthesiser back in
// over 60 ms.
type Decoder struct {
	state mbe.SynthState
	rng   *rand.Rand
	// phaseRng is a SEPARATE seeded source for the §6.3 voiced-phase
	// dispersion draws. Keeping it independent of rng means the unvoiced
	// excitation noise stream is byte-identical with or without the phase
	// dispersion, so callers + tests that pin noise-driven output are
	// unaffected. Both are deterministic for a given constructor seed.
	phaseRng *rand.Rand
	agc      *mbe.AGC
	// dc is the DC-removal high-pass applied to the synthesized PCM before
	// the AGC (matches OP25 dc_rmv.cc). Continuous across frames; reset only
	// on stream re-sync via Reset.
	dc mbe.DCBlock

	// smoother holds the IMBE adaptive-smoothing memory (running channel
	// error rate + local energy). frameErrs is the FEC corrected-bit count
	// for the next Decode, supplied via SetFrameErrors (voice.ErrorAware);
	// 0 when the caller doesn't track it, leaving smoothing inert.
	smoother  mbe.Smoother
	frameErrs int

	// One-frame cache for the frame-repeat path. Holds the shared
	// mbe.Params shape since the synthesis path consumes only that
	// subset (W0/L/Silent/Vl/Tl); the IMBE-specific Gm/Cik
	// intermediates are not needed for replay.
	lastGoodParams mbe.Params
	lastGoodLog2M  [mbe.MaxL + 1]float64
	lastGoodM      [mbe.MaxL + 1]float64

	// Consecutive bad-frame count. Increments once per replay,
	// resets to 0 on every good non-silent frame.
	badFrameCount int

	// segFadeRemaining is the number of upcoming good frames to fade in
	// (post-AGC) at a segment (re)start — stream onset and the first good
	// frame after a silence-window or idle-mute frame. Without it the first
	// synthesized frame of a segment starts from reset state (all harmonics
	// phase-aligned at n=0, a broadband HF impulse) and renders at full AGC
	// target, producing the reported high-pitched onset squeaks. Armed to
	// len(recoveryRampFactors) at those restarts; the post-AGC ramp reuses
	// recoveryRampFactors. Distinct from recoveryFramesRemaining (the
	// bad-streak recovery, which fades via pre-synth amplitude scaling with
	// the AGC frozen) so the two never double-apply.
	segFadeRemaining int

	// recoveryFramesRemaining is the number of upcoming good frames
	// that should run through the recovery ramp. Set to
	// len(recoveryRampFactors) by the bad-streak budget-exhaust
	// path; decremented each good frame; zero outside recovery.
	// The ramp index is len(recoveryRampFactors) -
	// recoveryFramesRemaining so the first recovery frame uses the
	// smallest factor and the last uses 1.0 (full amplitude).
	recoveryFramesRemaining int

	// lowB0RunCount is the number of consecutive low-b_0 idle-tone
	// frames seen so far (see IdleToneRunThreshold). Managed solely in
	// Decode's pre-switch accounting block — NOT in clearLastGood — so
	// the mute path doesn't reset its own run mid-stream.
	lowB0RunCount int

	// voiceStarted is true once a real (non-idle, non-silent) voiced
	// frame has been synthesized since the last (re)start. Until then the
	// stream is at a transmission onset, where the idle-tone mute engages
	// on the very first idle frame instead of leaking one (there is no
	// speech to protect). Re-armed (set false) at transmission-gap
	// boundaries — silence window, bad-streak clear, Reset — so each split
	// recording opens cleanly. See the muteTone gate in Decode.
	voiceStarted bool

	// stats accumulates the per-call VoiceStats summary. Cleared only by
	// ResetStats (the recorder calls it at call/segment boundaries) — NOT
	// by Reset, so a mid-call re-sync doesn't discard the running totals.
	stats callStats
}

// callStats is the decoder's running per-call telemetry, folded into a
// voice.VoiceStats by VoiceStats(). Pitch/L/voicing/gain means are taken
// over good non-silent frames (goodFrames); amplitude health (RMS, peak,
// clip) is taken over every emitted sample.
type callStats struct {
	frames    int
	voiced    int
	unvoiced  int
	silent    int
	idleMuted int
	bad       int
	repeated  int

	// b_0 range + first-frame capture for the all-idle diagnostic. b0Seen
	// guards the first-write of min/max; firstFrameHex holds the first
	// frame's raw bytes as hex.
	b0Seen        bool
	b0Min         int
	b0Max         int
	firstFrameHex string

	goodFrames    int
	f0Sum         float64
	f0Min         float64
	f0Max         float64
	lSum          float64
	voicedFracSum float64
	gainSum       float64
	gainMin       float64
	gainMax       float64

	sumSq          float64
	sampleCount    int
	postPeak       float64
	maxPreClipPeak float64
	clipSamples    int
}

// New returns a fresh Decoder. The unvoiced-excitation noise source
// is seeded from a fixed default so two decoders constructed via
// New() produce byte-identical output for the same frame stream
// (useful for tests + reproducibility). Production callers wanting
// genuinely-random noise across runs should use NewWithSeed with a
// time-derived seed.
func New() *Decoder {
	return NewWithSeed(0)
}

// NewWithSeed constructs a Decoder with an explicit seed for the
// internal noise source. Lets tests pin output across runs and lets
// production callers spread noise across decoders so two parallel
// calls don't share the same noise stream. AGC parameters use
// mbe.DefaultAGCConfig.
func NewWithSeed(seed int64) *Decoder {
	return NewWithConfig(seed, mbe.DefaultAGCConfig())
}

// NewWithConfig constructs a Decoder with an explicit noise seed +
// AGC configuration. Zero-value fields in cfg fall back to
// mbe.DefaultAGCConfig values, so callers can override only the
// parameters they care about (e.g. mbe.AGCConfig{TargetPeak: 16000}
// to drop the output level by ~3 dB without re-tuning attack/release).
func NewWithConfig(seed int64, cfg mbe.AGCConfig) *Decoder {
	return &Decoder{
		rng: rand.New(rand.NewSource(seed)),
		// Offset the phase source so it does not march in lockstep with the
		// noise source while staying deterministic for a given seed.
		phaseRng: rand.New(rand.NewSource(seed + phaseRngSeedOffset)),
		agc:      mbe.NewAGC(cfg),
		// Fade in the first frames of the very first segment (stream onset).
		segFadeRemaining: len(recoveryRampFactors),
	}
}

// Name returns the registry key. Matches VocoderName.
func (d *Decoder) Name() string { return VocoderName }

// FrameSize returns the per-frame input byte count (11 bytes / 88
// bits, packed MSB-first).
func (d *Decoder) FrameSize() int { return FrameBytes }

// Decode reads 88 info bits (post-FEC, MSB-first packed) from frame,
// runs the full TIA-102.BABA pipeline, and returns 160 int16 PCM
// samples at 8 kHz.
//
// Frame disposition:
//
//   - good non-silent frame: full synthesis pipeline; cache p +
//     log2M + M for the next bad frame's repeat path; reset
//     badFrameCount.
//   - silence-window frame (b_0 ∈ [216, 219]): emit §6.4 OA tail
//     fade-out into pcm[0..95]; reset SynthState + last-good cache
//   - badFrameCount; AGC envelope preserved across the silence.
//   - bad frame (UnpackParams error) with cached last-good frame
//     and badFrameCount < mbe.MaxBadFrames: replay last-good params
//     with M scaled by mbe.BadFrameAttenuation^badFrameCount; AGC
//     freezes so the attenuation is audible (signals signal loss).
//   - bad frame with no cache, or after mbe.MaxBadFrames consecutive
//     bad frames: emit silence; clear last-good cache + reset
//     SynthState; AGC envelope preserved.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("imbe: frame must be %d bytes (88 bits), got %d", FrameBytes, len(frame))
	}

	info := unpackInfoBits(frame)
	out := make([]int16, mbe.SamplesPerFrame)
	pcm := make([]float64, mbe.SamplesPerFrame)
	d.stats.frames++

	// Track the raw b_0 range + capture the first frame for the all-idle
	// diagnostic (see callStats / VoiceStats). Done for every frame,
	// before the disposition switch, so even bad/idle frames are counted.
	b0 := int(b0FromInfo(info))
	if !d.stats.b0Seen {
		d.stats.b0Seen = true
		d.stats.b0Min, d.stats.b0Max = b0, b0
		d.stats.firstFrameHex = hex.EncodeToString(frame)
	} else {
		if b0 < d.stats.b0Min {
			d.stats.b0Min = b0
		}
		if b0 > d.stats.b0Max {
			d.stats.b0Max = b0
		}
	}

	// Consume the corrected-bit count set by SetFrameErrors (0 if the caller
	// doesn't supply one) and advance the adaptive-smoothing error-rate
	// estimate. muteByER is true when the channel is so degraded the frame
	// should be silenced.
	correctedBits := d.frameErrs
	d.frameErrs = 0
	muteByER := d.smoother.UpdateErrorRate(correctedBits)

	p, err := UnpackParams(info)

	// Idle-tone run accounting. Done before the switch (a Go switch
	// can't fall through to the post-switch good-frame block) so a
	// below-threshold tone frame still synthesizes while the run count
	// advances. Kept here only — see clearLastGood — so the mute case
	// below doesn't reset the run it just acted on.
	tone := err == nil && !p.Silent && p.IdleTone
	if tone {
		d.lowB0RunCount++
	} else {
		// Any non-idle frame (ordinary b_0, silence, or hard error)
		// breaks the run.
		d.lowB0RunCount = 0
	}
	// At a transmission onset (no real voice synthesized yet) an idle
	// frame is pure warm-up buzz with no speech to protect, so mute it
	// immediately instead of leaking the first one. Once speech has
	// started, fall back to the run threshold so a lone idle frame inside
	// speech still leaks once rather than clipping audio.
	muteTone := tone && (d.lowB0RunCount >= IdleToneRunThreshold || !d.voiceStarted)

	switch {
	case err != nil && d.lastGoodParams.L > 0 && d.badFrameCount < mbe.MaxBadFrames:
		// Frame repeat: replay the last-good params with progressive
		// per-frame attenuation. Synthesis still runs (so OA tail +
		// phase memory continue rolling forward), AGC freezes so the
		// attenuation is audible.
		d.badFrameCount++
		atten := math.Pow(mbe.BadFrameAttenuation, float64(d.badFrameCount))
		repeatedM := d.lastGoodM
		for l := 1; l <= d.lastGoodParams.L; l++ {
			repeatedM[l] *= atten
		}
		d.synthFrame(d.lastGoodParams, &d.lastGoodLog2M, &repeatedM, pcm)
		d.applyOutput(pcm, out, true)
		d.stats.repeated++
		d.accumStats(out)
		return out, nil

	case err != nil:
		// Bad frame with no cache, or bad-frame budget exhausted:
		// emit silence + clear cache + reset SynthState. AGC envelope
		// preserved so audio level is consistent when good frames
		// return. Arm the recovery ramp so the first
		// len(recoveryRampFactors) good frames after the clear fade
		// back in at 0.4 → 0.7 → 1.0 amplitude rather than jumping
		// straight to full level.
		d.state.Reset()
		d.dc.Reset()
		d.clearLastGood()
		d.recoveryFramesRemaining = len(recoveryRampFactors)
		// The bad-streak recovery ramp is this segment's fade-in; don't also
		// run the post-AGC segment fade (avoid double-fading).
		d.segFadeRemaining = 0
		// Bad-streak clear ends the current voiced segment — re-arm the
		// onset idle-mute so the next over opens without a buzz leak.
		d.voiceStarted = false
		// Silence frame: no synthesized speech to DC-filter (the blocker was
		// reset above so the next over starts clean); go straight to the AGC.
		d.agc.Apply(pcm, out, true)
		d.stats.bad++
		d.accumStats(out)
		return out, nil

	case p.Silent:
		// b_0 ∈ [216, 219]: explicit silence indicator. Run the §6.4
		// overlap-add with no new noise so the prev-frame unvoiced
		// tail still fades into pcm[0..95] (no click on the silence
		// boundary), then reset SynthState + last-good cache so the
		// next non-silent frame starts from a clean baseline.
		mbe.SynthUnvoicedOverlapAdd(&d.state, p.MBE(), nil, nil, pcm)
		d.state.Reset()
		d.dc.Reset()
		d.clearLastGood()
		// Silence window marks a transmission gap — re-arm the onset
		// idle-mute so the next over opens without a buzz leak, and fade the
		// next segment in so its first frame doesn't blip at full scale.
		d.voiceStarted = false
		d.segFadeRemaining = len(recoveryRampFactors)
		d.recoveryFramesRemaining = 0
		// Silence/OA-tail frame: bypass the (reset) DC blocker so the tail's
		// non-overlap region stays clean; go straight to the AGC.
		d.agc.Apply(pcm, out, true)
		d.stats.silent++
		d.accumStats(out)
		return out, nil

	case muteTone:
		// Sustained idle-carrier tone: mute exactly like the silence
		// window so the ~350 Hz buzz never reaches the WAV / live
		// audio. The prev-frame unvoiced tail still fades into
		// pcm[0..95] (no click), then state + last-good cache reset so
		// the next real frame starts clean. lowB0RunCount is preserved
		// (managed above) so the rest of the run keeps muting.
		mbe.SynthUnvoicedOverlapAdd(&d.state, p.MBE(), nil, nil, pcm)
		d.state.Reset()
		d.dc.Reset()
		d.clearLastGood()
		// Idle-tone gap — fade the next segment in so it doesn't blip.
		d.segFadeRemaining = len(recoveryRampFactors)
		d.recoveryFramesRemaining = 0
		// Idle-tone mute / OA-tail frame: bypass the (reset) DC blocker.
		d.agc.Apply(pcm, out, true)
		d.stats.idleMuted++
		d.accumStats(out)
		return out, nil
	}

	// Good non-silent frame. A real (non-idle) voiced frame marks the
	// start of speech, so subsequent idle frames revert to the run-
	// threshold mute (a lone mid-speech idle leaks once rather than
	// clipping audio). A below-threshold idle frame still synthesizes
	// here but does not count as speech onset.
	d.badFrameCount = 0
	if !p.IdleTone {
		d.voiceStarted = true
	}
	m := p.MBE()
	var log2M [mbe.MaxL + 1]float64
	mbe.PredictLog2Ml(&d.state, m, &log2M)

	var M [mbe.MaxL + 1]float64
	mbe.AmplitudesFromLog2Ml(&log2M, m.L, &M)
	mbe.EnhanceAmplitudes(m, &M)

	// IMBE adaptive smoothing: on a degraded channel, cap error-induced
	// amplitude spikes and reclaim obviously-voiced harmonics (modifies M and
	// m.Vl in place); a clean channel is left untouched. A frame whose error
	// rate exceeds the mute threshold is silenced by zeroing the amplitudes,
	// which the voiced generator's amplitude tilt fades out cleanly.
	d.smoother.Smooth(&m, &M, correctedBits)
	if muteByER {
		for l := 1; l <= m.L; l++ {
			M[l] = 0
		}
	}

	// Recovery ramp after a bad-streak state clear: scale the post-
	// enhancement amplitudes by recoveryRampFactors over the next
	// len(recoveryRampFactors) good frames so the listener eases
	// back in. The ramped M flows through synthesis (so SynthState's
	// PrevMl reflects the actually-synthesised amplitudes) and is
	// cached as lastGoodM for the bad-frame replay path. That keeps
	// a fresh bad streak mid-recovery consistent with how the
	// previous frame actually sounded — replaying from the
	// pre-ramp full amplitude would create an audible jump.
	recoveryFreezeAGC := false
	if d.recoveryFramesRemaining > 0 {
		idx := len(recoveryRampFactors) - d.recoveryFramesRemaining
		factor := recoveryRampFactors[idx]
		for l := 1; l <= m.L; l++ {
			M[l] *= factor
		}
		d.recoveryFramesRemaining--
		// Freeze the AGC during the ramp so the attenuation is
		// audible — without freeze the envelope tracker would scale
		// the output back up to TargetPeak and the listener would
		// hear no fade-in.
		recoveryFreezeAGC = true
	}

	d.synthFrame(m, &log2M, &M, pcm)

	// Cache for the frame-repeat path on a future bad frame.
	d.lastGoodParams = m
	d.lastGoodLog2M = log2M
	d.lastGoodM = M

	d.applyOutput(pcm, out, recoveryFreezeAGC)
	// Segment-start fade-in (post-AGC): attenuate the first frames of a new
	// segment so the onset (a phase-aligned, HF-weighted impulse synthesized
	// from reset state) doesn't blip at full scale — the high-pitched-squeak
	// fix. Applied post-AGC so it's audible regardless of envelope state and
	// works at true onset (env==0 still seeds normally). Skipped while the
	// bad-streak recovery ramp is handling its own fade.
	if !recoveryFreezeAGC && d.segFadeRemaining > 0 {
		f := recoveryRampFactors[len(recoveryRampFactors)-d.segFadeRemaining]
		for i, s := range out {
			out[i] = int16(float64(s) * f)
		}
		d.segFadeRemaining--
	}
	d.accumGood(m, d.agc.LastGain())
	d.accumStats(out)
	return out, nil
}

// synthFrame runs the §6.3 voiced + §6.4 unvoiced overlap-add legs
// of the pipeline and rolls SynthState forward by log2M / M. Used
// by both the good-frame path and the bad-frame replay path so the
// two share identical synthesis behavior — the only difference is
// what M values get fed in.
func (d *Decoder) synthFrame(p mbe.Params, log2M *[mbe.MaxL + 1]float64, M *[mbe.MaxL + 1]float64, pcm []float64) {
	// §6.3 voiced-phase regeneration. Match the mbelib reference
	// PHIl[l] = PSIl[l] + (numUv * rand_phase()) / L for the upper
	// harmonics: draw a per-harmonic uniform phase offset in [−π, π) from
	// the decoder's separate phase source, scaled by the unvoiced-harmonic
	// fraction numUv/L. A mostly-voiced frame (small numUv — e.g. a
	// high-pitch female vowel) therefore gets near-zero dispersion and
	// stays intelligible; a noise-dominated frame gets up to full [−π, π).
	// SynthVoicedDispersed applies the offset to the synthesis phase only
	// (l > L/4, voiced), and UpdateVoicedState advances the phase memory
	// coherently — so the offset never accumulates into a random walk. The
	// draw runs for every harmonic each frame so the phase stream advances
	// at a fixed rate regardless of L (and, being a separate source, never
	// perturbs the unvoiced-noise stream below).
	numVoiced := 0
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] == 1 {
			numVoiced++
		}
	}
	var dispScale float64
	if p.L > 0 {
		dispScale = float64(p.L-numVoiced) / float64(p.L)
	}
	var phaseDisp [mbe.MaxL + 1]float64
	for l := 1; l <= mbe.MaxL; l++ {
		phaseDisp[l] = (d.phaseRng.Float64()*2 - 1) * math.Pi * dispScale
	}

	mbe.SynthVoicedDispersed(&d.state, p, M, pcm, &phaseDisp)
	noise := make([]float64, mbe.UnvoicedFFTSize)
	for i := range noise {
		noise[i] = d.rng.NormFloat64()
	}
	mbe.SynthUnvoicedOverlapAdd(&d.state, p, M, noise, pcm)
	d.state.UpdateLog2Ml(p, log2M)
	d.state.UpdateVoicedState(p, M)
}

// pcmSampleRateHz is the IMBE output PCM rate (8 kHz, 160 samples /
// 20 ms frame). Used to convert the fundamental W0 (rad/sample) to Hz for
// the VoiceStats pitch summary.
const pcmSampleRateHz = 8000

// applyOutput runs the DC-removal high-pass over the synthesized PCM and
// then the AGC into out. Used by the paths that emit real synthesized
// speech (the good-frame path and the frame-repeat path) so the DC filter
// runs continuously over voiced audio and the AGC sizes against the DC-free
// signal. The silence / idle-mute / bad-streak paths bypass it (and Reset
// the blocker) since they emit no speech. freezeEnvelope is forwarded to
// the AGC.
func (d *Decoder) applyOutput(pcm []float64, out []int16, freezeEnvelope bool) {
	d.dc.Process(pcm)
	d.agc.Apply(pcm, out, freezeEnvelope)
}

// accumStats folds the just-applied frame's AGC telemetry and output
// samples into the per-call stats. Call immediately after agc.Apply on
// every return path so the amplitude-health numbers cover all output.
func (d *Decoder) accumStats(out []int16) {
	d.stats.clipSamples += d.agc.LastClipSamples()
	if p := d.agc.LastMaxPreClipPeak(); p > d.stats.maxPreClipPeak {
		d.stats.maxPreClipPeak = p
	}
	for _, s := range out {
		f := float64(s)
		d.stats.sumSq += f * f
		if a := math.Abs(f); a > d.stats.postPeak {
			d.stats.postPeak = a
		}
	}
	d.stats.sampleCount += len(out)
}

// accumGood folds a good non-silent frame's pitch / harmonic count /
// voicing / applied gain into the per-call stats.
func (d *Decoder) accumGood(p mbe.Params, gain float64) {
	numVoiced := 0
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] == 1 {
			numVoiced++
		}
	}
	if p.L > 0 && 2*numVoiced >= p.L {
		d.stats.voiced++
	} else {
		d.stats.unvoiced++
	}
	f0 := p.W0 * pcmSampleRateHz / (2 * math.Pi)
	var vf float64
	if p.L > 0 {
		vf = float64(numVoiced) / float64(p.L)
	}
	if d.stats.goodFrames == 0 {
		d.stats.f0Min, d.stats.f0Max = f0, f0
		d.stats.gainMin, d.stats.gainMax = gain, gain
	} else {
		if f0 < d.stats.f0Min {
			d.stats.f0Min = f0
		}
		if f0 > d.stats.f0Max {
			d.stats.f0Max = f0
		}
		if gain < d.stats.gainMin {
			d.stats.gainMin = gain
		}
		if gain > d.stats.gainMax {
			d.stats.gainMax = gain
		}
	}
	d.stats.goodFrames++
	d.stats.f0Sum += f0
	d.stats.lSum += float64(p.L)
	d.stats.voicedFracSum += vf
	d.stats.gainSum += gain
}

// VoiceStats returns the accumulated per-call telemetry and ok == false
// when no frames have been decoded yet. Implements voice.StatProvider so
// the recorder + the offline `gophertrunk decode` tool can surface a
// per-call audio-quality summary.
func (d *Decoder) VoiceStats() (voice.VoiceStats, bool) {
	s := d.stats
	if s.frames == 0 {
		return voice.VoiceStats{}, false
	}
	vs := voice.VoiceStats{
		Frames:         s.frames,
		Voiced:         s.voiced,
		Unvoiced:       s.unvoiced,
		Silent:         s.silent,
		IdleMuted:      s.idleMuted,
		Bad:            s.bad,
		Repeated:       s.repeated,
		MinB0:          -1,
		MaxB0:          -1,
		FirstFrameHex:  s.firstFrameHex,
		MaxPreClipPeak: s.maxPreClipPeak,
		ClipSamples:    s.clipSamples,
		TotalSamples:   s.sampleCount,
	}
	if s.b0Seen {
		vs.MinB0 = s.b0Min
		vs.MaxB0 = s.b0Max
	}
	if s.goodFrames > 0 {
		g := float64(s.goodFrames)
		vs.MeanF0Hz = s.f0Sum / g
		vs.MinF0Hz = s.f0Min
		vs.MaxF0Hz = s.f0Max
		vs.MeanL = s.lSum / g
		vs.MeanVoicedFrac = s.voicedFracSum / g
		vs.MeanAGCGain = s.gainSum / g
		vs.MinAGCGain = s.gainMin
		vs.MaxAGCGain = s.gainMax
	}
	if s.sampleCount > 0 {
		rms := math.Sqrt(s.sumSq / float64(s.sampleCount))
		vs.OutputRMS = rms
		if rms > 0 {
			vs.CrestFactor = s.postPeak / rms
		}
	}
	return vs, true
}

// SetFrameErrors supplies the FEC corrected-bit count for the NEXT Decode
// call, driving the adaptive-smoothing error-rate estimate. Implements
// voice.ErrorAware; the recorder calls it (for P25 Phase 1) immediately
// before Decode. Callers that don't track errors simply never call it, and
// smoothing stays inert.
func (d *Decoder) SetFrameErrors(correctedBits int) { d.frameErrs = correctedBits }

// ResetStats clears the per-call telemetry. The recorder calls it at
// call / segment boundaries. Kept separate from Reset (which clears
// synthesis state on mid-call re-sync) so a re-sync doesn't discard the
// call's running totals.
func (d *Decoder) ResetStats() { d.stats = callStats{} }

// clearLastGood resets the frame-repeat cache + bad-frame counter.
// Called on silence-window frames, on the bad-frame budget being
// exceeded, and from the public Reset.
func (d *Decoder) clearLastGood() {
	d.lastGoodParams = mbe.Params{}
	d.lastGoodLog2M = [mbe.MaxL + 1]float64{}
	d.lastGoodM = [mbe.MaxL + 1]float64{}
	d.badFrameCount = 0
}

// unpackInfoBits expands 11 bytes (MSB-first) into an 88-element
// 0/1 byte slice — the format UnpackHeader / UnpackParams expects.
func unpackInfoBits(frame []byte) []byte {
	info := make([]byte, InfoBits)
	for i := 0; i < InfoBits; i++ {
		info[i] = (frame[i/8] >> (7 - uint(i)%8)) & 1
	}
	return info
}

// Reset clears all per-call synthesis state — the cross-frame
// log-amplitude prediction history, the voiced harmonic phase +
// amplitude memory, the §6.4 overlap-add tail, the AGC envelope,
// and the frame-repeat cache + bad-frame counter. Callers invoke
// it on stream re-sync (e.g., a frame-loss event from the upstream
// P25 LDU decoder) so the next frame starts from a clean baseline.
// The noise source is intentionally not re-seeded — noise
// reproducibility is a constructor concern (New / NewWithSeed),
// not a per-call concern.
func (d *Decoder) Reset() {
	d.state.Reset()
	d.agc.Reset()
	d.dc.Reset()
	d.smoother.Reset()
	d.frameErrs = 0
	d.clearLastGood()
	d.recoveryFramesRemaining = 0
	d.segFadeRemaining = len(recoveryRampFactors)
	d.lowB0RunCount = 0
	d.voiceStarted = false
}

// Close releases any resources held by the decoder. The pure-Go
// implementation holds none, so this is always a no-op.
func (d *Decoder) Close() error { return nil }

// Compile-time check that Decoder satisfies voice.Vocoder +
// voice.StatProvider.
var (
	_ voice.Vocoder      = (*Decoder)(nil)
	_ voice.StatProvider = (*Decoder)(nil)
	_ voice.ErrorAware   = (*Decoder)(nil)
)

func init() {
	voice.DefaultRegistry.Register(VocoderName, func() (voice.Vocoder, error) {
		return New(), nil
	})
}
