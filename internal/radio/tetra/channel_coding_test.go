package tetra

import (
	"math/rand"
	"testing"
)

// fillBits populates a bit slice from a deterministic PRNG so the
// per-channel round-trip tests exercise non-trivial content.
func fillBits(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.Intn(2))
	}
	return out
}

func TestEncodeDecodeSCHHDRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		seed       int64
		colourCode uint32
	}{
		{"random colour 0", 1, 0},
		{"random colour 0x1234", 2, 0x1234},
		{"random full 30-bit colour", 3, 0x3FFFFFFF},
		{"alternating bits, colour 0xABCDE", 4, 0xABCDE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := fillBits(124, tc.seed)
			type5 := EncodeSCHHD(info, tc.colourCode)
			if len(type5) != 216 {
				t.Fatalf("EncodeSCHHD produced %d bits, want 216", len(type5))
			}
			recovered, ok := DecodeSCHHD(type5, tc.colourCode)
			if !ok {
				t.Fatalf("DecodeSCHHD: CRC fail on clean round-trip")
			}
			for i, want := range info {
				if recovered[i] != want {
					t.Errorf("bit %d: got %d, want %d", i, recovered[i], want)
					break
				}
			}
		})
	}
}

func TestEncodeDecodeSCHFRoundTrip(t *testing.T) {
	info := fillBits(268, 5)
	type5 := EncodeSCHF(info, 0xDEADBEEF&0x3FFFFFFF)
	if len(type5) != 432 {
		t.Fatalf("EncodeSCHF produced %d bits, want 432", len(type5))
	}
	recovered, ok := DecodeSCHF(type5, 0xDEADBEEF&0x3FFFFFFF)
	if !ok {
		t.Fatalf("DecodeSCHF: CRC fail on clean round-trip")
	}
	for i, want := range info {
		if recovered[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, recovered[i], want)
			break
		}
	}
}

func TestEncodeDecodeSCHHURoundTrip(t *testing.T) {
	info := fillBits(92, 6)
	type5 := EncodeSCHHU(info, 0x7777)
	if len(type5) != 168 {
		t.Fatalf("EncodeSCHHU produced %d bits, want 168", len(type5))
	}
	recovered, ok := DecodeSCHHU(type5, 0x7777)
	if !ok {
		t.Fatalf("DecodeSCHHU: CRC fail on clean round-trip")
	}
	for i, want := range info {
		if recovered[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, recovered[i], want)
			break
		}
	}
}

func TestEncodeDecodeBSCHRoundTrip(t *testing.T) {
	info := fillBits(60, 7)
	type5 := EncodeBSCH(info)
	if len(type5) != 120 {
		t.Fatalf("EncodeBSCH produced %d bits, want 120", len(type5))
	}
	recovered, ok := DecodeBSCH(type5)
	if !ok {
		t.Fatalf("DecodeBSCH: CRC fail on clean round-trip")
	}
	for i, want := range info {
		if recovered[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, recovered[i], want)
			break
		}
	}
}

func TestEncodeDecodeAACHRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		seed       int64
		colourCode uint32
	}{
		{"colour 0", 8, 0},
		{"colour 0x12345", 9, 0x12345},
		{"colour 0x3FFFFFFF", 10, 0x3FFFFFFF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := fillBits(14, tc.seed)
			type5 := EncodeAACH(info, tc.colourCode)
			if len(type5) != 30 {
				t.Fatalf("EncodeAACH produced %d bits, want 30", len(type5))
			}
			recovered, errs := DecodeAACH(type5, tc.colourCode)
			if errs != 0 {
				t.Errorf("DecodeAACH: errs = %d, want 0 on clean round-trip", errs)
			}
			for i, want := range info {
				if recovered[i] != want {
					t.Errorf("bit %d: got %d, want %d", i, recovered[i], want)
					break
				}
			}
		})
	}
}

// TestDecodeSCHHDDetectsCRCError: flip enough bits in the type-5
// stream to overwhelm the inner Viterbi correction radius, and
// confirm the decoder reports CRC failure (not a silent pass).
// The K=5 R=2/3 RCPC + (216, 101) interleave is fairly robust —
// 3 bit flips often correct cleanly, so this test bumps to many
// adjacent flips to defeat the FEC and force the CRC check to
// observe corruption.
func TestDecodeSCHHDDetectsCRCError(t *testing.T) {
	info := fillBits(124, 11)
	type5 := EncodeSCHHD(info, 0x55555)
	// Flip 30 adjacent bits — guaranteed to overwhelm both the
	// Viterbi corrector and the post-deinterleave error spread.
	corrupted := append([]byte{}, type5...)
	for i := 50; i < 80; i++ {
		corrupted[i] ^= 1
	}
	_, ok := DecodeSCHHD(corrupted, 0x55555)
	if ok {
		t.Errorf("DecodeSCHHD reported CRC pass on a heavily-corrupted stream")
	}
}

// TestDecodeSCHHDWrongColourCodeFails: a stream scrambled with one
// colour code descrambled with a different one produces garbage at
// the deinterleave stage, which should fail the CRC.
func TestDecodeSCHHDWrongColourCodeFails(t *testing.T) {
	info := fillBits(124, 12)
	type5 := EncodeSCHHD(info, 0xAAAA)
	_, ok := DecodeSCHHD(type5, 0x5555)
	if ok {
		t.Errorf("DecodeSCHHD reported CRC pass with wrong colour code")
	}
}

// TestDecodeSCHHDCorrectsSingleBitError: a single bit flip in the
// type-5 stream should be corrected by the Viterbi inner decoder
// without breaking the CRC.
func TestDecodeSCHHDCorrectsSingleBitError(t *testing.T) {
	info := fillBits(124, 13)
	type5 := EncodeSCHHD(info, 0x1B2D4F)
	corrupted := append([]byte{}, type5...)
	corrupted[100] ^= 1
	recovered, ok := DecodeSCHHD(corrupted, 0x1B2D4F)
	if !ok {
		t.Errorf("DecodeSCHHD: CRC fail after single-bit correction; want pass")
	}
	for i, want := range info {
		if recovered[i] != want {
			t.Errorf("bit %d: got %d, want %d (single-bit correction)", i, recovered[i], want)
			break
		}
	}
}

// TestEncodeFunctionsRejectWrongSize: each EncodeXxx should return
// nil when handed the wrong input length.
func TestEncodeFunctionsRejectWrongSize(t *testing.T) {
	if got := EncodeSCHHD(make([]byte, 100), 0); got != nil {
		t.Errorf("EncodeSCHHD accepted 100-bit input, want nil")
	}
	if got := EncodeSCHF(make([]byte, 200), 0); got != nil {
		t.Errorf("EncodeSCHF accepted 200-bit input, want nil")
	}
	if got := EncodeSCHHU(make([]byte, 50), 0); got != nil {
		t.Errorf("EncodeSCHHU accepted 50-bit input, want nil")
	}
	if got := EncodeBSCH(make([]byte, 50)); got != nil {
		t.Errorf("EncodeBSCH accepted 50-bit input, want nil")
	}
	if got := EncodeAACH(make([]byte, 13), 0); got != nil {
		t.Errorf("EncodeAACH accepted 13-bit input, want nil")
	}
}

// TestCRCTetraK1Plus16AllZerosAllOnes provides a sanity anchor for
// the bit-level CRC implementation against two known inputs.
func TestCRCTetraK1Plus16AllZerosAllOnes(t *testing.T) {
	// CRC of all-zero K1-bit input with init=0xFFFF, no XOR-after
	// step would give a deterministic value; the init=0xFFFF +
	// XOR=0xFFFF spec form complements both ends.
	zeros := make([]byte, 124)
	crc0 := crcTetraK1Plus16(zeros)
	if crc0 == 0 {
		// 0xFFFF init shifted through with all-zero input yields a
		// non-zero residue; final XOR with 0xFFFF complements.
		// Confirm the result is non-zero (sanity, no specific value).
		t.Errorf("crcTetraK1Plus16(all zeros) = 0, expected non-zero residue")
	}
	ones := make([]byte, 124)
	for i := range ones {
		ones[i] = 1
	}
	crc1 := crcTetraK1Plus16(ones)
	if crc0 == crc1 {
		t.Errorf("crcTetraK1Plus16 yields same value for all-zero and all-one inputs (%#x)", crc0)
	}
}
