package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// runImport is the entry point for `gophertrunk import-pdf`. Parses
// one or more RadioReference PDFs and merges them into the user's
// config.yaml + per-system talkgroup CSVs.
//
//   -config string     path to existing config.yaml (default "./config.yaml")
//   -pdf string        path to a RadioReference PDF (repeatable)
//   -csv-dir string    where to write talkgroup CSVs (default: configDir)
//   -no-tui            skip TUI; merge straight from parsed defaults
//   -dry-run           print diff, write nothing
//   -force             overwrite an existing system block with the same name
func runImport(args []string) {
	fs := flag.NewFlagSet("import-pdf", flag.ExitOnError)
	cfgPath := fs.String("config", "./config.yaml", "path to existing config.yaml (merged in place)")
	csvDir := fs.String("csv-dir", "", "directory to write talkgroup CSVs (default: directory of -config)")
	noTUI := fs.Bool("no-tui", false, "skip the review TUI and write straight from parsed defaults")
	dryRun := fs.Bool("dry-run", false, "print the planned changes and exit without writing")
	force := fs.Bool("force", false, "overwrite an existing trunking.systems entry with the same name")
	wizard := fs.Bool("wizard", false, "launch the interactive config-builder wizard before any import")
	var pdfPaths repeatedString
	var csvPaths repeatedString
	fs.Var(&pdfPaths, "pdf", "path to a RadioReference PDF system export (repeatable)")
	fs.Var(&csvPaths, "csv", "path to a structured CSV bundle (repeatable; see docs/import.md)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `gophertrunk import-pdf — import system definitions into config.yaml

Sources:
  -pdf <file.pdf>   RadioReference.com PDF export (auto-extracts metadata,
                    sites, and talkgroups).
  -csv <file.csv>   Multi-section structured CSV bundle. One file per system,
                    with metadata / sites / talkgroups sections. See
                    docs/import.md for the format spec and an annotated example.

Both flags are repeatable and may be combined in a single invocation. The
parsed systems are merged into config.yaml (preserving comments) and a
per-system Trunk-Recorder-style talkgroup CSV is written next to the config
(or to -csv-dir).

By default the importer launches an interactive TUI so you can prune sites,
toggle Scan/Lockout flags, and set priorities before write. Pass -no-tui to
merge straight from parsed defaults.

Config-file builder:
  -wizard           Launch the interactive config-builder wizard. Walks you
                    through every section the daemon's loader expects (log,
                    API, auth, CORS, storage, recordings, retention, SDR
                    devices, scanner, audio) and writes a fully-annotated
                    config.yaml. Can be combined with -pdf / -csv: the
                    wizard runs first, then the existing site-review TUI
                    merges trunking systems on top. Can also be run without
                    any imports to produce a daemon-startable scaffold.

Usage:
  gophertrunk import-pdf -pdf <file.pdf> [-pdf <file.pdf>...] [-csv <file.csv>...] [flags]
  gophertrunk import-pdf -wizard                              (build a fresh config)
  gophertrunk import-pdf -wizard -pdf <file.pdf>              (wizard then import)

Flags:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if !*wizard && len(pdfPaths) == 0 && len(csvPaths) == 0 {
		fs.Usage()
		fail("at least one of -wizard, -pdf, or -csv is required")
	}

	// Wizard mode: run the interactive config builder first. The
	// resulting answers feed both the standalone "build a fresh
	// config" path and the "wizard then merge" path below.
	if *wizard {
		seed := defaultWizardAnswers()
		seed.ConfigPath = *cfgPath
		keep := len(pdfPaths) > 0 || len(csvPaths) > 0
		answers, wrote, err := runConfigWizard(seed, keep)
		if err != nil {
			fail("wizard: " + err.Error())
		}
		if !keep {
			if wrote {
				fmt.Fprintf(os.Stderr, "import-pdf: wrote %s\n", answers.ConfigPath)
			} else {
				fmt.Fprintln(os.Stderr, "import-pdf: wizard cancelled, no file written")
			}
			return
		}
		// Wizard + import: write the scaffold first so the merge path
		// has something to layer trunking.systems on top of.
		body, err := renderConfigYAML(answers)
		if err != nil {
			fail("wizard render: " + err.Error())
		}
		if err := os.WriteFile(answers.ConfigPath, body, 0o644); err != nil {
			fail("wizard write: " + err.Error())
		}
		fmt.Fprintf(os.Stderr, "import-pdf: wizard scaffold written to %s\n", answers.ConfigPath)
		*cfgPath = answers.ConfigPath
	}

	if len(pdfPaths) == 0 && len(csvPaths) == 0 {
		// Wizard-only path already handled above.
		return
	}

	// Parse every source up front. If any one fails we abort before
	// touching the user's config.
	parsed := make([]parsedSystem, 0, len(pdfPaths)+len(csvPaths))
	for _, p := range pdfPaths {
		sys, err := parsePDFFile(p)
		if err != nil {
			fail(err.Error())
		}
		parsed = append(parsed, sys)
		fmt.Fprintf(os.Stderr, "import-pdf: parsed PDF %s: %s (%d sites, %d talkgroups)\n",
			p, sys.Name, len(sys.Sites), len(sys.Talkgroups))
	}
	for _, p := range csvPaths {
		sys, err := parseCSVFile(p)
		if err != nil {
			fail(err.Error())
		}
		parsed = append(parsed, sys)
		fmt.Fprintf(os.Stderr, "import-pdf: parsed CSV %s: %s (%d sites, %d talkgroups)\n",
			p, sys.Name, len(sys.Sites), len(sys.Talkgroups))
	}

	opts := mergeOptions{
		ConfigPath: *cfgPath,
		CSVDir:     *csvDir,
		Force:      *force,
		DryRun:     *dryRun,
	}

	writeFn := func(systems []parsedSystem) (mergeResult, error) {
		return mergeIntoConfig(systems, opts)
	}

	if *noTUI || *dryRun {
		res, err := writeFn(parsed)
		if err != nil {
			fail(err.Error())
		}
		if *dryRun {
			renderDryRun(os.Stdout, res, *cfgPath)
			return
		}
		fmt.Fprintf(os.Stderr, "import-pdf: wrote %s\n", *cfgPath)
		for _, c := range res.CSVs {
			fmt.Fprintf(os.Stderr, "import-pdf: wrote %s\n", c.Path)
		}
		return
	}

	wrote, err := runImportTUI(parsed, writeFn)
	if err != nil {
		fail(err.Error())
	}
	if !wrote {
		fmt.Fprintln(os.Stderr, "import-pdf: no changes written")
	}
}

// repeatedString is a flag.Value that accumulates -pdf values into a
// slice (so the operator can pass multiple PDFs in one invocation).
type repeatedString []string

func (r *repeatedString) String() string { return strings.Join(*r, ",") }
func (r *repeatedString) Set(v string) error {
	*r = append(*r, v)
	return nil
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "import-pdf: "+msg)
	os.Exit(1)
}
