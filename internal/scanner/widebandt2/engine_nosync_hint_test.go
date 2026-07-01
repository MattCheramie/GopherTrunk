package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
)

// runNoSyncDiag drives `windows` diagnostics windows on a single Tier II
// channel whose typed counters stay at zero (no sync, no FEC — a fresh
// ConventionalChannel that never processed a burst), re-loading `chanIQ`
// worth of power before each flush. It returns every captured "strong
// signal but no sync" WARN.
func runNoSyncDiag(t *testing.T, chanIQ []complex64, windows int) []capturedRecord {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	ec := &engineChannel{
		freqHz:   446_500_000,
		sysName:  "dmr-simplex",
		protoTag: "dmr-tier2",
		tier2Cnt: tier2.New(tier2.Options{Bus: bus, SystemName: "dmr-simplex", FrequencyHz: 446_500_000}),
	}
	e := &Engine{
		log:      slog.New(handler),
		channels: []*engineChannel{ec},
		now:      time.Now,
	}

	base := time.Unix(1_700_000_000, 0)
	// First call seeds lastDiagAt (early return, no channel processing); each
	// subsequent call one Window later flushes exactly one diagnostics window.
	e.maybeLogDiagnostics(base)
	for i := 1; i <= windows; i++ {
		ec.pwr.Add(chanIQ)
		e.maybeLogDiagnostics(base.Add(time.Duration(i) * (iqpower.Window + time.Second)))
	}

	mu.Lock()
	defer mu.Unlock()
	var out []capturedRecord
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "no sync") {
			out = append(out, r)
		}
	}
	return out
}

// TestNoSyncHintFiresOnStrongSignal is the issue #836 diagnostic: a channel
// carrying a strong signal that never syncs (an uncorrected tuner offset)
// gets a WARN pointing the operator at sdr.ppm.
func TestNoSyncHintFiresOnStrongSignal(t *testing.T) {
	const n = 4096
	// loudIQ ≈ -3 dBFS, comfortably above noSyncHintDbFS (-45).
	recs := runNoSyncDiag(t, loudIQ(n), strongNoSyncWindowsNeeded+1)
	if len(recs) == 0 {
		t.Fatal("expected a 'strong signal but no sync' WARN, got none")
	}
	r := recs[0]
	if !strings.Contains(r.msg, "sdr.ppm") {
		t.Errorf("WARN msg = %q, want the sdr.ppm hint", r.msg)
	}
	if f, ok := r.attrs["freq_hz"]; !ok || f == nil {
		t.Errorf("WARN missing freq_hz attr; attrs=%v", r.attrs)
	}
	if db, ok := r.attrs["dbfs"].(float64); !ok || db < noSyncHintDbFS {
		t.Errorf("WARN dbfs attr = %v, want a strong (>= %.0f) level", r.attrs["dbfs"], noSyncHintDbFS)
	}
}

// TestNoSyncHintNeedsSustainedRun: a strong-but-no-sync condition shorter than
// strongNoSyncWindowsNeeded must not fire (guards against brief adjacent-carrier
// splatter tripping the hint).
func TestNoSyncHintNeedsSustainedRun(t *testing.T) {
	const n = 4096
	recs := runNoSyncDiag(t, loudIQ(n), strongNoSyncWindowsNeeded-1)
	if len(recs) != 0 {
		t.Errorf("hint fired after only %d windows, want no WARN before %d; got %d",
			strongNoSyncWindowsNeeded-1, strongNoSyncWindowsNeeded, len(recs))
	}
}

// TestNoSyncHintQuietChannelSilent: a channel below the strong-signal floor is
// a low-power case, not a tuner-offset case, so the no-sync hint stays silent.
func TestNoSyncHintQuietChannelSilent(t *testing.T) {
	const n = 4096
	recs := runNoSyncDiag(t, quietIQ(n), strongNoSyncWindowsNeeded+2)
	if len(recs) != 0 {
		t.Errorf("no-sync hint fired on a quiet channel; got %d WARN(s)", len(recs))
	}
}
