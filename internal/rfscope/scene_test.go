package rfscope

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func TestSceneSanitizeClampsNonFinite(t *testing.T) {
	t.Parallel()

	s := &Scene{
		NoiseFloorDbFS: float32(math.Inf(-1)),
		WindowSec:      math.NaN(),
		Bursts: []Burst{{
			ID:               1,
			ChannelPowerDBFS: math.Inf(1),
			SpectralFlatness: math.NaN(),
			PeakDbFS:         float32(math.NaN()),
			Features:         survey.ClassFeatures{SNRDb: math.Inf(1), BaudHz: math.NaN()},
		}},
		Channels: []Channel{{
			FreqHz:           450_000_000,
			DutyCycle:        math.NaN(),
			PeriodSec:        math.Inf(1),
			MedianFlatness:   math.NaN(),
			Timeline:         []Bin{{TSec: math.NaN(), PowerDbFS: float32(math.Inf(-1))}},
			BurstLenHist:     Hist{Edges: []float64{math.NaN(), 1.5}},
			InterArrivalHist: Hist{Edges: []float64{math.Inf(1)}},
		}},
		Emitters: []Emitter{{
			ID:              2,
			TotalAirtimeSec: math.NaN(),
			Fingerprint:     Fingerprint{SpectralFlatness: math.Inf(1), MedianBurstLenSec: math.NaN()},
		}},
		Conversations: []Conversation{{ID: 3, Correlation: math.NaN()}},
		Hierarchy: ProtocolHierarchy{
			Total: HierStats{SpectrumPct: math.NaN()},
			Nodes: []HierNode{{Stats: HierStats{TimePct: math.Inf(1)}}},
		},
	}

	s.Sanitize()

	// A successful marshal is the real assertion: encoding/json errors on NaN/Inf.
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal after Sanitize: %v", err)
	}
	if strings.Contains(string(b), "NaN") || strings.Contains(string(b), "Inf") {
		t.Fatalf("sanitized JSON still contains NaN/Inf: %s", b)
	}

	// Spot-check a few clamped fields.
	if s.WindowSec != 0 || s.Bursts[0].SpectralFlatness != 0 || s.Channels[0].PeriodSec != 0 {
		t.Fatalf("non-finite fields not clamped to 0: %+v", s)
	}
	if got := s.Channels[0].BurstLenHist.Edges[0]; got != 0 {
		t.Fatalf("hist edge not clamped: got %v", got)
	}
}

func TestSceneJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := &Scene{
		Source:   "fixture.cfile",
		CenterHz: 451_000_000,
		SpanHz:   2_400_000,
		Bursts: []Burst{{
			ID: 1, FreqHz: 451_012_500, StartSec: 0.5, EndSec: 1.25,
			Class: survey.ClassC4FM, OccupiedBwHz: 12_500, SpectralFlatness: 0.3,
		}},
		Channels: []Channel{{FreqHz: 451_012_500, BurstCount: 1, DutyCycle: 0.2}},
		Hierarchy: ProtocolHierarchy{
			Total: HierStats{BurstCount: 1, AirtimeSec: 0.75},
			Nodes: []HierNode{{Depth: 0, Label: "c4fm", Class: survey.ClassC4FM}},
		},
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Scene
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Source != want.Source || got.CenterHz != want.CenterHz {
		t.Fatalf("header round-trip mismatch: %+v", got)
	}
	if len(got.Bursts) != 1 || got.Bursts[0].Class != survey.ClassC4FM || got.Bursts[0].DurationSec() != 0.75 {
		t.Fatalf("burst round-trip mismatch: %+v", got.Bursts)
	}
	if len(got.Hierarchy.Nodes) != 1 || got.Hierarchy.Nodes[0].Label != "c4fm" {
		t.Fatalf("hierarchy round-trip mismatch: %+v", got.Hierarchy)
	}
}

func TestBurstDurationNeverNegative(t *testing.T) {
	t.Parallel()
	b := Burst{StartSec: 2.0, EndSec: 1.0}
	if b.DurationSec() != 0 {
		t.Fatalf("expected clamped 0 for inverted burst, got %v", b.DurationSec())
	}
}
