package configtui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/config"
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

type openModal struct {
	files []fileEntry
	idx   int
}

func newOpenModal(m *Model) modal {
	var files []fileEntry
	for _, d := range m.dirs {
		for _, p := range config.DirConfigFiles(d) {
			valid := true
			if _, err := config.Load(p); err != nil {
				valid = false
			}
			files = append(files, fileEntry{path: p, valid: valid})
		}
	}
	return &openModal{files: files}
}

func (o *openModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
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
		}
		return nil, nil
	}
	return o, nil
}

func (o *openModal) View(w, h int) string {
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
	b.WriteString("\n[↑↓] move   [enter] open   [esc] cancel")
	return boxTitle("Open config", b.String())
}

// ---- save-as ----

type saveModal struct {
	input textinput.Model
}

func newSaveModal(m *Model) modal {
	ti := textinput.New()
	ti.Prompt = "path: "
	def := "config.yaml"
	if len(m.dirs) > 0 {
		def = joinPath(m.dirs[0], "config.yaml")
	}
	ti.SetValue(def)
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
			m.doSave(p)
		}
		return nil, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *saveModal) View(w, h int) string {
	return boxTitle("Save config as", s.input.View()+"\n\n[enter] save   [esc] cancel")
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
