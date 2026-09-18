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

// acqReceiver stands in for a Tier II receiver whose coarse carrier acquirer
// is frozen at offHz, recording the rejections the engine hands it.
type acqReceiver struct {
	resetCountingReceiver
	offHz    float64
	rejected []float64
}

func (r *acqReceiver) CoarseCarrierOffsetHz() float64 { return r.offHz }
func (r *acqReceiver) AGCLevel() float64              { return 1 }
func (r *acqReceiver) MMClockMu() float64             { return 0 }
func (r *acqReceiver) MMClockSPS() float64            { return 10 }
func (r *acqReceiver) RejectCoarseCarrierOffset(hz float64) bool {
	r.rejected = append(r.rejected, hz)
	r.offHz = 0
	return true
}

// runDeafHealAcq drives `synced` windows with a sync at -49 dBFS, calls
// between (to move the fake acquirer), then `deaf` windows at the same power.
func runDeafHealAcq(t *testing.T, rx *acqReceiver, synced int, between func(), deaf int) *engineChannel {
	t.Helper()
	handler, _, _ := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442_387_500})
	ec := &engineChannel{freqHz: 442_387_500, sysName: "ipsc", protoTag: "dmr-tier2", tier2Cnt: cc, receiver: rx}
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}
	level := scaledIQ(4096, -49)
	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base)
	w, pos := 1, 0
	for i := 0; i < synced; i++ {
		d := syncDibits()
		cc.Process(d, pos)
		pos += len(d)
		ec.pwr.Add(level)
		e.maybeLogDiagnostics(base.Add(time.Duration(w) * (iqpower.Window + time.Second)))
		w++
	}
	if between != nil {
		between()
	}
	for i := 0; i < deaf; i++ {
		ec.pwr.Add(level)
		e.maybeLogDiagnostics(base.Add(time.Duration(w) * (iqpower.Window + time.Second)))
		w++
	}
	return ec
}

// TestDeafHealRejectsUnconfirmedCoarseOffset is the 15 Sep IPSC regression:
// the tap decoded at coarse offset 0 (GPSDO X310), the repeater's idle gap
// left a −20.1 kHz neighbour dominating the tap, the acquirer froze on it and
// the tap went deaf until the heal — every gap, 21 heals in 6 minutes. An
// engage the channel never synced under must be REJECTED at the heal so the
// receiver's acquirer cannot take that neighbour again. Fails against the old
// heal, which only reset the receiver (re-arming the acquirer for the same
// neighbour).
func TestDeafHealRejectsUnconfirmedCoarseOffset(t *testing.T) {
	rx := &acqReceiver{}
	ec := runDeafHealAcq(t, rx, 2, func() { rx.offHz = -20_100 }, tier2DeafHealWindows)
	if ec.deafHeals != 1 || rx.resets != 1 {
		t.Fatalf("heals=%d resets=%d, want 1/1", ec.deafHeals, rx.resets)
	}
	if len(rx.rejected) != 1 || rx.rejected[0] != -20_100 {
		t.Fatalf("rejected offsets = %v, want [-20100] (an engage the tap never synced under)", rx.rejected)
	}
}

// TestDeafHealKeepsConfirmedCoarseOffset: the acquirer's whole purpose is a
// mistuned dongle (issue #836). An offset the channel DECODED under for a
// whole window is the wanted signal's; a later genuine deaf stretch (the 12
// Sep latch) must reset the receiver but never reject that offset, or the
// dongle could never be corrected again.
func TestDeafHealKeepsConfirmedCoarseOffset(t *testing.T) {
	rx := &acqReceiver{offHz: 9000}
	ec := runDeafHealAcq(t, rx, 3, nil, tier2DeafHealWindows)
	if ec.deafHeals != 1 || rx.resets != 1 {
		t.Fatalf("heals=%d resets=%d, want 1/1", ec.deafHeals, rx.resets)
	}
	if len(rx.rejected) != 0 {
		t.Fatalf("rejected offsets = %v, want none for an offset the tap decoded under", rx.rejected)
	}
}

// TestDeafHealEngageMidWindowIsNotConfirmedByThatWindow: an engage that lands
// part-way through a window still carrying the tail of a decoding train must
// not be confirmed by that window's syncs — the offset has to hold across a
// synced window to count as decode evidence.
func TestDeafHealEngageMidWindowIsNotConfirmedByThatWindow(t *testing.T) {
	rx := &acqReceiver{}
	handler, _, _ := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442_387_500})
	ec := &engineChannel{freqHz: 442_387_500, sysName: "ipsc", protoTag: "dmr-tier2", tier2Cnt: cc, receiver: rx}
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}
	level := scaledIQ(4096, -49)
	base := time.Unix(1_700_000_000, 0)
	tick := func(w int) { e.maybeLogDiagnostics(base.Add(time.Duration(w) * (iqpower.Window + time.Second))) }
	tick(0)
	d := syncDibits()
	cc.Process(d, 0)
	ec.pwr.Add(level)
	tick(1) // synced at offset 0
	cc.Process(d, len(d))
	rx.offHz = -20_100 // engages during window 2, whose syncs came from the train's tail
	ec.pwr.Add(level)
	tick(2)
	for w := 3; w < 3+tier2DeafHealWindows; w++ {
		ec.pwr.Add(level)
		tick(w)
	}
	if len(rx.rejected) != 1 {
		t.Fatalf("rejected offsets = %v, want the mid-window engage rejected", rx.rejected)
	}
}

// TestDeafHealWarnIsRateLimited: every heal still resets the receiver and
// counts, but a tap that heals every idle gap (15 Sep: 21 in 6 min) WARNs
// once per tier2DeafHealWarnInterval and logs the rest at DEBUG.
func TestDeafHealWarnIsRateLimited(t *testing.T) {
	level := scaledIQ(4096, -51)
	resets, heals, warns := runDeafHeal(t, 2, 2*tier2DeafHealWindows, level, level)
	if resets != 2 || heals != 2 {
		t.Fatalf("resets=%d heals=%d after %d deaf windows, want 2/2", resets, heals, 2*tier2DeafHealWindows)
	}
	if len(warns) != 1 {
		t.Fatalf("deaf-tap WARNs = %d for 2 heals inside the interval, want 1", len(warns))
	}
}

// deafWindow is one diagnostics window for runDeafHealSeq: synced says the
// channel saw a sync in it, dbfs is its mean IQ power.
type deafWindow struct {
	synced bool
	dbfs   float64
}

// runDeafHealSeq drives an arbitrary window sequence through the deaf-tap
// guard on one Tier II channel (runDeafHeal's synced-then-deaf shape is the
// special case) and returns the receiver reset count.
func runDeafHealSeq(t *testing.T, seq []deafWindow) int {
	t.Helper()
	handler, _, _ := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	rx := &resetCountingReceiver{}
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442_387_500})
	ec := &engineChannel{freqHz: 442_387_500, sysName: "ipsc", protoTag: "dmr-tier2", tier2Cnt: cc, receiver: rx}
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}

	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base)
	pos := 0
	for w, win := range seq {
		if win.synced {
			d := syncDibits()
			cc.Process(d, pos)
			pos += len(d)
		}
		ec.pwr.Add(scaledIQ(4096, win.dbfs))
		e.maybeLogDiagnostics(base.Add(time.Duration(w+1) * (iqpower.Window + time.Second)))
	}
	return rx.resets
}

// TestDeafHealIgnoresTrainEdgeWindow is the 17 Sep IPSC regression: an idle
// repeater beacons in ~10 s trains with ~5–9 s gaps, and the ~1 s diagnostics
// window that straddles a train's tail carries a few syncs at a mean power
// that is mostly gap (the field log: decode_dbfs=−63 on a tap decoding at
// −48.4 dBFS). The old guard took that window's power as the channel's
// decoding level, so the following gap windows (−65 dBFS, inside the 6 dB
// margin) read as a deaf tap and reset a healthy receiver on every idle gap —
// six false heals in three minutes. The reference must not follow a single
// edge window down into the gap. Fails against the old engine (reset after
// the gap).
func TestDeafHealIgnoresTrainEdgeWindow(t *testing.T) {
	seq := []deafWindow{
		{true, -48.4}, {true, -48.4}, {true, -48.4}, {true, -48.4}, {true, -48.4},
		{true, -63.0}, // train tail: a few syncs, mostly gap in the mean
	}
	for i := 0; i < 3*tier2DeafHealWindows; i++ {
		seq = append(seq, deafWindow{false, -65.0}) // idle gap
	}
	if resets := runDeafHealSeq(t, seq); resets != 0 {
		t.Fatalf("healthy tap reset %d times across an idle beacon gap after a train-edge window (the 17 Sep false-heal bug), want 0", resets)
	}
}

// TestDeafHealTracksGenuineLevelChange: the bounded reference still follows a
// real, sustained level change, so a tap that keeps decoding at a lower level
// and THEN goes deaf there is healed — the reference decays a step per synced
// window rather than freezing at the historical peak.
func TestDeafHealTracksGenuineLevelChange(t *testing.T) {
	seq := []deafWindow{{true, -48}, {true, -48}, {true, -48}}
	for i := 0; i < 12; i++ {
		seq = append(seq, deafWindow{true, -58}) // sustained 10 dB lower, still decoding
	}
	for i := 0; i < tier2DeafHealWindows; i++ {
		seq = append(seq, deafWindow{false, -58}) // now deaf at that level
	}
	if resets := runDeafHealSeq(t, seq); resets != 1 {
		t.Fatalf("tap deaf at its (lowered) decoding level reset %d times, want 1", resets)
	}
}
