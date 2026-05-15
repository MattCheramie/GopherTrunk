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
	// Pure-Go RTL-SDR driver. Registers under the canonical name
	// "rtlsdr"; PR-09 removed the legacy CGO librtlsdr backend that
	// previously coexisted under "rtlsdr-cgo".
	_ "github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/purego"
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
		fmt.Println(version.String())
	case "sdr":
		runSDR(os.Args[2:])
	case "audio":
		runAudio(os.Args[2:])
	case "tui":
		runTUI(os.Args[2:])
	case "decode":
		runDecode(os.Args[2:])
	case "import-pdf":
		runImport(os.Args[2:])
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
  gophertrunk sdr list [--probe]      list discovered SDR devices (--probe opens each to fill TUNER + gains)
  gophertrunk audio list              list audio output devices
  gophertrunk tui [-server URL]       open the read-only operator TUI
  gophertrunk decode [flags]          decode a captured .raw frame stream into a WAV
  gophertrunk import-pdf [flags]      import a RadioReference PDF into config.yaml
  gophertrunk version                 print build version
  gophertrunk help                    show this message`)
}

func runDaemon(args []string) {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to YAML config (optional)")
	logLevel := fs.String("log-level", "", "override log level (debug|info|warn|error)")
	logFormat := fs.String("log-format", "", "override log format (text|json)")
	_ = fs.Parse(args)

	// No -config passed: walk the standard discovery precedence
	// ($GOPHERTRUNK_CONFIG → UserConfigDir → Documents → cwd) so
	// the Windows installer's chosen path (and equivalent setups
	// on other platforms) is picked up automatically. Empty result
	// means "use built-in defaults" — Load handles that case.
	if *cfgPath == "" {
		if discovered := config.Discover(); discovered != "" {
			fmt.Fprintf(os.Stderr, "config: loaded %s\n", discovered)
			*cfgPath = discovered
		}
	}

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

	logger.Info("gophertrunk starting", "version", version.String())

	// Patent-posture banner — AMBE+2 decoding is patent-encumbered
	// in some jurisdictions (DVSI IPR portfolio). The pure-Go
	// decoder ships unconditionally as a clean-room implementation,
	// but operators in those jurisdictions may need a license. The
	// full discussion lives in docs/vocoders.md §"Patent posture";
	// this one-line banner surfaces the link at startup so
	// operators see it without having to grep the repo. Suppress
	// with GOPHERTRUNK_QUIET_BANNER=1 (intended for CI / test
	// harnesses where the banner is just noise).
	if os.Getenv("GOPHERTRUNK_QUIET_BANNER") == "" {
		logger.Info("AMBE+2 voice decoding is patent-encumbered in some jurisdictions; see docs/vocoders.md §\"Patent posture\"")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	d, err := NewDaemon(cfg, version.String(), logger)
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
		fmt.Fprintln(os.Stderr, "usage: gophertrunk sdr list [--probe]")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listSDRs(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown sdr subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func listSDRs(args []string) {
	probe := false
	for _, a := range args {
		switch a {
		case "--probe", "-probe":
			probe = true
		default:
			fmt.Fprintf(os.Stderr, "unknown sdr list flag: %s\n", a)
			os.Exit(2)
		}
	}

	infos := sdr.EnumerateAll()
	if len(infos) == 0 {
		fmt.Println("no SDR devices found")
		return
	}

	// --probe: open each device long enough to run the demod + tuner
	// bring-up so TunerName and the gain ladder can be filled in. Each
	// device is closed before the next is opened to avoid claiming two
	// dongles at once. Failures don't abort the loop — the row just
	// keeps the empty fields from Enumerate and the error is printed
	// to stderr so the operator can see why probing failed.
	if probe {
		for i := range infos {
			d, err := sdr.DriverByName(infos[i].Driver)
			if err != nil {
				fmt.Fprintf(os.Stderr, "probe %s[%d]: %v\n", infos[i].Driver, infos[i].Index, err)
				continue
			}
			dev, err := d.Open(infos[i].Index)
			if err != nil {
				fmt.Fprintf(os.Stderr, "probe %s[%d]: %v\n", infos[i].Driver, infos[i].Index, err)
				continue
			}
			probed := dev.Info()
			infos[i].TunerName = probed.TunerName
			infos[i].Gains = probed.Gains
			_ = dev.Close()
		}
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
