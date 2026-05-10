package imbe

import (
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
//   - mbe.SynthVoiced: §6.3 voiced harmonic generator
//   - mbe.SynthUnvoicedOverlapAdd: §6.4 unvoiced FFT excitation + OA
//   - mbe.SynthState.Update{Log2Ml,VoicedState}: roll state forward
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
	agc   *mbe.AGC

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

	// recoveryFramesRemaining is the number of upcoming good frames
	// that should run through the recovery ramp. Set to
	// len(recoveryRampFactors) by the bad-streak budget-exhaust
	// path; decremented each good frame; zero outside recovery.
	// The ramp index is len(recoveryRampFactors) -
	// recoveryFramesRemaining so the first recovery frame uses the
	// smallest factor and the last uses 1.0 (full amplitude).
	recoveryFramesRemaining int
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
		agc: mbe.NewAGC(cfg),
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

	p, err := UnpackParams(info)

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
		d.agc.Apply(pcm, out, true)
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
		d.clearLastGood()
		d.recoveryFramesRemaining = len(recoveryRampFactors)
		d.agc.Apply(pcm, out, true)
		return out, nil

	case p.Silent:
		// b_0 ∈ [216, 219]: explicit silence indicator. Run the §6.4
		// overlap-add with no new noise so the prev-frame unvoiced
		// tail still fades into pcm[0..95] (no click on the silence
		// boundary), then reset SynthState + last-good cache so the
		// next non-silent frame starts from a clean baseline.
		mbe.SynthUnvoicedOverlapAdd(&d.state, p.MBE(), nil, nil, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.agc.Apply(pcm, out, true)
		return out, nil
	}

	// Good non-silent frame.
	d.badFrameCount = 0
	m := p.MBE()
	var log2M [mbe.MaxL + 1]float64
	mbe.PredictLog2Ml(&d.state, m, &log2M)

	var M [mbe.MaxL + 1]float64
	mbe.AmplitudesFromLog2Ml(&log2M, m.L, &M)
	mbe.EnhanceAmplitudes(m, &M)

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

	d.agc.Apply(pcm, out, recoveryFreezeAGC)
	return out, nil
}

// synthFrame runs the §6.3 voiced + §6.4 unvoiced overlap-add legs
// of the pipeline and rolls SynthState forward by log2M / M. Used
// by both the good-frame path and the bad-frame replay path so the
// two share identical synthesis behavior — the only difference is
// what M values get fed in.
func (d *Decoder) synthFrame(p mbe.Params, log2M *[mbe.MaxL + 1]float64, M *[mbe.MaxL + 1]float64, pcm []float64) {
	mbe.SynthVoiced(&d.state, p, M, pcm)
	noise := make([]float64, mbe.UnvoicedFFTSize)
	for i := range noise {
		noise[i] = d.rng.NormFloat64()
	}
	mbe.SynthUnvoicedOverlapAdd(&d.state, p, M, noise, pcm)
	d.state.UpdateLog2Ml(p, log2M)
	d.state.UpdateVoicedState(p, M)
}

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
	d.clearLastGood()
	d.recoveryFramesRemaining = 0
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
