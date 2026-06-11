package dmrlcn

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func grantAt(lcn uint8, at time.Time) events.DMRGrantObserved {
	return events.DMRGrantObserved{System: "S", LCN: lcn, GroupID: uint32(lcn) * 10, At: at}
}

func TestCorrelatorSingleMatch(t *testing.T) {
	c := newCorrelator(0, 0, 0)
	base := time.Unix(1_700_000_000, 0)
	c.addGrant(grantAt(5, base))

	// Onset 300 ms after the grant — inside [−100ms, 700ms].
	cand, ok := c.matchOnset(CarrierOnset{FreqHz: 851_125_000, At: base.Add(300 * time.Millisecond)})
	if !ok {
		t.Fatal("expected a match")
	}
	if cand.LCN != 5 || cand.FreqHz != 851_125_000 {
		t.Errorf("candidate = %+v", cand)
	}
	// The grant is consumed, so a second onset finds nothing.
	if _, ok := c.matchOnset(CarrierOnset{FreqHz: 851_125_000, At: base.Add(350 * time.Millisecond)}); ok {
		t.Error("matched grant should have been consumed")
	}
}

func TestCorrelatorOnsetOutsideWindow(t *testing.T) {
	c := newCorrelator(0, 0, 0)
	base := time.Unix(1_700_000_000, 0)
	c.addGrant(grantAt(5, base))

	// Too late (> 700 ms) and too early (before the grant) → no match.
	if _, ok := c.matchOnset(CarrierOnset{FreqHz: 1, At: base.Add(2 * time.Second)}); ok {
		t.Error("onset 2s after grant should not match")
	}
	if _, ok := c.matchOnset(CarrierOnset{FreqHz: 1, At: base.Add(-time.Second)}); ok {
		t.Error("onset before grant should not match")
	}
}

func TestCorrelatorAmbiguousMultiLCNDropped(t *testing.T) {
	c := newCorrelator(0, 0, 0)
	base := time.Unix(1_700_000_000, 0)
	c.addGrant(grantAt(5, base))
	c.addGrant(grantAt(9, base.Add(50*time.Millisecond)))

	// Two different LCNs in the window → ambiguous → no commit.
	if _, ok := c.matchOnset(CarrierOnset{FreqHz: 851_125_000, At: base.Add(300 * time.Millisecond)}); ok {
		t.Error("ambiguous multi-LCN window should not match")
	}
}

func TestCorrelatorSameLCNTwiceIsUnambiguous(t *testing.T) {
	c := newCorrelator(0, 0, 0)
	base := time.Unix(1_700_000_000, 0)
	// Same LCN granted twice near one onset — the LCN is still unambiguous.
	c.addGrant(grantAt(7, base))
	c.addGrant(grantAt(7, base.Add(40*time.Millisecond)))

	cand, ok := c.matchOnset(CarrierOnset{FreqHz: 852_000_000, At: base.Add(200 * time.Millisecond)})
	if !ok || cand.LCN != 7 {
		t.Fatalf("expected unambiguous LCN 7 match, got %+v ok=%v", cand, ok)
	}
}

func TestCorrelatorSpuriousOnsetNoGrant(t *testing.T) {
	c := newCorrelator(0, 0, 0)
	if _, ok := c.matchOnset(CarrierOnset{FreqHz: 851_000_000, At: time.Now()}); ok {
		t.Error("onset with no pending grant should not match")
	}
}

func TestCorrelatorExpire(t *testing.T) {
	c := newCorrelator(0, 0, 500*time.Millisecond)
	base := time.Unix(1_700_000_000, 0)
	c.addGrant(grantAt(1, base))
	c.addGrant(grantAt(2, base.Add(2*time.Second)))
	c.expire(base.Add(2 * time.Second))
	if c.pendingCount() != 1 {
		t.Errorf("pendingCount = %d, want 1 after expiring the stale grant", c.pendingCount())
	}
}
