package phase2

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Literal I-ISCH codewords from OP25's p25p2_isch table (value -> 40-bit
// codeword). iischCodewords is built from SDRTrunk's generator matrix, so
// this pins one reference against the other.
func TestIISCHCodewordsMatchOP25Table(t *testing.T) {
	for v, want := range map[int]uint64{
		0:   0x184229d461,
		1:   0x18761451f6,
		7:   0x18b37c5ac9,
		8:   0x1146a44f13,
		16:  0x1a4a2e239e,
		53:  0x16e36ed6f6,
		125: 0x0fe8a85adc,
		127: 0x0fb063f092,
	} {
		if got := iischCodewords[v]; got != want {
			t.Errorf("I-ISCH %d = %#x, OP25 %#x", v, got, want)
		}
	}
}

// OP25's p25p2_sync expects an I-ISCH naming timeslot loc*4+chn before
// 0,1, 4,5, 8,9 and decodes chn/loc from the top bits of the value.
func TestDecodeIISCHNamesTimeslot(t *testing.T) {
	for _, slot := range []int{0, 1, 4, 5, 8, 9} {
		cw := EncodeIISCH(IISCH{Channel: uint8(slot % 4), Location: uint8(slot / 4), Count: 2})
		// Seven bit errors are still corrected; the code is (40,9,16).
		cw ^= 0x8000000000 | 0x0100000000 | 0x10000000 | 0x200000 | 0x4000 | 0x80 | 0x1
		got, errs, ok := DecodeIISCH(cw)
		if !ok || got.Slot() != slot || got.Count != 2 || errs != 7 {
			t.Errorf("slot %d: got %+v (slot %d) errs=%d ok=%v", slot, got, got.Slot(), errs, ok)
		}
	}
	if _, _, ok := DecodeIISCH(SISCHSync); ok {
		t.Error("the S-ISCH sync word decoded as an I-ISCH")
	}
}

// OP25 which_slot[] and SDRTrunk's final-fragment C/D swap: the last pair
// of timeslots is channel 1 then channel 0.
func TestAirSlotLogicalChannel(t *testing.T) {
	want := []int{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0}
	for s, w := range want {
		if got := AirSlotLogicalChannel(s); got != w {
			t.Errorf("timeslot %d: channel %d, want %d", s, got, w)
		}
	}
}

// op25DUIDLookup is OP25's p25p2_duid _duid_lookup table, verbatim.
var op25DUIDLookup = [256]int16{
	0, 0, 0, -1, 0, -1, -1, 1, 0, -1, -1, 4, -1, 8, 2, -1,
	0, -1, -1, 1, -1, 1, 1, 1, -1, 3, 9, -1, 5, -1, -1, 1,
	0, -1, -1, 10, -1, 6, 2, -1, -1, 3, 2, -1, 2, -1, 2, 2,
	-1, 3, 7, -1, 11, -1, -1, 1, 3, 3, -1, 3, -1, 3, 2, -1,
	0, -1, -1, 4, -1, 6, 12, -1, -1, 4, 4, 4, 5, -1, -1, 4,
	-1, 13, 7, -1, 5, -1, -1, 1, 5, -1, -1, 4, 5, 5, 5, -1,
	-1, 6, 7, -1, 6, 6, -1, 6, 14, -1, -1, 4, -1, 6, 2, -1,
	7, -1, 7, 7, -1, 6, 7, -1, -1, 3, 7, -1, 5, -1, -1, 15,
	0, -1, -1, 10, -1, 8, 12, -1, -1, 8, 9, -1, 8, 8, -1, 8,
	-1, 13, 9, -1, 11, -1, -1, 1, 9, -1, 9, 9, -1, 8, 9, -1,
	-1, 10, 10, 10, 11, -1, -1, 10, 14, -1, -1, 10, -1, 8, 2, -1,
	11, -1, -1, 10, 11, 11, 11, -1, -1, 3, 9, -1, 11, -1, -1, 15,
	-1, 13, 12, -1, 12, -1, 12, 12, 14, -1, -1, 4, -1, 8, 12, -1,
	13, 13, -1, 13, -1, 13, 12, -1, -1, 13, 9, -1, 5, -1, -1, 15,
	14, -1, -1, 10, -1, 6, 12, -1, 14, 14, 14, -1, 14, -1, -1, 15,
	-1, 13, 7, -1, 11, -1, -1, 15, 14, -1, -1, 15, -1, 15, 15, 15,
}

// The DUID codewords (SDRTrunk DataUnitID) decode to their value through
// OP25's table, and every word DecodeDUID corrects agrees with it.
func TestDUIDMatchesOP25Lookup(t *testing.T) {
	for v := 0; v < 16; v++ {
		if got := op25DUIDLookup[EncodeDUID(DUID(v))]; int(got) != v {
			t.Errorf("DUID %d codeword %#x: OP25 reads %d", v, EncodeDUID(DUID(v)), got)
		}
	}
	for cw := 0; cw < 256; cw++ {
		if d := DecodeDUID(uint8(cw)); d != DUIDUnknown && int16(d) != op25DUIDLookup[cw] {
			t.Errorf("codeword %#02x: DecodeDUID %d, OP25 %d", cw, d, op25DUIDLookup[cw])
		}
	}
}

// Literal body-mask dibits from OP25 (apps/tdma/lfsr.py xorsyms, indexed
// as p25p2_tdma::handle_packet does: slot*180 + 10 + body dibit) for WACN
// 1 / System 0x018 / NAC 0x293, body dibits 0..15.
func TestAirScrambleMaskMatchesOP25(t *testing.T) {
	seed := framing.PN44SeedFromIdentity(0x1, 0x018, 0x293)
	for slot, want := range map[int]string{
		0:  "0001200221031202",
		3:  "0132332012312223",
		11: "3022312130301332",
	} {
		m := AirScrambleMask(seed, slot)
		for i, c := range want {
			if m[i] != uint8(c-'0') {
				t.Fatalf("slot %d body dibit %d: mask %d, OP25 %c", slot, i, m[i], c)
			}
		}
	}
}

// op25VCWMap is OP25's p25p2_vf extract_vcw, verbatim: on-air bit i of a
// 72-bit voice codeword goes to word op25VCWMap[i][0], bit [i][1]
// (words c0:24, c1:23, c2:11, c3:14 bits).
var op25VCWMap = [72][2]int{
	{0, 23}, {0, 5}, {1, 10}, {2, 3}, {0, 22}, {0, 4}, {1, 9}, {2, 2},
	{0, 21}, {0, 3}, {1, 8}, {2, 1}, {0, 20}, {0, 2}, {1, 7}, {2, 0},
	{0, 19}, {0, 1}, {1, 6}, {3, 13}, {0, 18}, {0, 0}, {1, 5}, {3, 12},
	{0, 17}, {1, 22}, {1, 4}, {3, 11}, {0, 16}, {1, 21}, {1, 3}, {3, 10},
	{0, 15}, {1, 20}, {1, 2}, {3, 9}, {0, 14}, {1, 19}, {1, 1}, {3, 8},
	{0, 13}, {1, 18}, {1, 0}, {3, 7}, {0, 12}, {1, 17}, {2, 10}, {3, 6},
	{0, 11}, {1, 16}, {2, 9}, {3, 5}, {0, 10}, {1, 15}, {2, 8}, {3, 4},
	{0, 9}, {1, 14}, {2, 7}, {3, 3}, {0, 8}, {1, 13}, {2, 6}, {3, 2},
	{0, 7}, {1, 12}, {2, 5}, {3, 1}, {0, 6}, {1, 11}, {2, 4}, {3, 0},
}

// The voice codeword this package hands DMR's AMBE decoder has OP25's
// Phase 2 layout: the unprotected words c2 and c3 and c0's data bits,
// read through OP25's own bit map, are the encoded payload.
func TestAirVoiceCodewordMatchesOP25Layout(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	info := make([]byte, voiceInfoBits)
	for i := range info {
		info[i] = byte(r.Intn(2))
	}
	pkt, err := EncodeAirPacket(0, DUIDVoice2, [][]byte{framing.PackBitsMSB(info), make([]byte, VoiceFrameBytes)}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	onAir := framing.DibitsToBits(pkt[AirISCHDibits+airVoiceFrameBody[0] : AirISCHDibits+airVoiceFrameBody[0]+airVoiceFrameDibits])
	var words [4][24]byte
	for i, m := range op25VCWMap {
		words[m[0]][m[1]] = onAir[i]
	}
	// load_i is MSB first: word bit k is value bit (len-1-k).
	for k := 0; k < 12; k++ {
		if words[0][23-k] != info[k] {
			t.Fatalf("c0 data bit %d differs from payload", k)
		}
	}
	for k := 0; k < 11; k++ {
		if words[2][10-k] != info[24+k] {
			t.Fatalf("c2 bit %d differs from payload", k)
		}
	}
	for k := 0; k < 14; k++ {
		if words[3][13-k] != info[35+k] {
			t.Fatalf("c3 bit %d differs from payload", k)
		}
	}
}

// buildAirStream lays out superframes of packets: voice on logical channel
// lch (4V, the pattern's 2V on its fifth burst), idle SACCH on the other.
func buildAirStream(t *testing.T, masks *AirMasks, superframes, lch int, frames func(burst int) [][]byte) []uint8 {
	t.Helper()
	var out []uint8
	burst := 0
	for sf := 0; sf < superframes; sf++ {
		for slot := 0; slot < AirSlotsPerSuperframe; slot++ {
			duid, voice := DUIDSACCHScrambled, [][]byte(nil)
			if AirSlotLogicalChannel(slot) == lch {
				duid = DUIDVoice4
				if burst%6 == 5 {
					duid = DUIDVoice2
				}
				voice = frames(burst)[:duid.VoiceFrames()]
				burst++
			}
			pkt, err := EncodeAirPacket(slot, duid, voice, masks, uint8(sf&3))
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, pkt...)
		}
	}
	return out
}

func testFrame(n, i int) []byte {
	info := make([]byte, voiceInfoBits)
	for k := range info {
		info[k] = byte((n*7 + i*3 + k) % 5 % 2)
	}
	return framing.PackBitsMSB(info)
}

// The framer locks on the S-ISCH under a constant rotation, names every
// timeslot from the I-ISCH, and the voice frames come back exactly.
func TestAirFramerRecoversVoiceOnItsChannel(t *testing.T) {
	seed := framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x161)
	masks := NewAirMasks(seed)
	frames := func(burst int) [][]byte {
		return [][]byte{testFrame(burst, 0), testFrame(burst, 1), testFrame(burst, 2), testFrame(burst, 3)}
	}
	stream := buildAirStream(t, masks, 3, 1, frames)
	r := rand.New(rand.NewSource(7))
	lead := make([]uint8, 333)
	for i := range lead {
		lead[i] = uint8(r.Intn(4))
	}
	stream = append(lead, stream...)
	for i := range stream {
		stream[i] = (stream[i] + 2) & 3 // constant rotation by 2
	}

	f := NewAirFramer()
	var got []AirBurst
	for i := 0; i < len(stream); i += 97 {
		got = append(got, f.Process(stream[i:min(i+97, len(stream))], i)...)
	}
	// Every packet is framed: the grid starts two packets before the
	// first S-ISCH, at the superframe's opening I-ISCH.
	if len(got) != 3*AirSlotsPerSuperframe {
		t.Fatalf("framed %d bursts of %d", len(got), 3*AirSlotsPerSuperframe)
	}
	voice := 0
	for _, b := range got {
		if b.LogicalChannel != AirSlotLogicalChannel(b.Slot) {
			t.Fatalf("burst at %d: slot %d channel %d", b.StartDibit, b.Slot, b.LogicalChannel)
		}
		if wantSlot := (b.StartDibit - len(lead)) / AirPacketDibits % AirSlotsPerSuperframe; b.Slot != wantSlot {
			t.Fatalf("burst at %d named timeslot %d, want %d", b.StartDibit, b.Slot, wantSlot)
		}
		if !b.DUID.IsVoice() {
			if b.LogicalChannel == 1 {
				t.Errorf("timeslot %d on the voice channel read DUID %v", b.Slot, b.DUID)
			}
			continue
		}
		if b.LogicalChannel != 1 {
			t.Fatalf("voice on channel %d (timeslot %d)", b.LogicalChannel, b.Slot)
		}
		fr, errs := ExtractAirVoiceFrames(b, masks)
		if errs != 0 || len(fr) != b.DUID.VoiceFrames() {
			t.Fatalf("timeslot %d: %d frames, %d corrections", b.Slot, len(fr), errs)
		}
		// buildAirStream numbers the voice channel's bursts in order.
		p := (b.StartDibit - len(lead)) / AirPacketDibits
		n := p / AirSlotsPerSuperframe * 6
		for s := 0; s < p%AirSlotsPerSuperframe; s++ {
			if AirSlotLogicalChannel(s) == 1 {
				n++
			}
		}
		for i, got := range fr {
			if !bytes.Equal(got, testFrame(n, i)) {
				t.Fatalf("burst %d frame %d: got %x, want %x", n, i, got, testFrame(n, i))
			}
		}
		voice++
	}
	if voice != 3*6 {
		t.Fatalf("%d voice bursts recovered", voice)
	}
}

// Descrambling with the wrong seed does not yield the payload: the mask is
// load-bearing (the old voice path never descrambled at all).
func TestAirVoiceNeedsTheRightSeed(t *testing.T) {
	right := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x161))
	wrong := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x162))
	want := testFrame(3, 1)
	pkt, err := EncodeAirPacket(4, DUIDVoice2, [][]byte{want, want}, right, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := AirBurst{Slot: 4, DUID: DUIDVoice2, Packet: pkt}
	if fr, _ := ExtractAirVoiceFrames(b, right); !bytes.Equal(fr[0], want) {
		t.Fatal("right seed did not recover the frame")
	}
	if fr, _ := ExtractAirVoiceFrames(b, wrong); bytes.Equal(fr[0], want) {
		t.Fatal("wrong seed recovered the frame")
	}
}
