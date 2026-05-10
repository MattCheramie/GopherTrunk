package ambe2

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// Frame parameters per AMBE+2 2400 bps. Every frame carries 49
// information bits over 20 ms of audio at 8 kHz mono.
const (
	// InfoBits is the per-frame information-bit count after the
	// upstream protocol layer (P25 P2 / DMR / NXDN) has applied
	// its FEC and produced the bare AMBE+2 information bits.
	InfoBits = 49

	// FrameBytes is the byte count callers pass to Decode: 49 bits
	// round up to 7 bytes with 7 unused trailing bits. Matches what
	// the libmbe wrapper accepted so upstream callers don't need to
	// repack.
	FrameBytes = 7

	// VocoderName is the registry key the daemon resolves at
	// startup. Same name libmbe registered under so existing
	// configs work without change.
	VocoderName = "ambe2"
)

// Decoder is the pure-Go AMBE+2 2400 decoder. Mirrors the imbe
// Decoder shape: one mbe.SynthState (cross-frame log2(Ml)
// prediction + voiced phase + amp memory + §6.4 OA tail), one
// math/rand source for the unvoiced excitation noise, one
// *mbe.AGC, and a one-frame cache of the last-good params for
// the frame-repeat path — all per-call so concurrent calls on
// different decoders don't share state.
//
// AMBE+2 has one cross-frame quantity that IMBE lacks: the
// overall gamma (gain). Each frame's absolute gamma is
//
//	gamma_curr = DeltaGamma_curr + 0.5 * gamma_prev
//
// where DeltaGamma is the per-frame value decoded from
// AmbePlusDg[b₂]. prevGamma below caches the previous frame's
// resolved gamma so this frame's UnpackParams output can be
// folded into the shared mbe.Params shape (see foldGammaIntoTl).
type Decoder struct {
	state mbe.SynthState
	rng   *rand.Rand
	agc   *mbe.AGC

	// AMBE+2-specific cross-frame state. The absolute gamma each
	// frame is DeltaGamma + 0.5*prevGamma; this caches the
	// resolved gamma so the next frame's fold has it available.
	prevGamma float64

	// One-frame cache for the bad-frame replay path. Holds the
	// post-fold mbe.Params (Tl values already centered + gamma
	// folded in) so a replay can call mbe.PredictLog2Ml + the
	// shared synthesis primitives without re-running the fold.
	lastGoodParams mbe.Params
	lastGoodLog2M  [mbe.MaxL + 1]float64
	lastGoodM      [mbe.MaxL + 1]float64
	badFrameCount  int
}

// New returns a fresh Decoder. The unvoiced-excitation noise
// source is seeded from a fixed default so two decoders
// constructed via New() produce byte-identical output for the
// same frame stream (useful for tests + reproducibility).
// Production callers wanting genuinely-random noise across runs
// should use NewWithSeed with a time-derived seed.
func New() *Decoder {
	return NewWithSeed(0)
}

// NewWithSeed constructs a Decoder with an explicit seed for the
// internal noise source. AGC parameters use mbe.DefaultAGCConfig.
func NewWithSeed(seed int64) *Decoder {
	return NewWithConfig(seed, mbe.DefaultAGCConfig())
}

// NewWithConfig constructs a Decoder with an explicit noise seed +
// AGC configuration. Zero-value fields in cfg fall back to
// mbe.DefaultAGCConfig values, so callers can override only the
// parameters they care about. Mirrors imbe.NewWithConfig.
func NewWithConfig(seed int64, cfg mbe.AGCConfig) *Decoder {
	return &Decoder{
		rng: rand.New(rand.NewSource(seed)),
		agc: mbe.NewAGC(cfg),
	}
}

// Name returns the registry key. Matches VocoderName.
func (d *Decoder) Name() string { return VocoderName }

// FrameSize returns the per-frame input byte count (7 bytes / 49
// information bits with 7 trailing padding bits).
func (d *Decoder) FrameSize() int { return FrameBytes }

// Decode reads 49 information bits (post-FEC, MSB-first packed
// into the first 49 bits of a 7-byte frame; trailing 7 bits
// ignored) and returns 160 int16 PCM samples at 8 kHz.
//
// Frame disposition:
//
//   - good voice frame: full synthesis pipeline (UnpackParams →
//     gamma fold → mbe.PredictLog2Ml → mbe.AmplitudesFromLog2Ml →
//     unvoiced scaling → mbe.EnhanceAmplitudes → mbe.SynthVoiced
//     + mbe.SynthUnvoicedOverlapAdd → state update → AGC). Caches
//     params + log2M + M for the next bad frame's replay; resets
//     badFrameCount.
//   - tone frame (b₀ ∈ {0x7E, 0x7F}) or silence: emits the §6.4 OA
//     tail fade-out into pcm[0..95]; resets SynthState +
//     last-good cache + prevGamma + badFrameCount; AGC envelope
//     preserved.
//   - bad frame (UnpackParams error) with cached last-good frame
//     and badFrameCount < mbe.MaxBadFrames: replays last-good
//     params with M scaled by mbe.BadFrameAttenuation^badFrameCount;
//     AGC freezes so the attenuation is audible.
//   - bad frame with no cache, or after mbe.MaxBadFrames consecutive
//     bad frames: emits silence; clears caches; AGC envelope
//     preserved.
//
// Bad-frame replay is largely defensive — AmbePlusLtable guarantees
// every voice b₀ resolves to a valid L ∈ [9, 56], so UnpackParams
// only returns an error on a wrong-length input (which Decode
// catches earlier). The structural path is retained to match the
// imbe Decoder's shape and to give future protocol-layer FEC
// signaling a place to land.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("ambe2: frame must be %d bytes (49 bits + 7 padding), got %d", FrameBytes, len(frame))
	}

	info := unpackInfoBits(frame)
	out := make([]int16, mbe.SamplesPerFrame)
	pcm := make([]float64, mbe.SamplesPerFrame)

	p, err := UnpackParams(info)

	switch {
	case err != nil && d.lastGoodParams.L > 0 && d.badFrameCount < mbe.MaxBadFrames:
		// Frame repeat: replay the last-good params with progressive
		// per-frame attenuation.
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
		// Bad frame with no cache or budget exhausted: emit silence
		// + clear state.
		d.state.Reset()
		d.clearLastGood()
		d.prevGamma = 0
		d.agc.Apply(pcm, out, true)
		return out, nil

	case p.Tone || p.Silent:
		// Tone-frame path: for now, emit the OA tail fade-out into
		// pcm[0..95] and reset state. Proper tone synthesis (single +
		// dual sinewave) is a follow-up; the tone-index extraction is
		// already preserved on p.B1/p.B2 for that work. Silent
		// frames (invalid tone index) hit the same path.
		mbe.SynthUnvoicedOverlapAdd(&d.state, p.Params, nil, nil, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.prevGamma = 0
		d.agc.Apply(pcm, out, true)
		return out, nil
	}

	// Good voice frame.
	d.badFrameCount = 0

	// Combine the per-frame delta with prev-frame gamma to recover
	// the absolute gain, then fold gamma + DC removal into Tl so
	// the shared mbe.PredictLog2Ml produces AMBE+2-spec output
	// without needing an AMBE+2-aware variant.
	gamma := p.DeltaGamma + 0.5*d.prevGamma
	folded := foldGammaIntoTl(p, gamma)

	var log2M [mbe.MaxL + 1]float64
	mbe.PredictLog2Ml(&d.state, folded, &log2M)

	var M [mbe.MaxL + 1]float64
	mbe.AmplitudesFromLog2Ml(&log2M, folded.L, &M)

	// AMBE+2-specific unvoiced amplitude scaling: unvoiced harmonics
	// are attenuated by Unvc = 0.2046/√w₀ before spectral
	// enhancement + synthesis. mbelib applies this between the
	// log2Ml→Ml conversion and the §6.2 enhancement, so we mirror
	// that ordering.
	for l := 1; l <= folded.L; l++ {
		if folded.Vl[l] == 0 {
			M[l] *= p.Unvc
		}
	}

	mbe.EnhanceAmplitudes(folded, &M)

	d.synthFrame(folded, &log2M, &M, pcm)

	// Cache for the frame-repeat path on a future bad frame.
	d.lastGoodParams = folded
	d.lastGoodLog2M = log2M
	d.lastGoodM = M
	d.prevGamma = gamma

	d.agc.Apply(pcm, out, false)
	return out, nil
}

// synthFrame runs the §6.3 voiced + §6.4 unvoiced overlap-add legs
// of the shared mbe synthesis and rolls SynthState forward. Used
// by both the good-frame path and the bad-frame replay path so the
// two share identical synthesis behaviour — only the M values
// differ between the two.
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
func (d *Decoder) clearLastGood() {
	d.lastGoodParams = mbe.Params{}
	d.lastGoodLog2M = [mbe.MaxL + 1]float64{}
	d.lastGoodM = [mbe.MaxL + 1]float64{}
	d.badFrameCount = 0
}

// unpackInfoBits expands the 7-byte (MSB-first) packed frame into a
// 49-element 0/1 byte slice — the shape UnpackParams expects. The
// trailing 7 bits of byte 6 are padding and ignored.
func unpackInfoBits(frame []byte) []byte {
	info := make([]byte, InfoBits)
	for i := 0; i < InfoBits; i++ {
		info[i] = (frame[i/8] >> (7 - uint(i)%8)) & 1
	}
	return info
}

// foldGammaIntoTl projects ambe2.Params into mbe.Params with the
// AMBE+2-specific gamma folded into Tl. The shared mbe.PredictLog2Ml
// expects the IMBE-style form
//
//	log2Ml[l] = γ·(prev_interp[l] − mean_prev_interp) + Tl[l]
//
// whereas AMBE+2's spec form is
//
//	log2Ml[l] = γ·(prev_interp[l] − mean_prev_interp)
//	            + (Tl[l] − mean(Tl))
//	            + (gamma − 0.5·log2(L))
//
// Folding Tl_folded[l] = Tl[l] − mean(Tl) + (gamma − 0.5·log2(L))
// up front makes the shared core produce the right output without
// an AMBE+2-aware variant.
//
// gamma is the absolute gain (DeltaGamma + 0.5·prev_gamma); the
// caller (Decoder.Decode) resolves it from cross-frame state.
//
// Silence + zero-L frames are returned as-is (Tl untouched) so
// callers don't need an extra guard.
func foldGammaIntoTl(p Params, gamma float64) mbe.Params {
	if p.L == 0 {
		return p.Params
	}
	var meanTl float64
	for l := 1; l <= p.L; l++ {
		meanTl += p.Tl[l]
	}
	meanTl /= float64(p.L)

	bigGamma := gamma - 0.5*math.Log2(float64(p.L))

	folded := p.Params // value copy of the embedded mbe.Params
	for l := 1; l <= p.L; l++ {
		folded.Tl[l] = p.Tl[l] - meanTl + bigGamma
	}
	return folded
}

// Reset clears all per-call synthesis state — the cross-frame
// log-amplitude prediction history, the voiced harmonic phase +
// amplitude memory, the §6.4 overlap-add tail, the AMBE+2 gamma
// memory, the AGC envelope, and the frame-repeat cache + bad-frame
// counter. Callers invoke it on stream re-sync (e.g., a frame-loss
// event from the upstream P25 P2 / DMR / NXDN decoder) so the
// next frame starts from a clean baseline.
func (d *Decoder) Reset() {
	d.state.Reset()
	d.agc.Reset()
	d.clearLastGood()
	d.prevGamma = 0
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
