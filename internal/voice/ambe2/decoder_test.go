package ambe2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
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
	if d.Name() != "ambe2" {
		t.Errorf("Name() = %q, want %q", d.Name(), "ambe2")
	}
}

func TestDecoderFrameSize(t *testing.T) {
	d := New()
	if d.FrameSize() != FrameBytes {
		t.Errorf("FrameSize() = %d, want %d", d.FrameSize(), FrameBytes)
	}
}

// TestDecodeReturnsSilenceSkeleton: the skeleton decoder emits
// silence per frame until PR-D plugs in parameter unpacking and
// PR-E wires synthesis. Output length must be exactly
// mbe.SamplesPerFrame (160) so the recorder + call pipeline can
// wire to this decoder today and start receiving real audio for
// free as the later pieces land.
func TestDecodeReturnsSilenceSkeleton(t *testing.T) {
	d := New()
	frame := make([]byte, FrameBytes)
	out, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != mbe.SamplesPerFrame {
		t.Errorf("len(out) = %d, want %d", len(out), mbe.SamplesPerFrame)
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (skeleton emits silence)", i, s)
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
	// Re-using after Close should still work — pure-Go decoder
	// holds no resources, and the Vocoder contract doesn't forbid
	// it for stateless implementations.
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

// TestNewWithConfigUsesCustomAGC pins that operator-supplied AGC
// tuning reaches the underlying *mbe.AGC instance. Mirrors the
// imbe-side test so AMBE+2 grows the same customization surface
// as it gains real synthesis.
func TestNewWithConfigUsesCustomAGC(t *testing.T) {
	custom := mbe.AGCConfig{TargetPeak: 16000}
	d := NewWithConfig(0, custom)
	def := mbe.DefaultAGCConfig()
	got := d.agc.Config()
	if got.TargetPeak != 16000 {
		t.Errorf("TargetPeak = %v, want 16000", got.TargetPeak)
	}
	if got.Attack != def.Attack {
		t.Errorf("Attack = %v, want default %v", got.Attack, def.Attack)
	}
}
