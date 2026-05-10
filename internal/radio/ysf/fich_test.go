package ysf

import "testing"

func TestFICHRoundTripCoversAllFrameTypes(t *testing.T) {
	cases := []FrameType{
		FrameTypeHeader, FrameTypeComms, FrameTypeTerminator, FrameTypeTest,
	}
	for _, ft := range cases {
		t.Run(ft.String(), func(t *testing.T) {
			in := FICH{
				FrameType:   ft,
				CallType:    CallTypeGroup,
				BlockNumber: 1,
				BlockTotal:  3,
				FrameNumber: 2,
				FrameTotal:  5,
				DataType:    DataTypeVDMode2,
				VoIP:        false,
				DataType2:   0,
				SquelchMode: false,
				SquelchCode: 0,
				Device:      0,
			}
			out, err := ParseFICH(AssembleFICH(in))
			if err != nil {
				t.Fatalf("ParseFICH: %v", err)
			}
			if out != in {
				t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
			}
		})
	}
}

func TestFICHRoundTripExercisesAllFields(t *testing.T) {
	// Every nontrivial field set to a different non-default value so a
	// bit-packing typo will surface immediately.
	in := FICH{
		FrameType:   FrameTypeComms,
		CallType:    CallTypeRadioID,
		BlockNumber: 0x2,
		BlockTotal:  0x3,
		FrameNumber: 0x5,
		FrameTotal:  0x7,
		DataType:    DataTypeVoiceFR,
		VoIP:        true,
		DataType2:   0x2,
		SquelchMode: true,
		SquelchCode: 0x5A, // 7-bit value spanning octets 2/3
		Device:      0x2,
	}
	out, err := ParseFICH(AssembleFICH(in))
	if err != nil {
		t.Fatalf("ParseFICH: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestFICHDetectsCRCError(t *testing.T) {
	in := FICH{FrameType: FrameTypeHeader, CallType: CallTypeGroup}
	b := AssembleFICH(in)
	b[2] ^= 0x40 // flip a bit inside the info section
	if _, err := ParseFICH(b); err != CRCError {
		t.Errorf("err = %v, want CRCError", err)
	}
}

func TestFICHRejectsWrongLength(t *testing.T) {
	if _, err := ParseFICH(make([]byte, 5)); err == nil {
		t.Error("ParseFICH accepted 5-byte input")
	}
	if _, err := ParseFICH(make([]byte, 7)); err == nil {
		t.Error("ParseFICH accepted 7-byte input")
	}
}

func TestFrameTypeStringCoverage(t *testing.T) {
	cases := map[FrameType]string{
		FrameTypeHeader:     "Header",
		FrameTypeComms:      "Communications",
		FrameTypeTerminator: "Terminator",
		FrameTypeTest:       "Test",
	}
	for ft, want := range cases {
		if got := ft.String(); got != want {
			t.Errorf("%X.String() = %q, want %q", uint8(ft), got, want)
		}
	}
}

func TestCallTypeStringCoverage(t *testing.T) {
	if got := CallTypeGroup.String(); got != "Group" {
		t.Errorf("Group string = %q", got)
	}
	if got := CallTypeRadioID.String(); got != "RadioID" {
		t.Errorf("RadioID string = %q", got)
	}
	if got := CallTypeReserved.String(); got != "Reserved(2)" {
		t.Errorf("Reserved(2) string = %q", got)
	}
}

func TestDataTypeStringCoverage(t *testing.T) {
	cases := map[DataType]string{
		DataTypeVDMode1: "VDMode1",
		DataTypeDataFR:  "DataFullRate",
		DataTypeVDMode2: "VDMode2",
		DataTypeVoiceFR: "VoiceFullRate",
	}
	for dt, want := range cases {
		if got := dt.String(); got != want {
			t.Errorf("%X.String() = %q, want %q", uint8(dt), got, want)
		}
	}
}

func TestFICHCRCUsesXMODEMInit(t *testing.T) {
	// All-zero info bytes should produce CRC = 0 with init=0x0000
	// (regression guard against accidentally using the CCITT/FALSE
	// 0xFFFF init shared with the rest of the codebase).
	out := AssembleFICH(FICH{})
	if out[4] != 0 || out[5] != 0 {
		t.Errorf("all-zero info → CRC = %02X%02X, want 0000", out[4], out[5])
	}
}
