package configtui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// importModal imports a RadioReference PDF / CSV file into the draft as a
// trunking system (via the injected parser).
type importModal struct {
	input  textinput.Model
	parsed *parsedImport
	err    string
}

type parsedImport struct {
	sys  config.SystemConfig
	rows []configbuilder.TalkgroupCSVRow
}

func newImportModal(m *Model) modal {
	ti := textinput.New()
	ti.Prompt = "file: "
	ti.Placeholder = "/path/to/export.pdf or .csv"
	ti.Focus()
	return &importModal{input: ti}
}

func (im *importModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil
	case "enter":
		if im.parsed != nil {
			m.addSystem(im.parsed.sys, im.parsed.rows)
			return nil, nil
		}
		path := strings.TrimSpace(im.input.Value())
		if path == "" {
			return im, nil
		}
		if m.parse == nil {
			im.err = "import parser not available"
			return im, nil
		}
		kind := ""
		switch strings.ToLower(filepath.Ext(path)) {
		case ".pdf":
			kind = "pdf"
		case ".csv":
			kind = "csv"
		default:
			im.err = "file must be .pdf or .csv"
			return im, nil
		}
		sys, rows, err := m.parse(path, kind)
		if err != nil {
			im.err = err.Error()
			return im, nil
		}
		im.err = ""
		im.parsed = &parsedImport{sys: sys, rows: rows}
		return im, nil
	}
	var cmd tea.Cmd
	im.input, cmd = im.input.Update(msg)
	return im, cmd
}

func (im *importModal) View(w, h int) string {
	if im.parsed != nil {
		s := im.parsed.sys
		body := fmt.Sprintf("Parsed system:\n  Name:     %s\n  Protocol: %s\n  Control:  %d channel(s)\n  Talkgroups: %d\n\n[enter] add to draft   [esc] cancel",
			s.Name, s.Protocol, len(s.ControlChannels), len(im.parsed.rows))
		return boxTitle("Import PDF / CSV", body)
	}
	body := im.input.View() + "\n\n[enter] parse   [esc] cancel"
	if im.err != "" {
		body += "\n" + stErr.Render("! "+im.err)
	}
	return boxTitle("Import PDF / CSV", body)
}
