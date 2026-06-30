package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/rfscope/webserver"
	rfscopeweb "github.com/MattCheramie/GopherTrunk/web/rfscope"
)

// runRFScopeServe is the entry point for `gophertrunk rfscope serve` — the
// standalone RF Scope web console. It runs the self-contained rfscope webserver
// (no SDR or daemon), serving the embedded SPA at / and the
// /api/v1/rfscope/* routes, analyzing uploaded captures in the browser. It is
// the browser counterpart to the `rfscope` CLI/TUI, modeled on `cryptolab serve`.
func runRFScopeServe(args []string) {
	fs := flag.NewFlagSet("rfscope serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8098", "listen address")
	open := fs.Bool("open", false, "open the console in the system browser once it is up")
	tmpDir := fs.String("tmp-dir", "", "directory for staged capture uploads (default: a fresh temp dir)")
	maxUpload := fs.Int64("max-upload", 0, "max capture upload size in bytes (0 = default 512 MiB)")
	logLevel := fs.String("log-level", "info", "log level: debug|info|warn|error")
	logFormat := fs.String("log-format", "text", "log format: text|json")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk rfscope serve — standalone RF Scope web console.

USAGE:
  gophertrunk rfscope serve [-addr host:port] [-open] [-tmp-dir dir] [-max-upload bytes]

Serves the RF Scope SPA at http://<addr>/ and the rfscope REST API, analyzing
uploaded IQ captures offline (no SDR or daemon). Build the SPA first with
`+"`make rfscope-web-build`"+`.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	rep := newReporter("rfscope serve")

	opts := webserver.Options{
		TempDir:        *tmpDir,
		MaxUploadBytes: *maxUpload,
		Logger:         gtlog.New(*logLevel, *logFormat),
	}
	if rfscopeweb.HasAssets() {
		opts.Assets = rfscopeweb.Assets()
	} else {
		fmt.Fprintln(os.Stderr, "rfscope serve: SPA not bundled (run `make rfscope-web-build` first); serving API only")
	}

	srv, err := webserver.New(opts)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("rfscope serve: %w", err))
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
				fmt.Fprintf(os.Stderr, "rfscope serve: open browser: %v\n", err)
			}
		}()
	}

	fmt.Fprintf(os.Stderr, "rfscope serve: listening on http://%s/\n", *addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		rep.Fatal(1, fmt.Errorf("rfscope serve: %w", err))
	}
}
