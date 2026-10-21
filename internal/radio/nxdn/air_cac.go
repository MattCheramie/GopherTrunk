package nxdn

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Real-air NXDN control channel (CAC) decode (issue #1261).
//
// The outbound RCCH frame is the airframe.go frame: FSW | LICH (8 dibits) |
// CAC (150 dibits = 300 bits) | 24 more dibits this decoder does not read,
// every post-FSW dibit PN9-scrambled. The 300 CAC bits are a 12x25 block
// interleave of a punctured (12 of every 14) rate-1/2 K=5 code — the same
// code as the SACCH/FACCH and YSF FICH, framing.ViterbiK5 — over
// 171 + 4 tail bits: SR (structure 2 + RAN 6) | 147-bit message | CRC-16.
// The CRC is shift-register CCITT (x^16+x^12+x^5+1) seeded 0xC3EE and run over
// all 171 bits; the remainder XOR 0xFFFF is 0 on a clean message.
//
// Layout facts are DSD-FME's (nxdn_deperm.c nxdn_cac / crc16cac, the
// PERM_12_25 table and cac_puncture in include/nxdn_const.h and
// nxdn_deperm.c) and the code is MMDVMHost's CNXDNConvolution; nothing is
// ported. They are checked against a real control channel: every one of the
// 214 frames in the #1261 reporter's 453.775 MHz capture passes the CRC, and
// the RAN the SR carries equals the site code mod 64 that the SITE_INFO in the
// same frames announces (two fields cross-checking each other).
//
// The package's older CAC model (cac.go, cac_channel.go: an 88-bit message
// behind an invented FSW and LICH coding) is not the air format and survives
// only for its own synthetic fixtures; ControlChannel.Process runs this one on
// real air.

const (
	// airCACDibits is the CAC field after the LICH: 300 bits.
	airCACDibits = 150
	airCACBits   = 2 * airCACDibits
	// airCACMotherBits is the depunctured rate-1/2 stream: 175 stages.
	airCACMotherBits = 350
	airCACStages     = airCACMotherBits / 2
	// airCACMessageBits is SR + message + CRC, ahead of the 4 tail bits.
	airCACMessageBits = 171
	// AirCACL3Bits is the layer-3 message the CAC carries after its SR.
	AirCACL3Bits = 147
)

// airCACPuncture marks the mother-code bits that are sent, per 14.
var airCACPuncture = [14]bool{true, true, true, false, true, true, true, true, true, true, true, false, true, true}

// AirCAC is one CRC-valid outbound CAC.
type AirCAC struct {
	// Structure and RAN are the SR field: 2 bits of message structure and
	// the 6-bit Radio Access Number.
	Structure uint8
	RAN       uint8
	// Message is the 147-bit layer-3 message, MSB-first: octet 0 is two
	// flag bits then the 6-bit message type (Type).
	Message [(AirCACL3Bits + 7) / 8]byte
	Type    uint8
	// Metric is the Viterbi path metric (corrected channel bits; 0 = clean).
	Metric int
}

// DecodeAirCAC decodes the CAC of one outbound frame. post is the descrambled
// post-FSW frame (DescrambleAir) of at least 8+150 dibits. It reports false
// unless the CAC passes its CRC.
func DecodeAirCAC(post []uint8) (AirCAC, bool) {
	if len(post) < AirLICHDibits+airCACDibits {
		return AirCAC{}, false
	}
	var bits [airCACBits]byte
	for i, d := range post[AirLICHDibits : AirLICHDibits+airCACDibits] {
		bits[2*i], bits[2*i+1] = d>>1&1, d&1
	}
	// Block deinterleave: the i-th bit on air is bit (i mod 25)*12 + i/25 of
	// the coded stream (written into 25 columns, read out by 12 rows).
	var coded [airCACBits]byte
	for i, b := range bits {
		coded[(i%25)*12+i/25] = b
	}
	mother := make([]byte, 0, airCACMotherBits)
	k := 0
	for p := 0; p < airCACMotherBits; p++ {
		if airCACPuncture[p%len(airCACPuncture)] {
			mother = append(mother, coded[k])
			k++
		} else {
			mother = append(mother, framing.DepunctureMark)
		}
	}
	out, metric := framing.ViterbiK5(mother, airCACStages)
	if airCACCRC(out[:airCACMessageBits]) != 0 {
		return AirCAC{}, false
	}
	c := AirCAC{Metric: metric}
	sr := framing.PackBitsMSB(out[:8])[0]
	c.Structure, c.RAN = sr>>6, sr&0x3F
	copy(c.Message[:], framing.PackBitsMSB(out[8:8+AirCACL3Bits]))
	c.Type = c.Message[0] & 0x3F
	return c, true
}

// airCACCRC runs the CAC's CRC over bits, CRC included; 0 means valid.
func airCACCRC(bits []byte) uint16 {
	crc := uint32(0xC3EE)
	for _, b := range bits {
		crc = (crc<<1 | uint32(b&1)) & 0x1FFFF
		if crc&0x10000 != 0 {
			crc = crc&0xFFFF ^ 0x1021
		}
	}
	return uint16(crc ^ 0xFFFF)
}

// RCCH message types carried on the real-air CAC (the 6-bit type in octet 0).
const (
	AirMsgVCallAssign    = 0x04 // VCALL_ASSGN
	AirMsgVCallAssignDup = 0x05 // VCALL_ASSGN_DUP
	AirMsgSiteInfo       = 0x18 // SITE_INFO
	AirMsgSrvInfo        = 0x19 // SRV_INFO
	AirMsgCCHInfo        = 0x1A // CCH_INFO
	AirMsgAdjSiteInfo    = 0x1B // ADJ_SITE_INFO
)

// msgBits reads n bits (n ≤ 32) of an MSB-first message from bit offset at.
func msgBits(m []byte, at, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		b := at + i
		v = v<<1 | uint32(m[b/8]>>(7-b%8)&1)
	}
	return v
}

// LocationID is the 24-bit NXDN location: a 2-bit category that sets how the
// rest splits into system and site codes (Global 10/12, Regional 14/8,
// Local 17/5 bits).
type LocationID uint32

// Split returns the category (0 global, 1 local, 2 regional) and the system
// and site codes.
func (l LocationID) Split() (category uint8, system uint32, site uint16) {
	v := uint32(l) & 0xFFFFFF
	category = uint8(v >> 22)
	switch category {
	case 0:
		return category, v >> 12 & 0x3FF, uint16(v & 0xFFF)
	case 2:
		return category, v >> 8 & 0x3FFF, uint16(v & 0xFF)
	case 1:
		return category, v >> 5 & 0x1FFFF, uint16(v & 0x1F)
	}
	return category, 0, 0
}

// AirSiteInfo is SITE_INFO: the control channel's own site identity.
type AirSiteInfo struct {
	Location LocationID
	// ChannelStructure, Service, Restriction and ChannelAccess are the raw
	// information words.
	ChannelStructure uint16
	Service          uint16
	Restriction      uint32
	ChannelAccess    uint32
	Version          uint8
	// AdjacentSites is the number of adjacent-site allocations announced.
	AdjacentSites uint8
	// ControlChannel1 / 2 are the site's control-channel numbers (0 = none).
	ControlChannel1, ControlChannel2 uint16
}

// ParseAirSiteInfo reads a SITE_INFO message (DSD-FME
// NXDN_decode_site_info's field positions).
func ParseAirSiteInfo(m []byte) AirSiteInfo {
	return AirSiteInfo{
		Location:         LocationID(msgBits(m, 8, 24)),
		ChannelStructure: uint16(msgBits(m, 32, 16)),
		Service:          uint16(msgBits(m, 48, 16)),
		Restriction:      msgBits(m, 64, 24),
		ChannelAccess:    msgBits(m, 88, 24),
		Version:          uint8(msgBits(m, 112, 8)),
		AdjacentSites:    uint8(msgBits(m, 120, 4)),
		ControlChannel1:  uint16(msgBits(m, 124, 10)),
		ControlChannel2:  uint16(msgBits(m, 134, 10)),
	}
}

// AirSrvInfo is SRV_INFO, the control channel's frequent service broadcast.
type AirSrvInfo struct {
	Location    LocationID
	Service     uint16
	Restriction uint32
}

// ParseAirSrvInfo reads a SRV_INFO message.
func ParseAirSrvInfo(m []byte) AirSrvInfo {
	return AirSrvInfo{
		Location:    LocationID(msgBits(m, 8, 24)),
		Service:     uint16(msgBits(m, 32, 16)),
		Restriction: msgBits(m, 48, 24),
	}
}

// AirVCallAssign is VCALL_ASSGN: a voice call assigned a traffic channel.
type AirVCallAssign struct {
	CCOption    uint8 // bit 7 emergency, 6 visitor, 5 priority paging
	CallType    uint8 // 3 bits; 1 = group call
	VoiceOption uint8 // 5 bits
	Source      uint16
	Destination uint16 // the talkgroup on a group call
	CallTimer   uint8
	Channel     uint16 // 10-bit traffic channel number
}

// ParseAirVCallAssign reads a VCALL_ASSGN (or VCALL_ASSGN_DUP) message in its
// channel-number form (DSD-FME NXDN_decode_VCALL_ASSGN, NXDN TS 1-E
// 6.4.1.23). Not yet seen on air by this decoder: the field positions are
// pinned to the reference, the capture that confirms them is still needed.
func ParseAirVCallAssign(m []byte) AirVCallAssign {
	return AirVCallAssign{
		CCOption:    uint8(msgBits(m, 8, 8)),
		CallType:    uint8(msgBits(m, 16, 3)),
		VoiceOption: uint8(msgBits(m, 19, 5)),
		Source:      uint16(msgBits(m, 24, 16)),
		Destination: uint16(msgBits(m, 40, 16)),
		CallTimer:   uint8(msgBits(m, 56, 6)),
		Channel:     uint16(msgBits(m, 62, 10)),
	}
}

// EncodeAirControlFrame builds a whole outbound RCCH frame on the real air
// format: FSW, an outbound control-channel LICH (CAC, FCT 0), the CAC carrying
// sr and the 147-bit message msg, and zero fill, all post-FSW dibits
// scrambled. It is the inverse of DecodeAirCAC, for test fixtures and the
// Signal Lab synthesiser; real air's 24 trailing dibits are not modelled.
func EncodeAirControlFrame(ran uint8, msg []byte) []uint8 {
	frame := make([]uint8, AirFrameDibits)
	copy(frame, AirFSW[:])
	post := frame[AirFSWDibits:]
	lich := uint8(AirRFCtRCCH<<6 | 1<<1) // RCCH, FCT 0, option 0, outbound
	lich |= (lich>>7 ^ lich>>6 ^ lich>>5 ^ lich>>4) & 1
	for i := 0; i < AirLICHDibits; i++ {
		// The LICH bit rides the MSB; MMDVMHost fills the LSB with 1.
		post[i] = (lich>>(7-i))&1<<1 | 1
	}
	copy(post[AirLICHDibits:], encodeAirCAC(ran&0x3F, msg))
	DescrambleAir(post)
	return frame
}

// encodeAirCAC returns the 150 CAC dibits for SR (structure 0, ran) and msg.
func encodeAirCAC(sr byte, msg []byte) []uint8 {
	bits := make([]byte, 0, airCACStages)
	for i := 7; i >= 0; i-- {
		bits = append(bits, sr>>uint(i)&1)
	}
	for i := 0; i < AirCACL3Bits; i++ {
		var b byte
		if i/8 < len(msg) {
			b = msg[i/8] >> (7 - uint(i%8)) & 1
		}
		bits = append(bits, b)
	}
	// The register sees the CRC's own bits last and they XOR straight into
	// it, so the CRC that zeroes the check is the check of the message
	// followed by 16 zero bits.
	crc := airCACCRC(append(append([]byte(nil), bits...), make([]byte, 16)...))
	for i := 15; i >= 0; i-- {
		bits = append(bits, byte(crc>>uint(i)&1))
	}
	bits = append(bits, 0, 0, 0, 0) // tail
	coded := make([]byte, 0, airCACMotherBits)
	var d1, d2, d3, d4 byte
	for _, d := range bits {
		coded = append(coded, d^d3^d4, d^d1^d2^d4)
		d4, d3, d2, d1 = d3, d2, d1, d
	}
	sent := make([]byte, 0, airCACBits)
	for i, b := range coded {
		if airCACPuncture[i%len(airCACPuncture)] {
			sent = append(sent, b)
		}
	}
	out := make([]uint8, airCACDibits)
	for i := 0; i < airCACBits; i += 2 {
		out[i/2] = sent[(i%25)*12+i/25]<<1 | sent[((i+1)%25)*12+(i+1)/25]
	}
	return out
}
