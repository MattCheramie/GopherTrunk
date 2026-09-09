package tier3

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestCSBKDebugLogParksRepeats pins the Tier III debug-log parking: an idle
// CC repeating the same C_ALOHA ~16×/s must not produce one debug line per
// burst (field report: 2177 identical lines in two minutes). The first
// occurrence logs immediately, unchanged repeats are summarised at
// csbkRepeatLogInterval with the suppressed count, and any opcode change
// logs immediately again.
func TestCSBKDebugLogParksRepeats(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	go func() {
		for range sub.C {
		}
	}()
	clock := time.Unix(1_700_000_000, 0)
	cc := New(Options{Bus: bus, Log: log, SystemName: "T3", FrequencyHz: 440_262_500,
		Now: func() time.Time { return clock }})

	aloha := CSBK{Opcode: OpAloha}
	// 200 identical Alohas over ~12 s of virtual time (60 ms apart).
	for i := 0; i < 200; i++ {
		cc.handleCSBK(0, aloha)
		clock = clock.Add(60 * time.Millisecond)
	}
	lines := strings.Count(buf.String(), "dmr/tier3: csbk")
	// 12 s span at a 10 s summary cadence: first line + ~1 summary.
	if lines > 3 {
		t.Fatalf("csbk debug lines = %d over 200 identical beacons, want <= 3 (parked)", lines)
	}
	if !strings.Contains(buf.String(), "repeats_suppressed") {
		t.Fatal("expected a parked summary line carrying repeats_suppressed")
	}

	// An opcode change must log immediately.
	buf.Reset()
	cc.handleCSBK(0, CSBK{Opcode: OpAhoy})
	if !strings.Contains(buf.String(), "dmr/tier3: csbk") {
		t.Fatal("opcode change did not log immediately")
	}
}
