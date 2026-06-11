package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const wbTestConfig = `# top comment
trunking:
  systems:
    # the regional DMR system
    - name: "Regional DMR"
      protocol: dmr
      control_channels:
        - 851037500   # primary CC
    - name: "Other P25"
      protocol: p25
      control_channels:
        - 770000000
`

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWriteDMRLearnedBandPlanPreservesComments(t *testing.T) {
	p := writeTemp(t, wbTestConfig)
	w, err := NewWriter(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteDMRLearnedBandPlan("Regional DMR", DMRLinearBandPlanConfig{
		BaseHz: 851_037_500, SpacingHz: 12_500, Offset: 1,
	}); err != nil {
		t.Fatalf("WriteDMRLearnedBandPlan: %v", err)
	}

	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{"# top comment", "# the regional DMR system", "# primary CC", "dmr_band_plan", "base_hz: 851037500", "spacing_hz: 12500", "offset: 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}

	// The plan must parse and resolve under the same shape the runtime uses.
	var cfg Config
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}
	var found *SystemConfig
	for i := range cfg.Trunking.Systems {
		if cfg.Trunking.Systems[i].Name == "Regional DMR" {
			found = &cfg.Trunking.Systems[i]
		}
	}
	if found == nil || found.DMRBandPlan == nil || found.DMRBandPlan.Linear == nil {
		t.Fatalf("learned plan not present after round-trip: %+v", found)
	}
	if found.DMRBandPlan.Linear.SpacingHz != 12_500 || found.DMRBandPlan.Linear.BaseHz != 851_037_500 {
		t.Errorf("linear = %+v", found.DMRBandPlan.Linear)
	}
}

func TestWriteDMRLearnedBandPlanRefusesExisting(t *testing.T) {
	body := wbTestConfig + "      dmr_band_plan:\n        linear:\n          base_hz: 1\n          spacing_hz: 12500\n          offset: 1\n"
	// Append the plan under the second system instead to keep indentation
	// valid; simpler: craft a config where the system already has a plan.
	body = `trunking:
  systems:
    - name: "Has Plan"
      protocol: dmr
      control_channels: [851037500]
      dmr_band_plan:
        linear:
          base_hz: 851037500
          spacing_hz: 12500
          offset: 1
`
	p := writeTemp(t, body)
	w, _ := NewWriter(p)
	err := w.WriteDMRLearnedBandPlan("Has Plan", DMRLinearBandPlanConfig{BaseHz: 1, SpacingHz: 12_500, Offset: 1})
	if err == nil || !strings.Contains(err.Error(), "already has") {
		t.Fatalf("expected refusal for existing plan, got %v", err)
	}
}

func TestWriteDMRLearnedBandPlanUnknownSystem(t *testing.T) {
	p := writeTemp(t, wbTestConfig)
	w, _ := NewWriter(p)
	if err := w.WriteDMRLearnedBandPlan("Nope", DMRLinearBandPlanConfig{BaseHz: 1, SpacingHz: 12_500, Offset: 1}); err == nil {
		t.Fatal("expected error for unknown system")
	}
}
