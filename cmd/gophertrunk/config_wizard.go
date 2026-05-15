package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// configWizardModel is the interactive bubbletea program for building
// a fresh config.yaml from scratch. Walks the operator through every
// section the daemon's loader cares about, then writes the resulting
// file (or hands the populated wizardAnswers back to the caller for
// downstream merging with -pdf / -csv imports).
//
// Each "step" presents a small form on its own screen. Up/Down or
// Tab/Shift-Tab moves between fields within a step; Enter advances to
// the next step (or commits a list builder); Esc backs up one step.
// Ctrl+C / q at any time aborts without writing.
//
// The model is intentionally hand-rolled (no bubbles/textinput) to
// stay consistent with import_tui.go's style and keep the binary's
// dependency surface small.
type configWizardModel struct {
	answers wizardAnswers
	step    int
	field   int
	// Per-step editing scratchpad. Each step writes its current
	// text-field buffer here so arrow keys can move between fields
	// without losing partial edits.
	buf []string
	// Per-step list-builder scratchpad (used by CORS + SDR steps).
	listBuf []string
	// devBuf holds the in-progress SDR device while the operator is
	// filling out its 5 fields. On Enter the device is appended.
	devBuf wizardSDR
	// devField tracks which field of devBuf is being edited.
	devField int
	width    int
	height   int
	status   string
	done     bool
	wrote    bool
}

// step is the static definition of one wizard screen. The dynamic
// part (current field index, current buffer) lives on the model.
type wizardStep struct {
	title  string
	hint   string
	fields []wizardField
}

// field describes one editable widget on a step.
type wizardField struct {
	label    string
	help     string
	kind     fieldKind
	choices  []string // for choice fields
	get      func(a *wizardAnswers) string
	set      func(a *wizardAnswers, v string)
}

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldInt
	fieldFloat
	fieldBool
	fieldChoice
	fieldCORSList    // edits answers.CORSAllowedOrigins via listBuf
	fieldSDRDevices  // edits answers.SDRDevices via devBuf
	fieldConfigPath
	fieldReview      // terminal preview step; Enter writes
)

// runConfigWizard launches the bubbletea program and returns the
// final answers + wrote flag.
//
// When `keep` is true the caller (-wizard combined with -pdf/-csv)
// just wants the wizardAnswers without writing them — the trunking
// section is filled in downstream by the existing merge path. In
// that mode the review step skips the file write and just returns.
func runConfigWizard(initial wizardAnswers, keep bool) (wizardAnswers, bool, error) {
	model := newConfigWizard(initial, keep)
	program := tea.NewProgram(model)
	final, err := program.Run()
	if err != nil {
		return initial, false, err
	}
	if m, ok := final.(configWizardModel); ok {
		return m.answers, m.wrote, nil
	}
	return initial, false, nil
}

func newConfigWizard(initial wizardAnswers, keep bool) configWizardModel {
	m := configWizardModel{answers: initial}
	if keep {
		// The "keep answers, don't write" mode flips the review step
		// description; the write path is short-circuited inside
		// commitReview.
		m.status = "wizard runs first; trunking systems land on top from -pdf / -csv"
	}
	m.loadStepBuffers()
	return m
}

// wizardSteps returns the static step plan. The function form lets
// the field accessors close over wizardAnswers cleanly. The order
// matches the YAML template's section order so the operator's mental
// model maps 1:1 onto the file they see at the end.
func wizardSteps() []wizardStep {
	return []wizardStep{
		// 0. Welcome / config path.
		{
			title: "Welcome",
			hint:  "Build a fresh config.yaml from scratch. Enter advances, Esc backs up, q aborts.",
			fields: []wizardField{
				{
					label: "Where should the wizard write the config?",
					help:  "Use the same path you'll pass to `gophertrunk run -config`. Existing files are overwritten on the review step (you'll see a preview first).",
					kind:  fieldConfigPath,
					get:   func(a *wizardAnswers) string { return a.ConfigPath },
					set:   func(a *wizardAnswers, v string) { a.ConfigPath = v },
				},
			},
		},

		// 1. Logging.
		{
			title: "Logging",
			hint:  "Daemon log verbosity + format. Defaults are fine for most operators.",
			fields: []wizardField{
				{
					label:   "Log level",
					help:    "info is the daemon default. debug emits per-frame DSP / FEC tracing.",
					kind:    fieldChoice,
					choices: []string{"info", "debug", "warn", "error"},
					get:     func(a *wizardAnswers) string { return a.LogLevel },
					set:     func(a *wizardAnswers, v string) { a.LogLevel = v },
				},
				{
					label:   "Log format",
					help:    "text is human-readable; json is for log aggregators (Loki, Splunk, etc.).",
					kind:    fieldChoice,
					choices: []string{"text", "json"},
					get:     func(a *wizardAnswers) string { return a.LogFormat },
					set:     func(a *wizardAnswers, v string) { a.LogFormat = v },
				},
			},
		},

		// 2. API addresses.
		{
			title: "API listeners",
			hint:  "Where the daemon's HTTP + gRPC servers bind.",
			fields: []wizardField{
				{
					label: "HTTP address",
					help:  "127.0.0.1:8080 keeps the API loopback-only. Use 0.0.0.0:8080 to operate from another LAN device (web UI / TUI / curl).",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.HTTPAddr },
					set:   func(a *wizardAnswers, v string) { a.HTTPAddr = v },
				},
				{
					label: "gRPC address",
					help:  "Leave blank to disable the gRPC server entirely (HTTP-only).",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.GRPCAddr },
					set:   func(a *wizardAnswers, v string) { a.GRPCAddr = v },
				},
			},
		},

		// 3. API auth.
		{
			title: "API authentication",
			hint:  "Bearer-token gate on mutation endpoints. auto is fine for single-host operator boxes.",
			fields: []wizardField{
				{
					label:   "Auth mode",
					help:    "auto bypasses the token check on loopback binds. required enforces it everywhere. disabled is wide-open (logged as a warning).",
					kind:    fieldChoice,
					choices: []string{"auto", "required", "disabled"},
					get:     func(a *wizardAnswers) string { return a.AuthMode },
					set:     func(a *wizardAnswers, v string) { a.AuthMode = v },
				},
				{
					label: "Token file (optional)",
					help:  "Path to a file containing the bearer token. Re-read on every request so you can rotate without restarting. Leave blank to skip.",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.AuthTokenFile },
					set:   func(a *wizardAnswers, v string) { a.AuthTokenFile = v },
				},
			},
		},

		// 4. CORS allow-list.
		{
			title: "Web UI CORS",
			hint:  "Allow the standalone web console (gophertrunk-web/) to call the daemon from another origin.",
			fields: []wizardField{
				{
					label: "Allowed origins",
					help:  "Type an origin and press Enter to add. Backspace deletes the last entry. \"null\" allows the SPA loaded via file:// (the most common case). Leave empty to keep CORS off.",
					kind:  fieldCORSList,
				},
			},
		},

		// 5. Metrics.
		{
			title: "Prometheus metrics",
			hint:  "Exposes /metrics on the HTTP API for Prometheus scraping.",
			fields: []wizardField{
				{
					label: "Enable /metrics endpoint?",
					help:  "Recommended on. Cheap to expose; lots of operational visibility.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.MetricsEnabled) },
					set:   func(a *wizardAnswers, v string) { a.MetricsEnabled = wizardParseBool(v) },
				},
			},
		},

		// 6. Storage paths.
		{
			title: "Persistent storage",
			hint:  "Where the daemon writes its SQLite call log and CC-hunter cache.",
			fields: []wizardField{
				{
					label: "Call log database path",
					help:  "SQLite file. Leave blank to disable call-log persistence (events still flow in-memory).",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.StoragePath },
					set:   func(a *wizardAnswers, v string) { a.StoragePath = v },
				},
				{
					label: "CC-hunter cache file",
					help:  "JSON file of the last-locked frequency per system, so the hunter doesn't restart from scratch after a daemon restart.",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.StorageCCCacheFile },
					set:   func(a *wizardAnswers, v string) { a.StorageCCCacheFile = v },
				},
			},
		},

		// 7. Recordings.
		{
			title: "Recordings",
			hint:  "Per-call WAV output. The recorder always runs whether audio playback is on or off.",
			fields: []wizardField{
				{
					label: "Output directory",
					help:  "WAV (and optional .raw) files land here, one subdirectory per system.",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.RecordingsDir },
					set:   func(a *wizardAnswers, v string) { a.RecordingsDir = v },
				},
				{
					label: "Recording sample rate (Hz)",
					help:  "4000–48000. 8000 matches IMBE / AMBE+2 voice-grade audio.",
					kind:  fieldInt,
					get:   func(a *wizardAnswers) string { return strconv.Itoa(a.RecordingsSampleHz) },
					set:   func(a *wizardAnswers, v string) { a.RecordingsSampleHz, _ = strconv.Atoi(v) },
				},
				{
					label: "Write .raw vocoder-frame sidecar?",
					help:  "Useful for offline re-decoding via `gophertrunk decode`. Negligible disk overhead.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.RecordingsWriteRaw) },
					set:   func(a *wizardAnswers, v string) { a.RecordingsWriteRaw = wizardParseBool(v) },
				},
				{
					label: "Enable CMA blind equalizer?",
					help:  "Simulcast / multipath mitigation in the FM voice chain. Off by default — turn on if you see ghosting on simulcast systems.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.EqualizerEnabled) },
					set:   func(a *wizardAnswers, v string) { a.EqualizerEnabled = wizardParseBool(v) },
				},
			},
		},

		// 8. Retention.
		{
			title: "Retention",
			hint:  "Automatic cleanup of old call rows + WAV files.",
			fields: []wizardField{
				{
					label: "Call-log retention (days)",
					help:  "0 disables the call-row sweep. Rows older than this are deleted on the next interval.",
					kind:  fieldInt,
					get:   func(a *wizardAnswers) string { return strconv.Itoa(a.RetentionCallLogDays) },
					set:   func(a *wizardAnswers, v string) { a.RetentionCallLogDays, _ = strconv.Atoi(v) },
				},
				{
					label: "Recording retention (days)",
					help:  "0 disables the WAV/raw sweep. Same as above but for the on-disk files.",
					kind:  fieldInt,
					get:   func(a *wizardAnswers) string { return strconv.Itoa(a.RetentionFilesDays) },
					set:   func(a *wizardAnswers, v string) { a.RetentionFilesDays, _ = strconv.Atoi(v) },
				},
				{
					label: "Sweep interval",
					help:  "Go duration: 1h, 30m, 24h. How often the sweeper runs.",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.RetentionInterval },
					set:   func(a *wizardAnswers, v string) { a.RetentionInterval = v },
				},
			},
		},

		// 9. SDR sample rate + device builder.
		{
			title: "SDR devices",
			hint:  "Pool of RTL-SDR dongles. At least one device is needed to lock on-air; the daemon starts without devices but won't decode anything.",
			fields: []wizardField{
				{
					label: "SDR sample rate (Hz)",
					help:  "225000–3200000. 2_400_000 is the sweet spot for RTL-SDR.",
					kind:  fieldInt,
					get:   func(a *wizardAnswers) string { return strconv.Itoa(a.SDRSampleHz) },
					set:   func(a *wizardAnswers, v string) { a.SDRSampleHz, _ = strconv.Atoi(v) },
				},
				{
					label: "Devices",
					help:  "Type values in the 5 device fields, press Enter to add. Backspace clears the in-progress device. d removes the last added device. Empty list is OK if you don't have hardware yet.",
					kind:  fieldSDRDevices,
				},
			},
		},

		// 10. Scanner.
		{
			title: "Scanner cockpit",
			hint:  "CC hunter + manual-tune behaviour.",
			fields: []wizardField{
				{
					label:   "Default scan mode",
					help:    "all follows every grant. list only follows grants whose talkgroup has Scan=true (Emergency grants bypass).",
					kind:    fieldChoice,
					choices: []string{"all", "list"},
					get:     func(a *wizardAnswers) string { return a.ScannerScanMode },
					set:     func(a *wizardAnswers, v string) { a.ScannerScanMode = v },
				},
				{
					label: "Enable CC hunter?",
					help:  "The multi-system control-channel scanner. Disable only if you're explicitly running one system on a fixed frequency.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.CCHuntEnabled) },
					set:   func(a *wizardAnswers, v string) { a.CCHuntEnabled = wizardParseBool(v) },
				},
				{
					label: "CC hunter dwell (ms)",
					help:  "How long the hunter sits on each candidate before giving up. 3000 is the daemon default.",
					kind:  fieldInt,
					get:   func(a *wizardAnswers) string { return strconv.Itoa(a.CCHuntDwellMs) },
					set:   func(a *wizardAnswers, v string) { a.CCHuntDwellMs, _ = strconv.Atoi(v) },
				},
				{
					label: "Force manual-tune scanner?",
					help:  "Reserves a Voice SDR for the conventional FM scanner even without any pre-configured channels (so the TUI's `f` key + manual-tune API can VFO-tune at runtime). Default false uses auto-detect when ≥ 2 Voice SDRs exist.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.ManualTuneEnabled) },
					set:   func(a *wizardAnswers, v string) { a.ManualTuneEnabled = wizardParseBool(v) },
				},
			},
		},

		// 11. Audio.
		{
			title: "Audio playback",
			hint:  "Live PCM to the host's speakers. Recording is independent.",
			fields: []wizardField{
				{
					label: "Enable live audio playback?",
					help:  "false keeps everything quiet (recorder still runs). true routes decoded PCM to the host's default sink.",
					kind:  fieldBool,
					get:   func(a *wizardAnswers) string { return boolStr(a.AudioEnabled) },
					set:   func(a *wizardAnswers, v string) { a.AudioEnabled = wizardParseBool(v) },
				},
				{
					label: "Audio device",
					help:  "Empty = system default sink. Linux: \"ioctl\" or \"ioctl:hw:C,D\" bypasses libasound2 entirely (useful in distroless / Alpine containers).",
					kind:  fieldText,
					get:   func(a *wizardAnswers) string { return a.AudioDevice },
					set:   func(a *wizardAnswers, v string) { a.AudioDevice = v },
				},
				{
					label: "Volume (0..1)",
					help:  "Software gain applied before the host sink. Match this to the recordings sample rate.",
					kind:  fieldFloat,
					get:   func(a *wizardAnswers) string { return fmt.Sprintf("%.2f", a.AudioVolume) },
					set: func(a *wizardAnswers, v string) {
						if f, err := strconv.ParseFloat(v, 64); err == nil {
							a.AudioVolume = f
						}
					},
				},
			},
		},

		// 12. Review.
		{
			title: "Review + write",
			hint:  "Press Enter (or W) to write the config. Esc to back up and edit. q to abort.",
			fields: []wizardField{
				{
					label: "",
					kind:  fieldReview,
				},
			},
		},
	}
}

// Init satisfies tea.Model.
func (m configWizardModel) Init() tea.Cmd { return nil }

// Update is the event loop.
func (m configWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m configWizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	steps := wizardSteps()
	s := steps[m.step]

	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Special: list-builder and device-builder fields swallow most
	// keys themselves so the operator can type without colliding with
	// the navigation hotkeys above.
	switch s.fields[m.field].kind {
	case fieldCORSList:
		return m.handleCORSList(msg)
	case fieldSDRDevices:
		return m.handleSDRBuilder(msg)
	case fieldReview:
		return m.handleReview(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab", "down":
		if m.field < len(s.fields)-1 {
			m.saveCurrentField()
			m.field++
			m.loadFieldBuffer()
		}
		return m, nil
	case "shift+tab", "up":
		if m.field > 0 {
			m.saveCurrentField()
			m.field--
			m.loadFieldBuffer()
		}
		return m, nil
	case "enter":
		m.saveCurrentField()
		if m.field < len(s.fields)-1 {
			m.field++
			m.loadFieldBuffer()
			return m, nil
		}
		return m.advanceStep()
	case "esc":
		return m.retreatStep()
	case "left", "right":
		switch s.fields[m.field].kind {
		case fieldChoice:
			dir := -1
			if msg.String() == "right" {
				dir = +1
			}
			m.cycleChoice(s.fields[m.field], dir)
		case fieldBool:
			// The footer hint promises ←/→ "changes the value" for
			// non-text fields. Booleans honour that contract too —
			// otherwise an operator who follows the on-screen hint
			// sees the toggle appear locked.
			cur := wizardParseBool(m.buf[m.field])
			m.buf[m.field] = boolStr(!cur)
		}
		return m, nil
	case "y":
		if s.fields[m.field].kind == fieldBool {
			m.buf[m.field] = "true"
		}
		return m, nil
	case "n":
		if s.fields[m.field].kind == fieldBool {
			m.buf[m.field] = "false"
		}
		return m, nil
	case " ":
		if s.fields[m.field].kind == fieldBool {
			cur := wizardParseBool(m.buf[m.field])
			m.buf[m.field] = boolStr(!cur)
		}
		return m, nil
	case "backspace":
		f := s.fields[m.field]
		if f.kind == fieldText || f.kind == fieldInt || f.kind == fieldFloat || f.kind == fieldConfigPath {
			if len(m.buf[m.field]) > 0 {
				m.buf[m.field] = m.buf[m.field][:len(m.buf[m.field])-1]
			}
		}
		return m, nil
	}

	// Printable characters: append to the current text-like field's buffer.
	r := msg.String()
	if len(r) == 1 && r[0] >= 0x20 && r[0] < 0x7f {
		f := s.fields[m.field]
		switch f.kind {
		case fieldText, fieldConfigPath:
			m.buf[m.field] += r
		case fieldInt:
			if r[0] == '-' || (r[0] >= '0' && r[0] <= '9') {
				m.buf[m.field] += r
			}
		case fieldFloat:
			if r[0] == '-' || r[0] == '.' || (r[0] >= '0' && r[0] <= '9') {
				m.buf[m.field] += r
			}
		}
	}
	return m, nil
}

// handleCORSList implements the multi-line list builder for CORS
// allowed_origins. Enter appends the buffer to the list; Backspace
// either trims the buffer or, if the buffer is empty, pops the last
// list entry; n/N moves to the next step.
func (m configWizardModel) handleCORSList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		return m.retreatStep()
	case "enter":
		// Empty buffer + a non-empty list = advance.
		if len(m.listBuf) == 0 && len(m.answers.CORSAllowedOrigins) > 0 {
			return m.advanceStep()
		}
		if scratch := strings.Join(m.listBuf, ""); scratch != "" {
			m.answers.CORSAllowedOrigins = append(m.answers.CORSAllowedOrigins, scratch)
			m.listBuf = m.listBuf[:0]
		} else {
			// Empty buffer + empty list = advance (operator opting out
			// of CORS entirely).
			return m.advanceStep()
		}
		return m, nil
	case "backspace":
		if len(m.listBuf) > 0 {
			m.listBuf = m.listBuf[:len(m.listBuf)-1]
		} else if len(m.answers.CORSAllowedOrigins) > 0 {
			m.answers.CORSAllowedOrigins = m.answers.CORSAllowedOrigins[:len(m.answers.CORSAllowedOrigins)-1]
		}
		return m, nil
	}
	r := msg.String()
	if len(r) == 1 && r[0] >= 0x20 && r[0] < 0x7f {
		m.listBuf = append(m.listBuf, r)
	}
	return m, nil
}

// handleSDRBuilder runs the 5-field device-add form. Tab moves between
// fields within the current in-progress device; Enter commits the
// device and resets the form; "x" removes the last committed device;
// "n" advances to the next step.
func (m configWizardModel) handleSDRBuilder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		return m.retreatStep()
	case "enter":
		// If the in-progress device is empty, advance (operator opting
		// out of pre-configured devices). Otherwise commit and reset.
		if m.devBuf == (wizardSDR{}) || m.devBuf.Serial == "" {
			return m.advanceStep()
		}
		// Backfill blanks with defaults so the daemon accepts the result.
		if m.devBuf.Role == "" {
			m.devBuf.Role = "auto"
		}
		if m.devBuf.Gain == "" {
			m.devBuf.Gain = "auto"
		}
		m.answers.SDRDevices = append(m.answers.SDRDevices, m.devBuf)
		m.devBuf = wizardSDR{}
		m.devField = 0
		return m, nil
	case "tab", "down":
		if m.devField < 4 {
			m.devField++
		}
		return m, nil
	case "shift+tab", "up":
		if m.devField > 0 {
			m.devField--
		}
		return m, nil
	case "left", "right":
		if m.devField == 1 { // role choice
			roles := []string{"control", "voice", "auto"}
			idx := 0
			for i, r := range roles {
				if m.devBuf.Role == r {
					idx = i
					break
				}
			}
			if msg.String() == "left" {
				idx = (idx + len(roles) - 1) % len(roles)
			} else {
				idx = (idx + 1) % len(roles)
			}
			m.devBuf.Role = roles[idx]
		}
		if m.devField == 4 { // bias_tee toggle
			m.devBuf.BiasTee = !m.devBuf.BiasTee
		}
		return m, nil
	case "y":
		if m.devField == 4 {
			m.devBuf.BiasTee = true
		}
		return m, nil
	case "N":
		if m.devField == 4 {
			m.devBuf.BiasTee = false
		}
		return m, nil
	case "backspace":
		switch m.devField {
		case 0:
			if n := len(m.devBuf.Serial); n > 0 {
				m.devBuf.Serial = m.devBuf.Serial[:n-1]
			}
		case 2:
			s := strconv.Itoa(m.devBuf.PPM)
			if len(s) > 0 {
				s = s[:len(s)-1]
			}
			if s == "" || s == "-" {
				m.devBuf.PPM = 0
			} else {
				m.devBuf.PPM, _ = strconv.Atoi(s)
			}
		case 3:
			if n := len(m.devBuf.Gain); n > 0 {
				m.devBuf.Gain = m.devBuf.Gain[:n-1]
			}
		}
		return m, nil
	case "d":
		// Remove the last committed device. Doubles as a "delete last"
		// affordance distinct from the in-progress buffer.
		if n := len(m.answers.SDRDevices); n > 0 {
			m.answers.SDRDevices = m.answers.SDRDevices[:n-1]
		}
		return m, nil
	}
	r := msg.String()
	if len(r) != 1 || r[0] < 0x20 || r[0] >= 0x7f {
		return m, nil
	}
	switch m.devField {
	case 0:
		m.devBuf.Serial += r
	case 2:
		if r[0] == '-' || (r[0] >= '0' && r[0] <= '9') {
			s := strconv.Itoa(m.devBuf.PPM) + r
			if v, err := strconv.Atoi(s); err == nil {
				m.devBuf.PPM = v
			}
		}
	case 3:
		m.devBuf.Gain += r
	}
	return m, nil
}

// handleReview shows the rendered YAML and either writes it or
// retreats to the previous step.
func (m configWizardModel) handleReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		return m.retreatStep()
	case "enter", "w":
		return m.commitReview()
	}
	return m, nil
}

// commitReview writes the rendered config to disk and quits. If the
// caller passed keep=true (wizard + import in one invocation), this
// just marks done=true and lets the caller pick up answers.
func (m configWizardModel) commitReview() (tea.Model, tea.Cmd) {
	out, err := renderConfigYAML(m.answers)
	if err != nil {
		m.status = "render error: " + err.Error()
		return m, nil
	}
	// Ensure parent dir exists.
	if dir := filepath.Dir(m.answers.ConfigPath); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	if err := os.WriteFile(m.answers.ConfigPath, out, 0o644); err != nil {
		m.status = "write error: " + err.Error()
		return m, nil
	}
	m.wrote = true
	m.done = true
	return m, tea.Quit
}

func (m configWizardModel) advanceStep() (tea.Model, tea.Cmd) {
	steps := wizardSteps()
	if m.step >= len(steps)-1 {
		return m, nil
	}
	m.step++
	m.field = 0
	m.loadStepBuffers()
	return m, nil
}

func (m configWizardModel) retreatStep() (tea.Model, tea.Cmd) {
	if m.step == 0 {
		return m, nil
	}
	m.step--
	m.field = 0
	m.loadStepBuffers()
	return m, nil
}

// loadStepBuffers populates m.buf with the current answers' values
// for every text-like field on the current step. Called on step entry.
func (m *configWizardModel) loadStepBuffers() {
	steps := wizardSteps()
	s := steps[m.step]
	m.buf = make([]string, len(s.fields))
	for i, f := range s.fields {
		if f.get != nil {
			m.buf[i] = f.get(&m.answers)
		}
	}
}

// loadFieldBuffer refreshes the current field's buffer from
// answers — used when navigating between fields with arrow keys.
func (m *configWizardModel) loadFieldBuffer() {
	steps := wizardSteps()
	f := steps[m.step].fields[m.field]
	if f.get != nil {
		m.buf[m.field] = f.get(&m.answers)
	}
}

// saveCurrentField persists the in-progress buffer back into the
// answers struct via the field's setter.
func (m *configWizardModel) saveCurrentField() {
	steps := wizardSteps()
	f := steps[m.step].fields[m.field]
	if f.set != nil {
		f.set(&m.answers, m.buf[m.field])
	}
}

func (m *configWizardModel) cycleChoice(f wizardField, dir int) {
	cur := m.buf[m.field]
	idx := 0
	for i, c := range f.choices {
		if c == cur {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(f.choices)) % len(f.choices)
	m.buf[m.field] = f.choices[idx]
}

// View renders the current step.
func (m configWizardModel) View() string {
	steps := wizardSteps()
	s := steps[m.step]
	var b strings.Builder

	header := lipgloss.NewStyle().Bold(true).Underline(true).Render(
		fmt.Sprintf("GopherTrunk config wizard  ·  step %d / %d  ·  %s",
			m.step+1, len(steps), s.title))
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render(s.hint))
	b.WriteString("\n\n")

	// Special-case the review step — render the YAML preview.
	if len(s.fields) > 0 && s.fields[0].kind == fieldReview {
		out, err := renderConfigYAML(m.answers)
		if err != nil {
			b.WriteString("error rendering preview: " + err.Error())
		} else {
			b.WriteString("Destination: ")
			b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.answers.ConfigPath))
			b.WriteString("\n\n")
			b.WriteString(previewLines(string(out), 24))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"[Enter] write to disk   [Esc] back   [q] abort"))
		if m.status != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.status))
		}
		return b.String()
	}

	for i, f := range s.fields {
		marker := "  "
		if i == m.field {
			marker = "▶ "
		}
		switch f.kind {
		case fieldCORSList:
			b.WriteString(marker + f.label + "\n")
			b.WriteString("    " + lipgloss.NewStyle().Faint(true).Render(f.help) + "\n")
			for _, v := range m.answers.CORSAllowedOrigins {
				b.WriteString("    • " + v + "\n")
			}
			b.WriteString("    + ")
			b.WriteString(strings.Join(m.listBuf, ""))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("█"))
			b.WriteString("\n\n")
		case fieldSDRDevices:
			b.WriteString(marker + f.label + "\n")
			b.WriteString("    " + lipgloss.NewStyle().Faint(true).Render(f.help) + "\n")
			for j, d := range m.answers.SDRDevices {
				b.WriteString(fmt.Sprintf("    [%d] serial=%s role=%s ppm=%d gain=%s bias=%t\n",
					j, d.Serial, d.Role, d.PPM, d.Gain, d.BiasTee))
			}
			b.WriteString("    in-progress:\n")
			b.WriteString(devFieldRow("serial  ", m.devBuf.Serial, m.devField == 0))
			b.WriteString(devFieldRow("role    ", m.devBuf.Role, m.devField == 1))
			b.WriteString(devFieldRow("ppm     ", strconv.Itoa(m.devBuf.PPM), m.devField == 2))
			b.WriteString(devFieldRow("gain    ", m.devBuf.Gain, m.devField == 3))
			b.WriteString(devFieldRow("bias_tee", boolStr(m.devBuf.BiasTee), m.devField == 4))
			b.WriteString("\n")
		default:
			b.WriteString(marker + f.label + ": ")
			val := m.buf[i]
			if f.kind == fieldChoice {
				b.WriteString("◄ " + val + " ►")
			} else if f.kind == fieldBool {
				b.WriteString(val)
			} else {
				b.WriteString(val)
				if i == m.field {
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("█"))
				}
			}
			b.WriteString("\n")
			if f.help != "" {
				b.WriteString("    " + lipgloss.NewStyle().Faint(true).Render(f.help) + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"[Tab] next field  [Shift+Tab] prev  [←/→ or y/n/Space] change value  [Enter] next step  [Esc] back  [q] abort"))
	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(m.status))
	}
	return b.String()
}

func devFieldRow(label, val string, active bool) string {
	marker := "      "
	if active {
		marker = "    → "
	}
	return marker + label + ": " + val + "\n"
}

func previewLines(s string, max int) string {
	lines := strings.SplitN(s, "\n", max+1)
	if len(lines) > max {
		return strings.Join(lines[:max], "\n") +
			"\n" + lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(
				"  … (%d total lines; full file written on Enter)",
				strings.Count(s, "\n")+1))
	}
	return s
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func wizardParseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "yes", "y", "1", "on":
		return true
	}
	return false
}
