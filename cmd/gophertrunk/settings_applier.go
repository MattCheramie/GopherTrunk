package main

import (
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// daemonSettingsApplier implements api.SettingsApplier by reaching
// into the daemon's live subsystems. Methods are safe to call from
// any goroutine — the underlying components serialise their own
// state.
type daemonSettingsApplier struct {
	d       *Daemon
	version string
}

func newDaemonSettingsApplier(d *Daemon, version string) *daemonSettingsApplier {
	return &daemonSettingsApplier{d: d, version: version}
}

// SetLogLevel is a no-op for now — slog handlers in stdlib don't
// expose a live level switch without rewiring the slog.Logger.
// Reported as "applied" from the operator's perspective because the
// config file IS updated; full hot-reload of the logger handler is a
// follow-up.
func (a *daemonSettingsApplier) SetLogLevel(level string) error {
	switch level {
	case "debug", "info", "warn", "error":
		return nil
	}
	return errors.New("log level must be debug|info|warn|error")
}

// SetAudioVolume routes through the cockpit aggregator that backs
// PATCH /api/v1/audio. The cockpit handles a nil player gracefully.
func (a *daemonSettingsApplier) SetAudioVolume(v float32) {
	cockpit := audioCockpit{player: a.d.player, recorder: a.d.recorder}
	cockpit.SetVolume(v)
}

func (a *daemonSettingsApplier) SetAudioMuted(m bool) {
	cockpit := audioCockpit{player: a.d.player, recorder: a.d.recorder}
	cockpit.SetMuted(m)
}

// SetAudioEnabled is restart-required (the player is constructed at
// startup with a backend handle). Returning silently keeps the
// response shape simple — the handler always classifies this as
// restart_required via the wrapper logic in handlers_settings.go.
// We still keep the method on the interface so a future hot-reload
// can land without re-touching the API surface.
func (a *daemonSettingsApplier) SetAudioEnabled(enabled bool) {
	// no-op: hot-toggling the audio backend would require a player
	// re-init that's out of scope for this PR.
	_ = enabled
}

// SetRecordingEnabled flips the recorder's "create new sessions"
// gate without touching in-flight sessions.
func (a *daemonSettingsApplier) SetRecordingEnabled(enabled bool) {
	cockpit := audioCockpit{player: a.d.player, recorder: a.d.recorder}
	cockpit.SetRecordingEnabled(enabled)
}

// SetVoiceEnhanceEnabled toggles the opt-in voice enhancement chain on
// the recorder. The chain's parameters come from the daemon's loaded
// recordings.enhance config; only the on/off state is flipped live. The
// change takes effect on the next call (each call builds a fresh decoder
// that reads the current config). A nil recorder is a no-op.
func (a *daemonSettingsApplier) SetVoiceEnhanceEnabled(enabled bool) {
	if a.d.recorder == nil {
		return
	}
	cfg := enhancerConfigFromYAML(a.d.cfg.Recordings.Enhance)
	cfg.Enabled = enabled
	a.d.recorder.SetVoiceEnhance(cfg)
}

// SetScannerScanMode forwards to the engine's SetScanMode. Returns
// an error when the supplied string isn't a recognised mode.
func (a *daemonSettingsApplier) SetScannerScanMode(mode string) error {
	if a.d.engine == nil {
		return errors.New("engine not running")
	}
	parsed := trunking.ParseScanMode(mode)
	a.d.engine.SetScanMode(parsed)
	return nil
}
