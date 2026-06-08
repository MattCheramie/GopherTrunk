package refcompare

import (
	"reflect"
	"testing"
)

func TestCompareIdenticalPasses(t *testing.T) {
	r := Compare([]uint16{0x167, 0x293}, []uint16{0x293, 0x167}, 40, 42, 1.0, 0.8)
	if !r.Pass {
		t.Errorf("identical NAC sets should pass: %s", r.Notes)
	}
	if r.Agreement != 1.0 {
		t.Errorf("agreement = %v, want 1.0", r.Agreement)
	}
	if !reflect.DeepEqual(r.CommonNACs, []uint16{0x167, 0x293}) {
		t.Errorf("common = %v", r.CommonNACs)
	}
}

func TestCompareDisagreementFails(t *testing.T) {
	// GT found 0x167; reference found 0x167 and 0x200 → agreement 1/2 = 0.5.
	r := Compare([]uint16{0x167}, []uint16{0x167, 0x200}, 40, 40, 1.0, 0.8)
	if r.Pass {
		t.Error("partial NAC agreement should fail at minAgreement=1.0")
	}
	if r.Agreement != 0.5 {
		t.Errorf("agreement = %v, want 0.5", r.Agreement)
	}
	if !reflect.DeepEqual(r.RefOnly, []uint16{0x200}) {
		t.Errorf("ref-only = %v, want [0x200]", r.RefOnly)
	}
}

func TestCompareYieldBar(t *testing.T) {
	// NACs agree but GT's frame yield is far below the reference's.
	r := Compare([]uint16{0x167}, []uint16{0x167}, 10, 100, 1.0, 0.8)
	if r.Pass {
		t.Errorf("low frame yield should fail: %s", r.Notes)
	}
	// Reference reported no frames → yield bar is skipped.
	r2 := Compare([]uint16{0x167}, []uint16{0x167}, 10, 0, 1.0, 0.8)
	if !r2.Pass {
		t.Errorf("yield bar should be skipped when reference reports 0 frames: %s", r2.Notes)
	}
}

func TestParseNACs(t *testing.T) {
	out := `
		[OP25] tuned to 420.050 MHz
		NAC 0x167 found, DUID=TSDU
		nac=0x167 (repeat)
		NAC: 659 decimal form
		random NACK line should not match a hex token
	`
	got := ParseNACs(out, nil)
	// 0x167 twice, 659 (=0x293) once, plus "NACK" → the "K" makes the token
	// non-hex after the separator, so it should be dropped or parsed as the
	// following token; assert the real NACs are present.
	want := map[uint16]bool{0x167: true, 0x293: true}
	for _, n := range got {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Errorf("ParseNACs missed %v (got %v)", want, got)
	}
}
