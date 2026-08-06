package nxdn

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildVCHInfoDibits encodes four 49-bit payloads into the 144-dibit
// (288-bit) Information field a VCH traffic frame carries.
func buildVCHInfoDibits(t *testing.T, payloads [4][]byte) []uint8 {
	t.Helper()
	var allBits []byte
	for i := 0; i < VCHVoiceFramesPerFrame; i++ {
		frame, err := EncodeVCHFrame(payloads[i])
		if err != nil {
			t.Fatalf("EncodeVCHFrame[%d]: %v", i, err)
		}
		allBits = append(allBits, frame...)
	}
	if len(allBits) != VCHVoiceFramesPerFrame*nxdnAMBEOnAirBits {
		t.Fatalf("info bits = %d, want %d", len(allBits), VCHVoiceFramesPerFrame*nxdnAMBEOnAirBits)
	}
	dibits := framing.BitsToDibits(allBits)
	if len(dibits) != InfoFieldDibits {
		t.Fatalf("info dibits = %d, want %d", len(dibits), InfoFieldDibits)
	}
	return dibits
}

func TestExtractVCHFramesRoundTrip(t *testing.T) {
	payloads := [4][]byte{mkInfo(1), mkInfo(2), mkInfo(3), mkInfo(4)}
	info := buildVCHInfoDibits(t, payloads)

	frames := ExtractVCHFrames(info)
	if len(frames) != VCHVoiceFramesPerFrame {
		t.Fatalf("got %d frames, want %d", len(frames), VCHVoiceFramesPerFrame)
	}
	for i, f := range frames {
		if len(f.Payload) != nxdnAMBEInfoBits {
			t.Fatalf("frame %d payload = %d bits, want %d", i, len(f.Payload), nxdnAMBEInfoBits)
		}
		for j := range payloads[i] {
			if f.Payload[j] != payloads[i][j] {
				t.Fatalf("frame %d bit %d: got %d want %d", i, j, f.Payload[j], payloads[i][j])
			}
		}
	}
}

func TestExtractVCHFramesRejectsWrongLength(t *testing.T) {
	if got := ExtractVCHFrames(make([]uint8, InfoFieldDibits-1)); got != nil {
		t.Errorf("wrong-length input should yield nil, got %d frames", len(got))
	}
}
