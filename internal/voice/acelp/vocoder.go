package acelp

import "github.com/MattCheramie/GopherTrunk/internal/voice"

// VocoderName is the registry key for the TETRA ACELP decoder.
const VocoderName = "tetra-acelp"

// speechFrameBytes is the packed size of one 137-bit TCH/S speech frame
// (MSB-first, last byte padded).
const speechFrameBytes = (speechFrameBits + 7) / 8 // 18

// vocoder adapts the ACELP Decoder to the voice.Vocoder interface. One input
// frame is a packed 137-bit TCH/S speech frame; Decode returns 240 int16 PCM
// samples at 8 kHz (30 ms). A frame shorter than 18 bytes is treated as an
// erased frame (BFI), so the decoder runs its frame-repeat concealment.
type vocoder struct {
	d *Decoder
}

// NewVocoder returns a fresh TETRA ACELP vocoder.
func NewVocoder() voice.Vocoder { return &vocoder{d: NewDecoder()} }

func (v *vocoder) Name() string   { return VocoderName }
func (v *vocoder) FrameSize() int { return speechFrameBytes }
func (v *vocoder) Reset()         { v.d = NewDecoder() }
func (v *vocoder) Close() error   { return nil }

// Decode renders one packed 137-bit speech frame to 240 PCM samples. The raw
// Decod_Tetra synthesis is post-processed by the reference's saturating x2
// (Post_Process, EN 300 395-2) so the output level matches the ETSI reference
// decoder sample-for-sample; without it GT's audio is 6 dB below reference.
func (v *vocoder) Decode(frame []byte) ([]int16, error) {
	bfi := len(frame) < speechFrameBytes
	var bits []byte
	if !bfi {
		bits = make([]byte, speechFrameBits)
		for i := 0; i < speechFrameBits; i++ {
			if frame[i>>3]&(0x80>>uint(i&7)) != 0 {
				bits[i] = 1
			}
		}
	}
	pcm := v.d.Decode(bits, bfi)
	for i, sfin := range pcm {
		pcm[i] = addOp(sfin, sfin) // Post_Process: saturating x2
	}
	return pcm, nil
}

// Compile-time check that vocoder satisfies voice.Vocoder.
var _ voice.Vocoder = (*vocoder)(nil)

func init() {
	voice.DefaultRegistry.Register(VocoderName, func() (voice.Vocoder, error) {
		return NewVocoder(), nil
	})
}
