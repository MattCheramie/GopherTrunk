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
	// Distinct from mbelib's "imbe" so both backends can be linked
	// into the same binary; the daemon picks one in config.
	VocoderName = "imbe-go"

	// AGC parameters. The synthesizer's float output magnitude is
	// stable per-frame (R_M0-preserving §6.2 enhancement holds total
	// energy constant across a frame) but varies wildly between
	// frames depending on Tl, voicing, and the §6.4 noise draw.
	// Without an AGC every frame would be either clipped (loud frames)
	// or near-silent (quiet frames). The AGC tracks the per-frame
	// peak with fast-attack / slow-release smoothing, then scales each
	// frame so the smoothed envelope hits agcTargetPeak.
	//
	// agcTargetPeak = 24000 sits ~3 dB below int16 max so the soft
	// knee of the envelope tracker has headroom for transients.
	// agcAttack > agcRelease so loud onsets are caught quickly while
	// gain ramps back up slowly during quiet passages — standard
	// AGC behavior that keeps speech intelligible without pumping.
	// agcMinGain / agcMaxGain prevent the envelope from sending
	// silence to full scale or compressing extreme transients to
	// inaudible levels.
	agcTargetPeak = 24000.0
	agcAttack     = 0.4
	agcRelease    = 0.02
	agcMinGain    = 10.0
	agcMaxGain    = 1e5
	agcNoiseFloor = 1e-3
)

// Decoder is the pure-Go IMBE 4400 decoder. It owns one SynthState
// (cross-frame log2(Ml) prediction memory + voiced phase + amp
// memory), one math/rand source for the §6.4 unvoiced excitation
// noise, and one AGC envelope tracker — all per-call so concurrent
// calls on different decoders don't share state.
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
type Decoder struct {
	state SynthState
	rng   *rand.Rand
	agc   float64 // smoothed peak envelope; 0 = fresh (next frame seeds it)
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

	d.applyAGC(pcm, out, p.Silent)
	return out, nil
}

// applyAGC tracks the per-frame peak with fast-attack / slow-release
// smoothing and scales pcm so the smoothed envelope hits
// agcTargetPeak. Frames whose peak falls below agcNoiseFloor leave
// the envelope unchanged so a tail fade-out into silence doesn't
// drag the envelope up artificially.
//
// First-frame seed: when d.agc == 0 (fresh decoder, post-Reset), the
// envelope is initialised directly to peak rather than via the attack
// coefficient. Without this seed the first frame would emerge ~2.5×
// over-gained (envelope = 0.4 · peak ⇒ gain = target / (0.4 · peak)
// ⇒ output peak = 2.5 · target ⇒ int16 saturation).
//
// Frozen mode (freezeEnvelope = true): apply the existing envelope's
// gain without updating it. The Decode() silent path uses this so a
// brief silence frame doesn't shift the AGC envelope based on the
// small §6.4 overlap-add fade-out content. The first frame after
// silence then applies the same gain as the last frame before
// silence — no audible level jump on speech-pause-speech transitions.
// Stream re-sync via the public Reset() does clear the envelope.
//
// agcMinGain / agcMaxGain prevent the envelope from sending
// silence to full scale or compressing extreme transients to
// inaudible levels. After the gain multiply, samples beyond int16
// range are hard-clipped at ±32767.
func (d *Decoder) applyAGC(pcm []float64, out []int16, freezeEnvelope bool) {
	if !freezeEnvelope {
		var peak float64
		for _, v := range pcm {
			if a := math.Abs(v); a > peak {
				peak = a
			}
		}
		if d.agc == 0 && peak > agcNoiseFloor {
			// First-frame seed: skip attack smoothing so the first frame
			// lands at exactly agcTargetPeak instead of 2.5× over.
			d.agc = peak
		} else if peak > agcNoiseFloor {
			coef := agcAttack
			if peak < d.agc {
				coef = agcRelease
			}
			d.agc += (peak - d.agc) * coef
		}
	}
	envelope := d.agc
	if envelope < agcNoiseFloor {
		envelope = agcNoiseFloor
	}
	gain := agcTargetPeak / envelope
	if gain < agcMinGain {
		gain = agcMinGain
	} else if gain > agcMaxGain {
		gain = agcMaxGain
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
// amplitude memory, the §6.4 overlap-add tail, and the AGC
// envelope. Callers invoke it on stream re-sync (e.g., a frame-
// loss event from the upstream P25 LDU decoder) so the next frame
// starts from a clean baseline. The noise source is intentionally
// not re-seeded — noise reproducibility is a constructor concern
// (New / NewWithSeed), not a per-call concern.
func (d *Decoder) Reset() {
	d.state.Reset()
	d.agc = 0
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
