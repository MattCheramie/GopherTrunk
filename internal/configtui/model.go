package configtui

import (
	"os"
	"reflect"
	"sort"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// ParseFunc parses one PDF/CSV import file (kind "pdf"/"csv") into a parsed
// system, injected by the cmd wrapper which owns the parser. Returns the
// system config + talkgroup rows to fold into the draft.
type ParseFunc func(path, kind string) (config.SystemConfig, []configbuilder.TalkgroupCSVRow, error)

type focusZone int

const (
	zoneSections focusZone = iota
	zoneForm
)

// Model is the standalone terminal Config Builder.
type Model struct {
	cfg        config.Config
	defaults   config.Config
	path       string
	mtime      int64
	dirty      bool
	talkgroups map[string][]configbuilder.TalkgroupCSVRow

	// injected deps
	dirs   []string
	rrAuth radioreference.Auth
	parse  ParseFunc

	width, height int
	focus         focusZone
	sectionIdx    int
	navpath       []pathStep // drill path within the active section
	cursor        int        // row/item cursor in the right pane

	editing bool
	input   textinput.Model
	editRow formRow

	validation map[string][]string
	errs       []string
	status     string

	modal    modal
	quitting bool
}

// New builds the model. dirs is the allowed config directories; rrAuth the
// RadioReference creds; parse the injected import parser; initialPath an
// optional file to open at start.
func New(dirs []string, rrAuth radioreference.Auth, parse ParseFunc, initialPath string) Model {
	ti := textinput.New()
	ti.Prompt = "› "
	m := Model{
		cfg:        config.Default(),
		defaults:   config.Default(),
		talkgroups: map[string][]configbuilder.TalkgroupCSVRow{},
		dirs:       dirs,
		rrAuth:     rrAuth,
		parse:      parse,
		focus:      zoneSections,
		input:      ti,
	}
	m.revalidate()
	if initialPath != "" {
		m.loadPath(initialPath)
	}
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

// ---- core helpers ----

func (m *Model) root() reflect.Value { return reflect.ValueOf(&m.cfg).Elem() }

// sectionField returns the Go config field name of the active section.
func (m *Model) sectionField() string {
	secs := configbuilder.Sections()
	if m.sectionIdx < 0 || m.sectionIdx >= len(secs) {
		return secs[0].CfgField
	}
	return secs[m.sectionIdx].CfgField
}

// cur resolves the value currently shown in the right pane (section root +
// drill path), dereferencing a non-nil pointer to its struct.
func (m *Model) cur() reflect.Value {
	v := resolve(m.root().FieldByName(m.sectionField()), m.navpath)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func (m *Model) markDirty() {
	m.dirty = true
	m.revalidate()
}

// revalidate re-runs ValidateAll and buckets errors by section key.
func (m *Model) revalidate() {
	m.validation = map[string][]string{}
	for _, e := range m.cfg.ValidateAll() {
		msg := e.Error()
		k := configbuilder.SectionOf(msg)
		m.validation[k] = append(m.validation[k], msg)
	}
}

func (m *Model) pushErr(s string) { m.errs = append(m.errs, s) }

// loadPath reads + parses a config file into the draft (without Load's
// validation, so an invalid file can be opened to fix), loads its talkgroup
// sidecars, and revalidates.
func (m *Model) loadPath(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		m.pushErr("open: " + err.Error())
		return
	}
	var cfg config.Config
	if err := config.Unmarshal(data, &cfg); err != nil {
		m.pushErr("parse: " + err.Error())
		return
	}
	m.cfg = cfg
	m.path = path
	if fi, err := os.Stat(path); err == nil {
		m.mtime = fi.ModTime().UnixNano()
	}
	m.dirty = false
	m.talkgroups = m.readTalkgroups(path)
	m.navpath = nil
	m.cursor = 0
	m.status = "Loaded " + path
	m.revalidate()
}

// readTalkgroups loads the talkgroup sidecars referenced by the config's
// systems, relative to the config file's directory.
func (m *Model) readTalkgroups(path string) map[string][]configbuilder.TalkgroupCSVRow {
	out := map[string][]configbuilder.TalkgroupCSVRow{}
	dir := dirOf(path)
	for _, s := range m.cfg.Trunking.Systems {
		rel := s.TalkgroupFile
		if rel == "" {
			continue
		}
		rows, err := configbuilder.ReadTalkgroupCSV(joinPath(dir, rel))
		if err != nil {
			continue
		}
		out[rel] = rows
	}
	return out
}

// configuredSections returns the set of section keys whose subtree differs
// from defaults (for the nav "configured" cue).
func (m *Model) configuredSections() map[string]bool {
	out := map[string]bool{}
	dv := reflect.ValueOf(&m.defaults).Elem()
	cv := m.root()
	for _, s := range configbuilder.Sections() {
		if !reflect.DeepEqual(cv.FieldByName(s.CfgField).Interface(), dv.FieldByName(s.CfgField).Interface()) {
			out[s.Key] = true
		}
	}
	return out
}

func sortedTabKeys() []string {
	keys := make([]string, 0, len(config.KnownUITabs))
	for k := range config.KnownUITabs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
