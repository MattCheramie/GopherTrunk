package dmr

import (
	"errors"
	"testing"
)

func TestSlotTypeRoundTrip(t *testing.T) {
	in := SlotType{ColorCode: 0xC, DataType: DTCSBK}
	bits := AssembleSlotType(in)
	if len(bits) != 20 {
		t.Fatalf("len = %d, want 20", len(bits))
	}
	got, errs, err := ParseSlotType(bits)
	if err != nil {
		t.Fatalf("ParseSlotType: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean codeword reported %d errors", errs)
	}
	if got != in {
		t.Errorf("parsed = %+v, want %+v", got, in)
	}
}

func TestSlotTypeAllValuesRoundTrip(t *testing.T) {
	// Exhaustive sweep across the 8-bit info space (16 colour codes ×
	// 16 data types) to guarantee the parity matrix encodes every
	// CC/DT combo without collision.
	for cc := uint8(0); cc < 16; cc++ {
		for dt := uint8(0); dt < 16; dt++ {
			in := SlotType{ColorCode: cc, DataType: DataType(dt)}
			bits := AssembleSlotType(in)
			got, errs, err := ParseSlotType(bits)
			if err != nil {
				t.Errorf("cc=%X dt=%X: %v", cc, dt, err)
				continue
			}
			if errs != 0 || got != in {
				t.Errorf("cc=%X dt=%X: round-trip = %+v errs=%d", cc, dt, got, errs)
			}
		}
	}
}

func TestSlotTypeCorrectsSingleErrors(t *testing.T) {
	in := SlotType{ColorCode: 0x5, DataType: DTVoiceLCHeader}
	bits := AssembleSlotType(in)
	for i := 0; i < 20; i++ {
		corrupted := append([]byte(nil), bits...)
		corrupted[i] ^= 1
		got, errs, err := ParseSlotType(corrupted)
		if err != nil {
			t.Fatalf("bit=%d: %v", i, err)
		}
		if got != in {
			t.Errorf("bit=%d: got=%+v want=%+v", i, got, in)
		}
		if errs != 1 {
			t.Errorf("bit=%d: errs=%d want=1", i, errs)
		}
	}
}

func TestSlotTypeRejectsHeavyCorruption(t *testing.T) {
	in := SlotType{ColorCode: 0x9, DataType: DTCSBK}
	bits := AssembleSlotType(in)
	// Flip every other bit (10 errors > t=3) — must be flagged uncorrectable.
	for i := 0; i < 20; i += 2 {
		bits[i] ^= 1
	}
	_, _, err := ParseSlotType(bits)
	if err == nil {
		t.Fatal("expected uncorrectable error")
	}
	if !errors.Is(err, ErrSlotTypeUncorrectable) {
		t.Errorf("err = %v, want ErrSlotTypeUncorrectable", err)
	}
}

func TestSlotTypeShortInputErrors(t *testing.T) {
	_, _, err := ParseSlotType(make([]byte, 19))
	if err == nil {
		t.Fatal("expected error on short input")
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
