package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// SettingsPanel renders each configured trunking system with its
// per-protocol FEC opt-in state — the source of truth for whether
// the live decoder is running the spec's FEC chain or the legacy
// raw-bit fixture path.
//
// The panel is read-only. Runtime mutation requires editing
// config.yaml + restarting the daemon (the opt-ins flow from
// SystemConfig → trunking.System → ccdecoder.PipelineFactory at
// startup; the connector reads them once per HuntProgress retune).
type SettingsPanel struct {
	tbl   table.Model
	count int
}

func NewSettings() *SettingsPanel {
	t := table.New(
		table.WithColumns(settingsColumns(80)),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return &SettingsPanel{tbl: t}
}

func (SettingsPanel) Title() string       { return "Settings" }
func (SettingsPanel) Keys() []key.Binding { return nil }

func (p *SettingsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	if len(s.Systems) != p.count {
		p.refresh(s.Systems)
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *SettingsPanel) refresh(sys []client.SystemDTO) {
	rows := make([]table.Row, 0, len(sys))
	for _, s := range sys {
		rows = append(rows, table.Row{
			s.Name,
			s.Protocol,
			fecSummary(s),
		})
	}
	p.tbl.SetRows(rows)
	p.count = len(sys)
}

func (p *SettingsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(settingsColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 4)
	}
	body := p.tbl.View()
	body += "\n\nEdit config.yaml + restart daemon to change; see the FEC opt-ins section in README.md."
	return panelFrame("Settings", width, height, focused, body)
}

func settingsColumns(w int) []table.Column {
	if w < 50 {
		w = 50
	}
	nameW := w * 22 / 100
	protoW := 11
	fecW := w - nameW - protoW - 4
	if fecW < 20 {
		fecW = 20
	}
	return []table.Column{
		{Title: "Name", Width: nameW},
		{Title: "Protocol", Width: protoW},
		{Title: "FEC opt-ins", Width: fecW},
	}
}

// fecSummary returns a one-line, protocol-scoped summary of the
// system's FEC opt-in state. Only the keys relevant to the system's
// protocol are emitted so the column doesn't drown in N/A noise.
func fecSummary(s client.SystemDTO) string {
	var parts []string
	switch strings.ToLower(s.Protocol) {
	case "tetra":
		if s.TETRAColourCode != 0 {
			ch := s.TETRAChannel
			if ch == "" {
				ch = "sch/hd"
			}
			parts = append(parts, fmt.Sprintf("channel coding: on (colour=%#x, %s)", s.TETRAColourCode, ch))
		} else {
			parts = append(parts, "channel coding: off (set tetra_colour_code to enable)")
		}
	case "ltr":
		parts = append(parts, "fcs: "+orOff(s.LTRFCSMode))
		parts = append(parts, "manchester: "+orOff(s.LTRManchesterMode))
	case "p25-phase2":
		parts = append(parts, "trellis: "+orOff(s.P25Phase2TrellisMode))
	case "nxdn":
		parts = append(parts, "viterbi: "+orOff(s.NXDNViterbiMode))
	case "edacs":
		parts = append(parts, "bch: "+orOff(s.EDACSBCHMode))
	case "mpt1327":
		parts = append(parts, "bch: "+orOff(s.MPT1327BCHMode))
	case "motorola":
		parts = append(parts, "bch: "+orOff(s.MotorolaBCHMode))
	default:
		parts = append(parts, "—")
	}
	return strings.Join(parts, "  ·  ")
}

// orOff returns the supplied mode string, or "off" when empty —
// the canonical legacy / not-configured rendering across protocols.
func orOff(s string) string {
	if s == "" {
		return "off"
	}
	return s
}
