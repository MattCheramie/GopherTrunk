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
	tbl      table.Model
	lastHash uint64
}

func NewActive() *ActivePanel {
	t := table.New(table.WithFocused(true), table.WithColumns(activeColumns(80)))
	t.SetStyles(tableStyles())
	return &ActivePanel{tbl: t}
}

func (ActivePanel) Title() string { return "Active calls" }

var activeEndKey = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "end call"))

func (ActivePanel) Keys() []key.Binding { return []key.Binding{activeEndKey} }

func (p *ActivePanel) selectedCall(s *state.SharedState) (client.ActiveCallDTO, bool) {
	idx := p.tbl.Cursor()
	if idx < 0 || idx >= len(s.ActiveCalls) {
		return client.ActiveCallDTO{}, false
	}
	return s.ActiveCalls[idx], true
}

func (p *ActivePanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	applyThemeIfChanged(msg, &p.tbl)
	if km, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(km, activeEndKey) {
			ac, found := p.selectedCall(s)
			if !found {
				return p, nil
			}
			req := state.WriteRequest{
				Confirm: fmt.Sprintf("End call on device %s (TG %d)?", ac.DeviceSerial, ac.Grant.GroupID),
				Label:   "ended call on " + ac.DeviceSerial,
				Kind:    state.WriteKindEndCall,
				EndCall: &state.EndCallReq{DeviceSerial: ac.DeviceSerial, Reason: "manual"},
			}
			return p, Emit(req)
		}
	}
	// Refresh on every update — the polling cadence is 1 s. The hash
	// gate keeps us from rebuilding the bubbles/table on every Update
	// call when nothing has changed.
	h := hashRows(s.ActiveCalls, func(ac client.ActiveCallDTO) string {
		return fmt.Sprintf("%s|%d|%d|%s|%d|%v|%v",
			ac.DeviceSerial, ac.Grant.GroupID, ac.Grant.SourceID,
			ac.Grant.System, ac.Grant.FrequencyHz,
			ac.Grant.Encrypted, ac.Grant.Emergency)
	})
	if h != p.lastHash {
		p.refresh(s.ActiveCalls)
		p.lastHash = h
	}
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
}

// HandleMouse moves the cursor on a left-click and forwards wheel
// ticks. After picking a row the `e` keybind ends that call.
func (p *ActivePanel) HandleMouse(msg tea.MouseMsg, localY int) tea.Cmd {
	handleTableMouse(&p.tbl, msg, localY, 0)
	return nil
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
