package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	fmt.Fprintln(os.Stderr, `gophertrunk — P25/DMR/NXDN trunking engine

USAGE:
  gophertrunk [run] [-config path]    run the daemon (interactive launcher on a TTY)
  gophertrunk -tui                    launch in-process TUI after daemon is ready
  gophertrunk -web                    open the bundled web UI after daemon is ready
  gophertrunk -headless               skip the launcher (default for non-TTY stdin)
  gophertrunk sdr list [--probe]      list discovered SDR devices (--probe opens each to fill TUNER + gains)
  gophertrunk audio list              list audio output devices
  gophertrunk tui [-server URL]       open the operator TUI against a remote daemon
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
	// Launcher flags. Mutually exclusive; default is "auto" which
	// prints an interactive menu on a TTY and stays headless
	// otherwise.
	wantTUI := fs.Bool("tui", false, "launch the in-process operator TUI after the daemon comes up")
	wantWeb := fs.Bool("web", false, "open the bundled web UI in the system browser after the daemon comes up")
	wantHL := fs.Bool("headless", false, "skip the launcher prompt; daemon runs silent (default for non-TTY stdin)")
	_ = fs.Parse(args)

	mode, err := pickLaunchMode(*wantTUI, *wantWeb, *wantHL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "launcher: %v\n", err)
		os.Exit(2)
	}

	// No -config passed: walk the standard discovery precedence
	// ($GOPHERTRUNK_CONFIG → UserConfigDir → Documents → cwd) so
	// the Windows installer's chosen path (and equivalent setups
	// on other platforms) is picked up automatically. When the
	// chosen directory holds more than one *.yaml / *.yml the
	// picker prompts the operator on stdin. Empty result means
	// "use built-in defaults" — Load handles that case.
	if *cfgPath == "" {
		discovered, err := config.DiscoverWith(config.DiscoverOptions{Pick: pickConfigInteractive})
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(2)
		}
		if discovered != "" {
			fmt.Fprintf(os.Stderr, "config: loaded %s\n", discovered)
			*cfgPath = discovered
		}
	}

	// First-run fast-fail: no config discoverable, no -config, and
	// stdin isn't a TTY → we can't prompt and there's no useful
	// daemon to run. Exit with EX_CONFIG so service managers can
	// distinguish "missing config" from generic failures.
	if *cfgPath == "" && !stdinIsTerminal() && os.Getenv("GOPHERTRUNK_CONFIG") == "" {
		fmt.Fprintln(os.Stderr,
			"gophertrunk: no config found and stdin is not a TTY; pass -config or set GOPHERTRUNK_CONFIG (see docs/quickstart.md)")
		os.Exit(78)
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
	logger, logSwap := gtlog.NewWithSwap(cfg.Log.Level, cfg.Log.Format)

	logger.Info("gophertrunk starting", "version", version.String())

	// Launcher pre-checks before we burn time spinning up the
	// daemon: an operator who passed -tui or -web with no HTTP API
	// in config should hear about it now, not after engine init.
	if (mode == launchTUI || mode == launchWeb) && cfg.API.HTTPAddr == "" {
		fmt.Fprintf(os.Stderr,
			"launcher: -%s requires api.http_addr in config (current value is empty)\n",
			launchModeFlag(mode))
		os.Exit(2)
	}
	if mode == launchTUI && (!stdinIsTerminal() || !stdoutIsTerminal()) {
		fmt.Fprintln(os.Stderr,
			"launcher: -tui requires an interactive terminal (stdin + stdout TTY); use -web or -headless")
		os.Exit(2)
	}

	preflightWarnings, err := preflight(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

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

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	d, err := NewDaemon(cfg, version.String(), logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon init: %v\n", err)
		os.Exit(1)
	}
	for _, w := range preflightWarnings {
		d.addWarning(w)
	}

	// Spawn the daemon's Run in a goroutine so the launcher can
	// gate on Ready and then attach the TUI / browser on the same
	// goroutine that owns stdin/stdout. Daemon.Run blocks until ctx
	// cancels, which is exactly what we want for the main goroutine
	// to wait on once the launcher has decided what to do.
	runErr := make(chan error, 1)
	go func() {
		runErr <- d.Run(ctx)
	}()

	// Wait for either Ready (HTTP listener bound, components
	// settled) or the daemon's Run to return early (essential
	// component failed before Ready fired).
	select {
	case <-d.Ready():
	case err := <-runErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Daemon is up. Hand control to the launcher.
	runLauncher(ctx, d, logger, logSwap, mode)

	// Wait for the daemon goroutine to finish (SIGINT/SIGTERM →
	// ctx cancels → Run unwinds → returns).
	if err := <-runErr; err != nil && !errors.Is(err, context.Canceled) {
		logger.Warn("daemon exited", "err", err)
		os.Exit(1)
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

// pickConfigInteractive is the DiscoverOptions.Pick callback for
// runDaemon. When stdin is a terminal, it prints a numbered list of
// the candidate configs and reads the operator's choice. When stdin
// isn't a terminal (Windows service, systemd unit, CI), it falls
// back to the first match with a stderr warning so the daemon can
// still start unattended — the operator can pin a specific file
// later via -config or GOPHERTRUNK_CONFIG.
func pickConfigInteractive(paths []string) (string, error) {
	if !stdinIsTerminal() {
		fmt.Fprintf(os.Stderr,
			"config: multiple config files in %s, defaulting to %s (set -config or GOPHERTRUNK_CONFIG to pick a specific one)\n",
			filepath.Dir(paths[0]), paths[0])
		return paths[0], nil
	}
	fmt.Fprintf(os.Stderr, "Multiple config files found in %s:\n", filepath.Dir(paths[0]))
	for i, p := range paths {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, filepath.Base(p))
	}
	fmt.Fprintf(os.Stderr, "Pick one [1-%d, default 1]: ", len(paths))

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		// EOF on stdin (closed pipe, Ctrl+D) — same fallback as
		// the non-TTY branch: keep the daemon startable.
		fmt.Fprintln(os.Stderr)
		return paths[0], nil
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return paths[0], nil
	}
	idx, perr := strconv.Atoi(line)
	if perr != nil || idx < 1 || idx > len(paths) {
		return "", fmt.Errorf("invalid config selection %q (want 1..%d)", line, len(paths))
	}
	return paths[idx-1], nil
}

// stdinIsTerminal returns true when stdin is attached to a character
// device (i.e. an interactive terminal). False for pipes, redirected
// input, service runners, and detached background processes.
func stdinIsTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// stdoutIsTerminal mirrors stdinIsTerminal for stdout. Both must be
// TTYs before -tui can drive bubbletea's alt-screen renderer.
func stdoutIsTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
