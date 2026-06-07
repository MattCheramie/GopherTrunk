package api

// DocLink points the web Config Builder at a documentation page for a
// config section. URL is opened in a new tab from the section's help
// affordance.
type DocLink struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// docsBase is the published documentation site. Jekyll renders docs/*.md
// to /<name>.html.
const docsBase = "https://gophertrunk.org/"

// configDocs maps each config section (the keys the builder's section nav
// uses) to its documentation link. Sections without a dedicated page point
// at the closest relevant guide.
var configDocs = map[string]DocLink{
	"log":            {Title: "Logging", URL: docsBase + "live-edits.html", Description: "Log level and format."},
	"diagnostics":    {Title: "Diagnostics", URL: docsBase + "hardening.html", Description: "Verbose error reporting."},
	"radioreference": {Title: "RadioReference", URL: docsBase + "hunt.html", Description: "API credentials for browse/import and the hunt duplicate check."},
	"api":            {Title: "API & Web", URL: docsBase + "web.html", Description: "HTTP/gRPC/WebSocket addresses, auth, CORS."},
	"sdr":            {Title: "SDR Hardware", URL: docsBase + "hardware.html", Description: "Sample rate, devices, rtl_tcp and SoapyRemote sources."},
	"trunking":       {Title: "Trunking Systems", URL: docsBase + "hunt.html", Description: "Systems, control channels, protocols, band plans, encryption keys."},
	"storage":        {Title: "Storage", URL: docsBase + "live-edits.html", Description: "Call-log database and control-channel cache."},
	"recordings":     {Title: "Recordings", URL: docsBase + "voice-calibration.html", Description: "Per-call WAV directory, sample rate, equalizer."},
	"metrics":        {Title: "Metrics", URL: docsBase + "web.html", Description: "Prometheus /metrics endpoint."},
	"retention":      {Title: "Retention", URL: docsBase + "live-edits.html", Description: "Data sweep intervals."},
	"scanner":        {Title: "Scanner", URL: docsBase + "hunt.html", Description: "Scan mode, CC hunter, conventional channels."},
	"tone_out":       {Title: "Tone-out", URL: docsBase + "pocsag.html", Description: "Two-tone / single-tone paging detection."},
	"audio":          {Title: "Audio", URL: docsBase + "voice-calibration.html", Description: "Live playback device, sample rate, volume."},
	"broadcast":      {Title: "Broadcast", URL: docsBase + "web.html", Description: "Broadcastify, RdioScanner, OpenMHz, Icecast feeds."},
	"baseband":       {Title: "Baseband", URL: docsBase + "constellation.html", Description: "IQ capture and replay."},
	"paging":         {Title: "Paging", URL: docsBase + "pocsag.html", Description: "POCSAG / FLEX decoders."},
	"aprs":           {Title: "APRS", URL: docsBase + "aprs.html", Description: "APRS / AX.25 AFSK receiver."},
	"ais":            {Title: "AIS", URL: docsBase + "ais.html", Description: "Marine AIS GMSK receiver."},
	"dsc":            {Title: "DSC", URL: docsBase + "dsc.html", Description: "Marine Digital Selective Calling."},
	"mdc1200":        {Title: "MDC1200", URL: docsBase + "mdc1200.html", Description: "Motorola MDC1200 signaling."},
	"adsb":           {Title: "ADS-B", URL: docsBase + "adsb.html", Description: "Aircraft tracking."},
	"m17":            {Title: "M17", URL: docsBase + "m17.html", Description: "M17 digital voice link-setup."},
	"web":            {Title: "Web UI", URL: docsBase + "web.html", Description: "Tab visibility in the operator console."},
	"import":         {Title: "Import", URL: docsBase + "import.html", Description: "Import systems from RadioReference PDF/CSV."},
}
