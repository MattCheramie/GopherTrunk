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

// TonesPanel renders the tone-alert ring buffer.
type TonesPanel struct {
	tbl  table.Model
	last int
}

func NewTones() *TonesPanel {
	t := table.New(table.WithFocused(true), table.WithColumns(toneColumns(80)))
	t.SetStyles(tableStyles())
	return &TonesPanel{tbl: t}
}

func (TonesPanel) Title() string       { return "Tone alerts" }
func (TonesPanel) Keys() []key.Binding { return nil }

func (p *TonesPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	if s.ToneAlerts != nil && s.ToneAlerts.Len() != p.last {
		p.refresh(s)
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *TonesPanel) refresh(s *state.SharedState) {
	all := s.ToneAlerts.Snapshot()
	rows := make([]table.Row, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- {
		ev := all[i]
		var t client.Tone
		_ = jsonUnmarshal(ev.Raw, &t)
		freqs := make([]string, 0, len(t.FrequenciesHz))
		for _, f := range t.FrequenciesHz {
			freqs = append(freqs, fmt.Sprintf("%.1f", f))
		}
		rows = append(rows, table.Row{
			ev.Time.Format("15:04:05"),
			t.Profile,
			t.DeviceSerial,
			strings.Join(freqs, " "),
		})
	}
	p.tbl.SetRows(rows)
	p.last = s.ToneAlerts.Len()
}

func (p *TonesPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(toneColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	return panelFrame(fmt.Sprintf("Tone alerts (%d)", p.last), width, height, focused, p.tbl.View())
}

func toneColumns(w int) []table.Column {
	if w < 40 {
		w = 40
	}
	timeW := 10
	profW := w * 24 / 100
	devW := w * 16 / 100
	freqW := w - timeW - profW - devW - 8
	if freqW < 8 {
		freqW = 8
	}
	return []table.Column{
		{Title: "Time", Width: timeW},
		{Title: "Profile", Width: profW},
		{Title: "Device", Width: devW},
		{Title: "Hz", Width: freqW},
	}
}
