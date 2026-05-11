package mpt1327

import "testing"

func TestCodeword48RoundTripPreservesOp(t *testing.T) {
	cases := []Codeword{
		{Type: TypeAddress, Prefix: 0x05, Ident: 0x123, Op: 0x000, Function: 0x1ABCD},
		{Type: TypeAddress, Prefix: 0x7F, Ident: 0x1FFF, Op: 0x3FF, Function: 0x1FFFF},
		{Type: TypeData, Prefix: 0x42, Ident: 0x0A5A, Op: 0x155, Function: 0x0AAAA},
		{Type: TypeAddress, Prefix: 0x01, Ident: 0x0001, Op: 0x001, Function: 0x00001},
	}
	for _, in := range cases {
		bytes := AssembleCodeword48(in)
		if len(bytes) != 6 {
			t.Fatalf("AssembleCodeword48(%+v) = %d bytes, want 6", in, len(bytes))
		}
		got, err := ParseCodeword48(bytes)
		if err != nil {
			t.Fatalf("ParseCodeword48: %v", err)
		}
		if got != in {
			t.Errorf("round-trip:\n  got  %+v\n  want %+v", got, in)
		}
	}
}

func TestCodewordBits48RoundTripPreservesOp(t *testing.T) {
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0x55,
		Ident:    0x1234,
		Op:       0x2AA,
		Function: 0x1ABCD,
	}
	bits := CodewordBits48(in)
	if len(bits) != 48 {
		t.Fatalf("CodewordBits48 returned %d bits, want 48", len(bits))
	}
	got, err := CodewordFromBits48(bits)
	if err != nil {
		t.Fatalf("CodewordFromBits48: %v", err)
	}
	if got != in {
		t.Errorf("round-trip:\n  got  %+v\n  want %+v", got, in)
	}
}

// TestLegacy38BitHelpersIgnoreOp verifies that the existing
// AssembleCodeword / ParseCodeword path stays 38-bit and silently
// drops the Op field — preserves back-compat for callers and tests
// that pre-date the Op field.
func TestLegacy38BitHelpersIgnoreOp(t *testing.T) {
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0x05,
		Ident:    0x123,
		Op:       0x3FF, // populated but should be dropped
		Function: 0x1ABCD,
	}
	bytes := AssembleCodeword(in)
	if len(bytes) != 5 {
		t.Fatalf("AssembleCodeword (legacy) = %d bytes, want 5", len(bytes))
	}
	got, err := ParseCodeword(bytes)
	if err != nil {
		t.Fatalf("ParseCodeword: %v", err)
	}
	if got.Op != 0 {
		t.Errorf("legacy round-trip preserved Op = %#x, want 0", got.Op)
	}
	// The other fields should still survive.
	if got.Type != in.Type || got.Prefix != in.Prefix ||
		got.Ident != in.Ident || got.Function != in.Function {
		t.Errorf("legacy round-trip lost a non-Op field:\n  got  %+v\n  want %+v (Op zeroed)", got, in)
	}
}

func TestCodewordFromBits48RejectsWrongLength(t *testing.T) {
	if _, err := CodewordFromBits48(make([]byte, 40)); err == nil {
		t.Errorf("CodewordFromBits48 accepted 40-bit input, want error")
	}
	if _, err := CodewordFromBits48(make([]byte, 38)); err == nil {
		t.Errorf("CodewordFromBits48 accepted 38-bit input, want error")
	}
}

func TestParseCodeword48RejectsWrongLength(t *testing.T) {
	if _, err := ParseCodeword48(make([]byte, 5)); err == nil {
		t.Errorf("ParseCodeword48 accepted 5-byte input, want error")
	}
	if _, err := ParseCodeword48(make([]byte, 7)); err == nil {
		t.Errorf("ParseCodeword48 accepted 7-byte input, want error")
	}
}

// TestOpFieldMaskedTo10Bits: AssembleCodeword48 must mask Op to its
// 10-bit width so a stray high-bit doesn't leak into Ident.
func TestOpFieldMaskedTo10Bits(t *testing.T) {
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0,
		Ident:    0,
		Op:       0xFFFF, // outside 10-bit range; should be masked to 0x3FF
		Function: 0,
	}
	bytes := AssembleCodeword48(in)
	got, _ := ParseCodeword48(bytes)
	if got.Op != 0x3FF {
		t.Errorf("Op not masked: got %#x, want 0x3FF", got.Op)
	}
	if got.Ident != 0 {
		t.Errorf("Op overflow leaked into Ident: Ident = %#x, want 0", got.Ident)
	}
}
