package imbe

import "fmt"

// IMBE 4400 channel-bit interleaver (TIA-102.BABA §7.5).
//
// After the per-vector FEC encode (channel.go) and the §7.4 PRBS
// scramble (scrambler.go), the 144 channel bits are interleaved
// across the eight u_n vectors so a localised on-air burst error
// spreads over several codewords instead of wiping out a single
// vector. The receiver must undo this permutation BEFORE the
// descrambler reads the u_0 seed and before the per-vector FEC
// runs — hence DecodeChannelToFrame's order is
// Deinterleave → Descramble → DecodeChannel.
//
// # Permutation source
//
// imbeDeinterleave below was derived from DSD's published P25
// Phase 1 IMBE interleave schedule (the iW/iX/iY/iZ tables in
// szechyjs/dsd include/p25p1_const.h, ISC-licensed), which the
// project already cites as a structural reference (see
// internal/voice/imbe/doc.go and docs/vocoders.md). DSD reads the
// 72 on-air voice dibits and scatters each dibit's two bits into an
// imbe_fr[row][col] vector array via:
//
//	imbe_fr[iW[j]][iX[j]] = dibit_hi   (on-air bit 2j)
//	imbe_fr[iY[j]][iZ[j]] = dibit_lo   (on-air bit 2j+1)
//
// Mapping DSD's (row, col) onto this package's vector-order layout
// (channel.go: u_0..u_3 at 23-bit strides 0/23/46/69, u_4..u_6 at
// 15-bit strides 92/107/122, u_7 at 137) yields the table below,
// where imbeDeinterleave[v] is the on-air bit index that supplies
// vector-order bit v. It is a verified bijection of 0..143 (see the
// init guard and TestInterleavePermutationIsBijection); regenerating
// it from the DSD tables reproduces these exact values.
//
// Real-air validation: this table reproduces the same permutation
// every open-source P25 decoder uses, and the full channel decode is
// now pinned against mbelib/DSD-faithful reference vectors in
// p25fec_refvec_test.go — a real P25 voice subframe decodes to the
// expected information frame bit-for-bit (issue #489 resolved).
var imbeDeinterleave = [ChannelBits]int{
	132, 127, 120, 115, 108, 103, 96, 91, 84, 79, 72, 67,
	60, 55, 48, 43, 36, 31, 24, 19, 12, 7, 0, 126,
	121, 114, 109, 102, 97, 90, 85, 78, 73, 66, 61, 54,
	49, 42, 37, 30, 25, 18, 13, 6, 1, 139, 122, 117,
	110, 105, 98, 93, 86, 81, 74, 69, 62, 57, 50, 45,
	38, 33, 26, 21, 14, 9, 2, 138, 133, 116, 111, 104,
	99, 92, 87, 80, 75, 68, 63, 56, 51, 44, 39, 32,
	27, 20, 15, 8, 3, 141, 134, 129, 64, 59, 52, 47,
	40, 35, 28, 23, 16, 11, 4, 140, 135, 128, 123, 10,
	5, 143, 136, 131, 124, 119, 112, 107, 100, 95, 88, 83,
	76, 71, 101, 94, 89, 82, 77, 70, 65, 58, 53, 46,
	41, 34, 29, 22, 17, 142, 137, 130, 125, 118, 113, 106,
}

// init panics if imbeDeinterleave is not a permutation of 0..143.
// A non-bijective table would silently corrupt every voice frame, so
// fail loudly at package load rather than ship a subtly-wrong codec.
func init() {
	var seen [ChannelBits]bool
	for v, t := range imbeDeinterleave {
		if t < 0 || t >= ChannelBits {
			panic(fmt.Sprintf("imbe: imbeDeinterleave[%d] = %d out of range", v, t))
		}
		if seen[t] {
			panic(fmt.Sprintf("imbe: imbeDeinterleave maps two vector bits to on-air bit %d", t))
		}
		seen[t] = true
	}
}

// Deinterleave undoes the §7.5 channel-bit interleave: it takes the
// 144 on-air channel bits (transmission order, one bit per byte,
// 0/1) and returns the 144 bits in vector order — the u_0..u_7
// layout DecodeChannel and the descrambler expect. Returns a fresh
// buffer; the input is not modified.
func Deinterleave(onAir []byte) ([]byte, error) {
	if len(onAir) != ChannelBits {
		return nil, fmt.Errorf("%w: got %d channel bits", ErrChannelLength, len(onAir))
	}
	vec := make([]byte, ChannelBits)
	for v, t := range imbeDeinterleave {
		vec[v] = onAir[t] & 1
	}
	return vec, nil
}

// Interleave is the inverse of Deinterleave: it takes 144 channel
// bits in vector order (post-FEC-encode, post-scramble) and returns
// the on-air transmission-order bits. Used by the encode/fixture
// path so synthesized LDUs match the real on-air permutation.
// Returns a fresh buffer; the input is not modified.
func Interleave(vec []byte) ([]byte, error) {
	if len(vec) != ChannelBits {
		return nil, fmt.Errorf("%w: got %d channel bits", ErrChannelLength, len(vec))
	}
	onAir := make([]byte, ChannelBits)
	for v, t := range imbeDeinterleave {
		onAir[t] = vec[v] & 1
	}
	return onAir, nil
}

// EncodeFrameToChannel is the on-air encode counterpart of
// DecodeChannelToFrame: 88 information bits → 144 on-air channel
// bits via per-vector FEC encode (EncodeChannel) → §7.4 scramble
// (Scramble) → §7.5 interleave (Interleave). The result is the
// transmission-order burst a real receiver sees, so test fixtures
// built with it exercise the full deinterleave→descramble→FEC
// inverse rather than a self-consistent shortcut.
func EncodeFrameToChannel(info []byte) ([]byte, error) {
	channel, err := EncodeChannel(info)
	if err != nil {
		return nil, err
	}
	scrambled, err := Scramble(channel)
	if err != nil {
		return nil, err
	}
	return Interleave(scrambled)
}
