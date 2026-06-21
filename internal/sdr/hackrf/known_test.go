package hackrf

import "testing"

func TestKnownVIDPIDs(t *testing.T) {
	known := KnownVIDPIDs()
	if len(known) == 0 {
		t.Fatal("KnownVIDPIDs returned nothing")
	}
	wantPIDs := map[uint16]bool{pidHackRFOne: false, pidHackRFJawbrk: false, pidHackRFRad1o: false}
	for _, k := range known {
		if k.VID != vidHackRF {
			t.Errorf("VID = %#04x, want %#04x", k.VID, vidHackRF)
		}
		if _, ok := wantPIDs[k.PID]; ok {
			wantPIDs[k.PID] = true
		}
		if k.Name == "" {
			t.Errorf("PID %#04x has empty Name", k.PID)
		}
	}
	for pid, seen := range wantPIDs {
		if !seen {
			t.Errorf("PID %#04x missing from KnownVIDPIDs", pid)
		}
	}
}
