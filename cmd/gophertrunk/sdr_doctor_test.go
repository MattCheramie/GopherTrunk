package main

import "testing"

// TestKnownDoctorDevicesIncludesHackRF guards that `sdr doctor` inspects
// HackRF VID/PIDs, not just RTL-SDR — otherwise a HackRF that fails to
// open (the error tells the operator to run the doctor) would be invisible
// to the very command that's supposed to diagnose its driver binding.
func TestKnownDoctorDevicesIncludesHackRF(t *testing.T) {
	devs := knownDoctorDevices()
	if len(devs) == 0 {
		t.Fatal("knownDoctorDevices returned nothing")
	}
	var haveHackRF, haveRTL bool
	for _, d := range devs {
		if d.VID == 0x1d50 && d.PID == 0x6089 {
			haveHackRF = true
		}
		if d.VID == 0x0bda {
			haveRTL = true
		}
	}
	if !haveHackRF {
		t.Error("HackRF One (1d50:6089) missing from doctor device list")
	}
	if !haveRTL {
		t.Error("RTL-SDR (vid 0bda) missing from doctor device list")
	}
}
