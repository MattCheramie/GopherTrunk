package phase1

import "testing"

func TestEncryptionSyncRoundTrip(t *testing.T) {
	in := EncryptionSync{
		MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		AlgorithmID:      0x84, // AES-256
		KeyID:            0xBEEF,
	}
	got, errs, err := ParseEncryptionSync(AssembleEncryptionSync(in))
	if err != nil {
		t.Fatalf("ParseEncryptionSync: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean ES corrected %d errors", errs)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
	if !got.Encrypted() {
		t.Error("AES-256 ES should report Encrypted")
	}
}

func TestEncryptionSyncClearCall(t *testing.T) {
	in := EncryptionSync{AlgorithmID: AlgorithmClear}
	got, _, err := ParseEncryptionSync(AssembleEncryptionSync(in))
	if err != nil {
		t.Fatalf("ParseEncryptionSync: %v", err)
	}
	if got.Encrypted() {
		t.Error("clear-voice ES should not report Encrypted")
	}
}

func TestEncryptionSyncCorrectsBitError(t *testing.T) {
	in := EncryptionSync{
		MessageIndicator: [9]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33},
		AlgorithmID:      0x81,
		KeyID:            0x1234,
	}
	blocks := AssembleEncryptionSync(in)
	blocks[1][7] ^= 1 // one correctable bit error within a Hamming codeword
	got, errs, err := ParseEncryptionSync(blocks)
	if err != nil {
		t.Fatalf("ParseEncryptionSync: %v", err)
	}
	if errs < 1 {
		t.Errorf("corrected %d errors, want >= 1", errs)
	}
	if got != in {
		t.Errorf("ES after correction = %+v, want %+v", got, in)
	}
}

// TestEncryptionSyncRSCorrection confirms the outer RS(24,16,9) layer
// recovers the correct algorithm/key after up to t=4 symbol errors — the
// residual corruption that, before the RS layer, produced the scatter of
// garbage Algorithm IDs seen in the field capture.
func TestEncryptionSyncRSCorrection(t *testing.T) {
	es := EncryptionSync{
		MessageIndicator: [9]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
		AlgorithmID:      0x84, // AES-256
		KeyID:            0xBEEF,
	}
	clean := AssembleEncryptionSync(es)
	for nerr := 0; nerr <= 4; nerr++ {
		positions := make([]int, nerr)
		for i := range positions {
			positions[i] = i * 3
		}
		got, _, err := ParseEncryptionSync(injectRSSymbolErrors(clean, positions))
		if err != nil {
			t.Fatalf("nerr %d: unexpected error %v", nerr, err)
		}
		if got != es {
			t.Fatalf("nerr %d: recovered %+v, want %+v", nerr, got, es)
		}
	}
}

// TestEncryptionSyncRSUncorrectable confirms a >t=4 burst is flagged.
func TestEncryptionSyncRSUncorrectable(t *testing.T) {
	es := EncryptionSync{AlgorithmID: 0x81, KeyID: 0x1234}
	clean := AssembleEncryptionSync(es)
	got, _, err := ParseEncryptionSync(injectRSSymbolErrors(clean, []int{0, 2, 4, 6, 8}))
	if err == nil {
		if got == es {
			t.Fatalf("5 symbol errors 'recovered' the original — impossible within t=4")
		}
		return
	}
	if err != ErrEncryptionSyncUncorrectable {
		t.Fatalf("unexpected error %v, want ErrEncryptionSyncUncorrectable", err)
	}
}
