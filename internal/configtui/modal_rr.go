package configtui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

type rrStep int

const (
	rrMode rrStep = iota
	rrStates
	rrCounties
	rrZip
	rrSystems
	rrError
)

// rrModal browses RadioReference.com (by state→county or by ZIP) and imports a
// system into the draft. Network calls are synchronous (the UI briefly blocks).
type rrModal struct {
	client *radioreference.Client
	step   rrStep
	err    string

	modeIdx int
	zip     textinput.Model

	geos []radioreference.GeoRef // states or counties
	hits []radioreference.SearchHit
	idx  int

	stid int
}

func newRRModal(m *Model) modal {
	c, err := radioreference.NewClient(m.rrAuth)
	if err != nil {
		return &rrModal{step: rrError, err: "RadioReference credentials not configured (set GOPHERTRUNK_RR_KEY / _USER / _PASS)"}
	}
	zi := textinput.New()
	zi.Prompt = "ZIP: "
	return &rrModal{client: c, step: rrMode, zip: zi}
}

func (r *rrModal) ctx() context.Context { return context.Background() }

func (r *rrModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	if msg.String() == "esc" {
		// Step back where it makes sense, else close.
		switch r.step {
		case rrCounties:
			r.step, r.idx = rrStates, 0
			return r, nil
		case rrSystems:
			r.step, r.idx = rrMode, 0
			return r, nil
		default:
			return nil, nil
		}
	}
	switch r.step {
	case rrError:
		return nil, nil
	case rrMode:
		return r.updateMode(msg, m)
	case rrZip:
		return r.updateZip(msg, m)
	case rrStates, rrCounties, rrSystems:
		return r.updateList(msg, m)
	}
	return r, nil
}

func (r *rrModal) updateMode(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if r.modeIdx > 0 {
			r.modeIdx--
		}
	case "down", "j":
		if r.modeIdx < 1 {
			r.modeIdx++
		}
	case "enter":
		if r.modeIdx == 0 { // by state/county
			geos, err := r.client.GetStateList(r.ctx())
			if err != nil {
				r.err = err.Error()
				return r, nil
			}
			r.geos, r.idx, r.step = geos, 0, rrStates
		} else { // by ZIP
			r.zip.Focus()
			r.step = rrZip
		}
	}
	return r, nil
}

func (r *rrModal) updateZip(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	if msg.String() == "enter" {
		hits, err := r.client.SearchByZip(r.ctx(), strings.TrimSpace(r.zip.Value()))
		if err != nil {
			r.err = err.Error()
			return r, nil
		}
		r.hits, r.idx, r.step = hits, 0, rrSystems
		return r, nil
	}
	var cmd tea.Cmd
	r.zip, cmd = r.zip.Update(msg)
	return r, cmd
}

func (r *rrModal) updateList(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	n := r.listLen()
	switch msg.String() {
	case "up", "k":
		if r.idx > 0 {
			r.idx--
		}
	case "down", "j":
		if r.idx < n-1 {
			r.idx++
		}
	case "enter":
		if n == 0 {
			return r, nil
		}
		switch r.step {
		case rrStates:
			r.stid = r.geos[r.idx].ID
			geos, err := r.client.GetCountyList(r.ctx(), r.stid)
			if err != nil {
				r.err = err.Error()
				return r, nil
			}
			r.geos, r.idx, r.step = geos, 0, rrCounties
		case rrCounties:
			hits, err := r.client.SearchByCounty(r.ctx(), r.geos[r.idx].ID)
			if err != nil {
				r.err = err.Error()
				return r, nil
			}
			r.hits, r.idx, r.step = hits, 0, rrSystems
		case rrSystems:
			full, err := r.client.GetFullSystem(r.ctx(), r.hits[r.idx].SID)
			if err != nil {
				r.err = err.Error()
				return r, nil
			}
			sys, rows := configbuilder.RRToSystemConfig(full)
			m.addSystem(sys, rows)
			return nil, nil
		}
	}
	return r, nil
}

func (r *rrModal) listLen() int {
	if r.step == rrSystems {
		return len(r.hits)
	}
	return len(r.geos)
}

func (r *rrModal) View(w, h int) string {
	switch r.step {
	case rrError:
		return boxTitle("RadioReference", stErr.Render(r.err)+"\n\n[esc] close")
	case rrMode:
		opts := []string{"By state / county", "By ZIP code"}
		var b strings.Builder
		for i, o := range opts {
			cur := "  "
			if i == r.modeIdx {
				cur = "▸ "
			}
			b.WriteString(cur + o + "\n")
		}
		b.WriteString(r.errLine() + "\n[↑↓] choose  [enter] select  [esc] cancel")
		return boxTitle("Browse RadioReference", b.String())
	case rrZip:
		return boxTitle("Browse RadioReference — ZIP", r.zip.View()+r.errLine()+"\n\n[enter] search  [esc] cancel")
	default:
		return boxTitle("Browse RadioReference — "+r.stepLabel(), r.listView()+r.errLine())
	}
}

func (r *rrModal) stepLabel() string {
	switch r.step {
	case rrStates:
		return "state"
	case rrCounties:
		return "county"
	case rrSystems:
		return "system"
	}
	return ""
}

func (r *rrModal) listView() string {
	var b strings.Builder
	n := r.listLen()
	if n == 0 {
		b.WriteString(stMuted.Render("(none)") + "\n")
	}
	// Window the list to ~15 rows around the cursor.
	start := 0
	if r.idx > 12 {
		start = r.idx - 12
	}
	for i := start; i < n && i < start+15; i++ {
		cur := "  "
		if i == r.idx {
			cur = "▸ "
		}
		if r.step == rrSystems {
			b.WriteString(fmt.Sprintf("%s%s  %s\n", cur, r.hits[i].Name, stMuted.Render(r.hits[i].Type)))
		} else {
			b.WriteString(cur + r.geos[i].Name + "\n")
		}
	}
	b.WriteString("\n[↑↓] move  [enter] " + r.enterLabel() + "  [esc] back")
	return b.String()
}

func (r *rrModal) enterLabel() string {
	if r.step == rrSystems {
		return "import"
	}
	return "open"
}

func (r *rrModal) errLine() string {
	if r.err == "" {
		return ""
	}
	return "\n" + stErr.Render("! "+r.err)
}
