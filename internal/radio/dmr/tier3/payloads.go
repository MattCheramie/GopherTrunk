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

// Aloha (CSBKO 0x19, C_ALOHA) — outbound Aloha message advertising the
// trunked control channel. Payload first nibble is the Site Time Slot
// (STS) bitmap; the remainder carries CC information. SystemID is the raw
// 16-bit network/site identity field (not dsd-neo's decoded syscode); the
// CC state machine uses it as a stable lock key.
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

// Broadcast announcement sub-types carried inside C_BCAST (CSBKO 0x28,
// OpBcast). The 5-bit anncd_type occupies the top of the first payload
// octet — ETSI TS 102 361-4 §7.2.x / IanWraith DMRDecode. Validated
// against real off-air C_BCAST bursts: Gen_Site_Params (7) decodes from
// payload[0]=0x3A, CallTimer_Parms (1) from payload[0]=0x0F.
const (
	AnncWDTSCC         uint8 = 0 // announce / withdraw TSCC
	AnncCallTimerParms uint8 = 1 // specify call-timer parameters
	AnncVoteNow        uint8 = 2 // vote-now advice
	AnncLocalTime      uint8 = 3 // broadcast local time
	AnncMassReg        uint8 = 4 // mass registration
	AnncChanFreq       uint8 = 5 // logical-channel / frequency relationship
	AnncAdjacentSite   uint8 = 6 // adjacent-site information
	AnncGenSiteParms   uint8 = 7 // general site parameters
)

// BroadcastAnnouncement is a parsed C_BCAST CSBK. Type selects how the
// remaining octets are interpreted; Payload retains the full 8-octet
// broadcast payload so type-specific helpers can re-read it.
type BroadcastAnnouncement struct {
	Type    uint8
	Payload [8]byte
}

// ParseBroadcast extracts the anncd_type from a C_BCAST payload.
func ParseBroadcast(p [8]byte) BroadcastAnnouncement {
	return BroadcastAnnouncement{Type: p[0] >> 3, Payload: p}
}

// AdjacentSiteStatus is an adjacent-site descriptor carried as the
// Adjacent_Site (anncd_type 6) sub-type of C_BCAST: the neighbour's
// system + site identifier, its control-channel LCN, and color code.
// The sub-fields follow the anncd_type octet, so they are read from
// payload[1:].
type AdjacentSiteStatus struct {
	SystemID  uint16
	SiteID    uint16
	LCN       uint16
	ColorCode uint8
}

func ParseAdjacentSite(p [8]byte) AdjacentSiteStatus {
	return AdjacentSiteStatus{
		SystemID:  binary.BigEndian.Uint16(p[1:3]),
		SiteID:    binary.BigEndian.Uint16(p[3:5]),
		LCN:       binary.BigEndian.Uint16(p[5:7]),
		ColorCode: p[7] >> 4,
	}
}

// SystemInfoBroadcast announces the System ID, RFSS, and site number for
// the camped site. For standard ETSI traffic this rides in the
// Gen_Site_Params (anncd_type 7) sub-type of C_BCAST (fields read from
// payload[1:] via ParseGenSiteParams). The Motorola Capacity Plus / Max
// vendor path reuses the struct over a raw 8-octet payload
// (ParseSystemInfoBroadcast).
type SystemInfoBroadcast struct {
	SystemID uint16
	RFSSID   uint8
	SiteID   uint8
	NetMask  uint8
}

// ParseGenSiteParams reads the camped-site identity from a C_BCAST
// Gen_Site_Params payload (sub-fields after the anncd_type octet).
func ParseGenSiteParams(p [8]byte) SystemInfoBroadcast {
	return SystemInfoBroadcast{
		SystemID: binary.BigEndian.Uint16(p[1:3]),
		RFSSID:   p[3],
		SiteID:   p[4],
		NetMask:  p[5],
	}
}

// ParseSystemInfoBroadcast reads a vendor (Motorola Cap+) system-info
// payload where the identity begins at octet 0. The SiteID octet doubles
// as the Capacity Plus rest-channel LCN pointer.
func ParseSystemInfoBroadcast(p [8]byte) SystemInfoBroadcast {
	return SystemInfoBroadcast{
		SystemID: binary.BigEndian.Uint16(p[0:2]),
		RFSSID:   p[2],
		SiteID:   p[3],
		NetMask:  p[4],
	}
}
