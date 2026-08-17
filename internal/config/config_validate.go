// Configuration validation: the per-section validator framework and every
// validateX helper. Split out of config.go, which now holds only the config
// schema (the YAML-mapped DTOs) plus loading/path resolution.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// sectionValidator pairs a config section's logical name (matching the
// keys the web Config Builder uses) with the helper that validates it.
// Each helper returns every error it finds in its section (one per failing
// list item plus any section-level checks) so the builder can surface them
// all at once.
type sectionValidator struct {
	name string
	fn   func(Config) []error
}

// sectionValidators returns the per-section validators in the same order
// the monolithic Validate() used to run them, so the first error reported
// by Validate() is unchanged. Sections with no rules (log, api, storage,
// metrics, …) are intentionally absent — ValidateSection treats an unknown
// or rule-free section as valid.
func sectionValidators() []sectionValidator {
	return []sectionValidator{
		{"sdr", Config.validateSDR},
		{"trunking", Config.validateTrunking},
		{"recordings", Config.validateRecordings},
		{"retention", Config.validateRetention},
		{"scanner", Config.validateScanner},
		{"audio", Config.validateAudio},
		{"broadcast", Config.validateBroadcast},
		{"baseband", Config.validateBaseband},
		{"web", Config.validateWeb},
	}
}

// Validate reports the first configuration error, keyed by section path
// (e.g. "trunking.systems[0]: name required"). It is the authoritative
// gate run by Load and the config Writer. The checks are organised into
// per-section helpers so the web Config Builder can validate one section
// at a time (ValidateSection) or collect every error (ValidateAll);
// Validate preserves the original first-error contract.
func (c Config) Validate() error {
	for _, v := range sectionValidators() {
		if errs := v.fn(c); len(errs) > 0 {
			return errs[0]
		}
	}
	return nil
}

// ValidateAll runs every section validator and returns every error found
// across the whole config. An empty slice means the config is valid. The
// web Config Builder uses this to light up every problem in one pass.
func (c Config) ValidateAll() []error {
	var errs []error
	for _, v := range sectionValidators() {
		errs = append(errs, v.fn(c)...)
	}
	return errs
}

// ValidateSection validates a single section by name (the keys returned by
// sectionValidators / used by the web Config Builder) and returns all of
// that section's errors. An unknown or rule-free section name yields nil
// (treated as valid). Cross-section checks (e.g. wideband channels
// referencing trunking.systems) run against the whole Config, so the
// caller should pass a fully-populated draft.
func (c Config) ValidateSection(section string) []error {
	for _, v := range sectionValidators() {
		if v.name == section {
			return v.fn(c)
		}
	}
	return nil
}

func (c Config) validateSDR() []error {
	var errs []error
	if c.SDR.SampleRate != 0 && (c.SDR.SampleRate < 225_000 || c.SDR.SampleRate > 20_000_000) {
		errs = append(errs, errors.New("sdr.sample_rate must be between 225 kHz and 20 MHz"))
	}
	seenSerials := make(map[string]int, len(c.SDR.Devices))
	for i, d := range c.SDR.Devices {
		switch d.Role {
		case "", "control", "voice", "auto", "wideband":
		default:
			errs = append(errs, fmt.Errorf("sdr.devices[%d]: role must be control|voice|auto|wideband", i))
			continue
		}
		if d.Role == "wideband" {
			if err := validateWidebandDevice(i, d, c.SDR.SampleRate, c.Trunking.Systems); err != nil {
				errs = append(errs, err)
				continue
			}
		}
		if d.DCAvoidOffsetHz != 0 {
			if d.DCAvoidOffsetHz < 0 {
				errs = append(errs, fmt.Errorf("sdr.devices[%d]: dc_avoid_offset_hz must be >= 0 (0 = auto)", i))
			} else if c.SDR.SampleRate != 0 && d.DCAvoidOffsetHz >= int(c.SDR.SampleRate)/2 {
				errs = append(errs, fmt.Errorf(
					"sdr.devices[%d]: dc_avoid_offset_hz (%d) must be < sample_rate/2 (%d)",
					i, d.DCAvoidOffsetHz, int(c.SDR.SampleRate)/2))
			}
		}
		if d.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[d.Serial]; dup {
			errs = append(errs, fmt.Errorf(
				"sdr.devices[%d]: duplicate serial %q (also at sdr.devices[%d]) — "+
					"one physical SDR cannot serve multiple roles; P25 trunking needs "+
					"separate dongles for control and voice",
				i, d.Serial, prev))
			continue
		}
		seenSerials[d.Serial] = i
	}
	// Validate rtl_tcp endpoints. Addr is required; role must match
	// the standard set; serial collisions with local devices are
	// rejected for the same reason serial dedup runs above.
	for i, r := range c.SDR.RTLTCP {
		if err := validateRTLTCPFields(i, r); err != nil {
			errs = append(errs, err)
			continue
		}
		if r.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[r.Serial]; dup {
			errs = append(errs, fmt.Errorf(
				"sdr.rtl_tcp[%d]: serial %q collides with sdr.devices[%d]",
				i, r.Serial, prev))
			continue
		}
		seenSerials[r.Serial] = i
	}
	// Validate SoapySDRServer endpoints. Same rules as rtl_tcp, plus the
	// stream protocol and sample format must be ones the driver supports.
	for i, s := range c.SDR.SoapyRemote {
		if err := validateSoapyFields(i, s); err != nil {
			errs = append(errs, err)
			continue
		}
		if s.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[s.Serial]; dup {
			errs = append(errs, fmt.Errorf(
				"sdr.soapy_remote[%d]: serial %q collides with sdr.devices[%d]",
				i, s.Serial, prev))
			continue
		}
		seenSerials[s.Serial] = i
	}
	// Validate ka9q-radio channels. Addr + SSRC are required; role, encoding
	// and serial collisions follow the same rules as the blocks above.
	for i, k := range c.SDR.Ka9qRadio {
		if err := validateKa9qFields(i, k); err != nil {
			errs = append(errs, err)
			continue
		}
		if k.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[k.Serial]; dup {
			errs = append(errs, fmt.Errorf(
				"sdr.ka9q_radio[%d]: serial %q collides with sdr.devices[%d]",
				i, k.Serial, prev))
			continue
		}
		seenSerials[k.Serial] = i
	}
	return errs
}

func validateKa9qFields(i int, k Ka9qRadioConfig) error {
	if k.Addr == "" {
		return fmt.Errorf("sdr.ka9q_radio[%d]: addr is required (mDNS name or host:port)", i)
	}
	if k.SSRC == 0 {
		return fmt.Errorf("sdr.ka9q_radio[%d]: ssrc is required (non-zero)", i)
	}
	switch k.Role {
	case "", "control", "voice", "auto":
	default:
		return fmt.Errorf("sdr.ka9q_radio[%d]: role must be control|voice|auto", i)
	}
	switch k.Encoding {
	case "", "s16be", "s16le", "f32le", "f32be":
	default:
		return fmt.Errorf("sdr.ka9q_radio[%d]: encoding must be s16be|s16le|f32le|f32be", i)
	}
	if k.Channels != 0 && k.Channels != 1 && k.Channels != 2 {
		return fmt.Errorf("sdr.ka9q_radio[%d]: channels must be 1 or 2", i)
	}
	return nil
}

func validateRTLTCPFields(i int, r RTLTCPConfig) error {
	if r.Addr == "" {
		return fmt.Errorf("sdr.rtl_tcp[%d]: addr is required (host:port)", i)
	}
	switch r.Role {
	case "", "control", "voice", "auto":
	default:
		return fmt.Errorf("sdr.rtl_tcp[%d]: role must be control|voice|auto", i)
	}
	return nil
}

func validateSoapyFields(i int, s SoapyRemoteConfig) error {
	if s.Addr == "" {
		return fmt.Errorf("sdr.soapy_remote[%d]: addr is required (host:port)", i)
	}
	switch s.Role {
	case "", "control", "voice", "auto":
	default:
		return fmt.Errorf("sdr.soapy_remote[%d]: role must be control|voice|auto", i)
	}
	switch s.Format {
	case "", "CS16", "cs16", "CF32", "cf32":
	default:
		return fmt.Errorf("sdr.soapy_remote[%d]: format must be CS16 or CF32", i)
	}
	switch s.StreamProtocol {
	case "", "tcp":
	default:
		return fmt.Errorf("sdr.soapy_remote[%d]: stream_protocol must be tcp", i)
	}
	diversityMRC := false
	switch strings.ToLower(strings.TrimSpace(s.Diversity)) {
	case "", "none", "off":
	case "mrc", "mrc-static", "mrc_static":
		diversityMRC = true
	default:
		return fmt.Errorf("sdr.soapy_remote[%d]: diversity must be mrc, mrc-static or empty", i)
	}
	// antennas[] selects an RX antenna per channel. At most two (RX0, RX1), and
	// more than one only makes sense under mrc (a single-channel stream opens
	// only channel 0). No empty entries — an empty string is a silent no-op that
	// reads like an intentional default and hides a typo.
	if len(s.Antennas) > 2 {
		return fmt.Errorf("sdr.soapy_remote[%d]: antennas has %d entries (max 2: RX channel 0 and 1)", i, len(s.Antennas))
	}
	if len(s.Antennas) > 1 && !diversityMRC {
		return fmt.Errorf("sdr.soapy_remote[%d]: antennas has %d entries but only one RX channel is opened without diversity: mrc", i, len(s.Antennas))
	}
	for j, a := range s.Antennas {
		if strings.TrimSpace(a) == "" {
			return fmt.Errorf("sdr.soapy_remote[%d]: antennas[%d] is empty", i, j)
		}
	}
	// An antenna= left in the flat args string reaches make() but does NOT set a
	// per-channel antenna the way antennas[] does (make() args can't carry a
	// comma-separated multi-value), so a config with both is ambiguous — point
	// at the field that actually applies per channel.
	if len(s.Antennas) > 0 {
		if args, err := s.DeviceArgs(); err == nil {
			if _, ok := args["antenna"]; ok {
				return fmt.Errorf("sdr.soapy_remote[%d]: set the antenna via the antennas: list, not antenna= in args (the args value applies to make() only, not per RX channel)", i)
			}
		}
	}
	// diversity_capture taps the pre-combine branches, which only exist under a
	// multi-channel (MRC) stream.
	if s.DiversityCapture != "" && !diversityMRC {
		return fmt.Errorf("sdr.soapy_remote[%d]: diversity_capture needs diversity: mrc (there are no separate branches to capture on a single-channel stream)", i)
	}
	if s.DiversityCaptureSeconds != 0 {
		if s.DiversityCapture == "" {
			return fmt.Errorf("sdr.soapy_remote[%d]: diversity_capture_seconds set without diversity_capture", i)
		}
		if s.DiversityCaptureSeconds < 1 || s.DiversityCaptureSeconds > 60 {
			return fmt.Errorf("sdr.soapy_remote[%d]: diversity_capture_seconds is %d (want 1..60; two CS16 branches are tens of MB/s)", i, s.DiversityCaptureSeconds)
		}
	}
	if s.DiversityCapture != "" {
		// Fail at config load, not several hundred MB into a stream.
		dir := filepath.Dir(s.DiversityCapture)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			return fmt.Errorf("sdr.soapy_remote[%d]: diversity_capture directory %q does not exist", i, dir)
		}
	}
	// stream_mtu is in bytes; 0 means SoapyRemote's default (1500). Reject
	// values that can't be a real endpoint MTU — too small to hold a useful
	// frame, or above the driver's 4 MiB per-transfer read guard.
	if s.StreamMTU != 0 && (s.StreamMTU < 64 || s.StreamMTU > 1<<20) {
		return fmt.Errorf("sdr.soapy_remote[%d]: stream_mtu %d out of range (64..1048576 bytes, 0 = default 1500)", i, s.StreamMTU)
	}
	// stream_window is in bytes; 0 means the client default (8 MiB). Bound it
	// to a sane socket-buffer range, and require it to hold at least one MTU so
	// the advertised credit (window/mtu) is never zero (which would starve the
	// server's sender).
	if s.StreamWindow != 0 {
		if s.StreamWindow < 1<<16 || s.StreamWindow > 256<<20 {
			return fmt.Errorf("sdr.soapy_remote[%d]: stream_window %d out of range (65536..268435456 bytes, 0 = default 8 MiB)", i, s.StreamWindow)
		}
		mtu := s.StreamMTU
		if mtu == 0 {
			mtu = 1500
		}
		if s.StreamWindow < mtu {
			return fmt.Errorf("sdr.soapy_remote[%d]: stream_window %d must be >= stream_mtu %d", i, s.StreamWindow, mtu)
		}
	}
	if _, err := s.DeviceArgs(); err != nil {
		return fmt.Errorf("sdr.soapy_remote[%d]: args: %w", i, err)
	}
	// SoapyRemote stream arguments must not be smuggled through args.
	// Everything in args goes to the remote make() call, but GopherTrunk
	// builds the SETUP_STREAM frame itself, so a remote:* stream knob left in
	// args is silently ignored for streaming (issue #876: a user set
	// remote:mtu=8000 in args and stayed on the 1500-byte default). Reject the
	// reserved keys and point at the dedicated field that actually takes effect.
	if reserved, dest := reservedStreamArg(s.Args); reserved != "" {
		return fmt.Errorf("sdr.soapy_remote[%d]: args contains stream argument %q, which is ignored in make() args; set the %q config key instead", i, reserved, dest)
	}
	// master_clock_rate is in Hz; a non-zero value below 1 MHz is almost
	// certainly a units slip (e.g. "61" meaning 61 MHz) — catch it early.
	if s.MasterClockHz != 0 && s.MasterClockHz < 1_000_000 {
		return fmt.Errorf("sdr.soapy_remote[%d]: master_clock_rate %d looks too low; it is in Hz (e.g. 61440000 for a B210)", i, s.MasterClockHz)
	}
	return nil
}

func (c Config) validateTrunking() []error {
	var errs []error
	if c.Trunking.CallTimeoutMs < 0 {
		errs = append(errs, fmt.Errorf("trunking.call_timeout_ms: %d ms must be ≥ 0", c.Trunking.CallTimeoutMs))
	}
	if c.Trunking.VoiceHangtimeMs < 0 {
		errs = append(errs, fmt.Errorf("trunking.voice_hangtime_ms: %d ms must be ≥ 0", c.Trunking.VoiceHangtimeMs))
	}
	switch c.Trunking.VoiceCallGrouping {
	case "", "transmission", "conversation":
	default:
		errs = append(errs, fmt.Errorf("trunking.voice_call_grouping: %q must be \"transmission\" or \"conversation\"", c.Trunking.VoiceCallGrouping))
	}
	for i, s := range c.Trunking.Systems {
		if err := validateSystem(i, s); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// validateSystem returns the first error in one trunking system (the
// builder reports one error per system; fix-and-revalidate surfaces the
// next).
func validateSystem(i int, s SystemConfig) error {
	if s.Name == "" {
		return fmt.Errorf("trunking.systems[%d]: name required", i)
	}
	proto, err := trunking.ParseProtocol(s.Protocol)
	if err != nil {
		return fmt.Errorf("trunking.systems[%d]: %w", i, err)
	}
	if s.DMRColorCode != nil {
		if *s.DMRColorCode > 15 {
			return fmt.Errorf("trunking.systems[%d].color_code: %d outside 0..15", i, *s.DMRColorCode)
		}
		if proto != trunking.ProtocolDMRTier2 && proto != trunking.ProtocolDMRTier1 {
			return fmt.Errorf("trunking.systems[%d].color_code: only valid for conventional DMR "+
				"(protocol dmr-tier2 or dmr-tier1), not %q", i, s.Protocol)
		}
	}
	seenBandPlanIDs := make(map[uint8]int, len(s.P25BandPlan))
	for k, e := range s.P25BandPlan {
		if e.ChannelID > 15 {
			return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: channel_id %d outside 0..15", i, k, e.ChannelID)
		}
		if prev, dup := seenBandPlanIDs[e.ChannelID]; dup {
			return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: duplicate channel_id %d (also at p25_band_plan[%d])", i, k, e.ChannelID, prev)
		}
		seenBandPlanIDs[e.ChannelID] = k
		if e.SpacingHz == 0 {
			return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: spacing_hz required (nonzero)", i, k)
		}
		if e.BaseHz == 0 {
			return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: base_hz required (nonzero)", i, k)
		}
	}
	if bp := s.DMRBandPlan; bp != nil {
		hasLinear := bp.Linear != nil
		hasTable := len(bp.Table) > 0
		switch {
		case hasLinear && hasTable:
			return fmt.Errorf("trunking.systems[%d].dmr_band_plan: set either linear or table, not both", i)
		case !hasLinear && !hasTable:
			return fmt.Errorf("trunking.systems[%d].dmr_band_plan: one of linear or table is required", i)
		}
		if hasLinear {
			if bp.Linear.SpacingHz == 0 {
				return fmt.Errorf("trunking.systems[%d].dmr_band_plan.linear: spacing_hz required (nonzero)", i)
			}
			if bp.Linear.BaseHz == 0 {
				return fmt.Errorf("trunking.systems[%d].dmr_band_plan.linear: base_hz required (nonzero)", i)
			}
		}
		if hasTable {
			seenLCN := make(map[uint16]int, len(bp.Table))
			for k, e := range bp.Table {
				if e.FreqHz == 0 {
					return fmt.Errorf("trunking.systems[%d].dmr_band_plan.table[%d]: freq_hz required (nonzero)", i, k)
				}
				if prev, dup := seenLCN[e.LCN]; dup {
					return fmt.Errorf("trunking.systems[%d].dmr_band_plan.table[%d]: duplicate lcn %d (also at table[%d])", i, k, e.LCN, prev)
				}
				seenLCN[e.LCN] = k
			}
		}
	}
	if bp := s.NXDNBandPlan; bp != nil {
		hasLinear := bp.Linear != nil
		hasTable := len(bp.Table) > 0
		switch {
		case hasLinear && hasTable:
			return fmt.Errorf("trunking.systems[%d].nxdn_band_plan: set either linear or table, not both", i)
		case !hasLinear && !hasTable:
			return fmt.Errorf("trunking.systems[%d].nxdn_band_plan: one of linear or table is required", i)
		}
		if hasLinear {
			if bp.Linear.SpacingHz == 0 {
				return fmt.Errorf("trunking.systems[%d].nxdn_band_plan.linear: spacing_hz required (nonzero)", i)
			}
			if bp.Linear.BaseHz == 0 {
				return fmt.Errorf("trunking.systems[%d].nxdn_band_plan.linear: base_hz required (nonzero)", i)
			}
		}
		if hasTable {
			seenCh := make(map[uint16]int, len(bp.Table))
			for k, e := range bp.Table {
				if e.FreqHz == 0 {
					return fmt.Errorf("trunking.systems[%d].nxdn_band_plan.table[%d]: freq_hz required (nonzero)", i, k)
				}
				if prev, dup := seenCh[e.Channel]; dup {
					return fmt.Errorf("trunking.systems[%d].nxdn_band_plan.table[%d]: duplicate channel %d (also at table[%d])", i, k, e.Channel, prev)
				}
				seenCh[e.Channel] = k
			}
		}
	}
	seenKeyIDs := make(map[uint16]struct{}, len(s.EncryptionKeys))
	for k, ek := range s.EncryptionKeys {
		switch strings.ToLower(strings.TrimSpace(ek.Algorithm)) {
		case "rc4", "arc4":
			// supported
		case "":
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: algorithm is required (use \"rc4\")", i, k)
		case "aes", "des":
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: algorithm %q is not supported yet (only \"rc4\")", i, k, ek.Algorithm)
		default:
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: unknown algorithm %q (use \"rc4\")", i, k, ek.Algorithm)
		}
		if _, dup := seenKeyIDs[ek.KeyID]; dup {
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: duplicate key_id %d", i, k, ek.KeyID)
		}
		seenKeyIDs[ek.KeyID] = struct{}{}
		b, err := decodeHexKey(ek.Key)
		if err != nil {
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: %w", i, k, err)
		}
		if len(b) > 32 {
			return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: key is %d bytes, must be 1..32", i, k, len(b))
		}
	}
	type rfssSite struct{ rfss, site uint8 }
	seenSites := make(map[rfssSite]int, len(s.Sites))
	for k, st := range s.Sites {
		if strings.TrimSpace(st.Name) == "" {
			return fmt.Errorf("trunking.systems[%d].sites[%d]: name required", i, k)
		}
		key := rfssSite{st.RFSS, st.Site}
		if prev, dup := seenSites[key]; dup {
			return fmt.Errorf("trunking.systems[%d].sites[%d]: duplicate rfss %d / site %d (also at sites[%d])", i, k, st.RFSS, st.Site, prev)
		}
		seenSites[key] = k
		if st.Latitude < -90 || st.Latitude > 90 {
			return fmt.Errorf("trunking.systems[%d].sites[%d].latitude: %g outside -90..90", i, k, st.Latitude)
		}
		if st.Longitude < -180 || st.Longitude > 180 {
			return fmt.Errorf("trunking.systems[%d].sites[%d].longitude: %g outside -180..180", i, k, st.Longitude)
		}
	}
	switch s.EncryptedCalls.Mode {
	case "", "follow", "metadata", "ignore":
	default:
		return fmt.Errorf("trunking.systems[%d].encrypted_calls.mode: %q must be \"follow\", \"metadata\", or \"ignore\"", i, s.EncryptedCalls.Mode)
	}
	if s.EncryptedCalls.MetadataFollowMs < 0 {
		return fmt.Errorf("trunking.systems[%d].encrypted_calls.metadata_follow_ms: %d ms must be ≥ 0", i, s.EncryptedCalls.MetadataFollowMs)
	}
	return nil
}

func (c Config) validateRecordings() []error {
	if c.Recordings.SampleRate != 0 && (c.Recordings.SampleRate < 4000 || c.Recordings.SampleRate > 48_000) {
		return []error{fmt.Errorf("recordings.sample_rate %d outside 4000..48000", c.Recordings.SampleRate)}
	}
	if c.Recordings.VoiceTapBufferChunks != 0 && (c.Recordings.VoiceTapBufferChunks < 1 || c.Recordings.VoiceTapBufferChunks > 1024) {
		return []error{fmt.Errorf("recordings.voice_tap_buffer_chunks %d outside 1..1024", c.Recordings.VoiceTapBufferChunks)}
	}
	if err := validateNamingTemplate("recordings.filename_template", c.Recordings.FilenameTemplate, false); err != nil {
		return []error{err}
	}
	if err := validateNamingTemplate("recordings.path_template", c.Recordings.PathTemplate, true); err != nil {
		return []error{err}
	}
	switch c.Recordings.Normalize.ApplyTo {
	case "", "recording", "distributed", "both":
	default:
		return []error{fmt.Errorf("recordings.normalize.apply_to %q invalid (use recording, distributed, or both)", c.Recordings.Normalize.ApplyTo)}
	}
	return nil
}

// namingTemplateTokens are the substitutions recognised in
// recordings.filename_template / path_template. Mirrored in
// internal/voice/recorder.go's expandNameTemplate — keep the two in sync.
var namingTemplateTokens = map[string]struct{}{
	"date": {}, "time": {}, "datetime": {}, "year": {}, "month": {}, "day": {},
	"tg": {}, "alpha": {}, "freq": {}, "src": {}, "ts": {}, "proto": {},
	"system": {}, "callid": {},
}

// validateNamingTemplate rejects a recording naming template that uses an
// unknown {token} (a typo that would otherwise render literally), a path
// separator in a filename template, or an absolute path template.
func validateNamingTemplate(field, tmpl string, allowSep bool) error {
	if tmpl == "" {
		return nil
	}
	if allowSep {
		if strings.HasPrefix(tmpl, "/") || filepath.IsAbs(tmpl) {
			return fmt.Errorf("%s: must be a relative path, got %q", field, tmpl)
		}
	} else if strings.ContainsAny(tmpl, `/\`) {
		return fmt.Errorf("%s: is a filename, not a path — remove the separator in %q", field, tmpl)
	}
	for i := 0; i < len(tmpl); {
		open := strings.IndexByte(tmpl[i:], '{')
		if open < 0 {
			break
		}
		open += i
		end := strings.IndexByte(tmpl[open:], '}')
		if end < 0 {
			return fmt.Errorf("%s: unterminated '{' in %q", field, tmpl)
		}
		end += open
		tok := tmpl[open+1 : end]
		if _, ok := namingTemplateTokens[tok]; !ok {
			return fmt.Errorf("%s: unknown token {%s} in %q (valid: date, time, datetime, year, month, day, tg, alpha, freq, src, ts, proto, system, callid)", field, tok, tmpl)
		}
		i = end + 1
	}
	return nil
}

func (c Config) validateRetention() []error {
	if c.Retention.Interval != "" {
		if _, err := parseDurationFlexible(c.Retention.Interval); err != nil {
			return []error{fmt.Errorf("retention.interval: %w", err)}
		}
	}
	return nil
}

func (c Config) validateAudio() []error {
	var errs []error
	if c.Audio.SampleRate != 0 && (c.Audio.SampleRate < 4000 || c.Audio.SampleRate > 48_000) {
		errs = append(errs, fmt.Errorf("audio.sample_rate %d outside 4000..48000", c.Audio.SampleRate))
	}
	if c.Audio.Volume != 0 && (c.Audio.Volume < 0 || c.Audio.Volume > 1) {
		errs = append(errs, fmt.Errorf("audio.volume %f outside 0..1", c.Audio.Volume))
	}
	return errs
}

func (c Config) validateScanner() []error {
	var errs []error
	switch c.Scanner.ScanMode {
	case "", "all", "list":
	default:
		errs = append(errs, fmt.Errorf("scanner.scan_mode must be \"all\" or \"list\""))
	}
	for i, ch := range c.Scanner.Conventional {
		if err := validateConvChannel(i, ch); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateConvChannel(i int, ch ConvChannelConfig) error {
	if ch.FrequencyHz == 0 {
		return fmt.Errorf("scanner.conventional[%d]: frequency_hz required", i)
	}
	switch ch.Mode {
	case "", "fm", "nfm":
	default:
		return fmt.Errorf("scanner.conventional[%d]: mode must be fm|nfm", i)
	}
	if ch.ActivityDebounceMs < 0 {
		return fmt.Errorf("scanner.conventional[%d]: activity_debounce_ms must be ≥ 0", i)
	}
	if ch.SquelchHysteresisDb < 0 {
		return fmt.Errorf("scanner.conventional[%d]: squelch_hysteresis_db must be ≥ 0", i)
	}
	switch ch.Tone.Mode {
	case "", "none":
	case "ctcss":
		if ch.Tone.CTCSSHz < 50 || ch.Tone.CTCSSHz > 300 {
			return fmt.Errorf("scanner.conventional[%d].tone.ctcss_hz %v outside 50..300 Hz",
				i, ch.Tone.CTCSSHz)
		}
	case "dcs":
		if len(ch.Tone.DCSCode) != 3 {
			return fmt.Errorf("scanner.conventional[%d].tone.dcs_code must be 3 octal digits", i)
		}
		for _, r := range ch.Tone.DCSCode {
			if r < '0' || r > '7' {
				return fmt.Errorf("scanner.conventional[%d].tone.dcs_code %q must be octal 0..7",
					i, ch.Tone.DCSCode)
			}
		}
	default:
		return fmt.Errorf("scanner.conventional[%d].tone.mode must be ctcss|dcs|none", i)
	}
	return nil
}

func (c Config) validateBroadcast() []error {
	if err := c.Broadcast.validate(); err != nil {
		return []error{err}
	}
	return nil
}

func (c Config) validateBaseband() []error {
	var errs []error
	for i, r := range c.Baseband.Record {
		if r.Serial == "" {
			errs = append(errs, fmt.Errorf("baseband.record[%d]: serial required", i))
			continue
		}
		if r.Dir == "" {
			errs = append(errs, fmt.Errorf("baseband.record[%d]: dir required", i))
		}
		switch strings.ToLower(strings.TrimSpace(r.Tap)) {
		case "", "wideband", "ddc":
		default:
			errs = append(errs, fmt.Errorf("baseband.record[%d]: tap must be wideband|ddc", i))
		}
	}
	for i, r := range c.Baseband.Replay {
		if r.File == "" {
			errs = append(errs, fmt.Errorf("baseband.replay[%d]: file required", i))
			continue
		}
		switch r.Role {
		case "", "control", "voice", "auto":
		default:
			errs = append(errs, fmt.Errorf("baseband.replay[%d]: role must be control|voice|auto", i))
		}
	}
	if a := c.Baseband.AutoRecord; a.Enabled {
		if a.Dir == "" {
			errs = append(errs, fmt.Errorf("baseband.auto_record: dir required when enabled"))
		}
		if a.Seconds <= 0 {
			errs = append(errs, fmt.Errorf("baseband.auto_record: seconds must be a positive integer"))
		}
		// Format is validated against the siglab sample-format set inline (the
		// widely-imported config package deliberately does not import siglab).
		// Keep this list in sync with siglab.ParseSampleFormat.
		switch strings.ToLower(strings.TrimSpace(a.Format)) {
		case "", "cs16", "f32", "u8":
		default:
			errs = append(errs, fmt.Errorf("baseband.auto_record: format must be cs16|f32|u8, got %q", a.Format))
		}
		if a.OnConcurrentCalls < 0 {
			errs = append(errs, fmt.Errorf("baseband.auto_record: on_concurrent_calls must not be negative"))
		}
		switch strings.ToLower(strings.TrimSpace(a.Tap)) {
		case "", "wideband", "ddc":
		default:
			errs = append(errs, fmt.Errorf("baseband.auto_record: tap must be wideband|ddc, got %q", a.Tap))
		}
		if _, err := a.CooldownDuration(); err != nil {
			errs = append(errs, fmt.Errorf("baseband.auto_record: cooldown %q: %w", a.Cooldown, err))
		}
		// No automatic trigger set is allowed: it leaves the feature armed for
		// the manual API trigger only (still a valid, useful configuration).
	}
	return errs
}

func (c Config) validateWeb() []error {
	for key := range c.Web.Tabs {
		if !KnownUITabs[key] {
			valid := make([]string, 0, len(KnownUITabs))
			for k := range KnownUITabs {
				valid = append(valid, k)
			}
			sort.Strings(valid)
			return []error{fmt.Errorf("web.tabs: unknown tab %q (valid: %s)", key, strings.Join(valid, ", "))}
		}
	}
	switch c.Web.IDBase {
	case "", "hex", "dec":
	default:
		return []error{fmt.Errorf("web.id_base: must be hex or dec, got %q", c.Web.IDBase)}
	}
	return nil
}

// widebandGuardFrac reserves this fraction of the dongle's IQ band at
// each edge as a guard against alias roll-off. Channel frequencies
// outside the resulting usable interval are rejected at config load.
// Mirrors the default passed to internal/dsp/tuner.NewDDCBank.
const widebandGuardFrac = 0.05

// validateWidebandDevice checks a wideband SDR entry's centre-freq,
// strategy, and channel list. sampleRateHz may be zero — Validate has
// already accepted that as "fall back to the pool default" — in which
// case the in-band check uses sdr.DefaultSampleRateHz so a missing
// rate doesn't bypass the per-channel sanity check.
//
// Each channel must reference a system whose protocol is either:
//   - "dmr-tier2" — Tier II conventional; the channel frequency is one
//     repeater carrier.
//   - "dmr"       — Tier III trunked; the channel frequency must match
//     one of the system's control_channels (the wideband dongle is
//     hosting that CC).
func validateWidebandDevice(idx int, d DeviceConfig, sampleRateHz uint32, systems []SystemConfig) error {
	if d.Serial == "" {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires serial (the daemon binds the channel list to the device by USB serial)", idx)
	}
	if d.VoiceTaps < 0 {
		return fmt.Errorf("sdr.devices[%d]: voice_taps %d out of range; 0 disables, a positive value allocates that many virtual voice DDC taps on the dongle",
			idx, d.VoiceTaps)
	}
	if d.SignallingTaps < 0 {
		return fmt.Errorf("sdr.devices[%d]: signalling_taps %d out of range; 0 disables, a positive value allocates that many P25 Phase 2 alias-harvesting DDC taps on the dongle",
			idx, d.SignallingTaps)
	}
	if d.CenterFreqHz == 0 {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires center_freq_hz", idx)
	}
	switch d.TunerStrategy {
	case "", "auto", "ddc", "polyphase":
	default:
		return fmt.Errorf("sdr.devices[%d]: tuner_strategy must be auto|ddc|polyphase, got %q", idx, d.TunerStrategy)
	}
	if len(d.Channels) == 0 {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires at least one channel", idx)
	}
	rate := sampleRateHz
	if rate == 0 {
		rate = 2_048_000 // sdr.DefaultSampleRateHz; avoid an import cycle by repeating it
	}
	usableHalfBand := float64(rate) * (0.5 - widebandGuardFrac)
	systemsByName := make(map[string]SystemConfig, len(systems))
	for _, s := range systems {
		systemsByName[s.Name] = s
	}
	seenFreq := make(map[uint32]int, len(d.Channels))
	for j, ch := range d.Channels {
		if ch.FrequencyHz == 0 {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: frequency_hz required", idx, j)
		}
		if ch.System == "" {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: system required", idx, j)
		}
		sys, ok := systemsByName[ch.System]
		if !ok {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: system %q is not declared in trunking.systems", idx, j, ch.System)
		}
		switch sys.Protocol {
		case "dmr-tier2", "dmr_tier2", "dmr-t2", "dmrtier2",
			"dmr-tier1", "dmr_tier1", "dmr-t1", "dmrtier1":
			// Tier II conventional / Tier I direct-mode — channel freq is a
			// repeater or simplex carrier, no relationship to
			// system.ControlChannels required.
		case "dmr", "p25", "p25-phase2", "p25_phase2", "p25p2", "tetra":
			// Trunked control-channel protocols — the wideband channel
			// MUST be one of the system's declared control channels.
			// Tier III DMR's CSBK chain, P25 Phase 1's TSBK chain, P25
			// Phase 2's H-DQPSK MAC chain, and TETRA's MAC/CMCE chain all
			// run on a frequency the system advertises in control_channels;
			// voice grants hop elsewhere (TETRA voice follows on a
			// role: voice SDR — the wideband path multiplexes control
			// channels, not the 144 kHz TETRA voice slots).
			matched := false
			for _, cc := range sys.ControlChannels {
				if cc == ch.FrequencyHz {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf(
					"sdr.devices[%d].channels[%d]: frequency_hz %d does not match any of system %q's "+
						"control_channels %v (wideband %s channels must sit on a declared control channel)",
					idx, j, ch.FrequencyHz, ch.System, sys.ControlChannels, sys.Protocol)
			}
		default:
			return fmt.Errorf(
				"sdr.devices[%d].channels[%d]: system %q has protocol %q; wideband currently supports "+
					"dmr-tier2 (Tier II conventional), dmr (Tier III trunked control channel), "+
					"p25 (Phase 1 trunked control channel), p25-phase2 (Phase 2 trunked control channel), "+
					"and tetra (trunked control channel)",
				idx, j, ch.System, sys.Protocol)
		}
		offset := float64(ch.FrequencyHz) - float64(d.CenterFreqHz)
		if offset > usableHalfBand || offset < -usableHalfBand {
			return fmt.Errorf(
				"sdr.devices[%d].channels[%d]: frequency_hz %d is %.1f kHz from center; usable band is ±%.1f kHz "+
					"(sample_rate %d Hz minus %.0f%% guard)",
				idx, j, ch.FrequencyHz, offset/1000, usableHalfBand/1000, rate, widebandGuardFrac*100)
		}
		if prev, dup := seenFreq[ch.FrequencyHz]; dup {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: duplicate frequency_hz %d (also at channels[%d])", idx, j, ch.FrequencyHz, prev)
		}
		seenFreq[ch.FrequencyHz] = j
	}
	return nil
}

// validate checks that every enabled broadcast feed carries the fields
// its backend requires. Disabled feeds are left unchecked so operators
// can pre-stage credentials.
func (b BroadcastConfig) validate() error {
	if b.MinDurationMs < 0 {
		return errors.New("broadcast.min_duration_ms must not be negative")
	}
	for i, f := range b.Broadcastify {
		if !f.Enabled {
			continue
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.broadcastify[%d]: api_key required", i)
		}
		if f.SystemID == 0 {
			return fmt.Errorf("broadcast.broadcastify[%d]: system_id required", i)
		}
	}
	for i, f := range b.RdioScanner {
		if !f.Enabled {
			continue
		}
		if f.URL == "" {
			return fmt.Errorf("broadcast.rdioscanner[%d]: url required", i)
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.rdioscanner[%d]: api_key required", i)
		}
		if f.SystemID == 0 {
			return fmt.Errorf("broadcast.rdioscanner[%d]: system_id required", i)
		}
	}
	for i, f := range b.OpenMHz {
		if !f.Enabled {
			continue
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.openmhz[%d]: api_key required", i)
		}
		if f.ShortName == "" {
			return fmt.Errorf("broadcast.openmhz[%d]: short_name required", i)
		}
	}
	for i, f := range b.Icecast {
		if !f.Enabled {
			continue
		}
		if f.Host == "" {
			return fmt.Errorf("broadcast.icecast[%d]: host required", i)
		}
		if f.Port == 0 {
			return fmt.Errorf("broadcast.icecast[%d]: port required", i)
		}
		if f.Password == "" {
			return fmt.Errorf("broadcast.icecast[%d]: password required", i)
		}
	}
	for i, f := range b.Webhook {
		if !f.Enabled {
			continue
		}
		if f.URL == "" {
			return fmt.Errorf("broadcast.webhook[%d]: url required", i)
		}
	}
	for i, f := range b.GrantWebhook {
		if !f.Enabled {
			continue
		}
		if f.URL == "" {
			return fmt.Errorf("broadcast.grant_webhook[%d]: url required", i)
		}
	}
	return nil
}

// parseDurationFlexible accepts a Go duration string. Wrapped here so
// the dependency lives in one place and tests can lean on it.
func parseDurationFlexible(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// decodeHexKey parses a hex-encoded encryption key. Surrounding and
// internal whitespace plus an optional "0x"/"0X" prefix are stripped
// so operators can paste keys in whatever form their radio-programming
// software displays them.
func decodeHexKey(s string) ([]byte, error) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, s)
	clean = strings.TrimPrefix(clean, "0x")
	clean = strings.TrimPrefix(clean, "0X")
	if clean == "" {
		return nil, errors.New("key is empty")
	}
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("key is not valid hex: %w", err)
	}
	return b, nil
}
