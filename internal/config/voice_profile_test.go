package config

import "testing"

func TestResolveVoiceCalibrationDefaultIsUntouched(t *testing.T) {
	r := RecordingsConfig{UnvoicedGain: 0, Enhance: EnhanceConfig{Enabled: true}}
	for _, p := range []string{"", "mbelib", "dsd-neo"} {
		r.VoiceProfile = p
		got := r.ResolveVoiceCalibration()
		if got.UnvoicedGain != 0 || got.Enhance.TiltHz != 0 || got.Enhance.HPFHz != 0 {
			t.Errorf("profile %q changed the default knobs: %+v", p, got)
		}
	}
}

func TestResolveVoiceCalibrationOP25(t *testing.T) {
	r := RecordingsConfig{VoiceProfile: "op25", Enhance: EnhanceConfig{Enabled: true}}
	got := r.ResolveVoiceCalibration()
	if got.UnvoicedGain != 1 {
		t.Errorf("op25 unvoiced gain = %v, want 1 (equal-power)", got.UnvoicedGain)
	}
	if got.Enhance.TiltHz >= 0 {
		t.Errorf("op25 tilt = %v, want disabled (<0)", got.Enhance.TiltHz)
	}
	if got.Enhance.HPFHz != 100 {
		t.Errorf("op25 hpf = %v, want 100", got.Enhance.HPFHz)
	}
	// The alias resolves the same way.
	r.VoiceProfile = "trunk-recorder"
	if got2 := r.ResolveVoiceCalibration(); got2 != got {
		t.Errorf("trunk-recorder alias = %+v, want %+v", got2, got)
	}
	// With the enhance chain off there is no tilt to undo — only the vocoder
	// level moves.
	r.Enhance.Enabled = false
	off := r.ResolveVoiceCalibration()
	if off.UnvoicedGain != 1 || off.Enhance.TiltHz != 0 || off.Enhance.HPFHz != 0 {
		t.Errorf("op25 with enhance off = %+v, want only unvoiced gain 1", off)
	}
}

func TestResolveVoiceCalibrationExplicitKnobsWin(t *testing.T) {
	r := RecordingsConfig{VoiceProfile: "op25", UnvoicedGain: 2.5,
		Enhance: EnhanceConfig{Enabled: true, TiltHz: 300, HPFHz: 200}}
	got := r.ResolveVoiceCalibration()
	if got.UnvoicedGain != 2.5 || got.Enhance.TiltHz != 300 || got.Enhance.HPFHz != 200 {
		t.Errorf("explicit knobs overridden by the profile: %+v", got)
	}
}

func TestValidateRejectsUnknownVoiceProfile(t *testing.T) {
	var c Config
	c.Recordings.VoiceProfile = "sdrtrunk"
	if errs := c.validateRecordings(); len(errs) == 0 {
		t.Fatal("unknown voice_profile accepted")
	}
	c.Recordings.VoiceProfile = "OP25"
	if errs := c.validateRecordings(); len(errs) != 0 {
		t.Fatalf("op25 rejected: %v", errs)
	}
}
