package ccdecoder

import (
	"strings"
	"testing"
	"time"
)

// pinPipelineClock freezes the pipeline's wall clock at the heartbeat's own
// timestamp so the 5 s CheckStale watchdog never fires during a long feed —
// these tests simulate BSCH liveness by rewinding lastSeenActivity, which
// keeps checkResync fed but does not refresh the REAL heartbeat timestamp,
// and under -race a multi-second feed would otherwise cross the wall-clock
// stale timeout and MarkLost mid-test.
func pinPipelineClock(p *tetraPipeline) {
	p.now = func() time.Time { return time.Unix(0, p.cc.LastActivityNano()) }
}

// feedNoDecodeBSCHAlive drives n samples of undecodable signal while keeping
// the ORDINARY activity heartbeat looking fresh — the 19 Aug field condition:
// SB/BSCH keeps decoding (feeding noteActivity, so checkResync and CheckStale
// never fire) while every SCH payload block fails CRC. The white-box
// lastSeenActivity decrement is the same trick TestTETRAResyncDecodeClearsBudget
// uses to simulate "a decode landed since the last check".
func feedNoDecodeBSCHAlive(p *tetraPipeline, n int64) {
	const chunk = 100_000
	for n > 0 {
		c := n
		if c > chunk {
			c = chunk
		}
		p.lastSeenActivity = p.cc.LastActivityNano() - 1 // pretend a BSCH just decoded
		feedNoDecode(p, c)
		n -= c
	}
}

// payloadBudget returns the payload-drought sample budget for the pipeline.
func payloadBudget(p *tetraPipeline) int64 {
	return int64(tetraPayloadResyncTimeout.Seconds() * p.rateHz)
}

// TestTETRAPayloadDroughtForcesResync is the regression for the 19 Aug "locked
// but deaf" field failure: a locked control channel whose sync bursts keep
// decoding (activity heartbeat alive) but which produces zero CRC-clean SCH
// payload must NOT sit there forever — after a full payload budget of
// processed signal the pipeline forces a resync. Fails against the old code,
// where the surviving BSCH satisfied every watchdog and nothing ever fired
// (the field log sat in this state for 12 minutes at a stretch).
func TestTETRAPayloadDroughtForcesResync(t *testing.T) {
	p, buf, _ := newResyncTestPipeline(t)
	pinPipelineClock(p)
	budget := payloadBudget(p)

	// Just under the budget: nothing may fire (neither the ordinary resync —
	// BSCH is alive — nor the payload one).
	feedNoDecodeBSCHAlive(p, budget-1)
	if strings.Contains(buf.String(), "dsp resync") {
		t.Fatalf("a resync fired before the payload budget was reached\n%s", buf.String())
	}

	// Crossing the budget fires exactly one payload-drought resync.
	feedNoDecodeBSCHAlive(p, 1)
	if got := strings.Count(buf.String(), "tetra: dsp resync (payload drought"); got != 1 {
		t.Fatalf("want exactly one payload-drought resync after a full budget, got %d\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), "signal-time decode drought") {
		t.Fatalf("the ordinary decode-drought resync fired despite a live BSCH heartbeat\n%s", buf.String())
	}
}

// TestTETRAPayloadDroughtEscalatesToLost pins the escalation: when the
// destructive reset alone does not recover payload decode (e.g. the re-primed
// AFC re-latches the same wrong alias on a weak carrier), the third
// consecutive fruitless budget declares the lock lost so the cchunt
// supervisor re-hunts and retunes from cold — bounding the field's 12-minute
// stucks at ~36 s.
func TestTETRAPayloadDroughtEscalatesToLost(t *testing.T) {
	p, buf, _ := newResyncTestPipeline(t)
	pinPipelineClock(p)
	budget := payloadBudget(p)

	// Feed in 100k chunks, so each firing consumes up to one chunk of
	// overshoot past its budget — pad each attempt by a chunk.
	const chunkPad = 100_000
	feedNoDecodeBSCHAlive(p, 2*(budget+chunkPad))
	if got := strings.Count(buf.String(), "tetra: dsp resync (payload drought"); got != 2 {
		t.Fatalf("want two payload-drought resyncs after two budgets, got %d\n%s", got, buf.String())
	}
	if !p.cc.Locked() {
		t.Fatal("lock lost before the escalation threshold")
	}

	feedNoDecodeBSCHAlive(p, budget+chunkPad)
	if !strings.Contains(buf.String(), "payload drought persists across resyncs") {
		t.Fatalf("third fruitless budget did not escalate\n%s", buf.String())
	}
	if p.cc.Locked() {
		t.Fatal("escalation did not declare the lock lost")
	}
}

// TestTETRAPayloadDecodeClearsBudgetAndStreak pins the recovery branch: a
// payload decode (the payload heartbeat advancing past the value the check
// last saw) clears both the accumulated budget and the resync streak.
func TestTETRAPayloadDecodeClearsBudgetAndStreak(t *testing.T) {
	p, _, _ := newResyncTestPipeline(t)
	pinPipelineClock(p)
	budget := payloadBudget(p)

	feedNoDecodeBSCHAlive(p, budget-1)
	if p.samplesSincePayload == 0 {
		t.Fatal("payload budget did not accumulate during the drought")
	}
	p.payloadResyncStreak = 1 // as if one drought reset already fired

	// A payload block decodes: heartbeat now differs from what the check saw.
	p.lastSeenPayload = p.cc.LastPayloadNano() - 1
	p.Process(nil)
	if p.samplesSincePayload != 0 {
		t.Fatalf("a payload decode did not clear the budget: %d", p.samplesSincePayload)
	}
	if p.payloadResyncStreak != 0 {
		t.Fatalf("a payload decode did not clear the resync streak: %d", p.payloadResyncStreak)
	}
}

// TestTETRAPayloadBudgetHeldWhileUnlocked pins the hunt-time guard: an
// unlocked channel accumulates no payload budget (re-acquisition is the hunt
// supervisor's job; piling destructive resets onto a signal-less channel
// helps nothing).
func TestTETRAPayloadBudgetHeldWhileUnlocked(t *testing.T) {
	p, buf, _ := newResyncTestPipeline(t)
	pinPipelineClock(p)
	p.cc.MarkLost()
	buf.Reset()

	feedNoDecodeBSCHAlive(p, payloadBudget(p)+1)
	if p.samplesSincePayload != 0 {
		t.Fatalf("payload budget accumulated while unlocked: %d", p.samplesSincePayload)
	}
	if strings.Contains(buf.String(), "payload drought") {
		t.Fatalf("payload-drought machinery fired while unlocked\n%s", buf.String())
	}
}
