package panels

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

type talkgroupsSort int

const (
	tgSortID talkgroupsSort = iota
	tgSortAlpha
	tgSortPriority
)

func (s talkgroupsSort) String() string {
	switch s {
	case tgSortID:
		return "ID"
	case tgSortAlpha:
		return "AlphaTag"
	case tgSortPriority:
		return "Priority"
	}
	return "?"
}

// TalkgroupsPanel renders all configured talkgroups with a
// substring filter and sort cycle.
type TalkgroupsPanel struct {
	tbl       table.Model
	filter    textinput.Model
	editing   bool
	sortBy    talkgroupsSort
	lastCount int
	lastSort  talkgroupsSort
	lastQuery string
}

func NewTalkgroups() *TalkgroupsPanel {
	t := table.New(table.WithFocused(true), table.WithColumns(talkgroupsColumns(80)))
	t.SetStyles(tableStyles())
	in := textinput.New()
	in.Placeholder = "filter…"
	in.CharLimit = 64
	in.Width = 32
	in.Prompt = "/"
	return &TalkgroupsPanel{tbl: t, filter: in, sortBy: tgSortID}
}

func (TalkgroupsPanel) Title() string { return "Talkgroups" }

var (
	tgFilterKey = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	tgSortKey   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort"))
	tgEscKey    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit filter"))
)

func (TalkgroupsPanel) Keys() []key.Binding {
	return []key.Binding{tgFilterKey, tgSortKey}
}

// FilterValue exposes the current filter for testing.
func (p *TalkgroupsPanel) FilterValue() string { return p.filter.Value() }

// SetFilterValue sets the filter from a test harness without keystroke
// simulation.
func (p *TalkgroupsPanel) SetFilterValue(v string) {
	p.filter.SetValue(v)
	p.lastCount = -1 // force refresh on next Update
}

// RowCount returns the number of rows currently in the table.
func (p *TalkgroupsPanel) RowCount() int { return len(p.tbl.Rows()) }

func (p *TalkgroupsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		if p.editing {
			switch {
			case key.Matches(m, tgEscKey):
				p.editing = false
				p.filter.Blur()
				return p, nil
			}
			var cmd tea.Cmd
			p.filter, cmd = p.filter.Update(msg)
			p.refresh(s.Talkgroups)
			return p, cmd
		}
		switch {
		case key.Matches(m, tgFilterKey):
			p.editing = true
			p.filter.Focus()
			return p, textinput.Blink
		case key.Matches(m, tgSortKey):
			p.sortBy = (p.sortBy + 1) % 3
			p.lastSort = -1 // force refresh
			p.refresh(s.Talkgroups)
			return p, nil
		}
	}
	if p.lastCount != len(s.Talkgroups) || p.lastSort != p.sortBy || p.lastQuery != p.filter.Value() {
		p.refresh(s.Talkgroups)
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *TalkgroupsPanel) refresh(tgs []client.TalkgroupDTO) {
	q := p.filter.Value()
	filtered := make([]client.TalkgroupDTO, 0, len(tgs))
	for _, tg := range tgs {
		if q != "" && !(containsFold(tg.AlphaTag, q) || containsFold(tg.Description, q) || containsFold(tg.Tag, q) || containsFold(tg.Group, q)) {
			continue
		}
		filtered = append(filtered, tg)
	}
	switch p.sortBy {
	case tgSortID:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	case tgSortAlpha:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].AlphaTag < filtered[j].AlphaTag })
	case tgSortPriority:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Priority > filtered[j].Priority })
	}
	rows := make([]table.Row, 0, len(filtered))
	for _, tg := range filtered {
		lock := ""
		if tg.Lockout {
			lock = "✓"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", tg.ID),
			tg.AlphaTag,
			tg.Tag,
			tg.Group,
			tg.Mode,
			fmt.Sprintf("%d", tg.Priority),
			lock,
		})
	}
	p.tbl.SetRows(rows)
	p.lastCount = len(tgs)
	p.lastSort = p.sortBy
	p.lastQuery = q
}

func (p *TalkgroupsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(talkgroupsColumns(width))
	p.tbl.SetWidth(width)
	if height > 6 {
		p.tbl.SetHeight(height - 4)
	}
	body := p.filter.View() + "  " + dashDim.Render(fmt.Sprintf("(sort: %s, %d rows)", p.sortBy.String(), len(p.tbl.Rows()))) + "\n" + p.tbl.View()
	return panelFrame("Talkgroups", width, height, focused, body)
}

func talkgroupsColumns(w int) []table.Column {
	if w < 60 {
		w = 60
	}
	idW := 8
	alphaW := w * 22 / 100
	tagW := w * 16 / 100
	groupW := w * 18 / 100
	modeW := 10
	priW := 6
	lockW := 6
	used := idW + alphaW + tagW + groupW + modeW + priW + lockW + 14
	if used >= w {
		alphaW -= used - w + 1
	}
	return []table.Column{
		{Title: "ID", Width: idW},
		{Title: "Alpha", Width: alphaW},
		{Title: "Tag", Width: tagW},
		{Title: "Group", Width: groupW},
		{Title: "Mode", Width: modeW},
		{Title: "Pri", Width: priW},
		{Title: "Lock", Width: lockW},
	}
}
