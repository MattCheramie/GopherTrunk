package nxdn

import (
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// Real-air NXDN frame decode for the voice (traffic) path (issue #1261).
//
// The rest of this package (frame.go, sync.go, lich.go, the CAC decoder)
// models an NXDN frame that does not exist on air: a 16-bit FSW 0xC55A, a
// 32-dibit SACCH, no scrambler. Its encoders and decoders share that
// invented layout, so their tests pass (the self-consistent trap again, as
// with the YSF FICH and the Motorola framing), while a real signal could
// never frame. The reporter's v1.2.9 "voice_frames=16" on a real NXDN96
// call was tolerance-1 matches of the invented 16-bit FSW in noise.
//
// The layout here is the one two independent decoders that are proven on
// air agree on — MMDVMHost (NXDNDefines.h, NXDNLICH.cpp, NXDNControl.cpp's
// SCRAMBLER table) and DSD-FME (nxdn_frame.c, nxdn_deperm.c,
// nxdn_voice.c) — and it is checked against the reporter's own captures
// (testdata/nxdn96_*, nxdn48_*):
//
//	FSW 10 dibits (20 bits, 0xCDF59) | LICH 8 | SACCH 30 | Info 144 = 192
//
// Every dibit after the FSW is scrambled by a 9-bit PN sequence (x^9+x^5+1,
// seed 0xE4, one bit per dibit) that inverts the symbol, i.e. flips the
// dibit's MSB in this package's mapping (01=+3, 00=+1, 10=-1, 11=-3).
// NXDN48 (6.25 kHz, 2400 sym/s) and NXDN96 (12.5 kHz, 4800 sym/s) share
// this frame; only the symbol rate differs.
//
// Only the voice path uses it so far. The control-channel (CAC) path still
// runs on the invented layout and needs the same rebuild against a capture.

const (
	// AirFSWDibits is the real frame sync word length: 20 bits.
	AirFSWDibits = 10
	// AirLICHDibits carries 8 LICH bits, one per dibit (the MSB).
	AirLICHDibits = 8
	// AirSACCHDibits is the 60-bit SACCH.
	AirSACCHDibits = 30
	// AirInfoDibits is the 288-bit information field: 4 × 72-bit AMBE+2
	// voice frames on a voice frame.
	AirInfoDibits = 144
	// AirFrameDibits is a whole frame, FSW included.
	AirFrameDibits = AirFSWDibits + AirLICHDibits + AirSACCHDibits + AirInfoDibits
	// airPostFSWDibits is what follows the FSW (the scrambled part).
	airPostFSWDibits = AirFrameDibits - AirFSWDibits

	airAMBEDibits = 36 // one 72-bit AMBE+2 frame
)

// AirFSW is the outbound frame sync word 0xCDF59 as dibits (MMDVMHost
// NXDN_FSW_BYTES {0xCD, 0xF5, 0x9_}).
var AirFSW = [AirFSWDibits]uint8{3, 0, 3, 1, 3, 3, 1, 1, 2, 1}

// airScramble[i] is 1 where post-FSW dibit i is inverted.
var airScramble = func() [airPostFSWDibits]uint8 {
	var t [airPostFSWDibits]uint8
	lfsr := uint16(0xE4)
	for i := range t {
		t[i] = uint8(lfsr & 1)
		fb := (lfsr ^ lfsr>>4) & 1
		lfsr = lfsr>>1 | fb<<8
	}
	return t
}()

// DescrambleAir removes the PN9 symbol scrambler from the dibits that
// follow an FSW, in place. Applying it twice is the identity, so it also
// scrambles.
func DescrambleAir(post []uint8) {
	for i := range post {
		if i >= len(airScramble) {
			return
		}
		post[i] ^= airScramble[i] << 1
	}
}

// LICH RF channel types (2 bits) on air.
const (
	AirRFCtRCCH  = 0 // control
	AirRFCtRTCH  = 1 // trunked traffic
	AirRFCtRDCH  = 2 // conventional / direct traffic
	AirRFCtRTCHC = 3 // composite control/traffic
)

// AirLICH is a decoded on-air LICH: RFCT(2) FCT(2) Option(2) Direction(1)
// Parity(1), MSB first.
type AirLICH struct {
	Raw       uint8
	RFCT      uint8
	FCT       uint8
	Option    uint8
	Direction uint8
	ParityOK  bool
}

// ParseAirLICH reads the LICH from the first 8 descrambled post-FSW dibits.
// Each dibit carries one LICH bit in its MSB. The parity bit is the XOR of
// the four RFCT/FCT bits (DSD-FME nxdn_frame.c).
func ParseAirLICH(d []uint8) AirLICH {
	var b uint8
	for i := 0; i < AirLICHDibits && i < len(d); i++ {
		b = b<<1 | (d[i]>>1)&1
	}
	par := (b>>7 ^ b>>6 ^ b>>5 ^ b>>4) & 1
	return AirLICH{
		Raw:       b,
		RFCT:      b >> 6 & 3,
		FCT:       b >> 4 & 3,
		Option:    b >> 2 & 3,
		Direction: b >> 1 & 1,
		ParityOK:  par == b&1,
	}
}

// IsTraffic reports whether the frame is on a traffic channel (trunked,
// conventional or composite), where the information field can carry voice.
func (l AirLICH) IsTraffic() bool {
	return l.RFCT == AirRFCtRTCH || l.RFCT == AirRFCtRDCH || l.RFCT == AirRFCtRTCHC
}

// VoiceHalves reports which halves of the information field carry voice
// (two AMBE frames each), from the LICH steal option: 3 = no steal (both),
// 2 = FACCH1 in the second half, 1 = FACCH1 in the first half, 0 = FACCH1
// in both (no voice). MMDVMHost NXDN_LICH_STEAL_*; DSD-FME voice=1/2/3.
// Voice exists only with an SACCH functional channel (FCT 0 or 2).
func (l AirLICH) VoiceHalves() (first, second bool) {
	if !l.IsTraffic() || (l.FCT != 0 && l.FCT != 2) {
		return false, false
	}
	switch l.Option {
	case 3:
		return true, true
	case 2:
		return true, false
	case 1:
		return false, true
	}
	return false, false
}

// VCHFrame is one decoded AMBE+2 voice frame.
type VCHFrame struct {
	// Payload is the 49-bit vocoder payload, one bit per byte (pack it
	// MSB-first to 7 bytes for the ambe2-dmr decoder).
	Payload []byte
	// Errors is the Golay-corrected bit count across C0/C1.
	Errors int
	// Slot is the frame's position in the information field, 0..3 (two
	// per half). A scrambled call's keystream advances one 49-bit block
	// per slot whether or not the slot decoded, so a descrambler needs
	// the position, not just the sequence of frames that decoded.
	Slot int
}

// DecodeAirVoice decodes the voice frames of one descrambled post-FSW
// frame (airPostFSWDibits dibits). It returns the LICH and the voice
// frames the LICH announces (none for a parity failure or a non-voice
// frame). The 72-bit NXDN AMBE+2 frame uses the DMR deinterleave and FEC
// verbatim: DSD-FME's nxdn_voice.c nW/nX/nY/nZ tables are DMR's
// rW/rX/rY/rZ, and MMDVMHost's NXDNAudio A/B/C tables are DMR's.
func DecodeAirVoice(post []uint8) (AirLICH, []VCHFrame) {
	if len(post) < airPostFSWDibits {
		return AirLICH{}, nil
	}
	lich := ParseAirLICH(post)
	if !lich.ParityOK {
		return lich, nil
	}
	first, second := lich.VoiceHalves()
	info := post[AirLICHDibits+AirSACCHDibits : AirLICHDibits+AirSACCHDibits+AirInfoDibits]
	var out []VCHFrame
	bits := make([]byte, 2*airAMBEDibits)
	for v := 0; v < 4; v++ {
		if (v < 2 && !first) || (v >= 2 && !second) {
			continue
		}
		for k, d := range info[v*airAMBEDibits : (v+1)*airAMBEDibits] {
			bits[2*k] = d >> 1 & 1
			bits[2*k+1] = d & 1
		}
		payload, errs, err := dmrvoice.DecodeAMBEFrame(bits)
		if err != nil {
			continue
		}
		out = append(out, VCHFrame{Payload: payload, Errors: errs, Slot: v})
	}
	return lich, out
}

// EncodeAirVoiceFrame builds one whole on-air voice frame (FSW included,
// scrambled) carrying four 49-bit vocoder payloads under an RDCH LICH with
// no FACCH steal. It is the test-side inverse of DecodeAirVoice.
func EncodeAirVoiceFrame(payloads [4][]byte) ([]uint8, error) {
	frame := make([]uint8, AirFrameDibits)
	copy(frame, AirFSW[:])
	post := frame[AirFSWDibits:]
	// RDCH, FCT 2 (SACCH superframe), option 3 (no steal), outbound... the
	// direction bit is 0 on the reporter's conventional captures.
	lich := uint8(AirRFCtRDCH<<6 | 2<<4 | 3<<2)
	lich |= (lich>>7 ^ lich>>6 ^ lich>>5 ^ lich>>4) & 1
	for i := 0; i < AirLICHDibits; i++ {
		// The LICH bit rides the MSB; MMDVMHost fills the LSB with 1.
		post[i] = (lich>>(7-i))&1<<1 | 1
	}
	info := post[AirLICHDibits+AirSACCHDibits:]
	for v, p := range payloads {
		bits, err := dmrvoice.EncodeAMBEFrame(p)
		if err != nil {
			return nil, err
		}
		for k := 0; k < airAMBEDibits; k++ {
			info[v*airAMBEDibits+k] = bits[2*k]<<1 | bits[2*k+1]
		}
	}
	DescrambleAir(post)
	return frame, nil
}
