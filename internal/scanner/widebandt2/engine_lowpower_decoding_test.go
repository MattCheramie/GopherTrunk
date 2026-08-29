package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// runDiagWindows drives `windows` diagnostics flushes on ec (6 s apart, past
// the 5 s WARN throttle) and returns the captured records. Like
// countLowPowerWarns but with a live bus so channels with a decode counter
// can publish their power events.
func runDiagWindows(t *testing.T, ec *engineChannel, windows int, chanIQ, wbIQ []complex64) []capturedRecord {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	go func() {
		for range sub.C {
		}
	}()
	e := &Engine{log: slog.New(handler), bus: bus, channels: []*engineChannel{ec}, now: time.Now}
	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base) // seeds lastDiagAt (no flush)
	for i := 1; i <= windows; i++ {
		ec.pwr.Add(chanIQ)
		e.widebandPwr.Add(wbIQ)
		e.maybeLogDiagnostics(base.Add(time.Duration(i) * 6 * time.Second))
	}
	mu.Lock()
	defer mu.Unlock()
	return append([]capturedRecord(nil), (*recs)...)
}

// TestLowPowerDecodingControlChannelIsSilent pins the Tier III field report:
// a control channel decoding every C_ALOHA beacon at -56 dBFS got the
// "channel iq power very low — carrier likely outside the captured passband"
// WARN every 5 s, against live decode evidence. Absolute dBFS is a
// gain-staging number, not a health verdict — a channel with a positive
// decode delta must never draw the low-power WARN (it gets the DEBUG power
// line instead). Before the fix this counts `windows` WARNs.
func TestLowPowerDecodingControlChannelIsSilent(t *testing.T) {
	const n = 4096
	var lifetime uint64
	ec := &engineChannel{
		freqHz:   440_262_500,
		sysName:  "FireT3",
		protoTag: "dmr-tier3",
		// Each diagnostics window sees fresh decodes, like a CC emitting
		// C_ALOHA every 60 ms.
		decoded: func() uint64 { lifetime += 17; return lifetime },
	}

	recs := runDiagWindows(t, ec, 4, quietIQ(n), loudIQ(n))
	warns, powerDebugs := 0, 0
	for _, r := range recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			warns++
		}
		if r.level == slog.LevelDebug && r.msg == "widebandt2: channel iq power" {
			powerDebugs++
		}
	}
	if warns != 0 {
		t.Fatalf("low-power WARNs = %d, want 0 (a decoding control channel is healthy whatever its absolute dBFS)", warns)
	}
	if powerDebugs == 0 {
		t.Fatalf("expected at least one DEBUG power line for a decoding low-power channel (power stays visible without the WARN)")
	}
}

// TestLowPowerNonDecodingControlChannelStillWarns pins the counterpart: the
// same control channel with a decode counter that never moves keeps the
// repeating WARN — silence with no decodes at the floor is a genuine fault.
func TestLowPowerNonDecodingControlChannelStillWarns(t *testing.T) {
	const n = 4096
	ec := &engineChannel{
		freqHz:   440_262_500,
		sysName:  "FireT3",
		protoTag: "dmr-tier3",
		decoded:  func() uint64 { return 0 },
	}

	recs := runDiagWindows(t, ec, 4, quietIQ(n), loudIQ(n))
	warns := 0
	for _, r := range recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			warns++
		}
	}
	if warns != 4 {
		t.Fatalf("low-power WARNs = %d, want 4 (a never-decoding control channel at the floor keeps warning)", warns)
	}
}
