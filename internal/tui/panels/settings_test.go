package panels

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestSettingsPanel_RendersFECSummaryPerProtocol(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{
		Systems: []client.SystemDTO{
			{Name: "Metro", Protocol: "tetra", TETRAColourCode: 0x12345, TETRAChannel: "sch/f"},
			{Name: "Suburb", Protocol: "tetra"},
			{Name: "Country", Protocol: "ltr", LTRFCSMode: "on", LTRManchesterMode: "soft"},
			{Name: "P2Sys", Protocol: "p25-phase2", P25Phase2TrellisMode: "on"},
			{Name: "Mining", Protocol: "edacs", EDACSBCHMode: "on"},
			{Name: "UK", Protocol: "mpt1327", MPT1327BCHMode: "on"},
			{Name: "PSC", Protocol: "nxdn", NXDNViterbiMode: "on"},
		},
	}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 140, Height: 30}, s)
	if p.count != 7 {
		t.Fatalf("rows = %d, want 7", p.count)
	}
	view := p.View(140, 30, true, s)
	wants := []string{
		"Settings",
		"Metro", "channel coding: on", "0x12345", "sch/f",
		"Suburb", "channel coding: off",
		"Country", "fcs: on", "manchester: soft",
		"P2Sys", "trellis: on",
		"Mining", "bch: on",
		"UK", "bch: on",
		"PSC", "viterbi: on",
		"config.yaml",
	}
	for _, w := range wants {
		if !strings.Contains(view, w) {
			t.Errorf("view missing %q in:\n%s", w, view)
		}
	}
}

func TestSettingsPanel_UnknownProtocolFallsBackToDash(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{
		Systems: []client.SystemDTO{
			{Name: "Strange", Protocol: "smartzone"},
		},
	}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20}, s)
	if got := fecSummary(s.Systems[0]); got != "—" {
		t.Errorf("fecSummary for unknown protocol = %q, want %q", got, "—")
	}
}

func TestSettingsPanel_EmptyModeRendersOff(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "off"},
		{"on", "on"},
		{"soft", "soft"},
	}
	for _, c := range cases {
		if got := orOff(c.in); got != c.want {
			t.Errorf("orOff(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
