package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runSiglabTest is the entry point for `gophertrunk test`. It feeds a capture
// (real-air or `gen`-synthesized) through the production pipeline described by
// its metadata sidecar, grades the decode against the metadata's acceptance
// criteria, and prints a pass/fail report. Exit code is 0 on pass, 1 on
// fail — so it slots straight into CI as a regression gate over a corpus of
// captures.
func runSiglabTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	capture := fs.String("capture", "", "capture file to validate (required)")
	metaPath := fs.String("meta", "", "metadata sidecar (default: auto-discovered by capture stem)")
	outFormat := fs.String("out-format", "text", "report format: text | json | yaml")
	out := fs.String("out", "", "write the report to this file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk test — decode a capture and grade it against acceptance criteria.

USAGE:
  gophertrunk test -capture <path> [-meta <path>] [-out-format text|json|yaml]

EXAMPLES:
  # Validate a gen-produced capture (metadata auto-discovered)
  gophertrunk test -capture p25.cfile

  # Validate a real-air capture with an explicit metadata file, JSON report
  gophertrunk test -capture samples/dmr/site.cfile -meta samples/dmr/site.metadata.yaml -out-format json

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("test")

	if *capture == "" {
		fs.Usage()
		rep.Fatalf(2, "-capture is required")
	}
	mp := *metaPath
	if mp == "" {
		mp = siglab.DiscoverMetadata(*capture)
		if mp == "" {
			rep.Fatalf(2, "no metadata found for %s (pass -meta or add a <stem>.metadata.json sidecar)", *capture)
		}
	}
	meta, err := siglab.LoadMetadata(mp)
	if err != nil {
		rep.Fatal(2, err)
	}
	cfg, err := meta.Config(true)
	if err != nil {
		rep.Fatal(2, err)
	}

	res, err := siglab.Run(*capture, cfg)
	if err != nil {
		rep.Fatal(1, err)
	}

	of := siglab.FormatTextSummary
	switch *outFormat {
	case "json":
		of = siglab.FormatJSON
	case "yaml", "yml":
		of = siglab.FormatYAML
	case "text", "":
		of = siglab.FormatTextSummary
	default:
		rep.Fatalf(2, "unknown -out-format %q (want text|json|yaml)", *outFormat)
	}
	if err := emitResult(res, of, *out); err != nil {
		rep.Fatal(1, err)
	}

	// Exit non-zero on a failing verdict so CI can gate on it.
	if res.Verdict != nil && !res.Verdict.Pass {
		os.Exit(1)
	}
}
