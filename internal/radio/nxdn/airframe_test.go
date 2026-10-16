package nxdn

import (
	"math/rand"
	"testing"
)

// The PN9 symbol scrambler must reproduce MMDVMHost's SCRAMBLER table
// (NXDNControl.cpp), an independent reference written as literal bytes:
// the first 20 bits (the FSW) are zero, and over the rest each dibit's
// first bit is the PN bit and its second bit is always zero.
func TestAirScrambleMatchesMMDVMHostTable(t *testing.T) {
	mmdvm := [48]byte{
		0x00, 0x00, 0x00, 0x82, 0xA0, 0x88, 0x8A, 0x00, 0xA2, 0xA8, 0x82, 0x8A, 0x82, 0x02,
		0x20, 0x08, 0x8A, 0x20, 0xAA, 0xA2, 0x82, 0x08, 0x22, 0x8A, 0xAA, 0x08, 0x28, 0x88,
		0x28, 0x28, 0x00, 0x0A, 0x02, 0x82, 0x20, 0x28, 0x82, 0x2A, 0xAA, 0x20, 0x22, 0x80,
		0xA8, 0x8A, 0x08, 0xA0, 0xAA, 0x02,
	}
	bit := func(i int) uint8 { return mmdvm[i/8] >> (7 - i%8) & 1 }
	for i := 0; i < 20; i++ {
		if bit(i) != 0 {
			t.Fatalf("MMDVMHost table bit %d inside the FSW is set", i)
		}
	}
	for i := 0; i < airPostFSWDibits; i++ {
		hi, lo := bit(20+2*i), bit(21+2*i)
		if lo != 0 || hi != airScramble[i] {
			t.Fatalf("post-FSW dibit %d: MMDVMHost %d%d, GopherTrunk PN %d", i, hi, lo, airScramble[i])
		}
	}
}

// LICH values from the issue #1261 reporter's NXDN96 and NXDN48 captures,
// read with DSD-FME's rules (nxdn_frame.c: parity = XOR of the top four
// bits; 0x56 = RDCH voice in both halves, 0x50 = FACCH1 in both).
func TestParseAirLICHOnAirValues(t *testing.T) {
	for _, tc := range []struct {
		raw           uint8
		ok            bool
		first, second bool
	}{
		{0xAC, true, true, true},   // 7-bit 0x56: RDCH, SACCH superframe, no steal
		{0xA0, true, false, false}, // 7-bit 0x50: RDCH, FACCH1 in both halves
		{0xA4, true, false, true},  // 7-bit 0x52: FACCH1 in 1, voice in 2
		{0xA8, true, true, false},  // 7-bit 0x54: voice in 1, FACCH1 in 2
		{0x6C, true, true, true},   // 7-bit 0x36: RTCH voice
		{0xAD, false, false, false},
		{0x03, false, false, false}, // RCCH with a bad parity bit
	} {
		l := ParseAirLICH(lichDibits(tc.raw))
		f, s := l.VoiceHalves()
		if l.ParityOK != tc.ok || (tc.ok && (f != tc.first || s != tc.second)) {
			t.Errorf("LICH %#02x: parity %v halves %v/%v, want %v %v/%v", tc.raw, l.ParityOK, f, s, tc.ok, tc.first, tc.second)
		}
	}
}

func lichDibits(raw uint8) []uint8 {
	d := make([]uint8, AirLICHDibits)
	for i := range d {
		d[i] = (raw>>(7-i))&1<<1 | 1
	}
	return d
}

func randPayloads(r *rand.Rand) [4][]byte {
	var p [4][]byte
	for v := range p {
		p[v] = make([]byte, 49)
		for i := range p[v] {
			p[v][i] = byte(r.Intn(2))
		}
	}
	return p
}

func TestAirVoiceFrameRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	want := randPayloads(r)
	frame, err := EncodeAirVoiceFrame(want)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != AirFrameDibits {
		t.Fatalf("frame = %d dibits, want %d", len(frame), AirFrameDibits)
	}
	post := append([]uint8(nil), frame[AirFSWDibits:]...)
	DescrambleAir(post)
	lich, got := DecodeAirVoice(post)
	if !lich.ParityOK || len(got) != 4 {
		t.Fatalf("lich %+v, %d voice frames", lich, len(got))
	}
	for v := range got {
		for i := range want[v] {
			if got[v].Payload[i] != want[v][i] {
				t.Fatalf("voice frame %d bit %d differs", v, i)
			}
		}
	}
}
