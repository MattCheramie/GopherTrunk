package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestMarshalRoundTrip confirms a config built in-memory marshals to YAML
// and parses back to an equivalent, valid config — the core contract the
// web Config Builder's save/load cycle relies on.
func TestMarshalRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.Trunking.Systems = []SystemConfig{{
		Name:            "Metro-P25",
		Protocol:        "p25",
		ControlChannels: []uint32{851_037_500, 851_062_500},
	}}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back Config
	if err := unmarshalStrict(data, &back); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("round-tripped config invalid: %v", err)
	}
	if len(back.Trunking.Systems) != 1 || back.Trunking.Systems[0].Name != "Metro-P25" {
		t.Fatalf("system did not round-trip: %+v", back.Trunking.Systems)
	}
	if got := back.Trunking.Systems[0].ControlChannels; len(got) != 2 || got[0] != 851_037_500 {
		t.Fatalf("control channels did not round-trip: %v", got)
	}
}

// unmarshalStrict parses via the same path as Load (sans file IO).
func unmarshalStrict(data []byte, out *Config) error {
	tmp, err := os.CreateTemp(t_TempDir(), "cfg-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	loaded, err := Load(tmp.Name())
	if err != nil {
		return err
	}
	*out = loaded
	return nil
}

// t_TempDir returns the OS temp dir; a tiny indirection so the helper
// above stays test-package-local without a *testing.T in scope.
func t_TempDir() string { return os.TempDir() }

func TestWriteConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Default()

	// New file: overwrite flag is irrelevant; mtime returned.
	mtime, err := WriteConfigFile(path, cfg, 0, false)
	if err != nil {
		t.Fatalf("write new: %v", err)
	}
	if mtime == 0 {
		t.Fatalf("expected non-zero mtime")
	}

	// Existing file without overwrite → ErrFileExists.
	if _, err := WriteConfigFile(path, cfg, 0, false); !errors.Is(err, ErrFileExists) {
		t.Fatalf("expected ErrFileExists, got %v", err)
	}

	// Overwrite with a stale guard mtime → ErrModifiedExternally.
	if _, err := WriteConfigFile(path, cfg, mtime-1, true); !errors.Is(err, ErrModifiedExternally) {
		t.Fatalf("expected ErrModifiedExternally, got %v", err)
	}

	// Overwrite with the correct guard mtime → success.
	if _, err := WriteConfigFile(path, cfg, mtime, true); err != nil {
		t.Fatalf("overwrite with correct guard: %v", err)
	}

	// Invalid config is never written.
	bad := Default()
	bad.SDR.SampleRate = 100 // below the 225 kHz floor
	if _, err := WriteConfigFile(filepath.Join(dir, "bad.yaml"), bad, 0, true); err == nil {
		t.Fatalf("expected validation error for bad config")
	}
	if _, err := os.Stat(filepath.Join(dir, "bad.yaml")); !os.IsNotExist(err) {
		t.Fatalf("invalid config must not be written to disk")
	}
}

func TestValidateSection(t *testing.T) {
	cfg := Default()
	cfg.SDR.SampleRate = 100 // invalid

	if errs := cfg.ValidateSection("sdr"); len(errs) == 0 {
		t.Fatalf("expected sdr section error")
	}
	// A section with no bearing on the sample-rate error is clean.
	if errs := cfg.ValidateSection("web"); len(errs) != 0 {
		t.Fatalf("web section should be valid: %v", errs)
	}
	// Unknown section names are treated as valid (nil).
	if errs := cfg.ValidateSection("nonexistent"); len(errs) != 0 {
		t.Fatalf("unknown section should be nil: %v", errs)
	}
	// ValidateAll surfaces the sdr error.
	all := cfg.ValidateAll()
	if len(all) == 0 {
		t.Fatalf("expected ValidateAll to report the sdr error")
	}
}

// TestValidateAllAccumulates confirms ValidateAll reports every problem at
// once: one error per failing list item, across multiple sections.
func TestValidateAllAccumulates(t *testing.T) {
	cfg := Default()
	cfg.SDR.SampleRate = 100 // sdr error #1
	cfg.Trunking.Systems = []SystemConfig{
		{Name: "", Protocol: "p25"},     // trunking systems[0]: name required
		{Name: "ok", Protocol: "bogus"}, // trunking systems[1]: bad protocol
	}
	cfg.Scanner.Conventional = []ConvChannelConfig{
		{FrequencyHz: 0}, // scanner conventional[0]: freq required
		{FrequencyHz: 460_000_000, Mode: "weird"}, // scanner conventional[1]: bad mode
	}
	all := cfg.ValidateAll()
	if len(all) < 5 {
		t.Fatalf("expected ≥5 accumulated errors (1 sdr + 2 trunking + 2 scanner), got %d: %v", len(all), all)
	}
	// ValidateSection returns just that section's multiple errors.
	if got := cfg.ValidateSection("scanner"); len(got) != 2 {
		t.Fatalf("expected 2 scanner errors, got %d: %v", len(got), got)
	}
	// Validate() keeps its first-error contract (the sdr error, first section).
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate should still return the first error")
	}
}
