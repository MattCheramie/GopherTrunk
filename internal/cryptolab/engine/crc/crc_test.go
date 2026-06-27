package crc

import "testing"

func TestCRC16GSMCompute(t *testing.T) {
	t.Parallel()
	// "123456789" check value for CRC-16/GSM is 0xCE3C (catalogued).
	got := CRC16GSM.Compute([]byte("123456789"))
	if got != 0xCE3C {
		t.Fatalf("CRC-16/GSM check = %#04x, want 0xCE3C", got)
	}
}

func sampleFrames(p Params, msgs [][]byte) []Sample {
	out := make([]Sample, len(msgs))
	for i, m := range msgs {
		out[i] = Sample{Data: m, CRC: p.Compute(m)}
	}
	return out
}

func TestRecoverCRC16GSM(t *testing.T) {
	t.Parallel()
	msgs := [][]byte{
		[]byte("RADIO-01"),
		[]byte("UNIT-007"),
		[]byte("DISPATCH"),
		[]byte("ALPHA-12-BRAVO"), // a longer frame disambiguates init/xorout
		[]byte("Z"),
	}
	samples := sampleFrames(CRC16GSM, msgs)
	found, err := Recover(samples, []int{16})
	if err != nil {
		t.Fatal(err)
	}
	hit := false
	for _, p := range found {
		if p.Poly == CRC16GSM.Poly && p.XorOut == CRC16GSM.XorOut &&
			!p.RefIn && !p.RefOut && p.Init == 0 {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("did not recover CRC-16/GSM; candidates: %+v", found)
	}
}

func TestRecoverReflectedCRC(t *testing.T) {
	t.Parallel()
	// CRC-16/ARC: poly 0x8005, refin/refout true, init 0, xorout 0.
	arc := Params{Width: 16, Poly: 0x8005, Init: 0, RefIn: true, RefOut: true, XorOut: 0}
	msgs := [][]byte{
		[]byte("ABCDEFGH"),
		[]byte("12345678"),
		[]byte("longer-frame-here"),
		[]byte("q"),
	}
	samples := sampleFrames(arc, msgs)
	found, err := Recover(samples, []int{16})
	if err != nil {
		t.Fatal(err)
	}
	hit := false
	for _, p := range found {
		if p.Poly == arc.Poly && p.RefIn && p.RefOut && p.Compute([]byte("verify")) == arc.Compute([]byte("verify")) {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("did not recover CRC-16/ARC; candidates: %+v", found)
	}
}
