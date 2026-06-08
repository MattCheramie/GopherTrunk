package configtui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

func newTestModel() Model {
	m := New([]string{"/tmp"}, radioreference.Auth{}, nil, "")
	m.width, m.height = 120, 40
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func send(m Model, keys ...string) Model {
	for _, k := range keys {
		tm, _ := m.Update(key(k))
		m = tm.(Model)
	}
	return m
}

// gotoSection moves the section cursor to the section with the given key and
// focuses the form.
func gotoSection(m Model, sectionKey string) Model {
	// reset to top
	for m.sectionIdx > 0 {
		m = send(m, "up")
	}
	for _, want := range sectionKeysOrder() {
		if want == sectionKey {
			break
		}
		m = send(m, "down")
	}
	return send(m, "enter") // focus form
}

func sectionKeysOrder() []string {
	var ks []string
	for _, s := range configbuilder.Sections() {
		ks = append(ks, s.Key)
	}
	return ks
}

func TestModelEditScalar(t *testing.T) {
	m := newTestModel()
	// Logging section: edit Level via select-cycle (space on the row).
	m = gotoSection(m, "log")
	// First row is Level (a select). Activate cycles it off the default.
	before := m.cfg.Log.Level
	m = send(m, "enter") // cycle select once
	if m.cfg.Log.Level == before {
		t.Fatalf("expected Log.Level to change from %q", before)
	}
	if !m.dirty {
		t.Fatalf("expected dirty after edit")
	}
}

func TestModelListAddAndValidate(t *testing.T) {
	m := newTestModel()
	m = gotoSection(m, "trunking")
	// Trunking form: Systems (list) is the first row.
	m = send(m, "enter") // drill into Systems list (empty)
	m = send(m, "a")     // add a system
	if len(m.cfg.Trunking.Systems) != 1 {
		t.Fatalf("expected 1 system, got %d", len(m.cfg.Trunking.Systems))
	}
	// A nameless system is invalid → trunking section has an error.
	if len(m.validation["trunking"]) == 0 {
		t.Fatalf("expected a trunking validation error for the nameless system")
	}
	// Drill into the system and confirm reflection produced rows.
	m = send(m, "enter")
	rows := structRows("SystemConfig", m.cur())
	if len(rows) == 0 {
		t.Fatalf("expected SystemConfig rows")
	}
}

func TestModelQuitGuard(t *testing.T) {
	m := newTestModel()
	m = gotoSection(m, "log")
	m = send(m, "enter") // make a change (dirty)
	// q while in the form zone shouldn't quit; go back to sections then q.
	m = send(m, "esc") // back to sections
	tm, _ := m.Update(key("q"))
	m = tm.(Model)
	if m.modal == nil {
		t.Fatalf("expected an unsaved-changes confirm modal on quit")
	}
}

func TestTalkgroupModal(t *testing.T) {
	m := newTestModel()
	m.cfg.Trunking.Systems = []config.SystemConfig{{Name: "Metro", Protocol: "p25"}}
	m = gotoSection(m, "trunking")
	m = send(m, "enter") // drill into Systems list
	m = send(m, "enter") // drill into system[0] form
	m = send(m, "t")     // open talkgroup editor
	if m.modal == nil {
		t.Fatalf("expected talkgroup modal")
	}
	m = send(m, "a")           // add a row
	m = send(m, "enter")       // edit Decimal cell (seeded "0")
	m = send(m, "1", "0", "1") // → "0101"
	m = send(m, "enter")       // commit cell
	m = send(m, "esc")         // close modal
	rows := m.talkgroups["metro-talkgroups.csv"]
	if len(rows) != 1 || rows[0].Decimal != 101 {
		t.Fatalf("expected 1 talkgroup decimal 101, got %+v", rows)
	}
	if m.cfg.Trunking.Systems[0].TalkgroupFile != "metro-talkgroups.csv" {
		t.Fatalf("expected TalkgroupFile set, got %q", m.cfg.Trunking.Systems[0].TalkgroupFile)
	}
}

func TestImportModal(t *testing.T) {
	parse := func(path, kind string) (config.SystemConfig, []configbuilder.TalkgroupCSVRow, error) {
		return config.SystemConfig{Name: "Imp", Protocol: "p25", ControlChannels: []uint32{851_000_000}},
			[]configbuilder.TalkgroupCSVRow{{Decimal: 7}}, nil
	}
	m := New([]string{"/tmp"}, radioreference.Auth{}, parse, "")
	m.width, m.height = 120, 40
	m = gotoSection(m, "trunking")
	m = send(m, "enter") // Systems list
	m = send(m, "i")     // import modal
	if m.modal == nil {
		t.Fatalf("expected import modal")
	}
	m = send(m, "x", ".", "c", "s", "v") // path "x.csv"
	m = send(m, "enter")                 // parse
	m = send(m, "enter")                 // add parsed system
	if len(m.cfg.Trunking.Systems) != 1 || m.cfg.Trunking.Systems[0].Name != "Imp" {
		t.Fatalf("expected imported system Imp, got %+v", m.cfg.Trunking.Systems)
	}
}

func TestRRModalNoCreds(t *testing.T) {
	m := newTestModel() // empty RR auth
	m = gotoSection(m, "trunking")
	m = send(m, "enter") // Systems list
	m = send(m, "r")     // RR browse
	if m.modal == nil {
		t.Fatalf("expected RR modal (error step) without creds")
	}
	// Esc closes without panic.
	m = send(m, "esc")
	if m.modal != nil {
		t.Fatalf("expected modal closed on esc")
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := New([]string{dir}, radioreference.Auth{}, nil, "")
	m.width, m.height = 120, 40
	m.cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "Metro", Protocol: "p25", ControlChannels: []uint32{851_037_500},
		TalkgroupFile: "metro.csv",
	}}
	m.talkgroups["metro.csv"] = []configbuilder.TalkgroupCSVRow{{Decimal: 101, AlphaTag: "PD"}}

	path := dir + "/config.yaml"
	(&m).doSave(path)
	if len(m.errs) != 0 {
		t.Fatalf("save errors: %v", m.errs)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if _, err := os.Stat(dir + "/metro.csv"); err != nil {
		t.Fatalf("talkgroup sidecar not written: %v", err)
	}

	// Reload into a fresh model and confirm the system + talkgroups survive.
	m2 := New([]string{dir}, radioreference.Auth{}, nil, "")
	(&m2).loadPath(path)
	if len(m2.cfg.Trunking.Systems) != 1 || m2.cfg.Trunking.Systems[0].Name != "Metro" {
		t.Fatalf("system did not round-trip: %+v", m2.cfg.Trunking.Systems)
	}
	rows := m2.talkgroups["metro.csv"]
	if len(rows) != 1 || rows[0].Decimal != 101 || rows[0].AlphaTag != "PD" {
		t.Fatalf("talkgroups did not round-trip: %+v", rows)
	}
}
