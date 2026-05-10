package dmr

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// SlotType is the 20-bit field framing each data/control burst (10 bits
// before sync + 10 bits after sync). Per ETSI TS 102 361-1 §6.4.2 + Annex
// B.1.1, the field carries 8 information bits (4-bit Color Code + 4-bit
// Data Type) followed by 12 parity bits computed by the (20,8,7)
// shortened Hamming / Golay slot-type code.
//
//	bits 0..3   : Color Code
//	bits 4..7   : Data Type
//	bits 8..19  : (20,8) Hamming parity
type SlotType struct {
	ColorCode uint8 // 4-bit
	DataType  DataType
}

// DataType enumerates the valid Data Type values per ETSI §9.1.2 Table 9.6.
type DataType uint8

const (
	DTPIHeader         DataType = 0x0
	DTVoiceLCHeader    DataType = 0x1
	DTTerminatorWithLC DataType = 0x2
	DTCSBK             DataType = 0x3
	DTMBCHeader        DataType = 0x4
	DTMBCContinuation  DataType = 0x5
	DTDataHeader       DataType = 0x6
	DT12Rate           DataType = 0x7
	DT34Rate           DataType = 0x8
	DTIdle             DataType = 0x9
	DT1Rate            DataType = 0xA
	DTReserved         DataType = 0xB
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

// ErrSlotTypeUncorrectable is returned by ParseSlotType when the (20,8)
// Hamming decoder cannot recover the codeword within its t=3 error-
// correction radius.
var ErrSlotTypeUncorrectable = errors.New("dmr: slot type Hamming(20,8) uncorrectable")

// ParseSlotType extracts CC and DataType from the 20-bit slot-type
// field, running the (20,8,7) shortened Hamming decoder over the
// codeword. Returns the decoded SlotType, the number of bit errors
// corrected (0 on a clean codeword), and a non-nil error if the
// codeword is uncorrectable.
//
// Callers pass the 20-bit concatenation that Burst.SlotTypeBitsAll()
// produces (bits[0..7] = info MSB-first, bits[8..19] = parity MSB-first).
func ParseSlotType(bits []byte) (SlotType, int, error) {
	if len(bits) < 20 {
		return SlotType{}, -1, fmt.Errorf("dmr: slot type needs 20 bits, got %d", len(bits))
	}
	var cw uint32
	for i := 0; i < 20; i++ {
		if bits[i]&1 != 0 {
			cw |= uint32(1) << uint(19-i)
		}
	}
	data, errs := framing.HammingDecode20_8(cw)
	if errs < 0 {
		return SlotType{}, -1, ErrSlotTypeUncorrectable
	}
	return SlotType{
		ColorCode: (data >> 4) & 0x0F,
		DataType:  DataType(data & 0x0F),
	}, errs, nil
}

// AssembleSlotType produces the 20-bit slot-type codeword (info + 12
// parity bits) ready to splice into a burst around the sync. Useful for
// tests and synthetic streams.
func AssembleSlotType(s SlotType) []byte {
	info := (uint8(s.ColorCode&0x0F) << 4) | uint8(s.DataType&0x0F)
	cw := framing.HammingEncode20_8(info)
	out := make([]byte, 20)
	for i := 0; i < 20; i++ {
		if cw&(uint32(1)<<uint(19-i)) != 0 {
			out[i] = 1
		}
	}
	return out
}
