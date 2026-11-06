package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// runTSBKHealthDiag drives the diagnostics loop over `windows` power
// windows on one P25 Phase 1 control channel whose TSBK counters advance
// by (dec, fail) per window, and returns every TSBK-health WARN.
func runTSBKHealthDiag(t *testing.T, dec, fail int64, windows int) []capturedRecord {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	bus := events.NewBus(64)
	defer bus.Close()
	var totDec, totFail int64
	ec := &engineChannel{
		freqHz:   451_250_000,
		sysName:  "p25-site",
		protoTag: "p25-phase1",
		decoded:  func() uint64 { return uint64(totDec) },
		tsbkCounts: func() (int64, int64) {
			return totDec, totFail
		},
	}
	e := &Engine{log: slog.New(handler), bus: bus, channels: []*engineChannel{ec}, now: time.Now}

	base := time.Unix(1_700_000_000, 0)
	step := iqpower.Window + time.Second
	e.maybeLogDiagnostics(base)
	for i := 1; i <= windows; i++ {
		totDec += dec
		totFail += fail
		ec.pwr.Add(loudIQ(4096))
		e.maybeLogDiagnostics(base.Add(time.Duration(i) * step))
	}
	mu.Lock()
	defer mu.Unlock()
	var out []capturedRecord
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "TSBKs") {
			out = append(out, r)
		}
	}
	return out
}

// windowsFor returns how many diagnostics windows cover d of stream time.
func windowsFor(d time.Duration) int {
	return int(d/(iqpower.Window+time.Second)) + 2
}

// TestTSBKHealthWarnsOnImpairedControlChannel pins #1242 on the wideband
// engine: a P25 CC failing ~59 % of its TSBKs (the reporter's rate) logged
// nothing at all. It must WARN once, and not repeat within the interval.
func TestTSBKHealthWarnsOnImpairedControlChannel(t *testing.T) {
	recs := runTSBKHealthDiag(t, 41, 59, windowsFor(3*tsbkHealthWindow))
	if len(recs) != 1 {
		t.Fatalf("got %d TSBK-health WARNs over three windows, want exactly 1", len(recs))
	}
	if got := recs[0].attrs["tsbk_err_pct"]; got != int64(59) {
		t.Errorf("tsbk_err_pct = %v, want 59", got)
	}
	if got := recs[0].attrs["freq_hz"]; got != uint64(451_250_000) {
		t.Errorf("freq_hz = %v (%T)", got, got)
	}
}

func TestTSBKHealthSilentOnCleanControlChannel(t *testing.T) {
	if recs := runTSBKHealthDiag(t, 99, 1, windowsFor(3*tsbkHealthWindow)); len(recs) != 0 {
		t.Fatalf("clean CC (1 %% errors) got %d TSBK-health WARNs", len(recs))
	}
}

func TestTSBKHealthNeedsEnoughAttempts(t *testing.T) {
	// 2 attempts per window ≈ 30 per health window: under the minimum.
	if recs := runTSBKHealthDiag(t, 1, 1, windowsFor(3*tsbkHealthWindow)); len(recs) != 0 {
		t.Fatalf("sparse CC got %d TSBK-health WARNs", len(recs))
	}
}
