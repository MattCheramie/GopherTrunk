package phase2

import "testing"

func TestCarrierWatchdogFiresAfterIdleLimit(t *testing.T) {
	w := NewCarrierWatchdog(2)
	limit := 2 * DibitsPerSuperframe
	total := 0
	for total+180 < limit {
		if w.Observe(180, 0) {
			t.Fatalf("fired early at %d dibits, limit is %d", total, limit)
		}
		total += 180
	}
	if !w.Observe(180*12, 0) {
		t.Fatalf("did not fire after %d idle dibits", limit)
	}
}

func TestCarrierWatchdogRearmsAfterFiring(t *testing.T) {
	w := NewCarrierWatchdog(1)
	if !w.Observe(DibitsPerSuperframe, 0) {
		t.Fatal("did not fire")
	}
	// A reset costs the receiver time to re-seed; it must get a full idle
	// period before being told to reset again.
	if w.Observe(DibitsPerSuperframe/2, 0) {
		t.Error("fired again immediately after re-arming")
	}
}

func TestCarrierWatchdogLockClearsIdle(t *testing.T) {
	w := NewCarrierWatchdog(1)
	w.Observe(DibitsPerSuperframe-1, 0)
	w.Observe(180, 1) // a superframe locked
	if w.Observe(180, 0) {
		t.Error("fired despite a recent lock clearing the idle count")
	}
}

func TestCarrierWatchdogDefaultLimit(t *testing.T) {
	w := NewCarrierWatchdog(0)
	if w.limit != ReacquireIdleSuperframes*DibitsPerSuperframe {
		t.Errorf("limit = %d, want %d", w.limit, ReacquireIdleSuperframes*DibitsPerSuperframe)
	}
}

func TestCarrierWatchdogResetClearsIdle(t *testing.T) {
	w := NewCarrierWatchdog(1)
	w.Observe(DibitsPerSuperframe-1, 0)
	w.Reset()
	if w.Observe(180, 0) {
		t.Error("fired after Reset cleared the idle count")
	}
}
