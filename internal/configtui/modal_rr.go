package configtui

import (
	"context"
	"fmt"
	"strconv"
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
	rrID
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
	idInput textinput.Model
	idState bool // by-ID: true = state id (stid), false = county id (ctid)

	geos []radioreference.GeoRef // states or counties
	hits []radioreference.SearchHit
	idx  int

	stid int
}

// effectiveRRAuth merges the RadioReference credentials being edited
// (m.cfg.RadioReference) over the startup auth (env), then backstops the app
// key with the built-in default. The edited config wins, so what the operator
// types into the RadioReference section is what browse/import and verify send.
func effectiveRRAuth(m *Model) radioreference.Auth {
	return radioreference.ResolveAuth(radioreference.Auth{
		AppKey:   firstNonEmptyStr(m.cfg.RadioReference.APIKey, m.rrAuth.AppKey),
		Username: firstNonEmptyStr(m.cfg.RadioReference.Username, m.rrAuth.Username),
		Password: firstNonEmptyStr(m.cfg.RadioReference.Password, m.rrAuth.Password),
	})
}

// firstNonEmptyStr returns the first argument that is non-empty after trimming.
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func newRRModal(m *Model) modal {
	c, err := radioreference.NewClient(effectiveRRAuth(m))
	if err != nil {
		return &rrModal{step: rrError, err: "RadioReference app key not configured (no built-in key in this build; set GOPHERTRUNK_RR_KEY or the api_key field)"}
	}
	zi := textinput.New()
	zi.Prompt = "ZIP: "
	ii := textinput.New()
	ii.Prompt = "ID: "
	return &rrModal{client: c, step: rrMode, zip: zi, idInput: ii}
}

func (r *rrModal) ctx() context.Context { return context.Background() }

func (r *rrModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	if msg.String() == "esc" {
		// Step back where it makes sense, else close.
		switch r.step {
		case rrCounties:
			r.step, r.idx = rrStates, 0
			return r, nil
		case rrZip, rrID, rrSystems:
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
	case rrID:
		return r.updateID(msg, m)
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
		if r.modeIdx < 2 {
			r.modeIdx++
		}
	case "enter":
		switch r.modeIdx {
		case 0: // by state/county
			geos, err := r.client.GetStateList(r.ctx())
			if err != nil {
				r.err = err.Error()
				return r, nil
			}
			r.geos, r.idx, r.step = geos, 0, rrStates
		case 1: // by ZIP
			r.zip.Focus()
			r.step = rrZip
		case 2: // by numeric ID
			r.idInput.Focus()
			r.step = rrID
		}
	}
	return r, nil
}

func (r *rrModal) updateID(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) {
	switch msg.String() {
	case "tab":
		r.idState = !r.idState
		return r, nil
	case "enter":
		n, err := strconv.Atoi(strings.TrimSpace(r.idInput.Value()))
		if err != nil {
			r.err = "id must be numeric"
			return r, nil
		}
		var hits []radioreference.SearchHit
		if r.idState {
			hits, err = r.client.SearchByState(r.ctx(), n)
		} else {
			hits, err = r.client.SearchByCounty(r.ctx(), n)
		}
		if err != nil {
			r.err = err.Error()
			return r, nil
		}
		r.hits, r.idx, r.step = hits, 0, rrSystems
		return r, nil
	}
	var cmd tea.Cmd
	r.idInput, cmd = r.idInput.Update(msg)
	return r, cmd
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
		opts := []string{"By state / county", "By ZIP code", "By ID (county/state)"}
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
	case rrID:
		kind := "county id (ctid)"
		if r.idState {
			kind = "state id (stid)"
		}
		return boxTitle("Browse RadioReference — ID",
			r.idInput.View()+"\nlooking up: "+kind+r.errLine()+"\n\n[tab] county/state  [enter] search  [esc] back")
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

// ---- RadioReference verify ----

// infoModal is a dismissable result box (any key closes it). Used to surface
// the synchronous RadioReference subscription check.
type infoModal struct {
	title string
	body  string
	bad   bool
}

func (i *infoModal) Update(msg tea.KeyMsg, m *Model) (modal, tea.Cmd) { return nil, nil }

func (i *infoModal) View(w, h int) string {
	body := stOK.Render(i.body)
	if i.bad {
		body = stErr.Render(i.body)
	}
	return boxTitle("RadioReference", body+"\n\n[any key] close")
}

// newRRVerifyModal verifies the edited RadioReference credentials against the
// account service and returns a dismissable result box reporting whether the
// subscription is premium.
func newRRVerifyModal(m *Model) modal {
	c, err := radioreference.NewClient(effectiveRRAuth(m))
	if err != nil {
		return &infoModal{body: "App key not configured (no built-in key in this build; set GOPHERTRUNK_RR_KEY or the api_key field).", bad: true}
	}
	acct, err := c.VerifyCredentials(context.Background())
	if err != nil {
		return &infoModal{body: "Not verified: " + err.Error(), bad: true}
	}
	switch {
	case acct.Premium && acct.Expires != "":
		return &infoModal{body: "✓ Premium subscription active until " + acct.Expires + " (" + acct.Username + ")."}
	case acct.Premium:
		return &infoModal{body: "✓ Premium subscription active (" + acct.Username + ")."}
	case acct.Expires != "":
		return &infoModal{body: "Logged in as " + acct.Username + ", but the subscription is not active (expired " + acct.Expires + ").", bad: true}
	default:
		return &infoModal{body: "Logged in as " + acct.Username + ", but no active premium subscription was found.", bad: true}
	}
}
