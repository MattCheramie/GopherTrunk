package siglab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"gopkg.in/yaml.v3"
)

// Metadata is the sidecar that accompanies a capture (real-air or
// synthesized) and describes how to decode it plus what a correct decode
// must produce. It generalizes the ad-hoc per-test loaders and the
// acceptance-criteria documented in samples/README.md into one schema the
// `test` harness consumes. It loads from JSON or YAML.
type Metadata struct {
	// Protocol is the CLI/config protocol name (p25, p25-phase2, dmr,
	// dmr-tier2, nxdn, dpmr, edacs, motorola, ltr, mpt1327, tetra, ysf,
	// dstar; the replay aliases p25p1/dmr-tier3 are also accepted).
	Protocol string `json:"protocol" yaml:"protocol"`
	// Source / ToolCrossCheck are free-text provenance (where the capture
	// came from, which reference receiver it was validated against).
	Source         string `json:"source,omitempty" yaml:"source,omitempty"`
	ToolCrossCheck string `json:"tool_cross_check,omitempty" yaml:"tool_cross_check,omitempty"`
	// SampleRateHz is required — a raw cfile/u8 carries no rate of its own.
	SampleRateHz float64 `json:"sample_rate_hz" yaml:"sample_rate_hz"`
	// CenterFreqHz is informational (the capture's nominal centre).
	CenterFreqHz uint32 `json:"center_freq_hz,omitempty" yaml:"center_freq_hz,omitempty"`
	// Format is the on-disk encoding (u8|f32); empty defaults to u8.
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
	// TuneHz / AutoTune / Conjugate / IQCorrect mirror the engine knobs for
	// captures that need them.
	TuneHz    float64 `json:"tune_hz,omitempty" yaml:"tune_hz,omitempty"`
	AutoTune  bool    `json:"auto_tune,omitempty" yaml:"auto_tune,omitempty"`
	Conjugate bool    `json:"conjugate,omitempty" yaml:"conjugate,omitempty"`
	IQCorrect bool    `json:"iq_correct,omitempty" yaml:"iq_correct,omitempty"`
	// System carries per-protocol decoder knobs by their YAML config-key
	// names (e.g. "tetra_colour_code": "1", "p25_phase1_demod_mode":
	// "cqpsk"), applied onto the trunking.System the engine drives.
	System map[string]string `json:"system,omitempty" yaml:"system,omitempty"`
	// Expected is the acceptance contract graded against the decode.
	Expected Acceptance `json:"expected" yaml:"expected"`
}

// LoadMetadata reads a Metadata document from path (JSON or YAML, chosen by
// extension; .json → JSON, everything else → YAML).
func LoadMetadata(path string) (*Metadata, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Metadata
	if strings.EqualFold(filepath.Ext(path), ".json") {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	} else {
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if m.Protocol == "" {
		return nil, fmt.Errorf("%s: metadata missing required \"protocol\"", path)
	}
	if m.SampleRateHz <= 0 {
		return nil, fmt.Errorf("%s: metadata missing required \"sample_rate_hz\"", path)
	}
	return &m, nil
}

// DiscoverMetadata finds the sidecar for a capture by stem: it tries
// <stem>.metadata.json, <stem>.metadata.yaml, then <capture>.json/.yaml.
// Returns "" when none exists.
func DiscoverMetadata(capturePath string) string {
	stem := strings.TrimSuffix(capturePath, filepath.Ext(capturePath))
	for _, cand := range []string{
		stem + ".metadata.json",
		stem + ".metadata.yaml",
		stem + ".metadata.yml",
		capturePath + ".json",
		capturePath + ".yaml",
	} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return ""
}

// Config builds an engine Config from the metadata for the given capture.
// collectIQDiag is threaded through because the harness wants the analyzer's
// decode-error rate available to the verdict.
func (m *Metadata) Config(collectIQDiag bool) (Config, error) {
	proto, err := ParseProtocolCLI(m.Protocol)
	if err != nil {
		return Config{}, err
	}
	format, err := ParseSampleFormat(m.Format)
	if err != nil {
		return Config{}, err
	}
	sys := trunking.System{Name: "siglab", Protocol: proto}
	applySystemKnobs(&sys, m.System)
	acc := m.Expected
	return Config{
		Protocol:      proto,
		System:        sys,
		SystemName:    "siglab",
		FrequencyHz:   m.CenterFreqHz,
		SampleRateHz:  m.SampleRateHz,
		Format:        format,
		TuneHz:        m.TuneHz,
		AutoTune:      m.AutoTune,
		Conjugate:     m.Conjugate,
		IQCorrect:     m.IQCorrect,
		CollectIQDiag: collectIQDiag,
		Acceptance:    &acc,
	}, nil
}

// applySystemKnobs maps the metadata's config-key/value pairs onto the
// trunking.System fields the pipeline factories read. Unknown keys are
// ignored (forward-compatible). Keys match the YAML config names operators
// already know from config.example.yaml.
func applySystemKnobs(sys *trunking.System, knobs map[string]string) {
	for k, v := range knobs {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "p25_phase1_demod_mode":
			sys.P25Phase1DemodMode = v
		case "p25_phase1_soft_decision":
			sys.P25Phase1SoftDecision = v
		case "p25_phase2_trellis_mode":
			sys.P25Phase2TrellisMode = v
		case "p25_phase2_rs_mode":
			sys.P25Phase2RSMode = v
		case "p25_phase2_interleave_mode":
			sys.P25Phase2InterleaveMode = v
		case "p25_phase2_scrambler_mode":
			sys.P25Phase2ScramblerMode = v
		case "p25_phase2_soft_decision":
			sys.P25Phase2SoftDecision = v
		case "p25_phase2_equalizer":
			sys.P25Phase2Equalizer = v
		case "p25_phase2_dc_block":
			sys.P25Phase2DCBlock = v
		case "p25_phase2_clock_mode":
			sys.P25Phase2ClockMode = v
		case "wacn":
			sys.WACN = uint32(parseUintKnob(v, 32))
		case "system_id", "sysid":
			sys.SystemID = uint16(parseUintKnob(v, 16))
		case "site":
			sys.Site = uint8(parseUintKnob(v, 8))
		case "tetra_colour_code", "tetra_color_code":
			sys.TETRAColourCode = uint32(parseUintKnob(v, 32))
		case "tetra_channel":
			sys.TETRAChannel = v
		case "tetra_channel_coding":
			sys.TETRAChannelCoding = v
		case "tetra_clock_mode":
			sys.TETRAClockMode = v
		case "nxdn_viterbi_mode":
			sys.NXDNViterbiMode = v
		case "nxdn_soft_decision":
			sys.NXDNSoftDecision = v
		case "nxdn_deviation_hz":
			sys.NXDNDeviationHz, _ = strconv.ParseFloat(v, 64)
		case "edacs_bch_mode":
			sys.EDACSBCHMode = v
		case "motorola_bch_mode":
			sys.MotorolaBCHMode = v
		case "mpt1327_bch_mode":
			sys.MPT1327BCHMode = v
		case "mpt1327_cwsc_tolerance":
			sys.MPT1327CWSCTolerance = v
		case "ltr_fcs_mode":
			sys.LTRFCSMode = v
		case "ltr_manchester_mode":
			sys.LTRManchesterMode = v
		case "dstar_fec_mode":
			sys.DStarFECMode = v
		}
	}
}

func parseUintKnob(s string, bits int) uint64 {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		u, _ := strconv.ParseUint(s[2:], 16, bits)
		return u
	}
	u, _ := strconv.ParseUint(s, 10, bits)
	return u
}
