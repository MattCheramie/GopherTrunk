package dstar

import "testing"

// The null frame is a valid AMBE frame: under the right on-air schedule
// its C0 and its descrambled C1 are both exact Golay codewords. Under any
// other schedule (the sequential placeholder GT carried before #1261) the
// chance of that is about 2^-22.
func TestDVVoiceDecodesMMDVMNullFrameWithoutCorrection(t *testing.T) {
	bits := NullAMBEFrame()
	payload, errs, err := DecodeDVVoiceBits(bits)
	if err != nil {
		t.Fatal(err)
	}
	if errs != 0 {
		t.Fatalf("null AMBE frame needed %d Golay corrections, want 0", errs)
	}
	// The inverse reproduces the on-air bits exactly, overall-parity cell
	// included.
	enc, err := EncodeDVVoiceBits(payload)
	if err != nil {
		t.Fatal(err)
	}
	for i := range bits {
		if enc[i] != bits[i] {
			t.Fatalf("re-encoded null frame differs at bit %d", i)
		}
	}
}
