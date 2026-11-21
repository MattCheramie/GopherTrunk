package dpmr

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TCH (traffic channel) voice payload of a dPMR payload frame.
//
// An 80 ms payload frame carries four 72-bit TCH blocks after its sync
// word (or channel code) and control channel; see airframe.go for the
// frame. In a voice transmission each block is one 20 ms vocoder frame
// with its own FEC (voice_ambe.go). The standard does not scramble or
// interleave the TCH of a voice frame beyond that.

const (
	// TCHSubframeDibits is one vocoder frame on the air: 72 bits.
	TCHSubframeDibits = 36
	// TCHFramesPerBurst is the number of vocoder frames in one 80 ms
	// payload frame.
	TCHFramesPerBurst = 4
	// TCHFieldDibits is the whole TCH payload of a frame: 4 x 36 dibits
	// (288 bits).
	TCHFieldDibits = TCHFramesPerBurst * TCHSubframeDibits
)

// TCHFrame is one decoded AMBE+2 voice frame carved from a TCH payload.
type TCHFrame struct {
	// Payload is the 49-bit vocoder payload, one bit per byte, in the
	// order DecodeTCHFrame emits (C0:12 + C1:12 + C2:11 + C3:14). Pack
	// it MSB-first to 7 bytes (framing.PackBitsMSB) for the ambe2
	// decoder.
	Payload []byte
	// Errors is the Golay-corrected bit count across the C0/C1
	// sub-vectors — a decode-quality signal to feed the vocoder's
	// error-aware smoothing.
	Errors int
}

// ExtractTCHFrames decodes the 144-dibit (288-bit) TCH payload of a
// dPMR voice frame into its TCHFramesPerBurst AMBE+2 voice frames.
// Returns nil if tchDibits is not exactly TCHFieldDibits long.
func ExtractTCHFrames(tchDibits []uint8) []TCHFrame {
	if len(tchDibits) != TCHFieldDibits {
		return nil
	}
	bitsAll := framing.DibitsToBits(tchDibits) // 288 bits
	out := make([]TCHFrame, 0, TCHFramesPerBurst)
	for i := 0; i < TCHFramesPerBurst; i++ {
		seg := bitsAll[i*dpmrAMBEOnAirBits : (i+1)*dpmrAMBEOnAirBits]
		payload, errs, err := DecodeTCHFrame(seg)
		if err != nil {
			continue
		}
		out = append(out, TCHFrame{Payload: payload, Errors: errs})
	}
	return out
}
