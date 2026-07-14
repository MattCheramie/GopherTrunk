package gtbundle

import (
	"encoding/json"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// sigmfVersion is the SigMF spec version the emitter targets.
const sigmfVersion = "1.0.0"

// sigmfMeta is the minimal SigMF metadata document (a .sigmf-meta sidecar).
// GopherTrunk does not read SigMF natively, but siglab.Metadata is a near-subset
// of the SigMF global object, so emitting one makes a bundled capture/slice
// interoperable with SigMF-aware tooling (inspectrum, GNU Radio, sigmf-python)
// for the community-sharing goal. It is opt-in (WriterOptions.EmitSigMF).
type sigmfMeta struct {
	Global   sigmfGlobal    `json:"global"`
	Captures []sigmfCapture `json:"captures"`
	// Annotations is always present (empty) — the spec requires the key.
	Annotations []any `json:"annotations"`
}

type sigmfGlobal struct {
	Datatype    string  `json:"core:datatype"`
	SampleRate  float64 `json:"core:sample_rate,omitempty"`
	Version     string  `json:"core:version"`
	Recorder    string  `json:"core:recorder,omitempty"`
	Description string  `json:"core:description,omitempty"`
}

type sigmfCapture struct {
	SampleStart int64   `json:"core:sample_start"`
	Frequency   float64 `json:"core:frequency,omitempty"`
}

// sigmfDatatype maps a GopherTrunk sample-format name to the SigMF core:datatype
// token. Returns "" for a format with no clean SigMF equivalent (e.g. the
// 2-channel s16 baseband WAV, which carries its own header).
func sigmfDatatype(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "u8", "cu8":
		return "cu8"
	case "f32", "cf32", "cf32_le":
		return "cf32_le"
	default:
		return ""
	}
}

// MetadataToSigMF renders a siglab.Metadata as a SigMF .sigmf-meta document. It
// returns ok=false when the capture format has no SigMF datatype equivalent, so
// the caller can skip emitting a sidecar rather than write a malformed one.
func MetadataToSigMF(m *siglab.Metadata) ([]byte, bool, error) {
	if m == nil {
		return nil, false, nil
	}
	dt := sigmfDatatype(m.Format)
	if dt == "" {
		return nil, false, nil
	}
	doc := sigmfMeta{
		Global: sigmfGlobal{
			Datatype:    dt,
			SampleRate:  m.SampleRateHz,
			Version:     sigmfVersion,
			Recorder:    "gophertrunk",
			Description: m.Source,
		},
		Captures:    []sigmfCapture{{SampleStart: 0, Frequency: float64(m.CenterFreqHz)}},
		Annotations: []any{},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(raw, '\n'), true, nil
}
