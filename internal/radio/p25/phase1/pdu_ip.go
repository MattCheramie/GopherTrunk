package phase1

import (
	"encoding/binary"
	"errors"
	"net/netip"
)

// IPv4 header extraction for P25 packet data. When an SNDCP message
// (see sndcp.go) encapsulates an IPv4 packet, ParseIPv4 surfaces its
// source / destination addresses and protocol — the "who is talking to
// whom" of a P25 data call. Unlike the PDU and SNDCP layers above it,
// the IPv4 header is a standard, fully-specified format (RFC 791), so
// this layer is exact rather than a working model.

// IPv4Packet is the parsed header of an IPv4 packet.
type IPv4Packet struct {
	HeaderLen   int // header length in bytes
	TotalLength uint16
	Protocol    uint8 // 1 = ICMP, 6 = TCP, 17 = UDP
	Source      netip.Addr
	Dest        netip.Addr
}

// ErrNotIPv4 is returned when the input is not a well-formed IPv4
// packet header.
var ErrNotIPv4 = errors.New("p25/phase1: not an IPv4 packet")

// ParseIPv4 decodes the fixed 20-byte IPv4 header at the start of b.
func ParseIPv4(b []byte) (IPv4Packet, error) {
	if len(b) < 20 {
		return IPv4Packet{}, ErrNotIPv4
	}
	if b[0]>>4 != 4 {
		return IPv4Packet{}, ErrNotIPv4
	}
	ihl := int(b[0]&0x0F) * 4
	if ihl < 20 || ihl > len(b) {
		return IPv4Packet{}, ErrNotIPv4
	}
	return IPv4Packet{
		HeaderLen:   ihl,
		TotalLength: binary.BigEndian.Uint16(b[2:4]),
		Protocol:    b[9],
		Source:      netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}),
		Dest:        netip.AddrFrom4([4]byte{b[16], b[17], b[18], b[19]}),
	}, nil
}
