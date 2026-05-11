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

// Codeword is one MPT 1327 address / data codeword — 38 information
// bits packed into the upper 38 bits of a 64-bit transmission unit
// (the lower 26 bits hold the BCH(63,38) parity, which the upstream
// FEC consumes before this package sees the codeword).
//
// Field layout in the 38-bit info portion:
//
//   bit 37 (1 bit)   Type    — 0 = address codeword, 1 = data
//                              codeword. Address codewords carry
//                              trunked signalling; data codewords
//                              transport short messages and aren't
//                              the focus here.
//   bits 36..30 (7) Prefix  — area code prefix.
//   bits 29..17 (13) Ident   — radio / fleet identity (the address).
//   bits 16..0  (17) Function — opcode-specific information bits.
//                              Decoded by the per-codeword accessors
//                              in opcodes.go.
//
// As with the other protocol packages, the field positions follow
// the most-cited public reference; vendor extensions repurpose the
// Function field's sub-layout. Cross-check before trusting live
// captures.
type Codeword struct {
	Type     CodewordType // 1-bit
	Prefix   uint8        // 7-bit
	Ident    uint16       // 13-bit
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
