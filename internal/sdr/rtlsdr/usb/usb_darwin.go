//go:build darwin

package usb

import "errors"

// ErrMacOSUnsupported is returned by every macOS USB call until the
// IOKit-via-purego backend lands (tracking issue:
// https://github.com/MattCheramie/GopherTrunk/issues/82). It wraps
// [ErrUnsupportedPlatform] so existing callers that already key off
// that sentinel see no behavior change, but the message points users
// at the specific issue so they can subscribe / contribute.
var ErrMacOSUnsupported = errors.New("usb: macOS IOKit transport not yet implemented; see https://github.com/MattCheramie/GopherTrunk/issues/82 (tracked for PR-10 of the librtlsdr → pure-Go rewrite)")

// platformEnumerator returns a stub [Enumerator] whose every method
// reports an error chained to [ErrUnsupportedPlatform]. macOS keeps
// this stub through PR-09; PR-10 replaces it with the real IOKit
// backend using github.com/ebitengine/purego.
//
// Day-one CGO_ENABLED=0 darwin builds compile and start; only live
// RTL-SDR dongle access is gated. The daemon's mock-IQ replay path,
// TUI, REST/SSE/WebSocket/gRPC APIs, and stored-call decoder all
// work on macOS today.
func platformEnumerator() Enumerator { return darwinEnumerator{} }

type darwinEnumerator struct{}

func (darwinEnumerator) Name() string { return "macos-stub" }

func (darwinEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	return nil, darwinErr()
}

func (darwinEnumerator) Open(Descriptor) (Transport, error) {
	return nil, darwinErr()
}

func darwinErr() error {
	// Return ErrMacOSUnsupported but with ErrUnsupportedPlatform in the
	// chain so errors.Is(err, usb.ErrUnsupportedPlatform) keeps working
	// for any caller that has already adopted that sentinel.
	return errJoin(ErrUnsupportedPlatform, ErrMacOSUnsupported)
}

// errJoin returns an error that satisfies errors.Is for both inputs.
// We don't pull in the Go 1.20 errors.Join helper directly because the
// printed form is two lines; for a stub message we want the one-line
// "macOS not yet implemented; see #82" text up front.
type joinedErr struct {
	primary, secondary error
}

func (j joinedErr) Error() string { return j.secondary.Error() }
func (j joinedErr) Is(target error) bool {
	return target == j.primary || target == j.secondary
}
func (j joinedErr) Unwrap() []error { return []error{j.primary, j.secondary} }

func errJoin(primary, secondary error) error {
	return joinedErr{primary: primary, secondary: secondary}
}
