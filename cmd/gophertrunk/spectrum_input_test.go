package main

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeU8ToneCapture writes a u8 IQ capture like `gophertrunk capture -format
// u8` makes: a tone at toneHz plus noise across the whole byte range.
func writeU8ToneCapture(t *testing.T, path string, rate, toneHz float64, n int) {
	t.Helper()
	rng := rand.New(rand.NewSource(1256))
	b := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		ph := 2 * math.Pi * toneHz * float64(i) / rate
		re := 0.5*math.Cos(ph) + 0.2*rng.NormFloat64()
		im := 0.5*math.Sin(ph) + 0.2*rng.NormFloat64()
		b[2*i] = byte(math.Max(0, math.Min(255, math.Round(127.5+127.5*re))))
		b[2*i+1] = byte(math.Max(0, math.Min(255, math.Round(127.5+127.5*im))))
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSpectrumReadsCaptureSidecar: the #1256 reporter ran `spectrum -in
// mpg0.raw` on a u8 1.6 MS/s capture and got rate=2000000, rms=NaN and 153
// NaN "carriers", because spectrum ignored the .metadata.json beside it and
// read the bytes as f32. With the sidecar it must decode u8 at 1.6 MS/s,
// centred where the capture was, and find the one real carrier.
func TestSpectrumReadsCaptureSidecar(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "mpg0.raw")
	writeU8ToneCapture(t, in, 1_600_000, 200_000, 1<<18)
	side := `{"protocol":"","sample_rate_hz":1600000,"center_freq_hz":868950000,"format":"u8","expected":{}}`
	if err := os.WriteFile(filepath.Join(dir, "mpg0.metadata.json"), []byte(side), 0o644); err != nil {
		t.Fatal(err)
	}

	format, rate, freq := "f32", 2_000_000.0, uint64(0) // the flag defaults
	iq, notes, err := loadSpectrumInput(in, map[string]bool{}, &format, &rate, &freq)
	if err != nil {
		t.Fatal(err)
	}
	if format != "u8" || rate != 1_600_000 || freq != 868_950_000 {
		t.Fatalf("settings from sidecar: format=%s rate=%.0f freq=%d", format, rate, freq)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "mpg0.metadata.json") {
		t.Fatalf("notes = %q, want one naming the sidecar", notes)
	}
	r := buildSpectrumReport(iq, rate, uint32(freq), 4096, 10, 12500)
	if math.IsNaN(r.RMSPowerDBFS) || math.IsInf(r.RMSPowerDBFS, 0) {
		t.Fatalf("rms = %v", r.RMSPowerDBFS)
	}
	if len(r.Carriers) != 1 {
		t.Fatalf("carriers = %+v, want the one tone", r.Carriers)
	}
	if c := r.Carriers[0]; math.Abs(c.OffsetHz-200_000) > 1000 || math.Abs(float64(c.FreqHz)-869_150_000) > 1000 {
		t.Fatalf("carrier = %+v, want +200 kHz (869.15 MHz)", c)
	}

	// A flag the user gives still wins over the sidecar.
	format, rate, freq = "f32", 2_000_000.0, uint64(0)
	_, _, _ = loadSpectrumInput(in, map[string]bool{"sample-rate": true}, &format, &rate, &freq)
	if rate != 2_000_000 || format != "u8" {
		t.Fatalf("explicit -sample-rate overridden: rate=%.0f format=%s", rate, format)
	}
}

// TestSpectrumRefusesNonFiniteSamples: without a sidecar, a u8 capture read
// as the f32 default is full of NaN/Inf. That is a wrong -format, never real
// IQ, so it is refused with a pointer to -format u8 instead of a NaN report.
func TestSpectrumRefusesNonFiniteSamples(t *testing.T) {
	in := filepath.Join(t.TempDir(), "nosidecar.raw")
	writeU8ToneCapture(t, in, 1_600_000, 200_000, 1<<18)
	format, rate, freq := "f32", 2_000_000.0, uint64(0)
	_, _, err := loadSpectrumInput(in, map[string]bool{}, &format, &rate, &freq)
	if err == nil || !strings.Contains(err.Error(), "-format u8") {
		t.Fatalf("err = %v, want a wrong-format refusal", err)
	}
}
