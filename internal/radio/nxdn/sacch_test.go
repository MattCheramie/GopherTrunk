package nxdn

import (
	"bytes"
	"math/rand"
	"testing"
)

func makeRandomPayload(seed int64, n int) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.Intn(2))
	}
	return out
}

func TestSACCHRoundTrip(t *testing.T) {
	payload := makeRandomPayload(1, 26)
	info := AppendSACCHCRC6(payload)
	channel := EncodeSACCH(info)
	if len(channel) != SACCHChannelBits {
		t.Fatalf("encoded len = %d, want %d", len(channel), SACCHChannelBits)
	}
	got, metric, err := DecodeSACCH(channel)
	if err != nil {
		t.Fatalf("DecodeSACCH: %v", err)
	}
	if metric != 0 {
		t.Errorf("clean channel metric = %d, want 0", metric)
	}
	if !bytes.Equal(got, info) {
		t.Errorf("round-trip mismatch:\n got %v\nwant %v", got, info)
	}
}

func TestSACCHRoundTripExhaustivePayloads(t *testing.T) {
	for seed := int64(0); seed < 16; seed++ {
		info := AppendSACCHCRC6(makeRandomPayload(seed, 26))
		channel := EncodeSACCH(info)
		got, _, err := DecodeSACCH(channel)
		if err != nil {
			t.Errorf("seed=%d: %v", seed, err)
			continue
		}
		if !bytes.Equal(got, info) {
			t.Errorf("seed=%d: round-trip failed", seed)
		}
	}
}

func TestSACCHCorrectsSingleBitErrors(t *testing.T) {
	info := AppendSACCHCRC6(makeRandomPayload(7, 26))
	for bit := 0; bit < SACCHChannelBits; bit++ {
		channel := EncodeSACCH(info)
		channel[bit] ^= 1
		got, metric, err := DecodeSACCH(channel)
		if err != nil {
			t.Errorf("bit=%d: %v", bit, err)
			continue
		}
		if metric == 0 {
			t.Errorf("bit=%d: metric = 0 after error, want > 0", bit)
		}
		if !bytes.Equal(got, info) {
			t.Errorf("bit=%d: error correction failed", bit)
		}
	}
}

func TestSACCHCorrectsScatteredErrors(t *testing.T) {
	info := AppendSACCHCRC6(makeRandomPayload(13, 26))
	channel := EncodeSACCH(info)
	channel[7] ^= 1
	channel[27] ^= 1
	channel[51] ^= 1
	got, metric, err := DecodeSACCH(channel)
	if err != nil {
		t.Fatalf("DecodeSACCH: %v", err)
	}
	if metric == 0 {
		t.Errorf("metric = 0 after errors, want > 0")
	}
	if !bytes.Equal(got, info) {
		t.Errorf("3-bit error correction failed")
	}
}

func TestSACCHRejectsHeavyCorruption(t *testing.T) {
	info := AppendSACCHCRC6(makeRandomPayload(21, 26))
	channel := EncodeSACCH(info)
	for i := range channel {
		channel[i] ^= 1
	}
	_, _, err := DecodeSACCH(channel)
	if err == nil {
		t.Fatal("heavy corruption should surface an error")
	}
}

func TestSACCHInterleaverIsBijection(t *testing.T) {
	var seen [SACCHChannelBits]bool
	for _, j := range sacchInterleavePerm {
		if j < 0 || j >= SACCHChannelBits {
			t.Fatalf("perm entry out of range: %d", j)
		}
		if seen[j] {
			t.Fatalf("perm entry %d duplicated", j)
		}
		seen[j] = true
	}
}

func TestSACCHPunctureBijection(t *testing.T) {
	// Every puncture position is unique, in range, and ascending.
	prev := -1
	for _, p := range sacchPuncturePositions {
		if p <= prev {
			t.Errorf("puncture positions not strictly ascending: %d after %d", p, prev)
		}
		if p < 0 || p >= 80 {
			t.Errorf("puncture position out of range: %d", p)
		}
		prev = p
	}
}

func TestSACCHCRC6RoundTrip(t *testing.T) {
	// Register preset 0x3F (MMDVMHost CNXDNCRC::createCRC6, DSD-FME crc6):
	// an all-zero payload has a NON-zero CRC, 0x2B. The earlier zero-preset
	// register gave 0 here and so "verified" every all-zero block — the
	// only blocks it verified on the issue #1261 real-air captures.
	zero := make([]byte, 26)
	if got := SACCHCRC6(zero, 26); got != 0x2B {
		t.Errorf("CRC of all-zero = %#x, want 0x2b", got)
	}
	for i := 0; i < 26; i++ {
		bit := make([]byte, 26)
		bit[i] = 1
		if got := SACCHCRC6(bit, 26); got == 0 {
			t.Errorf("CRC of single-bit payload at %d = 0; want non-zero", i)
		}
	}
}

func TestVerifySACCHCRC6(t *testing.T) {
	info := AppendSACCHCRC6(makeRandomPayload(31, 26))
	if !VerifySACCHCRC6(info) {
		t.Fatal("VerifySACCHCRC6 rejected a freshly-encoded info block")
	}
	info[5] ^= 1
	if VerifySACCHCRC6(info) {
		t.Fatal("VerifySACCHCRC6 accepted a payload with a flipped bit")
	}
}
