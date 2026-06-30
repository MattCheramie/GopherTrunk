package nxdn

// NXDN voice/data scrambler ("encryption"). The optional NXDN scrambler — the
// privacy mode Kenwood radios expose and AOR receivers auto-brute-force — XORs
// a pseudo-random keystream over the scrambled information field. The keystream
// comes from a 15-bit linear-feedback shift register seeded by the scrambling
// key, a 15-bit value (0..32767). Key 0 yields an all-zero keystream, i.e. no
// scrambling (clear). Because the key space is only 2^15 = 32768, the scrambler
// is trivially brute-forceable; the offline cryptolab `nxdn` tool does exactly
// that, using this primitive plus the NXDN frame CRCs as an oracle.
//
// IMPLEMENTATION NOTE — synthetic model, not yet hardware-confirmed. The tap
// polynomial and seed mapping below are an internally-consistent working model
// (a maximal-length 15-bit Fibonacci LFSR, x^15 + x^14 + 1): Scramble is its own
// inverse and the keystream is a balanced m-sequence of period 32767. They have
// NOT been confirmed bit-exact against a known-key capture. When such a capture
// is available, adjust the feedback in (*Scrambler).Next and the golden
// expectations in scramble_test.go; nothing else (the brute-force loop, the CRC
// oracles) depends on this choice. The XOR-symmetric structure and 15-bit key
// space are the spec-level facts the brute-forcer relies on.

const (
	// ScramblerKeyBits is the width of the NXDN scrambler key.
	ScramblerKeyBits = 15
	// ScramblerKeyMax is the largest valid key; the key space is
	// [0, ScramblerKeyMax], i.e. 32768 keys.
	ScramblerKeyMax = (1 << ScramblerKeyBits) - 1 // 32767
)

// Scrambler produces the NXDN scrambling keystream one bit at a time. Construct
// it with NewScrambler; Next returns successive keystream bits. The same helper
// serves both the scramble (TX) and descramble (RX) sides because XOR is
// symmetric — see Scramble / Descramble.
type Scrambler struct {
	state uint16 // 15-bit LFSR state; the MSB (bit 14) is the next output
}

// NewScrambler returns a Scrambler seeded by the low 15 bits of key.
func NewScrambler(key uint16) *Scrambler {
	return &Scrambler{state: key & ScramblerKeyMax}
}

// Next advances the LFSR by one step and returns the next keystream bit (0/1).
// The register is a maximal-length 15-bit Fibonacci LFSR with feedback
// x^15 + x^14 + 1: the output is the current MSB and the new LSB is the XOR of
// the top two stages.
func (s *Scrambler) Next() byte {
	out := byte((s.state >> 14) & 1)
	fb := ((s.state >> 14) ^ (s.state >> 13)) & 1
	s.state = ((s.state << 1) | fb) & ScramblerKeyMax
	return out
}

// Generate returns the first n keystream bits (each 0/1).
func (s *Scrambler) Generate(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = s.Next()
	}
	return out
}

// Scramble XORs the keystream seeded by key over bits (each entry 0/1,
// MSB-first), returning a new slice of the same length. The keystream restarts
// from the key seed for each call, so one Scramble covers one information field.
func Scramble(bits []byte, key uint16) []byte {
	s := NewScrambler(key)
	out := make([]byte, len(bits))
	for i, b := range bits {
		out[i] = (b & 1) ^ s.Next()
	}
	return out
}

// Descramble is the inverse of Scramble. XOR with the same keystream is
// symmetric, so the implementation is identical; the alias exists for
// call-site readability on the receive side.
func Descramble(bits []byte, key uint16) []byte { return Scramble(bits, key) }
