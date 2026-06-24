package mbe

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// VoiceEnhancer is the OPTIONAL, opt-in "sound-good" post-processing
// chain run over the synthesized PCM between the DC block and the AGC.
// It is disabled by default: GopherTrunk's spec path aims to be faithful
// to the transmission, and the residual thin/synthetic character of
// software AMBE+2 decode is intrinsic to software vocoding. This chain
// deliberately trades a little faithfulness for the kind of subjective
// "cleaner / louder / warmer" sound the rival decoders produce, for
// operators who care more about how it sounds than how faithful it is.
//
// What the rivals actually do (and what this replicates):
//
//   - OP25 low-passes its output at 3.4 kHz (telephone band) — trims the
//     out-of-band hiss / quantization buzz that reads as "tinny/harsh".
//   - Trunk Recorder ships EBU R128 loudness normalization ON by default
//     and offers a ~300 Hz high-pass to kill rumble; DSD/DSDPlus run an
//     aggressive output AGC to a high peak target. Uniform loudness is the
//     single biggest reason those tools sound "better".
//   - (SDRtrunk, by contrast, applies NO sweetening to digital voice — so
//     loudness/EQ is a choice, not a requirement.)
//
// The chain, in order:
//
//  1. High-pass (~250 Hz)  — rumble / sub-voice energy beyond the DC block
//  2. Presence shelf       — gentle high-shelf cut to warm the bright timbre
//  3. Low-pass (~3.4 kHz)  — band-limit to telephone band
//  4. Compressor           — optional soft-knee gain reduction (default off)
//
// Output loudness is handled separately by raising the decoder's AGC
// TargetPeak (see EnhancerConfig.AGCTarget / AGC.SetTargetPeak), NOT by a
// second gain stage in this chain, so the chain never fights the AGC's
// soft-limiter.
//
// A VoiceEnhancer is per-stream: the caller keeps one per decode stream
// and Resets it on re-sync. A nil receiver is a pass-through so callers
// can hold an optional *VoiceEnhancer and invoke it unconditionally,
// mirroring the existing *ToneTilt wiring. Not safe for concurrent use.
type VoiceEnhancer struct {
	hpf   *filter.Biquad // pre-shelf rumble high-pass
	shelf *ToneTilt      // presence/warmth high-shelf
	lpf   *filter.Biquad // band-limit low-pass
	comp  *compressor    // optional dynamic-range compression
}

// EnhancerConfig is the tunable shape of the voice enhancement chain.
// The zero value is disabled (a pass-through). DefaultEnhancerConfig
// supplies sensible starting parameters; zero-value fields in an enabled
// config backfill from those defaults via WithDefaults so a caller can
// flip Enabled and override only the parameters they care about.
type EnhancerConfig struct {
	// Enabled gates the whole chain. False ⇒ NewVoiceEnhancer returns
	// nil (pass-through) and the decoder output is byte-identical to the
	// faithful path.
	Enabled bool

	// HPFHz is the −3 dB corner of the rumble high-pass. 0 backfills to
	// the default; a negative value disables just the high-pass.
	HPFHz float64
	// LPFHz is the −3 dB corner of the band-limit low-pass. 0 backfills
	// to the default; a negative value disables just the low-pass.
	LPFHz float64
	// ShelfHz / ShelfDB describe the presence high-shelf: ShelfDB dB of
	// cut above ShelfHz. ShelfDB ≤ 0 disables just the shelf.
	ShelfHz float64
	ShelfDB float64

	// AGCTarget overrides the decoder AGC peak target (int16 units) when
	// the chain is enabled, so calls play back louder. 0 backfills to the
	// default; the decoder applies it via AGC.SetTargetPeak. It is read by
	// the decoder, not by Process.
	AGCTarget float64

	// Compress configures the optional soft-knee compressor (default off).
	Compress CompressConfig
}

// CompressConfig is the shape of the optional output compressor. Disabled
// by default — an opt-in within the opt-in chain — because the AGC + the
// raised TargetPeak already deliver most of the perceived loudness.
type CompressConfig struct {
	Enabled bool
	// ThresholdDB is the level (dBFS, negative) above which gain
	// reduction begins. Default -18.
	ThresholdDB float64
	// Ratio is the compression ratio above the threshold (e.g. 2 ⇒ 2:1).
	// Default 2. Values ≤ 1 disable compression.
	Ratio float64
	// AttackMs / ReleaseMs shape the envelope follower. Defaults 5 / 80.
	AttackMs  float64
	ReleaseMs float64
	// MakeupDB is a fixed post-compression gain in dB. Default 0 (the
	// raised AGC target carries makeup loudness).
	MakeupDB float64
}

// DefaultEnhancerConfig returns the starting parameters for an enabled
// chain. The band (250 Hz – 3400 Hz) is the classic telephone voice band
// (OP25 low-passes at 3.4 kHz); the shelf matches the measured
// GopherTrunk-vs-mbelib brightness offset (issue #644, ~1.5–2 dB hot
// across 1.5–4 kHz); the AGC target (22000 ≈ −3.5 dBFS) is louder than
// the faithful default (18000) but still clears the soft-limiter knee
// (26213).
func DefaultEnhancerConfig() EnhancerConfig {
	return EnhancerConfig{
		Enabled:   true,
		HPFHz:     250,
		LPFHz:     3400,
		ShelfHz:   1500,
		ShelfDB:   2.0,
		AGCTarget: 22000,
		Compress: CompressConfig{
			ThresholdDB: -18,
			Ratio:       2,
			AttackMs:    5,
			ReleaseMs:   80,
			MakeupDB:    0,
		},
	}
}

// WithDefaults backfills zero-value fields in an enabled config from
// DefaultEnhancerConfig so partial-override callers don't have to specify
// every parameter. A disabled config is returned unchanged.
func (cfg EnhancerConfig) WithDefaults() EnhancerConfig {
	if !cfg.Enabled {
		return cfg
	}
	d := DefaultEnhancerConfig()
	if cfg.HPFHz == 0 {
		cfg.HPFHz = d.HPFHz
	}
	if cfg.LPFHz == 0 {
		cfg.LPFHz = d.LPFHz
	}
	if cfg.ShelfHz == 0 {
		cfg.ShelfHz = d.ShelfHz
	}
	if cfg.ShelfDB == 0 {
		cfg.ShelfDB = d.ShelfDB
	}
	if cfg.AGCTarget == 0 {
		cfg.AGCTarget = d.AGCTarget
	}
	if cfg.Compress.Enabled {
		if cfg.Compress.ThresholdDB == 0 {
			cfg.Compress.ThresholdDB = d.Compress.ThresholdDB
		}
		if cfg.Compress.Ratio == 0 {
			cfg.Compress.Ratio = d.Compress.Ratio
		}
		if cfg.Compress.AttackMs == 0 {
			cfg.Compress.AttackMs = d.Compress.AttackMs
		}
		if cfg.Compress.ReleaseMs == 0 {
			cfg.Compress.ReleaseMs = d.Compress.ReleaseMs
		}
	}
	return cfg
}

// NewVoiceEnhancer builds the chain for a stream at sampleRate Hz.
// Returns nil (a pass-through) when cfg is disabled or sampleRate is
// non-positive, so the default faithful path stays byte-identical.
// Individual stages are skipped (nil / pass-through) when their
// parameters disable them, so e.g. a config that only wants band-limiting
// pays for nothing else.
func NewVoiceEnhancer(sampleRate float64, cfg EnhancerConfig) *VoiceEnhancer {
	if !cfg.Enabled || sampleRate <= 0 {
		return nil
	}
	cfg = cfg.WithDefaults()
	e := &VoiceEnhancer{}
	if cfg.HPFHz > 0 {
		e.hpf = filter.NewHighPass(sampleRate, cfg.HPFHz)
	}
	if cfg.ShelfDB > 0 {
		e.shelf = NewToneTilt(sampleRate, cfg.ShelfHz, cfg.ShelfDB)
	}
	if cfg.LPFHz > 0 {
		e.lpf = filter.NewLowPass(sampleRate, cfg.LPFHz)
	}
	if cfg.Compress.Enabled && cfg.Compress.Ratio > 1 {
		e.comp = newCompressor(sampleRate, cfg.Compress)
	}
	return e
}

// Process runs the enabled stages over pcm in place. A nil receiver is a
// no-op.
func (e *VoiceEnhancer) Process(pcm []float64) {
	if e == nil {
		return
	}
	e.hpf.Process(pcm)
	e.shelf.Process(pcm)
	e.lpf.Process(pcm)
	e.comp.process(pcm)
}

// Reset clears all stage state (call on re-sync alongside the DC block).
// A nil receiver is a no-op.
func (e *VoiceEnhancer) Reset() {
	if e == nil {
		return
	}
	e.hpf.Reset()
	e.shelf.Reset()
	e.lpf.Reset()
	e.comp.reset()
}

// compressor is a simple feed-forward soft-knee compressor with a
// peak-following envelope. It is intentionally gentle and off by default;
// the AGC + raised TargetPeak carry most of the loudness, so this exists
// for operators who want speech transients reined in for intelligibility.
type compressor struct {
	thresh  float64 // linear threshold (int16 units)
	ratio   float64
	attack  float64 // per-sample envelope-rise coefficient
	release float64 // per-sample envelope-fall coefficient
	makeup  float64 // linear makeup gain
	env     float64 // peak-following envelope (int16 units)
}

// fullScale is the int16 reference the compressor measures dBFS against.
const fullScale = 32767.0

func newCompressor(sampleRate float64, cfg CompressConfig) *compressor {
	// One-pole envelope coefficients from time constants: a step settles
	// to ~63% in one time constant.
	coef := func(ms float64) float64 {
		if ms <= 0 {
			return 1
		}
		return 1 - math.Exp(-1.0/(ms*0.001*sampleRate))
	}
	return &compressor{
		thresh:  math.Pow(10, cfg.ThresholdDB/20) * fullScale,
		ratio:   cfg.Ratio,
		attack:  coef(cfg.AttackMs),
		release: coef(cfg.ReleaseMs),
		makeup:  math.Pow(10, cfg.MakeupDB/20),
	}
}

func (c *compressor) process(pcm []float64) {
	if c == nil {
		return
	}
	for i, x := range pcm {
		a := math.Abs(x)
		if a > c.env {
			c.env += c.attack * (a - c.env)
		} else {
			c.env += c.release * (a - c.env)
		}
		gain := c.makeup
		if c.env > c.thresh && c.env > 0 {
			// Static soft curve: above the threshold the level is
			// compressed by the ratio. Computed on the smoothed envelope
			// so the gain doesn't chatter sample-to-sample.
			over := c.env / c.thresh                // > 1
			compressed := math.Pow(over, 1/c.ratio) // < over
			gain *= compressed / over
		}
		pcm[i] = x * gain
	}
}

func (c *compressor) reset() {
	if c == nil {
		return
	}
	c.env = 0
}
