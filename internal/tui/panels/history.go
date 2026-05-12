package panels

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// HistoryPanel renders ended calls from /api/v1/calls/history.
//
// Loading is on-demand: the root model issues the initial fetch
// when this panel becomes active and again when the user presses
// `r`. The panel does not poll continuously — history is a stable
// list and the daemon already paginates it.
type HistoryPanel struct {
	tbl      table.Model
	lastHash uint64
	reloadAt time.Time
}

// NewHistory constructs the panel.
func NewHistory() *HistoryPanel {
	t := table.New(table.WithFocused(true), table.WithColumns(historyColumns(80)))
	t.SetStyles(tableStyles())
	return &HistoryPanel{tbl: t}
}

func (HistoryPanel) Title() string { return "Call history" }

var historyReloadKey = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))

func (HistoryPanel) Keys() []key.Binding { return []key.Binding{historyReloadKey} }

// ReloadRequested returns true when the user pressed `r` since the
// last call. The root model checks this each Update and issues the
// reload Cmd as needed.
func (p *HistoryPanel) ReloadRequested() bool {
	if p.reloadAt.IsZero() {
		return false
	}
	p.reloadAt = time.Time{}
	return true
}

func (p *HistoryPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(m, historyReloadKey) {
			p.reloadAt = time.Now()
			return p, nil
		}
	}
	h := hashRows(s.History, func(r client.CallRow) string {
		return fmt.Sprintf("%s|%s|%d|%s|%d|%s|%s",
			r.StartedAt.Format(time.RFC3339Nano),
			r.EndedAt.Format(time.RFC3339Nano),
			r.GroupID, r.TalkgroupAlpha, r.SourceID,
			r.System, r.EndReason)
	})
	if h != p.lastHash {
		p.refresh(s.History)
		p.lastHash = h
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *HistoryPanel) refresh(rows []client.CallRow) {
	out := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		alpha := r.TalkgroupAlpha
		if alpha == "" {
			alpha = "—"
		}
		ended := "—"
		if !r.EndedAt.IsZero() {
			ended = r.EndedAt.Format("15:04:05")
		}
		out = append(out, table.Row{
			r.StartedAt.Format("01-02 15:04:05"),
			ended,
			fmt.Sprintf("%d", r.GroupID),
			alpha,
			fmt.Sprintf("%d", r.SourceID),
			r.System,
			r.EndReason,
		})
	}
	p.tbl.SetRows(out)
}

func (p *HistoryPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(historyColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	footer := dashDim.Render(fmt.Sprintf("%d rows — press r to reload", len(s.History)))
	if s.HistoryErr != nil {
		footer = dashErr.Render(s.HistoryErr.Error())
	}
	body := p.tbl.View() + "\n" + footer
	return panelFrame("Call history", width, height, focused, body)
}

func historyColumns(w int) []table.Column {
	if w < 60 {
		w = 60
	}
	startW := 16
	endW := 10
	tgW := 8
	alphaW := w * 22 / 100
	srcW := 8
	sysW := w * 12 / 100
	reasonW := w - startW - endW - tgW - alphaW - srcW - sysW - 14
	if reasonW < 8 {
		reasonW = 8
	}
	return []table.Column{
		{Title: "Started", Width: startW},
		{Title: "Ended", Width: endW},
		{Title: "TG", Width: tgW},
		{Title: "Alpha", Width: alphaW},
		{Title: "Src", Width: srcW},
		{Title: "Sys", Width: sysW},
		{Title: "Reason", Width: reasonW},
	}
}
