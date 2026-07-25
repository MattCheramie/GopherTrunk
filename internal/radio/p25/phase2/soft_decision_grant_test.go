package phase2

// Config-plumbing regression for the soft-decision front-end (issue #915):
// the operator-facing p25_phase2_soft_decision toggle reaches the voice
// composer / sigfollow only via the published grant's
// P25Phase2Decode.SoftDecision. These tests pin that the CC stamps the flag
// it was configured with (default off, on after SetSoftDecision) so a wiring
// regression at the ControlChannel layer fails here rather than silently
// dropping the receiver back to the hard slicer on live traffic.

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// grantSoftDecision drives one grant through a CC and returns the
// P25Phase2Decode.SoftDecision the published grant carried.
func grantSoftDecision(t *testing.T, setSoft bool) bool {
	t.Helper()
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	fixed := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	cc := New(Options{
		Bus:         bus,
		SystemName:  "test-sys",
		FrequencyHz: 851_012_500,
		Now:         func() time.Time { return fixed },
	})
	if setSoft {
		cc.SetSoftDecision(true)
	}

	cc.Ingest(grantPDU(0x1234, 0x00ABCD, 0x1, 0x005))

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g, ok := ev.Payload.(trunking.Grant)
			if !ok {
				t.Fatalf("grant payload type = %T", ev.Payload)
			}
			return g.P25Phase2Decode.SoftDecision
		case <-deadline:
			t.Fatal("no grant event")
			return false
		}
	}
}

// TestGrantSoftDecisionDefaultOff: an un-configured CC stamps SoftDecision=false,
// so live traffic keeps the historical hard-slicer path unless opted in.
func TestGrantSoftDecisionDefaultOff(t *testing.T) {
	if grantSoftDecision(t, false) {
		t.Error("default grant SoftDecision = true, want false (hard slicer)")
	}
}

// TestGrantSoftDecisionStamped: SetSoftDecision(true) reaches the grant, which
// is the only channel by which the composer / sigfollow learn to build a
// soft-decision receiver (issue #915).
func TestGrantSoftDecisionStamped(t *testing.T) {
	if !grantSoftDecision(t, true) {
		t.Error("grant SoftDecision = false after SetSoftDecision(true)")
	}
}
