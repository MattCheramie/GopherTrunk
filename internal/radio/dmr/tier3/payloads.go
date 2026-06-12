package tier3

import "encoding/binary"

// TVGrant (CSBKO 0x31) is the TalkGroup Voice Channel Grant per ETSI
// TS 102 361-4 §7.2. The channel-grant CSBKs share a common content
// layout that leads with the Logical Physical Channel Number, NOT with
// service options. Within the 8-octet payload (bits 16-79 of the CSBK,
// indexed here as payload bits 0-63):
//
//	payload bits 0-11  : LPCN (Logical Physical Channel Number, 12-bit)
//	payload bit  12    : Timeslot (0 = TS1, 1 = TS2) — the 13th bit of
//	                     the combined LPCN+TS "logical slot number"
//	payload bits 13-15 : status/flags (unused here)
//	octets 2-4         : Target/Group address (24-bit)
//	octets 5-7         : Source/subscriber address (24-bit)
//
// Cross-checked against the dsd-neo reference decoder (dmr_csbk_parse.c):
// LPCN at bit 16, target at bit 32, source at bit 56 of the full CSBK.
// The earlier layout read "LCN" from octet 7 — which is actually the low
// byte of the 24-bit source address — so the decoded LCN changed with
// every transmitting radio (issue #639).
//
// The LCN feeds a per-system band-plan resolver to recover the
// downlink frequency the engine retunes a Voice device to.
type TVGrant struct {
	GroupAddress uint32 // 24-bit
	SourceID     uint32 // 24-bit
	LCN          uint16 // 12-bit logical physical channel number
	Timeslot     uint8  // 0 = TS1, 1 = TS2
}

func ParseTVGrant(p [8]byte) TVGrant {
	return TVGrant{
		LCN:          uint16(p[0])<<4 | uint16(p[1])>>4,
		Timeslot:     (p[1] >> 3) & 0x01,
		GroupAddress: uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]),
		SourceID:     uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// PVGrant (CSBKO 0x30) is the Private Voice Channel Grant. Layout
// matches TVGrant but the destination address is a subscriber rather
// than a talkgroup. The same LPCN + Timeslot encoding applies.
type PVGrant struct {
	DestinationID uint32 // 24-bit
	SourceID      uint32 // 24-bit
	LCN           uint16
	Timeslot      uint8
}

func ParsePVGrant(p [8]byte) PVGrant {
	return PVGrant{
		LCN:           uint16(p[0])<<4 | uint16(p[1])>>4,
		Timeslot:      (p[1] >> 3) & 0x01,
		DestinationID: uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]),
		SourceID:      uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
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

// DataGrant is the parsed content of a data-channel grant CSBK —
// PD_GRANT (CSBKO 0x33, private data) or TD_GRANT (CSBKO 0x34, talkgroup
// data). The on-air content matches the voice channel-grant layout
// (LPCN + Timeslot + 24-bit target + 24-bit source); only the granted
// service (data vs voice) differs, so the parser is shared with TVGrant's
// bit layout and a Private discriminator. Unlike a voice grant a data
// grant does NOT cause the engine to retune a Voice device — packet data
// is not vocoded — so the control channel records the LCN observation for
// the band-plan learner but does not follow the call (see observeDataGrant).
type DataGrant struct {
	TargetID uint32 // 24-bit: talkgroup (TD) or subscriber (PD)
	SourceID uint32 // 24-bit subscriber
	LCN      uint16 // 12-bit logical physical channel number
	Timeslot uint8  // 0 = TS1, 1 = TS2
	Private  bool   // true = PD_GRANT (private), false = TD_GRANT (talkgroup)
}

// ParseDataGrant decodes a PD_GRANT / TD_GRANT payload. The bit layout is
// identical to ParseTVGrant; private selects PD (true) vs TD (false).
func ParseDataGrant(p [8]byte, private bool) DataGrant {
	return DataGrant{
		LCN:      uint16(p[0])<<4 | uint16(p[1])>>4,
		Timeslot: (p[1] >> 3) & 0x01,
		TargetID: uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]),
		SourceID: uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
		Private:  private,
	}
}

// Inbound / reverse-channel and acknowledgement CSBKs.
//
// C_RAND (CSBKO 0x1F) is the random-access service request a mobile sends
// to the TSCC on the reverse (uplink) channel — the inbound counterpart of
// the outbound C_AHOY poll (CSBKO 0x1C). C_ACKVIT / C_ACKD / C_ACKU
// (0x1E / 0x20 / 0x21) and the negative-ack (0x26) are the acknowledged-
// response family. None of these key up voice and GopherTrunk does not act
// on them; they are parsed for control-plane visibility (debug logging)
// the way a reference decoder surfaces them.
//
// Addressing follows the grant-family convention validated against the
// dsd-neo decoder: a 24-bit target at octets 2-4 and a 24-bit source at
// octets 5-7, with a leading Service octet (payload[0]) carrying the
// opcode-specific service kind. Pinned against the only real off-air
// vectors available — the C_AHOY and C_ACKD bursts in
// csbk_realvectors_test.go (C_AHOY: target 0x0002D0, source 0xFFFECA;
// C_ACKD: target 0x000345, source 0xFFFEC6). The finer per-opcode service
// sub-fields are left unparsed pending dedicated reverse-channel captures;
// no field here is invented beyond what those two vectors confirm.

// csbkTarget / csbkSource read the grant-family 24-bit target (octets 2-4)
// and source (octets 5-7) addresses shared by the addressed CSBKs.
func csbkTarget(p [8]byte) uint32 { return uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]) }
func csbkSource(p [8]byte) uint32 { return uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]) }

// RandomAccess is a parsed C_RAND (CSBKO 0x1F) inbound service request.
type RandomAccess struct {
	Service  uint8  // payload[0]: opcode-specific service kind
	TargetID uint32 // 24-bit
	SourceID uint32 // 24-bit requesting subscriber
}

// ParseRandomAccess decodes a C_RAND payload.
func ParseRandomAccess(p [8]byte) RandomAccess {
	return RandomAccess{Service: p[0], TargetID: csbkTarget(p), SourceID: csbkSource(p)}
}

// Ahoy is a parsed C_AHOY (CSBKO 0x1C) outbound poll / service request.
type Ahoy struct {
	Service  uint8
	TargetID uint32
	SourceID uint32
}

// ParseAhoy decodes a C_AHOY payload.
func ParseAhoy(p [8]byte) Ahoy {
	return Ahoy{Service: p[0], TargetID: csbkTarget(p), SourceID: csbkSource(p)}
}

// Ack is a parsed acknowledged-response CSBK (C_ACKVIT / C_ACKD / C_ACKU)
// or a negative acknowledgement (CSBKO 0x26). Opcode retains which variant
// it was so a single handler can log it.
type Ack struct {
	Opcode   CSBKOpcode
	Service  uint8
	TargetID uint32
	SourceID uint32
}

// ParseAck decodes an acknowledged-response / NACK payload, tagging it with
// the originating opcode.
func ParseAck(op CSBKOpcode, p [8]byte) Ack {
	return Ack{Opcode: op, Service: p[0], TargetID: csbkTarget(p), SourceID: csbkSource(p)}
}
