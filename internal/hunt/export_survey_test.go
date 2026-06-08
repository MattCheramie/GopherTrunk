package hunt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func sampleSurvey() *SignalSurvey {
	return &SignalSurvey{
		Signals: []DetectedSignal{
			{FreqHz: 851_012_500, SNRDb: 22.5, OccupiedBwHz: 12_500, Class: survey.ClassTrunkControl,
				Confidence: 0.9, Trunking: &TrunkingRef{Protocol: "p25", Locked: true, ControlHz: 851_012_500}},
			{FreqHz: 154_025_000, SNRDb: 14.0, OccupiedBwHz: 9_000, Class: survey.ClassNBFM,
				Confidence: 0.6, Analog: &survey.AnalogReport{Active: true, CTCSSHz: 100.0}},
			{FreqHz: 152_840_000, SNRDb: 18.0, OccupiedBwHz: 12_000, Class: survey.ClassPaging,
				Pages: []survey.PageRef{{Protocol: "flex", Capcode: 1234567, Text: "TEST"}}},
		},
	}
}

func TestWriteSurveyJSONRoundTrips(t *testing.T) {
	sv := sampleSurvey()
	var buf bytes.Buffer
	if err := WriteSurvey(&buf, sv, SurveyJSON); err != nil {
		t.Fatalf("WriteSurvey json: %v", err)
	}
	var got SignalSurvey
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Signals) != len(sv.Signals) {
		t.Fatalf("round-trip signals = %d, want %d", len(got.Signals), len(sv.Signals))
	}
	if got.Signals[0].Class != survey.ClassTrunkControl || got.Signals[0].Trunking == nil || !got.Signals[0].Trunking.Locked {
		t.Errorf("trunking signal did not round-trip: %+v", got.Signals[0])
	}
	if got.Signals[2].Pages == nil || got.Signals[2].Pages[0].Protocol != "flex" {
		t.Errorf("paging signal did not round-trip: %+v", got.Signals[2])
	}
}

func TestWriteSurveyCSVHasRowPerSignal(t *testing.T) {
	sv := sampleSurvey()
	var buf bytes.Buffer
	if err := WriteSurvey(&buf, sv, SurveyCSV); err != nil {
		t.Fatalf("WriteSurvey csv: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// 1 header + one row per signal.
	if len(lines) != 1+len(sv.Signals) {
		t.Fatalf("csv lines = %d, want %d\n%s", len(lines), 1+len(sv.Signals), buf.String())
	}
	if !strings.HasPrefix(lines[0], "freq_mhz,class,") {
		t.Errorf("unexpected header: %q", lines[0])
	}
	if !strings.Contains(buf.String(), "p25 locked") {
		t.Errorf("trunking decode summary missing:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "ctcss=100.0") {
		t.Errorf("analog decode summary missing:\n%s", buf.String())
	}
}

func TestParseSurveyFormat(t *testing.T) {
	for _, s := range []string{"", "json", "survey-json"} {
		if f, err := ParseSurveyFormat(s); err != nil || f != SurveyJSON {
			t.Errorf("ParseSurveyFormat(%q) = %v, %v", s, f, err)
		}
	}
	if f, err := ParseSurveyFormat("csv"); err != nil || f != SurveyCSV {
		t.Errorf("ParseSurveyFormat(csv) = %v, %v", f, err)
	}
	if _, err := ParseSurveyFormat("bogus"); err == nil {
		t.Error("expected error for bogus survey format")
	}
}
