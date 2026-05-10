//go:build darwin

package usb

import (
	"errors"
	"strings"
	"testing"
)

func TestDarwinEnumeratorName(t *testing.T) {
	if got, want := DefaultEnumerator().Name(), "macos-stub"; got != want {
		t.Errorf("backend Name() = %q, want %q", got, want)
	}
}

func TestDarwinListReturnsUnsupported(t *testing.T) {
	_, err := DefaultEnumerator().List(0, 0)
	if err == nil {
		t.Fatal("List returned nil error")
	}
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Errorf("err not in ErrUnsupportedPlatform chain: %v", err)
	}
	if !errors.Is(err, ErrMacOSUnsupported) {
		t.Errorf("err not in ErrMacOSUnsupported chain: %v", err)
	}
}

func TestDarwinOpenReturnsUnsupported(t *testing.T) {
	_, err := DefaultEnumerator().Open(Descriptor{})
	if err == nil {
		t.Fatal("Open returned nil error")
	}
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Errorf("err not in ErrUnsupportedPlatform chain: %v", err)
	}
}

func TestDarwinErrorMessageMentionsTrackingIssue(t *testing.T) {
	// The error users see must point them at the tracking issue so
	// they know it's a known deferral, not a missing-binary bug.
	_, err := DefaultEnumerator().List(0, 0)
	if err == nil {
		t.Fatal("nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "issues/82") {
		t.Errorf("error message %q does not reference tracking issue #82", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "macos") && !strings.Contains(strings.ToLower(msg), "iokit") {
		t.Errorf("error message %q does not mention macOS/IOKit", msg)
	}
}
