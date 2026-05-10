package imbe

import (
	"fmt"
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
	// Distinct from mbelib's "imbe" so both backends can be linked
	// into the same binary; the daemon picks one in config.
	VocoderName = "imbe-go"

	// pcmGain scales the float64 synthesis output before int16
	// clipping. The voiced step sums L sinusoids of up to
	// O(unit) amplitude each, so the instantaneous peak can reach
	// ~L when all cosines align; with L ≤ 56, picking 4096 (2^12)
	// gives ~3 bits of headroom for the typical mid-L voiced+unvoiced
	// mix without constantly saturating int16. The §6.2 spectral
	// amplitude enhancement (a follow-up polish PR) will replace
	// this with the spec-derived gain.
	pcmGain = 4096
)

// Decoder is the pure-Go IMBE 4400 decoder. It owns one SynthState
// (cross-frame log2(Ml) prediction memory + voiced phase + amp
// memory) and one math/rand source for the §6.4 unvoiced excitation
// noise — both per-call so concurrent calls on different decoders
// don't share state.
//
// Decode() runs the full TIA-102.BABA pipeline:
//   - bytes → 88 info bits
//   - UnpackParams: §5.3 / §5.4 / Annex E
//   - PredictLog2Ml: §6.1 cross-frame prediction
//   - AmplitudesFromLog2Ml: log2(Ml) → linear Ml
//   - SynthVoiced: §6.3 voiced harmonic generator
//   - SynthUnvoicedFromNoise: §6.4 unvoiced FFT excitation
//   - Update{Log2Ml,VoicedState}: roll state forward
//   - hard-clip + scale to int16 PCM
//
// Audio quality is "first pass": the §6.4 overlap-add window and
// §6.2 spectral amplitude enhancement are quality-polish steps that
// land in subsequent PRs. Without them the output has frame-edge
// click artifacts and untilted spectral envelope, but is otherwise
// intelligible voice.
type Decoder struct {
	state SynthState
	rng   *rand.Rand
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
// calls don't share the same noise stream.
func NewWithSeed(seed int64) *Decoder {
	return &Decoder{
		rng: rand.New(rand.NewSource(seed)),
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
// Frames with an invalid fundamental-frequency parameter (b_0 ≥ 220
// or b_0 in {208..215}) decode as silence so the upstream P25 LDU
// FEC slip doesn't crash the audio path; on these frames the
// SynthState is left untouched so the next valid frame picks up
// from the last-known-good prediction history.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("imbe: frame must be %d bytes (88 bits), got %d", FrameBytes, len(frame))
	}

	info := unpackInfoBits(frame)
	out := make([]int16, SamplesPerFrame)

	p, err := UnpackParams(info)
	if err != nil {
		// Bad frame — graceful silence. State preserved so the next
		// valid frame's cross-frame prediction has the previous
		// good frame's history.
		return out, nil
	}

	pcm := make([]float64, SamplesPerFrame)

	if p.Silent {
		// b_0 ∈ [216, 219]: explicit silence indicator. Run the §6.4
		// overlap-add with no new noise so the prev-frame unvoiced
		// tail still fades into pcm[0..95] (no click on the silence
		// boundary), then reset all cross-frame state so the next
		// non-silent frame starts from a clean baseline (§6.1
		// prediction needs no prev-frame anchor on a silence
		// boundary).
		SynthUnvoicedOverlapAdd(&d.state, p, nil, nil, pcm)
		d.state.Reset()
	} else {
		// §6.1: cross-frame log-amplitude recovery.
		var log2M [57]float64
		PredictLog2Ml(&d.state, p, &log2M)

		// log2(Ml) → linear Ml, then §6.2 spectral-amplitude
		// enhancement (per-harmonic weight from R_M0 + R_M1, plus
		// energy-preserving rescale).
		var M [57]float64
		AmplitudesFromLog2Ml(&log2M, p.L, &M)
		EnhanceAmplitudes(p, &M)

		// §6.3: voiced harmonic generator (additive into pcm).
		SynthVoiced(&d.state, p, &M, pcm)

		// §6.4: unvoiced FFT excitation with overlap-add (additive
		// into pcm). Threads PrevUnvoicedTail through SynthState so
		// frame boundaries are click-free.
		noise := make([]float64, UnvoicedFFTSize)
		for i := range noise {
			noise[i] = d.rng.NormFloat64()
		}
		SynthUnvoicedOverlapAdd(&d.state, p, &M, noise, pcm)

		// Roll cross-frame state forward.
		d.state.UpdateLog2Ml(p, &log2M)
		d.state.UpdateVoicedState(p, &M)
	}

	// Hard-clip + scale to int16. The pcmGain placeholder gives
	// headroom; the §6.2 enhancement polish PR will replace it with
	// the spec gain.
	for i, v := range pcm {
		s := v * pcmGain
		if s > 32767 {
			s = 32767
		} else if s < -32768 {
			s = -32768
		}
		out[i] = int16(s)
	}
	return out, nil
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
// amplitude memory, and any future overlap-add tail. Callers
// invoke it on stream re-sync (e.g., a frame-loss event from the
// upstream P25 LDU decoder) so the next frame starts from a clean
// baseline. The noise source is intentionally not re-seeded —
// noise reproducibility is a constructor concern (New /
// NewWithSeed), not a per-call concern.
func (d *Decoder) Reset() {
	d.state.Reset()
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
