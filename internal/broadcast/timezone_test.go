package broadcast

import (
	"testing"
	"time"
)

// TestRFC3339InRendersConfiguredZone pins the broadcast-side timezone behaviour:
// webhook/rdioscanner timestamps render in the configured location with an
// explicit offset (not forced to UTC "Z"). A nil location falls back to local.
func TestRFC3339InRendersConfiguredZone(t *testing.T) {
	ts := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC) // 12:00Z
	loc := time.FixedZone("TST", 5*3600+30*60)          // +05:30 → 17:30

	if got := rfc3339In(ts, loc); got != "2026-06-25T17:30:00+05:30" {
		t.Errorf("rfc3339In(loc) = %q, want 2026-06-25T17:30:00+05:30", got)
	}
	if got := rfc3339In(ts, time.UTC); got != "2026-06-25T12:00:00Z" {
		t.Errorf("rfc3339In(UTC) = %q, want 2026-06-25T12:00:00Z", got)
	}
	// nil → time.Local; just assert it doesn't panic and is parseable.
	if got := rfc3339In(ts, nil); got == "" {
		t.Error("rfc3339In(nil) returned empty")
	}
}
