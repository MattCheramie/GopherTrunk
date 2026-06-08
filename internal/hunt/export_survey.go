package hunt

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SurveyFormat selects an export encoding for a SignalSurvey inventory. It is
// separate from Format (which encodes a DiscoveredSystem) because the survey is
// a distinct artifact — the full classified-carrier list, not just the trunked
// map.
type SurveyFormat int

const (
	// SurveyJSON marshals the whole SignalSurvey (signals + embedded system).
	SurveyJSON SurveyFormat = iota
	// SurveyCSV writes one row per detected signal, for spreadsheets/diffs.
	SurveyCSV
)

func (f SurveyFormat) String() string {
	if f == SurveyCSV {
		return "survey-csv"
	}
	return "survey-json"
}

// FileExtension is the conventional extension for a survey export.
func (f SurveyFormat) FileExtension() string {
	if f == SurveyCSV {
		return "csv"
	}
	return "json"
}

// ParseSurveyFormat maps a CLI/REST value to a SurveyFormat.
func ParseSurveyFormat(s string) (SurveyFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "json", "survey-json", "survey":
		return SurveyJSON, nil
	case "csv", "survey-csv":
		return SurveyCSV, nil
	default:
		return SurveyJSON, fmt.Errorf("hunt: unknown survey format %q (want json|csv)", s)
	}
}

// WriteSurvey serializes sv to w in the requested format.
func WriteSurvey(w io.Writer, sv *SignalSurvey, f SurveyFormat) error {
	if sv == nil {
		return fmt.Errorf("hunt: no survey to export")
	}
	switch f {
	case SurveyCSV:
		return writeSurveyCSV(w, sv)
	default:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(sv)
	}
}

// writeSurveyCSV emits one row per detected signal. The decode column carries a
// compact per-class summary (trunking protocol, page count, or analog activity)
// so the CSV is useful without the nested JSON.
func writeSurveyCSV(w io.Writer, sv *SignalSurvey) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"freq_mhz", "class", "confidence", "snr_db", "occupied_bw_khz", "baud_hz", "decode",
	}); err != nil {
		return err
	}
	for _, s := range sv.Signals {
		row := []string{
			strconv.FormatFloat(float64(s.FreqHz)/1e6, 'f', 4, 64),
			string(s.Class),
			strconv.FormatFloat(s.Confidence, 'f', 2, 64),
			strconv.FormatFloat(float64(s.SNRDb), 'f', 1, 32),
			strconv.FormatFloat(float64(s.OccupiedBwHz)/1e3, 'f', 1, 64),
			strconv.FormatFloat(s.BaudHz, 'f', 0, 64),
			surveyDecodeSummary(s),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// surveyDecodeSummary renders the per-class decode result as one CSV cell.
func surveyDecodeSummary(s DetectedSignal) string {
	switch {
	case s.Trunking != nil:
		if s.Trunking.Locked {
			return s.Trunking.Protocol + " locked"
		}
		return s.Trunking.Protocol
	case len(s.Pages) > 0:
		return fmt.Sprintf("%s %d page(s)", s.Pages[0].Protocol, len(s.Pages))
	case s.Analog != nil && s.Analog.Active:
		out := "active"
		if s.Analog.CTCSSHz > 0 {
			out += fmt.Sprintf(" ctcss=%.1f", s.Analog.CTCSSHz)
		}
		if s.Analog.DCSCode != "" {
			out += " dcs=" + s.Analog.DCSCode
		}
		return out
	case s.Error != "":
		return "error: " + s.Error
	default:
		return ""
	}
}
