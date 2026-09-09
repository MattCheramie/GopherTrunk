package framing

// Extended Golay (24, 12, 8): triple-error-correcting / quadruple-error-
// detecting linear block code used widely in P25 and DMR framing.
//
// The implementation uses the standard systematic generator matrix
// G = [I_12 | B] where B is the 12×12 parity matrix below (taken from
// MacWilliams & Sloane, "The Theory of Error-Correcting Codes", §2.6).

var golayB = [12]uint16{
	0xC75, 0x63B, 0xF68, 0x7B4, 0x3DA, 0xD99,
	0xECC, 0x766, 0xCB3, 0xA51, 0xD2A, 0x995,
}

// GolayEncode24_12 encodes 12 data bits (in the low 12 bits of input) into
// a 24-bit codeword. Output layout: [data(12) | parity(12)] with the data
// bits in bits 23..12 and parity in bits 11..0.
func GolayEncode24_12(data uint16) uint32 {
	d := uint32(data & 0x0FFF)
	var parity uint32
	for i := 0; i < 12; i++ {
		if d&(1<<uint(i)) != 0 {
			parity ^= uint32(golayB[i])
		}
	}
	return d<<12 | parity
}

// golaySyndromeLeader maps a 12-bit syndrome to the minimum-weight error
// pattern that produces it, for every pattern of weight ≤ 3. Built at init.
//
// It replaces a 4096-codeword nearest-neighbour search that ran on every
// decode. That search is correct but costs about a thousand times as much, and
// this decoder sits on a per-sub-frame path — the saving is the difference
// between a Phase 2 chain that keeps up with real time and one that does not.
var golaySyndromeLeader map[uint32]uint32

func init() {
	golaySyndromeLeader = make(map[uint32]uint32, 2325)
	add := func(e uint32) {
		syn := golaySyndrome24(e)
		if old, ok := golaySyndromeLeader[syn]; !ok || PopCount64(uint64(e)) < PopCount64(uint64(old)) {
			golaySyndromeLeader[syn] = e
		}
	}
	add(0)
	for a := 0; a < 24; a++ {
		add(1 << uint(a))
		for b := a + 1; b < 24; b++ {
			add(1<<uint(a) | 1<<uint(b))
			for c := b + 1; c < 24; c++ {
				add(1<<uint(a) | 1<<uint(b) | 1<<uint(c))
			}
		}
	}
}

// golaySyndrome24 returns the 12-bit syndrome of a received word: the parity
// its data half implies, XORed with the parity it actually carries.
func golaySyndrome24(cw uint32) uint32 {
	d := cw >> 12 & 0x0FFF
	var parity uint32
	for i := 0; i < 12; i++ {
		if d&(1<<uint(i)) != 0 {
			parity ^= uint32(golayB[i])
		}
	}
	return (parity ^ cw) & 0x0FFF
}

// GolayDecode24_12 decodes a 24-bit codeword. Returns (data, errors) where
// errors is the corrected bit count, or -1 when the received word is more than
// 3 bits from every codeword — beyond the guaranteed correction radius.
//
// Note this is a different equivalent form of the Golay code from
// GolayDecodeCyclic23_12 in golay23_12.go, and the two are not
// interchangeable: a codeword of one looks like noise to the other. P25 and
// AMBE payloads use the cyclic form.
func GolayDecode24_12(cw uint32) (uint16, int) {
	cw &= 0x00FFFFFF
	e, ok := golaySyndromeLeader[golaySyndrome24(cw)]
	if !ok {
		return uint16(cw >> 12 & 0x0FFF), -1
	}
	return uint16((cw ^ e) >> 12 & 0x0FFF), PopCount64(uint64(e))
}
