package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
)

// countDiagDebugLines drives `windows` diagnostics flushes spaced `step`
// apart and returns how many "channel decode activity" and "channel iq
// power" DEBUG lines were emitted.
func countDiagDebugLines(t *testing.T, ec *engineChannel, windows int, step time.Duration, chanIQ, wbIQ []complex64) (activity, power int) {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}
	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base) // seeds lastDiagAt (no flush)
	for i := 1; i <= windows; i++ {
		ec.pwr.Add(chanIQ)
		e.widebandPwr.Add(wbIQ)
		e.maybeLogDiagnostics(base.Add(time.Duration(i) * step))
	}
	mu.Lock()
	defer mu.Unlock()
	for _, r := range *recs {
		if r.level != slog.LevelDebug {
			continue
		}
		switch {
		case strings.Contains(r.msg, "channel decode activity"):
			activity++
		case r.msg == "widebandt2: channel iq power":
			power++
		}
	}
	return activity, power
}

// TestSteadyChannelDiagnosticsPark pins the DMR IPSC log-spam fix: a channel
// in a steady state (same activity class, stable power) must log its
// per-channel DEBUG diagnostics at parkedLogInterval, not every window. The
// field report was one "channel decode activity" + one "channel iq power"
// line per channel per second, forever. 40 windows spaced 2 s apart span
// 80 s — with parking that's ceil(80/30)+1 = at most 4 of each line, without
// it 40.
func TestSteadyChannelDiagnosticsPark(t *testing.T) {
	const n = 4096
	cc := tier2.New(tier2.Options{SystemName: "IPSC", FrequencyHz: 443_237_500})
	ec := &engineChannel{freqHz: 443_237_500, sysName: "IPSC", protoTag: "dmr-tier2", tier2Cnt: cc}

	activity, power := countDiagDebugLines(t, ec, 40, 2*time.Second, loudIQ(n), loudIQ(n))
	if activity == 0 || power == 0 {
		t.Fatalf("parked diagnostics must still log periodically (activity=%d power=%d)", activity, power)
	}
	if activity > 5 {
		t.Fatalf("steady-state activity DEBUG lines = %d over 80 s, want <= 5 (parked at %v cadence, not per window)", activity, parkedLogInterval)
	}
	if power > 5 {
		t.Fatalf("steady-state power DEBUG lines = %d over 80 s, want <= 5 (parked at %v cadence, not per window)", power, parkedLogInterval)
	}
}
