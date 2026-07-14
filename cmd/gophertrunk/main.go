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
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/diag"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	// Pure-Go SDR drivers. Each registers itself under its canonical
	// name on init; the blank import is what actually links the
	// package into the binary.
	_ "github.com/MattCheramie/GopherTrunk/internal/sdr/airspy"
	_ "github.com/MattCheramie/GopherTrunk/internal/sdr/airspyhf"
	_ "github.com/MattCheramie/GopherTrunk/internal/sdr/hackrf"
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
	case "replay":
		runReplay(os.Args[2:])
	case "analyze":
		runAnalyze(os.Args[2:])
	case "identify":
		runIdentify(os.Args[2:])
	case "spectrum":
		runSpectrum(os.Args[2:])
	case "hunt":
		runHunt(os.Args[2:])
	case "rfscope":
		runRFScope(os.Args[2:])
	case "gen":
		runGen(os.Args[2:])
	case "capture":
		runCapture(os.Args[2:])
	case "test":
		runSiglabTest(os.Args[2:])
	case "siglab":
		// `siglab serve` launches the standalone web console; `siglab sweep`
		// runs the demod EVM/SNR-vs-SNR benchmark; bare `siglab` (and any
		// other subarg) stays the offline TUI.
		switch {
		case len(os.Args) > 2 && os.Args[2] == "serve":
			runSiglabServe(os.Args[3:])
		case len(os.Args) > 2 && os.Args[2] == "sweep":
			runSiglabSweep(os.Args[3:])
		default:
			runSiglabTUI(os.Args[2:])
		}
	case "config":
		// `config serve` launches the web Config Builder; `config tui` (and
		// bare `config`) launch the terminal Config Builder.
		switch {
		case len(os.Args) > 2 && os.Args[2] == "serve":
			runConfigServe(os.Args[3:])
		case len(os.Args) > 2 && os.Args[2] == "tui":
			runConfigTUI(os.Args[3:])
		default:
			runConfigTUI(os.Args[2:])
		}
	case "cryptolab":
		// Optional cryptographic-research toolkit. The real dispatch is
		// linked only with -tags cryptolab; the default build links a stub
		// that explains how to opt in (see cryptolab_{enabled,disabled}.go).
		runCryptolab(os.Args[2:])
	case "bundle":
		// GopherTrunk Bundle (.gtb.tar.gz): pack/inspect/verify/extract/commit a
		// single-file capture-to-analysis case. Distinct from hunt's CSV
		// "import bundle" (that CSV is stored inside a GopherTrunk Bundle).
		runBundle(os.Args[2:])
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
  gophertrunk sdr doctor [-v]         diagnose why a dongle is not recognized (per-OS driver-binding report)
  gophertrunk audio list              list audio output devices
  gophertrunk tui [-server URL]       open the operator TUI against a remote daemon
  gophertrunk decode [flags]          decode a captured .raw frame stream into a WAV
  gophertrunk replay [flags]          decode a raw IQ capture file offline (any protocol)
  gophertrunk analyze [flags]         decode + analyze a capture with structured export (json/yaml/csv)
  gophertrunk identify [flags]        auto-detect the protocol in a capture, then analyze it
  gophertrunk spectrum [flags]        print a capture's power spectrum + detected carriers + RMS (no decode)
  gophertrunk hunt [flags]            discover & map an unknown trunked system, export it + an RR submission package
  gophertrunk gen [flags]             synthesize a test IQ capture + metadata for a protocol
  gophertrunk capture [flags]         record raw IQ off a live SDR to a .cfile + metadata sidecar
  gophertrunk bundle <cmd> [flags]    build/inspect a GopherTrunk Bundle (.gtb.tar.gz): pack|info|verify|extract|add|commit
  gophertrunk test [flags]            decode a capture and grade it against acceptance criteria
  gophertrunk cryptolab <tool> ...    cryptographic-research toolkit (optional; build with -tags cryptolab)
  gophertrunk siglab [flags]          standalone replay/test/analysis TUI
  gophertrunk siglab serve [flags]    offline signal-analysis web console (browser UI)
  gophertrunk config serve [flags]    standalone web Config Builder/Editor (browser UI)
  gophertrunk config [tui] [flags]    standalone terminal Config Builder/Editor (no browser)
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
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures (also set via diagnostics.verbose_errors or GOPHERTRUNK_VERBOSE_ERRORS)")
	// IQ capture diagnostic — taps a live SDR's iqtap broker and writes
	// raw IQ samples to a file for offline analysis. Used to capture a
	// reproducible fixture for replay (issue #402).
	iqCapture := fs.String("iq-capture", "", "capture raw IQ from a live SDR for offline analysis (issue #402). Format: serial=<s>,path=<file>,seconds=<n>[,format=u8|f32] (default format=f32, GNU Radio cfile)")
	_ = fs.Parse(args)

	// Resolve verbose-error reporting from the flag now (config folds in
	// below, env was already picked up at package init) so every error
	// site in runDaemon routes through the shared diagnostics reporter.
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("")

	// Parse the iq-capture spec early so a typo doesn't blow up after
	// the daemon's been running for 10 seconds.
	captureSpec, err := parseIQCaptureSpec(*iqCapture)
	if err != nil {
		rep.Fatal(2, err)
	}

	mode, err := pickLaunchMode(*wantTUI, *wantWeb, *wantHL)
	if err != nil {
		rep.Fatalf(2, "launcher: %v", err)
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
			rep.Fatalf(2, "config: %v", err)
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
		rep.Fatalf(78,
			"gophertrunk: no config found and stdin is not a TTY; pass -config or set GOPHERTRUNK_CONFIG (see docs/quickstart.md)")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		rep.Fatalf(2, "config: %v", err)
	}
	// Fold the config's verbose-error setting into the resolved value so
	// the flag/env still win but config.diagnostics.verbose_errors takes
	// effect for everything from here on (daemon, API, gRPC).
	resolveVerbose(*verboseFlag, cfg.Diagnostics.VerboseErrors)
	cfg.Diagnostics.VerboseErrors = verboseErrors
	rep.Verbose = verboseErrors
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}
	if *logFormat != "" {
		cfg.Log.Format = *logFormat
	}
	logger, logSwap := gtlog.NewWithSwap(cfg.Log.Level, cfg.Log.Format)

	logger.Info("gophertrunk starting", "version", version.String())

	// Bound the resident footprint so a long live run isn't SIGKILLed by
	// the OS memory-pressure killer with no in-process trace (issue #492).
	applyMemoryLimit(cfg, logger)

	// Launcher pre-checks before we burn time spinning up the
	// daemon: an operator who passed -tui or -web with no HTTP API
	// in config should hear about it now, not after engine init.
	if (mode == launchTUI || mode == launchWeb) && cfg.API.HTTPAddr == "" {
		rep.Fatalf(2,
			"launcher: -%s requires api.http_addr in config (current value is empty)",
			launchModeFlag(mode))
	}
	if mode == launchTUI && (!stdinIsTerminal() || !stdoutIsTerminal()) {
		rep.Fatalf(2,
			"launcher: -tui requires an interactive terminal (stdin + stdout TTY); use -web or -headless")
	}

	preflightWarnings, err := preflight(cfg)
	if err != nil {
		rep.Fatal(2, err)
	}

	// Single-instance lock. Two daemons aimed at the same config
	// would race the RTL-SDR USB claim and crash both libusb hands;
	// this surfaces the contention as a clear error before either
	// touches the radio.
	releaseLock, err := acquireInstanceLock(*cfgPath)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer releaseLock()

	// Patent-posture notice — AMBE+2 decoding is patent-encumbered
	// in some jurisdictions (DVSI IPR portfolio). The pure-Go
	// decoder ships unconditionally as a clean-room implementation,
	// but operators in those jurisdictions may need a license. The
	// full discussion lives in docs/vocoders.md §"Patent posture".
	// Threaded through the startup-warnings channel so it surfaces
	// in the launcher menu / TUI dashboard / runtime DTO rather
	// than scrolling past on the daemon log right before the
	// interactive prompt. Suppress with GOPHERTRUNK_QUIET_BANNER=1
	// (CI / test harnesses).
	if os.Getenv("GOPHERTRUNK_QUIET_BANNER") == "" {
		preflightWarnings = append(preflightWarnings,
			"AMBE+2 voice decoding is patent-encumbered in some jurisdictions; see docs/vocoders.md §\"Patent posture\"")
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	d, err := NewDaemonWithPath(cfg, *cfgPath, version.String(), logger)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("daemon init: %w", err))
	}
	// One-time diagnostics banner in the daemon log so a captured log
	// carries the same macro context an error banner would — including
	// the dongles the now-open pool actually claimed. Suppressed by
	// GOPHERTRUNK_QUIET_BANNER (CI/tests).
	if banner := diag.FormatBannerPlain(d.newDiagCollector().SysInfo()); banner != "" {
		logger.Info("diagnostics", "banner", banner)
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
		// Convert a panic in the daemon run path into a logged error +
		// clean main-goroutine exit rather than a silent crash (#492).
		defer gtlog.Recover(logger, "daemon-run", func(err error) { runErr <- err })
		runErr <- d.Run(ctx)
	}()

	// SIGHUP → hot-reload config.yaml (Unix only; no-op on Windows).
	// watchReloadSignal registers signal.Notify synchronously and
	// then spawns its own goroutine, so the call returns
	// immediately and the signal handler is in place by the time
	// we move on.
	watchReloadSignal(ctx, d, logger)

	// Wait for either Ready (HTTP listener bound, components
	// settled) or the daemon's Run to return early (essential
	// component failed before Ready fired).
	select {
	case <-d.Ready():
	case err := <-runErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			rep.Fatal(1, fmt.Errorf("daemon: %w", err))
		}
		return
	}

	// --iq-capture (issue #402 diagnostic): with the daemon up,
	// subscribe to the requested SDR's broker and dump raw IQ to disk
	// for offline analysis via `gophertrunk replay`. Runs concurrently
	// with normal decoding; non-fatal on its own — a capture failure
	// is logged but doesn't tear the daemon down.
	if captureSpec.Serial != "" {
		br := d.IQBroker(captureSpec.Serial)
		if br == nil {
			rep.Fatalf(2,
				"iq-capture: no SDR with serial %q in pool (have: %v)",
				captureSpec.Serial, d.IQBrokerSerials())
		}
		go func() {
			defer gtlog.Recover(logger, "iq-capture", nil)
			if err := runIQCapture(ctx, br, captureSpec, logger); err != nil &&
				!errors.Is(err, context.Canceled) {
				logger.Warn("iq-capture: failed", "err", err)
			}
		}()
	}

	// Daemon is up. Hand control to the launcher.
	runLauncher(ctx, d, logger, logSwap, mode)

	// Wait for the daemon goroutine to finish (SIGINT/SIGTERM →
	// ctx cancels → Run unwinds → returns).
	runExitErr := <-runErr

	// Config hot-swap with "restart" mode: the operator picked a new
	// config file in the web Config Builder and asked for a full restart.
	// Tear the daemon down, free the single-instance lock + SDRs, then
	// re-exec into the new -config so every field (SDR, systems, …) takes
	// effect. performRestart replaces the process image on success.
	if newPath := d.RestartPath(); newPath != "" {
		d.Close()
		releaseLock()
		logger.Info("restarting into new config", "path", newPath)
		if err := performRestart(newPath); err != nil {
			logErr(logger, verboseErrors, "config restart failed", err)
			os.Exit(1)
		}
		return // unreachable on success (process image replaced)
	}

	if runExitErr != nil && !errors.Is(runExitErr, context.Canceled) {
		logErr(logger, verboseErrors, "daemon exited", runExitErr)
		os.Exit(1)
	}
}

func runSDR(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gophertrunk sdr (list [--probe] | doctor [-v])")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listSDRs(args[1:])
	case "doctor":
		runSDRDoctor(args[1:])
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

	infos, errs := sdr.EnumerateAll()
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, "enumerate:", err)
	}
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
			probed, err := probeDevice(d, infos[i].Index, probeTimeout)
			if err != nil {
				fmt.Fprintf(os.Stderr, "probe %s[%d]: %v\n", infos[i].Driver, infos[i].Index, err)
				continue
			}
			infos[i].TunerName = probed.TunerName
			infos[i].Gains = probed.Gains
		}
	}

	fmt.Print(formatSDRTable(infos))
}

// probeTimeout bounds how long a single device's open+info+close may take
// during `sdr list --probe`. Each USB control transfer inside Open already
// carries its own 1 s timeout, but the open as a whole is otherwise
// unbounded: a wedged device or transport (firmware that NAKs, a clone in a
// bad state, a stalled control ioctl) would hang the command with no output.
// This deadline guarantees `--probe` always returns.
const probeTimeout = 5 * time.Second

// probeDevice opens drv[idx], reads its Info, and closes it, all bounded by
// timeout. The open/info/close runs on a goroutine so a device that wedges
// can't block the caller: on timeout we return an error and let the
// goroutine finish (and close the handle) on its own — harmless for a
// short-lived CLI. Driver.Open takes no context, so the bound has to live
// here at the call site rather than inside the driver.
func probeDevice(drv sdr.Driver, idx int, timeout time.Duration) (sdr.Info, error) {
	type result struct {
		info sdr.Info
		err  error
	}
	done := make(chan result, 1)
	go func() {
		dev, err := drv.Open(idx)
		if err != nil {
			done <- result{err: err}
			return
		}
		info := dev.Info()
		_ = dev.Close()
		done <- result{info: info}
	}()
	select {
	case r := <-done:
		return r.info, r.err
	case <-time.After(timeout):
		return sdr.Info{}, fmt.Errorf("timed out after %s", timeout)
	}
}

// formatSDRTable renders the `sdr list` table with per-column widths sized
// to the widest value actually present (header label or any row), so values
// like a HackRF's 32-hex serial or the full "MAX2839+MAX5864" tuner string
// print in full instead of being silently truncated. The trailing gains
// column is free-form.
func formatSDRTable(infos []sdr.Info) string {
	const (
		hDriver  = "DRIVER"
		hIndex   = "IDX"
		hSerial  = "SERIAL"
		hTuner   = "TUNER"
		hProduct = "PRODUCT"
	)
	wDriver, wIndex, wSerial, wTuner, wProduct :=
		len(hDriver), len(hIndex), len(hSerial), len(hTuner), len(hProduct)
	for _, i := range infos {
		wDriver = max(wDriver, len(i.Driver))
		wIndex = max(wIndex, len(strconv.Itoa(i.Index)))
		wSerial = max(wSerial, len(i.Serial))
		wTuner = max(wTuner, len(i.TunerName))
		wProduct = max(wProduct, len(i.Product))
	}
	var b strings.Builder
	rowFmt := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%v\n",
		wDriver, wIndex, wSerial, wTuner, wProduct)
	fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %-*s  %-*s  gains(0.1 dB)\n",
		wDriver, hDriver, wIndex, hIndex, wSerial, hSerial, wTuner, hTuner, wProduct, hProduct)
	for _, i := range infos {
		fmt.Fprintf(&b, rowFmt, i.Driver, strconv.Itoa(i.Index), i.Serial, i.TunerName, i.Product, i.Gains)
	}
	return b.String()
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
