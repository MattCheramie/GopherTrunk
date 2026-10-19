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
