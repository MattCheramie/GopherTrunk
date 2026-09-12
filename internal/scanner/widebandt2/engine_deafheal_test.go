package widebandt2

import (
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
)

// resetCountingReceiver stands in for the Tier II narrowband receiver: it
// records how often the engine reset it.
type resetCountingReceiver struct{ resets int }

func (r *resetCountingReceiver) Process([]complex64) {}
func (r *resetCountingReceiver) Reset()              { r.resets++ }

// scaledIQ is n samples of a constant tone at the given dBFS.
func scaledIQ(n int, dbfs float64) []complex64 {
	a := float32(math.Pow(10, dbfs/20) / math.Sqrt2)
	c := make([]complex64, n)
	for i := range c {
		c[i] = complex(a, a)
	}
	return c
}

// syncDibits is a dibit stream carrying one BS-Data sync word, enough for the
// Tier II channel to count a sync hit (nothing decodes — the payload is zero).
func syncDibits() []uint8 {
	out := make([]uint8, 300)
	copy(out[100:], dmr.BSData.Dibits[:])
	return out
}

// runDeafHeal drives diagnostics windows on one Tier II channel: `synced`
// windows in which the channel sees a sync at `syncDbFS`, then `deaf` windows
// with no sync at `deafDbFS`. Returns the receiver reset count, the channel's
// heal counter and the WARNs captured.
func runDeafHeal(t *testing.T, synced, deaf int, syncIQ, deafIQ []complex64) (int, uint64, []capturedRecord) {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	rx := &resetCountingReceiver{}
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442_387_500})
	ec := &engineChannel{freqHz: 442_387_500, sysName: "ipsc", protoTag: "dmr-tier2", tier2Cnt: cc, receiver: rx}
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}

	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base)
	w := 1
	pos := 0
	for i := 0; i < synced; i++ {
		d := syncDibits()
		cc.Process(d, pos)
		pos += len(d)
		ec.pwr.Add(syncIQ)
		e.maybeLogDiagnostics(base.Add(time.Duration(w) * (iqpower.Window + time.Second)))
		w++
	}
	for i := 0; i < deaf; i++ {
		ec.pwr.Add(deafIQ)
		e.maybeLogDiagnostics(base.Add(time.Duration(w) * (iqpower.Window + time.Second)))
		w++
	}
	mu.Lock()
	defer mu.Unlock()
	var warns []capturedRecord
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "deaf") {
			warns = append(warns, r)
		}
	}
	return rx.resets, ec.deafHeals, warns
}

// TestDeafTier2TapIsResetAtItsDecodingLevel is the 12 Sep IPSC regression: a
// conventional-DMR tap that synced at -51 dBFS and then sees no sync for three
// windows while its power stays at that level is deaf, not idle — the receiver
// is reset (and the Tier II stream state with it) so the next burst decodes on
// a fresh chain, instead of the tap staying deaf for minutes. Fails against the
// old engine, which never reset a Tier II receiver at all.
func TestDeafTier2TapIsResetAtItsDecodingLevel(t *testing.T) {
	const n = 4096
	level := scaledIQ(n, -51)
	resets, heals, warns := runDeafHeal(t, 2, tier2DeafHealWindows, level, level)
	if resets != 1 || heals != 1 {
		t.Fatalf("receiver resets=%d heals=%d after %d deaf windows, want 1/1", resets, heals, tier2DeafHealWindows)
	}
	if len(warns) != 1 {
		t.Fatalf("deaf-tap WARNs = %d, want 1", len(warns))
	}
	if _, ok := warns[0].attrs["decode_dbfs"]; !ok {
		t.Errorf("WARN missing decode_dbfs attr; attrs=%v", warns[0].attrs)
	}
	// One window short of the threshold: no reset.
	if resets, _, _ := runDeafHeal(t, 2, tier2DeafHealWindows-1, level, level); resets != 0 {
		t.Errorf("reset after only %d deaf windows, want none before %d", tier2DeafHealWindows-1, tier2DeafHealWindows)
	}
}

// TestIdleTier2TapIsNotReset: an unkeyed repeater drops well below the level
// the tap decoded at — that is silence, not deafness, and must never reset the
// receiver (a conventional channel is idle most of the time). A tap that never
// synced at all has no reference level and is left alone too.
func TestIdleTier2TapIsNotReset(t *testing.T) {
	const n = 4096
	if resets, _, _ := runDeafHeal(t, 2, 4*tier2DeafHealWindows, scaledIQ(n, -51), scaledIQ(n, -70)); resets != 0 {
		t.Errorf("idle (carrier dropped) channel reset %d times, want 0", resets)
	}
	if resets, _, _ := runDeafHeal(t, 0, 4*tier2DeafHealWindows, nil, scaledIQ(n, -51)); resets != 0 {
		t.Errorf("never-synced channel reset %d times, want 0", resets)
	}
}
