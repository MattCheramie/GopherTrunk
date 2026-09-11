package receiver

import "testing"

// TestDMOOptionsMatchTheOnAirPipeline pins the shared DMO receiver knobs to the
// configuration the control pipeline decoded with on the 20 Aug #1003 capture,
// so neither DMO consumer can drift back to a private receiver setup.
func TestDMOOptionsMatchTheOnAirPipeline(t *testing.T) {
	o := DMOOptions(144_000)
	if o.SampleRateHz != 144_000 {
		t.Errorf("SampleRateHz = %v, want 144000", o.SampleRateHz)
	}
	if !o.EnableEqualizer {
		t.Errorf("EnableEqualizer = false; the blind CMA is required for DMO")
	}
	if o.EnableDCBlock {
		t.Errorf("EnableDCBlock = true; the DMO pipeline that decodes on air runs it off")
	}
	if !o.EnableAFC || !o.EnableChannelFilter {
		t.Errorf("AFC=%v channel filter=%v, want both on", o.EnableAFC, o.EnableChannelFilter)
	}
	if o.ClockMode != ClockGardner || o.GardnerGain != 0.005 {
		t.Errorf("clock %v gain %v, want Gardner / 0.005", o.ClockMode, o.GardnerGain)
	}
	if o.DibitSink != nil || o.SoftSink != nil || o.SymbolSink != nil {
		t.Errorf("DMOOptions must not attach sinks; callers own them")
	}
}
