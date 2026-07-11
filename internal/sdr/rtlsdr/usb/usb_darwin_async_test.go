//go:build darwin

package usb

import "testing"

// TestSyncBulkEnabled pins the GT_USB_SYNC_BULK fallback toggle: unset/empty and
// the usual falsey spellings keep the default asynchronous ReadPipeAsync path;
// any other value forces the legacy synchronous reapers. This is the operator's
// escape hatch if the async path misbehaves on a given macOS host, so its
// parsing must not silently invert.
func TestSyncBulkEnabled(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"no", false},
		{"off", false},
		{"1", true},
		{"true", true},
		{"yes", true},
		{"anything", true},
	}
	for _, c := range cases {
		t.Setenv("GT_USB_SYNC_BULK", c.val)
		if got := syncBulkEnabled(); got != c.want {
			t.Errorf("syncBulkEnabled(%q) = %v, want %v", c.val, got, c.want)
		}
	}
}
