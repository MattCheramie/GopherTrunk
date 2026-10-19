package sigfollow

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// testSeed scrambles the synthetic air streams; the grants carry it, as a
// live grant carries the seed the control channel derives from the site
// identity.
var testSeed = framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x161)

// voiceSlots is a superframe of VOICE-4 bursts with silent frames.
func voiceSlots() [p25p2.AirSlotsPerSuperframe]p25p2.AirSlotContent {
	var s [p25p2.AirSlotsPerSuperframe]p25p2.AirSlotContent
	for i := range s {
		frames := make([][]byte, 4)
		for j := range frames {
			frames[j] = make([]byte, p25p2.VoiceFrameBytes)
		}
		s[i] = p25p2.AirSlotContent{DUID: p25p2.DUIDVoice4, Voice: frames}
	}
	return s
}

// airStreamDibits encodes superframes in the air format behind a short
// lead-in that never matches the sync, plus one trailing voice superframe
// so the framer emits every MAC burst.
func airStreamDibits(t testing.TB, masks *p25p2.AirMasks, sfs ...[p25p2.AirSlotsPerSuperframe]p25p2.AirSlotContent) []uint8 {
	t.Helper()
	dibits := make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	sfs = append(sfs, voiceSlots())
	for n, sf := range sfs {
		d, err := p25p2.EncodeAirSuperframe(sf, masks, uint8(n&3))
		if err != nil {
			t.Fatal(err)
		}
		dibits = append(dibits, d...)
	}
	return dibits
}

// airBursts frames an air stream back into bursts, as the live receiver's
// dibit sink hands them to the dispatcher.
func airBursts(t testing.TB, masks *p25p2.AirMasks, sfs ...[p25p2.AirSlotsPerSuperframe]p25p2.AirSlotContent) []p25p2.AirBurst {
	t.Helper()
	bursts := p25p2.NewAirFramer().Process(airStreamDibits(t, masks, sfs...), 0)
	if len(bursts) == 0 {
		t.Fatal("air framer found no bursts")
	}
	return bursts
}

// macPDU packs messages into a MAC PDU of the channel's size.
func macPDU(t testing.TB, typ p25p2.AirMACType, duid p25p2.DUID, msgs ...[]byte) []byte {
	t.Helper()
	size := 21
	if duid == p25p2.DUIDFACCHScrambled || duid == p25p2.DUIDFACCHClear {
		size = 18
	}
	pdu, err := p25p2.AirMACPDUBytes(typ, 0, size, msgs...)
	if err != nil {
		t.Fatal(err)
	}
	return pdu
}
