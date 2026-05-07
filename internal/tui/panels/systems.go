package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// SystemsPanel renders the configured trunking systems.
type SystemsPanel struct {
	tbl    table.Model
	count  int
}

func NewSystems() *SystemsPanel {
	t := table.New(
		table.WithColumns(systemsColumns(80)),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return &SystemsPanel{tbl: t}
}

func (SystemsPanel) Title() string       { return "Systems" }
func (SystemsPanel) Keys() []key.Binding { return nil }

func (p *SystemsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	if len(s.Systems) != p.count {
		p.refresh(s.Systems)
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *SystemsPanel) refresh(sys []client.SystemDTO) {
	rows := make([]table.Row, 0, len(sys))
	for _, s := range sys {
		ccs := "—"
		if len(s.ControlChannels) > 0 {
			ccs = fmt.Sprintf("%d (%s)", len(s.ControlChannels), client.FormatFreqMHz(s.ControlChannels[0]))
		}
		ids := []string{}
		if s.WACN != 0 {
			ids = append(ids, fmt.Sprintf("WACN %X", s.WACN))
		}
		if s.SystemID != 0 {
			ids = append(ids, fmt.Sprintf("SYS %X", s.SystemID))
		}
		if s.RFSS != 0 || s.Site != 0 {
			ids = append(ids, fmt.Sprintf("RFSS %d/Site %d", s.RFSS, s.Site))
		}
		rows = append(rows, table.Row{s.Name, s.Protocol, ccs, strings.Join(ids, " ")})
	}
	p.tbl.SetRows(rows)
	p.count = len(sys)
}

func (p *SystemsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(systemsColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	return panelFrame("Systems", width, height, focused, p.tbl.View())
}

func systemsColumns(w int) []table.Column {
	if w < 40 {
		w = 40
	}
	nameW := w * 30 / 100
	protoW := 10
	ccW := w * 25 / 100
	idsW := w - nameW - protoW - ccW - 4
	if idsW < 10 {
		idsW = 10
	}
	return []table.Column{
		{Title: "Name", Width: nameW},
		{Title: "Protocol", Width: protoW},
		{Title: "Control channels", Width: ccW},
		{Title: "IDs", Width: idsW},
	}
}

// panelFrame is shared by all table panels: rounded border, title,
// focus colour.
func panelFrame(title string, w, h int, focused bool, body string) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	if focused {
		border = border.BorderForeground(lipgloss.Color("39"))
	}
	if w > 4 {
		border = border.Width(w - 2)
	}
	if h > 4 {
		border = border.Height(h - 2)
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(title)
	return border.Render(header + "\n" + body)
}

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("39"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("231")).
		Background(lipgloss.Color("57"))
	return s
}
