package rfscope

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func hierFixture() *Scene {
	return &Scene{
		CenterHz:  450_000_000,
		SpanHz:    1_000_000,
		WindowSec: 10,
		Bursts: []Burst{
			// Two C4FM bursts on the same channel.
			{ID: 1, FreqHz: 450_100_000, StartSec: 0, EndSec: 1.0, OccupiedBwHz: 12_500,
				Class: survey.ClassC4FM, Features: survey.ClassFeatures{BaudHz: 4800}},
			{ID: 2, FreqHz: 450_100_000, StartSec: 2.0, EndSec: 2.5, OccupiedBwHz: 12_500,
				Class: survey.ClassC4FM, Features: survey.ClassFeatures{BaudHz: 4800}},
			// One analog NBFM burst on another channel.
			{ID: 3, FreqHz: 450_300_000, StartSec: 0, EndSec: 2.0, OccupiedBwHz: 12_500,
				Class: survey.ClassNBFM},
		},
	}
}

func findNode(nodes []HierNode, depth int, label string) (HierNode, bool) {
	for _, n := range nodes {
		if n.Depth == depth && n.Label == label {
			return n, true
		}
	}
	return HierNode{}, false
}

func TestHierarchyAggregatesAndOrders(t *testing.T) {
	t.Parallel()
	sc := hierFixture()
	if err := (hierarchyAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if sc.Hierarchy.Total.BurstCount != 3 || sc.Hierarchy.Total.AirtimeSec != 3.5 {
		t.Fatalf("totals wrong: %+v", sc.Hierarchy.Total)
	}

	c4fm, ok := findNode(sc.Hierarchy.Nodes, 0, "c4fm")
	if !ok {
		t.Fatalf("missing c4fm class node: %+v", sc.Hierarchy.Nodes)
	}
	if c4fm.Stats.BurstCount != 2 || c4fm.Stats.AirtimeSec != 1.5 {
		t.Fatalf("c4fm stats: %+v", c4fm.Stats)
	}
	// SymbolVolume = 1.0*4800 + 0.5*4800 = 7200.
	if c4fm.Stats.SymbolVolume != 7200 {
		t.Fatalf("c4fm symbol volume: got %d want 7200", c4fm.Stats.SymbolVolume)
	}
	// One distinct channel of 12.5 kHz over a 1 MHz span = 1.25%.
	if c4fm.Stats.SpectrumPct < 1.24 || c4fm.Stats.SpectrumPct > 1.26 {
		t.Fatalf("c4fm spectrum pct: %v", c4fm.Stats.SpectrumPct)
	}
	// TimePct = 1.5 / 10 = 15%.
	if c4fm.Stats.TimePct < 14.9 || c4fm.Stats.TimePct > 15.1 {
		t.Fatalf("c4fm time pct: %v", c4fm.Stats.TimePct)
	}

	if _, ok := findNode(sc.Hierarchy.Nodes, 0, "nbfm"); !ok {
		t.Fatalf("missing nbfm class node")
	}

	// Deterministic ordering: the c4fm class node precedes the nbfm class node.
	var posC4FM, posNBFM = -1, -1
	for i, n := range sc.Hierarchy.Nodes {
		if n.Depth == 0 && n.Label == "c4fm" {
			posC4FM = i
		}
		if n.Depth == 0 && n.Label == "nbfm" {
			posNBFM = i
		}
	}
	if posC4FM < 0 || posNBFM < 0 || posC4FM > posNBFM {
		t.Fatalf("class ordering wrong: c4fm=%d nbfm=%d", posC4FM, posNBFM)
	}

	// A bandwidth bucket node must sit under the class node (depth 1).
	if _, ok := findNode(sc.Hierarchy.Nodes, 1, formatBandwidth(survey.SnapChannelBandwidth(12_500))); !ok {
		t.Fatalf("missing bandwidth bucket node: %+v", sc.Hierarchy.Nodes)
	}
}

func TestHierarchyViaRunner(t *testing.T) {
	t.Parallel()
	sc := hierFixture()
	if err := Run(context.Background(), sc, &Input{}, "hierarchy"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(sc.Hierarchy.Nodes) == 0 {
		t.Fatalf("runner produced no hierarchy nodes")
	}
}

func TestHierarchyEmptyScene(t *testing.T) {
	t.Parallel()
	sc := &Scene{}
	if err := (hierarchyAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze empty: %v", err)
	}
	if len(sc.Hierarchy.Nodes) != 0 {
		t.Fatalf("empty scene should yield no nodes")
	}
}
