package acelp

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestVocoderRegistered checks the tetra-acelp vocoder is in the default
// registry and builds through it.
func TestVocoderRegistered(t *testing.T) {
	v, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatalf("registry.New(%q): %v", VocoderName, err)
	}
	if v.Name() != VocoderName {
		t.Errorf("Name() = %q, want %q", v.Name(), VocoderName)
	}
	if v.FrameSize() != speechFrameBytes {
		t.Errorf("FrameSize() = %d, want %d", v.FrameSize(), speechFrameBytes)
	}
	if err := v.Close(); err != nil {
		t.Errorf("Close(): %v", err)
	}
}

// TestVocoderDecode checks a full packed frame decodes to 240 samples and a
// short/nil frame is handled as an erased frame (still 240 samples).
func TestVocoderDecode(t *testing.T) {
	v := NewVocoder()
	frame := make([]byte, speechFrameBytes)
	for i := range frame {
		frame[i] = byte(i * 17)
	}
	pcm, err := v.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(pcm) != frameLen {
		t.Fatalf("Decode returned %d samples, want %d", len(pcm), frameLen)
	}

	// erased frame (too short) still yields a concealed frame
	pcm, err = v.Decode(nil)
	if err != nil || len(pcm) != frameLen {
		t.Fatalf("erased decode: %d samples, err %v", len(pcm), err)
	}
}

// TestVocoderResetDeterministic checks Reset restores the decoder so the same
// frame sequence reproduces identical output.
func TestVocoderResetDeterministic(t *testing.T) {
	v := NewVocoder()
	frame := make([]byte, speechFrameBytes)
	for i := range frame {
		frame[i] = byte(i*29 + 3)
	}
	a, _ := v.Decode(frame)
	b, _ := v.Decode(frame)

	v.Reset()
	a2, _ := v.Decode(frame)
	b2, _ := v.Decode(frame)

	for i := range a {
		if a[i] != a2[i] || b[i] != b2[i] {
			t.Fatalf("Reset did not restore decoder state at sample %d", i)
		}
	}
}
