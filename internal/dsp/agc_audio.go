package dsp

import (
	"math"
	"time"
)

// AudioAGC is the real-valued counterpart of AGC, sized for the
// post-demod chain in internal/voice/composer. The IQ-domain AGC
// drives the *complex* baseband magnitude toward a target with a
// single adaptation rate, which works because FM has a constant
// envelope on air. After demod the signal is voice — bursty, with
// short loud transients separated by quieter passages — so a single
// rate either pumps badly (too fast) or never catches up (too slow).
//
// AudioAGC is a classic envelope-follower + gain stage:
//
//	abs   = |x|
//	if abs > level: level += (1 - α_attack) × (abs - level)   // ramp up fast
//	else:           level += (1 - α_release) × (abs - level)  // ramp down slow
//	gain  = clamp(reference / max(level, floor), 0, MaxGain)
//	y[n]  = x[n] × gain
//
// Attack / release are time constants (in samples after construction);
// short attack catches transients before they clip, long release
// keeps the gain steady through speech gaps so the output doesn't
// pump. Reference is the target |y| ≈ Reference; pick something that
// leaves headroom for downstream stages (typical 0.3 keeps int16
// conversion comfortably under MaxInt16 at the existing 10 000-scale
// in voice/composer).
//
// AudioAGC is not safe for concurrent Process calls — pin it to a
// single demod goroutine and Reset between calls.
type AudioAGC struct {
	reference   float32
	maxGain     float32
	attackCoef  float32 // 1 - exp(-1/(attack × fs))
	releaseCoef float32 // 1 - exp(-1/(release × fs))
	floor       float32 // dead-air floor on the level estimate

	level float32
}

// AudioAGCConfig configures NewAudioAGC. All time constants are in
// real time; the constructor folds in the sample rate.
type AudioAGCConfig struct {
	Reference  float32       // target |output| (default 0.3)
	Attack     time.Duration // ramp-up time constant (default 5 ms)
	Release    time.Duration // ramp-down time constant (default 200 ms)
	MaxGain    float32       // ceiling on adaptive gain (default 64.0)
	SampleRate float64       // sample rate of the audio stream (Hz, required)
}

// NewAudioAGC builds an envelope-follower-based AGC. Bad parameters
// trip a panic at startup so misconfiguration shows up loudly rather
// than silently producing wrong audio.
func NewAudioAGC(cfg AudioAGCConfig) *AudioAGC {
	if cfg.SampleRate <= 0 {
		panic("dsp: NewAudioAGC requires a positive sample rate")
	}
	if cfg.Reference <= 0 {
		cfg.Reference = 0.3
	}
	if cfg.Attack <= 0 {
		cfg.Attack = 5 * time.Millisecond
	}
	if cfg.Release <= 0 {
		cfg.Release = 200 * time.Millisecond
	}
	if cfg.MaxGain <= 0 {
		cfg.MaxGain = 64
	}
	attack := 1.0 - math.Exp(-1.0/(cfg.Attack.Seconds()*cfg.SampleRate))
	release := 1.0 - math.Exp(-1.0/(cfg.Release.Seconds()*cfg.SampleRate))
	return &AudioAGC{
		reference:   cfg.Reference,
		maxGain:     cfg.MaxGain,
		attackCoef:  float32(attack),
		releaseCoef: float32(release),
		floor:       cfg.Reference / cfg.MaxGain,
	}
}

// Reset clears the running envelope estimate so the next Process call
// starts from silence (gain begins at MaxGain).
func (a *AudioAGC) Reset() {
	a.level = 0
}

// Gain returns the current adaptive gain. Useful for diagnostics; not
// needed for normal operation.
func (a *AudioAGC) Gain() float32 {
	level := a.level
	if level < a.floor {
		level = a.floor
	}
	g := a.reference / level
	if g > a.maxGain {
		g = a.maxGain
	}
	return g
}

// Process applies the AGC to src and writes to dst (or appends to it).
// dst is reused if it has enough capacity; in-place src == dst is
// supported.
func (a *AudioAGC) Process(dst, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	level := a.level
	for i, x := range src {
		abs := x
		if abs < 0 {
			abs = -abs
		}
		if abs > level {
			level += a.attackCoef * (abs - level)
		} else {
			level += a.releaseCoef * (abs - level)
		}
		clamped := level
		if clamped < a.floor {
			clamped = a.floor
		}
		gain := a.reference / clamped
		if gain > a.maxGain {
			gain = a.maxGain
		}
		dst[i] = x * gain
	}
	a.level = level
	return dst
}
