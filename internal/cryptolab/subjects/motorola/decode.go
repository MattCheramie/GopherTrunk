package motorola

import "github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"

// DecodeChar applies the verified per-character decode in gauge
// coordinates: char = int8( Modd · (LUT[eo] − Hodd) ), returned as the
// plaintext byte. Arithmetic is mod 256.
func (t *Table) DecodeChar(hodd, modd, eo uint8) byte {
	return byte(modd * (t.Fwd[eo] - hodd))
}

// EncodeEven returns the even-position ciphertext byte that exposes high
// byte h (the inverse of the High read). Used to synthesize test corpora.
func (t *Table) EncodeEven(h uint8) byte { return t.Inv[h] }

// EncodeOdd returns the odd-position ciphertext byte that decodes to char
// under state (hodd, modd). modd must be odd. Used to synthesize test
// corpora and to verify candidate states.
func (t *Table) EncodeOdd(hodd, modd, char uint8) byte {
	// char = modd·(LUT[eo] − hodd)  ⇒  LUT[eo] = char·modd⁻¹ + hodd.
	value := char*gauge.ModInv(modd) + hodd
	return t.Inv[value]
}

// CandidatesForChar returns every state (Hodd, Modd with Modd odd)
// consistent with decoding char from the odd ciphertext byte eo. There are
// exactly 128 — one per odd multiplier — because Hodd is then forced. The
// cell solver intersects these candidate sets across every occurrence of a
// context until the state is pinned.
func (t *Table) CandidatesForChar(char, eo uint8) []gauge.State {
	lut := t.Fwd[eo]
	out := make([]gauge.State, 0, 128)
	for m := 1; m < 256; m += 2 {
		modd := uint8(m)
		// char = modd·(lut − hodd) ⇒ hodd = lut − char·modd⁻¹.
		hodd := lut - char*gauge.ModInv(modd)
		out = append(out, gauge.State{Hi: hodd, Mul: modd})
	}
	return out
}
