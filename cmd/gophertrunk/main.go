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

	if len(cfg.SDR.Devices) > 0 {
		pool := sdr.NewPool(logger)
		var hints []sdr.Hint
		for _, d := range cfg.SDR.Devices {
			hints = append(hints, sdr.Hint{Serial: d.Serial, Role: sdr.ParseRole(d.Role)})
		}
		if err := pool.Open(hints); err != nil {
			logger.Warn("sdr pool open failed", "err", err)
		} else {
			defer pool.Close()
			for _, e := range pool.Entries() {
				logger.Info("sdr ready", "driver", e.Info.Driver, "serial", e.Info.Serial, "role", e.Role.String())
			}
		}
	} else {
		logger.Info("no SDR devices configured; idling. Configure devices in YAML to begin.")
	}

	<-ctx.Done()
	logger.Info("shutting down")
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
