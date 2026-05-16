package framing

import (
	"math/rand"
	"testing"
)

func TestRM3014TetraRoundTripCleanCodeword(t *testing.T) {
	cases := [][]byte{
		make([]byte, 14), // all zeros
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, // all ones
		{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0}, // alternating
		{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}, // alternating opposite
		{1, 0, 0, 1, 1, 0, 1, 1, 0, 1, 0, 0, 1, 1}, // pattern
	}
	for _, info := range cases {
		cw := EncodeRM3014Tetra(info)
		if len(cw) != 30 {
			t.Fatalf("EncodeRM3014Tetra produced %d bits, want 30", len(cw))
		}
		got, errs := DecodeRM3014Tetra(cw)
		if errs != 0 {
			t.Errorf("clean codeword: errs = %d, want 0 (info=%v)", errs, info)
		}
		for i, want := range info {
			if got[i] != want {
				t.Errorf("info[%d]: got %d, want %d (input=%v)", i, got[i], want, info)
			}
		}
	}
}

func TestRM3014TetraEncoderIsSystematic(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 0}
	cw := EncodeRM3014Tetra(info)
	// First 14 bits of the codeword must equal the 14 info bits.
	for i := 0; i < 14; i++ {
		if cw[i] != info[i] {
			t.Errorf("systematic prefix[%d] = %d, want %d", i, cw[i], info[i])
		}
	}
}

func TestRM3014TetraCorrectsSingleBitError(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 0}
	cw := EncodeRM3014Tetra(info)
	for pos := 0; pos < 30; pos++ {
		corrupted := append([]byte{}, cw...)
		corrupted[pos] ^= 1
		got, errs := DecodeRM3014Tetra(corrupted)
		if errs != 1 {
			t.Errorf("pos=%d: errs=%d, want 1", pos, errs)
			continue
		}
		for i, want := range info {
			if got[i] != want {
				t.Errorf("pos=%d info[%d]: got %d, want %d", pos, i, got[i], want)
			}
		}
	}
}

func TestRM3014TetraRejectsWrongLength(t *testing.T) {
	if cw := EncodeRM3014Tetra(make([]byte, 13)); cw != nil {
		t.Errorf("EncodeRM3014Tetra accepted 13-bit input, want nil")
	}
	if cw := EncodeRM3014Tetra(make([]byte, 15)); cw != nil {
		t.Errorf("EncodeRM3014Tetra accepted 15-bit input, want nil")
	}
	if info, errs := DecodeRM3014Tetra(make([]byte, 29)); info != nil || errs != -1 {
		t.Errorf("DecodeRM3014Tetra accepted 29-bit input, got (%v, %d)", info, errs)
	}
}

func TestRM3014TetraRandomRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(0xAACC))
	for trial := 0; trial < 64; trial++ {
		info := make([]byte, 14)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		cw := EncodeRM3014Tetra(info)
		got, errs := DecodeRM3014Tetra(cw)
		if errs != 0 {
			t.Errorf("trial %d clean decode: errs = %d, want 0", trial, errs)
			continue
		}
		for i, want := range info {
			if got[i] != want {
				t.Errorf("trial %d info[%d]: got %d, want %d", trial, i, got[i], want)
			}
		}
	}
}

// TestRM3014TetraParityMatrixSanity asserts the generator matrix
// columns produce non-trivial parity for every input bit — protects
// against accidental drift in the matrix transcription. With a
// single-bit input vector, the resulting codeword should have its
// systematic position set plus whatever parity columns the matrix
// row populates.
func TestRM3014TetraParityMatrixSanity(t *testing.T) {
	for row := 0; row < 14; row++ {
		info := make([]byte, 14)
		info[row] = 1
		cw := EncodeRM3014Tetra(info)
		// Systematic position
		if cw[row] != 1 {
			t.Errorf("row %d: systematic bit not set", row)
		}
		// Parity bits must equal the parity matrix row
		for c := 0; c < 16; c++ {
			if cw[14+c] != rm3014ParityMatrix[row][c] {
				t.Errorf("row %d parity column %d: got %d, want %d", row, c, cw[14+c], rm3014ParityMatrix[row][c])
			}
		}
	}
}
