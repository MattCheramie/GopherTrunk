package panels

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

func TestHashRows_StableForIdenticalRows(t *testing.T) {
	rows := []client.SDRStatus{
		{Serial: "AAA", Driver: "rtlsdr", Role: "control"},
		{Serial: "BBB", Driver: "rtlsdr", Role: "voice"},
	}
	keyFn := func(d client.SDRStatus) string { return d.Serial + "|" + d.Role }
	h1 := hashRows(rows, keyFn)
	h2 := hashRows(rows, keyFn)
	if h1 != h2 {
		t.Fatalf("hashRows non-deterministic: %x vs %x", h1, h2)
	}
}

func TestHashRows_ChangesOnContentChange(t *testing.T) {
	a := []client.SDRStatus{{Serial: "AAA", Driver: "rtlsdr"}}
	b := []client.SDRStatus{{Serial: "BBB", Driver: "rtlsdr"}}
	keyFn := func(d client.SDRStatus) string { return d.Serial + "|" + d.Driver }
	if hashRows(a, keyFn) == hashRows(b, keyFn) {
		t.Fatal("hashRows collided across distinct content")
	}
}

func TestHashRows_ChangesOnOrderChange(t *testing.T) {
	a := []client.SDRStatus{
		{Serial: "AAA"}, {Serial: "BBB"},
	}
	b := []client.SDRStatus{
		{Serial: "BBB"}, {Serial: "AAA"},
	}
	keyFn := func(d client.SDRStatus) string { return d.Serial }
	if hashRows(a, keyFn) == hashRows(b, keyFn) {
		t.Fatal("hashRows did not detect reordering")
	}
}

func TestHashStringMap_StableWithMapOrder(t *testing.T) {
	m1 := map[string]float64{"a": 1, "b": 2, "c": 3}
	m2 := map[string]float64{"c": 3, "a": 1, "b": 2}
	if hashStringMap(m1) != hashStringMap(m2) {
		t.Fatal("hashStringMap is sensitive to insertion order — sort failed")
	}
}
