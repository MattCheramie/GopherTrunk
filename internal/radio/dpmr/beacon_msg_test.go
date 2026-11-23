package dpmr

import "testing"

// specField is one row of a clause 5.2 message table: its length and value.
type specField struct {
	n int
	v uint32
}

// specMessage lays rows out one after another, MSB first, as the tables
// list them. It shares nothing with the parsers' bit offsets, so a parser
// that reads a field from the wrong place fails here.
func specMessage(t *testing.T, rows ...specField) BeaconMessage {
	t.Helper()
	var m BeaconMessage
	pos := 0
	for _, r := range rows {
		for i := 0; i < r.n; i++ {
			m.Bits[pos+i] = uint8(r.v>>uint(r.n-1-i)) & 1
		}
		pos += r.n
	}
	if pos != 72 {
		t.Fatalf("table rows add up to %d bits, want 72", pos)
	}
	return m
}

// Table 5.30, the Broadcast Aloha, row by row. RSVD is nine bits: its value
// column reads "9 x 0" and the table only adds up to 72 that way.
func TestAlohaFollowsTable530(t *testing.T) {
	m := specMessage(t,
		specField{4, 0xB},      // MT = Broadcast
		specField{24, 0xABCDE}, // ID0+1
		specField{14, 0x2A5C},  // SYS
		specField{9, 0},        // RSVD
		specField{1, 1},        // INFILL
		specField{5, 17},       // MASK
		specField{4, 9},        // NRAND_WAIT
		specField{3, 0},        // MI_TYPE = Aloha
		specField{2, 2},        // SF
		specField{4, 11},       // BACKOFF
		specField{1, 1},        // ACTIVE
		specField{1, 0},        // REG
	)
	a, ok := m.Aloha()
	want := Aloha{MSID: 0xABCDE, SystemCode: 0x2A5C, Infill: true, Mask: 17, NRandWait: 9, Service: 2, Backoff: 11, Active: true}
	if !ok || a != want {
		t.Fatalf("Aloha = %+v ok=%v, want %+v", a, ok, want)
	}
	if got := EncodeAloha(want); got != m {
		t.Fatal("EncodeAloha does not lay the fields out as table 5.30 does")
	}
	if m.Name() != "B_ALOHA" {
		t.Fatalf("name %q", m.Name())
	}
}

// Table 5.38, the Goto Channel broadcast, row by row. The receive frequency
// is split around the communication mode: FR1 its nine most significant
// bits, FR2 its six least.
func TestGotoChannelFollowsTable538(t *testing.T) {
	const fr = 0x5A3C // 15 bits
	m := specMessage(t,
		specField{4, 0xB},       // MT = Broadcast
		specField{24, 0x02C00A}, // ID0+1: talkgroup 012345*
		specField{15, 0x1234},   // FT
		specField{9, fr >> 6},   // FR1
		specField{3, 1},         // M = voice + slow data
		specField{6, fr & 0x3F}, // FR2
		specField{3, 3},         // MI_TYPE = Goto Channel
		specField{1, 0},         // SEP = 3.125 kHz
		specField{4, 5},         // BAND = 400-500 MHz
		specField{1, 1},         // EF
		specField{1, 0},         // BRCST
		specField{1, 0},         // RSVD
	)
	g, ok := m.GotoChannel()
	want := GotoChannel{Called: 0x02C00A, TxField: 0x1234, RxField: fr, Mode: 1, Band: 5, Emergency: true}
	if !ok || g != want {
		t.Fatalf("GotoChannel = %+v ok=%v, want %+v", g, ok, want)
	}
	if got := EncodeGotoChannel(want); got != m {
		t.Fatal("EncodeGotoChannel does not lay the fields out as table 5.38 does")
	}
	if !g.Voice() {
		t.Fatal("mode 1 is a voice call")
	}
	// f = BAND + SEP x FR (table 5.65).
	if hz, ok := g.RxHz(); !ok || hz != 400_000_000+3125*fr {
		t.Fatalf("RxHz = %d ok=%v", hz, ok)
	}
	if hz, ok := g.TxHz(); !ok || hz != 400_000_000+3125*0x1234 {
		t.Fatalf("TxHz = %d ok=%v", hz, ok)
	}
}

// The bands of table 5.63, at their lower edges, with both separations,
// and the two codes that name no frequency.
func TestGotoChannelBands(t *testing.T) {
	edges := []uint32{0, 80, 160, 240, 320, 400, 480, 560, 640, 700, 750, 800, 850, 900}
	for b, mhz := range edges {
		g := GotoChannel{Band: uint8(b), RxField: 2, Sep5k: true}
		if hz, ok := g.RxHz(); !ok || hz != mhz*1_000_000+10_000 {
			t.Errorf("band %d: %d Hz ok=%v", b, hz, ok)
		}
	}
	for _, b := range []uint8{BandReserved, BandCustom} {
		if _, ok := (GotoChannel{Band: b}).RxHz(); ok {
			t.Errorf("band %d resolved to a frequency", b)
		}
	}
	// A PMR446 channel round-trips through the band fields.
	band, sep, field, ok := FrequencyFields(446_006_250)
	g := GotoChannel{Band: band, Sep5k: sep, RxField: field}
	if hz, ok2 := g.RxHz(); !ok || !ok2 || hz != 446_006_250 {
		t.Fatalf("446.00625 MHz: band %d sep5k %v field %d -> %d", band, sep, field, hz)
	}
}

// Table 5.99 (SYScast1) and tables 5.105 / 5.106 (the calling party in
// SYScast2 and SYScast3): 17-bit fields, MSB first.
func TestSYScastFields(t *testing.T) {
	// Reg = 1, RSVD = 00, SYScode = 0x1357.
	s1 := ParseSYScast1(1<<16 | 0x1357)
	if !s1.Registration || s1.SystemCode != 0x1357 {
		t.Fatalf("SYScast1 = %+v", s1)
	}
	if EncodeSYScast1(s1) != 1<<16|0x1357 {
		t.Fatal("EncodeSYScast1")
	}
	// SYTYPE 011, RSVD 0000, ten MSBs; SYTYPE 100, fourteen LSBs.
	const caller = 0xABCDEF
	sy2 := uint32(3)<<14 | caller>>14
	sy3 := uint32(4)<<14 | caller&0x3FFF
	if id, ok := CallingParty(sy2, sy3); !ok || id != caller {
		t.Fatalf("calling party %06X ok=%v", id, ok)
	}
	if a, b := EncodeCallingParty(caller); a != sy2 || b != sy3 {
		t.Fatal("EncodeCallingParty")
	}
	// A SYScast that carries something else is not a calling party.
	if _, ok := CallingParty(EncodeSYScast2(SYTypeFrameCounter, 5), sy3); ok {
		t.Fatal("frame counter read as a calling party")
	}
}

// Table A.2: the standard's own examples of the B2 address mapping.
func TestAddressB2Examples(t *testing.T) {
	for _, tc := range []struct {
		digits string
		id     uint32
		group  bool
	}{
		{"1234567", 0x1B91FD, false},
		{"468956*", 0x68BF08, true},
		{"012345*", 0x02C00A, true},
		{"0123460", 0x02C00B, false},
		{"999****", 0xDF6767, true},
	} {
		if id, ok := AddressID(tc.digits); !ok || id != tc.id {
			t.Errorf("%s -> %06X ok=%v, want %06X", tc.digits, id, ok, tc.id)
		}
		if d, ok := Address(tc.id); !ok || d != tc.digits {
			t.Errorf("%06X -> %q ok=%v, want %q", tc.id, d, ok, tc.digits)
		}
		if IsWildcardTalkgroup(tc.id) != tc.group {
			t.Errorf("%s: talkgroup %v", tc.digits, !tc.group)
		}
	}
	if _, ok := Address(0xDF6768); ok {
		t.Error("an address above 999**** mapped to digits")
	}
}
