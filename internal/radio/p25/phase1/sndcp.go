package phase1

import "errors"

// SNDCP — the Sub-Network Dependent Convergence Protocol layer P25
// packet data carries between the PDU and the encapsulated network
// packet. A reassembled PDU payload (see ReassemblePDU) whose SAP marks
// it as packet data begins with an SNDCP header; ParseSNDCP peels it
// off to expose the network-layer packet (commonly IPv4 — see
// pdu_ip.go).
//
// Working-model header layout (TIA-102.BBAB SNDCP tables are not in the
// repo): the 1-octet header packs a 4-bit PDU type and a 4-bit NSAPI;
// the network packet follows after a 1-octet reserved field.
//
//	byte 0 : bits 4-7 = PDU type, bits 0-3 = NSAPI
//	byte 1 : reserved
//	bytes 2.. : encapsulated network-layer packet

// SNDCPMessage is a parsed SNDCP message.
type SNDCPMessage struct {
	PDUType uint8  // SNDCP PDU type
	NSAPI   uint8  // Network Service Access Point Identifier
	Payload []byte // the encapsulated network-layer packet
}

// ErrSNDCPShort is returned when an SNDCP message is too short to parse.
var ErrSNDCPShort = errors.New("p25/phase1: SNDCP message needs at least 2 bytes")

// ParseSNDCP decodes an SNDCP message from a reassembled PDU payload.
func ParseSNDCP(b []byte) (SNDCPMessage, error) {
	if len(b) < 2 {
		return SNDCPMessage{}, ErrSNDCPShort
	}
	return SNDCPMessage{
		PDUType: b[0] >> 4,
		NSAPI:   b[0] & 0x0F,
		Payload: append([]byte(nil), b[2:]...),
	}, nil
}

// AssembleSNDCP is the inverse of ParseSNDCP.
func AssembleSNDCP(m SNDCPMessage) []byte {
	out := make([]byte, 2+len(m.Payload))
	out[0] = m.PDUType<<4 | m.NSAPI&0x0F
	copy(out[2:], m.Payload)
	return out
}
