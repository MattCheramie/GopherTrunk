package main

import (
	"fmt"
	"os"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// The daemon implements api.ConfigActivator (ActivateReload /
// ActivateRestart) and api.ConfigWriter (WritePatch / Path) so the web
// Config Builder can load/hot-swap the active config file. The writer
// methods delegate to the *current* d.writer under the config lock so a
// reload that re-points the file is reflected by later settings edits.

// Path returns the active config file path (api.ConfigWriter). Empty when
// no config file backs the daemon.
func (d *Daemon) Path() string { return d.ConfigPath() }

// WritePatch applies a settings patch to the active config.yaml
// (api.ConfigWriter), delegating to whichever writer is currently
// installed so a hot-swapped file is the one that gets edited.
func (d *Daemon) WritePatch(p config.Patch) (config.Config, error) {
	d.cfgMu.RLock()
	w := d.writer
	d.cfgMu.RUnlock()
	if w == nil {
		return config.Config{}, fmt.Errorf("no config file backs this daemon")
	}
	return w.WritePatch(p)
}

// ActivateReload re-points the daemon at path and reloads it in place,
// hot-applying what it can and returning the applied / restart-required
// field keys (api.ConfigActivator). Restart-required fields (SDR,
// trunking systems, recordings, …) need a full restart to take effect —
// the caller surfaces them so the operator can choose to restart.
func (d *Daemon) ActivateReload(path string) (applied, restartRequired []string, err error) {
	applied, restartRequired, err = d.reloadFrom(path)
	if err != nil {
		return nil, nil, err
	}
	d.log.Info("config activated (reload)", "path", path,
		"applied", applied, "restart_required", restartRequired)
	return applied, restartRequired, nil
}

// ActivateRestart re-execs the daemon into the config at path so every
// field takes effect (api.ConfigActivator). It validates the new config
// loads, records the target, and schedules the run context to unwind
// shortly after returning so the HTTP response flushes first; main then
// re-execs the binary with the new -config once Run has cleanly returned.
func (d *Daemon) ActivateRestart(path string) error {
	if _, err := config.Load(path); err != nil {
		return fmt.Errorf("new config does not load: %w", err)
	}
	d.mu.Lock()
	d.restartTo = path
	cancel := d.cancel
	d.mu.Unlock()

	// Defer the unwind so the handler's 202 reaches the client before the
	// API server is torn down.
	go func() {
		time.Sleep(300 * time.Millisecond)
		if cancel != nil {
			cancel()
		}
	}()
	d.log.Info("config activation requested (restart)", "path", path)
	return nil
}

// RestartPath returns the config path a restart was requested for, or ""
// when no restart is pending. main checks this after Run returns to decide
// whether to re-exec.
func (d *Daemon) RestartPath() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.restartTo
}

// buildRestartArgs rewrites the process argv so the successor loads
// newPath: every existing -config / --config token (in either "-config X"
// or "-config=X" form) is dropped and a single "-config <newPath>" is
// appended. An explicit -config flag beats $GOPHERTRUNK_CONFIG, so editing
// the flag is the only reliable way to force the new file.
func buildRestartArgs(orig []string, newPath string) []string {
	out := make([]string, 0, len(orig)+2)
	skipNext := false
	for i, a := range orig {
		if skipNext {
			skipNext = false
			continue
		}
		if i == 0 {
			out = append(out, a) // argv[0] (program name)
			continue
		}
		if a == "-config" || a == "--config" {
			skipNext = true // drop the following value too
			continue
		}
		if hasConfigEqPrefix(a) {
			continue // "-config=X" / "--config=X"
		}
		out = append(out, a)
	}
	return append(out, "-config", newPath)
}

// hasConfigEqPrefix reports whether arg is a "-config=" / "--config="
// inline-value flag.
func hasConfigEqPrefix(arg string) bool {
	for _, p := range []string{"-config=", "--config="} {
		if len(arg) >= len(p) && arg[:len(p)] == p {
			return true
		}
	}
	return false
}

// performRestart re-execs the daemon into newPath. Called by main after Run
// returns with a pending RestartPath and after the instance lock + SDRs are
// released. Returns only on failure (success replaces the process image on
// Unix / exits after spawning the successor on Windows).
func performRestart(newPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("restart: locate executable: %w", err)
	}
	return execSelf(exe, buildRestartArgs(os.Args, newPath), os.Environ())
}
