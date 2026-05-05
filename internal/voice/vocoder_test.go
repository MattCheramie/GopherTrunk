package voice

import "testing"

func TestNullVocoderProducesSilence(t *testing.T) {
	v := NewNullVocoder(11)
	if v.FrameSize() != 11 {
		t.Errorf("FrameSize = %d, want 11", v.FrameSize())
	}
	out, err := v.Decode(make([]byte, 11))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != 160 { // 20 ms at 8 kHz
		t.Errorf("len(out) = %d, want 160", len(out))
	}
	for i, s := range out {
		if s != 0 {
			t.Fatalf("non-zero sample at %d: %d", i, s)
		}
	}
}

func TestRegistryNamesAndNew(t *testing.T) {
	r := NewRegistry()
	r.Register("a", func() (Vocoder, error) { return NewNullVocoder(7), nil })
	r.Register("b", func() (Vocoder, error) { return NewNullVocoder(9), nil })
	names := r.Names()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("Names = %v, want [a b]", names)
	}
	v, err := r.New("a")
	if err != nil {
		t.Fatal(err)
	}
	if v.FrameSize() != 7 {
		t.Errorf("FrameSize = %d, want 7", v.FrameSize())
	}
	if _, err := r.New("does-not-exist"); err == nil {
		t.Error("expected error for unknown name")
	}
}

func TestDefaultRegistryHasNull(t *testing.T) {
	v, err := DefaultRegistry.New("null")
	if err != nil {
		t.Fatalf("DefaultRegistry.New(null): %v", err)
	}
	if v.Name() != "null" {
		t.Errorf("Name = %q, want null", v.Name())
	}
}
