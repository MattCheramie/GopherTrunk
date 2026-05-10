package ysf

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// FICH carries the structured fields extracted from a Frame
// Information Channel block. The Frame Sync Word identifies *that* a
// YSF transmission is on-air; the FICH identifies *what kind* — call
// type (group / radio ID), data type (voice / data variants), and the
// frame's position inside its block + transmission sequence.
//
// On-air the FICH is 100 channel bits: 32 bits of information + 16
// bits CRC, encoded with a K=5 ½-rate convolutional code (Trellis)
// and interleaved across the FICH region of the frame. This package
// ships the bit-level Parse / Assemble against the post-Trellis
// 48-bit info+CRC pair; the Trellis decoder + control-channel
// integration lands in a follow-up so the spec interpretation can
// be reviewed independently of the FEC math.
//
// Bit layout (32 info bits, MSB-first, packed into 4 octets) per the
// publicly documented Yaesu System Fusion specification — same shape
// open-source decoders (DSDcc, MMDVMHost) use:
//
//	octet 0 bits 7..6  FT   (Frame Type)
//	octet 0 bits 5..4  CT   (Call Type / Call Sign Mode)
//	octet 0 bits 3..2  BN   (Block Number)
//	octet 0 bits 1..0  BT   (Block Total)
//	octet 1 bits 7..5  FN   (Frame Number)
//	octet 1 bits 4..2  FT2  (Frame Total inside the block)
//	octet 1 bits 1..0  DT   (Data Type)
//	octet 2 bit  7     VoIP (1 = transmission carries WIRES-X / VoIP)
//	octet 2 bits 6..5  DT2  (Data Type 2 / Mode-2 sub-field)
//	octet 2 bit  4     SQM  (Squelch Mode: 0 = open, 1 = closed)
//	octet 2 bits 3..0  SQ   (Squelch Code, high 4 bits)
//	octet 3 bits 7..5  SQ   (Squelch Code, low 3 bits — 7 bits total)
//	octet 3 bits 4..3  DEV  (Device / Reserved)
//	octet 3 bits 2..0  ----  unused / reserved
//
// Octets 4..5 carry the CRC-16 trailer over octets 0..3, computed
// with poly 0x1021 and initial value 0x0000 (XMODEM / CCITT-true).
type FICH struct {
	FrameType    FrameType
	CallType     CallType
	BlockNumber  uint8 // 2-bit BN
	BlockTotal   uint8 // 2-bit BT
	FrameNumber  uint8 // 3-bit FN within the current block
	FrameTotal   uint8 // 3-bit FT total frames in the block
	DataType     DataType
	VoIP         bool
	DataType2    uint8 // 2-bit DT2 / Mode-2 sub-field
	SquelchMode  bool  // false = open carrier, true = code-squelch active
	SquelchCode  uint8 // 7-bit SQ
	Device       uint8 // 2-bit DEV / reserved
}

// FrameType is the 2-bit FT field — what kind of frame this is in
// the transmission's lifecycle. (Confusingly the spec also uses "FT"
// for "Frame Total"; this package exposes Frame Total as
// FrameTotal to keep the two distinct.)
type FrameType uint8

const (
	FrameTypeHeader     FrameType = 0x0 // start of transmission
	FrameTypeComms      FrameType = 0x1 // mid-transmission voice / data
	FrameTypeTerminator FrameType = 0x2 // end of transmission
	FrameTypeTest       FrameType = 0x3 // test / loopback
)

func (f FrameType) String() string {
	switch f {
	case FrameTypeHeader:
		return "Header"
	case FrameTypeComms:
		return "Communications"
	case FrameTypeTerminator:
		return "Terminator"
	case FrameTypeTest:
		return "Test"
	default:
		return fmt.Sprintf("FrameType(%X)", uint8(f))
	}
}

// CallType is the 2-bit CT / Call Sign Mode field.
type CallType uint8

const (
	CallTypeGroup    CallType = 0x0 // group call
	CallTypeRadioID  CallType = 0x1 // private (radio-ID-addressed) call
	CallTypeReserved CallType = 0x2
	CallTypeReservedB CallType = 0x3
)

func (c CallType) String() string {
	switch c {
	case CallTypeGroup:
		return "Group"
	case CallTypeRadioID:
		return "RadioID"
	case CallTypeReserved, CallTypeReservedB:
		return fmt.Sprintf("Reserved(%X)", uint8(c))
	default:
		return fmt.Sprintf("CallType(%X)", uint8(c))
	}
}

// DataType is the 2-bit DT field — which mode the transmission's
// payload is in. The "V/D mode 1" variants carry voice + data
// multiplexed in the DCH region; the full-rate variants carry only
// one of the two.
type DataType uint8

const (
	DataTypeVDMode1     DataType = 0x0 // V/D mode 1 (½-rate voice + data)
	DataTypeDataFR      DataType = 0x1 // Data Full Rate
	DataTypeVDMode2     DataType = 0x2 // V/D mode 2
	DataTypeVoiceFR     DataType = 0x3 // Voice Full Rate
)

func (d DataType) String() string {
	switch d {
	case DataTypeVDMode1:
		return "VDMode1"
	case DataTypeDataFR:
		return "DataFullRate"
	case DataTypeVDMode2:
		return "VDMode2"
	case DataTypeVoiceFR:
		return "VoiceFullRate"
	default:
		return fmt.Sprintf("DataType(%X)", uint8(d))
	}
}

// ErrFICHLength is returned by ParseFICH when the supplied buffer is
// not exactly 6 octets (4 octets info + 2 octets CRC).
var ErrFICHLength = errors.New("ysf: FICH info+CRC must be 6 octets")

// CRCError is returned by ParseFICH when the trailer CRC doesn't
// match the locally-computed CRC over the four info octets. The
// partially-parsed FICH is still returned so callers can log
// diagnostics.
var CRCError = errors.New("ysf: FICH CRC mismatch")

// ParseFICH consumes 6 octets (32 info bits + 16-bit CRC trailer)
// and returns a parsed FICH. The 16-bit trailer is CRC-CCITT over
// the leading 4 octets with initial value 0x0000.
func ParseFICH(b []byte) (FICH, error) {
	if len(b) != 6 {
		return FICH{}, fmt.Errorf("%w, got %d", ErrFICHLength, len(b))
	}
	f := FICH{
		FrameType:   FrameType(b[0] >> 6 & 0x3),
		CallType:    CallType(b[0] >> 4 & 0x3),
		BlockNumber: b[0] >> 2 & 0x3,
		BlockTotal:  b[0] & 0x3,
		FrameNumber: b[1] >> 5 & 0x7,
		FrameTotal:  b[1] >> 2 & 0x7,
		DataType:    DataType(b[1] & 0x3),
		VoIP:        b[2]&0x80 != 0,
		DataType2:   b[2] >> 5 & 0x3,
		SquelchMode: b[2]&0x10 != 0,
		// SQ (7 bits): bits 3..0 of octet 2 are the high 4 bits;
		// bits 7..5 of octet 3 are the low 3 bits.
		SquelchCode: (b[2]&0x0F)<<3 | b[3]>>5&0x7,
		Device:      b[3] >> 3 & 0x3,
	}
	wantCRC := framing.CRCCCITTWithInit(b[:4], 0x0000)
	gotCRC := binary.BigEndian.Uint16(b[4:6])
	if wantCRC != gotCRC {
		return f, CRCError
	}
	return f, nil
}

// AssembleFICH packs an FICH into the on-air 6-octet info+CRC layout.
// Out-of-range fields are silently truncated to their bit widths so
// the encoder doesn't panic on hand-built test vectors that exceed
// the spec.
func AssembleFICH(f FICH) []byte {
	out := make([]byte, 6)
	out[0] = byte(f.FrameType&0x3)<<6 |
		byte(f.CallType&0x3)<<4 |
		(f.BlockNumber&0x3)<<2 |
		(f.BlockTotal & 0x3)
	out[1] = (f.FrameNumber&0x7)<<5 |
		(f.FrameTotal&0x7)<<2 |
		byte(f.DataType&0x3)
	sqHi := (f.SquelchCode >> 3) & 0x0F
	sqLo := f.SquelchCode & 0x07
	out[2] = sqHi
	if f.VoIP {
		out[2] |= 0x80
	}
	out[2] |= (f.DataType2 & 0x3) << 5
	if f.SquelchMode {
		out[2] |= 0x10
	}
	out[3] = sqLo<<5 | (f.Device&0x3)<<3
	binary.BigEndian.PutUint16(out[4:6], framing.CRCCCITTWithInit(out[:4], 0x0000))
	return out
}
