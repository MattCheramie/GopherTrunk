package configtui

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

var (
	stTitle    = lipgloss.NewStyle().Bold(true)
	stCursor   = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	stMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	stErr      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	stOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	stAccent   = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	stHelpFoot = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "loading…"
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.viewSections(), m.viewForm())
	out := body + "\n" + m.viewFooter()
	if m.modal != nil {
		overlay := m.modal.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}
	return out
}

func (m Model) viewSections() string {
	secs := configbuilder.Sections()
	configured := m.configuredSections()
	var b strings.Builder
	b.WriteString(stTitle.Render("Sections") + "\n")
	for i, s := range secs {
		cursor := "  "
		if m.focus == zoneSections && i == m.sectionIdx {
			cursor = stCursor.Render("▸ ")
		} else if i == m.sectionIdx {
			cursor = "· "
		}
		dot := stMuted.Render("○")
		if configured[s.Key] {
			dot = stAccent.Render("●")
		}
		badge := ""
		if n := len(m.validation[s.Key]); n > 0 {
			badge = " " + stErr.Render(fmt.Sprintf("✗%d", n))
		}
		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, dot, s.Label, badge))
	}
	w := 26
	return lipgloss.NewStyle().Width(w).MarginRight(1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("238")).Render(b.String())
}

func (m Model) viewForm() string {
	secs := configbuilder.Sections()
	sec := secs[m.sectionIdx]
	var b strings.Builder
	b.WriteString(stTitle.Render(m.breadcrumb(sec)) + "\n")

	// Per-section validation banner.
	if errs := m.validation[sec.Key]; len(errs) > 0 {
		for _, e := range errs {
			b.WriteString(stErr.Render("✗ "+e) + "\n")
		}
	} else if m.validation != nil {
		b.WriteString(stOK.Render("✓ section validates") + "\n")
	}
	b.WriteString("\n")

	cur := m.cur()
	switch cur.Kind() {
	case reflect.Slice:
		b.WriteString(m.viewList(cur))
	case reflect.Map:
		b.WriteString(m.viewMap(cur))
	default:
		b.WriteString(m.viewStruct(cur))
	}
	return lipgloss.NewStyle().Width(maxInt(m.width-30, 30)).Render(b.String())
}

func (m Model) breadcrumb(sec configbuilder.SectionMeta) string {
	parts := []string{sec.Label}
	v := m.root().FieldByName(sec.CfgField)
	for _, st := range m.navpath {
		if st.Field != "" {
			parts = append(parts, humanize(st.Field))
			v = v.FieldByName(st.Field)
		} else {
			parts = append(parts, fmt.Sprintf("#%d", st.Index+1))
			v = v.Index(st.Index)
		}
		for v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
	}
	return strings.Join(parts, " › ")
}

func (m Model) viewStruct(cur reflect.Value) string {
	rows := structRows(structNameOf(cur), cur)
	if len(rows) == 0 {
		return stMuted.Render("(no editable fields)")
	}
	var b strings.Builder
	lastAdvanced := false
	for i, r := range rows {
		if r.advanced && !lastAdvanced {
			b.WriteString(stMuted.Render("— advanced —") + "\n")
		}
		lastAdvanced = r.advanced
		cursor := "  "
		focused := m.focus == zoneForm && i == m.cursor
		if focused {
			cursor = stCursor.Render("▸ ")
		}
		label := padRight(r.label, 22)
		var val string
		if focused && m.editing {
			val = m.input.View()
		} else {
			val = displayValue(cur.FieldByName(r.field), r)
		}
		b.WriteString(cursor + label + " " + val + "\n")
		if focused && r.help != "" && !m.editing {
			b.WriteString(stMuted.Render("    "+r.help) + "\n")
		}
	}
	return b.String()
}

func (m Model) viewList(slice reflect.Value) string {
	var b strings.Builder
	if slice.Len() == 0 {
		b.WriteString(stMuted.Render("(empty)") + "\n")
	}
	for i := 0; i < slice.Len(); i++ {
		cursor := "  "
		if m.focus == zoneForm && i == m.cursor {
			cursor = stCursor.Render("▸ ")
		}
		b.WriteString(cursor + itemTitle(slice.Index(i), i) + "\n")
	}
	b.WriteString("\n" + stMuted.Render("[a] add  [enter] edit  [d] remove  [D] duplicate  [J/K] move  [esc] back"))
	return b.String()
}

func (m Model) viewMap(mp reflect.Value) string {
	var b strings.Builder
	keys := sortedTabKeys()
	for i, k := range keys {
		cursor := "  "
		if m.focus == zoneForm && i == m.cursor {
			cursor = stCursor.Render("▸ ")
		}
		shown := true
		if !mp.IsNil() {
			if v := mp.MapIndex(reflect.ValueOf(k)); v.IsValid() {
				shown = v.Bool()
			}
		}
		mark := "[x]"
		if !shown {
			mark = "[ ]"
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, mark, k))
	}
	b.WriteString("\n" + stMuted.Render("[space] toggle visible   [esc] back"))
	return b.String()
}

func (m Model) viewFooter() string {
	var parts []string
	if m.path != "" {
		parts = append(parts, m.path)
	} else {
		parts = append(parts, "untitled")
	}
	if m.dirty {
		parts = append(parts, stErr.Render("● unsaved"))
	}
	if m.status != "" {
		parts = append(parts, stOK.Render(m.status))
	}
	line := strings.Join(parts, "  ")
	if len(m.errs) > 0 {
		line += "\n" + stErr.Render("! "+m.errs[len(m.errs)-1])
	}
	keys := "[tab] focus  [enter] edit/drill  [s] save  [o] open  [n] new  [v] validate  [y] yaml  [R] revert  [q] quit"
	return line + "\n" + stHelpFoot.Render(keys)
}

// itemTitle summarizes a list item by its most identifying field.
func itemTitle(v reflect.Value, i int) string {
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		for _, name := range []string{"Name", "Serial", "Label", "Addr", "File"} {
			f := v.FieldByName(name)
			if f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
				return f.String()
			}
		}
		if f := v.FieldByName("Protocol"); f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
			return strings.ToUpper(f.String())
		}
		if f := v.FieldByName("KeyID"); f.IsValid() {
			return fmt.Sprintf("Key %v", f.Interface())
		}
		if f := v.FieldByName("ChannelID"); f.IsValid() {
			return fmt.Sprintf("Channel %v", f.Interface())
		}
	}
	return fmt.Sprintf("item %d", i+1)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
