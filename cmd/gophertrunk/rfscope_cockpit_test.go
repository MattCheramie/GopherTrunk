package main

import (
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/rfscope"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	tea "github.com/charmbracelet/bubbletea"
)

func cockpitScene() *rfscope.Scene {
	tl := make([]rfscope.Bin, 20)
	for i := range tl {
		tl[i].Occupied = i < 10 // first half active
	}
	return &rfscope.Scene{
		CenterHz: 450_000_000, SpanHz: 2_000_000, WindowSec: 10, NoiseFloorDbFS: -90,
		Bursts:   []rfscope.Burst{{ID: 1, FreqHz: 450_100_000}},
		Channels: []rfscope.Channel{{FreqHz: 450_100_000, DominantClass: survey.ClassC4FM, DutyCycle: 0.5, Timeline: tl}},
		Emitters: []rfscope.Emitter{{ID: 1, Label: "digital@450.1000MHz", TotalAirtimeSec: 1.5, HopSet: []uint32{450_100_000, 450_500_000}}},
		Conversations: []rfscope.Conversation{
			{ID: 1, EmitterIDs: []uint64{1, 2}, Kind: "co-active", Correlation: 1},
		},
		Hierarchy: rfscope.ProtocolHierarchy{
			Nodes: []rfscope.HierNode{{Depth: 0, Label: "c4fm", Stats: rfscope.HierStats{BurstCount: 1, TimePct: 100}}},
		},
		Anomalies: []rfscope.Anomaly{{Severity: "alert", Kind: "encrypted", Detail: "strong-encrypted"}},
	}
}

func TestRenderSparkline(t *testing.T) {
	t.Parallel()
	tl := make([]rfscope.Bin, 20)
	for i := range tl {
		tl[i].Occupied = i < 10
	}
	s := renderSparkline(tl, 20)
	if len([]rune(s)) != 20 {
		t.Fatalf("sparkline width = %d, want 20", len([]rune(s)))
	}
	if !strings.ContainsRune(s, '█') || !strings.ContainsRune(s, '·') {
		t.Fatalf("sparkline should mix active/idle glyphs: %q", s)
	}
	// Empty timeline ⇒ all idle.
	if got := renderSparkline(nil, 8); got != strings.Repeat("·", 8) {
		t.Fatalf("empty sparkline = %q", got)
	}
}

func TestRenderPanels(t *testing.T) {
	t.Parallel()
	sc := cockpitScene()
	cases := map[string]string{
		renderSceneHeader(sc):        "450.0000 MHz",
		renderHierarchyPanel(sc):     "c4fm",
		renderChannelsPanel(sc, 20):  "450.1000",
		renderTalkersPanel(sc):       "digital@450.1000MHz",
		renderConversationsPanel(sc): "co-active",
		renderExpertPanel(sc):        "encrypted",
	}
	for out, want := range cases {
		if !strings.Contains(out, want) {
			t.Fatalf("panel missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(renderTalkersPanel(sc), "hop×2") {
		t.Fatalf("talkers panel should mark the hopper")
	}
	if got := sortedAnomalyKinds(sc); len(got) != 1 || got[0] != "encrypted" {
		t.Fatalf("sortedAnomalyKinds = %v", got)
	}
}

func TestRenderPanelsEmptyScene(t *testing.T) {
	t.Parallel()
	sc := &rfscope.Scene{}
	for _, out := range []string{
		renderHierarchyPanel(sc), renderChannelsPanel(sc, 20),
		renderTalkersPanel(sc), renderConversationsPanel(sc), renderExpertPanel(sc),
	} {
		if !strings.Contains(out, "(none)") {
			t.Fatalf("empty panel should say (none): %q", out)
		}
	}
}

func TestCockpitModelUpdate(t *testing.T) {
	t.Parallel()
	m := newCockpit(rfscope.NewMemSource(nil, 0, 1e6), rfscope.SegmentConfig{}, nil, "test", time.Second, false)

	// A scene message populates the model.
	upd, _ := m.Update(cockpitSceneMsg{scene: cockpitScene()})
	cm := upd.(cockpitModel)
	if cm.scene == nil {
		t.Fatalf("scene message did not populate the model")
	}
	if !strings.Contains(cm.View(), "Protocol hierarchy") {
		t.Fatalf("view should render panels once a scene is present")
	}

	// Freeze, then a new scene must NOT replace the frozen one.
	frozen, _ := cm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	fm := frozen.(cockpitModel)
	if !fm.frozen {
		t.Fatalf("space should freeze the view")
	}
	other := cockpitScene()
	other.CenterHz = 999
	after, _ := fm.Update(cockpitSceneMsg{scene: other})
	am := after.(cockpitModel)
	if am.scene.CenterHz == 999 {
		t.Fatalf("frozen view must not update from a new scene")
	}

	// 'q' quits.
	_, cmd := am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("q should return a quit command")
	}
}
