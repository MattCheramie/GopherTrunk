package imbe

import (
	"fmt"

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
)

// Decoder is the in-progress pure-Go IMBE decoder. The current
// implementation accepts validly-sized frames and returns silence —
// the channel-coding inverse, parameter unpacking, and speech
// synthesis land in follow-up PRs (see doc.go for the sequence).
// Wiring the daemon's recorder + composer to it now means the
// audio path lights up for free as each follow-up merges.
type Decoder struct {
	// Reserved for future per-frame state (last fundamental
	// frequency, last spectral amplitudes for frame-repeat on
	// bad-frame indicator, voicing memory). Holding the field now
	// keeps later PRs contained to logic changes.
	silenceBuf [SamplesPerFrame]int16
}

// New returns a fresh Decoder. Each call gets its own instance so
// the future per-call state stays isolated.
func New() *Decoder {
	return &Decoder{}
}

// Name returns the registry key. Matches VocoderName.
func (d *Decoder) Name() string { return VocoderName }

// FrameSize returns the per-frame input byte count (11 bytes / 88
// bits, packed MSB-first).
func (d *Decoder) FrameSize() int { return FrameBytes }

// Decode validates the frame size and returns a frame's worth of
// silence. When the synthesis layer lands, this stub becomes a
// drop-in: callers see audio appear without changing how they
// invoke the decoder.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("imbe: frame must be %d bytes (88 bits), got %d", FrameBytes, len(frame))
	}
	out := make([]int16, SamplesPerFrame)
	copy(out, d.silenceBuf[:])
	return out, nil
}

// Reset clears any per-call state. No-op until the synthesis layer
// adds frame-to-frame memory.
func (d *Decoder) Reset() {}

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
