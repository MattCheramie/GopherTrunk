package configbuilder

// SectionMeta names a top-level config section: Key is the snake/lowercase id
// used by the nav, docs map, and validator buckets; CfgField is the Go
// config.Config field name; Label is the display name. This is the Go source
// of truth that the TUI's section list and the schema-drift test consume; the
// web's SECTIONS array (web/configbuilder/src/sections/index.tsx) mirrors it.
type SectionMeta struct {
	Key      string
	CfgField string
	Label    string
}

// Sections returns the canonical ordered section list (matching the web
// builder's order).
func Sections() []SectionMeta {
	return []SectionMeta{
		{"trunking", "Trunking", "Trunking"},
		{"sdr", "SDR", "SDR"},
		{"api", "API", "API & Web"},
		{"scanner", "Scanner", "Scanner"},
		{"audio", "Audio", "Audio"},
		{"recordings", "Recordings", "Recordings"},
		{"storage", "Storage", "Storage"},
		{"retention", "Retention", "Retention"},
		{"metrics", "Metrics", "Metrics"},
		{"log", "Log", "Logging"},
		{"diagnostics", "Diagnostics", "Diagnostics"},
		{"radioreference", "RadioReference", "RadioReference"},
		{"web", "Web", "Web UI"},
		{"tone_out", "ToneOut", "Tone-out"},
		{"broadcast", "Broadcast", "Broadcast"},
		{"baseband", "Baseband", "Baseband"},
		{"paging", "Paging", "Paging"},
		{"aprs", "APRS", "APRS"},
		{"ais", "AIS", "AIS"},
		{"dsc", "DSC", "DSC"},
		{"mdc1200", "MDC1200", "MDC1200"},
		{"adsb", "ADSB", "ADS-B"},
		{"m17", "M17", "M17"},
	}
}

// SectionOf extracts the top-level config section key from a validation error
// message (everything before the first '.', ':', '[', or ' ' in the leading
// token). E.g. "trunking.systems[0]: name required" → "trunking".
func SectionOf(msg string) string {
	end := len(msg)
	for i, r := range msg {
		if r == '.' || r == ':' || r == '[' || r == ' ' {
			end = i
			break
		}
	}
	return msg[:end]
}
