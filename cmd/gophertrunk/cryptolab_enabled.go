//go:build cryptolab

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	// Blank imports register the toolkit's tools and subjects via init().
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/subjects/motorola"
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

// runCryptolab is the real cryptolab dispatch, linked only with
// -tags cryptolab. Usage:
//
//	gophertrunk cryptolab [global flags] <tool> [<mode>] [tool flags]
//
// Global flags precede the tool name (flag parsing stops at the first
// non-flag argument); the tool/mode then parse their own flags.
func runCryptolab(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			cryptolabUsage(os.Stdout)
			return
		case "list":
			cryptolabList(os.Stdout)
			return
		}
	}

	gfs := flag.NewFlagSet("cryptolab", flag.ContinueOnError)
	out := gfs.String("out", "", "artifact directory for survivor logs / checkpoints")
	resume := gfs.String("resume", "", "checkpoint file to resume from (resumable modes)")
	format := gfs.String("format", "text", "output format: text|json|jsonl|yaml|csv")
	logLevel := gfs.String("log-level", "info", "log level: debug|info|warn|error")
	logFormat := gfs.String("log-format", "text", "log format: text|json")
	gfs.Usage = func() { cryptolabUsage(os.Stderr) }
	if err := gfs.Parse(args); err != nil {
		os.Exit(2)
	}
	rest := gfs.Args()
	if len(rest) == 0 {
		cryptolabUsage(os.Stderr)
		os.Exit(2)
	}

	rep := newReporter("cryptolab")
	fmtKind, err := cryptolab.ParseFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}

	toolName := rest[0]
	tool, ok := cryptolab.Lookup(toolName)
	if !ok {
		rep.Fatalf(2, "unknown tool %q (try `gophertrunk cryptolab list`)", toolName)
	}

	// Resolve the mode and the args that belong to it.
	modeArgs := rest[1:]
	modeName := ""
	if len(tool.Modes()) > 1 {
		if len(modeArgs) == 0 || strings.HasPrefix(modeArgs[0], "-") {
			rep.Fatalf(2, "tool %q needs a mode (try `gophertrunk cryptolab list`)", toolName)
		}
		modeName = modeArgs[0]
		modeArgs = modeArgs[1:]
	} else if len(modeArgs) > 0 && !strings.HasPrefix(modeArgs[0], "-") && modeArgs[0] == tool.Modes()[0].Name() {
		modeName = modeArgs[0]
		modeArgs = modeArgs[1:]
	}
	_, mode, err := cryptolab.LookupMode(toolName, modeName)
	if err != nil {
		rep.Fatal(2, err)
	}

	env := cryptolab.Env{
		Logger:     gtlog.New(*logLevel, *logFormat),
		OutDir:     *out,
		ResumePath: *resume,
		Format:     fmtKind,
		Stdout:     os.Stdout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	res, err := mode.Run(ctx, modeArgs, env)
	if err != nil {
		rep.Fatal(1, err)
	}
	if err := cryptolab.WriteResult(os.Stdout, res, fmtKind); err != nil {
		rep.Fatal(1, err)
	}
}

func cryptolabUsage(w *os.File) {
	fmt.Fprintf(w, `gophertrunk cryptolab — optional RF cryptographic-research toolkit

USAGE:
  gophertrunk cryptolab [global flags] <tool> [<mode>] [tool flags]
  gophertrunk cryptolab list          list tools and modes
  gophertrunk cryptolab help          this help

GLOBAL FLAGS (must precede the tool name):
  -out <dir>        write survivor logs / checkpoints here
  -resume <file>    resume a resumable mode from a checkpoint
  -format <fmt>     text|json|jsonl|yaml|csv (default text)
  -log-level <lvl>  debug|info|warn|error (default info)
  -log-format <f>   text|json (default text)

EXAMPLES:
  gophertrunk cryptolab -out ./out alias gauge -csv alias_ground_truth.csv
  gophertrunk cryptolab alias structure -csv alias_ground_truth.csv
  gophertrunk cryptolab -resume ./out/cells/checkpoint.json alias cells -csv more.csv
  gophertrunk cryptolab brute xor -in cipher.bin -crib "UNIT "
  gophertrunk cryptolab stats scan -in payload.bin
  gophertrunk cryptolab lfsr bm -in keystream.bin
  gophertrunk cryptolab crc recover -in frames.txt -widths 16,8
  gophertrunk cryptolab descramble invert -in scrambled.s16 -out clear.s16
`)
}

func cryptolabList(w *os.File) {
	fmt.Fprintln(w, "cryptolab tools:")
	for _, t := range cryptolab.Tools() {
		fmt.Fprintf(w, "  %-12s %s\n", t.Name(), t.Synopsis())
		for _, m := range t.Modes() {
			fmt.Fprintf(w, "      %-10s %s\n", m.Name(), m.Synopsis())
		}
	}
}
