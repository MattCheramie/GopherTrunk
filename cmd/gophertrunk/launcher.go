package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/tui"
	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	gtweb "github.com/MattCheramie/GopherTrunk/web"
)

// launchMode discriminates the launcher's terminal action. launchAuto
// is the no-flags default — prompt the operator on a TTY, stay
// headless otherwise.
type launchMode int

const (
	launchAuto launchMode = iota
	launchTUI
	launchWeb
	launchHeadless
)

// pickLaunchMode reconciles the three mutually-exclusive flag values
// into a single launchMode. Multiple flags set at once is a hard
// error so the operator hears about the misuse instead of getting a
// silent precedence rule.
func pickLaunchMode(tui, web, headless bool) (launchMode, error) {
	set := 0
	if tui {
		set++
	}
	if web {
		set++
	}
	if headless {
		set++
	}
	if set > 1 {
		return launchAuto, errors.New("-tui, -web, and -headless are mutually exclusive")
	}
	switch {
	case tui:
		return launchTUI, nil
	case web:
		return launchWeb, nil
	case headless:
		return launchHeadless, nil
	default:
		return launchAuto, nil
	}
}

// launchModeFlag returns the operator-facing flag name for a mode,
// used in error messages so the printed text matches what the
// operator typed on the command line.
func launchModeFlag(m launchMode) string {
	switch m {
	case launchTUI:
		return "tui"
	case launchWeb:
		return "web"
	case launchHeadless:
		return "headless"
	}
	return "auto"
}

// runLauncher is the post-Ready hook. It surfaces any startup
// warnings, optionally prompts the operator for a UI, and then runs
// the chosen UI in-process (TUI) or out-of-process (browser open).
// Returns when the chosen UI has finished or immediately for
// headless modes — the caller is expected to keep waiting on Run.
func runLauncher(ctx context.Context, d *Daemon, log *slog.Logger, logSwap *gtlog.SwappableWriter, mode launchMode) {
	if mode == launchHeadless {
		printWarnings(d.StartupWarnings())
		return
	}

	// Auto: TTY → prompt; non-TTY → silent headless.
	if mode == launchAuto {
		if !stdinIsTerminal() {
			printWarnings(d.StartupWarnings())
			return
		}
		printWarnings(d.StartupWarnings())
		chosen, ok := promptLaunchChoice(ctx)
		if !ok {
			return // headless or cancelled
		}
		mode = chosen
	}

	switch mode {
	case launchTUI:
		if err := runInProcessTUI(ctx, d, log, logSwap); err != nil {
			fmt.Fprintf(os.Stderr, "launcher: tui exited with error: %v\n", err)
		}
	case launchWeb:
		if err := openWebUI(d, log); err != nil {
			fmt.Fprintf(os.Stderr, "launcher: %v\n", err)
		}
	}
}

// printWarnings sends the daemon's collected startup warnings to
// stderr (red-ish when supported). The launcher menu and headless
// path both call this so silently-degraded startups always surface.
func printWarnings(ws []string) {
	if len(ws) == 0 {
		return
	}
	const yellow, reset = "\033[33m", "\033[0m"
	color := stdoutIsTerminal()
	fmt.Fprintln(os.Stderr)
	for _, w := range ws {
		if color {
			fmt.Fprintf(os.Stderr, "%s! %s%s\n", yellow, w, reset)
		} else {
			fmt.Fprintf(os.Stderr, "! %s\n", w)
		}
	}
	fmt.Fprintln(os.Stderr)
}

// promptLaunchChoice prints the menu on stderr and reads stdin. The
// read happens in a goroutine so a SIGTERM arriving while we're
// blocked still unwinds the daemon cleanly. Returns ok=false when
// the operator cancels (ctrl-c, ctrl-d, blank line) — caller stays
// headless.
func promptLaunchChoice(ctx context.Context) (launchMode, bool) {
	// ── visual separator so this prompt is clearly distinct from
	// the earlier multi-config picker (which also reads stdin).
	fmt.Fprintln(os.Stderr, "──")
	fmt.Fprintln(os.Stderr, "Daemon ready. How do you want to drive it?")
	fmt.Fprintln(os.Stderr, "  [1] TUI       (in-process operator console)")
	fmt.Fprintln(os.Stderr, "  [2] Web       (open the bundled SPA in your browser)")
	fmt.Fprintln(os.Stderr, "  [3] Headless  (keep running silent)")
	fmt.Fprint(os.Stderr, "Choice [1-3, default 3]: ")

	type lineResult struct {
		line string
		err  error
	}
	resultCh := make(chan lineResult, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		resultCh <- lineResult{line: line, err: err}
	}()

	var line string
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr)
		return launchHeadless, false
	case r := <-resultCh:
		if r.err != nil {
			fmt.Fprintln(os.Stderr)
			return launchHeadless, false
		}
		line = r.line
	}

	line = strings.TrimSpace(line)
	switch line {
	case "", "3", "h", "H", "headless":
		return launchHeadless, false
	case "1", "t", "T", "tui":
		return launchTUI, true
	case "2", "w", "W", "web":
		return launchWeb, true
	}
	fmt.Fprintf(os.Stderr, "launcher: unknown choice %q, staying headless\n", line)
	return launchHeadless, false
}

// runInProcessTUI builds a TUI Model pointed at the local HTTP API
// and runs bubbletea synchronously. Daemon logs are redirected to a
// tempfile while the TUI owns the screen so background log lines
// don't bleed onto the alt-screen canvas; the previous writer is
// restored on TUI exit.
func runInProcessTUI(ctx context.Context, d *Daemon, log *slog.Logger, logSwap *gtlog.SwappableWriter) error {
	setup, err := prepareInProcessTUI(d, logSwap)
	if err != nil {
		return err
	}
	defer setup.cleanup(log)

	// Run bubbletea synchronously. The ctx isn't passed in (bubbletea
	// owns the terminal until it returns) — if the daemon is
	// cancelled mid-TUI the next API poll will fail and the operator
	// can quit with 'q'.
	_, err = setup.prog.Run()
	return err
}

// inProcessTUISetup holds everything prepareInProcessTUI assembled
// so runInProcessTUI can run + tear down without re-doing the work.
// Exposed package-internal for testing — tests construct the setup,
// inspect it, and then call cleanup without ever calling prog.Run().
type inProcessTUISetup struct {
	prog    *tea.Program
	model   tea.Model
	cli     *client.Client
	logFile *os.File
	logSwap *gtlog.SwappableWriter
}

// cleanup restores the swappable log writer + closes the temp log
// file. Logs the captured-log path on the restored writer so the
// operator sees where the daemon's pre-TUI output landed.
func (s *inProcessTUISetup) cleanup(log *slog.Logger) {
	if s.logFile != nil {
		s.logSwap.Restore()
		_ = s.logFile.Close()
		log.Info("tui: daemon log captured", "path", s.logFile.Name())
	}
}

// prepareInProcessTUI does everything runInProcessTUI does except
// the bubbletea Run() call. Factored out so unit tests can verify
// URL resolution, log redirection, and program construction without
// needing a real terminal.
func prepareInProcessTUI(d *Daemon, logSwap *gtlog.SwappableWriter) (*inProcessTUISetup, error) {
	addr := d.HTTPListenAddr()
	if addr == "" {
		return nil, errors.New("in-process TUI requires api.http_addr in config")
	}
	server := normaliseServerURL(addr)

	cli := client.New(server, 5*time.Second, false)

	// Redirect daemon logs so they don't scribble over the TUI.
	logFile, _ := os.CreateTemp("", "gophertrunk-tui-*.log")
	if logFile != nil {
		logSwap.Redirect(logFile)
	}

	m := tui.New(cli, tui.Options{Write: true})
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	return &inProcessTUISetup{
		prog:    prog,
		model:   m,
		cli:     cli,
		logFile: logFile,
		logSwap: logSwap,
	}, nil
}

// openWebUI opens the bundled web SPA pointed at the running
// daemon. Resolution order:
//
//  1. If the daemon binary was linked with embedded web/dist assets
//     (gtweb.HasAssets()), open the daemon URL directly — the
//     server hosts the SPA at "/" so no file:// hop is needed.
//  2. Otherwise search sibling-dir locations for a gophertrunk-web/
//     directory and open its index.html with a #server=<url> hash
//     so the SPA bootstraps against the daemon.
//  3. Fallback: print the URL + asset path to stderr so a remote
//     operator can open them on a machine that has a browser.
//
// Test seams. Production code injects nothing; tests override these
// to drive openWebUI's decision tree without spawning a real browser.
var (
	hasWebAssetsFn   = gtweb.HasAssets
	canOpenBrowserFn = canOpenBrowser
	openBrowserFn    = openBrowser
)

func openWebUI(d *Daemon, log *slog.Logger) error {
	addr := d.HTTPListenAddr()
	if addr == "" {
		return errors.New("-web requires api.http_addr in config")
	}
	serverURL := normaliseServerURL(addr)

	if hasWebAssetsFn() {
		if !canOpenBrowserFn() {
			printWebFallback(serverURL, "(embedded in daemon binary)")
			return nil
		}
		log.Info("launcher: opening embedded SPA", "url", serverURL)
		if err := openBrowserFn(serverURL); err != nil {
			printWebFallback(serverURL, "(embedded in daemon binary)")
		}
		return nil
	}

	assetPath := findWebAssets()
	target := buildWebTargetURL(assetPath, serverURL)
	if assetPath == "" {
		printWebFallback(serverURL, "")
		return nil
	}

	if !canOpenBrowserFn() {
		printWebFallback(serverURL, assetPath)
		return nil
	}
	log.Info("launcher: opening web UI", "target", target)
	if err := openBrowserFn(target); err != nil {
		printWebFallback(serverURL, assetPath)
		return nil
	}
	return nil
}

// findWebAssets walks the conventional install locations and returns
// the first path that contains an index.html. Returns "" when none
// of the candidates exist.
func findWebAssets() string {
	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "gophertrunk-web", "index.html"),
			filepath.Join(dir, "web", "dist", "index.html"),
			filepath.Join(filepath.Dir(dir), "share", "gophertrunk", "web", "index.html"),
		)
	}
	if home, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "gophertrunk", "web", "index.html"))
	}
	// Dev tree: ./web/dist
	candidates = append(candidates, filepath.Join("web", "dist", "index.html"))

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// buildWebTargetURL returns a file:// URL pointing at index.html
// with #server=<daemon URL> so the SPA bootstraps automatically.
func buildWebTargetURL(assetPath, serverURL string) string {
	if assetPath == "" {
		return serverURL
	}
	abs, err := filepath.Abs(assetPath)
	if err != nil {
		abs = assetPath
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	u.Fragment = "server=" + serverURL
	return u.String()
}

// canOpenBrowser probes the environment for a viable display target.
// On Linux/BSD we look for $DISPLAY or $WAYLAND_DISPLAY; on macOS /
// Windows the OS always has a session.
func canOpenBrowser() bool {
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	default:
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
}

// openBrowser shells out to the OS-specific URL opener. Returns the
// error from the underlying command. The 2 s timeout protects
// against an opener that hangs (broken xdg-mime database, etc.).
func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		// The opener typically backgrounds itself; treat a hung
		// foreground as success and let the OS reap the child.
		return nil
	}
}

// printWebFallback writes the human-readable fallback block when we
// couldn't launch a browser (no display, missing assets, etc.).
func printWebFallback(serverURL, assetPath string) {
	fmt.Fprintln(os.Stderr, "──")
	fmt.Fprintln(os.Stderr, "launcher: could not launch a browser on this host.")
	fmt.Fprintln(os.Stderr, "          Open this URL on a machine that has one:")
	fmt.Fprintf(os.Stderr, "            %s\n", serverURL)
	if assetPath != "" {
		fmt.Fprintf(os.Stderr, "          Web SPA assets are at %s\n", assetPath)
		fmt.Fprintln(os.Stderr, "          (open index.html in a browser, then enter the URL above)")
	} else {
		fmt.Fprintln(os.Stderr, "          No bundled web/ directory found; build with `make web-build`")
		fmt.Fprintln(os.Stderr, "          and host web/dist/ from any static server.")
	}
	fmt.Fprintln(os.Stderr)
}

// normaliseServerURL turns an HTTPAddr like ":8080" or
// "0.0.0.0:8080" into a clickable browser URL. Loopback binds become
// http://127.0.0.1:<port>; wildcard binds become http://localhost:<port>
// so the operator on the same host has something that resolves.
func normaliseServerURL(addr string) string {
	host, port := splitHostPort(addr)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "localhost"
	}
	if port == "" {
		port = "8080"
	}
	return "http://" + host + ":" + port
}

// splitHostPort splits "host:port" without using net.SplitHostPort
// (which errors on a bare port like ":8080"). Returns empty strings
// when the input doesn't fit the pattern.
func splitHostPort(addr string) (host, port string) {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
		port = addr[i+1:]
		// Validate port is numeric.
		if _, err := strconv.Atoi(port); err == nil {
			return host, port
		}
	}
	return "", ""
}
