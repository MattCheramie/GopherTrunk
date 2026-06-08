package config

import (
	"strings"
	"testing"
)

func TestMarshalMergePreservesComments(t *testing.T) {
	original := []byte(`# GopherTrunk config
log:
  level: info # current level
  format: text
sdr:
  sample_rate: 2400000 # 2.4 MS/s
trunking:
  systems:
    - name: Old
      protocol: p25
`)
	cfg := Default()
	cfg.Log.Level = "debug"
	cfg.SDR.SampleRate = 1_024_000
	cfg.Trunking.Systems = []SystemConfig{{Name: "New", Protocol: "p25"}}

	out, err := MarshalMerge(original, cfg)
	if err != nil {
		t.Fatalf("MarshalMerge: %v", err)
	}
	s := string(out)

	// Top-of-file and trailing comments survive.
	if !strings.Contains(s, "# GopherTrunk config") {
		t.Errorf("lost header comment:\n%s", s)
	}
	if !strings.Contains(s, "# current level") {
		t.Errorf("lost line comment on log.level:\n%s", s)
	}
	if !strings.Contains(s, "# 2.4 MS/s") {
		t.Errorf("lost line comment on sdr.sample_rate:\n%s", s)
	}
	// Edits are applied.
	if !strings.Contains(s, "level: debug") {
		t.Errorf("log.level edit not applied:\n%s", s)
	}
	if !strings.Contains(s, "sample_rate: 1024000") {
		t.Errorf("sdr.sample_rate edit not applied:\n%s", s)
	}
	if !strings.Contains(s, "name: New") || strings.Contains(s, "name: Old") {
		t.Errorf("system replacement not applied:\n%s", s)
	}

	// The merged output must still parse + validate.
	var back Config
	if err := Unmarshal(out, &back); err != nil {
		t.Fatalf("merged output unparseable: %v", err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("merged output invalid: %v", err)
	}
	if back.Log.Level != "debug" || back.SDR.SampleRate != 1_024_000 {
		t.Fatalf("round-trip mismatch: %+v", back.Log)
	}
}

func TestMarshalMergeEmptyOriginalFallsBack(t *testing.T) {
	out, err := MarshalMerge(nil, Default())
	if err != nil {
		t.Fatal(err)
	}
	var back Config
	if err := Unmarshal(out, &back); err != nil {
		t.Fatalf("fallback output unparseable: %v", err)
	}
}
