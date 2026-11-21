package dpmr

import (
	"math/rand"
	"strings"
	"testing"
)

// The literals in this file come from outside this package: ETSI
// TS 102 658 V2.6.1 for the sync words, the generator matrix and the field
// layout, and DSD-FME's own dPMR routines (ScrambledPMRBit,
// DeInterleave6x12DPmrBit, Hamming_12_8_decode, CRC7BitdPMR), compiled
// outside the repository, for the scrambling sequence and for the verdict
// on every control channel in airCCHVectors.

func dibitsOf(s string) []uint8 {
	out := make([]uint8, 0, len(s))
	for _, c := range s {
		if c >= '0' && c <= '3' {
			out = append(out, uint8(c-'0'))
		}
	}
	return out
}

func dibitsOfBinary(s string) []uint8 {
	s = strings.ReplaceAll(s, " ", "")
	out := make([]uint8, len(s)/2)
	for i := range out {
		out[i] = (s[2*i]-'0')<<1 | (s[2*i+1] - '0')
	}
	return out
}

func sameDibits(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAirSyncWordsAreTheStandards holds the sync words to clause 6.1, which
// prints each one in binary and in hex.
func TestAirSyncWordsAreTheStandards(t *testing.T) {
	cases := []struct {
		name   string
		got    []uint8
		binary string
	}{
		{"FS1", AirFS1Dibits(), "0101 0111 1111 1111 0101 1111 0111 0101 1101 0101 0111 0111"},
		{"FS2", AirFS2Dibits(), "0101 1111 1111 0111 0111 1101"},
		{"FS3", AirFS3Dibits(), "0111 1101 1101 1111 1111 0101"},
		{"FS4", wordToDibits(AirFS4, 24), "1111 1101 0101 0101 1111 0101 1101 1111 0111 1111 1101 1101"},
	}
	for _, c := range cases {
		if want := dibitsOfBinary(c.binary); !sameDibits(c.got, want) {
			t.Errorf("%s = %v, the standard has %v", c.name, c.got, want)
		}
		// Sync symbols are all outer ones (+3 or -3).
		for i, d := range c.got {
			if d&1 == 0 {
				t.Errorf("%s dibit %d is an inner symbol", c.name, i)
			}
		}
	}
	// "FS4 is a symbol-wise complement of FS1" (note to clause 6.1.4).
	if AirFS4 != AirFS1^0xAAAAAAAAAAAA {
		t.Errorf("FS4 is not the complement of FS1")
	}
	// The old model's words (sync.go) are not the air format.
	if uint64(AirFS2) == FS2Hex || AirFS1 == FS1Hex {
		t.Errorf("air sync words equal the legacy model's")
	}
	if airFS2Sign != 0b001111010110 || airFS3Sign != 0b011010111100 {
		t.Errorf("sign words FS2 %012b FS3 %012b", airFS2Sign, airFS3Sign)
	}
}

// TestAirScramblerMatchesReference: the first 72 bits of x^9 + x^5 + 1 from
// an all-ones register, as the reference decoder generates them.
func TestAirScramblerMatchesReference(t *testing.T) {
	const want = "111111111000001111011111000101110011001000001001010011101101000111100111"
	if got := bitString(airScramble[:72]); got != want {
		t.Fatalf("scrambler\n got %s\nwant %s", got, want)
	}
	// The sequence has period 511; the 120 bits kept are one stretch of it.
	for i := 9; i < len(airScramble); i++ {
		if airScramble[i] != airScramble[i-9]^airScramble[i-5] {
			t.Fatalf("bit %d breaks the recurrence", i)
		}
	}
}

// TestHamming128IsTheStandardsMatrix: table 7.2 lists the parity bits
// C3..C0 for each of the eight identity rows X7..1.
func TestHamming128IsTheStandardsMatrix(t *testing.T) {
	rows := [8]uint8{0b1110, 0b0111, 0b1010, 0b0101, 0b1011, 0b1100, 0b0110, 0b0011}
	for i, want := range rows {
		if got := hamming128Parity(0x80 >> uint(i)); got != want {
			t.Errorf("row %d (X%d): parity %04b, table 7.2 has %04b", i+1, 7-i, got, want)
		}
	}
	for d := 0; d < 256; d++ {
		cw := uint16(d)<<4 | uint16(hamming128Parity(uint8(d)))
		if data, fixed, ok := hamming128Decode(cw); !ok || fixed != 0 || int(data) != d {
			t.Fatalf("clean codeword %03x: data %02x fixed %d ok %v", cw, data, fixed, ok)
		}
		for bit := 0; bit < 12; bit++ {
			data, fixed, ok := hamming128Decode(cw ^ 1<<uint(bit))
			if !ok || fixed != 1 || int(data) != d {
				t.Fatalf("codeword %03x, bit %d flipped: data %02x fixed %d ok %v", cw, bit, data, fixed, ok)
			}
		}
	}
	// Three of the fifteen non-zero syndromes belong to no single error
	// (the code is a (15,11) Hamming code shortened by three positions).
	bad := 0
	for syn := 1; syn < 16; syn++ {
		if _, _, ok := hamming128Decode(uint16(syn)); !ok {
			bad++
		}
	}
	if bad != 3 {
		t.Errorf("%d uncorrectable syndromes, want 3", bad)
	}
}

// airCCHVectors are control channels as transmitted (36 dibits each).
// DSD-FME's own routines decode every one with a clean Hamming result, a
// good CRC7 and the fields listed.
var airCCHVectors = []struct {
	onAir string
	cch   AirCCH
}{
	{"233321032133130003020021223030032333", AirCCH{FrameNumber: 0, IDHalf: 0x123, Mode: 0, Version: 0, Format: 1}},
	{"333021033233131303021221023231033132", AirCCH{FrameNumber: 1, IDHalf: 0x456, Mode: 0, Version: 0, Format: 1}},
	{"033220332033230302221021322331132032", AirCCH{FrameNumber: 2, IDHalf: 0xABC, Mode: 0, Version: 0, Format: 1}},
	{"033120332232131002220221323131332032", AirCCH{FrameNumber: 3, IDHalf: 0xDEF, Mode: 0, Version: 0, Format: 1}},
	{"010233313021230302232000320300111012", AirCCH{FrameNumber: 2, IDHalf: 0xABC, Mode: 1, Version: 1, Format: 1, SlowData: 0x2A5A5}},
	{"012101332212111023321220303031212012", AirCCH{FrameNumber: 3, IDHalf: 0xDEF, Mode: 5, Version: 2, Format: 2, Emergency: true, SlowData: 0x00001}},
	{"220212300200323030233313322003112012", AirCCH{FrameNumber: 0, IDHalf: 0xFFF, Mode: 2, Version: 3, Format: 3, Emergency: true, SlowData: 0x3FFFF}},
	{"320112000101022331032113122002312112", AirCCH{FrameNumber: 1, IDHalf: 0x000, Mode: 2, Version: 3, Format: 3, Emergency: true, SlowData: 0x3FFFF}},
	{"333320033133011303020021103231013213", AirCCH{}},
}

func TestAirCCHReferenceVectors(t *testing.T) {
	for i, v := range airCCHVectors {
		got, ok := DecodeAirCCH(dibitsOf(v.onAir))
		if !ok {
			t.Errorf("vector %d: not decoded", i)
			continue
		}
		if got != v.cch {
			t.Errorf("vector %d: %+v, want %+v", i, got, v.cch)
		}
		if enc := EncodeAirCCH(v.cch); !sameDibits(enc, dibitsOf(v.onAir)) {
			t.Errorf("vector %d: encoder sends %v", i, enc)
		}
	}
	if !airCCHVectors[0].cch.Voice() || !airCCHVectors[4].cch.Voice() || !airCCHVectors[5].cch.Voice() {
		t.Errorf("modes 0, 1 and 5 are voice")
	}
	if airCCHVectors[6].cch.Voice() {
		t.Errorf("mode 2 is data")
	}
}

// TestAirCCHInterleaveSpreadsABurst: the 12 x 6 matrix puts consecutive
// on-air bits in different Hamming blocks, so a burst of six wrong bits
// costs each block one error and the control channel still decodes.
func TestAirCCHInterleaveSpreadsABurst(t *testing.T) {
	want := airCCHVectors[4].cch
	for start := 0; start+6 <= 72; start++ {
		d := dibitsOf(airCCHVectors[4].onAir)
		for k := start; k < start+6; k++ {
			d[k/2] ^= 1 << uint(1-k%2)
		}
		got, ok := DecodeAirCCH(d)
		if !ok {
			t.Fatalf("burst at bit %d: not decoded", start)
		}
		if got.Corrected != 6 {
			t.Fatalf("burst at bit %d: %d corrections, want 6", start, got.Corrected)
		}
		got.Corrected = 0
		if got != want {
			t.Fatalf("burst at bit %d: %+v", start, got)
		}
	}
	// Two errors in one block are beyond the code.
	d := dibitsOf(airCCHVectors[4].onAir)
	d[0] ^= 2 // on-air bit 0: block 0
	d[3] ^= 2 // on-air bit 6: block 0 again
	if _, ok := DecodeAirCCH(d); ok {
		t.Errorf("two errors in one block decoded")
	}
}

// TestAirCCHRejectsNoise: a 7-bit checksum behind a code that "corrects"
// most random blocks passes noise now and then. The framer therefore asks
// for more than a checksum before it locks; this measures what the
// checksum alone lets through.
func TestAirCCHRejectsNoise(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	const trials = 200_000
	pass, clean := 0, 0
	d := make([]uint8, AirCCHDibits)
	for n := 0; n < trials; n++ {
		for i := range d {
			d[i] = uint8(r.Intn(4))
		}
		if cch, ok := DecodeAirCCH(d); ok {
			pass++
			if cch.Corrected <= airLockCorrectedMax {
				clean++
			}
		}
	}
	// (13/16)^6 / 128 = 0.22 % pass the checksum; almost none of those
	// needed two corrections or fewer.
	if rate := float64(pass) / trials; rate < 0.001 || rate > 0.004 {
		t.Errorf("checksum passed %.3f %% of noise fields, expected about 0.22 %%", 100*rate)
	}
	if clean > 3 {
		t.Errorf("%d of %d noise fields passed with at most %d corrections", clean, trials, airLockCorrectedMax)
	}
}

func TestAirChannelCode(t *testing.T) {
	// 57 5F 77: twelve outer symbols, signs 000100110101.
	d := wordToDibits(0x575F77, AirSyncDibits)
	code, outer := AirChannelCode(d)
	if code != 0x575F77 || outer != 12 {
		t.Fatalf("code %06X outer %d", code, outer)
	}
	// A symbol received as an inner one keeps its sign.
	d[3] &^= 1
	if code, outer = AirChannelCode(d); code != 0x575F77 || outer != 11 {
		t.Fatalf("after one inner symbol: code %06X outer %d", code, outer)
	}
}
