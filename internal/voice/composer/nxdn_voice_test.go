package composer

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// TestNXDNVCHFrameRendersToPCM closes the loop at the frame-format
// boundary: a voice frame decoded from an on-air NXDN frame, packed
// MSB-first to 7 bytes exactly as runNXDNVoiceChain's sink does, must be
// accepted by the "ambe2-dmr" (3600x2450) vocoder the recorder maps
// "nxdn"/"nxdn48" to, and produce one 20 ms PCM frame.
func TestNXDNVCHFrameRendersToPCM(t *testing.T) {
	var payloads [4][]byte
	x := uint32(1)
	for v := range payloads {
		payloads[v] = make([]byte, 49)
		for i := range payloads[v] {
			x = 1664525*x + 1013904223
			payloads[v][i] = byte((x >> 24) & 1)
		}
	}
	frame, err := nxdn.EncodeAirVoiceFrame(payloads)
	if err != nil {
		t.Fatal(err)
	}
	post := append([]uint8(nil), frame[nxdn.AirFSWDibits:]...)
	nxdn.DescrambleAir(post)
	_, frames := nxdn.DecodeAirVoice(post)
	if len(frames) != 4 {
		t.Fatalf("decoded %d voice frames, want 4", len(frames))
	}

	packed := framing.PackBitsMSB(frames[0].Payload)
	if len(packed) != ambe2.FrameBytes {
		t.Fatalf("packed frame = %d bytes, want %d (vocoder FrameBytes)", len(packed), ambe2.FrameBytes)
	}
	pcm, err := ambe2.NewDMR().Decode(packed)
	if err != nil {
		t.Fatalf("ambe2-dmr Decode: %v", err)
	}
	if len(pcm) != mbe.SamplesPerFrame {
		t.Fatalf("PCM = %d samples, want %d (20 ms @ 8 kHz)", len(pcm), mbe.SamplesPerFrame)
	}
}

func TestNXDNSymbolRateFromProtocol(t *testing.T) {
	if got := nxdnSymbolRate("nxdn48"); got != 2400 {
		t.Errorf("nxdn48 → %v, want 2400", got)
	}
	if got := nxdnSymbolRate("nxdn"); got != 4800 {
		t.Errorf("nxdn → %v, want 4800", got)
	}
}
