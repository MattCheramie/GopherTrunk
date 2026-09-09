package phase2

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 Phase 2 voice bursts: where the AMBE+2 codewords sit in a burst, and how
// one is lifted out of it.
//
// A voice burst has the same 180-dibit shape as an ACCH burst — the DUID names
// which it is — and its four voice frames are threaded *between* the four
// scattered DUID dibits, not laid out contiguously:
//
//	dibit  20        DUID
//	dibit  21..56    voice frame 0        (36 dibits)
//	dibit  57        DUID
//	dibit  58..93    voice frame 1
//	dibit  94..105   ESS (encryption sync fragment)
//	dibit 106..141   voice frame 2
//	dibit 142        DUID
//	dibit 143..178   voice frame 3
//	dibit 179        DUID
//
// A 2V burst carries frames 0 and 1 and the ESS; a 4V burst carries all four.
// Both references agree on every offset, and the geometry is self-checking:
// the frame boundaries land exactly on the DUID dibits.
//
// The payload is PN44-scrambled like an ACCH burst's, and at the same slot
// phase — this package previously extracted voice frames without descrambling
// at all, from an even 36-dibit grid starting at dibit 32, selected by an ISCH
// slot type that does not match the air (issue #915). Three independent
// reasons for a voice burst to yield nothing.

// Voice burst geometry.
const (
	// VoiceCodewordDibits is the on-wire width of one FEC-wrapped AMBE+2
	// voice codeword: 72 bits.
	VoiceCodewordDibits = 36
	// ESSDibitOffset and ESSDibits locate the encryption-sync fragment that
	// follows the second voice frame in every voice burst.
	ESSDibitOffset = 94
	ESSDibits      = 12
)

// voiceFrameOffsets are the burst-dibit offsets of the voice codewords.
var voiceFrameOffsets = [4]int{21, 58, 106, 143}

// VoiceFrameOffsets returns the burst-dibit offsets of the voice codewords a
// burst of this type carries, or nil if it carries none.
func VoiceFrameOffsets(bt BurstType) []int {
	switch bt {
	case BurstVoice4:
		return voiceFrameOffsets[:4]
	case BurstVoice2:
		return voiceFrameOffsets[:2]
	}
	return nil
}

// IsVoice reports whether the burst carries AMBE+2 voice.
func (b BurstType) IsVoice() bool { return b == BurstVoice4 || b == BurstVoice2 }

// ErrNotVoiceBurst is returned when a burst's DUID does not name a voice type.
var ErrNotVoiceBurst = errors.New("p25/phase2: burst is not a voice burst")

// voiceDeinterleave maps each of the 72 on-air bits of a voice codeword to its
// position in the concatenated c0‖c1‖c2‖c3 codeword, most-significant bit of
// each field first.
//
// The interleave is a 4 × 18 column-major deal: writing the four codeword
// fields end to end as a 72-bit sequence, on-air bit 4k+c carries sequence
// element 18c+k. So the first column is c0's top 18 bits, the second finishes
// c0 and starts c1, and so on. Derived from the pattern in the reference
// implementations rather than transcribed from their tables, and pinned by
// TestVoiceDeinterleaveIsColumnMajor.
var voiceDeinterleave = func() [72]int {
	var m [72]int
	for k := 0; k < 18; k++ {
		for c := 0; c < 4; c++ {
			m[4*k+c] = 18*c + k
		}
	}
	return m
}()

// Field widths of the AMBE+2 voice codeword, in c0‖c1‖c2‖c3 order.
const (
	vcwC0Bits = 24 // Golay(24,12) protected
	vcwC1Bits = 23 // Golay(23,12) protected, masked by a PRNG keyed on u0
	vcwC2Bits = 11 // unprotected
	vcwC3Bits = 14 // unprotected
	// VoiceInfoBits is the recovered vocoder frame: u0‖u1‖u2‖u3.
	VoiceInfoBits = 12 + 12 + vcwC2Bits + vcwC3Bits // 49
)

// c1Mask returns the 23-bit pseudo-random mask applied to c1, generated from
// the decoded u0 by the AMBE+2 PRNG: seed 16*u0, then
// p[n] = (173*p[n-1] + 13849) mod 65536, taking bit 15 of each step.
func c1Mask(u0 uint16) uint32 {
	p := uint32(u0) * 16
	var mask uint32
	for n := 0; n < 23; n++ {
		p = (173*p + 13849) & 0xFFFF
		mask = mask<<1 | (p>>15)&1
	}
	return mask
}

// DecodeVoiceCodeword recovers one AMBE+2 vocoder frame from a
// VoiceCodewordDibits-long, already-descrambled voice codeword.
//
// It returns the VoiceInfoBits information bits (u0‖u1‖u2‖u3) and the number of
// bit errors the two Golay decoders corrected. The error is non-nil only for a
// malformed input length — the code is perfect, so the decode itself always
// produces a codeword.
func DecodeVoiceCodeword(dibits []uint8) ([]byte, int, error) {
	if len(dibits) != VoiceCodewordDibits {
		return nil, 0, fmt.Errorf("p25/phase2: voice codeword needs %d dibits, got %d",
			VoiceCodewordDibits, len(dibits))
	}
	onAir := framing.DibitsToBits(dibits)
	var cw [72]byte
	for i, pos := range voiceDeinterleave {
		cw[pos] = onAir[i]
	}

	pack := func(bits []byte) uint32 {
		var v uint32
		for _, b := range bits {
			v = v<<1 | uint32(b&1)
		}
		return v
	}
	// c0 is the extended (24,12) form: the 23-bit cyclic codeword followed by
	// an overall parity bit, which is dropped here — the cyclic decoder
	// already corrects to the code's full radius.
	c0 := pack(cw[:vcwC0Bits]) >> 1
	c1 := pack(cw[vcwC0Bits : vcwC0Bits+vcwC1Bits])

	u0, e0 := framing.GolayDecodeCyclic23_12(c0)
	u1, e1 := framing.GolayDecodeCyclic23_12(c1 ^ c1Mask(u0))

	out := make([]byte, 0, VoiceInfoBits)
	for i := 11; i >= 0; i-- {
		out = append(out, byte(u0>>uint(i)&1))
	}
	for i := 11; i >= 0; i-- {
		out = append(out, byte(u1>>uint(i)&1))
	}
	out = append(out, cw[vcwC0Bits+vcwC1Bits:]...)

	// The cyclic Golay(23,12) is perfect, so neither decode can fail: every
	// received word is within three bits of exactly one codeword. What the
	// error count reports is how hard it had to work, and a burst sitting at
	// the radius on every frame is the signature of a wrong descramble or a
	// mis-framed burst rather than a marginal one.
	return out, e0 + e1, nil
}

// ExtractBurstVoiceFrames pulls every AMBE+2 frame out of a voice burst.
//
// burst must be BurstDibits long and still scrambled, exactly as it came off
// the air; slot is its position within the superframe and seq one superframe of
// PN44, as for DecodeACCHBurst. Voice payload is scrambled like everything else
// in the burst.
//
// frames[i] is a 7-byte vocoder frame for internal/voice/ambe2. A frame whose
// Golay failed is returned zeroed so a caller can repeat the previous frame
// rather than drop the slot, and is counted in uncorrectable.
func ExtractBurstVoiceFrames(burst []uint8, slot int, seq []byte) (frames [][]byte, errs, uncorrectable int, err error) {
	bt := BurstTypeOf(burst)
	offsets := VoiceFrameOffsets(bt)
	if offsets == nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrNotVoiceBurst, bt)
	}
	payload := make([]uint8, PayloadDibits)
	copy(payload, burst[ISCHRegionDibits:BurstDibits])
	if len(seq) >= SuperframeScrambleBits {
		bits := framing.DibitsToBits(payload)
		off := slot*2*BurstDibits + ScrambleOriginBit
		for i := range bits {
			bits[i] ^= seq[(off+i)%SuperframeScrambleBits]
		}
		payload = framing.BitsToDibits(bits)
	}

	frames = make([][]byte, len(offsets))
	for i, off := range offsets {
		p := off - ISCHRegionDibits // payload-relative
		info, n, decErr := DecodeVoiceCodeword(payload[p : p+VoiceCodewordDibits])
		errs += n
		if decErr != nil {
			uncorrectable++
			frames[i] = make([]byte, (VoiceInfoBits+7)/8)
			continue
		}
		frames[i] = framing.PackBitsMSB(info)
	}
	return frames, errs, uncorrectable, nil
}

// EncodeVoiceCodeword is the inverse of DecodeVoiceCodeword: it wraps
// VoiceInfoBits information bits in the AMBE+2 FEC and interleaves them into a
// VoiceCodewordDibits-long on-air codeword. Used to build fixtures.
func EncodeVoiceCodeword(info []byte) []uint8 {
	var cw [72]byte
	take := func(off, n int) uint16 {
		var v uint16
		for i := 0; i < n; i++ {
			v = v<<1 | uint16(info[off+i]&1)
		}
		return v
	}
	u0 := take(0, 12)
	u1 := take(12, 12)

	c0 := framing.GolayEncodeCyclic23_12(u0)
	// The extended form appends an overall parity bit.
	parity := byte(framing.PopCount64(uint64(c0)) & 1)
	for i := 0; i < 23; i++ {
		cw[i] = byte(c0 >> uint(22-i) & 1)
	}
	cw[23] = parity

	c1 := framing.GolayEncodeCyclic23_12(u1) ^ c1Mask(u0)
	for i := 0; i < vcwC1Bits; i++ {
		cw[vcwC0Bits+i] = byte(c1 >> uint(vcwC1Bits-1-i) & 1)
	}
	copy(cw[vcwC0Bits+vcwC1Bits:], info[24:])

	onAir := make([]byte, 72)
	for i, pos := range voiceDeinterleave {
		onAir[i] = cw[pos]
	}
	return framing.BitsToDibits(onAir)
}

// EncodeVoiceBurst builds a BurstDibits-long voice burst carrying the given
// vocoder frames, scrambled for slot.
//
// Each frame is a packed vocoder frame in the form ExtractBurstVoiceFrames
// returns — VoiceInfoBits information bits packed MSB-first into
// VoiceFrameBytes bytes — so the two are inverses. A missing or short frame is
// encoded as silence.
func EncodeVoiceBurst(bt BurstType, frames [][]byte, slot int, seq []byte) []uint8 {
	offsets := VoiceFrameOffsets(bt)
	if offsets == nil {
		return nil
	}
	payload := make([]uint8, PayloadDibits)
	for i, off := range offsets {
		info := make([]byte, VoiceInfoBits)
		if i < len(frames) && len(frames[i]) >= VoiceFrameBytes {
			copy(info, framing.UnpackBitsMSB(frames[i][:VoiceFrameBytes], VoiceInfoBits))
		}
		copy(payload[off-ISCHRegionDibits:], EncodeVoiceCodeword(info))
	}
	if len(seq) >= SuperframeScrambleBits {
		bits := framing.DibitsToBits(payload)
		off := slot*2*BurstDibits + ScrambleOriginBit
		for i := range bits {
			bits[i] ^= seq[(off+i)%SuperframeScrambleBits]
		}
		payload = framing.BitsToDibits(bits)
	}
	burst := make([]uint8, BurstDibits)
	copy(burst[ISCHRegionDibits:], payload)
	var parity byte
	for bit := 0; bit < 4; bit++ {
		if byte(bt)&(1<<bit) != 0 {
			parity ^= duidParityRows[bit]
		}
	}
	cwDUID := byte(bt)<<4 | parity
	for i, pos := range duidDibitPositions {
		burst[pos] = cwDUID >> (6 - 2*i) & 3
	}
	return burst
}

// VoiceBurstGolayErrors reports how many bits the two Golay decoders correct
// across every voice frame of a burst descrambled at slot. It is the
// slot-phase discriminator for a superframe that carries no decodable ACCH
// burst: the right phase leaves the FEC with nothing to do, and a wrong one
// sits near the code's correction radius on every frame.
func VoiceBurstGolayErrors(burst []uint8, slot int, seq []byte) (errs int, ok bool) {
	frames, errs, _, err := ExtractBurstVoiceFrames(burst, slot, seq)
	if err != nil || len(frames) == 0 {
		return 0, false
	}
	return errs, true
}

// voiceSlotProbe is the cheap form of VoiceBurstGolayErrors used by the
// slot-phase vote: it descrambles and Golay-checks only the burst's first
// voice frame. One frame already separates the phases — distance 0 against
// roughly 3 — and doing all four costs four times as much on a path that runs
// six candidates for every sub-frame of every superframe.
func voiceSlotProbe(burst []uint8, slot int, seq []byte) (errs int, ok bool) {
	if len(burst) < BurstDibits || !BurstTypeOf(burst).IsVoice() {
		return 0, false
	}
	const off = 21 // first voice frame
	src := burst[off : off+VoiceCodewordDibits]
	if len(seq) >= SuperframeScrambleBits {
		bits := framing.DibitsToBits(src)
		base := slot*2*BurstDibits + ScrambleOriginBit + (off-ISCHRegionDibits)*2
		for i := range bits {
			bits[i] ^= seq[(base+i)%SuperframeScrambleBits]
		}
		src = framing.BitsToDibits(bits)
	}
	onAir := framing.DibitsToBits(src)
	var cw [72]byte
	for i, pos := range voiceDeinterleave {
		cw[pos] = onAir[i]
	}
	var c0 uint32
	for i := 0; i < vcwC0Bits; i++ {
		c0 = c0<<1 | uint32(cw[i]&1)
	}
	_, e := framing.GolayDecodeCyclic23_12(c0 >> 1)
	return e, true
}
