package mbe

import "math"

// Bad-frame replay constants shared across MBE-family decoders.
// When the upstream protocol layer's FEC reports a slip and the
// decoder cannot recover an MBE Params from the frame bits, the
// decoder replays the cached last-good params with a per-frame
// attenuation; after MaxBadFrames consecutive replays the cache
// clears and the decoder emits silence.
const (
	// MaxBadFrames is the number of consecutive bad frames the
	// frame-repeat path replays before giving up and emitting
	// silence + clearing state. The TIA-102.BABA spec range is
	// 1..6; mbelib uses 6, which gives the upstream FEC roughly
	// 120 ms of grace before the audio path drops out completely.
	// After MaxBadFrames bad frames the cumulative attenuation is
	// BadFrameAttenuation^6 ≈ 0.118 — quiet enough that an extended
	// bad streak fades naturally rather than looping the same
	// envelope.
	MaxBadFrames = 6

	// BadFrameAttenuation is the per-frame multiplier applied to
	// the cached last-good amplitudes during a frame-repeat. A
	// single bad frame plays at 70% of the prev good frame's
	// amplitudes; six in a row taper to ~12%. Balance between
	// hiding the FEC slip (no abrupt mute) and signalling the
	// listener that signal is degrading (audible volume drop).
	BadFrameAttenuation = 0.7
)

// AGCConfig holds the per-frame AGC parameters. The synthesizer's
// float output magnitude is stable per-frame (the §6.2
// enhancement's R_M0-preserving rescale holds total energy
// constant across a frame) but varies wildly between frames
// depending on Tl, voicing, and the §6.4 noise draw. Without an
// AGC every frame would be either clipped (loud frames) or
// near-silent (quiet frames). The AGC tracks the per-frame peak
// with fast-attack / slow-release smoothing, then scales each
// frame so the smoothed envelope hits TargetPeak.
//
// Zero-value fields fall back to DefaultAGCConfig values, so a
// caller can override only the field they care about (e.g.
// AGCConfig{TargetPeak: 16000} to drop output level by 4 dB).
//
// The current AGC is a level-only design — disabling it entirely
// means dropping back to a constant gain, which we don't expose as
// a separate option (callers wanting that can pass equal Attack +
// Release to fully smooth the envelope).
type AGCConfig struct {
	// TargetPeak is the post-AGC peak amplitude target in int16
	// units. Default 18000 (~5.2 dB below int16 max) so ordinary
	// frames land well clear of the soft-limiter knee (26213) and
	// only true transients ever engage it. Measured across the field
	// C4FM + CQPSK captures, 18000 holds limited-sample rate ≈0.01%
	// while keeping a natural crest factor (~9).
	TargetPeak float64

	// Attack is the per-frame envelope rise coefficient. Standard
	// IIR coefficient: 1.0 = instant tracking, 0.0 = no update.
	// Default 0.5 — fast-attack / slow-release, the classic envelope
	// follower. Fast attack catches loud onsets so they don't
	// overshoot into the limiter; the slow Release preserves
	// inter-frame dynamics. A confound-free sweep over the field
	// captures peaks crest factor at attack≈0.5 (~9, matching a
	// reference decoder); both much slower (0.05, onset overshoot
	// flattens via the limiter) and instant (1.0, per-frame
	// renormalisation flattens) give lower crest.
	Attack float64

	// Release is the per-frame envelope fall coefficient. Smaller
	// than Attack so gain ramps back up slowly during quiet
	// passages — standard AGC behavior that keeps speech
	// intelligible without pumping. Default 0.02.
	Release float64

	// MinGain bounds the lowest gain the AGC can apply (it may
	// attenuate — the IMBE/AMBE synthesiser already emits near-int16
	// scale, so a loud frame needs gain ≈1, not amplification).
	// Default 0.05. NOTE: a too-high floor here is catastrophic — a
	// floor of 10 forced ≥10× amplification of already-loud frames
	// straight into the limiter, which was the root cause of the
	// "robotic"/over-driven field reports.
	MinGain float64

	// MaxGain bounds the highest gain the AGC can apply, preventing
	// silence from being amplified to full scale by a stale low
	// envelope. Default 1e5. NOTE: this is a coarse bound only — a
	// genuinely quiet (but valid) synthesised frame can have a peak of a
	// few units and legitimately needs a gain of several thousand to reach
	// TargetPeak, so this cap cannot be tightened much without starving
	// quiet speech. The per-frame output ceiling in Apply (not MaxGain) is
	// what stops a stale low envelope from over-amplifying a *louder*
	// following frame into the limiter.
	MaxGain float64

	// NoiseFloor is the per-frame peak threshold below which the
	// envelope skips its update. Lets a §6.4 OA tail fade-out into
	// silence without dragging the envelope down. Default 1e-3.
	NoiseFloor float64
}

// DefaultAGCConfig returns the AGC parameters the IMBE / AMBE+2
// decoders use when constructed without an explicit override.
// Callers wanting a partial override can take this struct, mutate
// individual fields, and pass it back; or pass an AGCConfig{}
// with only the field they care about set — zero-value fields
// backfill from the defaults via WithDefaults.
func DefaultAGCConfig() AGCConfig {
	return AGCConfig{
		TargetPeak: 18000.0,
		Attack:     0.5,
		Release:    0.02,
		MinGain:    0.05,
		MaxGain:    1e5,
		NoiseFloor: 1e-3,
	}
}

// Soft-limiter geometry. Below softLimitKnee samples pass through
// linearly (the common case once TargetPeak leaves headroom); above the
// knee the excess is compressed with tanh so the output approaches — but
// never reaches — the int16 rail. This replaces the old hard clip at
// ±32767, whose abrupt corners injected harmonic distortion that read as
// "robotic" whenever a transient (or the old over-hot gain) exceeded full
// scale.
const (
	softLimitRail = 32767.0
	softLimitKnee = 0.80 * softLimitRail // 26213
)

// softLimit maps x into (−32767, 32767), passing |x| ≤ knee through
// unchanged and tanh-compressing the region above the knee. limited
// reports whether the knee was engaged (used for clip telemetry).
func softLimit(x float64) (y float64, limited bool) {
	a := math.Abs(x)
	if a <= softLimitKnee {
		return x, false
	}
	over := (a - softLimitKnee) / (softLimitRail - softLimitKnee)
	comp := softLimitKnee + (softLimitRail-softLimitKnee)*math.Tanh(over)
	if x < 0 {
		comp = -comp
	}
	return comp, true
}

// WithDefaults backfills any zero-value fields in cfg from
// DefaultAGCConfig so partial-override calls don't have to specify
// every parameter. Returns the merged config.
func (cfg AGCConfig) WithDefaults() AGCConfig {
	d := DefaultAGCConfig()
	if cfg.TargetPeak == 0 {
		cfg.TargetPeak = d.TargetPeak
	}
	if cfg.Attack == 0 {
		cfg.Attack = d.Attack
	}
	if cfg.Release == 0 {
		cfg.Release = d.Release
	}
	if cfg.MinGain == 0 {
		cfg.MinGain = d.MinGain
	}
	if cfg.MaxGain == 0 {
		cfg.MaxGain = d.MaxGain
	}
	if cfg.NoiseFloor == 0 {
		cfg.NoiseFloor = d.NoiseFloor
	}
	return cfg
}

// AGC is the per-frame fast-attack / slow-release peak-envelope
// tracker shared across MBE-family decoders. The smoothed envelope
// scales each frame's float PCM to AGCConfig.TargetPeak; samples that
// would exceed the int16 range are soft-limited (see softLimit), not
// hard-clipped.
//
// Concurrent calls to Apply on the same AGC are not safe; each
// decoder owns one AGC instance.
type AGC struct {
	cfg AGCConfig
	env float64 // smoothed peak envelope; 0 = fresh (next frame seeds it)

	// Per-frame telemetry from the most recent Apply call, read by the
	// IMBE decoder which accumulates its own per-call totals. Kept
	// per-frame (not cumulative) so a mid-call Reset — which the upstream
	// decoder triggers on re-sync — never discards a call's running totals.
	lastGain        float64 // gain applied on the most recent frame
	lastMaxPreClip  float64 // largest pre-limit sample magnitude this frame
	lastClipSamples int     // samples that engaged the soft limiter this frame
}

// NewAGC constructs an AGC with the supplied config. Zero-value
// fields in cfg backfill from DefaultAGCConfig.
func NewAGC(cfg AGCConfig) *AGC {
	return &AGC{cfg: cfg.WithDefaults()}
}

// Config returns the AGC's effective configuration (post-defaults
// backfill).
func (a *AGC) Config() AGCConfig { return a.cfg }

// SetTargetPeak overrides the post-AGC peak target (int16 units),
// letting a caller make the output louder or quieter than the faithful
// default after construction — the optional voice enhancement chain uses
// it to raise the level toward what the louder rival decoders produce. A
// non-positive value is ignored so a disabled/zero config can't silence
// the output. The running envelope is preserved (the next frame just
// scales toward the new target).
func (a *AGC) SetTargetPeak(peak float64) {
	if peak <= 0 {
		return
	}
	a.cfg.TargetPeak = peak
}

// Envelope returns the current smoothed peak envelope. Useful for
// tests + introspection; production callers don't normally read
// it.
func (a *AGC) Envelope() float64 { return a.env }

// Reset clears the envelope so the next Apply call seeds from the
// frame's peak again. Used on stream re-sync (e.g., a frame-loss event
// from the upstream protocol decoder). Per-frame telemetry is left as-is
// (it is overwritten on the next Apply) so a mid-call Reset doesn't lose
// the caller's running per-call totals.
func (a *AGC) Reset() { a.env = 0 }

// LastGain returns the gain applied on the most recent Apply call.
func (a *AGC) LastGain() float64 { return a.lastGain }

// LastMaxPreClipPeak returns the largest pre-limit sample magnitude
// (int16 units) on the most recent Apply call.
func (a *AGC) LastMaxPreClipPeak() float64 { return a.lastMaxPreClip }

// LastClipSamples returns how many samples engaged the soft limiter on
// the most recent Apply call.
func (a *AGC) LastClipSamples() int { return a.lastClipSamples }

// Apply tracks the per-frame peak with fast-attack / slow-release
// smoothing and writes pcm scaled to the config's TargetPeak into
// out. Frames whose peak falls below cfg.NoiseFloor leave the
// envelope unchanged so a tail fade-out into silence doesn't drag
// the envelope up artificially.
//
// First-frame seed: when a.env == 0 (fresh AGC, post-Reset), the
// envelope is initialised directly to peak rather than via the
// attack coefficient. Without this seed the first frame would
// emerge ~2.5× over-gained (envelope = Attack · peak ⇒ gain =
// TargetPeak / (Attack · peak) ⇒ output peak = (1/Attack) ·
// TargetPeak ⇒ int16 saturation at the default Attack = 0.4).
//
// Frozen mode (freezeEnvelope = true): apply the existing
// envelope's gain without updating it. Callers use this on silence
// frames so a brief silence doesn't shift the envelope based on
// the small §6.4 OA tail content; bad-frame replays use it so the
// per-frame attenuation is audible (signals signal degradation).
//
// MinGain / MaxGain prevent the envelope from sending silence to
// full scale or compressing extreme transients to inaudible levels.
// After the gain multiply, samples are passed through softLimit so the
// output approaches but never reaches the int16 rail (no hard-clip
// corners). Per-call telemetry (gain, pre-limit peak, limited-sample
// count) is accumulated for the VoiceStats summary.
func (a *AGC) Apply(pcm []float64, out []int16, freezeEnvelope bool) {
	cfg := a.cfg
	// The per-frame peak is needed in both modes: to update the envelope
	// (non-frozen) and to cap the gain so a frozen frame can't be amplified
	// past the limiter by a stale envelope (the ceiling below).
	var peak float64
	for _, v := range pcm {
		if abs := math.Abs(v); abs > peak {
			peak = abs
		}
	}
	if !freezeEnvelope {
		if a.env == 0 && peak > cfg.NoiseFloor {
			// First-frame seed: skip attack smoothing so the first
			// frame lands at exactly TargetPeak instead of 1/Attack× over.
			a.env = peak
		} else if peak > cfg.NoiseFloor {
			coef := cfg.Attack
			if peak < a.env {
				coef = cfg.Release
			}
			a.env += (peak - a.env) * coef
		}
	}
	envelope := a.env
	if envelope < cfg.NoiseFloor {
		envelope = cfg.NoiseFloor
	}
	gain := cfg.TargetPeak / envelope
	if gain < cfg.MinGain {
		gain = cfg.MinGain
	} else if gain > cfg.MaxGain {
		gain = cfg.MaxGain
	}
	// Per-frame output ceiling: never amplify THIS frame's own peak past the
	// soft-limiter knee, regardless of how stale or low the smoothed envelope
	// is. Without it, a near-silent onset frame seeds/holds a tiny envelope and
	// the next louder frame — or a frozen bad-frame replay / idle-mute that
	// reuses the held gain (Apply with freezeEnvelope=true does not update the
	// envelope) — is multiplied by a gain up to MaxGain, slamming every sample
	// into the limiter. Field captures showed mean_gain≈1e5, peak_preclip≈8e7,
	// clip_pct≈65% on short call onsets from exactly this. The ceiling only
	// engages when the AGC would otherwise push the output past the knee, so
	// ordinary frames (output ≈ TargetPeak, well under the knee) and the
	// inter-frame dynamics the fast-attack/slow-release loop produces are
	// unchanged — it is a one-sided clamp that can only reduce clipping.
	if peak > 0 {
		if ceil := softLimitKnee / peak; gain > ceil {
			gain = ceil
		}
	}
	a.lastGain = gain
	a.lastMaxPreClip = 0
	a.lastClipSamples = 0
	for i, v := range pcm {
		s := v * gain
		if abs := math.Abs(s); abs > a.lastMaxPreClip {
			a.lastMaxPreClip = abs
		}
		limited, didLimit := softLimit(s)
		if didLimit {
			a.lastClipSamples++
		}
		out[i] = int16(limited)
	}
}
