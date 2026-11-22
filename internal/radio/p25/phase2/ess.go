package phase2

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// The Encryption Synchronization Signal (ESS): a Phase 2 call's Algorithm ID,
// Key ID and Message Indicator, carried inside the voice bursts themselves.
//
// MAC_PTT gives the same three at key-up (AirMAC.PTT), but a receiver that
// starts on a call after its key-up, or loses that one burst, never sees it.
// The ESS repeats every 360 ms for as long as the radio talks. Ported from
// OP25 (p25p2_tdma.cc handle_4V2V_ess); SDRTrunk's Voice4Timeslot /
// Voice2Timeslot give the same positions (ESS-B at body bit 148, ESS-A at
// 148 and 246).
//
// A talk spurt is a repeating run of five voice bursts on the call's logical
// channel: four VOICE-4 and one VOICE-2 (18 vocoder frames, 360 ms).
//
//	VOICE-4   body dibits 74..85: four hexbits of ESS-B. The four bursts
//	          carry its 16 hexbits in order: ALGID (8 bits), KID (16),
//	          MI (72).
//	VOICE-2   body dibits 74..121 and 123..158 (the DUID's third dibit sits
//	          at 122): the 28 hexbits of ESS-A, the Reed-Solomon parity over
//	          ESS-B.
//
// Together they are one word of the RS(63,35) code the MAC bursts use,
// shortened to (44,16): ESS-B at RS positions 19..34, ESS-A at 35..62. It
// corrects 14 wrong hexbits, or more missing ones: a VOICE-4 burst that was
// not received is four erasures.
//
// The run is not aligned to the superframe: it starts at whichever of the
// channel's five voice timeslots the radio keyed on. The VOICE-2 burst marks
// its end, so the four VOICE-4 bursts before a VOICE-2 are ESS-B's four
// parts, in order.

// essBodyStart is the body dibit where the ESS field starts in both voice
// burst types (OP25 &xored_burst[84], less its 10-dibit lead).
const essBodyStart = 74

// essVoiceSlots is how many voice timeslots a logical channel has per
// superframe (timeslots 0..9; 10 and 11 carry the SACCH).
const essVoiceSlots = 5

// essMaxAgeDibits bounds how old a VOICE-4 burst's ESS-B part may be when the
// VOICE-2 burst arrives. A channel's timeslot index recurs every superframe
// (12 packets), so a part from the run before is at least that old.
const essMaxAgeDibits = AirSlotsPerSuperframe*AirPacketDibits - AirPacketDibits

// ESSTracker assembles the ESS of one logical channel's voice bursts. Feed
// it that channel's voice bursts in order; it is not safe for concurrent
// use.
type ESSTracker struct {
	part [essVoiceSlots][4]byte // ESS-B hexbits by voice timeslot index
	at   [essVoiceSlots]int     // StartDibit of the burst that carried them
	have [essVoiceSlots]bool
}

// Reset forgets the parts collected so far.
func (t *ESSTracker) Reset() { *t = ESSTracker{} }

// essHexbits reads n hexbits from descrambled body dibits starting at off,
// skipping the DUID dibit at body position 122.
func essHexbits(body, mask []uint8, off, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		var h byte
		for k := 0; k < 3; k++ {
			if off == airDUIDBody[2] {
				off++
			}
			h = h<<2 | (body[off]^mask[off])&3
			off++
		}
		out[i] = h
	}
	return out
}

// Add takes one voice burst. A VOICE-4 burst's ESS-B part is stored; a
// VOICE-2 burst completes the word, and when it decodes Add returns the
// encryption sync, how many hexbits the decode corrected or filled in, and
// true. The sync names the superframe that follows (OP25 promotes it after
// the VOICE-2 burst), which matters to a decryptor and not to a caller that
// only wants the algorithm and key.
func (t *ESSTracker) Add(b AirBurst, masks *AirMasks) (EncryptionSync, int, bool) {
	if masks == nil || len(b.Packet) != AirPacketDibits || !b.DUID.IsVoice() {
		return EncryptionSync{}, 0, false
	}
	idx := b.Slot >> 1
	if b.Slot < 0 || idx >= essVoiceSlots {
		return EncryptionSync{}, 0, false // timeslots 10 / 11 carry no voice
	}
	body, mask := b.Body(), masks.Slot(b.Slot)
	if b.DUID == DUIDVoice4 {
		copy(t.part[idx][:], essHexbits(body, mask, essBodyStart, 4))
		t.at[idx], t.have[idx] = b.StartDibit, true
		return EncryptionSync{}, 0, false
	}

	// VOICE-2: the four bursts before it, in order, are at the next four
	// timeslot indexes (the run wraps around the five).
	const dataAt = framing.RS6335K - 16 // 19: RS index of ESS-B's first hexbit
	rx := make([]byte, framing.RS6335N)
	var erasures []int
	for k := 0; k < 4; k++ {
		s := (idx + 1 + k) % essVoiceSlots
		age := b.StartDibit - t.at[s]
		if t.have[s] && age > 0 && age <= essMaxAgeDibits {
			copy(rx[dataAt+4*k:], t.part[s][:])
		} else {
			for i := 0; i < 4; i++ {
				erasures = append(erasures, dataAt+4*k+i)
			}
		}
		t.have[s] = false
	}
	copy(rx[framing.RS6335K:], essHexbits(body, mask, essBodyStart, 28))
	cw, corrected, err := framing.DecodeRS63_35(rx, erasures)
	if err != nil {
		return EncryptionSync{}, 0, false
	}
	// The shortened positions are zero by construction: a decode that
	// "corrected" one is a miscorrection.
	for _, v := range cw[:dataAt] {
		if v != 0 {
			return EncryptionSync{}, 0, false
		}
	}
	var bits [96]byte
	for i := 0; i < 16; i++ {
		for j := 0; j < 6; j++ {
			bits[6*i+j] = cw[dataAt+i] >> uint(5-j) & 1
		}
	}
	var raw [12]byte
	for i, v := range bits {
		raw[i/8] |= v << uint(7-i%8)
	}
	es := EncryptionSync{AlgorithmID: raw[0], KeyID: uint16(raw[1])<<8 | uint16(raw[2])}
	copy(es.MessageIndicator[:], raw[3:])
	return es, corrected + len(erasures), true
}
