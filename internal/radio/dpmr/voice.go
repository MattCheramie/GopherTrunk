package dpmr

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TCH (traffic voice channel) decode for dPMR Mode 3.
//
// A dPMR frame is 80 ms / 384 bits / 192 dibits at 2400 sym/s
// (TS 102 658 §4.4.2): a 48-bit frame sync (FS1 at the superframe
// start, FS2 mid-superframe), a 48-bit CCH (channel-control) field,
// and 288 bits of TCH payload carrying 4 AMBE+2 voice frames of 72
// on-air bits each. Each 20 ms AMBE+2 frame decodes (voice_ambe.go)
// to a 49-bit vocoder payload; four of them span the 80 ms frame.
//
// ⚠️ UNVERIFIED ON AIR — see voice_ambe.go. The 4×72 carve, the CCH
// width, and the AMBE deinterleave are transcribed from spec and
// validated only by the synthetic round-trip; a real dPMR voice
// capture is needed to confirm them (and the codebook variant).

const (
	// CCHDibits is the channel-control field between the frame sync
	// and the TCH payload: 48 bits / 24 dibits.
	CCHDibits = 24
	// TCHSubframeDibits is one AMBE+2 voice frame on the air: 72
	// bits / 36 dibits.
	TCHSubframeDibits = 36
	// TCHFramesPerBurst is the number of AMBE+2 voice frames one
	// 80 ms dPMR frame carries.
	TCHFramesPerBurst = 4
	// TCHFieldDibits is the whole TCH payload: 4 × 36 = 144 dibits
	// (288 bits).
	TCHFieldDibits = TCHFramesPerBurst * TCHSubframeDibits
	// postSyncDibitsTraffic is the count of dibits collected after
	// the 24-dibit FS1/FS2 match for a full voice frame: 24 CCH +
	// 144 TCH = 168 (the whole 192-dibit frame minus the sync).
	postSyncDibitsTraffic = CCHDibits + TCHFieldDibits
)

// TCHFrame is one decoded AMBE+2 voice frame carved from a TCH burst.
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
// Returns nil if tchDibits is not exactly TCHFieldDibits long. Frames
// whose AMBE FEC hard-fails are skipped (so a partially corrupt burst
// still yields its good frames).
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
