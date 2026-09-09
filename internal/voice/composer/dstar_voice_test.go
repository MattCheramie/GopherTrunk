package composer

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// TestDStarDVFrameRendersToPCM closes the D-STAR voice loop end to end
// at the frame-format boundary: a DV frame's 49-bit payload, packed
// MSB-first to 7 bytes exactly as runDStarVoiceChain's sink does, must
// be accepted by the base "ambe2" (AMBE 3600x2400) vocoder the recorder
// maps "dstar" to, and produce one 20 ms PCM frame. (The upstream
// carve/FEC remain unverified on air — mirrors the NXDN test.)
func TestDStarDVFrameRendersToPCM(t *testing.T) {
	// A synthetic 49-bit payload round-tripped through the AMBE FEC, so
	// it is a well-formed DV voice frame (mirrors what VoiceChannel yields).
	info := make([]byte, 49)
	x := uint32(1)
	for i := range info {
		x = 1664525*x + 1013904223
		info[i] = byte((x >> 24) & 1)
	}
	onair, err := dstar.EncodeDVVoiceBits(info)
	if err != nil {
		t.Fatalf("EncodeDVVoiceBits: %v", err)
	}
	payload, _, err := dstar.DecodeDVVoiceBits(onair)
	if err != nil {
		t.Fatalf("DecodeDVVoiceBits: %v", err)
	}

	packed := framing.PackBitsMSB(payload)
	if len(packed) != ambe2.FrameBytes {
		t.Fatalf("packed frame = %d bytes, want %d (vocoder FrameBytes)", len(packed), ambe2.FrameBytes)
	}

	dec := ambe2.New() // the base "ambe2" 3600x2400 vocoder dstar maps to
	pcm, err := dec.Decode(packed)
	if err != nil {
		t.Fatalf("ambe2 Decode: %v", err)
	}
	if len(pcm) != mbe.SamplesPerFrame {
		t.Fatalf("PCM = %d samples, want %d (20 ms @ 8 kHz)", len(pcm), mbe.SamplesPerFrame)
	}
}
