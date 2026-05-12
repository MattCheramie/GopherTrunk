package framing

// PN44 scrambling sequence for P25 Phase 2 Two-Slot TDMA per
// TIA-102.BBAC-1 §7.2.5 — the bit-level XOR scrambler that wraps
// every voice and signaling burst between demodulation and FEC.
//
// The scrambler is a 44-bit linear-feedback shift register with the
// generator polynomial
//
//	G(x) = x^44 + x^40 + x^35 + x^29 + x^24 + x^10 + 1
//
// shifting one bit per clock and outputting bit 43. The seed is
// computed once per call from the (WACN_ID, System_ID, Color Code)
// triple — the same values the Network Status Broadcast MAC message
// publishes — and the sequence restarts at the beginning of every
// 360 ms superframe (4320 bits).
//
//	seed_external = (WACN_ID · 2^24) + (System_ID · 2^12) + Color_Code
//	  where WACN_ID is 20 bits, System_ID is 12 bits, Color_Code is 12 bits.
//	If seed_external == 0, set seed_external = 2^44 - 1.
//
// The inbound seed is the outbound seed advanced by 243 LFSR cycles
// (the SH(243) matrix in Figure 7-4 of the spec, computed directly
// here as 243 clock advances on the LFSR rather than as a matrix
// multiply — the two are equivalent and the iterative form is
// trivially testable).
//
// This module ships the PN44 primitive — encoder + descrambler +
// seed calculation — plus its associated round-trip tests. Wiring
// the descrambler into the Phase 2 ControlChannel Process adapter
// is gated on superframe-position tracking; see Figure 7-5 in the
// spec for the offset table per slot, and the matching wiring TODO
// in internal/radio/p25/phase2/process.go.

// pn44Mask masks a uint64 to the 44 LSBs.
const pn44Mask uint64 = (1 << 44) - 1

// PN44Scrambler walks the PN44 sequence one bit at a time.
type PN44Scrambler struct {
	state uint64
}

// NewPN44Scrambler returns a scrambler initialised with the supplied
// 44-bit seed. Callers typically derive the seed via
// PN44SeedFromIdentity (outbound) and PN44SeedInbound (inbound).
//
// A seed of 0 maps to (2^44 - 1) per spec; this also matches the
// behaviour when the (WACN_ID, System_ID, Color_Code) triple is
// all-zero.
func NewPN44Scrambler(seed uint64) *PN44Scrambler {
	s := seed & pn44Mask
	if s == 0 {
		s = pn44Mask
	}
	return &PN44Scrambler{state: s}
}

// Next returns the next bit of the PN44 sequence. The bit is read
// from position 43 of the state (the most-significant bit of the
// 44-bit register) before the register shifts and the new feedback
// bit clocks into position 0.
//
// The feedback bit is the XOR of the state bits at the polynomial's
// non-leading tap positions (0, 10, 24, 29, 35, 40).
func (s *PN44Scrambler) Next() byte {
	out := byte((s.state >> 43) & 1)
	fb := byte((s.state>>40 ^
		s.state>>35 ^
		s.state>>29 ^
		s.state>>24 ^
		s.state>>10 ^
		s.state) & 1)
	s.state = ((s.state << 1) | uint64(fb)) & pn44Mask
	return out
}

// Advance clocks the LFSR n bits without consuming the output. Used
// to position the scrambler at the per-burst offset that Figure 7-5
// of the spec defines.
func (s *PN44Scrambler) Advance(n int) {
	for i := 0; i < n; i++ {
		s.Next()
	}
}

// State returns the current 44-bit LFSR state. Exposed primarily so
// tests can validate against precomputed reference values.
func (s *PN44Scrambler) State() uint64 {
	return s.state
}

// Apply XOR-scrambles in-place across the bit slice. Each bit must
// be 0 or 1; the scrambler clocks once per element.
//
// Because scrambling is bitwise XOR, the same Apply call descrambles
// a previously-scrambled stream when invoked with an LFSR initialised
// to the same seed and offset.
func (s *PN44Scrambler) Apply(bits []byte) {
	for i := range bits {
		bits[i] ^= s.Next()
	}
}

// PN44SeedFromIdentity computes the outbound (downlink) PN44 seed
// from the (WACN_ID, System_ID, Color_Code) triple per
// TIA-102.BBAC-1 §7.2.5 equation (5):
//
//	seed_external = WACN_ID·2^24 + System_ID·2^12 + Color_Code
//
// All three values fit into their spec-defined widths (WACN_ID 20
// bits, System_ID 12 bits, Color_Code 12 bits). Values that exceed
// their widths are masked silently — passing oversize values is a
// caller bug; the mask keeps the seed within the 44-bit field.
//
// A computed seed of 0 maps to (2^44 - 1) per spec.
func PN44SeedFromIdentity(wacnID uint32, systemID uint16, colorCode uint16) uint64 {
	seed := (uint64(wacnID&0x000FFFFF) << 24) |
		(uint64(systemID&0x0FFF) << 12) |
		uint64(colorCode&0x0FFF)
	if seed == 0 {
		return pn44Mask
	}
	return seed
}

// PN44SeedInbound returns the inbound (uplink) seed for the supplied
// outbound seed. Per spec equation (8) the inbound seed equals the
// outbound seed advanced by 243 LFSR cycles via the SH(243) matrix
// in Figure 7-4; computed here by directly clocking the scrambler
// 243 times (equivalent to the matrix form, and trivially testable
// without transcribing the 44×44 SH(243) matrix).
func PN44SeedInbound(outboundSeed uint64) uint64 {
	s := NewPN44Scrambler(outboundSeed)
	for i := 0; i < 243; i++ {
		s.Next()
	}
	return s.state
}

// PN44SlotOffsetsOutbound lists the spec-defined offsets into the
// 4320-bit outbound (downlink) scrambling sequence at which each of
// the 12 outbound slots begins per TIA-102.BBAC-1 §7.2.5 Figure 7-5.
//
// A burst-level descrambler that doesn't have full superframe sync
// can probe each offset in turn — descramble + verify against the
// outer RS(24, 16, 9) code — and accept the offset whose syndrome
// is zero. The blind-probe form is implemented in
// internal/radio/p25/phase2/process.go under ScramblerProbe mode.
//
// Slot offsets are 0 + k × 360 for k = 0..11. The 360-bit spacing
// reflects the 30 ms slot duration at 12 ksym/s × 2 bits/symbol /
// 2 slots — a slot carries 360 channel bits worth of scrambling
// sequence.
var PN44SlotOffsetsOutbound = [12]int{
	0, 360, 720, 1080, 1440, 1800,
	2160, 2520, 2880, 3240, 3600, 3960,
}

// PN44SlotOffsetsInbound is the inbound counterpart of
// PN44SlotOffsetsOutbound per Figure 7-5. The 12 inbound slots
// share the same 360-bit spacing but the VCH0 and VCH1 traffic
// lanes interleave at half-slot boundaries, so a comprehensive
// blind probe walks every 180-bit candidate. For now the table
// mirrors the outbound slot grid; refining to the precise
// inbound-slot grid (per VCH0 / VCH1 offsets) is a follow-up that
// only affects inbound captures and lands together with a
// per-slot identifier.
var PN44SlotOffsetsInbound = [12]int{
	0, 360, 720, 1080, 1440, 1800,
	2160, 2520, 2880, 3240, 3600, 3960,
}
