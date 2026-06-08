package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/configtui"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// runConfigTUI launches the standalone terminal Config Builder/Editor — the
// console-only counterpart to `gophertrunk config serve`. It edits config
// files on the local filesystem (no daemon, no browser) and is also the
// landing point for bare `gophertrunk config` and the legacy
// `import-pdf -wizard`.
func runConfigTUI(args []string) {
	fs := flag.NewFlagSet("config tui", flag.ExitOnError)
	configDir := fs.String("config-dir", "", "directory to browse and save configs in (default: the standard discovery locations)")
	cfgFile := fs.String("config", "", "config file to open at start")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk config tui — standalone terminal Config Builder/Editor.

USAGE:
  gophertrunk config tui [-config-dir dir] [-config file]
  gophertrunk config                       (same thing)

Build, validate, and save config.yaml from the terminal: edit every section,
import from RadioReference.com / PDF / CSV, manage talkgroups, and preview the
YAML — mirroring the web Config Builder (gophertrunk config serve) for hosts
without a browser. RadioReference uses GOPHERTRUNK_RR_KEY / _USER / _PASS.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if !stdinIsTerminal() || !stdoutIsTerminal() {
		fmt.Fprintln(os.Stderr, "config tui: needs an interactive terminal; for headless use `gophertrunk config serve` or edit config.yaml directly")
		os.Exit(1)
	}

	dirs := []string{}
	if d := strings.TrimSpace(*configDir); d != "" {
		dirs = []string{d}
	} else {
		dirs = config.CandidateDirs()
	}

	rrAuth := radioreference.Auth{
		AppKey:   strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_KEY")),
		Username: strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_USER")),
		Password: strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_PASS")),
	}

	initial := ""
	if f := strings.TrimSpace(*cfgFile); f != "" {
		initial = configbuilder.ExpandConfigPath(f)
	}
	m := configtui.New(dirs, rrAuth, importParseFunc(), initial)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "config tui: %v\n", err)
		os.Exit(1)
	}
}

// importParseFunc adapts the cmd-local PDF/CSV parser to the configtui
// ParseFunc the import modal calls, collapsing each parsed system's control
// channels and mapping its talkgroups.
func importParseFunc() configtui.ParseFunc {
	return func(path, kind string) (config.SystemConfig, []configbuilder.TalkgroupCSVRow, error) {
		var k api.ImportSourceKind
		switch strings.ToLower(kind) {
		case "pdf":
			k = api.ImportSourcePDF
		case "csv":
			k = api.ImportSourceCSV
		default:
			return config.SystemConfig{}, nil, fmt.Errorf("unsupported import kind %q (want pdf or csv)", kind)
		}
		ps, err := parseImportFile(path, k)
		if err != nil {
			return config.SystemConfig{}, nil, err
		}
		var ccs []uint32
		seen := map[uint32]bool{}
		for _, site := range ps.Sites {
			for _, f := range site.Frequencies {
				if f.ControlChannel && !seen[f.Hz] {
					seen[f.Hz] = true
					ccs = append(ccs, f.Hz)
				}
			}
		}
		sys := config.SystemConfig{Name: ps.Name, Protocol: ps.Protocol, ControlChannels: ccs}
		var rows []configbuilder.TalkgroupCSVRow
		for _, tg := range ps.Talkgroups {
			rows = append(rows, configbuilder.TalkgroupCSVRow{
				Decimal:     tg.Dec,
				AlphaTag:    tg.AlphaTag,
				Description: tg.Description,
				Tag:         tg.Tag,
				Group:       tg.Group,
				Mode:        tg.Mode,
			})
		}
		return sys, rows, nil
	}
}
