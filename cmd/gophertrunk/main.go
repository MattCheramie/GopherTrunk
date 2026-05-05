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

	// Blank import: under default builds this pulls in the stub
	// (no init effect); under `make build TAGS=mbelib` it pulls in
	// the CGO wrapper that registers the `imbe` and `ambe2`
	// vocoders against voice.DefaultRegistry. Either way, the
	// daemon's import set picks the right path automatically.
	_ "github.com/MattCheramie/GopherTrunk/internal/voice/mbelib"
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
