package configbuilder

// AdvancedFields lists the config fields that the builders intentionally do
// NOT give a bespoke widget: they round-trip through an "Advanced (JSON)"
// editor (web) / a raw-JSON or generic editor (TUI). Keyed by Go struct name.
// This is the single definition mirrored by the web's PROTOCOL_KNOBS
// (Trunking.tsx) + DEVICE_ADVANCED (SDR.tsx) and verified by the schema-drift
// tests, so adding a protocol knob is a one-line, reviewed decision.
var AdvancedFields = map[string][]string{
	"DeviceConfig": {
		"BlogV4", "BlogV4Lite", "IQCorrect", "IQInvert", "DCAvoid", "DCAvoidOffsetHz",
		"NarrowbandFilter", "FPGADCBlock", "RFAmp",
	},
	"SystemConfig": {
		// TETRA
		"TETRAColourCode", "TETRAMCC", "TETRAMNC", "TETRAChannel", "TETRAChannelCoding", "TETRAClockMode",
		"TETRAStatusIntervalSecs",
		// LTR
		"LTRFCSMode", "LTRManchesterMode",
		// P25 Phase 1
		"P25Phase1DemodMode",
		// P25 Phase 2
		"P25Phase2TrellisMode", "P25Phase2RSMode", "P25Phase2InterleaveMode",
		"P25Phase2ScramblerMode", "P25Phase2SoftDecision", "P25Phase2Equalizer", "P25Phase2ClockMode",
		// DMR
		"DMRInterleavedVoice", "DMRColorCode",
		// NXDN
		"NXDNViterbiMode", "NXDNDeviationHz",
		// EDACS
		"EDACSBCHMode",
		// MPT1327
		"MPT1327BCHMode", "MPT1327CWSCTolerance",
		// Motorola
		"MotorolaBCHMode",
		// D-STAR
		"DStarFECMode",
	},
}

// IsAdvanced reports whether structName.field is in the AdvancedFields set.
func IsAdvanced(structName, field string) bool {
	for _, f := range AdvancedFields[structName] {
		if f == field {
			return true
		}
	}
	return false
}
