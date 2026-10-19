package phase2

import (
	"fmt"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// EncodeAirPacket builds one 180-dibit outbound timeslot packet in the air
// format AirFramer decodes: the ISCH timeslot slot carries (an I-ISCH with
// superframe counter count before 0,1, 4,5, 8,9, the S-ISCH otherwise),
// then the body with duid's codeword. A voice DUID takes len(voice) ==
// duid.VoiceFrames() 7-byte AMBE+2 frames, FEC-encoded and scrambled with
// masks; every other body dibit is scrambled zero. For synthesized test
// streams; the layout is the one the airframe.go references define.
func EncodeAirPacket(slot int, duid DUID, voice [][]byte, masks *AirMasks, count uint8) ([]uint8, error) {
	if slot < 0 || slot >= AirSlotsPerSuperframe {
		return nil, fmt.Errorf("p25/phase2: timeslot %d out of range", slot)
	}
	if len(voice) != duid.VoiceFrames() {
		return nil, fmt.Errorf("p25/phase2: %v burst takes %d voice frames, got %d", duid, duid.VoiceFrames(), len(voice))
	}
	pkt := make([]uint8, AirPacketDibits)
	var isch uint64
	switch slot % 4 {
	case 0, 1:
		isch = EncodeIISCH(IISCH{Channel: uint8(slot % 4), Location: uint8(slot / 4), Count: count})
	default:
		isch = SISCHSync
	}
	for i := 0; i < AirISCHDibits; i++ {
		pkt[i] = uint8(isch>>(2*(AirISCHDibits-1-i))) & 3
	}
	body := pkt[AirISCHDibits:]
	if masks != nil {
		copy(body, masks.Slot(slot))
	}
	for i, f := range voice {
		info := framing.UnpackBitsMSB(f, voiceInfoBits)
		onAir, err := dmrvoice.EncodeAMBEFrame(info)
		if err != nil {
			return nil, err
		}
		cw := framing.BitsToDibits(onAir)
		off := airVoiceFrameBody[i]
		for k, d := range cw {
			body[off+k] ^= d
		}
	}
	cw := EncodeDUID(duid)
	for i, off := range airDUIDBody {
		body[off] = cw >> (2 * (3 - i)) & 3
	}
	return pkt, nil
}

// EncodeAirMACPacket builds one 180-dibit timeslot packet carrying a MAC PDU
// on a signalling DUID (SACCH / FACCH, scrambled or clear, or the LCCH) in
// the air format DecodeAirMAC reads: the PDU (header byte first; 18 bytes
// for FACCH, 21 for SACCH, 21 for the LCCH, whose last 4 bits are dropped
// to make room for its CRC-16) gets its CRC, the RS(63,35) parity, and is
// laid into the body, scrambled with masks for the scrambled DUIDs.
func EncodeAirMACPacket(slot int, duid DUID, pdu []byte, masks *AirMasks, count uint8) ([]uint8, error) {
	lay, _, scrambled, ok := airMACFor(duid)
	if !ok {
		return nil, fmt.Errorf("p25/phase2: %v is not a signalling DUID", duid)
	}
	if scrambled && masks == nil {
		return nil, fmt.Errorf("p25/phase2: %v needs scrambling masks", duid)
	}
	pkt, err := EncodeAirPacket(slot, DUIDSACCHClear, nil, nil, count)
	if err != nil {
		return nil, err
	}
	dataBits := lay.pduBits
	if lay.crc16 {
		dataBits -= 16
	}
	bits := framing.UnpackBitsMSB(pdu, dataBits)
	if lay.crc16 {
		crc := airCRC16(append(append([]byte(nil), bits...), make([]byte, 16)...))
		for i := 15; i >= 0; i-- {
			bits = append(bits, byte(crc>>uint(i)&1))
		}
	} else {
		crc := airCRC12(bits)
		for i := 11; i >= 0; i-- {
			bits = append(bits, byte(crc>>uint(i)&1))
		}
	}
	data := make([]byte, framing.RS6335K)
	for k := 0; 6*k < len(bits); k++ {
		var h byte
		for j := 0; j < 6; j++ {
			h = h<<1 | bits[6*k+j]
		}
		data[lay.first+k] = h
	}
	cw, err := framing.EncodeRS63_35(data)
	if err != nil {
		return nil, err
	}
	var txBits []byte
	for _, h := range cw[lay.first : lay.first+lay.present] {
		for j := 5; j >= 0; j-- {
			txBits = append(txBits, h>>uint(j)&1)
		}
	}
	dib := framing.BitsToDibits(txBits)
	body := pkt[AirISCHDibits:]
	k := 0
	for _, sp := range lay.spans {
		for i := sp[0]; i < sp[1]; i++ {
			d := dib[k]
			if scrambled {
				d ^= masks.Slot(slot)[i]
			}
			body[i] = d
			k++
		}
	}
	dcw := EncodeDUID(duid)
	for i, off := range airDUIDBody {
		body[off] = dcw >> (2 * (3 - i)) & 3
	}
	return pkt, nil
}

// AirMACPDUBytes builds a MAC PDU of size bytes (18 for FACCH, 21 for SACCH
// and the LCCH) with the given type and offset in its header, the messages
// packed after it and Null (0x00) fill to the end. Each message must have
// the length OP25's mac_msg_len (or, for a manufacturer message, its own
// length byte) gives its opcode, or Messages will not split it back out.
// For synthesized test streams.
func AirMACPDUBytes(t AirMACType, offset uint8, size int, msgs ...[]byte) ([]byte, error) {
	pdu := make([]byte, size)
	pdu[0] = byte(t)<<5 | (offset&7)<<2
	n := 1
	for _, m := range msgs {
		if n+len(m) > size {
			return nil, fmt.Errorf("p25/phase2: MAC messages overflow a %d-byte PDU", size)
		}
		n += copy(pdu[n:], m)
	}
	return pdu, nil
}

// AirMACPTTBytes builds an 18-byte MAC_PTT PDU (the layout PTT reads).
func AirMACPTTBytes(p MACPTT) []byte {
	pdu := make([]byte, 18)
	pdu[0] = byte(AirMACPTT) << 5
	copy(pdu[1:10], p.Encryption.MessageIndicator[:])
	pdu[10] = p.Encryption.AlgorithmID
	pdu[11] = byte(p.Encryption.KeyID >> 8)
	pdu[12] = byte(p.Encryption.KeyID)
	pdu[13] = byte(p.Source >> 16)
	pdu[14] = byte(p.Source >> 8)
	pdu[15] = byte(p.Source)
	pdu[16] = byte(p.Group >> 8)
	pdu[17] = byte(p.Group)
	return pdu
}

// AirSlotContent is one timeslot of a synthesized air stream: a voice DUID
// with its frames, a signalling DUID with its MAC PDU (MAC non-nil), or any
// other DUID as a scrambled-zero filler body.
type AirSlotContent struct {
	DUID  DUID
	Voice [][]byte
	MAC   []byte
}

// EncodeAirSuperframe builds one 2160-dibit superframe from twelve
// timeslots' content (EncodeAirPacket / EncodeAirMACPacket per slot).
func EncodeAirSuperframe(slots [AirSlotsPerSuperframe]AirSlotContent, masks *AirMasks, count uint8) ([]uint8, error) {
	out := make([]uint8, 0, AirSlotsPerSuperframe*AirPacketDibits)
	for i, s := range slots {
		var pkt []uint8
		var err error
		if s.MAC != nil {
			pkt, err = EncodeAirMACPacket(i, s.DUID, s.MAC, masks, count)
		} else {
			pkt, err = EncodeAirPacket(i, s.DUID, s.Voice, masks, count)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, pkt...)
	}
	return out, nil
}
