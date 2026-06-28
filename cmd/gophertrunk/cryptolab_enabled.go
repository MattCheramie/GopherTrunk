//go:build cryptolab

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/webserver"
	cryptolabweb "github.com/MattCheramie/GopherTrunk/web/cryptolab"
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
		case "serve":
			runCryptolabServe(args[1:])
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

// runCryptolabServe launches the standalone cryptolab web console — the
// browser counterpart to the CLI. It serves the embedded SPA and the
// cryptolab REST API entirely offline against uploaded files. Mirrors the
// `siglab serve` / `config serve` pattern.
func runCryptolabServe(args []string) {
	fs := flag.NewFlagSet("cryptolab serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8096", "listen address")
	open := fs.Bool("open", false, "open the console in the system browser once it is up")
	tmpDir := fs.String("tmp-dir", "", "directory for staged uploads (default: a fresh temp dir)")
	maxUpload := fs.Int64("max-upload", 0, "max upload size in bytes (0 = default 512 MiB)")
	logLevel := fs.String("log-level", "info", "log level: debug|info|warn|error")
	logFormat := fs.String("log-format", "text", "log format: text|json")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk cryptolab serve — standalone cryptolab web console.

USAGE:
  gophertrunk cryptolab serve [-addr host:port] [-open] [-tmp-dir dir] [-max-upload bytes]

Serves the Crypto Lab SPA at http://<addr>/ and the cryptolab REST API,
running offline against uploaded files. Build the SPA first with
`+"`make cryptolab-web-build`"+`.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	rep := newReporter("cryptolab serve")

	opts := webserver.Options{
		TempDir:        *tmpDir,
		MaxUploadBytes: *maxUpload,
		Logger:         gtlog.New(*logLevel, *logFormat),
	}
	if cryptolabweb.HasAssets() {
		opts.Assets = cryptolabweb.Assets()
	} else {
		fmt.Fprintln(os.Stderr, "cryptolab serve: SPA not bundled (run `make cryptolab-web-build` first); serving API only")
	}

	srv, err := webserver.New(opts)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("cryptolab serve: %w", err))
	}
	defer srv.Close()

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	if *open {
		go func() {
			time.Sleep(300 * time.Millisecond)
			if err := openBrowser("http://" + *addr + "/"); err != nil {
				fmt.Fprintf(os.Stderr, "cryptolab serve: open browser: %v\n", err)
			}
		}()
	}

	fmt.Fprintf(os.Stderr, "cryptolab serve: listening on http://%s/\n", *addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		rep.Fatal(1, fmt.Errorf("cryptolab serve: %w", err))
	}
}

func cryptolabUsage(w *os.File) {
	fmt.Fprintf(w, `gophertrunk cryptolab — optional RF cryptographic-research toolkit

USAGE:
  gophertrunk cryptolab [global flags] <tool> [<mode>] [tool flags]
  gophertrunk cryptolab serve [flags] launch the web console in a browser
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
