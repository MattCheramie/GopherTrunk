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
	// units. Default 24000 (~3 dB below int16 max for transient
	// headroom).
	TargetPeak float64

	// Attack is the per-frame envelope rise coefficient. Standard
	// IIR coefficient: 1.0 = instant tracking, 0.0 = no update.
	// Default 0.4 — fast enough to catch loud onsets without
	// over-shoot.
	Attack float64

	// Release is the per-frame envelope fall coefficient. Smaller
	// than Attack so gain ramps back up slowly during quiet
	// passages — standard AGC behavior that keeps speech
	// intelligible without pumping. Default 0.02.
	Release float64

	// MinGain bounds the lowest gain the AGC can apply, preventing
	// runaway compression on extreme transients. Default 10.
	MinGain float64

	// MaxGain bounds the highest gain the AGC can apply, preventing
	// silence from being amplified to full scale by a stale low
	// envelope. Default 1e5.
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
		TargetPeak: 24000.0,
		Attack:     0.4,
		Release:    0.02,
		MinGain:    10.0,
		MaxGain:    1e5,
		NoiseFloor: 1e-3,
	}
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
// scales each frame's float PCM to AGCConfig.TargetPeak; samples
// beyond int16 range are hard-clipped at ±32767.
//
// Concurrent calls to Apply on the same AGC are not safe; each
// decoder owns one AGC instance.
type AGC struct {
	cfg AGCConfig
	env float64 // smoothed peak envelope; 0 = fresh (next frame seeds it)
}

// NewAGC constructs an AGC with the supplied config. Zero-value
// fields in cfg backfill from DefaultAGCConfig.
func NewAGC(cfg AGCConfig) *AGC {
	return &AGC{cfg: cfg.WithDefaults()}
}

// Config returns the AGC's effective configuration (post-defaults
// backfill).
func (a *AGC) Config() AGCConfig { return a.cfg }

// Envelope returns the current smoothed peak envelope. Useful for
// tests + introspection; production callers don't normally read
// it.
func (a *AGC) Envelope() float64 { return a.env }

// Reset clears the envelope so the next Apply call seeds from the
// frame's peak again. Used on stream re-sync (e.g., a frame-loss
// event from the upstream protocol decoder).
func (a *AGC) Reset() { a.env = 0 }

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
// After the gain multiply, samples beyond int16 range are
// hard-clipped at ±32767.
func (a *AGC) Apply(pcm []float64, out []int16, freezeEnvelope bool) {
	cfg := a.cfg
	if !freezeEnvelope {
		var peak float64
		for _, v := range pcm {
			if abs := math.Abs(v); abs > peak {
				peak = abs
			}
		}
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
	for i, v := range pcm {
		s := v * gain
		if s > 32767 {
			s = 32767
		} else if s < -32768 {
			s = -32768
		}
		out[i] = int16(s)
	}
}
