package imbe

import "fmt"

// IMBE 4400 pseudo-random scrambler (TIA-102.BABA §7.4 / §7.4.1).
// Whitens the channel bits of u_1..u_6 by XORing a 114-bit PRBS
// derived from the 12 information bits of u_0. u_0 itself and u_7
// are transmitted unscrambled — u_0 because it carries the seed,
// u_7 because it's the unprotected least-sensitive bits.
//
// PRBS algorithm: a 16-bit linear-congruential generator with
//
//	multiplier = 173
//	increment  = 13849
//	modulus    = 65536
//	seed       = u_0_info × 16
//	bit[i]     = pr[i] >> 15  (high bit of the 16-bit state)
//
// matching the C reference implementation in mbelib's
// mbe_demodulateImbe7200x4400Data (ISC-licensed, attribution
// preserved at the bottom of this file). The generator is invoked
// 115 times so pr[0] holds the seed × 16 and pr[1..114] supply 114
// scrambling bits — one per scrambled channel bit.

// PRBSLength is how many PRBS bits the generator yields per IMBE
// frame: 23 bits × 3 (u_1..u_3) + 15 bits × 3 (u_4..u_6) = 114.
// Plus pr[0] = seed, total 115 LCG iterations.
const PRBSLength = 114

// PRBSSeedFromU0 reads the 12 information bits of u_0 from a
// 144-bit channel buffer and returns the 16-bit PRBS seed
// (u_0_info × 16). The info bits are the first 12 bits of the u_0
// region — the systematic data bits that GolayEncode23_12 places at
// the high end of the codeword.
func PRBSSeedFromU0(channel []byte) uint16 {
	var info uint16
	for i := 0; i < u0InfoBits; i++ {
		info = (info << 1) | uint16(channel[u0Offset+i]&1)
	}
	return info << 4
}

// PRBS expands the 16-bit seed into PRBSLength scrambling bits.
// The returned slice's value[i] is one of {0, 1}; XORing each with
// the matching channel bit is the scramble (and descramble — XOR
// is self-inverse).
func PRBS(seed uint16) [PRBSLength]byte {
	const mul, inc, mod uint32 = 173, 13849, 65536
	state := uint32(seed)
	var bits [PRBSLength]byte
	for i := 0; i < PRBSLength; i++ {
		state = (mul*state + inc) % mod
		// high bit of the 16-bit state — equivalent to mbelib's
		// `pr[i] / 32768`.
		bits[i] = byte(state >> 15)
	}
	return bits
}

// Scramble applies the IMBE PRBS to the 144-bit channel buffer in
// place: u_0 and u_7 are left untouched, u_1..u_3 (23 bits each)
// and u_4..u_6 (15 bits each) are XORed with the PRBS. Returns
// the channel buffer for chaining; the input is mutated. Returns
// an error only if the buffer is the wrong length.
//
// Scramble and Descramble are the same operation (XOR is
// self-inverse); the two names exist for documentation. A clean
// frame round-trips: Scramble(Scramble(x)) == x.
func Scramble(channel []byte) ([]byte, error) {
	if len(channel) != ChannelBits {
		return nil, fmt.Errorf("%w: got %d channel bits", ErrChannelLength, len(channel))
	}
	prbs := PRBS(PRBSSeedFromU0(channel))
	k := 0
	for _, region := range [][2]int{
		{u1Offset, u1Bits},
		{u2Offset, u2Bits},
		{u3Offset, u3Bits},
		{u4Offset, u4Bits},
		{u5Offset, u5Bits},
		{u6Offset, u6Bits},
	} {
		off, n := region[0], region[1]
		for i := 0; i < n; i++ {
			channel[off+i] ^= prbs[k]
			k++
		}
	}
	return channel, nil
}

// Descramble is an alias for Scramble. Both names are exported so
// call sites read naturally on encode (Scramble) and decode
// (Descramble) paths.
func Descramble(channel []byte) ([]byte, error) {
	return Scramble(channel)
}

// PRBS reference: szechyjs/mbelib mbe_demodulateImbe7200x4400Data
// (imbe7200x4400.c). ISC license, copyright 2010 mbelib Author. The
// LCG constants above (multiplier 173, increment 13849, modulus
// 65536, output bit 15) come from that implementation, which in
// turn implements TIA-102.BABA §7.4.
