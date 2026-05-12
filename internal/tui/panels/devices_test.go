package panels

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestDevicesPanel_RendersSnapshot(t *testing.T) {
	p := NewDevices()
	s := &state.SharedState{
		Devices: []client.SDRStatus{
			{Driver: "rtlsdr", Serial: "AAA", TunerName: "R820T2", Role: "control",
				Attached: true, GainTenthDB: 496, PPM: 1},
			{Driver: "rtlsdr", Serial: "BBB", TunerName: "R820T2", Role: "voice",
				Attached: true, GainAuto: true, BiasTee: true},
		},
	}
	// First Update populates the table.
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	if got := len(p.tbl.Rows()); got != 2 {
		t.Fatalf("rows = %d, want 2", got)
	}
	view := p.View(120, 30, true, s)
	for _, want := range []string{"AAA", "BBB", "control", "voice", "auto", "49.6 dB"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q\n%s", want, view)
		}
	}
}

func TestDevicesPanel_EmptyHint(t *testing.T) {
	p := NewDevices()
	s := &state.SharedState{}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	view := p.View(120, 30, true, s)
	if !strings.Contains(view, "no devices opened") {
		t.Errorf("empty hint missing\n%s", view)
	}
}
