package soapyremote

import "testing"

// TestSetCenterFreqSameFrequencySkipsRetuneAndRecalibrate: re-issuing the
// frequency the device is already on (the cc-hunt supervisor does this when it
// re-locks the same carrier) must be a no-op — no SET_FREQUENCY RPC, and no MRC
// recalibration: the LO never re-locks, so the frozen branch-phase constant is
// still valid. Before this guard, every same-carrier re-hunt reset a perfectly
// good calibration (and, before the anchor was made sticky, flipped it).
func TestSetCenterFreqSameFrequencySkipsRetuneAndRecalibrate(t *testing.T) {
	srv := newFakeSoapyServer(t)
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16", Diversity: "mrc"}}, testLogger())
	devIf, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer devIf.Close()
	dev := devIf.(*device)

	if err := dev.SetCenterFreq(851_000_000); err != nil {
		t.Fatalf("SetCenterFreq(first): %v", err)
	}
	dev.mrc.rearm.Store(false) // consume the legitimate first-tune rearm
	countFreqCalls := func() int {
		n := 0
		for _, c := range srv.recorded() {
			if c.id == callSetFrequency && c.freqHz == 851_000_000 {
				n++
			}
		}
		return n
	}
	first := countFreqCalls()
	if first == 0 {
		t.Fatal("first tune issued no SET_FREQUENCY")
	}

	if err := dev.SetCenterFreq(851_000_000); err != nil {
		t.Fatalf("SetCenterFreq(repeat): %v", err)
	}
	if got := countFreqCalls(); got != first {
		t.Errorf("repeat tune to the same frequency issued %d extra SET_FREQUENCY RPCs, want 0", got-first)
	}
	if dev.mrc.rearm.Load() {
		t.Error("repeat tune to the same frequency re-armed MRC recalibration — the LO did not re-lock")
	}

	// A genuinely different frequency still tunes and re-arms.
	if err := dev.SetCenterFreq(852_000_000); err != nil {
		t.Fatalf("SetCenterFreq(new): %v", err)
	}
	if !dev.mrc.rearm.Load() {
		t.Error("a real retune did not re-arm MRC recalibration")
	}
}
