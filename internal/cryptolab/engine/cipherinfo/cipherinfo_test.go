package cipherinfo

import (
	"strings"
	"testing"
)

func TestTEA1BackdoorRecorded(t *testing.T) {
	t.Parallel()
	info, ok := Lookup("tetra", 0x01)
	if !ok {
		t.Fatal("TEA1 not in the knowledge base")
	}
	if info.Strength != StrengthBroken {
		t.Errorf("TEA1 strength = %s, want broken", info.Strength)
	}
	if info.EffectiveKeyBits != 32 {
		t.Errorf("TEA1 effective key bits = %d, want 32 (the backdoor)", info.EffectiveKeyBits)
	}
	if !info.BruteForceable {
		t.Error("TEA1 should be flagged brute-forceable")
	}
	if !strings.Contains(info.Reference, "CVE-2022-24402") {
		t.Errorf("TEA1 reference = %q, want the CVE", info.Reference)
	}
}

func TestP25Grades(t *testing.T) {
	t.Parallel()
	adp, _ := Lookup("p25", 0xAA)
	if !adp.BruteForceable || adp.Strength != StrengthWeak {
		t.Errorf("ADP should be weak + brute-forceable, got %+v", adp)
	}
	if !adp.Bundled {
		t.Error("ADP should be marked bundled (p25crypto implements it)")
	}
	aes, _ := Lookup("p25", 0x84)
	if aes.Strength != StrengthStrong || aes.BruteForceable {
		t.Errorf("AES-256 should be strong and not brute-forceable, got %+v", aes)
	}
}

func TestProtocolAliasing(t *testing.T) {
	t.Parallel()
	a, ok1 := Lookup("p25-phase1", 0xAA)
	b, ok2 := Lookup("APCO25", 0xAA)
	if !ok1 || !ok2 || a.Name != b.Name {
		t.Fatalf("protocol aliases should resolve to the same entry: %v/%v", a, b)
	}
}

func TestUnknownAlgorithm(t *testing.T) {
	t.Parallel()
	if _, ok := Lookup("p25", 0x42); ok {
		t.Error("expected no entry for an unknown algid")
	}
	if _, ok := Lookup("mystery", 0x01); ok {
		t.Error("expected no entry for an unknown protocol")
	}
}
