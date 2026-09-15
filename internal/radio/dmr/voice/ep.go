package voice

import (
	"fmt"
	"math/bits"

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
)

// DMR "Enhanced Privacy" (DMRA algorithm 0x21, RC4) — the known-key
// descramble for the voice path (issue #1187).
//
// The construction below is the one two independent open decoders agree on
// and that DSD-FME has verified on air; SDRTrunk (Apache-2.0) parses the same
// headers and embedded parameters but does not decrypt. Nothing here is
// ported code — these are protocol facts, cross-checked:
//
//   - The RC4 key is the operator's key (typically 40 bits) followed by the
//     4-byte Message Indicator (MI), MSB first. The first 256 keystream
//     bytes are discarded (DSD-FME `dropL = 256`, the same convention as P25
//     ADP), and the keystream is then applied 7 bytes per AMBE+2 frame — the
//     49-bit FEC-decoded vocoder payload packed MSB-first into 7 bytes —
//     contiguously across the 18 frames of a voice superframe (126 bytes).
//   - Each superframe restarts the keystream with a NEW MI: the MI advances
//     once per superframe through a 32-bit LFSR with characteristic
//     polynomial x^32 + x^4 + x^2 + 1 (AdvanceMI). The PI header's MI is the
//     first superframe's.
//   - Every superframe also carries its own MI in the voice bursts, for late
//     entry (Motorola patent EP2347540B1, "embedding encryption parameters in
//     voice super frame"): each of the 18 frames donates one 4-bit nibble in
//     its unprotected C3 sub-vector, and the 72 bits reassemble into three
//     Golay(24,12) codewords whose 36 data bits are the 32-bit MI plus a
//     4-bit CRC (ExtractEmbeddedIV). A receiver that missed the PI header can
//     therefore still decrypt from the next full superframe.
//   - A frame that carries the AMBE+2 silence vector is transmitted in clear
//     and is passed through untouched, but still consumes its 7 keystream
//     bytes (DSD-FME's silence guard).
//
// Verify-before-close (#764/#771): the primitives are pinned by literal
// vectors from independent implementations (ep_test.go), but the whole chain
// is on-air-verified only once a known-key capture decodes to intelligible
// audio through the daemon — docs/dmr-encryption.md.

const (
	// EPKeystreamDrop is the number of leading RC4 keystream bytes discarded
	// before the first voice frame.
	EPKeystreamDrop = 256
	// EPFrameBytes is the keystream span one 49-bit AMBE+2 frame consumes.
	EPFrameBytes = 7
	// EPSuperframeBytes is the keystream span of one 18-frame superframe.
	EPSuperframeBytes = EPFrameBytes * FramesPerSuperframe
	// EPMIBytes is the Message Indicator length appended to the key.
	EPMIBytes = 4
)

// ambeSilence is the AMBE+2 silence vector (DSD-FME's 0xF801A99F8CE080,
// 56 bits MSB-first of which the first 49 are the frame), one bit per byte.
var ambeSilence = func() [ambeInfoBits]byte {
	const v uint64 = 0xF801A99F8CE080
	var out [ambeInfoBits]byte
	for i := range out {
		out[i] = byte((v >> uint(55-i)) & 1)
	}
	return out
}()

// AdvanceMI steps a 32-bit Message Indicator to the next superframe's value:
// 32 shifts of the LFSR with characteristic polynomial x^32 + x^4 + x^2 + 1
// (DSD-FME dmr_pi.c LFSR()).
func AdvanceMI(mi uint32) uint32 {
	l := uint64(mi)
	for i := 0; i < 32; i++ {
		bit := ((l >> 31) ^ (l >> 3) ^ (l >> 1)) & 1
		l = (l << 1) | bit
	}
	return uint32(l)
}

// EPKeystream returns n bytes of the Enhanced Privacy keystream for key and
// mi — RC4 keyed with key‖MI, positioned past the EPKeystreamDrop warm-up.
func EPKeystream(key []byte, mi uint32, n int) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("dmr/voice: enhanced privacy needs a key")
	}
	k := make([]byte, 0, len(key)+EPMIBytes)
	k = append(k, key...)
	k = append(k, byte(mi>>24), byte(mi>>16), byte(mi>>8), byte(mi))
	c, err := rc4.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("dmr/voice: enhanced privacy: %w", err)
	}
	c.KeyStream(EPKeystreamDrop)
	return c.KeyStream(n), nil
}

// IsAMBESilence reports whether a 49-bit payload (one bit per byte) is the
// AMBE+2 silence vector.
func IsAMBESilence(info []byte) bool {
	if len(info) != ambeInfoBits {
		return false
	}
	for i, b := range info {
		if b&1 != ambeSilence[i] {
			return false
		}
	}
	return true
}

// DescrambleSuperframe XORs one superframe's keystream onto frames IN PLACE.
// frames holds the superframe's FEC-decoded 49-bit payloads in transmission
// order (one bit per byte); a nil entry is a frame that failed FEC and is
// skipped, still consuming its keystream span so the later frames stay
// aligned. Silence-vector frames pass through in clear. It returns how many
// frames were descrambled and how many were silence.
func DescrambleSuperframe(key []byte, mi uint32, frames [][]byte) (descrambled, silence int, err error) {
	if len(frames) > FramesPerSuperframe {
		return 0, 0, fmt.Errorf("dmr/voice: %d frames exceed a superframe", len(frames))
	}
	ks, err := EPKeystream(key, mi, EPSuperframeBytes)
	if err != nil {
		return 0, 0, err
	}
	for i, f := range frames {
		if f == nil {
			continue
		}
		if len(f) != ambeInfoBits {
			return descrambled, silence, fmt.Errorf("dmr/voice: frame %d has %d bits, want %d", i, len(f), ambeInfoBits)
		}
		if IsAMBESilence(f) {
			silence++
			continue
		}
		span := ks[i*EPFrameBytes : (i+1)*EPFrameBytes]
		for b := 0; b < ambeInfoBits; b++ {
			f[b] ^= (span[b>>3] >> uint(7-(b&7))) & 1
		}
		descrambled++
	}
	return descrambled, silence, nil
}

// ivNibble returns the 4-bit embedded-IV fragment a 72-bit on-air AMBE+2
// frame carries: on-air bits 71, 67, 63, 59 (SDRTrunk FRAME_x_IV_FRAGMENT,
// MSB first), which the DMR deinterleave places at C3[0..3] — the same four
// bits DSD-FME reads after deinterleaving (ep_test.go pins the equivalence
// through this package's own rW/rX/rY/rZ tables).
func ivNibble(frame []byte) uint8 {
	return (frame[71]&1)<<3 | (frame[67]&1)<<2 | (frame[63]&1)<<1 | frame[59]&1
}

// golayDecode2412 decodes an extended Golay(24,12) codeword laid out MSB
// first as data(12) ‖ parity(11) ‖ overall even parity(1) — SDRTrunk's
// Golay24 layout, whose CHECKSUMS table is this package's golayGen. It
// returns the data and whether the word was within the code's 3-error
// correction radius (a 4-error word, caught by the overall parity, is not).
func golayDecode2412(cw uint32) (uint16, bool) {
	cw &= 0xFFFFFF
	cw23 := cw >> 1
	data, _ := golayDecode2312(cw23)
	corrected := golayEncode2312(data)
	errs := bits.OnesCount32(cw23 ^ corrected)
	if bits.OnesCount32(corrected<<1|cw&1)%2 != 0 {
		errs++ // the overall-parity bit itself disagrees
	}
	return data, errs <= 3
}

// golayEncode2412 is the inverse of golayDecode2412 (test fixtures).
func golayEncode2412(data uint16) uint32 {
	cw := golayEncode2312(data) << 1
	if bits.OnesCount32(cw)%2 != 0 {
		cw |= 1
	}
	return cw
}

// crc4 is the 4-bit check over the 32-bit embedded MI: polynomial x^4+x+1,
// zero initial state, output inverted (DSD-FME dmr_le.c crc4; SDRTrunk's
// 0xF-initial-fill formulation is the same function — both are pinned by
// the same literal vectors in ep_test.go).
func crc4(mi uint32) uint8 {
	var crc uint8
	for i := 31; i >= 0; i-- {
		fb := (crc>>3)&1 ^ uint8((mi>>uint(i))&1)
		crc = (crc << 1) & 0xF
		if fb != 0 {
			crc ^= 0x3
		}
	}
	return crc ^ 0xF
}

// ExtractEmbeddedIV reassembles the Message Indicator a voice superframe
// embeds across its 18 on-air AMBE+2 frames (bursts A–F, three nibbles per
// burst). ok is false when any frame is missing, any of the three
// Golay(24,12) codewords is beyond correction, or the CRC-4 fails; corrected
// is the number of nibble bits the Golay decode repaired.
//
// The gate is deliberately weak on its own: a Golay(24,12) sphere of radius
// 3 covers ~57% of all 24-bit words, so a clear superframe's vocoder bits
// pass all three codewords plus the CRC-4 about 1% of the time. Callers
// therefore only act on it for a call that is otherwise known to be
// encrypted (PI header, LC service options), and EPTracker trusts a
// verified-but-corrected IV over its own LFSR prediction only when the two
// agree.
func ExtractEmbeddedIV(frames [FramesPerSuperframe][]byte) (iv uint32, corrected int, ok bool) {
	var frag [BurstsPerSuperframe][FramesPerBurst]uint8
	for b := 0; b < BurstsPerSuperframe; b++ {
		for k := 0; k < FramesPerBurst; k++ {
			f := frames[b*FramesPerBurst+k]
			if len(f) != AMBEFrameBits {
				return 0, 0, false
			}
			frag[b][k] = ivNibble(f)
		}
	}
	var data36 uint64
	for j := 0; j < 3; j++ {
		// Codeword j: data = A[j] B[j] C[j], parity = D[j] E[j] F[j].
		d := uint32(frag[0][j])<<8 | uint32(frag[1][j])<<4 | uint32(frag[2][j])
		p := uint32(frag[3][j])<<8 | uint32(frag[4][j])<<4 | uint32(frag[5][j])
		cw := d<<12 | p
		data, gok := golayDecode2412(cw)
		if !gok {
			return 0, 0, false
		}
		corrected += bits.OnesCount32(cw ^ golayEncode2412(data))
		data36 = data36<<12 | uint64(data)
	}
	iv = uint32(data36 >> 4)
	if crc4(iv) != uint8(data36&0xF) {
		return 0, 0, false
	}
	return iv, corrected, true
}

// EmbedIV is the transmitter side of ExtractEmbeddedIV: it writes the
// nibbles for iv (with its Golay parity and CRC-4) into the 18 on-air frames
// in place, overwriting each frame's C3[0..3] bits exactly as a radio does
// AFTER encryption. Test fixtures and the replay harness use it.
func EmbedIV(frames [FramesPerSuperframe][]byte, iv uint32) {
	data36 := uint64(iv)<<4 | uint64(crc4(iv))
	for j := 0; j < 3; j++ {
		data := uint16(data36>>uint(24-12*j)) & 0xFFF
		cw := golayEncode2412(data)
		nib := [BurstsPerSuperframe]uint8{
			uint8(cw >> 20 & 0xF), uint8(cw >> 16 & 0xF), uint8(cw >> 12 & 0xF),
			uint8(cw >> 8 & 0xF), uint8(cw >> 4 & 0xF), uint8(cw & 0xF),
		}
		for b := 0; b < BurstsPerSuperframe; b++ {
			f := frames[b*FramesPerBurst+j]
			f[71] = nib[b] >> 3 & 1
			f[67] = nib[b] >> 2 & 1
			f[63] = nib[b] >> 1 & 1
			f[59] = nib[b] & 1
		}
	}
}

// EPTracker follows the Message Indicator across a call's superframes. The
// PI header seeds it; each superframe's embedded IV, when it verifies,
// confirms (or corrects) the prediction; otherwise the LFSR prediction
// carries the chain across a superframe whose embedded IV was damaged.
type EPTracker struct {
	known bool
	next  uint32
	// Mismatches counts superframes whose verified embedded IV disagreed
	// with the LFSR prediction — the instrument for a wrong advance rule or
	// a missed superframe.
	Mismatches int
	// Predicted counts superframes decoded on the prediction alone.
	Predicted int
}

// SetHeaderMI seeds the tracker from a PI header: mi is the NEXT
// superframe's Message Indicator.
func (t *EPTracker) SetHeaderMI(mi uint32) {
	t.known = true
	t.next = mi
}

// Next returns the MI for the superframe now being decoded, given that
// superframe's embedded IV (embeddedOK false when it did not verify, and
// corrected the Golay repairs it needed), and advances the prediction. ok is
// false when no MI is known at all.
//
// A verified embedded IV that disagrees with the LFSR prediction is adopted
// only when it decoded CLEAN (no Golay corrections): a clean triple codeword
// plus CRC-4 is ~2^-40 by chance, whereas a corrected one has the ~1% false
// verification rate of the radius-3 sphere. Otherwise the prediction holds
// and the disagreement is counted in Mismatches.
func (t *EPTracker) Next(embedded uint32, embeddedOK bool, corrected int) (mi uint32, ok bool) {
	switch {
	case embeddedOK && (!t.known || embedded == t.next || corrected == 0):
		if t.known && embedded != t.next {
			t.Mismatches++
		}
		mi = embedded
	case embeddedOK: // corrected AND disagrees: hold the chain
		t.Mismatches++
		mi = t.next
		t.Predicted++
	case t.known:
		mi = t.next
		t.Predicted++
	default:
		return 0, false
	}
	t.known = true
	t.next = AdvanceMI(mi)
	return mi, true
}

// Known reports whether the tracker holds an MI for the next superframe.
func (t *EPTracker) Known() bool { return t.known }

// Reset forgets the chain (a new transmission on the same channel).
func (t *EPTracker) Reset() {
	t.known = false
	t.next = 0
}
