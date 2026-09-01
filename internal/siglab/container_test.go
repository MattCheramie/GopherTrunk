package siglab

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// testIQ builds a deterministic complex tone so the round-trips have real,
// non-degenerate content (a constant-modulus rotating phasor plus a slow
// amplitude ramp), exercising the full int16 range without clipping.
func testIQ(n int) []complex64 {
	iq := make([]complex64, n)
	for i := range iq {
		amp := 0.2 + 0.7*float64(i)/float64(n)
		ph := 2 * math.Pi * 0.031 * float64(i)
		iq[i] = complex(float32(amp*math.Cos(ph)), float32(amp*math.Sin(ph)))
	}
	return iq
}

func writeContainer(t *testing.T, path string, format SampleFormat, rate int, iq []complex64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	c, err := NewIQContainer(f, format, rate)
	if err != nil {
		t.Fatalf("NewIQContainer(%v): %v", format, err)
	}
	// Write in a couple of uneven chunks to exercise block boundaries.
	if err := c.Write(iq[:len(iq)/3]); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := c.Write(iq[len(iq)/3:]); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := c.Samples(); got != int64(len(iq)) {
		t.Errorf("Samples()=%d, want %d", got, len(iq))
	}
	if err := c.Finalize(); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// TestIQContainerWAVBodyMatchesCS16 pins the contract that a wav dump is exactly
// the cs16 body plus a 44-byte RIFF header — so a wav capture replays
// byte-for-byte identically to the cs16 equivalent.
func TestIQContainerWAVBodyMatchesCS16(t *testing.T) {
	iq := testIQ(5000)
	dir := t.TempDir()

	cs16Path := filepath.Join(dir, "c.cs16")
	wavPath := filepath.Join(dir, "c.wav")
	writeContainer(t, cs16Path, FormatS16, 48000, iq)
	writeContainer(t, wavPath, FormatWAV, 48000, iq)

	cs16, _ := os.ReadFile(cs16Path)
	wav, _ := os.ReadFile(wavPath)

	// cs16 body == EncodeCapture(cs16).
	if want := EncodeCapture(iq, FormatS16); !bytes.Equal(cs16, want) {
		t.Fatalf("cs16 body mismatch: got %d bytes, want %d", len(cs16), len(want))
	}
	// wav = 44-byte header + identical body.
	if len(wav) != iqWavHeaderSize+len(cs16) {
		t.Fatalf("wav length %d, want header(%d)+body(%d)", len(wav), iqWavHeaderSize, len(cs16))
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("wav missing RIFF/WAVE header")
	}
	if !bytes.Equal(wav[iqWavHeaderSize:], cs16) {
		t.Fatalf("wav body differs from cs16 body")
	}
}

// TestIQContainerFLACRoundTrips proves the FLAC container is lossless: decoding
// it back yields the exact same int16 samples as the cs16 body, it is a real
// FLAC file (fLaC signature), and it is smaller than cs16.
func TestIQContainerFLACRoundTrips(t *testing.T) {
	iq := testIQ(20000)
	dir := t.TempDir()

	cs16Path := filepath.Join(dir, "c.cs16")
	flacPath := filepath.Join(dir, "c.flac")
	writeContainer(t, cs16Path, FormatS16, 48000, iq)
	writeContainer(t, flacPath, FormatFLAC, 48000, iq)

	raw, _ := os.ReadFile(flacPath)
	if len(raw) < 4 || string(raw[0:4]) != "fLaC" {
		t.Fatalf("not a FLAC file (signature = %q)", raw[:min(4, len(raw))])
	}
	cs16, _ := os.ReadFile(cs16Path)
	if len(raw) >= len(cs16) {
		t.Errorf("flac (%d bytes) not smaller than cs16 (%d bytes)", len(raw), len(cs16))
	}

	// Decode the FLAC back through the read path and compare the sw16 body.
	f, err := os.Open(flacPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fr, rate, err := newFLACSW16Reader(f)
	if err != nil {
		t.Fatalf("newFLACSW16Reader: %v", err)
	}
	if rate != 48000 {
		t.Errorf("flac rate = %d, want 48000", rate)
	}
	decoded := new(bytes.Buffer)
	if _, err := decoded.ReadFrom(fr); err != nil {
		t.Fatalf("decode flac body: %v", err)
	}
	if !bytes.Equal(decoded.Bytes(), cs16) {
		t.Fatalf("flac round-trip body differs from cs16 (got %d, want %d bytes)", decoded.Len(), len(cs16))
	}
}

// TestIQContainerWAVReplayable confirms a wav dump decodes through the same
// engine input path as cs16 to the same complex samples.
func TestIQContainerWAVReplayable(t *testing.T) {
	iq := testIQ(3000)
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "c.wav")
	writeContainer(t, wavPath, FormatWAV, 96000, iq)

	f, err := os.Open(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg := Config{Format: FormatWAV}
	r, err := prepareWAVInput(f, &cfg)
	if err != nil {
		t.Fatalf("prepareWAVInput: %v", err)
	}
	if cfg.SampleRateHz != 96000 {
		t.Errorf("rate from wav header = %v, want 96000", cfg.SampleRateHz)
	}
	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if want := EncodeCapture(iq, FormatS16); !bytes.Equal(body.Bytes(), want) {
		t.Fatalf("wav-stripped body != cs16 body")
	}
}
