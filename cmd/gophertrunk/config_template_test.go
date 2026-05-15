package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestRenderConfigYAML_DefaultsLoadAndValidate is the keystone test for
// the wizard: a fresh wizardAnswers (the "operator hit Enter through
// every step" path) must produce a YAML that internal/config.Load
// accepts and Validate signs off on. If this passes, the wizard cannot
// emit something the daemon refuses to load.
func TestRenderConfigYAML_DefaultsLoadAndValidate(t *testing.T) {
	t.Parallel()
	a := defaultWizardAnswers()
	out, err := renderConfigYAML(a)
	if err != nil {
		t.Fatalf("renderConfigYAML: %v", err)
	}

	path := writeTempYAML(t, out)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load on wizard defaults: %v\n--- yaml ---\n%s", err, out)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config.Validate on wizard defaults: %v", err)
	}
}

// TestRenderConfigYAML_FullyPopulatedLoadAndValidate exercises the
// non-default paths through the template (custom SDR devices, a CORS
// allow-list, a token file, audio enabled, etc.). The same Load +
// Validate cycle must succeed.
func TestRenderConfigYAML_FullyPopulatedLoadAndValidate(t *testing.T) {
	t.Parallel()
	a := defaultWizardAnswers()
	a.HTTPAddr = "0.0.0.0:8080"
	a.GRPCAddr = ""
	a.AuthMode = "required"
	a.AuthTokenFile = "/etc/gophertrunk/api-token"
	a.CORSAllowedOrigins = []string{"null", "http://laptop.local:8000"}
	a.SDRDevices = []wizardSDR{
		{Serial: "00000001", Role: "control", PPM: 0, Gain: "auto", BiasTee: false},
		{Serial: "00000002", Role: "voice", PPM: 1, Gain: "496", BiasTee: true},
	}
	a.AudioEnabled = true
	a.AudioVolume = 0.5
	a.ManualTuneEnabled = true
	a.ScannerScanMode = "list"

	out, err := renderConfigYAML(a)
	if err != nil {
		t.Fatalf("renderConfigYAML: %v", err)
	}

	path := writeTempYAML(t, out)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load on populated answers: %v\n--- yaml ---\n%s", err, out)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config.Validate on populated answers: %v", err)
	}

	// Confirm the round-trip preserved a few representative values.
	if cfg.API.HTTPAddr != "0.0.0.0:8080" {
		t.Errorf("HTTPAddr = %q, want 0.0.0.0:8080", cfg.API.HTTPAddr)
	}
	if got := cfg.API.Auth.Mode; string(got) != "required" {
		t.Errorf("auth.mode = %q, want required", got)
	}
	if got := cfg.API.Auth.TokenFile; got != "/etc/gophertrunk/api-token" {
		t.Errorf("auth.token_file = %q", got)
	}
	if len(cfg.API.CORS.AllowedOrigins) != 2 {
		t.Errorf("cors.allowed_origins len = %d, want 2", len(cfg.API.CORS.AllowedOrigins))
	}
	if n := len(cfg.SDR.Devices); n != 2 {
		t.Fatalf("sdr.devices len = %d, want 2", n)
	}
	if cfg.SDR.Devices[1].BiasTee != true {
		t.Errorf("device[1].bias_tee = false, want true")
	}
	if !cfg.Audio.Enabled {
		t.Errorf("audio.enabled false, want true")
	}
	if cfg.Audio.Volume < 0.49 || cfg.Audio.Volume > 0.51 {
		t.Errorf("audio.volume = %v, want ~0.5", cfg.Audio.Volume)
	}
	if !cfg.Scanner.ManualTuneEnabled {
		t.Errorf("scanner.manual_tune_enabled false, want true")
	}
}

// TestRenderConfigYAML_QuotingHandlesSpecialChars confirms that fields
// containing quotes / backslashes / spaces don't break the YAML decode.
func TestRenderConfigYAML_QuotingHandlesSpecialChars(t *testing.T) {
	t.Parallel()
	a := defaultWizardAnswers()
	a.StoragePath = `C:\GopherTrunk\calls "primary".db`
	a.RecordingsDir = `/data/with spaces`
	a.AudioDevice = `hw:0,0`

	out, err := renderConfigYAML(a)
	if err != nil {
		t.Fatalf("renderConfigYAML: %v", err)
	}
	path := writeTempYAML(t, out)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load on special chars: %v\n--- yaml ---\n%s", err, out)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config.Validate on special chars: %v", err)
	}
	if cfg.Storage.Path != `C:\GopherTrunk\calls "primary".db` {
		t.Errorf("storage.path round-trip mismatch: %q", cfg.Storage.Path)
	}
	if cfg.Recordings.Dir != `/data/with spaces` {
		t.Errorf("recordings.dir round-trip mismatch: %q", cfg.Recordings.Dir)
	}
	if cfg.Audio.Device != "hw:0,0" {
		t.Errorf("audio.device round-trip mismatch: %q", cfg.Audio.Device)
	}
}

// TestRenderConfigYAML_SectionsInOrder is a guard against accidental
// template reflow. The downstream merge path in import_writer.go
// expects trunking: to appear after sdr: so PDF/CSV imports land in
// the right place when chained with the wizard.
func TestRenderConfigYAML_SectionsInOrder(t *testing.T) {
	t.Parallel()
	out, err := renderConfigYAML(defaultWizardAnswers())
	if err != nil {
		t.Fatalf("renderConfigYAML: %v", err)
	}
	s := string(out)
	want := []string{
		"\nlog:",
		"\napi:",
		"\nmetrics:",
		"\nstorage:",
		"\nrecordings:",
		"\nretention:",
		"\nsdr:",
		"\ntrunking:",
		"\nscanner:",
		"\naudio:",
		"\ntone_out:",
	}
	last := -1
	for _, w := range want {
		idx := strings.Index(s, w)
		if idx < 0 {
			t.Errorf("missing section %q in rendered YAML", strings.TrimSpace(w))
			continue
		}
		if idx <= last {
			t.Errorf("section %q appeared at %d but previous section ended at %d (out of order)", w, idx, last)
		}
		last = idx
	}
}

func writeTempYAML(t *testing.T, body []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}
