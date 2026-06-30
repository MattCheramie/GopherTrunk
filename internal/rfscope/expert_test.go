package rfscope

import (
	"context"
	"testing"
)

func anomalyKinds(as []Anomaly) map[string]Anomaly {
	m := map[string]Anomaly{}
	for _, a := range as {
		m[a.Kind] = a
	}
	return m
}

func TestExpertAnalyzerRules(t *testing.T) {
	t.Parallel()
	sc := &Scene{
		SpanHz: 2_000_000, WindowSec: 10,
		Emitters: []Emitter{
			{ID: 1, Fingerprint: Fingerprint{CenterHz: 450_100_000, ClassFamily: "digital"}, HopSet: []uint32{450_100_000, 450_500_000, 450_900_000}},
		},
		Channels: []Channel{
			{FreqHz: 450_200_000, DutyCycle: 0.02, BurstCount: 5, OccupiedBwHz: 12_500, MedianFlatness: 0.1},  // intermittent
			{FreqHz: 450_700_000, DutyCycle: 0.50, BurstCount: 1, OccupiedBwHz: 12_500, MedianFlatness: 0.92}, // noise-like
		},
		AnalyzerOutputs: map[string]any{
			"entropy": []EntropyResult{
				{EmitterID: 1, FreqHz: 450_100_000, Class: "strong-encrypted", EntropyBits: 7.99, Recommended: "key material required"},
			},
		},
	}

	if err := (expertAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	kinds := anomalyKinds(sc.Anomalies)
	for _, want := range []string{"hopper", "intermittent", "noise-like", "encrypted"} {
		if _, ok := kinds[want]; !ok {
			t.Fatalf("missing %q anomaly; got %+v", want, sc.Anomalies)
		}
	}
	if kinds["encrypted"].Severity != "alert" {
		t.Fatalf("encrypted should be an alert, got %q", kinds["encrypted"].Severity)
	}
	// Alerts must sort ahead of notes.
	if sc.Anomalies[0].Severity != "alert" {
		t.Fatalf("most severe anomaly should be first, got %q", sc.Anomalies[0].Severity)
	}
}

func TestExpertWideNarrowCarrier(t *testing.T) {
	t.Parallel()
	sc := &Scene{
		SpanHz: 10_000_000, WindowSec: 10,
		Channels: []Channel{
			{FreqHz: 451_000_000, OccupiedBwHz: 200_000, DutyCycle: 0.5, BurstCount: 1}, // wide vs raster
		},
	}
	if err := (expertAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if _, ok := anomalyKinds(sc.Anomalies)["wide-carrier"]; !ok {
		t.Fatalf("want wide-carrier anomaly, got %+v", sc.Anomalies)
	}
}

func TestExpertAppendsNotOverwrites(t *testing.T) {
	t.Parallel()
	sc := &Scene{
		Anomalies: []Anomaly{{Severity: "note", Kind: "preexisting", Detail: "seeded"}},
		Emitters:  []Emitter{{ID: 1, HopSet: []uint32{1, 2}}},
	}
	if err := (expertAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if _, ok := anomalyKinds(sc.Anomalies)["preexisting"]; !ok {
		t.Fatalf("expert must append, not overwrite pre-seeded anomalies")
	}
}
