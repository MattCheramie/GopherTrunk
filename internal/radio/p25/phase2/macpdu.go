package phase2

import (
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Structure of a decoded P25 Phase 2 ACCH message.
//
// What DecodeACCHBurst returns is not a single MAC PDU. It is a header byte
// followed by a *sequence* of MAC structures, each with its own opcode and its
// own length — a busy channel routinely packs a GROUP VOICE CHANNEL USER and
// an ADJACENT STATUS BROADCAST into one 144-bit FACCH-S payload.
//
// The header byte carries two fields:
//
//	bits 7-5  MAC PDU type — what the channel is doing (idle, active, PTT, …)
//	bits 4-2  offset to the first 4V voice burst
//	bits 1-0  reserved
//
// Reading that byte as an opcode, which this package did until 2026-09-01,
// mixes the two: 0x9C is not opcode 156, it is an ACTIVE channel whose first
// voice burst is 7 slots out, and the opcode the caller wants is in the *next*
// byte.

// MACPDUType is the MAC PDU type from the header byte's top three bits.
type MACPDUType uint8

const (
	// MACSignal carries signalling only, with no associated voice.
	MACSignal MACPDUType = 0
	// MACPushToTalk begins a transmission; it carries the encryption sync
	// (ALGID / KID / MI) for the call that follows.
	MACPushToTalk MACPDUType = 1
	// MACEndPushToTalk ends a transmission.
	MACEndPushToTalk MACPDUType = 2
	// MACIdle is an idle channel.
	MACIdle MACPDUType = 3
	// MACActive is an active channel carrying voice.
	MACActive MACPDUType = 4
	// MACHangtime is the hang-time following a transmission.
	MACHangtime MACPDUType = 6
)

func (t MACPDUType) String() string {
	switch t {
	case MACSignal:
		return "SIGNAL"
	case MACPushToTalk:
		return "PTT"
	case MACEndPushToTalk:
		return "END_PTT"
	case MACIdle:
		return "IDLE"
	case MACActive:
		return "ACTIVE"
	case MACHangtime:
		return "HANGTIME"
	}
	return "RESERVED"
}

// CarriesVoice reports whether the channel state implies an active voice
// transmission in the same superframe.
func (t MACPDUType) CarriesVoice() bool {
	return t == MACPushToTalk || t == MACActive
}

// carriesStructureSequence reports whether the PDU body is a sequence of MAC
// structures to walk, rather than one structure spanning the whole PDU.
//
// SIGNAL, IDLE, ACTIVE and HANGTIME carry a sequence. PTT and END PTT are
// single structures whose opcode is the header byte itself — that is how both
// reference decoders report them (0x20 / 0x24 push-to-talk, 0x5C end
// push-to-talk), and it keeps the encryption-sync fields at the offsets a
// caller expects.
//
// The two reserved types, 5 and 7, are treated the same way. OP25 ignores type
// 5 entirely; SDRtrunk reports it as a single vendor structure named by the
// header byte, and every type-5 PDU in the ground-truth corpus is one it
// labels an unknown vendor opcode. Neither decoder knows the body's framing,
// so walking into it would invent structures rather than find them.
func (t MACPDUType) carriesStructureSequence() bool {
	switch t {
	case MACSignal, MACIdle, MACActive, MACHangtime:
		return true
	}
	return false
}

// MACMessage is one decoded ACCH message: the channel state from its header
// plus every MAC structure the walk could delimit.
type MACMessage struct {
	// Type is the channel state from the header byte.
	Type MACPDUType
	// VoiceOffset is the header's offset field: how many slots ahead the
	// first 4V voice burst sits. Meaningful for MACPushToTalk and MACActive.
	VoiceOffset uint8
	// Structures holds the MAC structures in transmission order.
	Structures []MACPDU
	// Truncated reports that the walk stopped early because a structure's
	// opcode has no known length — the structures before it are still good,
	// but the rest of the message was not read.
	//
	// This is deliberate: guessing a length desynchronises the walk and the
	// bytes that follow parse as a structure that was never transmitted,
	// which is how a fabricated grant with a plausible talkgroup gets into
	// the archive. Stopping loses information; guessing invents it.
	Truncated bool
}

// ErrShortMACMessage is returned when a message is too short to hold even a
// header byte.
var ErrShortMACMessage = errors.New("p25/phase2: MAC message too short")

// macStructureLen gives the on-wire length in bytes of each MAC structure this
// decoder can delimit, keyed by opcode.
//
// The spec that carries the full 256-entry length table is paywalled, so these
// lengths were *derived* instead, from a ground-truth corpus: 185 distinct MAC
// PDUs decoded from real P25 Phase 2 air by SDRtrunk, which reports the opcode
// of every structure it finds in each one. For a two-structure PDU the
// second structure's opcode byte pins the first structure's length, and taking
// the intersection of the candidate positions over every observation leaves
// exactly one surviving length per opcode:
//
//	0x01  7 bytes  n=46   GROUP VOICE CHANNEL USER ABBREVIATED
//	0x42  9 bytes  n=32   GROUP VOICE CHANNEL GRANT UPDATE IMPLICIT
//	0x7C  9 bytes  n=43   ADJACENT STATUS BROADCAST IMPLICIT
//	0x80  8 bytes  n=14   MOTOROLA GROUP REGROUP VOICE CHANNEL USER ABBREV
//	0x83  7 bytes  n=1    MOTOROLA GROUP REGROUP VOICE CHANNEL UPDATE
//
// n is how many independent PDUs agree. 0x83 rests on a single observation and
// is the one entry here that should not be trusted far; it appears as a
// trailing structure, where its length does not affect anything already parsed.
//
// Opcodes absent from this table stop the walk rather than being guessed at.
var macStructureLen = map[Opcode]int{
	0x01: 7,
	0x42: 9,
	0x7C: 9,
	0x80: 8,
	0x83: 7,
}

// macNullInformation is the padding opcode: it fills whatever is left of the
// message, so it terminates the walk without contributing a structure.
const macNullInformation Opcode = 0x00

// ParseACCHMessage splits a decoded ACCH message into its header and MAC
// structures. msg is the bit slice DecodeACCHBurst returns — information bits
// followed by their CRC-12 — and the CRC is dropped here.
//
// A PTT or END PTT message is a single structure spanning the whole PDU, and
// its opcode is the header byte itself; that is how the reference decoders
// report it (0x20 / 0x24 push-to-talk, 0x5C end push-to-talk) and it keeps the
// encryption-sync fields at the offsets a caller expects.
func ParseACCHMessage(msg []byte) (MACMessage, error) {
	if len(msg) < 12+8 {
		return MACMessage{}, ErrShortMACMessage
	}
	info := framing.PackBitsMSB(msg[:len(msg)-12])
	if len(info) < 1 {
		return MACMessage{}, ErrShortMACMessage
	}
	out := MACMessage{
		Type:        MACPDUType(info[0] >> 5 & 0x7),
		VoiceOffset: info[0] >> 2 & 0x7,
	}
	if !out.Type.carriesStructureSequence() {
		pdu, err := ParseMACPDU(info)
		if err != nil {
			return out, err
		}
		out.Structures = []MACPDU{pdu}
		return out, nil
	}
	for ptr := 1; ptr < len(info); {
		op := Opcode(info[ptr])
		if op == macNullInformation {
			break // padding to the end of the PDU
		}
		n, ok := macStructureLen[op]
		if !ok || ptr+n > len(info) {
			// The opcode is readable and the bytes are vouched for by the RS
			// and the CRC; only where this structure *ends* is unknown. Emit
			// it with everything that follows and stop: MAC structure fields
			// are laid out from the structure's start, so a caller reading
			// them is unaffected by trailing bytes, and dropping a structure
			// we can name loses more than it protects.
			//
			// The walk stops either way. Continuing past an unknown length
			// would resynchronise onto the wrong byte and manufacture a
			// structure that was never transmitted.
			out.Truncated = true
			if pdu, err := ParseMACPDU(info[ptr:]); err == nil {
				out.Structures = append(out.Structures, pdu)
			}
			break
		}
		pdu, err := ParseMACPDU(info[ptr : ptr+n])
		if err != nil {
			out.Truncated = true
			break
		}
		out.Structures = append(out.Structures, pdu)
		ptr += n
	}
	return out, nil
}

// SlotType maps the MAC PDU type onto the sub-frame SlotType the rest of this
// package routes on, so callers that switch on a slot type — the encryption
// sync rides the PTT slot, see mac_standard.go — keep working now that the
// channel state comes from the MAC header rather than from the ISCH.
func (t MACPDUType) SlotType() SlotType {
	switch t {
	case MACPushToTalk:
		return SlotTypeMACPTT
	case MACEndPushToTalk:
		return SlotTypeMACEnd
	case MACIdle:
		return SlotTypeMACIdle
	case MACActive:
		return SlotTypeMACActive
	case MACHangtime:
		return SlotTypeMACHangtime
	case MACSignal:
		return SlotTypeMACSignaling
	}
	return SlotTypeUnknown
}
