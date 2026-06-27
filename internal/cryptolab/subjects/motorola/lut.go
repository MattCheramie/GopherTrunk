// Package motorola implements a cryptolab subject for a length-seeded,
// keyless, byte-oriented alias obfuscator: the output substitution table and
// the per-character decode equation are established, while the per-character
// state update is not. The package supplies the recovered output
// substitution table, the decode primitives, the corpus loader, and four
// incremental recovery modes (gauge sweep, structure enumeration, cell
// solver, from-seed simulation) wired into the cryptolab tool registry.
//
// Every constant and procedure here is derived from observed
// plaintext↔ciphertext data and structural analysis only. These research
// modes are independent of GopherTrunk's live decode path and do not change
// its verification gating.
package motorola

// lutSigned is the recovered 256-entry output substitution table in the
// signed (int8) gauge coordinates the cryptanalysis used. It is a bijection
// over 0..255 and round-trips every byte of the ground-truth corpus given
// the per-character keystream. It is recovered only up to the global affine
// gauge (see engine/gauge); the gauge recovery mode searches for a frame in
// which a clean closed form appears.
var lutSigned = [256]int8{
	21, -119, -1, -62, 90, 120, -65, 17, 110, 39, 75, 24, -2, -61, 0, 60,
	-77, -74, 87, -45, 32, 118, 116, 74, -115, 42, 86, -79, -46, -49, -97, 84,
	-18, 78, -128, 63, -50, -109, 9, -33, 109, -43, 106, 114, 54, 23, 56, 91,
	-11, -55, 126, 85, -118, 58, -112, 66, 53, 16, 52, -8, -37, -104, 68, -107,
	-113, 11, -25, 50, 67, 35, 30, -36, 83, 121, -17, 40, -73, 77, 81, -51,
	4, 94, 65, -126, -87, 20, -127, -116, -89, 88, -106, 28, 70, 13, 96, -66,
	-23, -72, -94, -96, -117, 8, -54, -35, -82, 125, -44, 89, -90, 25, -28, 117,
	-86, -67, 102, 43, -64, -30, 76, -101, -76, 46, 41, -3, -69, 1, -92, 113,
	-75, 33, -71, -21, 31, 45, -42, -13, -110, 10, 122, -32, -93, -52, 47, -98,
	99, -122, -47, 111, -108, 104, 97, -14, 55, -15, -27, -84, 64, -57, 51, -6,
	72, -63, -20, 5, 112, -4, -29, -39, -88, 115, 103, 62, -99, 26, 80, -85,
	108, -58, 22, 18, 124, -123, 29, 59, 6, -59, -114, -91, -53, -31, -80, 49,
	37, 107, -78, 3, -34, -102, 100, -5, 82, 12, -24, 119, -22, 105, -41, 7,
	-40, 15, -10, -9, 73, -124, -7, -121, 38, 2, 95, 48, 44, -95, -81, -125,
	36, -100, -111, 123, 92, -120, 98, -103, -105, -26, 61, 14, -16, -56, -12, 127,
	-19, -70, 69, 101, -38, 93, 27, 79, -60, 19, -48, 34, -83, 57, 71, -68,
}

// Table is the recovered output LUT in unsigned byte coordinates plus its
// inverse permutation.
type Table struct {
	Fwd [256]uint8 // Fwd[ciphertext byte] = high-byte/value
	Inv [256]uint8 // Inv[value] = ciphertext byte
}

// recovered is the validated table, built once.
var recovered = buildTable()

// Recovered returns the recovered output substitution table.
func Recovered() *Table { return recovered }

func buildTable() *Table {
	t := &Table{}
	var seen [256]bool
	for i, v := range lutSigned {
		u := uint8(v)
		t.Fwd[i] = u
		t.Inv[u] = uint8(i)
		if seen[u] {
			panic("alias: recovered LUT is not a bijection")
		}
		seen[u] = true
	}
	return t
}

// High returns the observable accumulator high byte for an even-position
// ciphertext byte: H = LUT[enc_even].
func (t *Table) High(encEven byte) uint8 { return t.Fwd[encEven] }
