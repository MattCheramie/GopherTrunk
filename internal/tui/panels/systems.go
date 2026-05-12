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

// SystemsPanel renders the configured trunking systems.
type SystemsPanel struct {
	tbl      table.Model
	lastHash uint64
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
	applyThemeIfChanged(msg, &p.tbl)
	h := hashRows(s.Systems, func(sys client.SystemDTO) string {
		return fmt.Sprintf("%s|%s|%v|%d|%d|%d|%d",
			sys.Name, sys.Protocol, sys.ControlChannels,
			sys.WACN, sys.SystemID, sys.RFSS, sys.Site)
	})
	if h != p.lastHash {
		p.refresh(s.Systems)
		p.lastHash = h
	}
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
		row := p.tbl.SelectedRow()
		if len(row) > 0 {
			return p, EmitSystemDetail(row[0])
		}
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
}

func (p *SystemsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(systemsColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	return panelFrame("Systems", width, height, focused, p.tbl.View())
}

// Cursor exposes the current table cursor index for tests.
func (p *SystemsPanel) Cursor() int { return p.tbl.Cursor() }

// Reveal positions the cursor on the row whose Name matches the
// given key. No-op when the row is absent; the operator still ends
// up on the panel.
func (p *SystemsPanel) Reveal(key string) {
	for i, row := range p.tbl.Rows() {
		if len(row) > 0 && row[0] == key {
			p.tbl.SetCursor(i)
			return
		}
	}
}

// HandleMouse moves the cursor on a left-click and forwards wheel
// ticks. Chrome clicks are ignored.
func (p *SystemsPanel) HandleMouse(msg tea.MouseMsg, localY int) tea.Cmd {
	handleTableMouse(&p.tbl, msg, localY, 0)
	return nil
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

