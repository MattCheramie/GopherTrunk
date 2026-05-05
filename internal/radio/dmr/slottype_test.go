package dmr

import "testing"

func TestSlotTypeRoundTrip(t *testing.T) {
	in := SlotType{ColorCode: 0xC, DataType: DTCSBK}
	bits := AssembleSlotType(in)
	if len(bits) != 20 {
		t.Fatalf("len = %d, want 20", len(bits))
	}
	got, err := ParseSlotType(bits)
	if err != nil {
		t.Fatalf("ParseSlotType: %v", err)
	}
	if got != in {
		t.Errorf("parsed = %+v, want %+v", got, in)
	}
}

func TestDataTypeString(t *testing.T) {
	cases := map[DataType]string{
		DTCSBK:             "CSBK",
		DTVoiceLCHeader:    "VoiceLCHeader",
		DTTerminatorWithLC: "TerminatorWithLC",
		DataType(0xE):      "DataType(E)",
	}
	for dt, want := range cases {
		if got := dt.String(); got != want {
			t.Errorf("DataType(%X).String() = %s, want %s", uint8(dt), got, want)
		}
	}
}
