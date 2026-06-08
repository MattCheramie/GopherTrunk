package configtui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	if m.dirty {
		m.modal = newConfirmModal("Discard unsaved changes and quit?", func(mm *Model) tea.Cmd {
			mm.quitting = true
			return tea.Quit
		})
		return m, nil
	}
	m.quitting = true
	return m, tea.Quit
}

func (m Model) requestNew() (tea.Model, tea.Cmd) {
	if m.dirty {
		m.modal = newConfirmModal("Discard unsaved changes and start a new config?", func(mm *Model) tea.Cmd {
			mm.resetNew()
			return nil
		})
		return m, nil
	}
	m.resetNew()
	return m, nil
}

func (m *Model) resetNew() {
	m.cfg = config.Default()
	m.path = ""
	m.mtime = 0
	m.dirty = false
	m.talkgroups = map[string][]configbuilder.TalkgroupCSVRow{}
	m.navpath = nil
	m.cursor = 0
	m.status = "New config"
	m.revalidate()
}

func (m *Model) revert() {
	if m.path != "" {
		m.loadPath(m.path)
		m.status = "Reverted to " + m.path
	} else {
		m.resetNew()
	}
}

func (m Model) startSave() (tea.Model, tea.Cmd) {
	if m.path == "" {
		m.modal = newSaveModal(&m)
		return m, nil
	}
	m.doSave(m.path)
	return m, nil
}

// doSave writes the talkgroup sidecars then the config file (validated +
// comment-preserving on overwrite). Errors are surfaced as toasts.
func (m *Model) doSave(path string) {
	dir := dirOf(path)
	for rel, rows := range m.talkgroups {
		if err := configbuilder.WriteTalkgroupCSV(joinPath(dir, rel), rows); err != nil {
			m.pushErr("talkgroup " + rel + ": " + err.Error())
			return
		}
	}
	guard := m.mtime
	if path != m.path {
		guard = 0
	}
	mt, err := config.WriteConfigFile(path, m.cfg, guard, true)
	if err != nil {
		m.pushErr("save: " + err.Error())
		return
	}
	m.path = path
	m.mtime = mt
	m.dirty = false
	m.status = "Saved " + path
	m.talkgroups = m.readTalkgroups(path)
}

// addSystem appends a trunking system to the draft, staging its talkgroup
// sidecar (deriving a TalkgroupFile from the name when rows are present).
// Used by the RadioReference and PDF/CSV import modals.
func (m *Model) addSystem(sys config.SystemConfig, rows []configbuilder.TalkgroupCSVRow) {
	if len(rows) > 0 {
		rel := slug(sys.Name) + "-talkgroups.csv"
		sys.TalkgroupFile = rel
		m.talkgroups[rel] = rows
	}
	m.cfg.Trunking.Systems = append(m.cfg.Trunking.Systems, sys)
	m.status = "Added system " + sys.Name
	m.dirty = true
	m.revalidate()
}

// slug turns a system name into a filename-safe stem (mirrors the web's slug).
func slug(name string) string {
	if name == "" {
		name = "system"
	}
	var b []rune
	prevDash := false
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b = append(b, r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			b = append(b, r+32)
			prevDash = false
		default:
			if !prevDash {
				b = append(b, '-')
				prevDash = true
			}
		}
	}
	s := string(b)
	for len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		s = "system"
	}
	return s
}

// readOriginal returns the on-disk bytes of the current file (for comment-
// preserving YAML preview), or nil.
func (m *Model) readOriginal() []byte {
	if m.path == "" {
		return nil
	}
	b, _ := os.ReadFile(m.path)
	return b
}
