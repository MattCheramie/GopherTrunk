package nxdn

// NXDN voice scrambler (the Type-1 "scrambler" cipher — the privacy mode
// Kenwood / Icom radios expose as a 15-bit key, cipher type 1 in the SACCH
// VCALL's cipher field). It XORs a keystream over the 49-bit AMBE+2 payload
// of every voice slot. The keystream is a 15-bit linear-feedback shift
// register seeded by the key, a value 1..32767 (0 is "no scrambling"):
// 2^15 keys, so the cipher is trivially brute-forceable; the cryptolab
// `nxdn brute` tool does exactly that with this primitive.
//
// The register and its schedule are CAPTURE-PINNED (issue #1261): two
// known-key captures from the reporter's Kenwood radios (keys 32767 and
// 12345, internal/radio/nxdn/testdata/scrambler_key*_frames.json) decode
// to speech only with this register — out = state bit 0, feedback = bit 0
// XOR bit 1 shifted in at bit 14 (the construction DSD-FME's NXDN decoder
// uses; read for the protocol fact, nothing ported) — and only with this
// schedule: reseeded from the key at the first frame of each SACCH
// superframe (SACCH structure field 3), then run continuously, 4 × 49 bits
// per LICH-valid air frame whether or not that frame carried voice, slot v
// taking bits [49v, 49v+49). A brute force over all 32768 keys ranks the
// radios' programmed key first on both captures; the earlier model here
// (a left-shift x^15+x^14+1 register, restarted per frame) decoded nothing.
// See VoiceCipherTracker for the schedule.

const (
	// ScramblerKeyBits is the width of the NXDN scrambler key.
	ScramblerKeyBits = 15
	// ScramblerKeyMax is the largest valid key; the key space is
	// [0, ScramblerKeyMax], i.e. 32768 keys.
	ScramblerKeyMax = (1 << ScramblerKeyBits) - 1 // 32767

	// VoiceSlotBits is the AMBE+2 payload of one voice slot the keystream
	// covers.
	VoiceSlotBits = 49
	// ScramblerBitsPerFrame is the keystream one air frame consumes:
	// four voice slots, whether or not the frame carried voice.
	ScramblerBitsPerFrame = 4 * VoiceSlotBits
)

// Scrambler produces the NXDN scrambling keystream one bit at a time.
// Construct it with NewScrambler; Next returns successive keystream bits.
// The same helper serves both the scramble (TX) and descramble (RX) sides
// because XOR is symmetric — see Scramble / Descramble.
type Scrambler struct {
	state uint16 // 15-bit LFSR state; bit 0 is the next output
}

// NewScrambler returns a Scrambler seeded by the low 15 bits of key.
func NewScrambler(key uint16) *Scrambler {
	s := &Scrambler{}
	s.Reset(key)
	return s
}

// Reset reseeds the register from key.
func (s *Scrambler) Reset(key uint16) { s.state = key & ScramblerKeyMax }

// Next advances the LFSR by one step and returns the next keystream bit
// (0/1): the output is state bit 0, and the register shifts right with
// bit 0 XOR bit 1 entering at bit 14 (x^15 + x + 1, maximal length).
func (s *Scrambler) Next() byte {
	out := byte(s.state & 1)
	fb := ((s.state >> 1) ^ s.state) & 1
	s.state = ((s.state >> 1) | (fb << 14)) & ScramblerKeyMax
	return out
}

// Generate returns the next n keystream bits (each 0/1).
func (s *Scrambler) Generate(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = s.Next()
	}
	return out
}

// Skip advances the register n bits without producing output.
func (s *Scrambler) Skip(n int) {
	for i := 0; i < n; i++ {
		s.Next()
	}
}

// Scramble XORs the keystream seeded by key over bits (each entry 0/1,
// MSB-first), returning a new slice of the same length. The keystream
// restarts from the key seed for each call: this is the one-block primitive
// (the cryptolab tool's unit); the air schedule is VoiceCipherTracker's.
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
