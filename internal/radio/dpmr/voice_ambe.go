package dpmr

import (
	"fmt"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// The 72-bit vocoder frame of a dPMR voice TCH block.
//
// dPMR voice is AMBE+2 at 3600 bit/s: 49 vocoder bits and 23 bits of FEC
// in 72 bits every 20 ms. ETSI TS 102 658 only says the TCH carries
// "forward error corrected vocoder data" and leaves the layout to the
// vocoder. It is the layout DMR and NXDN use, bit for bit: the 36 dibits
// spread over the C0..C3 sub-vectors by the same schedule, Golay (23,12)
// on C0 and C1, C1 scrambled by a sequence seeded from C0. The two
// decoders that receive dPMR voice on the air, dsdcc (SDRangel) and
// DSD-FME, both carry that schedule for dPMR, element for element the
// one in internal/radio/dmr/voice. So this file hands the frame to that
// package, as the NXDN air format does.
//
// History: until #1374 this file ran the Golay step with a different
// (23,12) code, and until the air-format rebuild it split the 72 bits into
// C0|C1|C2|C3 in order, a placeholder that was never the air layout. With
// either, a clean frame decoded to the wrong vocoder parameters.
// TestTCHFrameIsTheAMBESilenceFrame pins the layout to the frame a
// vocoder puts on the air for silence.

const (
	dpmrAMBEOnAirBits = 72 // on-air bits per AMBE+2 TCH voice frame
	dpmrAMBEInfoBits  = 49 // vocoder-payload bits per frame (ambe_d)
)

// DecodeTCHFrame decodes one 72-bit on-air AMBE+2 TCH voice frame (72
// bits, one bit per byte, first on the air first) into the 49-bit vocoder
// payload (one bit per byte). It returns the payload and the number of
// Golay errors corrected across C0 and C1. Pack the payload MSB-first to
// 7 bytes (framing.PackBitsMSB) and hand it to the ambe2 decoder.
func DecodeTCHFrame(frame []byte) ([]byte, int, error) {
	if len(frame) != dpmrAMBEOnAirBits {
		return nil, 0, fmt.Errorf("dpmr: TCH AMBE frame must be %d bits, got %d", dpmrAMBEOnAirBits, len(frame))
	}
	return dmrvoice.DecodeAMBEFrame(frame)
}

// EncodeTCHFrame is the inverse of DecodeTCHFrame: it wraps a 49-bit
// vocoder payload into a 72-bit on-air AMBE+2 TCH frame.
func EncodeTCHFrame(info []byte) ([]byte, error) {
	if len(info) != dpmrAMBEInfoBits {
		return nil, fmt.Errorf("dpmr: TCH AMBE payload must be %d bits, got %d", dpmrAMBEInfoBits, len(info))
	}
	return dmrvoice.EncodeAMBEFrame(info)
}
