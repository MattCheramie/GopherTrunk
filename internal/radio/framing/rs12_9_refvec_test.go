package framing

import "testing"

// rsRefMul is a self-contained russian-peasant GF(2^8) multiply
// (reduction polynomial 0x11D) used only by this test. It is wholly
// independent of rs12_9.go's log/exp tables, so agreement below
// confirms our table-based gfMul and the RS encode are mathematically
// correct rather than merely self-consistent.
func rsRefMul(a, b byte) byte {
	var p byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			p ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1D
		}
		b >>= 1
	}
	return p
}

// rsRefEncode reproduces the de-facto on-air DMR RS(12,9) parity LFSR
// (MMDVMHost CRS129::encode, dmr_utils rs129.encode) from scratch with
// the independent rsRefMul and the canonical generator
// g(x) = (x+α)(x+α²)(x+α³) = x³ + 14x² + 56x + 64 (coeffs low→high
// {64,56,14}). MMDVMHost stores the same POLY = {64U,56U,14U,...} and
// emits parity as in[9]=p2, in[10]=p1, in[11]=p0 — matched here.
func rsRefEncode(msg [9]byte) [3]byte {
	gen := [3]byte{64, 56, 14} // {g0,g1,g2}
	var p [3]byte
	for _, d := range msg {
		fb := d ^ p[2]
		p[2] = p[1] ^ rsRefMul(fb, gen[2])
		p[1] = p[0] ^ rsRefMul(fb, gen[1])
		p[0] = rsRefMul(fb, gen[0])
	}
	return [3]byte{p[2], p[1], p[0]} // cw[9], cw[10], cw[11]
}

// TestRS129MatchesIndependentReferenceEncoder pins our RS(12,9) to the
// MMDVMHost / dmr_utils on-air convention without going through our own
// EncodeRS12_9 — it rebuilds the parity from an independent GF multiply
// and generator. This is the non-self-referential cross-check called
// for in the issue #527 follow-up: a systematic field/generator/seed
// error would diverge here even though every round-trip test passes.
//
// It also confirms the Voice LC Header seed handling end to end:
// EncodeRS12_9 (unmasked) XOR seed must verify under that seed.
func TestRS129MatchesIndependentReferenceEncoder(t *testing.T) {
	msgs := [][9]byte{
		{0x00, 0x10, 0x20, 0x00, 0x0c, 0x30, 0x2f, 0x9b, 0xe5}, // a real-shaped group-voice LC
		{0x00, 0x00, 0x00, 0x00, 0x01, 0x23, 0x45, 0x67, 0x89}, // the tuple our fixtures use
		{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7},
		{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x55},
	}
	for _, msg := range msgs {
		want := rsRefEncode(msg)
		cw := EncodeRS12_9(msg)
		got := [3]byte{cw[9], cw[10], cw[11]}
		if got != want {
			t.Errorf("EncodeRS12_9 parity = % x, independent reference = % x (msg % x)",
				got[:], want[:], msg[:])
			continue
		}
		// Seeded on-air codeword (Voice LC Header) must verify.
		onair := make([]byte, 12)
		copy(onair[:9], msg[:])
		for i := 0; i < 3; i++ {
			onair[9+i] = got[i] ^ RS129SeedVoiceLCHeader[i]
		}
		if !VerifyRS12_9(onair, RS129SeedVoiceLCHeader) {
			t.Errorf("VerifyRS12_9 rejected a valid seeded codeword % x", onair)
		}
	}
}
