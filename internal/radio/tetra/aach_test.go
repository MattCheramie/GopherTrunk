package tetra

import "testing"

// aachType1 builds the 14 type-1 bits (one bit per byte, MSB first) of an
// ACCESS-ASSIGN PDU from a 2-bit header and two 6-bit fields — the inverse
// of ParseAccessAssign's bit packing, so tests can state fields directly.
func aachType1(header, field1, field2 uint8) []byte {
	bits := make([]byte, 14)
	bits[0] = (header >> 1) & 1
	bits[1] = header & 1
	for i := 0; i < 6; i++ {
		bits[2+i] = (field1 >> (5 - i)) & 1
		bits[8+i] = (field2 >> (5 - i)) & 1
	}
	return bits
}

func TestParseAccessAssignDownlinkUsage(t *testing.T) {
	cases := []struct {
		name        string
		header      uint8
		field1      uint8
		wantUsage   uint8
		wantControl bool
		wantTraffic bool
	}{
		// Header 0: downlink is common control regardless of field 1.
		{"header0 common control", 0, 0x2A, DLUsageCommonControl, true, false},
		// Header 1..3: field 1 is the downlink usage marker.
		{"assigned control", 1, DLUsageAssignedControl, DLUsageAssignedControl, true, false},
		{"common control marker", 2, DLUsageCommonControl, DLUsageCommonControl, true, false},
		{"unallocated", 1, DLUsageUnallocated, DLUsageUnallocated, false, false},
		{"reserved", 3, DLUsageReserved, DLUsageReserved, false, false},
		{"traffic marker 4", 1, 4, 4, false, true},
		{"traffic marker 63", 3, 63, 63, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			aa, ok := ParseAccessAssign(aachType1(tc.header, tc.field1, 0x15))
			if !ok {
				t.Fatal("ParseAccessAssign returned !ok for 14 bits")
			}
			if aa.Header != tc.header {
				t.Errorf("Header = %d, want %d", aa.Header, tc.header)
			}
			if got := aa.DownlinkUsage(); got != tc.wantUsage {
				t.Errorf("DownlinkUsage() = %d, want %d", got, tc.wantUsage)
			}
			if got := aa.IsControlChannel(); got != tc.wantControl {
				t.Errorf("IsControlChannel() = %v, want %v", got, tc.wantControl)
			}
			if got := aa.IsTraffic(); got != tc.wantTraffic {
				t.Errorf("IsTraffic() = %v, want %v", got, tc.wantTraffic)
			}
		})
	}
}

func TestParseAccessAssignShort(t *testing.T) {
	if _, ok := ParseAccessAssign(make([]byte, 13)); ok {
		t.Error("ParseAccessAssign accepted fewer than 14 bits")
	}
}

// TestAccessAssignSurvivesAACHRoundTrip runs a known ACCESS-ASSIGN through
// GopherTrunk's own AACH FEC chain (EncodeAACH → DecodeAACH) and confirms
// ParseAccessAssign recovers the downlink usage marker — i.e. the AACH the
// live decoder hands to dispatchSlice yields the correct per-slot control/
// traffic classification (issue #925).
func TestAccessAssignSurvivesAACHRoundTrip(t *testing.T) {
	const colour = 0x1234
	// A voice slot: header selects DL usage marker, marker = traffic.
	type1 := aachType1(1, 37, 0x2A)
	type5 := EncodeAACH(type1, colour)
	if len(type5) != 30 {
		t.Fatalf("EncodeAACH produced %d type-5 bits, want 30", len(type5))
	}
	recovered, errs := DecodeAACH(type5, colour)
	if errs != 0 {
		t.Fatalf("DecodeAACH errs = %d, want 0 on clean round-trip", errs)
	}
	aa, ok := ParseAccessAssign(recovered)
	if !ok {
		t.Fatal("ParseAccessAssign returned !ok after round-trip")
	}
	if aa.DownlinkUsage() != 37 || !aa.IsTraffic() || aa.IsControlChannel() {
		t.Errorf("round-tripped usage = %d (traffic=%v control=%v), want 37 / traffic",
			aa.DownlinkUsage(), aa.IsTraffic(), aa.IsControlChannel())
	}
}
