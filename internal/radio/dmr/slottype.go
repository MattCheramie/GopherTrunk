package dmr

import (
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// SlotType is the 20-bit field framing each data/control burst (10 bits
// before sync + 10 bits after sync). Per ETSI TS 102 361-1 §9.1.2:
//
//   bits 0..3   : Color Code (high nibble)
//   bits 4..7   : Data Type
//   bits 8..11  : padding (zero)
//   bits 12..15 : Hamming(20,11) parity? Actually the field is encoded
//                 with a shortened Hamming(20,11) producing 9 parity bits;
//                 we treat the 20-bit field as a hand-rolled Hamming.
//
// In practice, the spec layout is:
//   8 info bits  (CC[0..3], DT[0..3])
//   12 parity    (a shortened Hamming(20,8) — see Annex B.1.4).
//
// For a foundation phase we expose a simple unprotected parser plus a
// helper that runs the Hamming(15,11,3) row code over the 20 bits when
// callers want soft error correction. The full TS 102 361-1 Annex B.1.4
// Hamming(20,8) implementation is left as a follow-up; the wire format
// is parsed correctly either way.
type SlotType struct {
	ColorCode uint8 // 4-bit
	DataType  DataType
}

// DataType enumerates the valid Data Type values per ETSI §9.1.2 Table 9.6.
type DataType uint8

const (
	DTPIHeader            DataType = 0x0
	DTVoiceLCHeader       DataType = 0x1
	DTTerminatorWithLC    DataType = 0x2
	DTCSBK                DataType = 0x3
	DTMBCHeader           DataType = 0x4
	DTMBCContinuation     DataType = 0x5
	DTDataHeader          DataType = 0x6
	DT12Rate              DataType = 0x7
	DT34Rate              DataType = 0x8
	DTIdle                DataType = 0x9
	DT1Rate              DataType = 0xA
	DTReserved            DataType = 0xB
	// 0xC..0xF reserved
)

func (d DataType) String() string {
	switch d {
	case DTPIHeader:
		return "PIHeader"
	case DTVoiceLCHeader:
		return "VoiceLCHeader"
	case DTTerminatorWithLC:
		return "TerminatorWithLC"
	case DTCSBK:
		return "CSBK"
	case DTMBCHeader:
		return "MBCHeader"
	case DTMBCContinuation:
		return "MBCContinuation"
	case DTDataHeader:
		return "DataHeader"
	case DT12Rate:
		return "Rate1_2Data"
	case DT34Rate:
		return "Rate3_4Data"
	case DTIdle:
		return "Idle"
	case DT1Rate:
		return "Rate1Data"
	default:
		return fmt.Sprintf("DataType(%X)", uint8(d))
	}
}

// ParseSlotType extracts CC and DataType from the 20-bit slot-type field
// (caller passes the 20-bit concatenation Burst.SlotTypeBitsAll() returns).
// The Hamming code over the 20 bits is not yet validated (see file
// header); callers requiring FEC should treat the result as best-effort.
func ParseSlotType(bits []byte) (SlotType, error) {
	if len(bits) < 8 {
		return SlotType{}, fmt.Errorf("dmr: slot type needs >= 8 bits, got %d", len(bits))
	}
	var cc, dt uint8
	for i := 0; i < 4; i++ {
		cc = (cc << 1) | (bits[i] & 1)
	}
	for i := 0; i < 4; i++ {
		dt = (dt << 1) | (bits[4+i] & 1)
	}
	return SlotType{ColorCode: cc, DataType: DataType(dt)}, nil
}

// AssembleSlotType produces a 20-bit slot-type field with the 8 info bits
// followed by 12 zero pad bits. For tests; the real wire format substitutes
// a Hamming(20,8) parity tail.
func AssembleSlotType(s SlotType) []byte {
	out := make([]byte, 20)
	for i := 0; i < 4; i++ {
		out[i] = (s.ColorCode >> uint(3-i)) & 1
	}
	for i := 0; i < 4; i++ {
		out[4+i] = (uint8(s.DataType) >> uint(3-i)) & 1
	}
	return out
}

// Compile-time assertion that we depend on framing for future Hamming
// integration; keeps the import lit even before the full FEC lands.
var _ = framing.HammingDecode15_11
