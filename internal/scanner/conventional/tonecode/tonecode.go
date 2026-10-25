// Package tonecode holds the sub-audible tone facts that both the
// conventional scanner's CTCSS / DCS detectors and config validation need:
// the DCS on-air codeword, how close two CTCSS tones can be and still be told
// apart, and whether two tone gates can open on the same transmission. It has
// no dependencies so internal/config can use it without importing the DSP.
package tonecode

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
)

// MinSeparableCTCSSHz is the closest two CTCSS tones can sit and still be
// told apart by the scanner's 250 ms Goertzel blocks. Only the EIA pair
// 150.0 / 151.4 Hz (1.4 Hz apart) falls inside it: each leaks ~66 % into the
// other, so a gate on either opens on both, as on most radios.
const MinSeparableCTCSSHz = 2.0

// DCSMatchDistance is how many bits a received 23-bit DCS word may differ
// from a target rotation and still match.
const DCSMatchDistance = 2

// DCSCodewordMask keeps the 23 bits of a DCS word.
const DCSCodewordMask = 1<<23 - 1

// dcsGolayPoly is the Golay(23,12) generator x^11+x^9+x^7+x^6+x^5+x+1.
const dcsGolayPoly = 0xAE3

// DCSCodeword converts a 3-digit octal DCS code (e.g. "023") into the 23-bit
// word a transmitter cycles on air, with bit i the i-th bit sent: bits 0..8
// the code (least significant bit first), bits 9..11 the fixed 0 0 1, bits
// 12..22 the Golay parity. The parity is the remainder of the systematic
// Golay(23,12) encoding with generator x^11+x^9+x^7+x^6+x^5+x+1 (0xAE3),
// taken over the on-air bit order; the conventional package's
// dcs_reference_test.go pins it against the published parity equations for
// all 512 codes (023 → 0x763813).
func DCSCodeword(code string) (uint32, error) {
	if len(code) != 3 {
		return 0, fmt.Errorf("dcs: code %q must be 3 octal digits", code)
	}
	for _, r := range code {
		if r < '0' || r > '7' {
			return 0, fmt.Errorf("dcs: code %q must be octal 0..7", code)
		}
	}
	c, err := strconv.ParseUint(code, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("dcs: parse %q: %w", code, err)
	}
	data := uint32(c) | 1<<11 // code + the fixed 0 0 1
	// Systematic cyclic encoding over the transmit order: with bit i the
	// coefficient of x^(22-i), the data occupies x^22..x^11 and the
	// parity x^10..x^0 is (data · x^11) mod g(x).
	var reg uint32
	for i := 0; i < 12; i++ {
		fb := (data>>uint(i))&1 ^ (reg>>10)&1
		reg = (reg << 1) & 0x7FF
		if fb != 0 {
			reg ^= dcsGolayPoly & 0x7FF
		}
	}
	w := data
	for i := 0; i < 11; i++ {
		w |= ((reg >> uint(10-i)) & 1) << uint(12+i)
	}
	return w, nil
}

// DCSRotations returns every cyclic rotation of cw (23 of them) interleaved
// with the bit-inverse of each, so even entries are the normal sense and odd
// entries the inverted one. 46 entries: small enough that a linear scan
// beats any hashing scheme.
func DCSRotations(cw uint32) []uint32 {
	out := make([]uint32, 0, 46)
	r := cw & DCSCodewordMask
	for i := 0; i < 23; i++ {
		out = append(out, r)
		out = append(out, (^r)&DCSCodewordMask)
		// Cyclic left-shift by one bit inside a 23-bit field.
		msb := (r >> 22) & 1
		r = ((r << 1) | msb) & DCSCodewordMask
	}
	return out
}

// Gate is a tone squelch as configured: Mode "ctcss" or "dcs" ("" / "none"
// for no tone gate), with the config's own field spellings.
type Gate struct {
	Mode        string
	CTCSSHz     float64
	DCSCode     string
	DCSPolarity string
}

// Toned reports whether g gates on a tone at all.
func (g Gate) Toned() bool {
	m := strings.ToLower(strings.TrimSpace(g.Mode))
	return m == "ctcss" || m == "dcs"
}

// String names g the way an operator writes it: "CTCSS 123.0 Hz",
// "DCS 023N" / "DCS 023I" / "DCS 023 (both)", or "" for no tone gate.
func (g Gate) String() string {
	switch strings.ToLower(strings.TrimSpace(g.Mode)) {
	case "ctcss":
		return fmt.Sprintf("CTCSS %.1f Hz", g.CTCSSHz)
	case "dcs":
		n, i := dcsSenses(g.DCSPolarity)
		switch {
		case n && i:
			return "DCS " + g.DCSCode + " (both)"
		case i:
			return "DCS " + g.DCSCode + "I"
		default:
			return "DCS " + g.DCSCode + "N"
		}
	}
	return ""
}

// ID is a short, stable, lower-case token for g, for identifiers such as a
// virtual device serial: "ctcss123.0", "dcs023n", "dcs023i", "dcs023b", or
// "" for no tone gate.
func (g Gate) ID() string {
	switch strings.ToLower(strings.TrimSpace(g.Mode)) {
	case "ctcss":
		return fmt.Sprintf("ctcss%.1f", g.CTCSSHz)
	case "dcs":
		n, i := dcsSenses(g.DCSPolarity)
		switch {
		case n && i:
			return "dcs" + g.DCSCode + "b"
		case i:
			return "dcs" + g.DCSCode + "i"
		default:
			return "dcs" + g.DCSCode + "n"
		}
	}
	return ""
}

// dcsSenses reports which NRZ senses a DCS polarity setting accepts:
// "" / "normal" / "n" the normal sense, "inverted" / "i" the inverted one,
// "both" either.
func dcsSenses(polarity string) (normal, inverted bool) {
	switch strings.ToLower(strings.TrimSpace(polarity)) {
	case "inverted", "i":
		return false, true
	case "both":
		return true, true
	default:
		return true, false
	}
}

// Overlap reports whether one transmission can open both gates, so two
// logical channels on one frequency cannot be told apart by them. An untoned
// gate opens on any carrier, so it overlaps every gate. A CTCSS gate overlaps
// another CTCSS gate within MinSeparableCTCSSHz, and never a DCS gate. Two
// DCS gates overlap when some received word lies within DCSMatchDistance of
// an accepted rotation of each; DCS words are Golay(23,12) codewords (7 bits
// apart), so that happens only for the same code or an alias (023N is the
// same bits as 047I). An unparsable DCS code reports an error.
func Overlap(a, b Gate) (bool, error) {
	if !a.Toned() || !b.Toned() {
		return true, nil
	}
	am := strings.ToLower(strings.TrimSpace(a.Mode))
	bm := strings.ToLower(strings.TrimSpace(b.Mode))
	if am != bm {
		return false, nil
	}
	if am == "ctcss" {
		return math.Abs(a.CTCSSHz-b.CTCSSHz) < MinSeparableCTCSSHz, nil
	}
	ra, err := acceptedDCS(a)
	if err != nil {
		return false, err
	}
	rb, err := acceptedDCS(b)
	if err != nil {
		return false, err
	}
	for _, x := range ra {
		for _, y := range rb {
			if bits.OnesCount32(x^y) <= 2*DCSMatchDistance {
				return true, nil
			}
		}
	}
	return false, nil
}

// acceptedDCS is the set of rotations a DCS gate opens on.
func acceptedDCS(g Gate) ([]uint32, error) {
	cw, err := DCSCodeword(g.DCSCode)
	if err != nil {
		return nil, err
	}
	normal, inverted := dcsSenses(g.DCSPolarity)
	var out []uint32
	for i, r := range DCSRotations(cw) {
		if (i%2 == 1 && inverted) || (i%2 == 0 && normal) {
			out = append(out, r)
		}
	}
	return out, nil
}
