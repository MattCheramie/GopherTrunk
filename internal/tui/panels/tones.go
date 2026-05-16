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
	tbl      table.Model
	last     int
	lastHash uint64
}

func NewTones() *TonesPanel {
	t := table.New(table.WithFocused(true), table.WithColumns(toneColumns(80)))
	t.SetStyles(tableStyles())
	return &TonesPanel{tbl: t}
}

func (TonesPanel) Title() string { return "Tone alerts" }

var toneResetKey = key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reset detector"))

func (TonesPanel) Keys() []key.Binding { return []key.Binding{toneResetKey} }

// selectedDevice extracts the device serial of the currently
// highlighted row, returning ("", false) when the table is empty.
func (p *TonesPanel) selectedDevice() (string, bool) {
	idx := p.tbl.Cursor()
	rows := p.tbl.Rows()
	if idx < 0 || idx >= len(rows) {
		return "", false
	}
	// Column 2 is "Device" per toneColumns.
	return rows[idx][2], true
}

func (p *TonesPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	applyThemeIfChanged(msg, &p.tbl)
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, toneResetKey) {
		serial, ok := p.selectedDevice()
		if !ok {
			return p, nil
		}
		req := state.WriteRequest{
			Confirm:   fmt.Sprintf("Reset tone-out match progress on device %s?", serial),
			Label:     "reset tone detector for " + serial,
			Kind:      state.WriteKindResetTone,
			ResetTone: &state.ResetToneReq{DeviceSerial: serial},
		}
		return p, Emit(req)
	}
	if s.ToneAlerts != nil {
		// Only the Len changes drive a refresh today (ring buffer is
		// append-only). Hashing the snapshot is overkill — Len is the
		// canonical invalidation signal.
		if s.ToneAlerts.Len() != p.last {
			p.refresh(s)
		}
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

// HandleMouse moves the cursor on a left-click and forwards wheel
// ticks. After picking a row the `R` keybind resets that device's
// tone-detector.
func (p *TonesPanel) HandleMouse(msg tea.MouseMsg, localY int) tea.Cmd {
	handleTableMouse(&p.tbl, msg, localY, 0)
	return nil
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
