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

// runTUI launches the read-only operator TUI against a running
// daemon. v1 is a viewer only; mutation controls land in a follow-up.
func runTUI(args []string) {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8080", "daemon base URL")
	insecure := fs.Bool("insecure", false, "skip TLS verification (https only)")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout")
	noColor := fs.Bool("no-color", false, "disable ANSI colours")
	_ = fs.Parse(args)

	cli := client.New(*server, *timeout, *insecure)
	m := tui.New(cli, tui.Options{NoColor: *noColor})
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
