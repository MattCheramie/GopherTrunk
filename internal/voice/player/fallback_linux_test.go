//go:build linux

package player

import (
	"errors"
	"strings"
	"testing"
)

// TestNewALSABackend_FallsBackToIoctlOnDlopenFailure stubs loadALSAFn
// to simulate a missing libasound2.so.2 and asserts the backend
// constructor reaches the direct-ioctl path. The ioctl backend
// itself is allowed to fail (no /dev/snd in CI) — we only care that
// the *fallback was attempted*, which the distinct error wording
// reveals.
func TestNewALSABackend_FallsBackToIoctlOnDlopenFailure(t *testing.T) {
	prev := loadALSAFn
	loadALSAFn = func() error {
		return errors.New("dlopen libasound.so.2: image not found")
	}
	defer func() { loadALSAFn = prev }()

	_, err := newALSABackend(Config{
		Enabled:    true,
		SampleRate: 8000,
		BufferMs:   80,
	})
	// Expect an ioctl-prefixed error if /dev/snd isn't available in
	// the test environment, or no error if it is.
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "ioctl-alsa:") {
		t.Fatalf("error not from ioctl path: %v", err)
	}
}

// TestNewALSABackend_NoFallbackWhenDisabled asserts the
// DisableAutoFallback knob actually disables the fallback.
func TestNewALSABackend_NoFallbackWhenDisabled(t *testing.T) {
	prev := loadALSAFn
	loadALSAFn = func() error {
		return errors.New("dlopen libasound.so.2: image not found")
	}
	defer func() { loadALSAFn = prev }()

	_, err := newALSABackend(Config{
		Enabled:             true,
		SampleRate:          8000,
		DisableAutoFallback: true,
	})
	if err == nil {
		t.Fatal("expected dlopen error to bubble up, got nil")
	}
	if !strings.Contains(err.Error(), "dlopen") {
		t.Fatalf("expected dlopen error, got: %v", err)
	}
}

// TestListDevices_IncludesNullAndDefault makes sure the new entries
// land in the output. Ioctl entries are platform-dependent (no
// /dev/snd in CI sandboxes) so we don't assert on them here.
func TestListDevices_IncludesNullAndDefault(t *testing.T) {
	got := ListDevices()
	wantPrefixes := []string{"default", "null"}
	for _, want := range wantPrefixes {
		var found bool
		for _, d := range got {
			if strings.HasPrefix(d, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %q entry in %v", want, got)
		}
	}
}
