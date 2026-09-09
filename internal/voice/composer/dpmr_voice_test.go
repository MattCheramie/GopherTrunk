package composer

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// TestDPMRTCHFrameRendersToPCM closes the dPMR voice loop end to end at
// the frame-format boundary: a TCH voice frame's 49-bit payload, packed
// MSB-first to 7 bytes exactly as runDPMRVoiceChain's sink does, must
// be accepted by the "ambe2-dmr" (3600x2450) vocoder the recorder maps
// "dpmr" to, and produce one 20 ms PCM frame. (The upstream carve/FEC/
// codebook remain unverified on air — mirrors the NXDN test.)
func TestDPMRTCHFrameRendersToPCM(t *testing.T) {
	// A synthetic 49-bit payload round-tripped through the AMBE FEC, so
	// it is a well-formed TCH frame (mirrors what ExtractTCHFrames yields).
	info := make([]byte, 49)
	x := uint32(1)
	for i := range info {
		x = 1664525*x + 1013904223
		info[i] = byte((x >> 24) & 1)
	}
	onair, err := dpmr.EncodeTCHFrame(info)
	if err != nil {
		t.Fatalf("EncodeTCHFrame: %v", err)
	}
	payload, _, err := dpmr.DecodeTCHFrame(onair)
	if err != nil {
		t.Fatalf("DecodeTCHFrame: %v", err)
	}

	packed := framing.PackBitsMSB(payload)
	if len(packed) != ambe2.FrameBytes {
		t.Fatalf("packed frame = %d bytes, want %d (vocoder FrameBytes)", len(packed), ambe2.FrameBytes)
	}

	dec := ambe2.NewDMR() // the "ambe2-dmr" 3600x2450 vocoder dpmr maps to
	pcm, err := dec.Decode(packed)
	if err != nil {
		t.Fatalf("ambe2-dmr Decode: %v", err)
	}
	if len(pcm) != mbe.SamplesPerFrame {
		t.Fatalf("PCM = %d samples, want %d (20 ms @ 8 kHz)", len(pcm), mbe.SamplesPerFrame)
	}
}
