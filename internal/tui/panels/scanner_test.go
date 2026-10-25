package panels

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func mkScannerState() *state.SharedState {
	return &state.SharedState{
		Scanner: client.ScannerStatusDTO{
			ScanMode:            "list",
			TalkgroupScanCount:  4,
			TalkgroupTotalCount: 12,
			Systems: []client.SystemHuntStatusDTO{
				{Name: "Alpha", Protocol: "p25", State: "locked", LockedFreqHz: 851_000_000},
				{Name: "Bravo", Protocol: "dmr", State: "hunting", AttemptedFreqHz: 460_000_000, AttemptIndex: 1, TotalCandidates: 3},
			},
			Conventional: client.ConvScannerStatusDTO{
				Enabled:      true,
				State:        "scanning",
				DeviceSerial: "CONV-1",
				Channels: []client.ConvChannelStatusDTO{
					{Index: 0, Label: "Sheriff", FrequencyHz: 155_895_000, Mode: "fm"},
					{Index: 1, Label: "Marine 16", FrequencyHz: 156_800_000, Mode: "fm", Active: true},
				},
			},
		},
	}
}

func TestScannerPanelRendersAllSections(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	view := p.View(120, 30, true, s)
	for _, want := range []string{
		"Trunked systems", "Alpha", "Bravo", "p25", "dmr",
		"Conventional channels", "Sheriff", "Marine 16",
		"Talkgroup scan list", "mode=list", "enabled=4 / total=12",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q\n%s", want, view)
		}
	}
}

func TestScannerPanel_HoldEmitsWriteRequest(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	// First Update populates cursor at row 0 (Alpha system).
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, s)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}, s)
	if cmd == nil {
		t.Fatal("h key should emit a WriteRequest Cmd")
	}
	msg := cmd()
	wa, ok := msg.(WriteActionMsg)
	if !ok {
		t.Fatalf("msg type = %T, want WriteActionMsg", msg)
	}
	if wa.Request.Kind != state.WriteKindScannerHuntHold {
		t.Errorf("kind = %v, want ScannerHuntHold", wa.Request.Kind)
	}
	if wa.Request.ScannerHunt == nil || wa.Request.ScannerHunt.System != "Alpha" {
		t.Errorf("system = %+v, want Alpha", wa.Request.ScannerHunt)
	}
}

func TestScannerPanel_DwellEmitsForConvRow(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	// Move cursor down past the 2 systems to the conv rows.
	for i := 0; i < 3; i++ {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, s)
	}
	// Now on conv index 1.
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s)
	if cmd == nil {
		t.Fatal("enter on conv row should emit a Cmd")
	}
	wa := cmd().(WriteActionMsg)
	if wa.Request.Kind != state.WriteKindScannerConvDwell {
		t.Errorf("kind = %v, want ScannerConvDwell", wa.Request.Kind)
	}
	if wa.Request.ScannerConv == nil || wa.Request.ScannerConv.Index != 1 {
		t.Errorf("index = %+v, want 1", wa.Request.ScannerConv)
	}
}

func TestScannerPanel_MCyclesScanMode(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")}, s)
	if cmd == nil {
		t.Fatal("m key should emit a Cmd")
	}
	wa := cmd().(WriteActionMsg)
	if wa.Request.Kind != state.WriteKindScannerMode {
		t.Errorf("kind = %v, want ScannerMode", wa.Request.Kind)
	}
	// Current is "list" → next should be "all".
	if wa.Request.ScannerMode == nil || wa.Request.ScannerMode.Mode != "all" {
		t.Errorf("mode = %+v, want all", wa.Request.ScannerMode)
	}
}

func TestScannerPanel_RetuneConfirms(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	// Cursor on Alpha (first row).
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, s)
	if cmd == nil {
		t.Fatal("r key should emit a Cmd")
	}
	wa := cmd().(WriteActionMsg)
	if wa.Request.Confirm == "" {
		t.Errorf("retune should require confirmation; Confirm = %q", wa.Request.Confirm)
	}
	if wa.Request.Kind != state.WriteKindScannerHuntRetune {
		t.Errorf("kind = %v, want ScannerHuntRetune", wa.Request.Kind)
	}
}

func TestScannerPanel_EmptyShowsHints(t *testing.T) {
	p := NewScanner()
	s := &state.SharedState{}
	view := p.View(120, 30, true, s)
	if !strings.Contains(view, "no trunked systems configured") {
		t.Errorf("missing 'no trunked systems' hint in empty render")
	}
	if !strings.Contains(view, "conventional scanner not configured") {
		t.Errorf("missing 'conventional scanner not configured' hint")
	}
}

// Issue #1288: the continuous wideband conventional channels get their own
// read-only section, and a config without them renders no such section.
func TestScannerPanelRendersWidebandConventional(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	if view := p.View(120, 40, true, s); strings.Contains(view, "Wideband conventional") {
		t.Errorf("section rendered with no wideband channels\n%s", view)
	}
	s.Scanner.Conventional.WidebandChannels = []client.WidebandConvChannelDTO{
		{Label: "Tower", FrequencyHz: 118_300_000, Mode: "am", WidebandSerial: "WB-1", Active: true},
		{Label: "Ground", FrequencyHz: 119_900_000, Mode: "am", WidebandSerial: "WB-1"},
	}
	view := p.View(120, 40, true, s)
	for _, want := range []string{"Wideband conventional channels (always on)", "Tower", "Ground", "118.3000", "on WB-1"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q\n%s", want, view)
		}
	}
}

// Issue #1288 §12: a tap whose IQ has stopped is flagged; an idle healthy one
// is not (a quiet channel is not a fault).
func TestScannerPanelFlagsStalledWidebandTap(t *testing.T) {
	p := NewScanner()
	s := mkScannerState()
	s.Scanner.Conventional.WidebandChannels = []client.WidebandConvChannelDTO{
		{Label: "Quiet", FrequencyHz: 118_300_000, Mode: "am", WidebandSerial: "WB-1", Health: "ok"},
		{Label: "Dead", FrequencyHz: 119_900_000, Mode: "am", WidebandSerial: "WB-1", Health: "stalled", IQDrops: 7},
	}
	view := p.View(140, 40, true, s)
	if n := strings.Count(view, "IQ STALLED"); n != 1 {
		t.Errorf("IQ STALLED shown %d times, want once (the stalled tap only)\n%s", n, view)
	}
	if !strings.Contains(view, "7 dropped") {
		t.Errorf("view missing the drop count\n%s", view)
	}
}
