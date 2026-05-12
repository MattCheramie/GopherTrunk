package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui"
	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// runTUI launches the operator TUI against a running daemon.
//
// The TUI authenticates to the daemon's mutation endpoints with a
// Bearer token when `-token` / `-token-file` (or the
// GOPHERTRUNK_API_TOKEN env var) is set. Tokens are required when the
// daemon runs with `api.auth.mode: required` or `auto` on a public
// bind; loopback binds under `auto` work without a token.
func runTUI(args []string) {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8080", "daemon base URL")
	insecure := fs.Bool("insecure", false, "skip TLS verification (https only)")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout")
	noColor := fs.Bool("no-color", false, "disable ANSI colours")
	write := fs.Bool("write", false, "enable mutation keybindings (end call, set priority/lockout, retention sweep, tone reset). Daemon must also accept mutations (api.auth.mode or legacy api.allow_mutations).")
	token := fs.String("token", "", "API bearer token (sent as Authorization: Bearer <token>); falls back to GOPHERTRUNK_API_TOKEN")
	tokenFile := fs.String("token-file", "", "path to a file containing the API bearer token; re-read on every request so daemon-side rotation works without a restart")
	_ = fs.Parse(args)

	cli := client.New(*server, *timeout, *insecure)
	// Resolve token: -token-file > -token > GOPHERTRUNK_API_TOKEN
	if *tokenFile != "" {
		if err := cli.SetTokenFile(*tokenFile); err != nil {
			fmt.Fprintf(os.Stderr, "tui: -token-file: %v\n", err)
			os.Exit(1)
		}
	} else if *token != "" {
		cli.SetToken(*token)
	} else if env := os.Getenv("GOPHERTRUNK_API_TOKEN"); env != "" {
		cli.SetToken(env)
	}

	m := tui.New(cli, tui.Options{NoColor: *noColor, Write: *write})
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
