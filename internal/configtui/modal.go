package configtui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// modal is an overlay that owns all key input while open. Returning a nil
// modal closes it.
type modal interface {
	Update(tea.KeyMsg, *Model) (modal, tea.Cmd)
	View(width, height int) string
}

var modalBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(1, 2)

func boxTitle(title, body string) string {
	return modalBox.Render(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n" + body)
}

// ---- confirm ----

type confirmModal struct {
	msg   string
	onYes func(*Model) tea.Cmd
}

func newConfirmModal(msg string, onYes func(*Model) tea.Cmd) modal {
	return &confirmModal{msg: msg, onYes: onYes}
}

func (c *confirmModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return nil, c.onYes(m)
	case "n", "esc", "q":
		return nil, nil
	}
	return c, nil
}

func (c *confirmModal) View(w, h int) string {
	return boxTitle("Confirm", c.msg+"\n\n[y] yes    [n] no")
}

// ---- open ----

type fileEntry struct {
	path  string
	valid bool
}

type openMode int

const (
	openBrowse openMode = iota
	openRename
	openConfirmDelete
)

type openModal struct {
	dirs  []string
	files []fileEntry
	idx   int
	mode  openMode
	input textinput.Model
	err   string
}

func newOpenModal(m *Model) modal {
	o := &openModal{dirs: m.dirs, input: textinput.New()}
	o.refresh()
	return o
}

func (o *openModal) refresh() {
	o.files = nil
	for _, d := range o.dirs {
		for _, p := range config.DirConfigFiles(d) {
			valid := true
			if _, err := config.Load(p); err != nil {
				valid = false
			}
			o.files = append(o.files, fileEntry{path: p, valid: valid})
		}
	}
	if o.idx >= len(o.files) {
		o.idx = max(0, len(o.files)-1)
	}
}

func (o *openModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch o.mode {
	case openRename:
		switch msg.String() {
		case "esc":
			o.mode = openBrowse
		case "enter":
			from := o.files[o.idx].path
			to := joinPath(dirOf(from), strings.TrimSpace(o.input.Value()))
			if err := os.Rename(from, to); err != nil {
				o.err = err.Error()
			}
			o.mode = openBrowse
			o.refresh()
		default:
			var cmd tea.Cmd
			o.input, cmd = o.input.Update(msg)
			return o, cmd
		}
		return o, nil
	case openConfirmDelete:
		switch msg.String() {
		case "y", "enter":
			if err := os.Remove(o.files[o.idx].path); err != nil {
				o.err = err.Error()
			}
			o.mode = openBrowse
			o.refresh()
		case "n", "esc":
			o.mode = openBrowse
		}
		return o, nil
	}

	// browse
	switch msg.String() {
	case "esc", "q":
		return nil, nil
	case "up", "k":
		if o.idx > 0 {
			o.idx--
		}
	case "down", "j":
		if o.idx < len(o.files)-1 {
			o.idx++
		}
	case "enter":
		if len(o.files) > 0 {
			m.loadPath(o.files[o.idx].path)
			return nil, nil
		}
	case "r":
		if len(o.files) > 0 {
			o.mode = openRename
			o.input.SetValue(filepath.Base(o.files[o.idx].path))
			o.input.CursorEnd()
			o.input.Focus()
		}
	case "d":
		if len(o.files) > 0 {
			o.mode = openConfirmDelete
		}
	}
	return o, nil
}

func (o *openModal) View(w, h int) string {
	if o.mode == openRename {
		return boxTitle("Rename file", "rename: "+o.input.View()+"\n\n[enter] rename   [esc] cancel")
	}
	if o.mode == openConfirmDelete {
		return boxTitle("Delete file", "Delete "+o.files[o.idx].path+" ?\n\n[y] yes   [n] no")
	}
	if len(o.files) == 0 {
		return boxTitle("Open config", "No config files found in the discovery directories.\n\n[esc] cancel")
	}
	var b strings.Builder
	for i, f := range o.files {
		cursor := "  "
		if i == o.idx {
			cursor = "▸ "
		}
		flag := ""
		if !f.valid {
			flag = " ⚠"
		}
		b.WriteString(cursor + f.path + flag + "\n")
	}
	if o.err != "" {
		b.WriteString(stErr.Render("! " + o.err))
		b.WriteString("\n")
	}
	b.WriteString("\n[↑↓] move   [enter] open   [r] rename   [d] delete   [esc] cancel")
	return boxTitle("Open config", b.String())
}

// ---- save-as ----

type saveModal struct {
	input textinput.Model
}

func newSaveModal(m *Model) modal {
	ti := textinput.New()
	ti.Prompt = "path: "
	ti.SetValue(m.defaultSavePath())
	ti.CursorEnd()
	ti.Focus()
	return &saveModal{input: ti}
}

func (s *saveModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil
	case "enter":
		p := strings.TrimSpace(s.input.Value())
		if p != "" {
			m.doSave(configbuilder.ExpandConfigPath(p))
		}
		return nil, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *saveModal) View(w, h int) string {
	return boxTitle("Save config as",
		s.input.View()+"\n\nMissing parent folders are created automatically.\n[enter] save   [esc] cancel")
}

// ---- yaml preview ----

type yamlModal struct {
	vp viewport.Model
}

func newYAMLModal(m *Model) modal {
	var (
		data []byte
		err  error
	)
	if orig := m.readOriginal(); len(orig) > 0 {
		data, err = config.MarshalMerge(orig, m.cfg)
	} else {
		data, err = config.Marshal(m.cfg)
	}
	body := string(data)
	if err != nil {
		body = "error: " + err.Error()
	}
	w := min(m.width-8, 100)
	if w < 20 {
		w = 20
	}
	hh := min(m.height-8, 30)
	if hh < 6 {
		hh = 6
	}
	vp := viewport.New(w, hh)
	vp.SetContent(body)
	return &yamlModal{vp: vp}
}

func (y *yamlModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return nil, nil
	}
	var cmd tea.Cmd
	y.vp, cmd = y.vp.Update(msg)
	return y, cmd
}

func (y *yamlModal) View(w, h int) string {
	return boxTitle("config.yaml preview", y.vp.View()+"\n\n[↑↓/pgup/pgdn] scroll   [esc] close")
}
