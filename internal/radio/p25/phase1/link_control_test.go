package phase1

import "testing"

func TestHamming10_6RoundTrip(t *testing.T) {
	for v := 0; v < 64; v++ {
		data := make([]byte, 6)
		for i := range data {
			data[i] = byte(v>>uint(5-i)) & 1
		}
		cw := encodeHamming10_6(data)
		if len(cw) != 10 {
			t.Fatalf("codeword len = %d, want 10", len(cw))
		}
		got, errs := decodeHamming10_6(cw)
		if errs != 0 {
			t.Errorf("clean codeword for %d corrected %d errors", v, errs)
		}
		for i := range data {
			if got[i] != data[i] {
				t.Errorf("v=%d: data bit %d = %d, want %d", v, i, got[i], data[i])
			}
		}
	}
}

func TestHamming10_6CorrectsSingleError(t *testing.T) {
	data := []byte{1, 0, 1, 1, 0, 1}
	cw := encodeHamming10_6(data)
	for pos := 0; pos < 10; pos++ {
		bad := append([]byte(nil), cw...)
		bad[pos] ^= 1
		got, errs := decodeHamming10_6(bad)
		if errs != 1 {
			t.Errorf("flip at %d: corrected %d errors, want 1", pos, errs)
		}
		for i := range data {
			if got[i] != data[i] {
				t.Errorf("flip at %d: data bit %d not recovered", pos, i)
			}
		}
	}
}

func TestLinkControlRoundTrip(t *testing.T) {
	in := LinkControl{
		LCFormat:       0x44,
		ServiceOptions: 0xC0,
		TalkgroupID:    0x1234,
		SourceID:       0x00ABCD,
	}
	got, errs, err := ParseLinkControl(AssembleLinkControl(in))
	if err != nil {
		t.Fatalf("ParseLinkControl: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean LC corrected %d errors", errs)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestLinkControlCorrectsBitError(t *testing.T) {
	in := LinkControl{LCFormat: 0x00, ServiceOptions: 0x00, TalkgroupID: 0x5678, SourceID: 0x00BEEF}
	blocks := AssembleLinkControl(in)
	// One bit error per 40-bit block is within the per-codeword
	// Hamming(10,6,3) correction radius.
	blocks[0][3] ^= 1
	blocks[3][27] ^= 1
	got, errs, err := ParseLinkControl(blocks)
	if err != nil {
		t.Fatalf("ParseLinkControl: %v", err)
	}
	if errs < 2 {
		t.Errorf("corrected %d errors, want >= 2", errs)
	}
	if got != in {
		t.Errorf("LC after correction = %+v, want %+v", got, in)
	}
}

func TestParseLinkControlLengthError(t *testing.T) {
	var blocks [LDULCESBlockCount][]byte
	for i := range blocks {
		blocks[i] = make([]byte, LDULCESBlockBits)
	}
	blocks[2] = make([]byte, 39) // wrong length
	if _, _, err := ParseLinkControl(blocks); err == nil {
		t.Error("ParseLinkControl accepted a wrong-length block")
	}
}
