package mpt1327

import (
	"errors"
	"fmt"
)

// BitSink consumes the raw stream of bits an MPT 1327 receiver
// decodes from IQ. baseIdx is the absolute bit index of bits[0]
// across the stream lifetime — monotonically non-decreasing across
// calls, and reset to 0 by Receiver.Reset so a retune produces a
// fresh baseline. MPT 1327 is 2-level (1200 baud FFSK on top of
// NBFM audio); the 4-level trunked protocols use a DibitSink
// instead. Wire this into a future ControlChannel.Process adapter
// (cross-call bit buffering → 64-bit codeword slice → upstream BCH
// decode → ParseCodeword → Ingest) so the connector can drive the
// MPT 1327 CC state machine on live IQ.
type BitSink func(bits []byte, baseIdx int)

// Codeword is one MPT 1327 address / data codeword. The package
// supports two wire-level layouts:
//
//   - 38-bit information field (legacy): the historical
//     gophertrunk model used by tests + fixtures that pre-date
//     the BCH wiring. Type + Prefix + Ident + Function fields
//     are populated; Op stays at zero.
//
//   - 48-bit information field (spec-complete): adds the 10-bit
//     Op field per the most-cited MPT 1327 reference, between
//     Ident and Function. Populated by the BCH wiring in
//     process.go under SetBCHMode(BCHOn) so the spec's full
//     information set reaches the state machine.
//
// Field layout (48-bit, MSB-first per field):
//
//   bit 47 (1 bit)   Type    — 0 = address codeword, 1 = data
//                              codeword.
//   bits 46..40 (7) Prefix  — area code prefix.
//   bits 39..27 (13) Ident   — radio / fleet identity (the address).
//   bits 26..17 (10) Op      — operation / opcode field per the
//                              spec. The 4-bit Kind the legacy
//                              `Function >> 13` packs corresponds
//                              to Op's high 4 bits; deployments
//                              that want to interpret the full
//                              10-bit Op (extended opcodes,
//                              vendor sub-types) can read it
//                              under BCHOn.
//   bits 16..0  (17) Function — opcode-specific information bits.
//                              Decoded by the per-codeword accessors
//                              in opcodes.go.
//
// The 38-bit legacy layout drops Op and shifts Function into bits
// 16..0 of a 38-bit info field. AssembleCodeword / ParseCodeword
// preserve the legacy 38-bit wire format for back-compat; the new
// AssembleCodeword48 / ParseCodeword48 + CodewordFromBits48 /
// CodewordBits48 helpers operate on the 48-bit format.
//
// As with the other protocol packages, the field positions follow
// the most-cited public reference; vendor extensions repurpose the
// Function field's sub-layout. Cross-check before trusting live
// captures.
type Codeword struct {
	Type     CodewordType // 1-bit
	Prefix   uint8        // 7-bit
	Ident    uint16       // 13-bit
	Op       uint16       // 10-bit (only populated under the 48-bit / BCHOn path)
	Function uint32       // 17-bit
}

// AssembleCodeword packs a Codeword into 5 bytes (40 bits, with the
// upper 38 bits carrying the codeword and the bottom 2 bits zero-
// padded). Used by tests and for any future encoder work.
func AssembleCodeword(c Codeword) []byte {
	var v uint64
	v |= uint64(c.Type&1) << 37
	v |= uint64(c.Prefix&0x7F) << 30
	v |= uint64(c.Ident&0x1FFF) << 17
	v |= uint64(c.Function & 0x1FFFF)
	v <<= (40 - 38) // left-align into 5 bytes for clean MSB-first transport
	return []byte{
		byte((v >> 32) & 0xFF),
		byte((v >> 24) & 0xFF),
		byte((v >> 16) & 0xFF),
		byte((v >> 8) & 0xFF),
		byte(v & 0xFF),
	}
}

// ParseCodeword reads 5 bytes (the upper 38 bits MSB-first as
// produced by AssembleCodeword) into a Codeword.
func ParseCodeword(info []byte) (Codeword, error) {
	if len(info) != 5 {
		return Codeword{}, fmt.Errorf("mpt1327: codeword info must be 5 bytes, got %d", len(info))
	}
	v := uint64(info[0])<<32 | uint64(info[1])<<24 | uint64(info[2])<<16 |
		uint64(info[3])<<8 | uint64(info[4])
	v >>= (40 - 38)
	return Codeword{
		Type:     CodewordType((v >> 37) & 1),
		Prefix:   uint8((v >> 30) & 0x7F),
		Ident:    uint16((v >> 17) & 0x1FFF),
		Function: uint32(v & 0x1FFFF),
	}, nil
}

// CodewordFromBits packs 38 MSB-first bits (each entry 0/1) into a
// Codeword.
func CodewordFromBits(bits []byte) (Codeword, error) {
	if len(bits) != 38 {
		return Codeword{}, errors.New("mpt1327: codeword requires 38 bits")
	}
	info := make([]byte, 5)
	for i := 0; i < 38; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseCodeword(info)
}

// CodewordBits returns the 38 MSB-first bits of a Codeword.
func CodewordBits(c Codeword) []byte {
	bytes := AssembleCodeword(c)
	out := make([]byte, 38)
	for i := 0; i < 38; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}

// AssembleCodeword48 packs a Codeword into 6 bytes (48 bits)
// MSB-first per the spec's 48-bit information field — same as
// AssembleCodeword but with the 10-bit Op field included between
// Ident and Function. Use this when round-tripping through the
// BCH(64,48,2) framing primitive or any other path that expects
// the full 48-bit info word.
func AssembleCodeword48(c Codeword) []byte {
	var v uint64
	v |= uint64(c.Type&1) << 47
	v |= uint64(c.Prefix&0x7F) << 40
	v |= uint64(c.Ident&0x1FFF) << 27
	v |= uint64(c.Op&0x3FF) << 17
	v |= uint64(c.Function & 0x1FFFF)
	return []byte{
		byte((v >> 40) & 0xFF),
		byte((v >> 32) & 0xFF),
		byte((v >> 24) & 0xFF),
		byte((v >> 16) & 0xFF),
		byte((v >> 8) & 0xFF),
		byte(v & 0xFF),
	}
}

// ParseCodeword48 reads 6 bytes (48 bits MSB-first as produced by
// AssembleCodeword48) into a Codeword with all five fields
// populated.
func ParseCodeword48(info []byte) (Codeword, error) {
	if len(info) != 6 {
		return Codeword{}, fmt.Errorf("mpt1327: codeword48 info must be 6 bytes, got %d", len(info))
	}
	v := uint64(info[0])<<40 | uint64(info[1])<<32 | uint64(info[2])<<24 |
		uint64(info[3])<<16 | uint64(info[4])<<8 | uint64(info[5])
	return Codeword{
		Type:     CodewordType((v >> 47) & 1),
		Prefix:   uint8((v >> 40) & 0x7F),
		Ident:    uint16((v >> 27) & 0x1FFF),
		Op:       uint16((v >> 17) & 0x3FF),
		Function: uint32(v & 0x1FFFF),
	}, nil
}

// CodewordFromBits48 packs 48 MSB-first bits (each entry 0/1) into
// a Codeword with all five fields populated.
func CodewordFromBits48(bits []byte) (Codeword, error) {
	if len(bits) != 48 {
		return Codeword{}, errors.New("mpt1327: codeword48 requires 48 bits")
	}
	info := make([]byte, 6)
	for i := 0; i < 48; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseCodeword48(info)
}

// CodewordBits48 returns the 48 MSB-first bits of a Codeword
// including the Op field.
func CodewordBits48(c Codeword) []byte {
	bytes := AssembleCodeword48(c)
	out := make([]byte, 48)
	for i := 0; i < 48; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}
