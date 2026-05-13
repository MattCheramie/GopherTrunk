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

	// Tone-frame sine oscillator phase. Carried across consecutive
	// tone frames so a sustained tone at one frequency is
	// click-free between frame boundaries. Cleared on any non-tone
	// frame (voice / silence / dual-tone fallback) and on Reset.
	tonePhase float64
	// Second oscillator phase used for dual-tone (DTMF) synthesis.
	// The first oscillator runs from tonePhase; toneDualPhase
	// tracks the second sinusoid so a held DTMF key stays
	// click-free across frame boundaries. Cleared whenever
	// tonePhase is cleared.
	toneDualPhase float64

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

	case p.Tone && !p.Silent && p.B1 >= 5 && p.B1 <= 122:
		// Single tone: synthesise a sinewave at b1·31.25 Hz scaled
		// by b2. Phase is carried across consecutive tone frames
		// in d.tonePhase so a held tone is click-free. Voice
		// SynthState resets — voice + tone state are orthogonal.
		// AGC tracks (no freeze) so b2 volume changes are audible.
		d.synthSingleTone(p.B1, p.B2, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.prevGamma = 0
		d.agc.Apply(pcm, out, false)
		return out, nil

	case p.Tone && !p.Silent && p.B1 >= 128 && p.B1 <= 143:
		// DTMF dual-tone (b1 ∈ [128, 143]): summed sinewaves at
		// the 16-key DTMF matrix (ITU-T Q.23: 4 row × 4 column
		// frequencies). The b1 → key mapping follows the AMBE+2
		// tone-frame layout shared by mbelib + DSD-FME +
		// DSDcc — 128 is "1", 143 is "D". See ambeDualToneTable
		// below for the per-index frequency pair.
		//
		// Knox / call-alert dual-tones (b1 ∈ [144, 163]) are
		// vendor-specific; operators that register a per-vendor
		// override via SetKnoxTone get dual-tone synthesis there
		// too (case branch below). Without an override, those
		// indices fall through to the silence branch.
		freqA, freqB := ambeDualToneTable[p.B1-128][0], ambeDualToneTable[p.B1-128][1]
		d.synthDualTone(freqA, freqB, p.B2, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.prevGamma = 0
		d.agc.Apply(pcm, out, false)
		return out, nil

	case p.Tone && !p.Silent && p.B1 >= KnoxIndexLow && p.B1 <= KnoxIndexHigh:
		// Knox / call-alert dual-tone (b1 ∈ [144, 163]): the public
		// AMBE+2 spec doesn't document these frequencies, but
		// operators with a vendor reference can register pairs via
		// ambe2.SetKnoxTone. When a registration exists for b1,
		// synthesise the same summed-sinewave dual-tone DTMF uses;
		// otherwise fall through to the silence branch below.
		if freqA, freqB, ok := KnoxTone(p.B1); ok {
			d.synthDualTone(freqA, freqB, p.B2, pcm)
			d.state.Reset()
			d.clearLastGood()
			d.prevGamma = 0
			d.agc.Apply(pcm, out, false)
			return out, nil
		}
		fallthrough

	case p.Tone || p.Silent:
		// Knox / call-alert dual-tone (b1 ∈ [144, 163]) without a
		// registered override, invalid tone index, or explicit
		// silence: emit the §6.4 OA tail fade-out into pcm[0..95]
		// and reset state. The vendor-specific frequency table
		// for knox tones isn't in the public AMBE+2 spec;
		// operators who need them can call SetKnoxTone from their
		// vendor-specific init() and add their per-vendor pairs.
		mbe.SynthUnvoicedOverlapAdd(&d.state, p.Params, nil, nil, pcm)
		d.state.Reset()
		d.clearLastGood()
		d.prevGamma = 0
		d.tonePhase = 0
		d.toneDualPhase = 0
		d.agc.Apply(pcm, out, true)
		return out, nil
	}

	// Good voice frame. Clear tonePhase so a tone → voice → tone
	// sequence doesn't pick up the pre-voice phase on the second
	// tone (the voice frame breaks oscillator continuity).
	d.badFrameCount = 0
	d.tonePhase = 0
	d.toneDualPhase = 0

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

// synthSingleTone fills pcm[0..mbe.SamplesPerFrame-1] with a
// sinewave at b1·31.25 Hz scaled by b2-derived amplitude. The
// oscillator phase is carried across frames in d.tonePhase so two
// consecutive tone frames at the same b1 are click-free.
//
// b1 is the AMBE+2 single-tone index in [5, 122] (callers gate
// for that range). The corresponding frequency is b1·31.25 Hz,
// covering 156.25 Hz at b1=5 up to 3812.5 Hz at b1=122 — the
// audible band an 8 kHz Nyquist supports.
//
// b2 is the 8-bit volume index from the AMBE+2 tone-frame
// extraction. We map it to a peak amplitude of (b2/255)·8192 —
// the absolute level is mostly cosmetic since the AGC normalises
// to AGCConfig.TargetPeak, but the b2 scaling makes per-frame
// volume changes audible within a single tone sequence (before
// the AGC has time to compensate).
func (d *Decoder) synthSingleTone(b1, b2 int, pcm []float64) {
	const (
		toneStepHz   = 31.25  // b1 step from §AMBE+2 tone-frame definition
		peakAmpScale = 8192.0 // pre-AGC headroom for the synthesised tone
		volMax       = 255.0  // b2 is 8-bit
	)
	freqHz := float64(b1) * toneStepHz
	amp := peakAmpScale * float64(b2) / volMax
	twoPi := 2 * math.Pi
	phaseStep := twoPi * freqHz / float64(mbe.PCMSampleRate)
	for n := 0; n < mbe.SamplesPerFrame; n++ {
		pcm[n] = amp * math.Sin(d.tonePhase)
		d.tonePhase += phaseStep
	}
	// Wrap phase to keep it bounded across long-running tone
	// sequences so the float64 accumulator doesn't drift.
	d.tonePhase = math.Mod(d.tonePhase, twoPi)
	if d.tonePhase < 0 {
		d.tonePhase += twoPi
	}
}

// synthDualTone fills pcm with the sum of two equal-amplitude
// sinewaves at freqA and freqB Hz, scaled by b2 (the same 8-bit
// volume index single-tone uses). Each oscillator carries its own
// phase across frame boundaries (d.tonePhase + d.toneDualPhase) so
// a held DTMF key is click-free across the two-frame minimum.
//
// Per-tone amplitude is half of the single-tone peak so the summed
// peak still fits the same headroom (avoiding pre-AGC clipping on
// the simultaneous-phase sample). The AGC normalises afterwards
// the same way as single-tone, so the two paths land at the same
// loudness.
func (d *Decoder) synthDualTone(freqA, freqB float64, b2 int, pcm []float64) {
	const (
		peakAmpScale = 8192.0 // matches synthSingleTone headroom
		volMax       = 255.0
	)
	twoPi := 2 * math.Pi
	// Half-amplitude per oscillator so the summed peak ≈ peakAmpScale
	// at perfect phase alignment.
	amp := 0.5 * peakAmpScale * float64(b2) / volMax
	stepA := twoPi * freqA / float64(mbe.PCMSampleRate)
	stepB := twoPi * freqB / float64(mbe.PCMSampleRate)
	for n := 0; n < mbe.SamplesPerFrame; n++ {
		pcm[n] = amp*math.Sin(d.tonePhase) + amp*math.Sin(d.toneDualPhase)
		d.tonePhase += stepA
		d.toneDualPhase += stepB
	}
	d.tonePhase = math.Mod(d.tonePhase, twoPi)
	if d.tonePhase < 0 {
		d.tonePhase += twoPi
	}
	d.toneDualPhase = math.Mod(d.toneDualPhase, twoPi)
	if d.toneDualPhase < 0 {
		d.toneDualPhase += twoPi
	}
}

// ambeDualToneTable maps AMBE+2 tone-frame b1 indices in the
// dual-tone range [128, 143] to their (low, high) DTMF frequency
// pair in Hz. ITU-T Q.23 defines the standard 4×4 DTMF matrix
// (rows 697 / 770 / 852 / 941 Hz × cols 1209 / 1336 / 1477 /
// 1633 Hz). The b1 → key mapping follows the AMBE+2 layout shared
// across mbelib / DSDcc / DSD-FME: b1=128 is "1", 143 is "D".
//
// Indices [144, 163] are knox / call-alert tones whose
// frequencies are vendor-specific (Motorola Trbo vs. Hytera vs.
// generic). The public AMBE+2 spec doesn't document them, so
// those rows stay at the zero default and the decoder's case
// branch routes them to silence. Operators with a specific
// vendor's reference can extend the table below.
var ambeDualToneTable = [36][2]float64{
	// DTMF 1..D, indexed by b1 - 128.
	0: {697, 1209},  // 128: "1"
	1: {697, 1336},  // 129: "2"
	2: {697, 1477},  // 130: "3"
	3: {697, 1633},  // 131: "A"
	4: {770, 1209},  // 132: "4"
	5: {770, 1336},  // 133: "5"
	6: {770, 1477},  // 134: "6"
	7: {770, 1633},  // 135: "B"
	8: {852, 1209},  // 136: "7"
	9: {852, 1336},  // 137: "8"
	10: {852, 1477}, // 138: "9"
	11: {852, 1633}, // 139: "C"
	12: {941, 1209}, // 140: "*"
	13: {941, 1336}, // 141: "0"
	14: {941, 1477}, // 142: "#"
	15: {941, 1633}, // 143: "D"
	// Indices 16..35 (b1 144..163) are vendor-specific knox /
	// call-alert tones — left at zero; decoder routes them to
	// silence until a per-vendor table lands.
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
	d.tonePhase = 0
	d.toneDualPhase = 0
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
