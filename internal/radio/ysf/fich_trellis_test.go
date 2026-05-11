package ysf

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

func TestFICHTrellisRoundTrip(t *testing.T) {
	// Build a representative FICH (Header / Group / VDMode2),
	// assemble it into the 6-octet info+CRC buffer, run the
	// Trellis encoder, then decode and confirm we recover the
	// same 48 bits with metric=0.
	original := FICH{
		FrameType:   FrameTypeHeader,
		CallType:    CallTypeGroup,
		BlockNumber: 0,
		BlockTotal:  3,
		FrameNumber: 0,
		FrameTotal:  7,
		DataType:    DataTypeVDMode2,
		VoIP:        false,
		SquelchMode: true,
		SquelchCode: 42,
		Device:      0,
	}
	infoOctets := AssembleFICH(original)
	infoBits := UnpackBits(infoOctets)
	if len(infoBits) != FICHInfoBits {
		t.Fatalf("infoBits = %d, want %d", len(infoBits), FICHInfoBits)
	}

	channel, err := EncodeFICHTrellis(infoBits)
	if err != nil {
		t.Fatalf("EncodeFICHTrellis: %v", err)
	}
	if len(channel) != FICHChannelBits {
		t.Errorf("channel bits = %d, want %d", len(channel), FICHChannelBits)
	}

	recovered, metric, err := DecodeFICHTrellis(channel)
	if err != nil {
		t.Fatalf("DecodeFICHTrellis: %v", err)
	}
	if metric != 0 {
		t.Errorf("metric = %d, want 0 (clean round trip)", metric)
	}
	for i := range infoBits {
		if recovered[i] != infoBits[i] {
			t.Fatalf("bit %d differs: got %d want %d", i, recovered[i], infoBits[i])
		}
	}

	// Pack the recovered bits back into octets and feed them
	// through ParseFICH to confirm CRC + field layout survives.
	roundtripOctets := PackBits(recovered)
	parsed, err := ParseFICH(roundtripOctets)
	if err != nil {
		t.Fatalf("ParseFICH after Trellis decode: %v", err)
	}
	if parsed.FrameType != original.FrameType {
		t.Errorf("FrameType = %v, want %v", parsed.FrameType, original.FrameType)
	}
	if parsed.SquelchCode != original.SquelchCode {
		t.Errorf("SquelchCode = %d, want %d", parsed.SquelchCode, original.SquelchCode)
	}
}

func TestFICHTrellisCorrectsSingleBitError(t *testing.T) {
	// K=5 ½-rate is rate-1/2 with free distance 7 — the Viterbi
	// decoder reliably corrects up to 3 bit errors per stage. We
	// test the easy case (1 error) for confidence; deeper
	// stress lives in the framing.viterbi tests.
	original := FICH{
		FrameType: FrameTypeComms,
		CallType:  CallTypeGroup,
		DataType:  DataTypeVoiceFR,
	}
	infoBits := UnpackBits(AssembleFICH(original))
	channel, err := EncodeFICHTrellis(infoBits)
	if err != nil {
		t.Fatal(err)
	}
	channel[10] ^= 1

	recovered, metric, err := DecodeFICHTrellis(channel)
	if err != nil {
		t.Fatal(err)
	}
	if metric == 0 {
		t.Errorf("metric = 0 with injected error; expected nonzero penalty")
	}
	for i := range infoBits {
		if recovered[i] != infoBits[i] {
			t.Fatalf("bit %d not corrected: got %d want %d", i, recovered[i], infoBits[i])
		}
	}
}

func TestFICHTrellisRejectsWrongLength(t *testing.T) {
	if _, _, err := DecodeFICHTrellis(make([]byte, 50)); err == nil {
		t.Errorf("expected error for short buffer, got nil")
	}
	if _, err := EncodeFICHTrellis(make([]byte, 32)); err == nil {
		t.Errorf("expected error for short info, got nil")
	}
}

func TestPackUnpackBitsRoundTrip(t *testing.T) {
	// PackBits / UnpackBits are tiny but load-bearing helpers —
	// ensure they round-trip cleanly across odd byte values.
	octets := []byte{0x00, 0xFF, 0xA5, 0x5A, 0x12, 0x34}
	bits := UnpackBits(octets)
	if len(bits) != len(octets)*8 {
		t.Fatalf("UnpackBits len = %d, want %d", len(bits), len(octets)*8)
	}
	got := PackBits(bits)
	for i := range octets {
		if got[i] != octets[i] {
			t.Errorf("octet %d: got %#x want %#x", i, got[i], octets[i])
		}
	}
}

func TestDepunctureMarkSurvivesViterbi(t *testing.T) {
	// Sanity: confirm the framing primitive recognises the
	// DepunctureMark sentinel even when callers feed it inline.
	infoBits := make([]byte, FICHInfoBits)
	for i := range infoBits {
		infoBits[i] = byte((i * 5) % 2)
	}
	channel, err := EncodeFICHTrellis(infoBits)
	if err != nil {
		t.Fatal(err)
	}
	// Mark a couple of bits as "no info" (simulating puncturing).
	channel[0] = framing.DepunctureMark
	channel[7] = framing.DepunctureMark
	recovered, _, err := DecodeFICHTrellis(channel)
	if err != nil {
		t.Fatal(err)
	}
	for i := range infoBits {
		if recovered[i] != infoBits[i] {
			t.Fatalf("bit %d: got %d want %d (mark should be cost-free)",
				i, recovered[i], infoBits[i])
		}
	}
}
