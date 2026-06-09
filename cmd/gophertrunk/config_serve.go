package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
	"github.com/MattCheramie/GopherTrunk/internal/version"
	configbuilderweb "github.com/MattCheramie/GopherTrunk/web/configbuilder"
)

// runConfigServe is the entry point for `gophertrunk config serve` — a
// standalone web Config Builder/Editor. It runs the api.Server with only
// the config-builder subsystem wired (no SDR, daemon, or storage), serving
// the embedded builder SPA at / and the /api/v1/config/* routes. Browse
// existing config files, build a new one section-by-section with inline
// validation, browse/import from RadioReference, import PDF/CSV, and save.
func runConfigServe(args []string) {
	fs := flag.NewFlagSet("config serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8077", "listen address")
	open := fs.Bool("open", false, "open the builder in the system browser once it is up")
	configDir := fs.String("config-dir", "", "directory to browse and save configs in (default: the standard discovery locations)")
	token := fs.String("token", "", "bearer token required for save/mutations (needed on non-loopback binds)")
	tokenFile := fs.String("token-file", "", "path to a file holding the bearer token (reloaded per request)")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk config serve — standalone web Config Builder/Editor.

USAGE:
  gophertrunk config serve [-addr host:port] [-open] [-config-dir dir]

Serves the Config Builder SPA at http://<addr>/ and the config REST API.
Browse and edit config files, validate each section, browse/import from
RadioReference.com, import PDF/CSV, and save — no SDR or daemon required.

RadioReference browse/import uses GOPHERTRUNK_RR_KEY / GOPHERTRUNK_RR_USER
/ GOPHERTRUNK_RR_PASS when set.

The default loopback bind (127.0.0.1) trusts local requests, so saving
works out of the box. Binding to a non-loopback -addr requires a bearer
token on the save endpoint (api.auth) — reads, validate, and Download YAML
still work, but writes will be rejected until auth is configured.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("config serve")

	bus := events.NewBus(64)
	defer bus.Close()

	opts := api.ServerOptions{
		Addr:    *addr,
		Bus:     bus,
		Version: version.String(),
		// Auth: default mode trusts loopback (saves work on 127.0.0.1
		// out of the box). A -token / -token-file makes the save endpoint
		// require that bearer token on non-loopback binds (the SPA has a
		// token field); reads stay open either way.
		Auth: api.AuthConfig{Token: strings.TrimSpace(*token), TokenFile: strings.TrimSpace(*tokenFile)},
		// Parse-only importer: daemonImporter.Parse never touches the
		// (nil) Daemon, so POST /api/v1/config/parse works standalone.
		Importer: &daemonImporter{},
		ConfigBuilder: api.ConfigBuilderOptions{
			Enabled:   true,
			ConfigDir: *configDir,
			RadioReference: radioreference.ResolveAuth(radioreference.Auth{
				AppKey:   strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_KEY")),
				Username: strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_USER")),
				Password: strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_PASS")),
			}),
		},
	}
	// Standalone: serve the builder SPA at / (like `siglab serve`).
	if configbuilderweb.HasAssets() {
		opts.WebAssets = configbuilderweb.Assets()
	} else {
		fmt.Fprintln(os.Stderr, "config serve: SPA not bundled (run `make configbuilder-web-build` first); serving API only")
	}

	srv, err := api.NewServer(opts)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("config serve: %w", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *open {
		go func() {
			time.Sleep(300 * time.Millisecond)
			url := "http://" + *addr + "/"
			if err := openBrowser(url); err != nil {
				fmt.Fprintf(os.Stderr, "config serve: open browser: %v\n", err)
			}
		}()
	}

	fmt.Fprintf(os.Stderr, "config serve: listening on http://%s/\n", *addr)
	if err := srv.Run(ctx); err != nil {
		rep.Fatal(1, fmt.Errorf("config serve: %w", err))
	}
}
