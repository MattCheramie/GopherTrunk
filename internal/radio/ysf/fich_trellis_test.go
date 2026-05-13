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

func TestFICHOnAirRoundTrip(t *testing.T) {
	// 48 info bits → encode (trellis + puncture + interleave) →
	// decode (deinterleave + depuncture + Viterbi) → recover the same
	// info bits with metric 0.
	original := FICH{
		FrameType:   FrameTypeHeader,
		CallType:    CallTypeGroup,
		BlockNumber: 1,
		BlockTotal:  3,
		FrameNumber: 4,
		FrameTotal:  7,
		DataType:    DataTypeVDMode2,
		SquelchMode: true,
		SquelchCode: 17,
	}
	infoBits := UnpackBits(AssembleFICH(original))

	onAir, err := EncodeFICHOnAir(infoBits)
	if err != nil {
		t.Fatalf("EncodeFICHOnAir: %v", err)
	}
	if len(onAir) != FICHOnAirBits {
		t.Fatalf("on-air bits = %d, want %d", len(onAir), FICHOnAirBits)
	}

	recovered, metric, err := DecodeFICHOnAir(onAir)
	if err != nil {
		t.Fatalf("DecodeFICHOnAir: %v", err)
	}
	if metric != 0 {
		t.Errorf("metric = %d, want 0 (clean round trip through on-air codec)", metric)
	}
	for i := range infoBits {
		if recovered[i] != infoBits[i] {
			t.Fatalf("bit %d differs: got %d want %d", i, recovered[i], infoBits[i])
		}
	}

	// Confirm the recovered info still passes ParseFICH end-to-end.
	parsed, err := ParseFICH(PackBits(recovered))
	if err != nil {
		t.Fatalf("ParseFICH after on-air round-trip: %v", err)
	}
	if parsed.SquelchCode != original.SquelchCode {
		t.Errorf("SquelchCode = %d, want %d", parsed.SquelchCode, original.SquelchCode)
	}
}

func TestFICHOnAirRecoversFromSingleBitFlip(t *testing.T) {
	// Flip one on-air bit and confirm the Viterbi corrects through
	// the interleaver/puncture stages. Loop over every on-air bit
	// index so we don't accidentally pick the one position the
	// decoder can't repair.
	original := FICH{
		FrameType: FrameTypeComms,
		CallType:  CallTypeGroup,
		DataType:  DataTypeVoiceFR,
	}
	infoBits := UnpackBits(AssembleFICH(original))
	onAir, err := EncodeFICHOnAir(infoBits)
	if err != nil {
		t.Fatal(err)
	}

	failures := 0
	for pos := 0; pos < FICHOnAirBits; pos++ {
		corrupted := make([]byte, FICHOnAirBits)
		copy(corrupted, onAir)
		corrupted[pos] ^= 1
		recovered, _, err := DecodeFICHOnAir(corrupted)
		if err != nil {
			t.Fatalf("pos %d: DecodeFICHOnAir: %v", pos, err)
		}
		ok := true
		for i := range infoBits {
			if recovered[i] != infoBits[i] {
				ok = false
				break
			}
		}
		if !ok {
			failures++
		}
	}
	// K=5 ½-rate has free distance 7 — every single-bit error is
	// well within correction capacity.
	if failures != 0 {
		t.Errorf("%d / %d single-bit positions not corrected by Viterbi", failures, FICHOnAirBits)
	}
}

func TestFICHInterleavePermBijective(t *testing.T) {
	// Every depunctured-bit index (0..99) must appear exactly once in
	// the interleave permutation, otherwise on-air bits double-tap or
	// drop on the wire.
	seen := make(map[int]bool, FICHOnAirBits)
	for _, idx := range fichInterleavePerm {
		if idx < 0 || idx >= FICHOnAirBits {
			t.Errorf("permutation entry %d out of range [0, %d)", idx, FICHOnAirBits)
		}
		if seen[idx] {
			t.Errorf("duplicate permutation entry %d", idx)
		}
		seen[idx] = true
	}
	if len(seen) != FICHOnAirBits {
		t.Errorf("permutation covers %d unique entries, want %d", len(seen), FICHOnAirBits)
	}
}

func TestFICHPuncturePositionsExactly4(t *testing.T) {
	// Sanity check on the puncture-position table: exactly four
	// positions, all within the channel-bit range, strictly
	// increasing (the encoder/decoder loops rely on the ordering).
	if got := FICHChannelBits - FICHOnAirBits; got != len(fichPuncturePositions) {
		t.Errorf("puncture count = %d, want %d (FICHChannelBits - FICHOnAirBits)",
			len(fichPuncturePositions), got)
	}
	for i, p := range fichPuncturePositions {
		if p < 0 || p >= FICHChannelBits {
			t.Errorf("puncture position [%d] = %d out of range [0, %d)",
				i, p, FICHChannelBits)
		}
		if i > 0 && p <= fichPuncturePositions[i-1] {
			t.Errorf("puncture positions not strictly increasing at [%d]: %d <= %d",
				i, p, fichPuncturePositions[i-1])
		}
	}
}

func TestFICHOnAirRejectsWrongLength(t *testing.T) {
	if _, _, err := DecodeFICHOnAir(make([]byte, 50)); err == nil {
		t.Error("DecodeFICHOnAir accepted 50-bit buffer")
	}
	if _, _, err := DecodeFICHOnAir(make([]byte, 104)); err == nil {
		t.Error("DecodeFICHOnAir accepted 104-bit buffer (that's pre-puncture length)")
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
