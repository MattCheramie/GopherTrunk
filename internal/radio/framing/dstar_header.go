package framing

// D-STAR DV-mode PCH (Preamble + Header) FEC chain per the JARL
// DV-mode specification.
//
//   41-byte information field (328 bits)
//     → append 4-bit conv flush tail (332 bits)
//     → K=5 R=1/2 convolutional encode (664 channel bits)
//     → puncture 4 trailing channel bits (660 channel bits)
//     → 15-bit LFSR PN15 scrambler
//     → 24 × 28 block interleaver (write column-major, read row-major;
//       12 cells unused per burst)
//     = 660 on-wire bits emitted by the GMSK modulator
//
// The convolutional code reuses ViterbiK5 (G1=0x19, G2=0x17) — same
// polynomials MMDVMHost / DSDcc / OpenDV all use for the D-STAR
// header. The scrambler and interleaver are self-consistent
// encode/decode pairs; matching MMDVMHost / DSDcc's exact
// permutation tables for live-air decode is a follow-up calibration
// step against a captured transmission.

const (
	// DStarHeaderInfoBits is the size of the structured information
	// field that ParseHeader consumes (41 bytes * 8).
	DStarHeaderInfoBits = 41 * 8
	// DStarHeaderTailBits is the number of zero flush bits appended
	// to the information field before convolutional encoding (K-1
	// for K=5).
	DStarHeaderTailBits = 4
	// DStarHeaderInputBits is the conv encoder's input bit count
	// (info + tail).
	DStarHeaderInputBits = DStarHeaderInfoBits + DStarHeaderTailBits
	// DStarHeaderRawChannelBits is the conv encoder's raw output
	// bit count (R=1/2, so 2 × input).
	DStarHeaderRawChannelBits = DStarHeaderInputBits * 2
	// DStarHeaderChannelBits is the on-wire bit count after the
	// post-conv-encoder puncture (4 trailing channel bits dropped
	// to land at the JARL-spec 660-bit on-wire header window).
	DStarHeaderChannelBits = 660
	// DStarHeaderInterleaveRows / DStarHeaderInterleaveCols define
	// the 22 × 30 block interleaver grid. 22 * 30 = 660 cells — an
	// exact fit, no unused grid positions. The encoder writes
	// column-major (in[0] → grid[0][0], in[1] → grid[1][0], …,
	// in[21] → grid[21][0], in[22] → grid[0][1]) and the decoder
	// reads column-major from a row-major-written grid, so the
	// round-trip is the identity for any 660-bit channel stream.
	DStarHeaderInterleaveRows = 22
	DStarHeaderInterleaveCols = 30
)

// EncodeDStarHeaderFEC takes 41 header bytes (the AssembleHeader
// output including the CRC trailer) and returns the 660-bit on-wire
// stream the modulator emits. The chain is conv-encode → puncture →
// scramble → interleave.
func EncodeDStarHeaderFEC(info []byte) []byte {
	if len(info) != 41 {
		return nil
	}
	// Pack info bytes MSB-first into 328 bits, then append 4 zero
	// tail bits.
	in := make([]byte, DStarHeaderInputBits)
	for i, b := range info {
		for j := 0; j < 8; j++ {
			in[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	// Conv encode R=1/2 K=5, polynomials G1=0x19 (1+x^3+x^4), G2=0x17
	// (1+x+x^2+x^4). State convention matches ViterbiK5: bits
	// (d1,d2,d3,d4) from most-recent to oldest. Encoder shifts the
	// new input bit into d1; outputs are emitted as (g1, g2) pairs.
	channel := make([]byte, DStarHeaderRawChannelBits)
	var d1, d2, d3, d4 byte
	for s, input := range in {
		g1 := (input ^ d3 ^ d4) & 1
		g2 := (input ^ d1 ^ d2 ^ d4) & 1
		channel[2*s] = g1
		channel[2*s+1] = g2
		d4 = d3
		d3 = d2
		d2 = d1
		d1 = input & 1
	}
	// Puncture: drop the last 4 channel bits to land at 660.
	channel = channel[:DStarHeaderChannelBits]
	// Scramble: PN15 LFSR.
	scrambleDStarHeader(channel)
	// Interleave: 24 rows × 28 cols column-major write, row-major
	// read. Output is 660 bits.
	return interleaveDStarHeader(channel)
}

// DecodeDStarHeaderFEC takes the 660-bit on-wire stream and runs
// deinterleave → descramble → depuncture → Viterbi K=5 to recover
// the 41-byte information field. Returns (info, true) on success.
// Returns (nil, false) when the recovered tail bits don't terminate
// in the zero state (indicating an unrecoverable bit-error burst).
func DecodeDStarHeaderFEC(channel []byte) ([]byte, bool) {
	if len(channel) != DStarHeaderChannelBits {
		return nil, false
	}
	// Reverse the chain: deinterleave first.
	deinter := deinterleaveDStarHeader(channel)
	// Descramble (same LFSR — XOR is self-inverse).
	scrambleDStarHeader(deinter)
	// Depuncture: append 4 DepunctureMark bytes to restore the conv
	// encoder's full output length.
	depunc := make([]byte, DStarHeaderRawChannelBits)
	copy(depunc, deinter)
	for i := DStarHeaderChannelBits; i < DStarHeaderRawChannelBits; i++ {
		depunc[i] = DepunctureMark
	}
	// Viterbi decode K=5 R=1/2. The decoder returns DStarHeaderInputBits
	// (= 332) bits including the 4 zero flush bits at the end.
	decoded, _ := ViterbiK5(depunc, DStarHeaderInputBits)
	if len(decoded) != DStarHeaderInputBits {
		return nil, false
	}
	// Verify the trailing flush bits are zero (Viterbi survivor
	// ended in state 0 → tail bits should be zero). If not, the
	// chain landed on a non-terminating survivor and the recovered
	// payload is unreliable.
	for i := DStarHeaderInfoBits; i < DStarHeaderInputBits; i++ {
		if decoded[i] != 0 {
			return nil, false
		}
	}
	// Pack the 328 info bits MSB-first into 41 bytes.
	out := make([]byte, 41)
	for i := 0; i < 41; i++ {
		var v byte
		for j := 0; j < 8; j++ {
			v = (v << 1) | (decoded[i*8+j] & 1)
		}
		out[i] = v
	}
	return out, true
}

// scrambleDStarHeader XORs the supplied 660 channel bits with a
// PN15 keystream. The function is self-inverse: calling it twice
// returns the original sequence, so encode and decode share the
// same implementation.
//
// PN15 polynomial: x^15 + x + 1, taps at register positions 14 and 0.
// Initial register value: 0x0001 (canonical PN15 init).
func scrambleDStarHeader(bits []byte) {
	reg := uint16(0x0001)
	for i := range bits {
		// Output keystream bit is the LSB of the register.
		ks := byte(reg & 1)
		bits[i] ^= ks
		// Galois LFSR step: feedback = (bit 0 XOR bit 14).
		fb := (reg & 1) ^ ((reg >> 14) & 1)
		reg = (reg >> 1) | (fb << 14)
		reg &= 0x7FFF
	}
}

// interleaveDStarHeader writes the supplied 660 channel bits
// column-major into the 22×30 grid then reads them out row-major.
// Exact fit, no unused cells.
func interleaveDStarHeader(in []byte) []byte {
	if len(in) != DStarHeaderChannelBits {
		return nil
	}
	const rows = DStarHeaderInterleaveRows
	const cols = DStarHeaderInterleaveCols
	grid := make([]byte, rows*cols)
	// Column-major write: in[0] → grid[0][0], in[1] → grid[1][0],
	// …, in[21] → grid[21][0], in[22] → grid[0][1], etc.
	for i, b := range in {
		row := i % rows
		col := i / rows
		grid[row*cols+col] = b
	}
	// Row-major read: emit grid[0][0..cols-1], grid[1][0..cols-1],
	// …, grid[rows-1][0..cols-1].
	out := make([]byte, DStarHeaderChannelBits)
	w := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			out[w] = grid[r*cols+c]
			w++
		}
	}
	return out
}

// deinterleaveDStarHeader is the inverse of interleaveDStarHeader:
// writes row-major, reads column-major.
func deinterleaveDStarHeader(in []byte) []byte {
	if len(in) != DStarHeaderChannelBits {
		return nil
	}
	const rows = DStarHeaderInterleaveRows
	const cols = DStarHeaderInterleaveCols
	grid := make([]byte, rows*cols)
	// Row-major write: in[0] → grid[0][0], in[1] → grid[0][1], ….
	w := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			grid[r*cols+c] = in[w]
			w++
		}
	}
	// Column-major read: emit grid[0..rows-1][0], grid[0..rows-1][1],
	// ….
	out := make([]byte, DStarHeaderChannelBits)
	w = 0
	for c := 0; c < cols; c++ {
		for r := 0; r < rows; r++ {
			out[w] = grid[r*cols+c]
			w++
		}
	}
	return out
}
