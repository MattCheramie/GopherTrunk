package configtui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

func newTestModel() Model {
	m := New([]string{"/tmp"}, true, radioreference.Auth{}, nil, "")
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
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
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
	m := New([]string{"/tmp"}, true, radioreference.Auth{}, parse, "")
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

// TestDefaultSavePath_PrefersDiscoveredOverCandidateDir is the #1120
// regression. Under auto-discovery (no operator -config-dir) the Save-As
// modal must default to the file the daemon actually loads — resolved via
// configbuilder.DefaultConfigPath — not to dirs[0], which on Windows is
// %APPDATA%\GopherTrunk, a location the daemon does not read when the
// installer seeds the config under <Documents>\GopherTrunk\config.
//
// Fails against the old newSaveModal (which unconditionally overrode the
// default with joinPath(m.dirs[0], "config.yaml")).
func TestDefaultSavePath_PrefersDiscoveredOverCandidateDir(t *testing.T) {
	// The file the daemon discovers (highest-precedence discovery input).
	want := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("GOPHERTRUNK_CONFIG", want)

	// Simulate the auto-discovery candidate list with an AppData-style dir
	// first, exactly as config.CandidateDirs() yields on Windows.
	appData := filepath.Join("appdata", "GopherTrunk")
	m := New([]string{appData, "documents"}, false /* dirExplicit */, radioreference.Auth{}, nil, "")

	if got := (&m).defaultSavePath(); got != want {
		t.Errorf("defaultSavePath() = %q, want %q (must follow the daemon's discovery, not dirs[0])", got, want)
	}
	if got := (&m).defaultSavePath(); got == filepath.Join(appData, "config.yaml") {
		t.Errorf("defaultSavePath() defaulted to dirs[0] %q — the #1120 override is back", got)
	}
}

// TestDefaultSavePath_HonorsExplicitDir keeps an operator-supplied
// -config-dir winning over discovery.
func TestDefaultSavePath_HonorsExplicitDir(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", filepath.Join(t.TempDir(), "elsewhere.yaml"))
	dir := t.TempDir()
	m := New([]string{dir}, true /* dirExplicit */, radioreference.Auth{}, nil, "")
	want := filepath.Join(dir, "config.yaml")
	if got := (&m).defaultSavePath(); got != want {
		t.Errorf("defaultSavePath() = %q, want %q (explicit -config-dir must win)", got, want)
	}
}

// TestDefaultSavePath_SavesBackToOpenFile keeps the open file as the
// highest-precedence save default.
func TestDefaultSavePath_SavesBackToOpenFile(t *testing.T) {
	dir := t.TempDir()
	open := filepath.Join(dir, "opened.yaml")
	if _, err := config.WriteConfigFile(open, config.Default(), 0, true); err != nil {
		t.Fatal(err)
	}
	m := New([]string{"appdata"}, false, radioreference.Auth{}, nil, open)
	if got := (&m).defaultSavePath(); got != open {
		t.Errorf("defaultSavePath() = %q, want %q (must save back to the open file)", got, open)
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := New([]string{dir}, true, radioreference.Auth{}, nil, "")
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
	m2 := New([]string{dir}, true, radioreference.Auth{}, nil, "")
	(&m2).loadPath(path)
	if len(m2.cfg.Trunking.Systems) != 1 || m2.cfg.Trunking.Systems[0].Name != "Metro" {
		t.Fatalf("system did not round-trip: %+v", m2.cfg.Trunking.Systems)
	}
	rows := m2.talkgroups["metro.csv"]
	if len(rows) != 1 || rows[0].Decimal != 101 || rows[0].AlphaTag != "PD" {
		t.Fatalf("talkgroups did not round-trip: %+v", rows)
	}
}

func TestOpenModalFileOps(t *testing.T) {
	dir := t.TempDir()
	if _, err := config.WriteConfigFile(dir+"/a.yaml", config.Default(), 0, true); err != nil {
		t.Fatal(err)
	}
	m := New([]string{dir}, true, radioreference.Auth{}, nil, "")
	m.width, m.height = 120, 40

	m = send(m, "o") // open modal lists a.yaml
	if m.modal == nil {
		t.Fatalf("expected open modal")
	}
	// Rename a.yaml -> b.yaml.
	m = send(m, "r")
	for i := 0; i < len("a.yaml"); i++ {
		m = send(m, "backspace")
	}
	m = send(m, "b", ".", "y", "a", "m", "l")
	m = send(m, "enter")
	if _, err := os.Stat(dir + "/b.yaml"); err != nil {
		t.Fatalf("rename target missing: %v", err)
	}
	if _, err := os.Stat(dir + "/a.yaml"); !os.IsNotExist(err) {
		t.Fatalf("old name should be gone")
	}
	// Delete b.yaml.
	m = send(m, "d", "y")
	if _, err := os.Stat(dir + "/b.yaml"); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted")
	}
}

func TestListConfirmRemove(t *testing.T) {
	m := newTestModel()
	m.cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "A", Protocol: "p25"}, {Name: "B", Protocol: "p25"},
	}
	m = gotoSection(m, "trunking")
	m = send(m, "enter") // drill into Systems list
	m = send(m, "d")     // request remove of item 0
	if m.modal == nil {
		t.Fatalf("expected confirm modal before removal")
	}
	if len(m.cfg.Trunking.Systems) != 2 {
		t.Fatalf("nothing should be removed before confirm")
	}
	m = send(m, "y") // confirm
	if len(m.cfg.Trunking.Systems) != 1 || m.cfg.Trunking.Systems[0].Name != "B" {
		t.Fatalf("expected only B to remain, got %+v", m.cfg.Trunking.Systems)
	}
}

func TestRRModalByIDMode(t *testing.T) {
	m := newTestModel() // empty RR auth → error step, but mode nav still safe
	m = gotoSection(m, "trunking")
	m = send(m, "enter") // Systems list
	m = send(m, "r")     // RR modal
	if m.modal == nil {
		t.Fatalf("expected RR modal")
	}
	// Without creds the modal is in the error step; esc closes without panic.
	m = send(m, "esc")
	if m.modal != nil {
		t.Fatalf("expected modal closed")
	}
}
