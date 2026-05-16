package nxdn

import "fmt"

// LICH is the 8-bit Link Information Channel field per NXDN technical
// specification §6.2.2. Bit layout (MSB-first within the 8-bit info field):
//
//   bit 0  : RFCT  — RF Channel Type (0 = RCCH/control, 1 = RDCH/traffic)
//   bit 1  : FCT   — Function Channel Type LSB
//   bit 2  : FCT   — Function Channel Type MSB
//                     00 = NSACCH, 01 = NUDCH, 10 = Frame Step,
//                     11 = Reserved
//   bit 3  : Option (high bit)
//   bit 4  : Option (low bit)
//   bit 5  : Reserved (0)
//   bit 6  : Direction (0 = outbound BS→MS, 1 = inbound MS→BS)
//   bit 7  : Parity   (even parity over bits 0..6)
//
// On the wire, every information bit is transmitted twice (1-to-2
// repetition), so an 8-bit LICH info block becomes 16 wire bits =
// LICHWireDibits.
//
// Decoder: take majority vote per info bit pair; if both copies disagree
// the bit is flagged as soft. Parity is then validated.

type RFChannelType uint8

const (
	RFChControl RFChannelType = 0 // RCCH
	RFChTraffic RFChannelType = 1 // RDCH
)

func (r RFChannelType) String() string {
	if r == RFChControl {
		return "RCCH"
	}
	return "RDCH"
}

type FunctionChannelType uint8

const (
	FCTNSACCH    FunctionChannelType = 0
	FCTNUDCH     FunctionChannelType = 1
	FCTFrameStep FunctionChannelType = 2
	FCTReserved  FunctionChannelType = 3
)

func (f FunctionChannelType) String() string {
	switch f {
	case FCTNSACCH:
		return "NSACCH"
	case FCTNUDCH:
		return "NUDCH"
	case FCTFrameStep:
		return "FrameStep"
	default:
		return "Reserved"
	}
}

type Direction uint8

const (
	DirectionOutbound Direction = 0
	DirectionInbound  Direction = 1
)

func (d Direction) String() string {
	if d == DirectionOutbound {
		return "outbound"
	}
	return "inbound"
}

// LICH holds the parsed 8-bit information field.
type LICH struct {
	RFCh      RFChannelType
	FCT       FunctionChannelType
	Option    uint8 // 2 bits
	Direction Direction
	ParityOK  bool // parity check result
}

// AssembleLICH packs an 8-bit information field per the layout above. The
// caller's parity-bit input is ignored; we compute it.
func AssembleLICH(l LICH) byte {
	var b byte
	b |= byte(l.RFCh) << 7
	b |= byte(l.FCT&1) << 6
	b |= byte((l.FCT>>1)&1) << 5
	b |= byte((l.Option>>1)&1) << 4
	b |= byte(l.Option&1) << 3
	// bit 5 reserved (0)
	b |= byte(l.Direction&1) << 1
	// parity over bits 0..6 (which are stored at byte positions 7..1)
	parity := byte(0)
	for i := 1; i < 8; i++ {
		parity ^= (b >> uint(i)) & 1
	}
	b |= parity & 1
	return b
}

// ParseLICH decodes an 8-bit information field. ParityOK reflects whether
// the trailing parity bit matches even-parity over bits 0..6.
func ParseLICH(b byte) LICH {
	parity := byte(0)
	for i := 1; i < 8; i++ {
		parity ^= (b >> uint(i)) & 1
	}
	return LICH{
		RFCh:      RFChannelType((b >> 7) & 1), // info bit 0 → byte bit 7
		FCT:       FunctionChannelType(((b >> 4) & 2) | ((b >> 6) & 1)),
		Option:    (b >> 3) & 0x3,          // bits 4..3 → Option[1..0]
		Direction: Direction((b >> 1) & 1), // info bit 6 → byte bit 1
		ParityOK:  (b & 1) == parity,
	}
}

// EncodeLICHWire converts an 8-bit information field into 16 wire bits by
// doubling each bit (b -> b,b). Returns 16 bits MSB-first within the byte.
func EncodeLICHWire(info byte) []byte {
	out := make([]byte, 16)
	for i := 0; i < 8; i++ {
		bit := (info >> uint(7-i)) & 1
		out[2*i] = bit
		out[2*i+1] = bit
	}
	return out
}

// DecodeLICHWire reverses EncodeLICHWire via majority voting on each bit
// pair. Returns the 8-bit info byte and the number of pairs that
// disagreed (each disagreement implies at least one bit error among that
// pair). Errors >0 means the LICH was noisy; the caller should treat the
// result as soft.
func DecodeLICHWire(wire []byte) (byte, int) {
	if len(wire) != 16 {
		panic(fmt.Sprintf("nxdn: DecodeLICHWire requires 16 bits, got %d", len(wire)))
	}
	var b byte
	disagreements := 0
	for i := 0; i < 8; i++ {
		a := wire[2*i] & 1
		c := wire[2*i+1] & 1
		var bit byte
		if a == c {
			bit = a
		} else {
			disagreements++
			bit = a // tie → take first; could be either, neither is reliable
		}
		b |= bit << uint(7-i)
	}
	return b, disagreements
}
