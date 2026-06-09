package configbuilder

// SectionMeta names a top-level config section and carries its builder help:
// Key is the snake/lowercase id (nav, docs, validator buckets); CfgField is the
// Go config.Config field name; Label is the display name; Instructions is the
// section help text; DocTitle/DocURL are the "open docs" link. This is the Go
// source of truth that BOTH the web Config Builder (via /api/v1/config/docs)
// and the terminal builder consume, so their section help stays identical.
type SectionMeta struct {
	Key          string
	CfgField     string
	Label        string
	Instructions string
	DocTitle     string
	DocURL       string
}

const docsBase = "https://gophertrunk.org/"

// Sections returns the canonical ordered section list (matching the web
// builder's order) with help + docs.
func Sections() []SectionMeta {
	return []SectionMeta{
		{"trunking", "Trunking", "Trunking",
			"The trunked radio networks to follow. Each system needs a unique name, a protocol, and at least one control-channel frequency. Add systems by hand, by browsing RadioReference.com, or by importing a RadioReference PDF/CSV.",
			"Trunking Systems", docsBase + "hunt.html"},
		{"sdr", "SDR", "SDR",
			"Sample rate (225 kHz–20 MHz) every tuner is programmed to, plus local devices and remote (rtl_tcp / SoapySDR) sources. P25 trunking needs separate control and voice dongles (distinct serials).",
			"SDR Hardware", docsBase + "hardware.html"},
		{"api", "API", "API & Web",
			"HTTP/gRPC listen addresses and access control. allow_mutations lets the web UI write changes (talkgroup edits, settings, this builder's saves).",
			"API & Web", docsBase + "web.html"},
		{"scanner", "Scanner", "Scanner",
			"Scan-list mode for the trunking engine plus the control-channel hunter and the conventional FM scan list.",
			"Scanner", docsBase + "hunt.html"},
		{"audio", "Audio", "Audio",
			"Live playback of decoded voice. Sample rate (4000–48000 Hz) should match recordings.sample_rate. Volume is 0–1.",
			"Audio", docsBase + "voice-calibration.html"},
		{"recordings", "Recordings", "Recordings",
			"Per-call WAV recorder output directory and sample rate (4000–48000 Hz), plus the optional blind equalizer.",
			"Recordings", docsBase + "voice-calibration.html"},
		{"storage", "Storage", "Storage",
			"Where the call-log database and the control-channel cache live.",
			"Storage", docsBase + "live-edits.html"},
		{"retention", "Retention", "Retention",
			"Background sweeper that ages out call-log rows and recorded files. Zero days disables the corresponding sweep.",
			"Retention", docsBase + "live-edits.html"},
		{"metrics", "Metrics", "Metrics",
			"Mount a Prometheus /metrics endpoint on the API HTTP server.",
			"Metrics", docsBase + "web.html"},
		{"log", "Log", "Logging",
			"Controls daemon log verbosity and output format, plus the optional decoded-message log.",
			"Logging", docsBase + "live-edits.html"},
		{"diagnostics", "Diagnostics", "Diagnostics",
			"Verbose errors print full error chains + stack traces and expand API error envelopes (which expose host/dongle info). Enable only on trusted networks.",
			"Diagnostics", docsBase + "hardening.html"},
		{"radioreference", "RadioReference", "RadioReference",
			"Read-only RadioReference.com API credentials. Used by the hunt duplicate check and by this builder's RadioReference browse/import. Can also be supplied via GOPHERTRUNK_RR_KEY / _USER / _PASS env vars instead of storing the secret here.",
			"RadioReference", docsBase + "hunt.html"},
		{"web", "Web", "Web UI",
			"Show or hide navigation tabs in the operator console. Unchecked tabs are hidden from the nav (the route stays reachable by URL).",
			"Web UI", docsBase + "web.html"},
		{"tone_out", "ToneOut", "Tone-out",
			"Two-tone / single-tone paging detection. Supply two tones (A then B) for sequential paging, or one for single-tone supervision. Durations are Go duration strings (e.g. 250ms, 1.5s).",
			"Tone-out", docsBase + "pocsag.html"},
		{"broadcast", "Broadcast", "Broadcast",
			"Stream completed calls out to aggregators (Broadcastify, RdioScanner, OpenMHz) or a live Icecast/ShoutCast mount. A feed with enabled=false is kept but skipped.",
			"Broadcast", docsBase + "web.html"},
		{"baseband", "Baseband", "Baseband",
			"Wideband IQ recording and offline replay. 'Record' taps live tuners to WAV; 'Replay' mounts recorded WAVs as virtual tuners (record at sdr.sample_rate for real-time-correct playback).",
			"Baseband", docsBase + "constellation.html"},
		{"paging", "Paging", "Paging",
			"POCSAG and FLEX pager decoders. The POCSAG / FLEX lists each pin one SDR to a single paging frequency; a Wideband group fans several channels off one SDR via a DDC bank.",
			"Paging", docsBase + "pocsag.html"},
		{"aprs", "APRS", "APRS",
			"APRS / AX.25 Bell-202 AFSK receiver. Each channel pins an SDR to an APRS frequency (144.39 MHz is the North-America primary).",
			"APRS", docsBase + "aprs.html"},
		{"ais", "AIS", "AIS",
			"Marine AIS GMSK receiver. Class-A vessels alternate between 161.975 MHz (87B) and 162.025 MHz (88B); pin one SDR to each to catch both.",
			"AIS", docsBase + "ais.html"},
		{"dsc", "DSC", "DSC",
			"Marine Digital Selective Calling receiver. VHF channel 70 (156.525 MHz) carries distress/urgency/safety alerts and routine call-ups.",
			"DSC", docsBase + "dsc.html"},
		{"mdc1200", "MDC1200", "MDC1200",
			"Motorola MDC1200 FFSK signaling receiver. Target the conventional analog voice channels you monitor — bursts ride at the head/tail of each transmission.",
			"MDC1200", docsBase + "mdc1200.html"},
		{"adsb", "ADSB", "ADS-B",
			"Aircraft tracking. Consume Mode-S frames from a BEAST upstream (dump1090 / readsb, typically host:30005), or pin an SDR to 1090 MHz for the native PPM receiver.",
			"ADS-B", docsBase + "adsb.html"},
		{"m17", "M17", "M17",
			"M17 digital-voice link-layer receiver (decodes link-setup metadata). Simplex calling is commonly 144.975 MHz (2 m) / 433.475 MHz (70 cm).",
			"M17", docsBase + "m17.html"},
		{"lora", "LoRa", "LoRa",
			"Wide-band LoRa decoder. Pin an SDR to an ISM centre frequency (915 MHz US / 868 MHz EU) and split its IQ band into parallel LoRa sub-channels with spreading-factor auto-detection. LoRaWAN frames are MAC-decoded and, with session keys, decrypted.",
			"LoRa", docsBase + "lora.html"},
	}
}

// SectionByKey returns the section metadata for a snake/lowercase key.
func SectionByKey(key string) (SectionMeta, bool) {
	for _, s := range Sections() {
		if s.Key == key {
			return s, true
		}
	}
	return SectionMeta{}, false
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
