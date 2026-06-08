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

// readOriginal returns the on-disk bytes of the current file (for comment-
// preserving YAML preview), or nil.
func (m *Model) readOriginal() []byte {
	if m.path == "" {
		return nil
	}
	b, _ := os.ReadFile(m.path)
	return b
}
