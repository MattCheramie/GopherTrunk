package lora

import (
	"encoding/hex"
	"testing"
)

// Literal air-format vectors for the LoRa bit layer (#586).
//
// GopherTrunk's original bit layer was invented and self-consistent: its
// modulator and demodulator shared the same made-up whitening, CRC,
// header, Hamming layout and interleaver, so every round-trip test passed
// while no real Semtech frame could ever decode. These vectors come from
// two INDEPENDENT open implementations, each validated against real
// radios, transcribed separately and cross-checked against each other
// (2916 random SF/CR/CRC/LDRO/payload cases, identical chip sequences):
//
//   - gr-lora_sdr (tapparelj, EPFL; validated against SX1276): whitening
//     table, header checksum, add_crc, hamming_enc, interleaver,
//     gray_demap (TX: Gray decode then +1).
//   - jkadbear/LoRaPHY (validated against SX1278): encode().
//
// Nothing was ported; the references were read for the format only.
// The two disagree only on the CONTENT of padding codewords in a short
// final block (zero codewords vs 0xFF bytes), which no receiver reads;
// GopherTrunk pads with zero codewords like gr-lora_sdr, and every vector
// below is identical under both.

// TestWhiteningSequenceMatchesReference pins the first 48 whitening bytes
// (gr-lora_sdr tables.h whitening_seq / LoRaPHY whitening_seq).
func TestWhiteningSequenceMatchesReference(t *testing.T) {
	want := []byte{0xFF, 0xFE, 0xFC, 0xF8, 0xF0, 0xE1, 0xC2, 0x85, 0x0B, 0x17, 0x2F, 0x5E, 0xBC, 0x78, 0xF1, 0xE3,
		0xC6, 0x8D, 0x1A, 0x34, 0x68, 0xD0, 0xA0, 0x40, 0x80, 0x01, 0x02, 0x04, 0x08, 0x11, 0x23, 0x47,
		0x8E, 0x1C, 0x38, 0x71, 0xE2, 0xC4, 0x89, 0x12, 0x25, 0x4B, 0x97, 0x2E, 0x5C, 0xB8, 0x70, 0xE0}
	got := make([]byte, len(want))
	Whiten(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("whitening byte %d = 0x%02X, want 0x%02X (got %X)", i, got[i], want[i], got)
		}
	}
}

func TestHammingCodewordsMatchReference(t *testing.T) {
	want := map[int][16]uint16{
		1: {0x00, 0x11, 0x09, 0x18, 0x05, 0x14, 0x0C, 0x1D, 0x03, 0x12, 0x0A, 0x1B, 0x06, 0x17, 0x0F, 0x1E},
		2: {0x00, 0x22, 0x13, 0x31, 0x0B, 0x29, 0x18, 0x3A, 0x05, 0x27, 0x16, 0x34, 0x0E, 0x2C, 0x1D, 0x3F},
		3: {0x00, 0x45, 0x27, 0x62, 0x16, 0x53, 0x31, 0x74, 0x0B, 0x4E, 0x2C, 0x69, 0x1D, 0x58, 0x3A, 0x7F},
		4: {0x00, 0x8B, 0x4E, 0xC5, 0x2D, 0xA6, 0x63, 0xE8, 0x17, 0x9C, 0x59, 0xD2, 0x3A, 0xB1, 0x74, 0xFF},
	}
	for cr, cws := range want {
		for n, w := range cws {
			if got := HammingEncode4(uint8(n), cr); got != w {
				t.Errorf("cr=%d nibble=%X: codeword 0x%02X, want 0x%02X", cr, n, got, w)
			}
		}
	}
}

func TestPayloadCRCMatchesReference(t *testing.T) {
	cases := []struct {
		hex  string
		want uint16
	}{
		{"48656c6c6f", 0x8196}, // "Hello"
		{"deadbeef", 0xEC80},
		{"476f706865725472756e6b", 0x55C9}, // "GopherTrunk"
		{"000102030405060708090a0b0c0d0e0f10111213", 0xC900},
	}
	for _, c := range cases {
		p, _ := hex.DecodeString(c.hex)
		if got := PayloadCRC(p); got != c.want {
			t.Errorf("PayloadCRC(%s) = 0x%04X, want 0x%04X", c.hex, got, c.want)
		}
	}
}

// refFrames are complete data-symbol chip sequences (everything after the
// SFD) as both references produce them.
var refFrames = []struct {
	name    string
	payload string // hex
	sf, cr  int
	crc     bool
	ldro    bool
	chips   []uint16
}{
	{"deadbeef_sf7_cr1", "deadbeef", 7, 1, true, false,
		[]uint16{29, 49, 97, 1, 29, 17, 61, 101, 119, 51, 20, 1, 22, 2, 16, 5, 126, 50}},
	{"hello_sf8_cr4", "48656c6c6f", 8, 4, true, false,
		[]uint16{237, 65, 193, 253, 117, 221, 113, 1, 155, 130, 49, 173, 118, 198, 12, 127, 10, 2, 0, 196, 160, 208, 80, 40}},
	{"hello_sf9_cr2_nocrc", "48656c6c6f", 9, 2, false, false,
		[]uint16{369, 189, 473, 49, 245, 445, 229, 97, 179, 449, 25, 151, 395, 315}},
	{"gophertrunk_sf12_cr3_ldro", "476f706865725472756e6b", 12, 3, true, true,
		[]uint16{1517, 245, 997, 1745, 1861, 57, 2249, 2041, 2397, 1661, 781, 3277, 1893, 2869, 909, 3161, 4041, 2649, 1157, 2473, 1685, 93, 5, 1, 2045, 1, 1, 253, 125}},
	{"seq20_sf10_cr1", "000102030405060708090a0b0c0d0e0f10111213", 10, 1, true, false,
		[]uint16{677, 333, 861, 93, 853, 149, 789, 597, 106, 822, 682, 718, 813, 881, 231, 305, 776, 284, 87, 301, 341, 984, 482, 967, 33, 528, 642, 901, 1, 1, 512, 256, 1}},
	{"one_sf11_cr4_ldro_nocrc", "88", 11, 4, false, true,
		[]uint16{369, 189, 65, 2045, 493, 1537, 129, 445}},
}

// TestEncodeMatchesReferenceChips is the failing-first pin: the old,
// invented bit layer produced different chips for every one of these.
func TestEncodeMatchesReferenceChips(t *testing.T) {
	for _, c := range refFrames {
		p, _ := hex.DecodeString(c.payload)
		got := encodeFrameSymbols(p, c.sf, c.cr, c.crc, c.ldro)
		if len(got) != len(c.chips) {
			t.Errorf("%s: %d chips, want %d", c.name, len(got), len(c.chips))
			continue
		}
		if n := FrameSymbolCount(c.sf, c.cr, len(p), c.crc, c.ldro); n != len(c.chips) {
			t.Errorf("%s: FrameSymbolCount = %d, want %d", c.name, n, len(c.chips))
		}
		for i := range got {
			if got[i] != c.chips[i] {
				t.Errorf("%s: chip %d = %d, want %d\n got  %v\n want %v", c.name, i, got[i], c.chips[i], got, c.chips)
				break
			}
		}
	}
}

// TestDecodeReferenceChips decodes the reference chips: header fields,
// payload and CRC must all come out right.
func TestDecodeReferenceChips(t *testing.T) {
	for _, c := range refFrames {
		p, _ := hex.DecodeString(c.payload)
		f, ok := decodeFrameFromSymbols(c.chips, c.sf, c.ldro)
		if !ok || !f.HeaderOK || !f.CRCOK {
			t.Errorf("%s: ok=%v header=%v crc=%v", c.name, ok, f.HeaderOK, f.CRCOK)
			continue
		}
		if f.CR != c.cr || f.HasCRC != c.crc || string(f.Payload) != string(p) {
			t.Errorf("%s: cr=%d crc=%v payload=%x, want cr=%d crc=%v payload=%s", c.name, f.CR, f.HasCRC, f.Payload, c.cr, c.crc, c.payload)
		}
		if f.FECFixed != 0 || f.FECErrors != 0 {
			t.Errorf("%s: clean chips reported FEC fixed=%d errors=%d", c.name, f.FECFixed, f.FECErrors)
		}
	}
}

// TestDecodeCorrectsReferenceChipErrors flips one chip of a 4/8 frame by
// ±1 bin. A one-bin error changes one Gray-coded bit, i.e. one bit of one
// codeword, which 4/8 corrects; in the reduced-rate header block the
// rounding /4 absorbs it entirely.
func TestDecodeCorrectsReferenceChipErrors(t *testing.T) {
	c := refFrames[1] // hello_sf8_cr4
	n := uint16(1) << uint(c.sf)
	for i := range c.chips {
		for _, d := range []uint16{1, n - 1} {
			chips := append([]uint16(nil), c.chips...)
			chips[i] = (chips[i] + d) % n
			f, ok := decodeFrameFromSymbols(chips, c.sf, c.ldro)
			if !ok || !f.CRCOK || string(f.Payload) != "Hello" {
				t.Fatalf("chip %d %+d: ok=%v crc=%v payload=%q", i, int16(d), ok, f.CRCOK, f.Payload)
			}
		}
	}
}

// TestModulatedIQDemodulatesToReferenceChips checks the chirp layer on
// the same vectors: the modulator's IQ must demodulate to exactly the
// reference chip values (the symbol-v chirp is gr-lora_sdr's
// build_upchirp(id=v) up to a constant phase), and decode end to end.
func TestModulatedIQDemodulatesToReferenceChips(t *testing.T) {
	for _, c := range refFrames {
		p, _ := hex.DecodeString(c.payload)
		mode := LDROOff
		if c.ldro {
			mode = LDROOn
		}
		iq := pad(LoRaModulateMode(p, c.sf, c.cr, DefaultOversample, BW125, c.crc, 0x12, mode), 300, 300)
		m := NewDemodulatorMode(c.sf, DefaultOversample, BW125, mode)
		f, ok := m.DecodeFirst(iq)
		if !ok || !f.CRCOK || string(f.Payload) != string(p) {
			t.Errorf("%s: ok=%v crc=%v payload=%x", c.name, ok, f.CRCOK, f.Payload)
			continue
		}
		for i, v := range f.RawSymbols {
			if v != c.chips[i] {
				t.Errorf("%s: demodulated chip %d = %d, want %d", c.name, i, v, c.chips[i])
				break
			}
		}
	}
}
