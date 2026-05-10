package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	_ "github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
	"github.com/MattCheramie/GopherTrunk/internal/version"

	// Blank import: pulls in the pure-Go IMBE decoder so the daemon
	// registers the "imbe" vocoder name regardless of build tags.
	// This is the sole IMBE backend in default builds.
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/imbe"

	// Blank import: pulls in the pure-Go AMBE+2 decoder so the
	// daemon registers the "ambe2" vocoder name (P25 Phase 2, DMR,
	// NXDN voice) regardless of build tags. The skeleton currently
	// emits silence; PR-D plugs in 49-bit parameter unpacking and
	// PR-E wires the shared mbe synthesis pipeline. See
	// docs/vocoders.md for the AMBE+2 patent posture.
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
)

func main() {
	if len(os.Args) < 2 {
		runDaemon(os.Args[1:])
		return
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(version.Version)
	case "sdr":
		runSDR(os.Args[2:])
	case "tui":
		runTUI(os.Args[2:])
	case "daemon", "run":
		runDaemon(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		runDaemon(os.Args[1:])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `gophertrunk — headless P25/DMR/NXDN trunking engine

USAGE:
  gophertrunk [run] [-config path]    run the daemon (default)
  gophertrunk sdr list                list discovered SDR devices
  gophertrunk tui [-server URL]       open the read-only operator TUI
  gophertrunk version                 print build version
  gophertrunk help                    show this message`)
}

func runDaemon(args []string) {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to YAML config (optional)")
	logLevel := fs.String("log-level", "", "override log level (debug|info|warn|error)")
	logFormat := fs.String("log-format", "", "override log format (text|json)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}
	if *logFormat != "" {
		cfg.Log.Format = *logFormat
	}
	logger := gtlog.New(cfg.Log.Level, cfg.Log.Format)

	logger.Info("gophertrunk starting", "version", version.Version)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	d, err := NewDaemon(cfg, version.Version, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon init: %v\n", err)
		os.Exit(1)
	}
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		logger.Warn("daemon exited", "err", err)
	}
}

func runSDR(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gophertrunk sdr list")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listSDRs()
	default:
		fmt.Fprintf(os.Stderr, "unknown sdr subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func listSDRs() {
	infos := sdr.EnumerateAll()
	if len(infos) == 0 {
		fmt.Println("no SDR devices found")
		return
	}
	fmt.Printf("%-8s  %-3s  %-16s  %-8s  %-8s  gains(0.1 dB)\n", "DRIVER", "IDX", "SERIAL", "TUNER", "PRODUCT")
	for _, i := range infos {
		fmt.Printf("%-8s  %-3d  %-16s  %-8s  %-8s  %v\n",
			i.Driver, i.Index, truncate(i.Serial, 16), truncate(i.TunerName, 8), truncate(i.Product, 8), i.Gains)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
