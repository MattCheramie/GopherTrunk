package api

import (
	"testing"
	"time"
)

// TestGRPCTimestampsRenderConfiguredZone pins the API-side timezone behaviour:
// string-formatted response timestamps render in the configured location with
// an explicit offset, not forced to UTC. Covers the shared tsFmt helper (also
// used by the active-call converter) and the RID-history converter.
func TestGRPCTimestampsRenderConfiguredZone(t *testing.T) {
	loc := time.FixedZone("TST", 5*3600+30*60) // +05:30
	start := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)

	if got := tsFmt(start, loc); got != "2026-06-25T17:30:00+05:30" {
		t.Errorf("tsFmt(loc) = %q, want 2026-06-25T17:30:00+05:30", got)
	}
	if got := tsFmt(start, time.UTC); got != "2026-06-25T12:00:00Z" {
		t.Errorf("tsFmt(UTC) = %q, want 2026-06-25T12:00:00Z", got)
	}
	if got := tsFmt(start, nil); got == "" { // nil → local, must not panic
		t.Error("tsFmt(nil) returned empty")
	}

	row := callRowToPB(CallRow{ID: 1, StartedAt: start, EndedAt: start.Add(time.Minute)}, loc)
	if row.StartedAt != "2026-06-25T17:30:00+05:30" {
		t.Errorf("callRowToPB StartedAt = %q, want offset-bearing local", row.StartedAt)
	}
}
