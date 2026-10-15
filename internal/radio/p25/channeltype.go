package p25

// TDMAChannelType is what one 4-bit IDEN_UP_TDMA Channel Type code means:
// access mode, nominal channel bandwidth and slots per carrier. The same
// code rides the Phase 1 TSBK (opcode 0x33) and both Phase 2 MAC forms
// (0x73 abbreviated, 0xF3 extended), so both decoders share this table.
type TDMAChannelType struct {
	TDMA            bool
	BandwidthHz     uint32
	SlotsPerCarrier uint8
}

// ParseTDMAChannelType maps a Channel Type code per TIA-102.AABC,
// cross-checked against two independent decoders that agree: SDRTrunk
// `ChannelType` (TYPE_0..TYPE_5) and OP25 `tk_p25.py`
// (`slots_per_carrier = [1,1,1,2,4,2,...]`).
//
//	0 FDMA 12.5 kHz 1 slot (half-rate vocoder)
//	1 FDMA 12.5 kHz 1 slot (full-rate vocoder)
//	2 FDMA 6.25 kHz 1 slot
//	3 TDMA 12.5 kHz 2 slots  (the Phase 2 carrier)
//	4 TDMA 25 kHz   4 slots
//	5 TDMA 12.5 kHz 2 slots  (H-D8PSK simulcast)
//
// Codes 6..15 are reserved. OP25 treats them as 2-slot TDMA; we follow
// it, since a reserved code is no evidence of FDMA. Their bandwidth
// reads 0.
func ParseTDMAChannelType(code uint8) TDMAChannelType {
	switch code & 0x0F {
	case 0x0, 0x1:
		return TDMAChannelType{TDMA: false, BandwidthHz: 12_500, SlotsPerCarrier: 1}
	case 0x2:
		return TDMAChannelType{TDMA: false, BandwidthHz: 6_250, SlotsPerCarrier: 1}
	case 0x3, 0x5:
		return TDMAChannelType{TDMA: true, BandwidthHz: 12_500, SlotsPerCarrier: 2}
	case 0x4:
		return TDMAChannelType{TDMA: true, BandwidthHz: 25_000, SlotsPerCarrier: 4}
	default:
		return TDMAChannelType{TDMA: true, BandwidthHz: 0, SlotsPerCarrier: 2}
	}
}
