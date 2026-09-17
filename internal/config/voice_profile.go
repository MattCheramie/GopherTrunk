package config

import "strings"

// VoiceProfile names a reference decoder's audio balance that
// RecordingsConfig.VoiceProfile selects (see that field for the measured
// basis of each).
type VoiceProfile string

const (
	// VoiceProfileMbelib is the default calibration: mbelib / dsd-neo /
	// DSD-FME levels (unvoiced_gain 5.49, radio tilt 450 Hz).
	VoiceProfileMbelib VoiceProfile = "mbelib"
	// VoiceProfileOP25 is the OP25 / trunk-recorder balance.
	VoiceProfileOP25 VoiceProfile = "op25"
)

// OP25 profile values. The unvoiced level is the spec's equal-power reading
// (OP25's imbe_vocoder synthesises unvoiced bands at that level; the 10 Sep
// A/B swept 5.49 → 1 and the log-spectral distance to OP25 fell
// monotonically, 3.57 → 2.69 dB). The enhance-chain values undo the
// dsd-neo-matched low-frequency shaping that measured −5 dB at 100–300 Hz
// against OP25: the first-order radio tilt off (negative disables it) and
// the rumble high-pass moved from 250 Hz to 100 Hz.
const (
	op25UnvoicedGain = 1.0
	op25TiltHz       = -1
	op25HPFHz        = 100
)

// ParseVoiceProfile canonicalises a recordings.voice_profile value. The
// empty string is the default (mbelib). Aliases: "dsd" / "dsd-neo" /
// "dsd-fme" ⇒ mbelib; "trunk-recorder" / "tr" ⇒ op25.
func ParseVoiceProfile(s string) (VoiceProfile, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "mbelib", "dsd", "dsd-neo", "dsd-fme", "default":
		return VoiceProfileMbelib, true
	case "op25", "trunk-recorder", "trunkrecorder", "tr":
		return VoiceProfileOP25, true
	}
	return "", false
}

// ResolvedVoiceCalibration is the recorder-facing result of applying the
// voice profile to the explicit recordings knobs.
type ResolvedVoiceCalibration struct {
	UnvoicedGain float64
	Enhance      EnhanceConfig
}

// ResolveVoiceCalibration applies RecordingsConfig.VoiceProfile to the
// explicit knobs: a preset only fills values still at their zero (default)
// setting, so an operator's explicit unvoiced_gain / enhance.tilt_hz /
// enhance.hpf_hz always wins over the profile. An unknown profile resolves
// like the default (Validate has already rejected it).
func (r RecordingsConfig) ResolveVoiceCalibration() ResolvedVoiceCalibration {
	out := ResolvedVoiceCalibration{UnvoicedGain: r.UnvoicedGain, Enhance: r.Enhance}
	profile, _ := ParseVoiceProfile(r.VoiceProfile)
	if profile != VoiceProfileOP25 {
		return out
	}
	if out.UnvoicedGain == 0 {
		out.UnvoicedGain = op25UnvoicedGain
	}
	// The tilt and high-pass live in the opt-in enhance chain; when it is
	// off there is nothing to undo (the raw decode carries no tilt).
	if out.Enhance.Enabled {
		if out.Enhance.TiltHz == 0 {
			out.Enhance.TiltHz = op25TiltHz
		}
		if out.Enhance.HPFHz == 0 {
			out.Enhance.HPFHz = op25HPFHz
		}
	}
	return out
}
