package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/version"
	siglabweb "github.com/MattCheramie/GopherTrunk/web/siglab"
)

// runSiglabServe is the entry point for `gophertrunk siglab serve` — a
// standalone, offline signal-analysis web console. It runs the api.Server
// with only the siglab subsystem wired (no SDR, daemon, or storage), serving
// the embedded Signal Lab SPA at / and the /api/v1/siglab/* routes. It is the
// browser counterpart to the `siglab` TUI: upload a capture, run the engine,
// visualize the result, synthesize an idealized reference, and compare.
func runSiglabServe(args []string) {
	fs := flag.NewFlagSet("siglab serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8099", "listen address")
	open := fs.Bool("open", false, "open the console in the system browser once it is up")
	tmpDir := fs.String("tmp-dir", "", "directory for staged capture uploads (default: a fresh temp dir)")
	maxUpload := fs.Int64("max-upload", 0, "max capture upload size in bytes (0 = default 512 MiB)")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk siglab serve — standalone offline signal-analysis web console.

USAGE:
  gophertrunk siglab serve [-addr host:port] [-open] [-tmp-dir dir] [-max-upload bytes]

Serves the Signal Lab SPA at http://<addr>/ and the siglab REST API. Runs
entirely offline against uploaded capture files — no SDR or daemon required.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("siglab serve")

	bus := events.NewBus(64)
	defer bus.Close()

	opts := api.ServerOptions{
		Addr:    *addr,
		Bus:     bus,
		Version: version.String(),
		Siglab: api.SiglabOptions{
			Enabled:        true,
			TempDir:        *tmpDir,
			MaxUploadBytes: *maxUpload,
		},
	}
	if siglabweb.HasAssets() {
		opts.WebAssets = siglabweb.Assets()
	} else {
		fmt.Fprintln(os.Stderr, "siglab serve: SPA not bundled (run `make siglab-web-build` first); serving API only")
	}

	srv, err := api.NewServer(opts)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("siglab serve: %w", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *open {
		go func() {
			time.Sleep(300 * time.Millisecond)
			url := "http://" + *addr + "/"
			if err := openBrowser(url); err != nil {
				fmt.Fprintf(os.Stderr, "siglab serve: open browser: %v\n", err)
			}
		}()
	}

	fmt.Fprintf(os.Stderr, "siglab serve: listening on http://%s/\n", *addr)
	if err := srv.Run(ctx); err != nil {
		rep.Fatal(1, fmt.Errorf("siglab serve: %w", err))
	}
}
