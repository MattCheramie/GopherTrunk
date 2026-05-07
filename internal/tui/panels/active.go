package panels

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// ActivePanel renders calls currently being followed.
type ActivePanel struct {
	tbl       table.Model
	lastCount int
}

func NewActive() *ActivePanel {
	t := table.New(table.WithFocused(true), table.WithColumns(activeColumns(80)))
	t.SetStyles(tableStyles())
	return &ActivePanel{tbl: t}
}

func (ActivePanel) Title() string       { return "Active calls" }
func (ActivePanel) Keys() []key.Binding { return nil }

func (p *ActivePanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	// Refresh on every update — the polling cadence is 1 s and the
	// table is small.
	p.refresh(s.ActiveCalls)
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *ActivePanel) refresh(calls []client.ActiveCallDTO) {
	rows := make([]table.Row, 0, len(calls))
	for _, ac := range calls {
		alpha := "—"
		if ac.Talkgroup != nil && ac.Talkgroup.AlphaTag != "" {
			alpha = ac.Talkgroup.AlphaTag
		}
		flags := ""
		if ac.Grant.Encrypted {
			flags += "E"
		}
		if ac.Grant.Emergency {
			flags += "!"
		}
		rows = append(rows, table.Row{
			since(ac.StartedAt),
			fmt.Sprintf("%d", ac.Grant.GroupID),
			alpha,
			fmt.Sprintf("%d", ac.Grant.SourceID),
			ac.Grant.System,
			client.FormatFreqMHz(ac.Grant.FrequencyHz),
			ac.DeviceSerial,
			flags,
		})
	}
	p.tbl.SetRows(rows)
	p.lastCount = len(calls)
}

func (p *ActivePanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(activeColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	return panelFrame(fmt.Sprintf("Active calls (%d)", len(s.ActiveCalls)), width, height, focused, p.tbl.View())
}

func activeColumns(w int) []table.Column {
	if w < 60 {
		w = 60
	}
	startW := 6
	tgW := 8
	alphaW := w * 22 / 100
	srcW := 8
	sysW := w * 14 / 100
	freqW := 16
	devW := w - startW - tgW - alphaW - srcW - sysW - freqW - 6 - 16
	if devW < 8 {
		devW = 8
	}
	return []table.Column{
		{Title: "Started", Width: startW},
		{Title: "TG", Width: tgW},
		{Title: "Alpha", Width: alphaW},
		{Title: "Src", Width: srcW},
		{Title: "Sys", Width: sysW},
		{Title: "Freq", Width: freqW},
		{Title: "Device", Width: devW},
		{Title: "E/!", Width: 4},
	}
}
