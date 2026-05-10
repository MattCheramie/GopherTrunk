package imbe

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// Frame parameters per TIA-102.BABA. Every IMBE 4400 frame carries
// 88 information bits over 20 ms of audio at 8 kHz mono.
const (
	// InfoBits is the per-frame information-bit count after channel
	// FEC has been applied + verified.
	InfoBits = 88

	// FrameBytes is InfoBits packed MSB-first into octets, rounded up.
	// Matches the mbelib wrapper's frame length so callers can pass
	// the same byte slice to either backend.
	FrameBytes = 11

	// SamplesPerFrame is the PCM count one Decode call produces.
	// IMBE is fixed at 8 kHz × 20 ms = 160 samples.
	SamplesPerFrame = 160

	// PCMSampleRate is the recorder's expected output rate.
	PCMSampleRate = 8000

	// FrameDurationMs documents the 20 ms cadence.
	FrameDurationMs = 20

	// VocoderName is the registry key the daemon resolves at startup.
	// This is the canonical "imbe" name; the pure-Go decoder is the
	// sole IMBE backend in default builds.
	VocoderName = "imbe"

	// MaxBadFrames is the number of consecutive bad frames the
	// frame-repeat path will replay before giving up and emitting
	// silence + clearing state. The TIA-102.BABA spec range is
	// 1..6; mbelib uses 6, which gives the upstream P25 LDU FEC
	// roughly 120 ms of grace before the audio path drops out
	// completely. After MaxBadFrames bad frames the cumulative
	// attenuation is BadFrameAttenuation^6 ≈ 0.118 — quiet enough
	// that an extended bad streak fades naturally rather than
	// looping the same envelope.
	MaxBadFrames = 6

	// BadFrameAttenuation is the per-frame multiplier applied to
	// the cached last-good amplitudes during a frame-repeat. A
	// single bad frame plays at 70% of the prev good frame's
	// amplitudes; six in a row taper to ~12%. This is a balance
	// between hiding the FEC slip (no abrupt mute) and signalling
	// the listener that signal is degrading (audible volume drop).
	BadFrameAttenuation = 0.7
)

// AGCConfig holds the per-frame AGC parameters. The synthesizer's
// float output magnitude is stable per-frame (R_M0-preserving §6.2
// enhancement holds total energy constant across a frame) but
// varies wildly between frames depending on Tl, voicing, and the
// §6.4 noise draw. Without an AGC every frame would be either
// clipped (loud frames) or near-silent (quiet frames). The AGC
// tracks the per-frame peak with fast-attack / slow-release
// smoothing, then scales each frame so the smoothed envelope hits
// TargetPeak.
//
// Zero-value fields fall back to DefaultAGCConfig values, so a
// caller can override only the field they care about (e.g.
// AGCConfig{TargetPeak: 16000} to drop output level by 4 dB).
//
// The current AGC is a level-only design — disabling it entirely
// means dropping back to a constant gain, which we don't expose as
// a separate option (callers wanting that can pass equal Attack +
// Release to fully smooth the envelope). The §6.2-derived
// per-frame R_M0 normalization the synthesizer already does keeps
// inter-frame magnitude variation modest enough that AGC is a
// polish layer rather than the only loudness control.
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

// DefaultAGCConfig returns the AGC parameters Decoder.applyAGC uses
// when constructed via New() / NewWithSeed(). NewWithConfig callers
// can take this struct, override individual fields, and pass it back
// for partial-overrides.
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

// withDefaults backfills any zero-value fields in cfg from
// DefaultAGCConfig so partial-override calls don't have to specify
// every parameter. Returns the merged config.
func (cfg AGCConfig) withDefaults() AGCConfig {
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

// Decoder is the pure-Go IMBE 4400 decoder. It owns one SynthState
// (cross-frame log2(Ml) prediction memory + voiced phase + amp
// memory), one math/rand source for the §6.4 unvoiced excitation
// noise, one AGC envelope tracker, and a one-frame cache of the
// last-good parameters for the frame-repeat path — all per-call
// so concurrent calls on different decoders don't share state.
//
// Decode() runs the full TIA-102.BABA pipeline:
//   - bytes → 88 info bits
//   - UnpackParams: §5.3 / §5.4 / Annex E
//   - PredictLog2Ml: §6.1 cross-frame prediction
//   - AmplitudesFromLog2Ml: log2(Ml) → linear Ml
//   - EnhanceAmplitudes: §6.2 spectral-amplitude enhancement
//   - SynthVoiced: §6.3 voiced harmonic generator
//   - SynthUnvoicedOverlapAdd: §6.4 unvoiced FFT excitation + OA
//   - Update{Log2Ml,VoicedState}: roll state forward
//   - applyAGC: per-frame fast-attack / slow-release peak tracker
//     scaling to agcTargetPeak, then int16 clip
//
// On a bad frame (UnpackParams returns ErrInvalidFundamental, etc.),
// Decode replays the last good frame's amplitudes scaled by
// BadFrameAttenuation^badFrameCount. After MaxBadFrames consecutive
// bad frames the cache is cleared and Decode returns silence so an
// extended bad streak fades naturally instead of looping the same
// envelope forever.
type Decoder struct {
	state  SynthState
	rng    *rand.Rand
	agc    float64   // smoothed peak envelope; 0 = fresh (next frame seeds it)
	agcCfg AGCConfig // tunable AGC parameters; defaults via DefaultAGCConfig()

	// One-frame cache for the frame-repeat path. Populated after a
	// good non-silent frame; consumed by the bad-frame path; cleared
	// on silence-window frames + on Reset + when MaxBadFrames is
	// exceeded.
	lastGoodParams Params
	lastGoodLog2M  [57]float64
	lastGoodM      [57]float64

	// Consecutive bad-frame count. Increments once per replay,
	// resets to 0 on every good non-silent frame.
	badFrameCount int
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
// DefaultAGCConfig.
func NewWithSeed(seed int64) *Decoder {
	return NewWithConfig(seed, DefaultAGCConfig())
}

// NewWithConfig constructs a Decoder with an explicit noise seed +
// AGC configuration. Zero-value fields in cfg fall back to
// DefaultAGCConfig values, so callers can override only the
// parameters they care about (e.g. AGCConfig{TargetPeak: 16000} to
// drop the output level by ~3 dB without re-tuning attack/release).
func NewWithConfig(seed int64, cfg AGCConfig) *Decoder {
	return &Decoder{
		rng:    rand.New(rand.NewSource(seed)),
		agcCfg: cfg.withDefaults(),
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
//     + badFrameCount; AGC envelope preserved across the silence.
//   - bad frame (UnpackParams error) with cached last-good frame
//     and badFrameCount < MaxBadFrames: replay last-good params
//     with M scaled by BadFrameAttenuation^badFrameCount; AGC
//     freezes so the attenuation is audible (signals signal loss).
//   - bad frame with no cache, or after MaxBadFrames consecutive
//     bad frames: emit silence; clear last-good cache + reset
//     SynthState; AGC envelope preserved.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("imbe: frame must be %d bytes (88 bits), got %d", FrameBytes, len(frame))
	}

	info := unpackInfoBits(frame)
	out := make([]int16, SamplesPerFrame)
	pcm := make([]float64, SamplesPerFrame)

	p, err := UnpackParams(info)

	switch {
	case err != nil && d.lastGoodParams.L > 0 && d.badFrameCount < MaxBadFrames:
		// Frame repeat: replay the last-good params with progressive
		// per-frame attenuation. Synthesis still runs (so OA tail +
		// phase memory continue rolling forward), AGC freezes so the
		// attenuation is audible.
		d.badFrameCount++
		atten := math.Pow(BadFrameAttenuation, float64(d.badFrameCount))
		repeatedM := d.lastGoodM
		for l := 1; l <= d.lastGoodParams.L; l++ {
			repeatedM[l] *= atten
		}
		d.synthFrame(d.lastGoodParams, &d.lastGoodLog2M, &repeatedM, pcm)
		d.applyAGC(pcm, out, true)
		return out, nil

	case err != nil:
		// Bad frame with no cache, or bad-frame budget exhausted:
		// emit silence + clear cache + reset SynthState. AGC envelope
		// preserved so audio level is consistent when good frames
		// return.
		d.state.Reset()
		d.clearLastGood()
		d.applyAGC(pcm, out, true)
		return out, nil

	case p.Silent:
		// b_0 ∈ [216, 219]: explicit silence indicator. Run the §6.4
		// overlap-add with no new noise so the prev-frame unvoiced
		// tail still fades into pcm[0..95] (no click on the silence
		// boundary), then reset SynthState + last-good cache so the
		// next non-silent frame starts from a clean baseline.
		SynthUnvoicedOverlapAdd(&d.state, p, nil, nil, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.applyAGC(pcm, out, true)
		return out, nil
	}

	// Good non-silent frame.
	d.badFrameCount = 0
	var log2M [57]float64
	PredictLog2Ml(&d.state, p, &log2M)

	var M [57]float64
	AmplitudesFromLog2Ml(&log2M, p.L, &M)
	EnhanceAmplitudes(p, &M)

	d.synthFrame(p, &log2M, &M, pcm)

	// Cache for the frame-repeat path on a future bad frame.
	d.lastGoodParams = p
	d.lastGoodLog2M = log2M
	d.lastGoodM = M

	d.applyAGC(pcm, out, false)
	return out, nil
}

// synthFrame runs the §6.3 voiced + §6.4 unvoiced overlap-add legs
// of the pipeline and rolls SynthState forward by log2M / M. Used
// by both the good-frame path and the bad-frame replay path so the
// two share identical synthesis behavior — the only difference is
// what M values get fed in.
func (d *Decoder) synthFrame(p Params, log2M *[57]float64, M *[57]float64, pcm []float64) {
	SynthVoiced(&d.state, p, M, pcm)
	noise := make([]float64, UnvoicedFFTSize)
	for i := range noise {
		noise[i] = d.rng.NormFloat64()
	}
	SynthUnvoicedOverlapAdd(&d.state, p, M, noise, pcm)
	d.state.UpdateLog2Ml(p, log2M)
	d.state.UpdateVoicedState(p, M)
}

// clearLastGood resets the frame-repeat cache + bad-frame counter.
// Called on silence-window frames, on the bad-frame budget being
// exceeded, and from the public Reset.
func (d *Decoder) clearLastGood() {
	d.lastGoodParams = Params{}
	d.lastGoodLog2M = [57]float64{}
	d.lastGoodM = [57]float64{}
	d.badFrameCount = 0
}

// applyAGC tracks the per-frame peak with fast-attack / slow-release
// smoothing and scales pcm so the smoothed envelope hits
// d.agcCfg.TargetPeak. Frames whose peak falls below
// d.agcCfg.NoiseFloor leave the envelope unchanged so a tail
// fade-out into silence doesn't drag the envelope up artificially.
//
// First-frame seed: when d.agc == 0 (fresh decoder, post-Reset), the
// envelope is initialised directly to peak rather than via the attack
// coefficient. Without this seed the first frame would emerge ~2.5×
// over-gained (envelope = Attack · peak ⇒ gain = TargetPeak /
// (Attack · peak) ⇒ output peak = (1/Attack) · TargetPeak ⇒
// int16 saturation at the default Attack = 0.4).
//
// Frozen mode (freezeEnvelope = true): apply the existing envelope's
// gain without updating it. The Decode() silent path uses this so a
// brief silence frame doesn't shift the AGC envelope based on the
// small §6.4 overlap-add fade-out content; the bad-frame replay
// path uses it so the per-frame attenuation is audible (signals
// signal degradation). Stream re-sync via the public Reset() does
// clear the envelope.
//
// MinGain / MaxGain prevent the envelope from sending silence to
// full scale or compressing extreme transients to inaudible levels.
// After the gain multiply, samples beyond int16 range are
// hard-clipped at ±32767.
func (d *Decoder) applyAGC(pcm []float64, out []int16, freezeEnvelope bool) {
	cfg := d.agcCfg
	if !freezeEnvelope {
		var peak float64
		for _, v := range pcm {
			if a := math.Abs(v); a > peak {
				peak = a
			}
		}
		if d.agc == 0 && peak > cfg.NoiseFloor {
			// First-frame seed: skip attack smoothing so the first frame
			// lands at exactly TargetPeak instead of 1/Attack× over.
			d.agc = peak
		} else if peak > cfg.NoiseFloor {
			coef := cfg.Attack
			if peak < d.agc {
				coef = cfg.Release
			}
			d.agc += (peak - d.agc) * coef
		}
	}
	envelope := d.agc
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
	d.agc = 0
	d.clearLastGood()
}

// Close releases any resources held by the decoder. The pure-Go
// implementation holds none, so this is always a no-op.
func (d *Decoder) Close() error { return nil }

// Compile-time check that Decoder satisfies voice.Vocoder.
var _ voice.Vocoder = (*Decoder)(nil)

func init() {
	voice.DefaultRegistry.Register(VocoderName, func() (voice.Vocoder, error) {
		return New(), nil
	})
}
