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
// Used by the SIGHUP handler in main and (in the future) by the
// /api/v1/runtime/reload endpoint when that lands.
func (d *Daemon) Reload() (string, error) {
	if d.cfgPath == "" {
		return "", fmt.Errorf("reload: no config file backs this daemon")
	}
	newCfg, err := config.Load(d.cfgPath)
	if err != nil {
		return "", err
	}

	// Serialise the diff-apply phase against concurrent Cfg() reads
	// (test harnesses + the runtime DTO builder may race with us).
	d.cfgMu.Lock()
	defer d.cfgMu.Unlock()

	app := newDaemonSettingsApplier(d, "")
	var applied, restartRequired []string

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

	// Cold fields — diff and flag but don't apply.
	coldDiffs := map[string]bool{
		"log.format":             newCfg.Log.Format != old.Log.Format,
		"api.http_addr":          newCfg.API.HTTPAddr != old.API.HTTPAddr,
		"api.grpc_addr":          newCfg.API.GRPCAddr != old.API.GRPCAddr,
		"api.auth.mode":          newCfg.API.Auth.Mode != old.API.Auth.Mode,
		"audio.enabled":          newCfg.Audio.Enabled != old.Audio.Enabled,
		"audio.device":           newCfg.Audio.Device != old.Audio.Device,
		"audio.sample_rate":      newCfg.Audio.SampleRate != old.Audio.SampleRate,
		"audio.buffer_ms":        newCfg.Audio.BufferMs != old.Audio.BufferMs,
		"recordings.dir":         newCfg.Recordings.Dir != old.Recordings.Dir,
		"recordings.sample_rate": newCfg.Recordings.SampleRate != old.Recordings.SampleRate,
		"recordings.write_raw":   newCfg.Recordings.WriteRaw != old.Recordings.WriteRaw,
		"sdr.sample_rate":        newCfg.SDR.SampleRate != old.SDR.SampleRate,
		"retention.call_log_days": newCfg.Retention.CallLogDays != old.Retention.CallLogDays,
		"retention.files_days":   newCfg.Retention.FilesDays != old.Retention.FilesDays,
		"retention.interval":     newCfg.Retention.Interval != old.Retention.Interval,
		"storage.path":           newCfg.Storage.Path != old.Storage.Path,
		"storage.cc_cache_file":  newCfg.Storage.CCCacheFile != old.Storage.CCCacheFile,
		"metrics.enabled":        newCfg.Metrics.Enabled != old.Metrics.Enabled,
		"scanner.cc_hunt.enabled": newCfg.Scanner.CCHunt.Enabled != old.Scanner.CCHunt.Enabled,
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
	summary := fmt.Sprintf("config reloaded: applied=%d restart_required=%d", len(applied), len(restartRequired))
	if len(applied) > 0 {
		summary += " (applied: " + joinComma(applied) + ")"
	}
	if len(restartRequired) > 0 {
		summary += " (restart needed: " + joinComma(restartRequired) + ")"
	}
	return summary, nil
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
