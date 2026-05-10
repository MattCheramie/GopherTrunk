package imbe

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

func TestDecoderRegistered(t *testing.T) {
	v, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatalf("DefaultRegistry.New(%q): %v", VocoderName, err)
	}
	defer v.Close()
	if v.Name() != VocoderName {
		t.Errorf("Name() = %q, want %q", v.Name(), VocoderName)
	}
}

func TestDecoderName(t *testing.T) {
	d := New()
	if d.Name() != "imbe-go" {
		t.Errorf("Name() = %q, want %q", d.Name(), "imbe-go")
	}
}

func TestDecoderFrameSize(t *testing.T) {
	d := New()
	if d.FrameSize() != 11 {
		t.Errorf("FrameSize() = %d, want 11", d.FrameSize())
	}
}

func TestDecodeReturnsSilenceForValidFrame(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes) // contents irrelevant — stub returns silence
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), SamplesPerFrame)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (stub silence)", i, s)
		}
	}
}

func TestDecodeRejectsShortFrame(t *testing.T) {
	d := New()
	if _, err := d.Decode(make([]byte, FrameBytes-1)); err == nil {
		t.Error("expected error for short frame")
	}
	if _, err := d.Decode(make([]byte, FrameBytes+1)); err == nil {
		t.Error("expected error for long frame")
	}
}

func TestResetAndCloseAreSafe(t *testing.T) {
	d := New()
	d.Reset()
	if err := d.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	// Re-using after Close should still work — pure-Go decoder holds
	// no resources, and the Vocoder contract doesn't forbid it for
	// stateless implementations.
	if _, err := d.Decode(make([]byte, FrameBytes)); err != nil {
		t.Errorf("Decode after Close: %v", err)
	}
}

func TestRegistryReturnsFreshInstancePerCall(t *testing.T) {
	a, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := voice.DefaultRegistry.New(VocoderName)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if a == b {
		t.Error("expected each Registry.New call to return a fresh instance")
	}
}

func TestNameAppearsInRegistryListing(t *testing.T) {
	names := voice.DefaultRegistry.Names()
	for _, n := range names {
		if n == VocoderName {
			return
		}
	}
	t.Errorf("registry names %v missing %q", names, VocoderName)
}
