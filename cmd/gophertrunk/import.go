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

Usage:
  gophertrunk import-pdf -pdf <file.pdf> [-pdf <file.pdf>...] [-csv <file.csv>...] [flags]

Flags:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if len(pdfPaths) == 0 && len(csvPaths) == 0 {
		fs.Usage()
		fail("at least one -pdf or -csv argument is required")
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
