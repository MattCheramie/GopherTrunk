package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// importTUIModel drives the post-parse review/edit flow. Three views,
// pushed in sequence:
//
//   - viewSystems     – top-level list of imported systems
//   - viewSystemTabs  – per-system view with a Sites / Talkgroups tab toggle
//   - viewEditAlpha   – modal text input for renaming a talkgroup's Alpha Tag
//
// Hotkeys (global): w write+exit, q quit-without-writing, ?/h help.
// Sites tab:        space toggle Include
// Talkgroups tab:   s scan, l lockout, e edit alpha, 0-9 priority
type importTUIModel struct {
	systems []parsedSystem
	sysIdx  int
	view    tuiView
	tab     tuiTab
	cursor  int
	editing bool
	editBuf string
	writeFn func([]parsedSystem) (mergeResult, error)
	status  string
	confirm string
	width   int
	height  int
	wrote   bool
}

type tuiView int

const (
	viewSystems tuiView = iota
	viewSystemTabs
)

type tuiTab int

const (
	tabSites tuiTab = iota
	tabTalkgroups
)

// newImportTUI is the constructor used by runImport.
func newImportTUI(systems []parsedSystem, writeFn func([]parsedSystem) (mergeResult, error)) importTUIModel {
	return importTUIModel{
		systems: systems,
		view:    viewSystems,
		writeFn: writeFn,
	}
}

// Init satisfies tea.Model.
func (m importTUIModel) Init() tea.Cmd { return nil }

// Update is the central event loop. Hotkeys are dispatched by view/tab.
func (m importTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.editing {
			return m.updateEditAlpha(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m importTUIModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "w":
		_, err := m.writeFn(m.systems)
		if err != nil {
			m.status = "ERROR: " + err.Error()
			return m, nil
		}
		m.wrote = true
		m.status = "Wrote config + CSVs. Press q to exit."
		return m, tea.Quit
	}

	switch m.view {
	case viewSystems:
		return m.updateSystemsView(msg)
	case viewSystemTabs:
		return m.updateSystemTabsView(msg)
	}
	return m, nil
}

func (m importTUIModel) updateSystemsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.sysIdx > 0 {
			m.sysIdx--
		}
	case "down", "j":
		if m.sysIdx < len(m.systems)-1 {
			m.sysIdx++
		}
	case "enter", "right", "l":
		if len(m.systems) > 0 {
			m.view = viewSystemTabs
			m.tab = tabSites
			m.cursor = 0
		}
	}
	return m, nil
}

func (m importTUIModel) updateSystemTabsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sys := &m.systems[m.sysIdx]
	switch msg.String() {
	case "esc", "left", "h":
		m.view = viewSystems
		m.cursor = 0
		return m, nil
	case "tab", "T":
		if m.tab == tabSites {
			m.tab = tabTalkgroups
		} else {
			m.tab = tabSites
		}
		m.cursor = 0
		return m, nil
	}

	switch m.tab {
	case tabSites:
		return m.updateSitesTab(msg, sys)
	case tabTalkgroups:
		return m.updateTalkgroupsTab(msg, sys)
	}
	return m, nil
}

func (m importTUIModel) updateSitesTab(msg tea.KeyMsg, sys *parsedSystem) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(sys.Sites)-1 {
			m.cursor++
		}
	case "pgup":
		m.cursor = clampCursor(m.cursor-pageJump(m.visibleRows()), len(sys.Sites))
	case "pgdown":
		m.cursor = clampCursor(m.cursor+pageJump(m.visibleRows()), len(sys.Sites))
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		if len(sys.Sites) > 0 {
			m.cursor = len(sys.Sites) - 1
		}
	case " ", "space":
		if m.cursor < len(sys.Sites) {
			sys.Sites[m.cursor].Include = !sys.Sites[m.cursor].Include
		}
	}
	return m, nil
}

func (m importTUIModel) updateTalkgroupsTab(msg tea.KeyMsg, sys *parsedSystem) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(sys.Talkgroups)-1 {
			m.cursor++
		}
	case "pgup":
		m.cursor = clampCursor(m.cursor-pageJump(m.visibleRows()), len(sys.Talkgroups))
	case "pgdown":
		m.cursor = clampCursor(m.cursor+pageJump(m.visibleRows()), len(sys.Talkgroups))
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		if len(sys.Talkgroups) > 0 {
			m.cursor = len(sys.Talkgroups) - 1
		}
	case "s":
		if m.cursor < len(sys.Talkgroups) {
			sys.Talkgroups[m.cursor].Scan = !sys.Talkgroups[m.cursor].Scan
		}
	case "L":
		// uppercase L toggles lockout (lowercase l is "right" in vim
		// navigation so we use uppercase to avoid the collision).
		if m.cursor < len(sys.Talkgroups) {
			sys.Talkgroups[m.cursor].Lockout = !sys.Talkgroups[m.cursor].Lockout
		}
	case "e":
		if m.cursor < len(sys.Talkgroups) {
			m.editing = true
			m.editBuf = sys.Talkgroups[m.cursor].AlphaTag
		}
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.cursor < len(sys.Talkgroups) {
			sys.Talkgroups[m.cursor].Priority = int(msg.String()[0] - '0')
		}
	}
	return m, nil
}

func (m importTUIModel) updateEditAlpha(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.systems[m.sysIdx].Talkgroups[m.cursor].AlphaTag = m.editBuf
		m.editing = false
	case tea.KeyEsc:
		m.editing = false
	case tea.KeyBackspace:
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
	case tea.KeyRunes, tea.KeySpace:
		m.editBuf += msg.String()
	}
	return m, nil
}

// View renders the current screen. Style is intentionally minimal —
// matches the project's existing internal/tui aesthetic (block of text
// with lipgloss borders, no fancy widgets).
func (m importTUIModel) View() string {
	var body string
	switch m.view {
	case viewSystems:
		body = m.renderSystemsList()
	case viewSystemTabs:
		body = m.renderSystemTabs()
	}
	footer := m.renderFooter()
	if m.editing {
		body = body + "\n\n" + m.renderEditModal()
	}
	return body + "\n" + footer
}

func (m importTUIModel) renderSystemsList() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Systems to Import"))
	b.WriteString("\n")
	for i, sys := range m.systems {
		cursor := "  "
		if i == m.sysIdx {
			cursor = "▶ "
		}
		ccCount := len(collectControlChannels(sys))
		fmt.Fprintf(&b, "%s%-40s  %d sites  %d CCs  %d talkgroups\n",
			cursor, trunc(sys.Name, 40), len(sys.Sites), ccCount, len(sys.Talkgroups))
	}
	return b.String()
}

func (m importTUIModel) renderSystemTabs() string {
	if m.sysIdx >= len(m.systems) {
		return ""
	}
	sys := &m.systems[m.sysIdx]
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("%s — %s",
		sys.Name, tabLabel(m.tab))))
	b.WriteString("\n")
	page := m.visibleRows()
	switch m.tab {
	case tabSites:
		total := len(sys.Sites)
		start, end := pageBounds(m.cursor, total, page)
		b.WriteString(hintStyle.Render(positionLabel("Site", m.cursor, total, start, end)))
		b.WriteString("\n")
		for i := start; i < end; i++ {
			site := sys.Sites[i]
			cursor := "  "
			if i == m.cursor {
				cursor = "▶ "
			}
			marker := "[ ]"
			if site.Include {
				marker = "[x]"
			}
			ccCount := 0
			for _, f := range site.Frequencies {
				if f.ControlChannel {
					ccCount++
				}
			}
			fmt.Fprintf(&b, "%s%s  RFSS %d Site %d  %-35s %-12s %d freqs  %d CCs\n",
				cursor, marker, site.RFSS, site.SiteID,
				trunc(site.SiteName, 35), site.Cty,
				len(site.Frequencies), ccCount)
		}
	case tabTalkgroups:
		total := len(sys.Talkgroups)
		start, end := pageBounds(m.cursor, total, page)
		b.WriteString(hintStyle.Render(positionLabel("Talkgroup", m.cursor, total, start, end)))
		b.WriteString("\n")
		for i := start; i < end; i++ {
			tg := sys.Talkgroups[i]
			cursor := "  "
			if i == m.cursor {
				cursor = "▶ "
			}
			scan := " "
			if tg.Scan {
				scan = "S"
			}
			lockout := " "
			if tg.Lockout {
				lockout = "L"
			}
			pri := " "
			if tg.Priority > 0 {
				pri = fmt.Sprintf("%d", tg.Priority)
			}
			fmt.Fprintf(&b, "%s[%s%s%s] %-6d %-18s %-30s %s\n",
				cursor, scan, lockout, pri, tg.Dec,
				trunc(tg.AlphaTag, 18), trunc(tg.Description, 30), tg.Tag)
		}
	}
	return b.String()
}

// positionLabel formats the "<Noun> N of M" indicator under the
// header. When the whole list fits on screen we drop the
// "(showing X-Y)" suffix since it's redundant.
func positionLabel(noun string, cursor, total, start, end int) string {
	if total == 0 {
		return fmt.Sprintf("(no %ss)", strings.ToLower(noun))
	}
	if end-start >= total {
		return fmt.Sprintf("%s %d of %d", noun, cursor+1, total)
	}
	return fmt.Sprintf("%s %d of %d  (showing %d-%d)", noun, cursor+1, total, start+1, end)
}

func (m importTUIModel) renderEditModal() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	return box.Render(fmt.Sprintf("Edit Alpha Tag (enter/esc):\n %s_", m.editBuf))
}

func (m importTUIModel) renderFooter() string {
	help := "↑/↓ move  enter open  esc back  w write+exit  q quit"
	switch m.view {
	case viewSystemTabs:
		switch m.tab {
		case tabSites:
			help = "↑/↓ move  pgup/pgdn page  g/G first/last  space toggle  tab switch  esc back  w write  q quit"
		case tabTalkgroups:
			help = "↑/↓ move  pgup/pgdn page  g/G first/last  s scan  L lockout  0-9 priority  e edit  tab switch  esc back  w write  q quit"
		}
	}
	footer := hintStyle.Render(help)
	if m.status != "" {
		footer = m.status + "\n" + footer
	}
	return footer
}

// pageJump returns the cursor delta for one pgup/pgdown — one screen
// minus one row of overlap, so the user keeps a familiar anchor row
// across the jump. Floors at 1 so tiny terminals still advance.
func pageJump(visible int) int {
	if visible <= 1 {
		return 1
	}
	return visible - 1
}

// clampCursor pins the cursor inside [0, total). When total is 0 the
// cursor is forced to 0 — callers should still guard with len() > 0
// before dereferencing.
func clampCursor(c, total int) int {
	if c < 0 {
		return 0
	}
	if total <= 0 {
		return 0
	}
	if c >= total {
		return total - 1
	}
	return c
}

func tabLabel(t tuiTab) string {
	if t == tabSites {
		return "Sites"
	}
	return "Talkgroups"
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "…"
}

func pageBounds(cursor, total, page int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if total <= 0 {
		return 0, 0
	}
	start := cursor - page/2
	if start < 0 {
		start = 0
	}
	end := start + page
	if end > total {
		end = total
		start = end - page
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// visibleRows returns how many list rows fit in the current terminal
// for the Sites / Talkgroups tabs. Reserves a fixed budget for header
// (1 line + position-indicator line), footer (2 lines, including the
// optional status line), and a safety margin. Falls back to 20 when
// the model hasn't yet received a tea.WindowSizeMsg (first paint on
// some terminals).
func (m importTUIModel) visibleRows() int {
	if m.height == 0 {
		return 20
	}
	const reserve = 6
	n := m.height - reserve
	if n < 5 {
		return 5
	}
	return n
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	hintStyle   = lipgloss.NewStyle().Faint(true)
)

// runImportTUI is the entry point used by runImport when --no-tui is
// not passed. Returns wrote=true if the operator successfully wrote
// the config, false otherwise (so the CLI knows whether to print a
// "no changes" message).
func runImportTUI(systems []parsedSystem, writeFn func([]parsedSystem) (mergeResult, error)) (bool, error) {
	model := newImportTUI(systems, writeFn)
	program := tea.NewProgram(model)
	final, err := program.Run()
	if err != nil {
		return false, err
	}
	if im, ok := final.(importTUIModel); ok {
		return im.wrote, nil
	}
	return false, nil
}
