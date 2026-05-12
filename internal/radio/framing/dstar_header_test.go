package framing

import (
	"bytes"
	"testing"
)

// TestDStarHeaderFECRoundTrip encodes a known 41-byte header through
// the full FEC chain (conv + puncture + scramble + interleave) and
// decodes it back, verifying every byte round-trips.
func TestDStarHeaderFECRoundTrip(t *testing.T) {
	// Sample header: FLAG1=0, callsigns padded with spaces, CRC=0
	// (the framing package doesn't care about CRC validity — it
	// just round-trips the byte content).
	info := []byte{
		0x00, 0x00, 0x00, // FLAG1..3
		'K', 'D', '0', 'A', 'A', 'A', ' ', 'B', // RPT2
		'K', 'D', '0', 'A', 'A', 'A', ' ', 'G', // RPT1
		'C', 'Q', 'C', 'Q', 'C', 'Q', ' ', ' ', // UR
		'W', 'B', '7', 'X', 'Y', 'Z', ' ', ' ', // MY1
		'S', 'U', 'F', 'X', // MY2 (4 chars)
		0xAB, 0xCD, // CRC trailer
	}
	if len(info) != 41 {
		t.Fatalf("test info bytes: %d, want 41", len(info))
	}
	channel := EncodeDStarHeaderFEC(info)
	if len(channel) != DStarHeaderChannelBits {
		t.Fatalf("encoded length = %d, want %d", len(channel), DStarHeaderChannelBits)
	}
	for i, b := range channel {
		if b > 1 {
			t.Errorf("channel[%d] = %d, want 0 or 1", i, b)
		}
	}
	got, ok := DecodeDStarHeaderFEC(channel)
	if !ok {
		t.Fatal("DecodeDStarHeaderFEC returned ok=false on clean channel")
	}
	if !bytes.Equal(got, info) {
		t.Errorf("round-trip mismatch:\n got = % x\nwant = % x", got, info)
	}
}

// TestDStarHeaderFECCorrects1BitError verifies the convolutional
// Viterbi decoder corrects a single bit error in the channel
// stream — the practical headline for the FEC layer's value.
func TestDStarHeaderFECCorrects1BitError(t *testing.T) {
	info := []byte{
		0x00, 0x00, 0x00,
		'K', 'D', '0', 'A', 'A', 'A', ' ', 'B',
		'K', 'D', '0', 'A', 'A', 'A', ' ', 'G',
		'C', 'Q', 'C', 'Q', 'C', 'Q', ' ', ' ',
		'W', 'B', '7', 'X', 'Y', 'Z', ' ', ' ',
		'S', 'U', 'F', 'X',
		0xAB, 0xCD,
	}
	channel := EncodeDStarHeaderFEC(info)
	// Flip a single bit somewhere in the middle of the stream.
	channel[100] ^= 1
	got, ok := DecodeDStarHeaderFEC(channel)
	if !ok {
		t.Fatal("DecodeDStarHeaderFEC failed on 1-bit error (expected K=5 R=1/2 to absorb it)")
	}
	if !bytes.Equal(got, info) {
		t.Errorf("1-bit-error correction failed:\n got = % x\nwant = % x", got, info)
	}
}

// TestScrambleSelfInverse confirms the PN15 scrambler is its own
// inverse — encode and decode share the same implementation.
func TestScrambleSelfInverse(t *testing.T) {
	in := make([]byte, DStarHeaderChannelBits)
	for i := range in {
		in[i] = byte(i & 1)
	}
	orig := make([]byte, len(in))
	copy(orig, in)
	scrambleDStarHeader(in)
	if bytes.Equal(in, orig) {
		t.Error("scrambleDStarHeader was a no-op")
	}
	scrambleDStarHeader(in)
	if !bytes.Equal(in, orig) {
		t.Error("scrambleDStarHeader is not self-inverse")
	}
}

// TestDStarInterleaveRoundTrip confirms interleave + deinterleave is
// the identity, regardless of which bit positions actually carry
// data.
func TestDStarInterleaveRoundTrip(t *testing.T) {
	in := make([]byte, DStarHeaderChannelBits)
	for i := range in {
		in[i] = byte(i & 1)
	}
	out := deinterleaveDStarHeader(interleaveDStarHeader(in))
	if !bytes.Equal(out, in) {
		t.Error("interleave/deinterleave is not the identity")
	}
}
