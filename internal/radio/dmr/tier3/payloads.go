package tier3

import "encoding/binary"

// TVGrant (CSBKO 0x30) is the TalkGroup Voice Channel Grant per ETSI
// TS 102 361-4 §7.1.2.1. Payload layout (8 octets):
//
//	octet 0   : Service Options
//	octet 1-3 : Destination address (talkgroup, 24-bit)
//	octet 4-6 : Source address (subscriber, 24-bit)
//	octet 7   : bit 7 = Timeslot (0 = TS1, 1 = TS2)
//	            bits 6-0 = LCN (Logical Channel Number, 7-bit)
//
// The LCN feeds a per-system band-plan resolver to recover the
// downlink frequency the engine retunes a Voice device to.
type TVGrant struct {
	ServiceOptions uint8
	GroupAddress   uint32 // 24-bit
	SourceID       uint32 // 24-bit
	LCN            uint8  // 7-bit logical channel number
	Timeslot       uint8  // 0 = TS1, 1 = TS2
}

func ParseTVGrant(p [8]byte) TVGrant {
	return TVGrant{
		ServiceOptions: p[0],
		GroupAddress:   uint32(p[1])<<16 | uint32(p[2])<<8 | uint32(p[3]),
		SourceID:       uint32(p[4])<<16 | uint32(p[5])<<8 | uint32(p[6]),
		LCN:            p[7] & 0x7F,
		Timeslot:       (p[7] >> 7) & 0x01,
	}
}

// PVGrant (CSBKO 0x31) is the Private Voice Channel Grant. Layout
// matches TVGrant but the destination address is a subscriber rather
// than a talkgroup. The same LCN + Timeslot encoding applies.
type PVGrant struct {
	ServiceOptions uint8
	DestinationID  uint32 // 24-bit
	SourceID       uint32 // 24-bit
	LCN            uint8
	Timeslot       uint8
}

func ParsePVGrant(p [8]byte) PVGrant {
	return PVGrant{
		ServiceOptions: p[0],
		DestinationID:  uint32(p[1])<<16 | uint32(p[2])<<8 | uint32(p[3]),
		SourceID:       uint32(p[4])<<16 | uint32(p[5])<<8 | uint32(p[6]),
		LCN:            p[7] & 0x7F,
		Timeslot:       (p[7] >> 7) & 0x01,
	}
}

// Aloha (CSBKO 0x04) — outbound Aloha message advertising the trunked
// control channel. Payload first nibble is the Site Time Slot (STS) bitmap;
// the remainder carries CC information.
type Aloha struct {
	SyncRandom    uint8 // 4 bits
	NRandWaits    uint8 // 4 bits
	BackoffNumber uint8 // 4 bits
	UplinkActive  bool
	SystemID      uint16
	Reserved      uint16
}

func ParseAloha(p [8]byte) Aloha {
	return Aloha{
		SyncRandom:    p[0] >> 4,
		NRandWaits:    p[0] & 0x0F,
		BackoffNumber: p[1] >> 4,
		UplinkActive:  p[1]&0x08 != 0,
		SystemID:      binary.BigEndian.Uint16(p[2:4]),
		Reserved:      binary.BigEndian.Uint16(p[6:8]),
	}
}

// AdjacentSiteStatus (CSBKO 0x38). Carries an adjacent-site descriptor:
// the site identifier, its control-channel LCN, color code, and a
// quality / availability flag set.
type AdjacentSiteStatus struct {
	SystemID  uint16
	SiteID    uint16
	LCN       uint16
	ColorCode uint8
	Status    uint8
}

func ParseAdjacentSiteStatus(p [8]byte) AdjacentSiteStatus {
	return AdjacentSiteStatus{
		SystemID:  binary.BigEndian.Uint16(p[0:2]),
		SiteID:    binary.BigEndian.Uint16(p[2:4]),
		LCN:       binary.BigEndian.Uint16(p[4:6]),
		ColorCode: p[6] >> 4,
		Status:    p[6] & 0x0F,
	}
}

// SystemInfoBroadcast (CSBKO 0x39). Announces the System ID, RFSS, and
// site number associated with the listening channel.
type SystemInfoBroadcast struct {
	SystemID uint16
	RFSSID   uint8
	SiteID   uint8
	NetMask  uint8
	Reserved uint16
}

func ParseSystemInfoBroadcast(p [8]byte) SystemInfoBroadcast {
	return SystemInfoBroadcast{
		SystemID: binary.BigEndian.Uint16(p[0:2]),
		RFSSID:   p[2],
		SiteID:   p[3],
		NetMask:  p[4],
		Reserved: binary.BigEndian.Uint16(p[6:8]),
	}
}
