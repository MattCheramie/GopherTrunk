package phase2

import (
	"encoding/binary"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 Phase 2 MAC signalling in the real air format: how a SACCH, FACCH or
// LCCH burst framed by AirFramer becomes MAC PDU bytes and MAC messages.
// Ported from OP25 (p25p2_tdma.cc handle_acch_frame / process_mac_pdu /
// decode_mac_msg, crc16.h); the burst positions agree with SDRTrunk's
// FacchTimeslot / SacchTimeslot. There is no trellis code and no RS(24,16,9)
// on Phase 2 MAC: the burst body carries hexbits of a shortened, punctured
// RS(63,35) code whose information part is the MAC PDU plus a CRC.
//
//	FACCH (DUID 9 scrambled / 15 clear): body dibits 1-36, 38-68, 90-121,
//	       123-158 = 270 bits = hexbits 9..53 of the RS word (0..8 shortened,
//	       54..62 punctured); 144 PDU bits + CRC-12 = hexbits 9..34.
//	SACCH (DUID 3 / 12): body dibits 1-36, 38-121, 123-158 = 312 bits =
//	       hexbits 5..56 (0..4 shortened, 57..62 punctured); 168 PDU bits +
//	       CRC-12 = hexbits 5..34.
//	LCCH (DUID 13, the TDMA control channel's outbound, clear): the SACCH
//	       layout with 180 bits checked by CRC-16.
//
// A MAC PDU's first byte is its header: bits 7-5 the PDU type, bits 4-2 the
// offset to the first voice burst. MAC_PTT and MAC_END_PTT carry fixed
// fields (message indicator, algorithm, key, source, group); the other types
// carry a sequence of MAC messages, each one MACPDU (opcode + payload), whose
// lengths come from the opcode (OP25's mac_msg_len table).

// AirMACChannel is which logical signalling channel a burst carried.
type AirMACChannel uint8

const (
	AirFACCH AirMACChannel = iota + 1
	AirSACCH
	AirLCCH
)

func (c AirMACChannel) String() string {
	switch c {
	case AirFACCH:
		return "facch"
	case AirSACCH:
		return "sacch"
	case AirLCCH:
		return "lcch"
	}
	return "?"
}

// AirMACType is a MAC PDU's type, the header's top three bits.
type AirMACType uint8

const (
	AirMACSignal   AirMACType = 0
	AirMACPTT      AirMACType = 1
	AirMACEndPTT   AirMACType = 2
	AirMACIdle     AirMACType = 3
	AirMACActive   AirMACType = 4
	AirMACHangtime AirMACType = 6
)

func (t AirMACType) String() string {
	switch t {
	case AirMACSignal:
		return "signal"
	case AirMACPTT:
		return "ptt"
	case AirMACEndPTT:
		return "end-ptt"
	case AirMACIdle:
		return "idle"
	case AirMACActive:
		return "active"
	case AirMACHangtime:
		return "hangtime"
	}
	return "reserved"
}

// AirMAC is one CRC-valid MAC PDU decoded off the air.
type AirMAC struct {
	Channel AirMACChannel
	Type    AirMACType
	// Offset is the header's offset field (bursts to the first VOICE-4).
	Offset uint8
	// Bytes is the PDU, header first: 18 bytes (FACCH), 21 (SACCH) or 23
	// (LCCH, the last nibble padding).
	Bytes []byte
	// Corrected counts the RS symbols the decode changed.
	Corrected int
}

// airMACLayout is one signalling channel's burst geometry.
type airMACLayout struct {
	spans    [][2]int // body-dibit [start, end) ranges, in bit order
	first    int      // RS index of the first received hexbit
	present  int      // received hexbits
	pduBits  int      // PDU bits (before the CRC-12; the LCCH's include its CRC-16)
	crc16    bool
	outBytes int
}

var (
	airFACCH = airMACLayout{
		spans: [][2]int{{1, 37}, {38, 69}, {90, 122}, {123, 159}},
		first: 9, present: 45, pduBits: 144, outBytes: 18,
	}
	airSACCH = airMACLayout{
		spans: [][2]int{{1, 37}, {38, 122}, {123, 159}},
		first: 5, present: 52, pduBits: 168, outBytes: 21,
	}
	airLCCH = airMACLayout{
		spans: airSACCH.spans,
		first: 5, present: 52, pduBits: 180, crc16: true, outBytes: 23,
	}
)

// IsSignalling reports whether the burst carries a MAC PDU (SACCH, FACCH
// or LCCH, scrambled or clear).
func (d DUID) IsSignalling() bool {
	_, _, _, ok := airMACFor(d)
	return ok
}

func airMACFor(d DUID) (airMACLayout, AirMACChannel, bool, bool) {
	switch d {
	case DUIDFACCHScrambled:
		return airFACCH, AirFACCH, true, true
	case DUIDFACCHClear:
		return airFACCH, AirFACCH, false, true
	case DUIDSACCHScrambled:
		return airSACCH, AirSACCH, true, true
	case DUIDSACCHClear:
		return airSACCH, AirSACCH, false, true
	case DUIDLCCHScrambled:
		return airLCCH, AirLCCH, true, true
	case DUIDLCCHClear:
		return airLCCH, AirLCCH, false, true
	}
	return airMACLayout{}, 0, false, false
}

// DecodeAirMAC decodes a signalling burst. masks descrambles the scrambled
// DUIDs (SACCH-S, FACCH-S); with nil masks only clear bursts decode. ok is
// false for a non-signalling burst, an RS failure or a CRC mismatch.
func DecodeAirMAC(b AirBurst, masks *AirMasks) (AirMAC, bool) {
	lay, ch, scrambled, ok := airMACFor(b.DUID)
	if !ok || len(b.Packet) != AirPacketDibits || (scrambled && masks == nil) {
		return AirMAC{}, false
	}
	body := b.Body()
	var mask []uint8
	if scrambled {
		mask = masks.Slot(b.Slot)
	}
	var dib []uint8
	for _, sp := range lay.spans {
		for i := sp[0]; i < sp[1]; i++ {
			d := body[i]
			if mask != nil {
				d ^= mask[i]
			}
			dib = append(dib, d)
		}
	}
	bits := framing.DibitsToBits(dib)
	rx := make([]byte, framing.RS6335N)
	for k := 0; k < lay.present; k++ {
		var h byte
		for j := 0; j < 6; j++ {
			h = h<<1 | bits[6*k+j]
		}
		rx[lay.first+k] = h
	}
	// Shortened positions are known zeros; the punctured parity is erased.
	// Should a system not zero its shortened symbols, OP25's decode (both
	// erased) is the fallback; the CRC arbitrates.
	punct := make([]int, 0, framing.RS6335N)
	for i := lay.first + lay.present; i < framing.RS6335N; i++ {
		punct = append(punct, i)
	}
	if m, ok := decodeAirMACWord(rx, punct, lay, ch); ok {
		return m, true
	}
	both := append([]int(nil), punct...)
	for i := 0; i < lay.first; i++ {
		both = append(both, i)
	}
	return decodeAirMACWord(rx, both, lay, ch)
}

func decodeAirMACWord(rx []byte, erasures []int, lay airMACLayout, ch AirMACChannel) (AirMAC, bool) {
	cw, corrected, err := framing.DecodeRS63_35(rx, erasures)
	if err != nil {
		return AirMAC{}, false
	}
	checked := lay.pduBits
	if !lay.crc16 {
		checked += 12
	}
	bits := make([]byte, 0, checked+6)
	for k := lay.first; len(bits) < checked; k++ {
		for j := 5; j >= 0; j-- {
			bits = append(bits, cw[k]>>uint(j)&1)
		}
	}
	bits = bits[:checked]
	if lay.crc16 {
		if airCRC16(bits) != 0 {
			return AirMAC{}, false
		}
	} else if airCRC12(bits[:lay.pduBits]) != airBitsValue(bits[lay.pduBits:]) {
		return AirMAC{}, false
	}
	pdu := make([]byte, lay.outBytes)
	for i := 0; i < lay.pduBits; i++ {
		pdu[i/8] |= bits[i] << uint(7-i%8)
	}
	return AirMAC{
		Channel:   ch,
		Type:      AirMACType(pdu[0] >> 5),
		Offset:    pdu[0] >> 2 & 7,
		Bytes:     pdu,
		Corrected: corrected,
	}, true
}

func airBitsValue(bits []byte) uint16 {
	var v uint16
	for _, b := range bits {
		v = v<<1 | uint16(b)
	}
	return v
}

// airCRC12 is OP25's crc12: the message times x^12 modulo
// x^12+x^11+x^7+x^4+x^2+x+1, complemented.
func airCRC12(bits []byte) uint16 {
	poly := [13]byte{1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1}
	buf := make([]byte, len(bits)+12)
	copy(buf, bits)
	for i := range bits {
		if buf[i] != 0 {
			for j, p := range poly {
				buf[i+j] ^= p
			}
		}
	}
	return airBitsValue(buf[len(bits):]) ^ 0xFFF
}

// airCRC16 is OP25's crc16 over a whole LCCH (message and check bits);
// zero means valid.
func airCRC16(bits []byte) uint16 {
	const poly = 1<<12 | 1<<5 | 1
	var crc uint32
	for _, b := range bits {
		crc = (crc<<1 | uint32(b&1)) & 0x1FFFF
		if crc&0x10000 != 0 {
			crc = crc&0xFFFF ^ poly
		}
	}
	return uint16(crc^0xFFFF) & 0xFFFF
}

// MACPTT is the fixed content of a MAC_PTT PDU (OP25 handle_mac_ptt).
type MACPTT struct {
	Encryption EncryptionSync
	Source     uint32
	Group      uint16
}

// PTT returns a MAC_PTT PDU's fields.
func (m AirMAC) PTT() (MACPTT, bool) {
	if m.Type != AirMACPTT || len(m.Bytes) < 18 {
		return MACPTT{}, false
	}
	var p MACPTT
	copy(p.Encryption.MessageIndicator[:], m.Bytes[1:10])
	p.Encryption.AlgorithmID = m.Bytes[10]
	p.Encryption.KeyID = binary.BigEndian.Uint16(m.Bytes[11:13])
	p.Source = uint32(m.Bytes[13])<<16 | uint32(m.Bytes[14])<<8 | uint32(m.Bytes[15])
	p.Group = binary.BigEndian.Uint16(m.Bytes[16:18])
	return p, true
}

// MACEndPTT is the fixed content of a MAC_END_PTT PDU.
type MACEndPTT struct {
	ColorCode uint16
	Source    uint32
	Group     uint16
}

// EndPTT returns a MAC_END_PTT PDU's fields.
func (m AirMAC) EndPTT() (MACEndPTT, bool) {
	if m.Type != AirMACEndPTT || len(m.Bytes) < 18 {
		return MACEndPTT{}, false
	}
	return MACEndPTT{
		ColorCode: uint16(m.Bytes[1]&0x0F)<<8 | uint16(m.Bytes[2]),
		Source:    uint32(m.Bytes[13])<<16 | uint32(m.Bytes[14])<<8 | uint32(m.Bytes[15]),
		Group:     binary.BigEndian.Uint16(m.Bytes[16:18]),
	}, true
}

// airMACMsgLen is OP25's mac_msg_len: a MAC message's length in bytes by
// opcode (0 = unknown).
var airMACMsgLen = [256]uint8{
	0, 7, 8, 7, 0, 16, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 14, 15, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	5, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 7, 9, 0, 9, 8, 9, 0, 10, 10, 9, 0, 10, 0, 0, 0,
	0, 0, 0, 0, 9, 7, 0, 0, 10, 0, 7, 0, 10, 8, 14, 7,
	9, 9, 0, 0, 9, 0, 0, 9, 10, 0, 7, 10, 10, 7, 0, 9,
	9, 29, 9, 9, 9, 9, 10, 13, 9, 9, 9, 11, 9, 9, 0, 0,
	8, 0, 0, 7, 11, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	16, 0, 0, 11, 13, 11, 11, 11, 10, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	11, 0, 0, 8, 15, 12, 15, 32, 12, 12, 0, 27, 14, 29, 29, 32,
	0, 0, 0, 0, 0, 0, 9, 0, 14, 29, 11, 27, 14, 0, 40, 11,
	28, 0, 0, 14, 17, 14, 0, 0, 16, 8, 11, 0, 13, 19, 0, 0,
	0, 0, 16, 14, 0, 0, 12, 0, 22, 0, 11, 13, 11, 0, 15, 0,
}

// Messages splits a message-carrying MAC PDU into its MAC messages (OP25
// decode_mac_msg): Null (0x00) and Null-Avoid-Zero-Bias (0x08) fill is
// skipped, and a message of unknown length ends the walk. MAC_PTT and
// MAC_END_PTT carry no messages.
func (m AirMAC) Messages() []MACPDU {
	if m.Type == AirMACPTT || m.Type == AirMACEndPTT || len(m.Bytes) < 2 {
		return nil
	}
	n := len(m.Bytes)
	if m.Type == AirMACSignal && n > 18 {
		n = 18 // OP25: a MAC_SIGNAL's messages end at byte 18 (19-20 hold the NAC)
	}
	var out []MACPDU
	for ptr := 1; ptr < n; {
		op := m.Bytes[ptr]
		b1b2 := op >> 6
		var l int
		switch op {
		case 0x00:
			l = n - ptr
		case 0x08:
			if ptr+1 < n {
				l = int(m.Bytes[ptr+1] & 0x3F)
			}
		case 0x11:
			if ptr+1 < n {
				l = (int(m.Bytes[ptr+1]&3)+1)*2 + 2
			}
		case 0x12:
			if ptr+1 < n {
				l = (int(m.Bytes[ptr+1]&3)+1)*3 + 2
			}
		default:
			if b1b2 == 2 && ptr+2 < n {
				l = int(m.Bytes[ptr+2] & 0x3F)
			}
			if l == 0 {
				l = int(airMACMsgLen[op])
			}
		}
		if l == 0 || ptr+l > n {
			break
		}
		if op != 0x00 && op != 0x08 {
			if pdu, err := ParseMACPDU(m.Bytes[ptr : ptr+l]); err == nil {
				out = append(out, pdu)
			}
		}
		ptr += l
	}
	return out
}
