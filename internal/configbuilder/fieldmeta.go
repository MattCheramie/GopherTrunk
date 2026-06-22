package configbuilder

// SelectOption is one choice in a select / dropdown field (stored value +
// display label).
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldMeta is the shared, per-field metadata BOTH Config Builders consume:
// the terminal builder (internal/configtui) reads it directly; the web builder
// reads it over GET /api/v1/config/fieldmeta. It is keyed "StructName.FieldName"
// in the registry below.
//
//   - Label    overrides the humanized Go field name (empty ⇒ humanize).
//   - Help     is the comprehensive per-field help shown under the focused row
//     (TUI) / beside the widget (web). Every config leaf has one; the
//     fieldmeta coverage test fails if a new field lands without it, so a
//     schema change can't ship without updating the shared help (and thus
//     both editors).
//   - Options  turns a string/number into a select with these choices.
//   - Hz       renders a uint/int as an MHz/Hz frequency widget.
//   - FreqList renders a []uint32 as a frequency-list widget.
type FieldMeta struct {
	Label    string         `json:"label,omitempty"`
	Help     string         `json:"help,omitempty"`
	Options  []SelectOption `json:"options,omitempty"`
	Hz       bool           `json:"hz,omitempty"`
	FreqList bool           `json:"freqList,omitempty"`
}

func opts(pairs ...string) []SelectOption {
	out := make([]SelectOption, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, SelectOption{Value: pairs[i], Label: pairs[i+1]})
	}
	return out
}

// roleOpts is the control/voice/auto role selector shared by the remote SDR
// sources and baseband replay (no "wideband" — only physical devices fan out).
func roleOpts() []SelectOption {
	return opts("", "(auto)", "control", "control", "voice", "voice", "auto", "auto")
}

// protocolOpts is the trunking protocol selector (mirrors the web SystemForm).
func protocolOpts() []SelectOption {
	ps := []string{"p25", "p25-phase2", "dmr", "dmr-tier2", "dmr-tier1", "nxdn", "dpmr",
		"edacs", "motorola", "ltr", "mpt1327", "tetra", "ysf", "dstar"}
	out := make([]SelectOption, len(ps))
	for i, p := range ps {
		out[i] = SelectOption{Value: p, Label: p}
	}
	return out
}

func baudOpts() []SelectOption {
	return opts("512", "512", "1200", "1200", "2400", "2400")
}

// fieldMetas is the canonical per-field registry, keyed "StructName.FieldName".
// Every exported leaf of config.Config has an entry with non-empty Help; the
// container fields (lists / nested structs / maps) carry help too so the form
// header explains what the operator is about to drill into.
var fieldMetas = map[string]FieldMeta{
	// ---- Logging -----------------------------------------------------------
	"LogConfig.Level":            {Help: "Daemon log verbosity. Default info.", Options: opts("", "(default: info)", "debug", "debug", "info", "info", "warn", "warn", "error", "error")},
	"LogConfig.Format":           {Help: "Log line format: human text or structured JSON. Default text.", Options: opts("", "(default: text)", "text", "text", "json", "json")},
	"LogConfig.MessageLog":       {Help: "Optional human-readable per-event log of trunking activity (grants, lock/loss, affiliations, patches)."},
	"MessageLogConfig.Enabled":   {Help: "Turn the decoded-message log on. Needs a Path to write to."},
	"MessageLogConfig.Path":      {Help: "File the decoded-message log is written to. Empty disables it."},
	"MessageLogConfig.MaxSizeMB": {Help: "Rotate the message log at this size in MB. Default 16."},
	"LogConfig.PowerLog":         {Help: "Optional per-channel IQ-power log, gated on decode activity. By default records only low-power (weak-signal) windows on channels that are actually decoding."},
	"PowerLogConfig.Enabled":     {Help: "Turn the power log on. Needs a Path to write to."},
	"PowerLogConfig.Path":        {Help: "File the power log is written to. Empty disables it."},
	"PowerLogConfig.MaxSizeMB":   {Help: "Rotate the power log at this size in MB. Default 16."},
	"PowerLogConfig.AllWindows":  {Help: "Log every decode-active window instead of only low-power windows. Off by default."},

	// ---- Diagnostics -------------------------------------------------------
	"DiagnosticsConfig.VerboseErrors":    {Help: "Print full error chains + stack traces and expand API error envelopes (exposes host/dongle info). Enable only on trusted networks."},
	"DiagnosticsConfig.MemoryLimitMB":    {Help: "Soft heap limit (MiB) so the GC keeps the resident footprint bounded, guarding against an OS memory-pressure kill (Linux OOM / macOS jetsam). 0 auto-derives ~70% of physical RAM when known, else unbounded. GOMEMLIMIT always wins."},
	"DiagnosticsConfig.HeartbeatSeconds": {Help: "Period (seconds) of the runtime health log: uptime, goroutine count, and heap/sys bytes — turns a silent stop into a timeline. 0 uses the 60 s default; negative disables it."},

	// ---- RadioReference ----------------------------------------------------
	"RadioReferenceConfig.APIKey":   {Label: "API key", Help: "RadioReference.com SOAP app key. Optional — GopherTrunk ships with a built-in app key, so leave this blank unless you want to use your own (or set GOPHERTRUNK_RR_KEY)."},
	"RadioReferenceConfig.Username": {Help: "Your RadioReference.com account username. This and the password are what browse/import and 'Verify subscription' use to confirm your premium membership. Can come from GOPHERTRUNK_RR_USER instead."},
	"RadioReferenceConfig.Password": {Help: "Your RadioReference.com account password (required with username for premium browse/import/verify). Can come from GOPHERTRUNK_RR_PASS instead — prefer the env var to keeping the secret in config.yaml."},

	// ---- SDR ---------------------------------------------------------------
	"SDRConfig.SampleRate":         {Label: "Sample rate", Hz: true, Help: "IQ rate every tuner is programmed to (225 kHz–20 MHz). RTL dongles cap ~3.2 MHz; higher needs a wideband soapy_remote source. Primary CPU-load lever."},
	"SDRConfig.Devices":            {Help: "Locally-attached USB SDR dongles, selected by serial."},
	"SDRConfig.RTLTCP":             {Label: "rtl_tcp sources", Help: "Remote rtl_tcp endpoints mounted as virtual tuners. Plaintext — trusted networks / tunnels only."},
	"SDRConfig.SoapyRemote":        {Label: "SoapyRemote sources", Help: "Remote SoapySDRServer endpoints (USRP, Lime, bladeRF, HackRF, Airspy, RTL) mounted as virtual tuners. Plaintext — trusted networks / tunnels only."},
	"SDRConfig.WatchdogIntervalMs": {Help: "USB-disconnect watchdog poll interval (ms). 0 = default 30 s; negative disables it."},
	"SDRConfig.Autotune":           {Label: "Autotune", Help: "Track each dongle's carrier-frequency error on locked P25 Phase 1 control + voice and apply a digital correction so the demod's AFC starts near lock. Never rewrites hardware ppm — logs a suggested value. Off by default; safe to leave on."},

	"DeviceConfig.Serial":        {Help: "USB serial that selects this dongle. Run `gophertrunk sdr list` to see attached serials."},
	"DeviceConfig.Role":          {Help: "Pool role hint. control/voice pin a P25-trunking dongle; wideband fans several channels off one stick; auto lets the pool decide.", Options: opts("", "(auto)", "control", "control", "voice", "voice", "auto", "auto", "wideband", "wideband")},
	"DeviceConfig.PPM":           {Label: "PPM", Help: "Frequency-correction in parts-per-million. 0 suits any TCXO dongle."},
	"DeviceConfig.Gain":          {Help: "Tuner gain. 'auto' (or empty) is AGC; otherwise tenths of a dB (e.g. 496 → 49.6 dB). `gophertrunk sdr list` shows supported steps."},
	"DeviceConfig.BiasTee":       {Label: "Bias-tee", Help: "Enable the dongle's 5 V bias-tee to power an external LNA through the antenna SMA."},
	"DeviceConfig.BlogV4":        {Label: "Blog V4 mode", Help: "Force RTL-SDR Blog V4 mode (28.8 MHz ref + per-band routing) when EEPROM strings are blank so auto-detect misses it."},
	"DeviceConfig.BlogV4Lite":    {Label: "Blog V4 Lite", Help: "Select the two-band Blog V4 'Lite' variant. Set only on a V4L."},
	"DeviceConfig.CenterFreqHz":  {Label: "Center frequency", Hz: true, Help: "Centre of the IQ band a wideband dongle covers. Every channel must fall within ± sample_rate/2 (5 % guard). Wideband only."},
	"DeviceConfig.TunerStrategy": {Help: "DSP layout that extracts each narrow channel from the wide IQ. auto picks by channel count. Wideband only.", Options: opts("", "(auto)", "ddc", "ddc", "polyphase", "polyphase")},
	"DeviceConfig.Channels":      {Help: "Repeater carriers a wideband dongle monitors; each binds a frequency to a trunking system name."},
	"DeviceConfig.VoiceTaps":     {Help: "Per-grant DDC voice tuners carved from a wideband dongle so one SDR covers CC + voice. 0 = use the physical voice pool. Capped at 8."},
	"DeviceConfig.IQCorrect":     {Label: "I/Q correct", Help: "Blind I/Q-imbalance correction before decimation. Validate with `gophertrunk replay -iq-correct -diag` before enabling."},
	"DeviceConfig.IQInvert":      {Label: "I/Q invert", Help: "Conjugate raw IQ (negate Q) to undo a spectrum-inverted front end (needed by some Soapy/USRP chains; critical for π/4-DQPSK like TETRA)."},

	"DeviceConfig.DCAvoid":         {Label: "DC-spike avoid", Help: "Tune a control SDR's LO off-channel and mix the channel back to baseband, off the front-end DC spur and its I/Q-image (the offset tuning SDRTrunk/OP25 use). Helps marginal sites; off by default (issue #402)."},
	"DeviceConfig.DCAvoidOffsetHz": {Label: "DC-avoid offset", Hz: true, Help: "LO offset in Hz when DC-spike avoid is on. 0 = auto (sample_rate/4). Must be < sample_rate/2."},

	"DeviceChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "Repeater carrier frequency. Must lie inside the dongle's IQ band."},
	"DeviceChannelConfig.System":      {Help: "Name of the trunking.systems entry this carrier belongs to."},

	"RTLTCPConfig.Addr":             {Label: "Address", Help: "host:port the rtl_tcp server listens on, e.g. 192.168.1.50:1234. Required."},
	"RTLTCPConfig.Serial":           {Help: "Virtual device serial reported by the pool. Empty derives one from the address."},
	"RTLTCPConfig.Role":             {Help: "Pool role hint: control / voice / auto.", Options: roleOpts()},
	"RTLTCPConfig.PPM":              {Label: "PPM", Help: "Frequency-correction sent to the remote on open. 0 suits any TCXO dongle."},
	"RTLTCPConfig.Gain":             {Help: "Tuner gain: 'auto'/empty for AGC, else tenths of a dB."},
	"RTLTCPConfig.BiasTee":          {Label: "Bias-tee", Help: "Toggle the remote dongle's 5 V bias-tee (librtlsdr ≥ 0.7)."},
	"RTLTCPConfig.ConnectTimeoutMs": {Help: "TCP dial timeout in ms. 0 = driver default (3000)."},

	"SoapyRemoteConfig.Addr":             {Label: "Address", Help: "SoapySDRServer host:port, e.g. 192.168.1.60:55132 (bare host gets :55132). Required."},
	"SoapyRemoteConfig.Driver":           {Help: "SoapySDR device key on the server (uhd, lime, bladerf, hackrf, airspy, rtlsdr). Empty picks the server's first device."},
	"SoapyRemoteConfig.Args":             {Help: "Extra SoapySDR kwargs as key=value,key2=value2 (server-side device selection)."},
	"SoapyRemoteConfig.MasterClockHz":    {Label: "Master clock", Hz: true, Help: "USRP master/reference clock in Hz. A USRP only streams integer decimations of it, so set it to make sample_rate an exact divisor (e.g. 61.44 MHz on a B210 → 6.144 MS/s). 0 = device default. Shorthand for master_clock_rate in args."},
	"SoapyRemoteConfig.Serial":           {Help: "Virtual device serial reported by the pool. Empty derives one from the address."},
	"SoapyRemoteConfig.Role":             {Help: "Pool role hint: control / voice / auto.", Options: roleOpts()},
	"SoapyRemoteConfig.Format":           {Help: "Wire sample format: CS16 (16-bit, default) or CF32 (32-bit float).", Options: opts("", "(default)", "CS16", "CS16", "CF32", "CF32")},
	"SoapyRemoteConfig.StreamProtocol":   {Help: "Stream transport. Only tcp is implemented.", Options: opts("", "(default)", "tcp", "tcp")},
	"SoapyRemoteConfig.StreamMTU":        {Label: "Stream MTU", Help: "Stream endpoint MTU in bytes (sent as the remote:mtu stream arg). 0 = SoapyRemote default 1500. Raise for jumbo-frame / high-throughput links. This is a stream arg, so it cannot go in Args."},
	"SoapyRemoteConfig.StreamWindow":     {Label: "Stream window", Help: "Stream flow-control window in bytes (sent as the remote:window stream arg). 0 = client default 8 MiB. Raise on high-latency / high-bandwidth links to keep the pipe full. Must be >= stream MTU. This is a stream arg, so it cannot go in Args."},
	"SoapyRemoteConfig.PPM":              {Label: "PPM", Help: "Frequency-correction applied on open (best-effort)."},
	"SoapyRemoteConfig.Gain":             {Help: "Tuner gain: 'auto'/empty for AGC, else tenths of a dB."},
	"SoapyRemoteConfig.BiasTee":          {Label: "Bias-tee", Help: "Toggle the remote device's bias-tee (best-effort)."},
	"SoapyRemoteConfig.ConnectTimeoutMs": {Help: "TCP dial timeout in ms. 0 = driver default (3000)."},

	// ---- Trunking ----------------------------------------------------------
	"TrunkingConfig.Systems":           {Help: "Trunked radio networks to follow. Add by hand, by RadioReference browse, or by PDF/CSV import."},
	"TrunkingConfig.CallTimeoutMs":     {Help: "Inactivity window before the watchdog ends a call and frees its voice SDR. 0 = 30 s."},
	"TrunkingConfig.VoiceHangtimeMs":   {Help: "End-of-transmission window for every voice protocol — ends a call this long after the last voice frame. 0 = 3.5 s."},
	"TrunkingConfig.VoiceCallGrouping": {Help: "How recordings split: transmission (one file per over) or conversation (group same-TG overs). Empty = transmission.", Options: opts("", "(default: transmission)", "transmission", "transmission", "conversation", "conversation")},

	"EncryptedCallsConfig.Mode":             {Help: "Encrypted-call handling. follow (default) holds a voice SDR for the whole call; metadata follows briefly to capture talker alias / RID then releases (as soon as the alias completes, or after the follow window); ignore never ties up a voice SDR. Calls you have a matching encryption key for are always followed.", Options: opts("", "(default: follow)", "follow", "follow", "metadata", "metadata", "ignore", "ignore")},
	"EncryptedCallsConfig.MetadataFollowMs": {Help: "Metadata mode only: how long (ms) to follow an encrypted call after it is known encrypted before releasing the voice SDR. 0 = engine default (1500 ms)."},

	"SystemConfig.Name":            {Help: "Unique label for this system. Used in the UI and to name its talkgroup sidecar."},
	"SystemConfig.Protocol":        {Help: "Trunking protocol this system uses.", Options: protocolOpts()},
	"SystemConfig.ControlChannels": {Label: "Control channels", FreqList: true, Help: "One or more control-channel frequencies. The hunter tries each until one locks."},
	"SystemConfig.TalkgroupFile":   {Help: "Path to this system's talkgroup CSV sidecar (edit it with [t] / the talkgroup editor)."},
	"SystemConfig.RIDAliasFile":    {Label: "RID alias file", Help: "Optional CSV/JSON catalogue of radio-ID (subscriber) aliases for this system."},
	"SystemConfig.P25BandPlan":     {Label: "P25 band plan", Help: "Static IDEN_UP slot seeds for P25 Phase 1 sites that route grants through an unannounced channel ID. P25 Phase 1 only."},
	"SystemConfig.DMRBandPlan":     {Label: "DMR band plan", Help: "LCN→frequency map REQUIRED for DMR Tier III voice grants. Pick linear (grid) or table (explicit list). DMR only."},
	"SystemConfig.EncryptionKeys":  {Help: "Operator-held decryption keys (today DMR RC4 'Enhanced Privacy'). No key recovery is performed."},
	"SystemConfig.EncryptedCalls":  {Help: "How scarce voice SDRs are allocated to encrypted calls on this system, so encrypted traffic can't starve clear calls (issue #711)."},
	// SystemConfig advanced protocol knobs (edited under the Advanced group).
	"SystemConfig.TETRAColourCode":         {Label: "TETRA colour code", Help: "30-bit extended colour code seeding the TETRA descrambler. Set to the cell's colour code. TETRA only."},
	"SystemConfig.TETRAChannel":            {Label: "TETRA channel", Help: "Which TETRA logical channel each burst carries (sch/hd, sch/f, sch/hu, bsch, aach). Empty = sch/hd. TETRA only."},
	"SystemConfig.TETRAChannelCoding":      {Label: "TETRA channel coding", Help: "Gate the full ETSI channel-coding chain. on (default, live captures) or off (raw-dibit fixtures). TETRA only."},
	"SystemConfig.TETRAClockMode":          {Label: "TETRA clock mode", Help: "Symbol-timing recovery: gardner (default, live SDR) or naive (sample-aligned fixtures). TETRA only."},
	"SystemConfig.LTRFCSMode":              {Label: "LTR FCS mode", Help: "CRC-7 FCS check on LTR Status words. on (default) or off (un-populated fixtures). LTR only."},
	"SystemConfig.LTRManchesterMode":       {Label: "LTR Manchester mode", Help: "LTR sub-audible decode: soft (default), strict, or off/nrz. LTR only."},
	"SystemConfig.P25Phase1DemodMode":      {Label: "P25 Ph1 demod mode", Help: "Phase 1 symbol recovery: c4fm/fm (default) or cqpsk/lsm (simulcast LSM sites). P25 Phase 1 only."},
	"SystemConfig.P25Phase2TrellisMode":    {Label: "P25 Ph2 trellis", Help: "½-rate trellis FEC on the Phase 2 MAC PDU. on (default) or off (pre-stripped fixtures). P25 Phase 2 only."},
	"SystemConfig.P25Phase2RSMode":         {Label: "P25 Ph2 RS", Help: "Outer Reed-Solomon RS(24,16,9) verification. off (default) or on. P25 Phase 2 only."},
	"SystemConfig.P25Phase2InterleaveMode": {Label: "P25 Ph2 interleave", Help: "Per-burst block deinterleave before trellis decode. off (default) or on. P25 Phase 2 only."},
	"SystemConfig.P25Phase2ScramblerMode":  {Label: "P25 Ph2 scrambler", Help: "PN44 descrambling of the MAC PDU. on (default, all on-air) or off (unscrambled fixtures). P25 Phase 2 only."},
	"SystemConfig.P25Phase2ClockMode":      {Label: "P25 Ph2 clock", Help: "Symbol-timing recovery: gardner (default, live SDR) or naive (fixtures). P25 Phase 2 only."},
	"SystemConfig.DMRInterleavedVoice":     {Label: "DMR interleaved voice", Help: "Override the 2-slot interleaved DMR voice decoder. Unset = on for DMR Tier III (single-slot otherwise); set true/false to force. DMR only."},
	"SystemConfig.NXDNViterbiMode":         {Label: "NXDN Viterbi mode", Help: "K=5 Viterbi FEC on the NXDN CAC. spec (default), on (older fixtures), or off. NXDN only."},
	"SystemConfig.NXDNDeviationHz":         {Label: "NXDN deviation", Help: "Peak FM deviation (Hz) the NXDN slicer is calibrated to. 0 = spec 1800 Hz; try 2400 for bimodal captures. NXDN only."},
	"SystemConfig.EDACSBCHMode":            {Label: "EDACS BCH mode", Help: "BCH(40,28,2) FEC on the EDACS CCW. on (default) or off (pre-stripped fixtures). EDACS only."},
	"SystemConfig.MPT1327BCHMode":          {Label: "MPT1327 BCH mode", Help: "BCH(63,38) FEC on the MPT1327 codeword. on (default) or off (pre-stripped fixtures). MPT1327 only."},
	"SystemConfig.MPT1327CWSCTolerance":    {Label: "MPT1327 CWSC tolerance", Help: "Hamming-distance threshold for the MPT1327 sync code. Empty = 2; 0/exact, or 0–15. MPT1327 only."},
	"SystemConfig.MotorolaBCHMode":         {Label: "Motorola BCH mode", Help: "BCH(64,16,11) FEC on the Motorola Type II OSW. on (default) or off (pre-stripped fixtures). Motorola only."},
	"SystemConfig.DStarFECMode":            {Label: "D-STAR FEC mode", Help: "JARL DV-header FEC chain. off (default) or on. D-STAR only."},

	"P25BandPlanEntryConfig.ChannelID":   {Help: "4-bit IDEN_UP slot index (0–15)."},
	"P25BandPlanEntryConfig.BaseHz":      {Label: "Base frequency", Hz: true, Help: "IDEN_UP base frequency for this slot."},
	"P25BandPlanEntryConfig.SpacingHz":   {Label: "Spacing", Hz: true, Help: "Channel spacing for this slot."},
	"P25BandPlanEntryConfig.TxOffsetHz":  {Label: "TX offset (Hz)", Help: "Transmit offset per the on-air IDEN_UP field (signed Hz)."},
	"P25BandPlanEntryConfig.BandwidthHz": {Label: "Bandwidth", Hz: true, Help: "Channel bandwidth (informational; not used to resolve frequencies)."},

	"DMRBandPlanConfig.Linear":           {Help: "Regular base+spacing grid: freq = base + (lcn − offset) × spacing."},
	"DMRBandPlanConfig.Table":            {Help: "Explicit LCN→frequency list for sites that don't fall on a regular grid."},
	"DMRLinearBandPlanConfig.BaseHz":     {Label: "Base frequency", Hz: true, Help: "Downlink frequency of the first LCN."},
	"DMRLinearBandPlanConfig.SpacingHz":  {Label: "Spacing", Hz: true, Help: "Frequency step between consecutive LCNs."},
	"DMRLinearBandPlanConfig.Offset":     {Help: "LCN the grid starts numbering from. Set 1 for sites that number LCNs from 1."},
	"DMRBandPlanTableEntryConfig.LCN":    {Label: "LCN", Help: "12-bit Logical (Physical) Channel Number from the grant CSBK."},
	"DMRBandPlanTableEntryConfig.FreqHz": {Label: "Frequency", Hz: true, Help: "Downlink frequency this LCN maps to."},

	"EncryptionKeyConfig.KeyID":     {Label: "Key ID", Help: "Key identifier the radios carry in the privacy header, so a system that rotates keys still resolves."},
	"EncryptionKeyConfig.Algorithm": {Help: "Decryption algorithm. Today only DMR RC4 (Enhanced Privacy).", Options: opts("rc4", "rc4 (DMR Enhanced Privacy)")},
	"EncryptionKeyConfig.Key":       {Help: "Raw key, hex-encoded. Spaces and a 0x prefix are tolerated."},

	"SystemConfig.Sites": {Label: "Sites", Help: "Optional human-readable names for the P25 sites of this system, keyed by RFSS/Site. Presentation metadata only — surfaced via GET /api/v1/sites. P25 only."},
	"SiteConfig.RFSS":    {Label: "RFSS ID", Help: "RF Sub-System ID of the site (from the control channel's RFSS Status Broadcast)."},
	"SiteConfig.Site":    {Label: "Site ID", Help: "Site ID within the RFSS (from the control channel's RFSS Status Broadcast)."},
	"SiteConfig.Name":    {Label: "Name", Help: "Human-readable site name shown in GET /api/v1/sites, e.g. \"Mt Anakie\"."},

	// ---- API & Web ---------------------------------------------------------
	"APIConfig.HTTPAddr":            {Label: "HTTP address", Help: "TCP listen address for REST/SSE/WebSocket, e.g. 127.0.0.1:8080. Empty disables it."},
	"APIConfig.GRPCAddr":            {Label: "gRPC address", Help: "TCP listen address for the gRPC server. Empty disables it."},
	"APIConfig.AllowMutations":      {Help: "Legacy wide-open gate. true logs a deprecation warning and maps to auth.mode: disabled. Prefer auth.mode."},
	"APIConfig.Auth":                {Help: "Bearer-token policy gating the write endpoints."},
	"APIConfig.Rigctld":             {Label: "rigctld address", Help: "Expose the control SDR's tuning over the Hamlib rigctld protocol, e.g. 127.0.0.1:4532. No auth — bind loopback."},
	"APIConfig.CORS":                {Label: "CORS", Help: "Cross-origin browser access. Off by default; enable to serve the web UI from a different origin."},
	"APIConfig.TLSCert":             {Label: "TLS cert", Help: "PEM certificate path. With TLSKey, switches HTTP+gRPC to TLS (restart to rotate)."},
	"APIConfig.TLSKey":              {Label: "TLS key", Help: "PEM private-key path. Set together with TLSCert to enable TLS."},
	"APICORSConfig.AllowedOrigins":  {Help: "Exact origins echoed in Access-Control-Allow-Origin. Use 'null' for file://, '*' for any (don't combine with credentials)."},
	"APIAuthConfig.Mode":            {Help: "Auth policy: auto (token off loopback, on otherwise), required, or disabled.", Options: opts("", "(auto)", "auto", "auto", "required", "required", "disabled", "disabled")},
	"APIAuthConfig.Token":           {Help: "Inline bearer token. Prefer TokenFile so the secret stays out of config.yaml."},
	"APIAuthConfig.TokenFile":       {Help: "Path to a file holding the bearer token (re-read per request, so rotation needs no restart)."},
	"APIAuthConfig.TrustedNetworks": {Help: "CIDRs whose source addresses bypass the token under auto mode. Loopback is always trusted."},

	"WebConfig.Tabs": {Help: "Show/hide navigation tabs in the operator console. Hidden tabs stay reachable by URL."},

	// ---- Storage -----------------------------------------------------------
	"StorageConfig.Path":        {Help: "SQLite call-log database file. Empty disables call history (the daemon still runs)."},
	"StorageConfig.CCCacheFile": {Label: "CC cache file", Help: "JSON cache the control-channel hunter uses to skip dead frequencies. Empty disables it."},

	// ---- Recordings --------------------------------------------------------
	"RecordingsConfig.Dir":           {Label: "Directory", Help: "Output directory for per-call WAV recordings."},
	"RecordingsConfig.SampleRate":    {Label: "Sample rate", Help: "Recorder PCM rate (4000–48000 Hz). Match audio.sample_rate to avoid a resample stage."},
	"RecordingsConfig.WriteRaw":      {Help: "Also write the raw (un-equalized) audio alongside the processed WAV."},
	"RecordingsConfig.SkipEncrypted": {Label: "Skip encrypted", Help: "Don't record calls flagged encrypted. Aborts and deletes the file if encryption is only detected mid-call."},
	"RecordingsConfig.Equalizer":     {Help: "Per-call CMA blind equalizer in the FM voice chain — useful on simulcast systems."},
	"EqualizerConfig.Enabled":        {Help: "Turn the blind equalizer on."},
	"EqualizerConfig.Taps":           {Help: "Equalizer FIR length. Default 8 when enabled."},
	"EqualizerConfig.StepSize":       {Help: "CMA adaptation step size. Default 1e-4 when enabled."},
	"RecordingsConfig.Normalize":     {Help: "Per-call EBU R128 loudness normalization — rewrites each finished WAV to a consistent perceptual level (pure Go, no ffmpeg)."},
	"NormalizeConfig.Enabled":        {Help: "Turn loudness normalization on."},
	"NormalizeConfig.ApplyTo":        {Label: "Apply to", Help: "Which audio to normalize: recording (rewrite WAV), distributed (outbound MP3 only), or both. Default recording.", Options: opts("", "(default: recording)", "recording", "recording", "distributed", "distributed", "both", "both")},
	"NormalizeConfig.TargetLUFS":     {Label: "Target LUFS", Help: "Integrated-loudness target. Default -16 when enabled."},
	"NormalizeConfig.TruePeakDBTP":   {Label: "True peak (dBTP)", Help: "True-peak ceiling in dBTP. Default -1.5 when enabled."},
	"NormalizeConfig.MaxBoostDB":     {Label: "Max gain (dB)", Help: "Cap on applied gain in either direction so quiet calls don't amplify hiss. Default 12 when enabled."},

	// ---- Metrics -----------------------------------------------------------
	"MetricsConfig.Enabled":     {Help: "Mount a Prometheus /metrics endpoint on the API HTTP server."},
	"MetricsConfig.DetailedFEC": {Help: "Opt into per-protocol FEC correction-depth histograms (gophertrunk_tetra_viterbi_corrections). Off by default — the buckets only make sense when profiling on-air recovery margins."},

	// ---- Retention ---------------------------------------------------------
	"RetentionConfig.CallLogDays": {Help: "Delete call-log rows older than this many days. 0 disables the call-log sweep."},
	"RetentionConfig.LogDays":     {Help: "Delete decoder-log rows (pager, aprs, vessel, dsc, aircraft, mdc1200, m17) older than this. 0 disables."},
	"RetentionConfig.FilesDays":   {Help: "Delete recorded WAV files older than this many days. 0 disables the file sweep."},
	"RetentionConfig.Interval":    {Help: "How often the sweeper runs (Go duration string, e.g. 1h). Default 1h."},

	// ---- Tone-out ----------------------------------------------------------
	"ToneOutConfig.Profiles":               {Help: "Paging-tone alarms to monitor. Two tones (A then B) for sequential paging, one for single-tone supervision."},
	"ToneProfileConfig.Name":               {Help: "Profile name (used in alerts and logs)."},
	"ToneProfileConfig.AlphaTag":           {Help: "Short display tag shown when this profile fires."},
	"ToneProfileConfig.Tones":              {Help: "Tone sequence: A-tone then B-tone for two-tone paging, or a single tone."},
	"ToneProfileConfig.ToleranceHz":        {Label: "Tolerance (Hz)", Help: "Frequency match window around each tone, in Hz."},
	"ToneProfileConfig.MagnitudeThreshold": {Help: "Minimum tone magnitude to count as present (squelches noise)."},
	"ToneProfileConfig.MaxGap":             {Help: "Largest gap allowed between A and B tones (Go duration, e.g. 250ms)."},
	"ToneProfileConfig.Cooldown":           {Help: "Minimum time between repeat alerts for this profile (Go duration)."},
	"ToneProfileConfig.System":             {Help: "Optional system name this profile is associated with."},
	"ToneProfileConfig.GroupID":            {Label: "Group ID", Help: "Optional talkgroup ID to tag the detection with."},
	"ToneProfileToneConfig.FrequencyHz":    {Label: "Frequency (Hz)", Help: "Tone frequency in Hz (e.g. 1122.5)."},
	"ToneProfileToneConfig.MinDuration":    {Help: "Minimum on-time to accept the tone (Go duration, e.g. 250ms)."},
	"ToneProfileToneConfig.MaxDuration":    {Help: "Maximum on-time before the tone is rejected. 0 disables the upper bound."},

	// ---- Scanner -----------------------------------------------------------
	"ScannerConfig.ScanMode":           {Help: "Trunking scan mode: all (follow every grant) or list (only TGs flagged Scan). Empty = all.", Options: opts("", "(default: all)", "all", "all", "list", "list")},
	"ScannerConfig.CCHunt":             {Label: "CC hunt", Help: "Multi-system control-channel hunter dwell + backoff tuning."},
	"ScannerConfig.Conventional":       {Help: "Fixed-frequency analog FM scan list."},
	"ScannerConfig.ManualTuneEnabled":  {Help: "Force-build the conventional scanner so the TUI 'f' key / manual_tune works even with no static channels (steals one voice SDR)."},
	"ScannerConfig.ManualTuneDisabled": {Help: "Veto the auto-detect rule — build the conventional scanner only when channels are listed or ManualTuneEnabled is set."},
	"CCHuntConfig.Enabled":             {Help: "Run the control-channel hunter. Defaults on when any trunked system is configured."},
	"CCHuntConfig.DwellMs":             {Help: "Per-frequency wait before declaring no lock. Default 3000 ms."},
	"CCHuntConfig.BackoffMs":           {Help: "Initial sleep after exhausting a system's CC list. Default 5000 ms; doubles per failure."},
	"CCHuntConfig.MaxBackoffMs":        {Help: "Cap on the exponential backoff. Default 60000 ms."},
	"ConvChannelConfig.Label":          {Help: "Display name for this conventional channel."},
	"ConvChannelConfig.FrequencyHz":    {Label: "Frequency", Hz: true, Help: "Channel center frequency."},
	"ConvChannelConfig.Mode":           {Help: "Demodulation: fm (wide) or nfm (narrow).", Options: opts("", "(fm)", "fm", "fm", "nfm", "nfm")},
	"ConvChannelConfig.SquelchDbFS":    {Label: "Squelch (dBFS)", Help: "Squelch threshold in dBFS. Default -50."},
	"ConvChannelConfig.HangtimeMs":     {Help: "Hold time after carrier drop before the channel is released. Default 1500 ms."},
	"ConvChannelConfig.Priority":       {Help: "Scan priority 1–10 (higher wins). 0 = unset."},
	"ConvChannelConfig.Tone":           {Help: "Optional CTCSS/DCS sub-audible squelch gate."},
	"ConvToneConfig.Mode":              {Help: "Sub-audible gate: none, ctcss, or dcs.", Options: opts("", "none", "none", "none", "ctcss", "ctcss", "dcs", "dcs")},
	"ConvToneConfig.CTCSSHz":           {Label: "CTCSS (Hz)", Help: "Target CTCSS frequency (50–300 Hz). Required when mode is ctcss."},
	"ConvToneConfig.DCSCode":           {Label: "DCS code", Help: "3-digit octal DCS code. Required when mode is dcs."},

	// ---- Audio -------------------------------------------------------------
	"AudioConfig.Enabled":      {Help: "Enable live speaker playback of decoded voice. Off by default (headless-safe). Recordings are unaffected."},
	"AudioConfig.Device":       {Help: "Backend output device. Empty/default = system sink; 'null' forces the no-op backend."},
	"AudioConfig.SampleRate":   {Label: "Sample rate", Help: "Playback rate (4000–48000 Hz). Default 8000; match recordings.sample_rate."},
	"AudioConfig.BufferMs":     {Help: "Playback queue depth in ms. Default 80."},
	"AudioConfig.Volume":       {Help: "Initial software gain, 0–1. Default 0.8."},
	"AudioConfig.Muted":        {Help: "Start muted."},
	"AudioConfig.LiveLoudness": {Label: "Live loudness", Help: "Real-time AGC on the live network stream (WebUI/gRPC) so its loudness tracks the normalized recordings. Digital live audio only; the on-disk WAV keeps its own normalization. Off by default."},

	// ---- Broadcast ---------------------------------------------------------
	"BroadcastConfig.MinDurationMs":   {Help: "Drop calls shorter than this from every feed. 0 streams any length."},
	"BroadcastConfig.Workers":         {Help: "Concurrent upload goroutines. 0 uses the package default."},
	"BroadcastConfig.Broadcastify":    {Help: "Broadcastify Calls upload feeds."},
	"BroadcastConfig.RdioScanner":     {Help: "RdioScanner call-upload feeds."},
	"BroadcastConfig.OpenMHz":         {Label: "OpenMHz", Help: "OpenMHz upload feeds."},
	"BroadcastConfig.Icecast":         {Help: "Live Icecast/ShoutCast mountpoint feeds."},
	"BroadcastifyFeedConfig.Enabled":  {Help: "Enable this feed. false keeps it configured but skipped."},
	"BroadcastifyFeedConfig.Name":     {Help: "Feed label for logs/metrics."},
	"BroadcastifyFeedConfig.APIKey":   {Label: "API key", Help: "Broadcastify Calls API key."},
	"BroadcastifyFeedConfig.SystemID": {Label: "System ID", Help: "Broadcastify system ID to upload under."},
	"BroadcastifyFeedConfig.Systems":  {Help: "GopherTrunk system names to stream. Empty = every system."},
	"RdioScannerFeedConfig.Enabled":   {Help: "Enable this feed. false keeps it configured but skipped."},
	"RdioScannerFeedConfig.Name":      {Help: "Feed label for logs/metrics."},
	"RdioScannerFeedConfig.URL":       {Label: "URL", Help: "RdioScanner ingest URL."},
	"RdioScannerFeedConfig.APIKey":    {Label: "API key", Help: "RdioScanner API key."},
	"RdioScannerFeedConfig.SystemID":  {Label: "System ID", Help: "RdioScanner system ID to upload under."},
	"RdioScannerFeedConfig.Systems":   {Help: "GopherTrunk system names to stream. Empty = every system."},
	"OpenMHzFeedConfig.Enabled":       {Help: "Enable this feed. false keeps it configured but skipped."},
	"OpenMHzFeedConfig.Name":          {Help: "Feed label for logs/metrics."},
	"OpenMHzFeedConfig.APIKey":        {Label: "API key", Help: "OpenMHz API key."},
	"OpenMHzFeedConfig.ShortName":     {Help: "OpenMHz system short name."},
	"OpenMHzFeedConfig.Systems":       {Help: "GopherTrunk system names to stream. Empty = every system."},
	"IcecastFeedConfig.Enabled":       {Help: "Enable this feed. false keeps it configured but skipped."},
	"IcecastFeedConfig.Name":          {Help: "Feed label for logs/metrics."},
	"IcecastFeedConfig.Host":          {Help: "Icecast/ShoutCast server hostname."},
	"IcecastFeedConfig.Port":          {Help: "Icecast/ShoutCast server port."},
	"IcecastFeedConfig.Mount":         {Help: "Mountpoint path, e.g. /stream.mp3."},
	"IcecastFeedConfig.Username":      {Help: "Source username (often 'source')."},
	"IcecastFeedConfig.Password":      {Help: "Source password for the mountpoint."},
	"IcecastFeedConfig.StreamName":    {Help: "Public stream name listeners see."},
	"IcecastFeedConfig.Systems":       {Help: "GopherTrunk system names to stream. Empty = every system."},

	// ---- Baseband ----------------------------------------------------------
	"BasebandConfig.Record":       {Help: "Tap live tuners and write their wideband IQ to WAV."},
	"BasebandConfig.Replay":       {Help: "Mount recorded WAVs as virtual tuners for offline decode."},
	"BasebandRecordConfig.Serial": {Help: "SDR serial whose IQ stream is recorded."},
	"BasebandRecordConfig.Dir":    {Label: "Directory", Help: "Directory the IQ recordings are written into."},
	"BasebandReplayConfig.File":   {Help: "Path to the baseband WAV recording to replay."},
	"BasebandReplayConfig.Serial": {Help: "Virtual device serial the pool reports. Empty generates one."},
	"BasebandReplayConfig.Role":   {Help: "Pool role: control / voice / auto.", Options: roleOpts()},
	"BasebandReplayConfig.Loop":   {Help: "Restart the recording on EOF for a continuous source. Default true."},

	// ---- Paging ------------------------------------------------------------
	"PagingConfig.POCSAG":               {Label: "POCSAG", Help: "POCSAG channels, each pinning one SDR to a paging frequency."},
	"PagingConfig.FLEX":                 {Label: "FLEX", Help: "FLEX channels, each pinning one SDR to a paging frequency (1600 bps)."},
	"PagingConfig.Wideband":             {Help: "Wideband groups fanning several paging channels off one SDR via a DDC bank."},
	"PagingPOCSAGConfig.Serial":         {Help: "SDR serial to pin to this POCSAG channel."},
	"PagingPOCSAGConfig.FrequencyHz":    {Label: "Frequency", Hz: true, Help: "POCSAG channel frequency."},
	"PagingPOCSAGConfig.BaudHz":         {Label: "Baud", Help: "POCSAG bit rate. 1200 is most common; 512 legacy, 2400 high-throughput (DAPNET).", Options: baudOpts()},
	"PagingFLEXConfig.Serial":           {Help: "SDR serial to pin to this FLEX channel."},
	"PagingFLEXConfig.FrequencyHz":      {Label: "Frequency", Hz: true, Help: "FLEX channel frequency."},
	"PagingWidebandConfig.Serial":       {Help: "SDR serial for this wideband paging group."},
	"PagingWidebandConfig.CenterFreqHz": {Label: "Center frequency", Hz: true, Help: "Dongle centre frequency. 0 auto-computes the midpoint of the channels."},
	"PagingWidebandConfig.Channels":     {Help: "Paging channels carved from this dongle's IQ band (any mix of POCSAG/FLEX)."},
	"PagingWidebandChannel.Protocol":    {Help: "Decoder for this channel.", Options: opts("pocsag", "POCSAG", "flex", "FLEX")},
	"PagingWidebandChannel.FrequencyHz": {Label: "Frequency", Hz: true, Help: "Channel frequency (must fall inside the group's IQ window)."},
	"PagingWidebandChannel.BaudHz":      {Label: "Baud", Help: "POCSAG bit rate (FLEX ignores it).", Options: baudOpts()},

	// ---- APRS --------------------------------------------------------------
	"APRSConfig.Channels":           {Help: "APRS/AX.25 AFSK channels. 144.39 MHz is the North-America primary."},
	"APRSChannelConfig.Serial":      {Help: "SDR serial to pin to this APRS channel."},
	"APRSChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "APRS channel frequency (144.39 MHz NA primary)."},
	"APRSChannelConfig.DropBadFCS":  {Label: "Drop bad FCS", Help: "Discard frames that fail the CRC instead of showing them flagged."},
	"APRSChannelConfig.DropNonUI":   {Label: "Drop non-UI", Help: "Discard non-UI (connected-mode) AX.25 frames."},

	// ---- AIS ---------------------------------------------------------------
	"AISConfig.Channels":               {Help: "Marine AIS GMSK channels. Pin 161.975 (87B) and 162.025 (88B) to catch class-A alternation."},
	"AISChannelConfig.Serial":          {Help: "SDR serial to pin to this AIS channel."},
	"AISChannelConfig.FrequencyHz":     {Label: "Frequency", Hz: true, Help: "AIS channel frequency (161.975 / 162.025 MHz)."},
	"AISChannelConfig.DropBadFCS":      {Label: "Drop bad FCS", Help: "Discard frames that fail the CRC instead of showing them flagged."},
	"AISChannelConfig.DropNonPosition": {Help: "Keep only position reports, dropping other AIS message types."},

	// ---- DSC ---------------------------------------------------------------
	"DSCConfig.Channels":           {Help: "Marine DSC FFSK channels. VHF ch 70 = 156.525 MHz carries distress/safety alerts."},
	"DSCChannelConfig.Serial":      {Help: "SDR serial to pin to this DSC channel."},
	"DSCChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "DSC channel frequency (156.525 MHz = VHF ch 70)."},
	"DSCChannelConfig.DropBadFCS":  {Label: "Drop bad FCS", Help: "Discard BCH-marginal sequences instead of showing them flagged."},

	// ---- MDC1200 -----------------------------------------------------------
	"MDC1200Config.Channels":           {Help: "Motorola MDC1200 FFSK channels. Target the conventional analog voice channels you monitor."},
	"MDC1200ChannelConfig.Serial":      {Help: "SDR serial to pin to this MDC1200 channel."},
	"MDC1200ChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "Analog voice channel frequency to watch for MDC1200 bursts."},
	"MDC1200ChannelConfig.DropBadCRC":  {Label: "Drop bad CRC", Help: "Discard CRC-failed bursts instead of showing them flagged."},

	// ---- ADS-B -------------------------------------------------------------
	"ADSBConfig.BeastUpstreams":     {Help: "BEAST Mode-S upstreams (dump1090 / readsb), typically host:30005."},
	"ADSBConfig.Channels":           {Help: "SDRs pinned to 1090 MHz for the native PPM Mode-S receiver."},
	"ADSBBeastConfig.Addr":          {Label: "Address", Help: "BEAST upstream host:port (e.g. 127.0.0.1:30005)."},
	"ADSBBeastConfig.Name":          {Help: "Label for logs/metrics."},
	"ADSBChannelConfig.Serial":      {Help: "SDR serial to pin to 1090 MHz."},
	"ADSBChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "Receiver frequency. 0 defaults to 1090 MHz. SDR must sample ≥ 2 Msps."},

	// ---- M17 ---------------------------------------------------------------
	"M17Config.Channels":           {Help: "M17 channels. Simplex calling is commonly 144.975 (2 m) / 433.475 MHz (70 cm)."},
	"M17ChannelConfig.Serial":      {Help: "SDR serial to pin to this M17 channel."},
	"M17ChannelConfig.FrequencyHz": {Label: "Frequency", Hz: true, Help: "M17 channel frequency."},

	// ---- LoRa --------------------------------------------------------------
	"LoRaConfig.Channels":                  {Help: "LoRa SDR captures. Each pins one SDR and fans its IQ band into one or more LoRa sub-channels."},
	"LoRaChannelConfig.Serial":             {Help: "SDR serial to pin to this LoRa capture."},
	"LoRaChannelConfig.CenterHz":           {Label: "Center", Hz: true, Help: "Dongle centre frequency for the wide-band capture (e.g. 915 MHz US / 868 MHz EU ISM)."},
	"LoRaChannelConfig.Bandwidth":          {Label: "Bandwidth", Hz: true, Help: "LoRa channel bandwidth for every sub-channel: 125000, 250000 or 500000 Hz."},
	"LoRaChannelConfig.Oversample":         {Help: "Samples per chip the demodulator runs at. 0 defaults to 2."},
	"LoRaChannelConfig.SubChannels":        {Help: "LoRa carriers within the dongle's IQ band, each decoded in parallel."},
	"LoRaChannelConfig.LoRaWANKeys":        {Help: "Optional LoRaWAN device session keys; enable MIC verification and payload decryption."},
	"LoRaSubChannelConfig.OffsetHz":        {Label: "Offset", Hz: true, Help: "Carrier offset from the capture centre frequency."},
	"LoRaSubChannelConfig.SpreadingFactor": {Help: "Spreading factor 7..12. 0 auto-detects across SF7..12."},
	"LoRaSubChannelConfig.SyncWord":        {Help: "LoRa sync word: 0x12 (18) private networks, 0x34 (52) LoRaWAN."},
	"LoRaSubChannelConfig.Label":           {Help: "Display label for this sub-channel."},
	"LoRaWANKeyConfig.DevAddr":             {Help: "LoRaWAN device address (8 hex digits) the keys belong to."},
	"LoRaWANKeyConfig.NwkSKey":             {Help: "Network session key (32 hex digits) — verifies the message integrity code."},
	"LoRaWANKeyConfig.AppSKey":             {Help: "Application session key (32 hex digits) — decrypts the frame payload."},
}

// FieldMetaFor returns the shared metadata for structName.field. An absent
// entry yields the zero FieldMeta (humanized label, plain widget).
func FieldMetaFor(structName, field string) FieldMeta {
	return fieldMetas[structName+"."+field]
}

// FieldMetas returns the whole registry (the web builder serves it over
// /api/v1/config/fieldmeta). The returned map is the live registry; callers
// must not mutate it.
func FieldMetas() map[string]FieldMeta {
	return fieldMetas
}
