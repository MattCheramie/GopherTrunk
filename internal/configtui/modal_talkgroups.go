package configtui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

var tgCols = []string{"Decimal", "Alpha", "Description", "Tag", "Group", "Mode"}

// talkgroupModal edits the staged talkgroup CSV rows for the current system as
// a small table (one column per CSV field).
type talkgroupModal struct {
	rel     string
	sysName string
	rows    []configbuilder.TalkgroupCSVRow
	rowIdx  int
	colIdx  int
	editing bool
	input   textinput.Model
}

func newTalkgroupModal(m *Model) modal {
	sys := m.cur()
	name := sys.FieldByName("Name").String()
	rel := sys.FieldByName("TalkgroupFile").String()
	if rel == "" {
		rel = slug(name) + "-talkgroups.csv"
	}
	rows := append([]configbuilder.TalkgroupCSVRow(nil), m.talkgroups[rel]...)
	ti := textinput.New()
	return &talkgroupModal{rel: rel, sysName: name, rows: rows, input: ti}
}

// commit stages the working rows back into the model and ensures the system
// references the sidecar.
func (tm *talkgroupModal) commit(m *Model) {
	m.talkgroups[tm.rel] = append([]configbuilder.TalkgroupCSVRow(nil), tm.rows...)
	if f := m.cur().FieldByName("TalkgroupFile"); f.IsValid() && f.String() == "" {
		f.SetString(tm.rel)
	}
	m.dirty = true
}

func (tm *talkgroupModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	if tm.editing {
		switch msg.String() {
		case "esc":
			tm.editing = false
			return tm, nil
		case "enter":
			tm.setCell(tm.input.Value())
			tm.editing = false
			tm.commit(m)
			return tm, nil
		}
		var cmd tea.Cmd
		tm.input, cmd = tm.input.Update(msg)
		return tm, cmd
	}

	switch msg.String() {
	case "esc", "q":
		return nil, nil
	case "up", "k":
		if tm.rowIdx > 0 {
			tm.rowIdx--
		}
	case "down", "j":
		if tm.rowIdx < len(tm.rows)-1 {
			tm.rowIdx++
		}
	case "left", "h":
		if tm.colIdx > 0 {
			tm.colIdx--
		}
	case "right", "l":
		if tm.colIdx < len(tgCols)-1 {
			tm.colIdx++
		}
	case "a":
		tm.rows = append(tm.rows, configbuilder.TalkgroupCSVRow{})
		tm.rowIdx = len(tm.rows) - 1
		tm.commit(m)
	case "d":
		if len(tm.rows) > 0 {
			tm.rows = append(tm.rows[:tm.rowIdx], tm.rows[tm.rowIdx+1:]...)
			if tm.rowIdx >= len(tm.rows) && tm.rowIdx > 0 {
				tm.rowIdx--
			}
			tm.commit(m)
		}
	case "enter":
		if len(tm.rows) > 0 {
			tm.editing = true
			tm.input.SetValue(tm.cellValue(tm.rowIdx, tm.colIdx))
			tm.input.CursorEnd()
			tm.input.Focus()
		}
	}
	return tm, nil
}

func (tm *talkgroupModal) cellValue(row, col int) string {
	r := tm.rows[row]
	switch col {
	case 0:
		return strconv.FormatUint(uint64(r.Decimal), 10)
	case 1:
		return r.AlphaTag
	case 2:
		return r.Description
	case 3:
		return r.Tag
	case 4:
		return r.Group
	case 5:
		return r.Mode
	}
	return ""
}

func (tm *talkgroupModal) setCell(text string) {
	r := &tm.rows[tm.rowIdx]
	switch tm.colIdx {
	case 0:
		if n, err := strconv.ParseUint(strings.TrimSpace(text), 10, 32); err == nil {
			r.Decimal = uint32(n)
		}
	case 1:
		r.AlphaTag = text
	case 2:
		r.Description = text
	case 3:
		r.Tag = text
	case 4:
		r.Group = text
	case 5:
		r.Mode = text
	}
}

func (tm *talkgroupModal) View(w, h int) string {
	var b strings.Builder
	b.WriteString(stMuted.Render(tm.rel) + "\n\n")
	// header
	b.WriteString("   ")
	for c, name := range tgCols {
		cell := padRight(name, tgColWidth(c))
		if c == tm.colIdx {
			cell = stAccent.Render(cell)
		}
		b.WriteString(cell + " ")
	}
	b.WriteString("\n")
	if len(tm.rows) == 0 {
		b.WriteString(stMuted.Render("(no talkgroups — press 'a' to add)") + "\n")
	}
	start := 0
	if tm.rowIdx > 12 {
		start = tm.rowIdx - 12
	}
	for i := start; i < len(tm.rows) && i < start+15; i++ {
		cur := "  "
		if i == tm.rowIdx {
			cur = stCursor.Render("▸ ")
		}
		b.WriteString(cur)
		for c := range tgCols {
			val := tm.cellValue(i, c)
			if i == tm.rowIdx && c == tm.colIdx && tm.editing {
				val = tm.input.Value() + "█"
			}
			b.WriteString(padRight(trunc(val, tgColWidth(c)), tgColWidth(c)) + " ")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + stMuted.Render(fmt.Sprintf("%d talkgroup(s)   [↑↓←→] move  [enter] edit  [a] add  [d] remove  [esc] close", len(tm.rows))))
	return boxTitle("Talkgroups — "+tm.sysName, b.String())
}

func tgColWidth(c int) int {
	switch c {
	case 0:
		return 8 // decimal
	case 1:
		return 16 // alpha
	case 2:
		return 20 // description
	default:
		return 12
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
