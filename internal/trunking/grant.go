package trunking

import (
	"fmt"
	"time"
)

// Grant is the protocol-agnostic voice-channel grant payload published on
// the events bus by P25/DMR/NXDN control-channel decoders. The trunking
// engine subscribes to events of kind events.KindGrant and dispatches
// them through the priority + voice-device pool.
//
// FrequencyHz must be filled in by the protocol layer (P25 derives it from
// IdentifierUpdate band-plan TSBKs, DMR/NXDN from the configured System).
// If FrequencyHz is zero, the engine logs and drops the grant.
type Grant struct {
	System      string // System name, matches trunking.System.Name
	Protocol    string // "p25" / "dmr" / "nxdn"
	GroupID     uint32 // talkgroup or destination subscriber address
	SourceID    uint32 // originator (subscriber unit)
	FrequencyHz uint32 // voice channel frequency
	ChannelID   uint8  // raw channel ID (P25 band-plan ID, DMR LCN high)
	ChannelNum  uint16 // raw channel number within the ID
	// RFSSID and SiteID identify the P25 site whose control channel
	// granted the call, decoded from the site's RFSS Status Broadcast
	// (TSBK 0x3A) and accumulated in the control channel's NetworkModel.
	// They let downstream consumers (Prometheus exporters, dashboards)
	// label a grant by site instead of resolving the opaque ChannelID
	// against a manual lookup table (issue #698). Both stay zero on
	// non-P25 grants and until the first RFSS Status TSBK has landed.
	RFSSID uint8
	SiteID uint8
	// NAC is the control channel's Network Access Code (Phase 2: the
	// Network Status Broadcast Color Code, equal to the Phase 1 NAC per
	// spec). It is a coarse sanity-check field — not unique per site —
	// so site labelling must key on (RFSSID, SiteID) (issue #698). Zero
	// on non-P25 grants.
	NAC uint16
	// Timeslot identifies the TDMA logical channel a call occupies on
	// its carrier, 1-based: 0 = not applicable / unknown (P25 Phase 1,
	// NXDN, analog — frequency alone identifies the call), 1 = TS1,
	// 2 = TS2. DMR Tier III carries two independent calls on one
	// 12.5 kHz carrier (TS1 + TS2), so the engine treats
	// (FrequencyHz, Timeslot) as the call identity rather than
	// frequency alone. Populated by the protocol layer; the DMR CSBK
	// parser maps its 0-based slot bit (0 = TS1, 1 = TS2) onto this
	// 1-based convention so 0 stays reserved for "no slot".
	Timeslot  uint8
	Encrypted bool
	Emergency bool
	// DMRInterleavedVoice mirrors the system-level
	// trunking.System.DMRInterleavedVoice opt-in onto the grant so the
	// voice composer selects the 2-slot interleaved decoder and routes
	// this call to its timeslot by matching the embedded Link Control's
	// talkgroup to GroupID. Set by the DMR Tier III control channel;
	// false (the default) keeps the single-slot decoder. Ignored for
	// non-DMR grants.
	DMRInterleavedVoice bool
	// TETRAColourExt is the cell's 30-bit extended colour code (MCC/MNC/CC),
	// the scrambler seed the voice chain descrambles the granted call's traffic
	// frames with. Zero on non-TETRA grants and until the colour code is learned.
	TETRAColourExt uint32
	// TETRAUsageMarker is the call's downlink usage marker (ETSI EN 300 392-2
	// §21.4.7), the per-slot control-vs-traffic identifier the AACH broadcasts in
	// every downlink slot. On a same-carrier SCBS it — not the unreliable
	// channel-allocation timeslot field — is what maps a granted call to the
	// physical TDMA slot carrying its speech, so the voice chain demultiplexes
	// concurrent calls by matching this against each burst's AACH marker. Traffic
	// markers are >= 4 (DLUsageTraffic); 0 means "unknown" (grant addressed by SSI
	// without a usage marker), which falls the voice chain back to CRC-gated
	// single-call isolation. Zero on non-TETRA grants.
	TETRAUsageMarker uint8
	// AlgorithmID and KeyID carry the encryption parameters the
	// protocol's privacy header advertises (the DMR PI header, etc.).
	// They are meaningful only when Encrypted is true and stay zero
	// until a privacy header has been parsed. Persisted to the call
	// log so an operator can see which key a recorded call needs.
	AlgorithmID uint8
	KeyID       uint16
	DataCall    bool // false = voice call (default)
	// Individual marks a grant whose GroupID is NOT a talkgroup but an
	// individual destination — a unit-to-unit target radio (WUID), a
	// telephone-interconnect target, or an SNDCP data unit. P25 group
	// talkgroups are 16-bit; these targets are 24-bit subscriber addresses,
	// so recording them as "talkgroups" produces phantom >16-bit entries
	// (e.g. a unit-to-unit call surfacing as bogus TG 140957). Discovery /
	// hunt skips individual grants when building the talkgroup list. False
	// (the default) for group calls and Motorola patch grants, which ARE
	// talkgroups.
	Individual bool
	// ProVoice marks the grant as an EDACS ProVoice (digital) call. The
	// vocoder is patent + trade-secret encumbered so we cannot ship a
	// built-in decoder; the recorder treats this flag as a directive to
	// emit a `.raw` frame sidecar regardless of its global WriteRaw
	// setting, so researchers can decode out-of-band.
	ProVoice bool
	// PatchedGroups, when non-empty, lists the member talkgroups of a
	// patch / dynamic-regroup super-group: the call on GroupID is
	// physically the shared traffic of these groups. The engine fills
	// it from its PatchRegistry so the call can be attributed to every
	// member. Empty for an ordinary (non-patched) grant.
	PatchedGroups []uint32
	// P25Phase1DemodMode mirrors the system-level
	// trunking.System.P25Phase1DemodMode setting so the voice composer
	// can pick the matching symbol-recovery path on grants for the
	// system (C4FM vs CQPSK / LSM). The control-channel decoder
	// already honours the setting via the ccdecoder connector; without
	// this field every voice grant landed in a hardcoded C4FM voice
	// receiver and never decoded on CQPSK/LSM-modulated sites
	// (issue #356 follow-up). Populated by the protocol layer when it
	// publishes the grant; ignored for non-P25-Phase-1 grants.
	P25Phase1DemodMode string
	// P25Phase2Decode carries the per-channel FEC parameters the voice
	// composer's Phase 2 chain needs to decode MAC PDUs that
	// interleave with voice subframes on the traffic channel (talker
	// alias, in-call signalling). Populated by the Phase 2 control
	// channel when publishing the grant; zero on non-Phase-2 grants.
	P25Phase2Decode P25Phase2Decode
	At              time.Time
	// CallID is a process-monotonic identifier the voice pool assigns when
	// it binds a device to this grant (VoicePool.Bind); a real handoff
	// (VoicePool.Retune) preserves it so a followed call keeps one identity.
	// It lets the downstream voice chain tag each decoded frame with the
	// call it belongs to and the recorder reject a frame whose CallID
	// doesn't match the open session — closing the cross-call audio-bleed
	// window when a voice-tap serial is immediately reused for the next
	// call. Zero on grants that never went through the voice pool (synthetic
	// / conventional follows); a zero-vs-zero match is a no-op, preserving
	// prior behaviour.
	CallID uint64
}

// P25Phase2Decode is the protocol-neutral mirror of
// p25/phase2.MACDecodeConfig: primitive fields so this struct lives
// in the trunking package without pulling a Phase 2 import. The
// composer translates back to phase2.MACDecodeConfig at use time.
//
// Trellis / RS / Interleave / Scrambler are numerically aligned to
// the phase2 enum constants of the same name (TrellisOff = 0,
// TrellisOn = 1, etc.). The composer round-trips them by casting.
type P25Phase2Decode struct {
	Trellis    uint8
	RS         uint8
	Interleave uint8
	Scrambler  uint8
	Seed       uint64
	// SoftDecision mirrors phase2.MACDecodeConfig.SoftDecision: when set,
	// the voice composer / sigfollow build the Phase 2 traffic-channel
	// receiver with soft-decision demod (per-bit soft Viterbi on the MAC
	// trellis) instead of the hard slicer, recovering ~1.5-2 dB of coding
	// gain on weak signals (issue #915). Default false keeps today's hard
	// path byte-for-byte.
	SoftDecision bool
}

// String renders a one-line summary of a Grant for log output.
func (g Grant) String() string {
	flags := ""
	if g.Encrypted {
		flags += "E"
		// Append ALGID / KID once the in-call signalling has surfaced
		// them so the log line is self-describing for operators
		// triaging encrypted traffic. Zero values are suppressed
		// because a Phase 1 grant publishes before the LDU2 lands.
		if g.AlgorithmID != 0 || g.KeyID != 0 {
			flags += fmt.Sprintf("(alg=0x%02X,key=0x%04X)", g.AlgorithmID, g.KeyID)
		}
	}
	if g.Emergency {
		flags += "!"
	}
	if g.DataCall {
		flags += "D"
	}
	if g.ProVoice {
		flags += "P"
	}
	ts := ""
	if g.Timeslot != 0 {
		ts = fmt.Sprintf(" ts%d", g.Timeslot)
	}
	return fmt.Sprintf("%s/%s tg=%d src=%d freq=%d%s %s", g.System, g.Protocol, g.GroupID, g.SourceID, g.FrequencyHz, ts, flags)
}

// EndReason classifies why a call ended; carried in CallEnd events so the
// API layer can surface the cause to UIs.
type EndReason uint8

const (
	EndReasonUnknown EndReason = iota
	// EndReasonNormal is the carrier-drop natural end: either the CC
	// announced a channel release / talk-off, or — far more common
	// on P25 where no such announcement is ever sent — the watchdog
	// reaped a call whose Touch advanced past StartedAt (frames were
	// decoded and then the transmitter stopped). Operator-visible
	// meaning: the call ended cleanly, no decode problem.
	EndReasonNormal
	// EndReasonTimeout is the silent-from-start decode failure: the
	// watchdog reaped a call whose LastHeardAt never moved past
	// StartedAt — not a single LDU / voice subframe was delivered.
	// This is the real failure mode (wrong demod mode, gain too low,
	// LSM site decoded as C4FM, etc.) — distinct from EndReasonNormal
	// above, which fires when the radio simply stopped transmitting.
	EndReasonTimeout
	EndReasonPreempted  // higher-priority grant kicked us off
	EndReasonLockout    // talkgroup is locked out by policy
	EndReasonNoVoiceSDR // every Voice-role SDR was busy
	EndReasonError
	EndReasonManual // operator ended the call via API / TUI
	// EndReasonEncrypted is the encrypted-call-handling teardown: the
	// trunking.encrypted_calls policy released the voice SDR because the
	// call was discovered to be encrypted — either immediately
	// (mode: ignore) or after the metadata-follow window (mode:
	// metadata). Distinct from the reasons above so operators can see a
	// tuner was freed by policy, not by a decode problem or a higher-
	// priority preemption. Issue #711.
	EndReasonEncrypted
)

func (r EndReason) String() string {
	switch r {
	case EndReasonNormal:
		return "normal"
	case EndReasonTimeout:
		return "timeout"
	case EndReasonPreempted:
		return "preempted"
	case EndReasonLockout:
		return "lockout"
	case EndReasonNoVoiceSDR:
		return "no-voice-sdr"
	case EndReasonError:
		return "error"
	case EndReasonManual:
		return "manual"
	case EndReasonEncrypted:
		return "encrypted"
	default:
		return "unknown"
	}
}

// CallStart is the payload of an events.KindCallStart event. The engine
// publishes this once a Voice device has been retuned to the grant's
// frequency; downstream pipelines (the demod composer, the recorder)
// subscribe and start consuming IQ.
type CallStart struct {
	Grant        Grant
	Talkgroup    *TalkGroup // resolved via the engine's TalkgroupDB; nil if unknown
	DeviceSerial string     // which Voice SDR is following the call
	StartedAt    time.Time
}

// CallSegment is the payload of an events.KindCallSegment event. The
// voice composer publishes it at an end-of-transmission boundary when
// per-transmission recording is enabled, so the recorder closes the
// current file and starts a fresh one for the next over. At marks the
// boundary instant; the recorder uses it as the new segment's start
// timestamp.
type CallSegment struct {
	DeviceSerial string
	At           time.Time
}

// CallEnd is the payload of an events.KindCallEnd event.
type CallEnd struct {
	Grant        Grant
	Talkgroup    *TalkGroup
	DeviceSerial string
	StartedAt    time.Time
	EndedAt      time.Time
	Reason       EndReason
	// SignalDbFS is the call's mean received channel power in dBFS,
	// measured by the voice composer over the call's baseband IQ. nil when
	// no measurement was taken (calls ended by the watchdog, preemption, or
	// shutdown, or decoded outside the composer). It is a channel-power /
	// RSSI-style figure — NOT calibrated absolute RSSI and NOT SNR/EVM.
	SignalDbFS *float64
	// EVMPct and SNRDb are the call's demod quality — RMS error-vector
	// magnitude (%) and estimated symbol SNR (dB) — measured by the voice
	// composer over the settled decode. Unlike SignalDbFS these ARE the
	// demod-quality figures to compare against another decoder (SDRTrunk).
	// nil when unmeasured — currently only P25 Phase 1 chains feed the demod
	// taps that populate them (issue #878 follow-up).
	EVMPct *float64
	SNRDb  *float64
}

// Duration returns how long the call ran.
func (c CallEnd) Duration() time.Duration { return c.EndedAt.Sub(c.StartedAt) }

// CallComplete is the payload of an events.KindCallComplete event. The
// recorder publishes it once a call's WAV has been flushed and closed,
// so the outbound-streaming subsystem (internal/broadcast) can read the
// finished file and upload it to call aggregators. AudioPath is the
// absolute or working-directory-relative path to the .wav the recorder
// wrote; SampleRate is its PCM rate in Hz.
type CallComplete struct {
	Grant        Grant
	Talkgroup    *TalkGroup
	DeviceSerial string
	StartedAt    time.Time
	EndedAt      time.Time
	Reason       EndReason
	AudioPath    string
	SampleRate   uint32
}

// Duration returns how long the call ran.
func (c CallComplete) Duration() time.Duration { return c.EndedAt.Sub(c.StartedAt) }

// CallEncryption is the payload of an events.KindCallEncryption event.
// It is published by the voice composer when an in-call Encryption Sync
// is recovered (P25 Phase 1 LDU2 carries it; the grant TSBK has only
// the encrypted flag). DeviceSerial keys the update to a specific
// active call; the engine backfills the bound ActiveCall.Grant's
// AlgorithmID / KeyID and republishes the event with System / Protocol
// / GroupID populated so SSE + TUI consumers can patch their live
// view without re-deriving the call's identity.
//
// MessageIndicator carries the 72-bit per-call cryptographic sync
// vector — not surfaced in any DTO today, but plumbed through for
// future key-discovery tooling.
type CallEncryption struct {
	DeviceSerial     string
	System           string // filled in by the engine on republish
	Protocol         string
	GroupID          uint32
	AlgorithmID      uint8
	KeyID            uint16
	MessageIndicator [9]byte
	At               time.Time
}

// CallSourceUpdate is the payload of an events.KindCallSourceUpdate
// event. The voice composer publishes one when it recovers the
// source radio ID + encryption state from in-call traffic-channel
// signalling — e.g. a P25 Phase 2 GROUP_VOICE_CHANNEL_USER PDU
// where the CC grant arrived in a compressed form without those
// fields (src=0 + enc=false). The engine subscribes, backfills the
// bound ActiveCall.Grant.SourceID + .Encrypted via the voice pool,
// and republishes the event with System / Protocol / GroupID
// populated so SSE + TUI consumers can patch their live view.
// DeviceSerial keys the update to a specific active call.
type CallSourceUpdate struct {
	DeviceSerial string
	System       string // filled in by the engine on republish
	Protocol     string
	GroupID      uint32 // filled in by the engine on republish from the bound Grant
	SourceID     uint32
	Encrypted    bool
	At           time.Time
}

// SiteUpdate is the payload of an events.KindSiteUpdate event. The P25
// control-channel decoder publishes one each time it parses an RFSS
// Status Broadcast (TSBK 0x3A), naming the site it is currently camped
// on and the control-channel frequency it heard it on. The SiteTracker
// accumulates these into the queryable table behind GET /api/v1/sites
// (issue #698). ControlChannelHz comes from the decoder's tuned
// frequency — grants carry only the voice ChannelID, not the control
// channel — so a SiteUpdate is the only place the two are joined.
type SiteUpdate struct {
	System           string
	RFSSID           uint8
	SiteID           uint8
	ControlChannelHz uint32
	// ControlChannelCarrierOffsetHz is the demodulator's measured carrier
	// offset (signed Hz) of the locked control carrier relative to the tuned
	// centre, as of this update. A large value means the locked carrier sits
	// well off the configured frequency — e.g. a stronger neighbouring site
	// bleeding through at 12.5 kHz spacing (issue #815) — so the reported site
	// identity may not belong to the configured frequency. Zero on demod paths
	// without an AFC stage (CQPSK) or when the offset is genuinely ~0.
	ControlChannelCarrierOffsetHz int32 `json:"control_channel_carrier_offset_hz,omitempty"`
	// ControlChannelTSBKErrorRate is the percentage (0..100) of TSBK blocks on
	// this control channel that failed Viterbi or CRC, cumulative since the
	// decoder started — a frame-error rate, not a pre-FEC bit-error rate. It
	// tracks decode quality independently of carrier lock: a well-locked carrier
	// at range can still show a high rate (issue #858). ControlChannelTSBKCount
	// is the total attempts the rate was measured over (the denominator, and a
	// confidence weight); zero means no TSBK has been attempted yet — a fresh
	// lock, or a non-TSBK path such as Phase 2 TDMA — and the rate is undefined.
	ControlChannelTSBKErrorRate float64 `json:"control_channel_tsbk_error_rate,omitempty"`
	ControlChannelTSBKCount     int64   `json:"control_channel_tsbk_count,omitempty"`
	WACN                        uint32
	SystemID                    uint16
	// Hex renderings of the identity numbers (P25 field convention),
	// alongside the decimal fields above so JSON consumers get both. Empty
	// when the corresponding value is unknown (zero).
	WACNHex     string `json:"WACNHex,omitempty"`
	SystemIDHex string `json:"SystemIDHex,omitempty"`
	RFSSIDHex   string `json:"RFSSIDHex,omitempty"`
	SiteIDHex   string `json:"SiteIDHex,omitempty"`
	// Topology, when non-nil, is the full network-configuration snapshot of the
	// camped site (identity + primary/secondary control channels + neighbours +
	// band plan) as of this update. It is built on the decoder's Process
	// goroutine and handed off here, so the SiteTracker can render the live
	// network-configuration report without touching the decoder's internals.
	Topology *TopologySnapshot
	At       time.Time
}
