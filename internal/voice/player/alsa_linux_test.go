//go:build linux

package player

import (
	"log/slog"
	"testing"
)

// TestLoadALSA confirms that libasound2.so.2 is reachable on a host
// with the runtime library installed. Skips when the library is
// absent so the test passes on minimal containers (CI image,
// stripped-down Alpine, etc.) — the headless fallback in Player.New
// keeps the daemon usable in that environment.
func TestLoadALSA(t *testing.T) {
	if err := loadALSA(); err != nil {
		t.Skipf("libasound2.so.2 not available on this host: %v", err)
	}
	if sndPCMOpen == nil {
		t.Error("sndPCMOpen function pointer should be bound after loadALSA")
	}
	if sndStrerror == nil {
		t.Error("sndStrerror function pointer should be bound after loadALSA")
	}
}

// TestNewALSABackend_HeadlessFallback verifies that Player.New
// degrades gracefully when the ALSA backend can't open a device.
// On a CI runner without a sound card, snd_pcm_open returns -ENOENT
// or similar and the factory returns an error; Player.New catches
// that and constructs the no-op player. The whole test stays green
// either way (real card present or absent) — the contract is "never
// crash the daemon".
func TestNewALSABackend_HeadlessFallback(t *testing.T) {
	p, err := New(Config{Enabled: true, SampleRate: 8000}, slog.Default())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	// Whether or not BackendEnabled is true depends on the host; the
	// contract is just "Player is non-nil and WritePCM doesn't panic".
	if p == nil {
		t.Fatal("New returned nil player")
	}
	if err := p.WritePCM("test", []int16{0, 1, 2}); err != nil {
		t.Errorf("WritePCM after backend init: %v", err)
	}
}

// TestALSAErrorString verifies the error-message helper returns a
// non-empty string even when the symbol resolver hasn't bound
// snd_strerror. The fallback ("errno=N") is the safety net that
// keeps log lines useful in degraded environments.
func TestALSAErrorString(t *testing.T) {
	// Force the fallback path by clearing the resolved symbol.
	saved := sndStrerror
	sndStrerror = nil
	defer func() { sndStrerror = saved }()

	got := alsaErrorString(-32) // -EPIPE on Linux
	if got == "" {
		t.Error("alsaErrorString returned empty string")
	}
	if got != "errno=-32" {
		t.Errorf("alsaErrorString fallback = %q, want errno=-32", got)
	}
}
