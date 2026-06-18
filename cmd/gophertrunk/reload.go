package main

import (
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// Reload re-loads the daemon's config from disk and diff-applies the
// fields the daemon knows how to hot-reload. Restart-required fields
// that changed are logged but left untouched in-memory — the operator
// has to bounce the daemon to pick them up. Returns a one-line
// summary suitable for the headless console / events bus.
//
// Used by the SIGHUP handler in main and by the config-builder
// "load this config" action (via ActivateReload) when the operator
// switches the active config file from the web UI.
func (d *Daemon) Reload() (string, error) {
	if d.cfgPath == "" {
		return "", fmt.Errorf("reload: no config file backs this daemon")
	}
	applied, restartRequired, err := d.reloadFrom(d.cfgPath)
	if err != nil {
		return "", err
	}
	summary := fmt.Sprintf("config reloaded: applied=%d restart_required=%d", len(applied), len(restartRequired))
	if len(applied) > 0 {
		summary += " (applied: " + joinComma(applied) + ")"
	}
	if len(restartRequired) > 0 {
		summary += " (restart needed: " + joinComma(restartRequired) + ")"
	}
	return summary, nil
}

// reloadFrom loads the config at path, diff-applies the hot-reloadable
// fields against the in-memory config, and returns the applied vs
// restart-required field keys. When path differs from the daemon's
// current config file the writer + cfgPath are re-pointed so future
// PATCH /api/v1/settings edits and the runtime DTO's config_path track
// the now-active file — this is what lets the web UI "hot-swap" the
// config file the daemon uses. Restart-required fields (SDR, trunking
// systems, recordings, …) still need a full restart to take effect.
func (d *Daemon) reloadFrom(path string) (applied, restartRequired []string, err error) {
	newCfg, err := config.Load(path)
	if err != nil {
		return nil, nil, err
	}

	// Serialise the diff-apply phase against concurrent Cfg() reads
	// (test harnesses + the runtime DTO builder may race with us).
	d.cfgMu.Lock()
	defer d.cfgMu.Unlock()

	// Re-point the writer + path when switching to a different file so
	// the rest of the daemon (settings writer, ConfigPath) follows.
	if path != d.cfgPath {
		w, werr := config.NewWriter(path)
		if werr != nil {
			return nil, nil, fmt.Errorf("reload: open writer for %s: %w", path, werr)
		}
		d.writer = w
		d.cfgPath = path
	}

	app := newDaemonSettingsApplier(d, "")

	// Compare hot-reloadable fields against the in-memory config.
	old := d.cfg

	if newCfg.Log.Level != old.Log.Level {
		if err := app.SetLogLevel(newCfg.Log.Level); err == nil {
			applied = append(applied, "log.level")
		} else {
			restartRequired = append(restartRequired, "log.level")
		}
	}
	if newCfg.Audio.Volume != old.Audio.Volume {
		app.SetAudioVolume(newCfg.Audio.Volume)
		applied = append(applied, "audio.volume")
	}
	if newCfg.Audio.Muted != old.Audio.Muted {
		app.SetAudioMuted(newCfg.Audio.Muted)
		applied = append(applied, "audio.muted")
	}
	if newCfg.Scanner.ScanMode != old.Scanner.ScanMode {
		if err := app.SetScannerScanMode(newCfg.Scanner.ScanMode); err == nil {
			applied = append(applied, "scanner.scan_mode")
		} else {
			restartRequired = append(restartRequired, "scanner.scan_mode")
		}
	}
	if newCfg.Trunking.EncryptedCalls.Mode != old.Trunking.EncryptedCalls.Mode {
		if err := app.SetTrunkingEncryptedMode(newCfg.Trunking.EncryptedCalls.Mode); err == nil {
			applied = append(applied, "trunking.encrypted_calls.mode")
		} else {
			restartRequired = append(restartRequired, "trunking.encrypted_calls.mode")
		}
	}
	if newCfg.Trunking.EncryptedCalls.MetadataFollowMs != old.Trunking.EncryptedCalls.MetadataFollowMs {
		app.SetTrunkingEncryptedMetadataFollowMs(newCfg.Trunking.EncryptedCalls.MetadataFollowMs)
		applied = append(applied, "trunking.encrypted_calls.metadata_follow_ms")
	}

	// Cold fields — diff and flag but don't apply.
	coldDiffs := map[string]bool{
		"log.format":              newCfg.Log.Format != old.Log.Format,
		"api.http_addr":           newCfg.API.HTTPAddr != old.API.HTTPAddr,
		"api.grpc_addr":           newCfg.API.GRPCAddr != old.API.GRPCAddr,
		"api.auth.mode":           newCfg.API.Auth.Mode != old.API.Auth.Mode,
		"audio.enabled":           newCfg.Audio.Enabled != old.Audio.Enabled,
		"audio.device":            newCfg.Audio.Device != old.Audio.Device,
		"audio.sample_rate":       newCfg.Audio.SampleRate != old.Audio.SampleRate,
		"audio.buffer_ms":         newCfg.Audio.BufferMs != old.Audio.BufferMs,
		"recordings.dir":          newCfg.Recordings.Dir != old.Recordings.Dir,
		"recordings.sample_rate":  newCfg.Recordings.SampleRate != old.Recordings.SampleRate,
		"recordings.write_raw":    newCfg.Recordings.WriteRaw != old.Recordings.WriteRaw,
		"sdr.sample_rate":         newCfg.SDR.SampleRate != old.SDR.SampleRate,
		"retention.call_log_days": newCfg.Retention.CallLogDays != old.Retention.CallLogDays,
		"retention.files_days":    newCfg.Retention.FilesDays != old.Retention.FilesDays,
		"retention.interval":      newCfg.Retention.Interval != old.Retention.Interval,
		"storage.path":            newCfg.Storage.Path != old.Storage.Path,
		"storage.cc_cache_file":   newCfg.Storage.CCCacheFile != old.Storage.CCCacheFile,
		"metrics.enabled":         newCfg.Metrics.Enabled != old.Metrics.Enabled,
		"scanner.cc_hunt.enabled": newCfg.Scanner.CCHunt.Enabled != old.Scanner.CCHunt.Enabled,
		// When switching to a different file, the set of trunking systems
		// almost always changes — surface it as restart-required so the
		// operator knows decoding won't repoint without a bounce.
		"trunking.systems": !sameSystemNames(newCfg, old),
	}
	for key, changed := range coldDiffs {
		if changed {
			restartRequired = append(restartRequired, key)
		}
	}

	// Store the new config so future PATCH /api/v1/settings reads
	// see the freshly-loaded values when computing diffs.
	d.cfg = newCfg

	sort.Strings(applied)
	sort.Strings(restartRequired)
	return applied, restartRequired, nil
}

// sameSystemNames reports whether the two configs declare the same set of
// trunking system names, used to flag a system-list change as
// restart-required when hot-swapping config files.
func sameSystemNames(a, b config.Config) bool {
	if len(a.Trunking.Systems) != len(b.Trunking.Systems) {
		return false
	}
	seen := make(map[string]int, len(a.Trunking.Systems))
	for _, s := range a.Trunking.Systems {
		seen[s.Name]++
	}
	for _, s := range b.Trunking.Systems {
		seen[s.Name]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
