// Package events implements an in-process pub/sub bus used by the engine to
// publish trunking events. A separate API surface (gRPC, WebSocket) subscribes
// without coupling to the engine.
package events

import (
	"sync"
	"sync/atomic"
	"time"
)

type Kind string

const (
	KindSDRAttached Kind = "sdr.attached"
	KindSDRDetached Kind = "sdr.detached"
	KindCCLocked    Kind = "cc.locked"
	KindCCLost      Kind = "cc.lost"
	KindCallStart   Kind = "call.start"
	KindCallEnd     Kind = "call.end"
	// KindCallComplete fires after the recorder has finished writing a
	// call's WAV to disk — the WAV header length fields are patched and
	// the file is closed and ready to read. It carries the same call
	// metadata as KindCallEnd plus the on-disk audio path
	// (trunking.CallComplete). The outbound-streaming subsystem
	// (internal/broadcast) subscribes to it to upload completed calls
	// to Broadcastify Calls / RdioScanner / OpenMHz / Icecast. Distinct
	// from KindCallEnd because KindCallEnd fires the instant the engine
	// releases the voice channel, before the WAV is flushed.
	KindCallComplete Kind = "call.complete"
	// KindCallSegment fires when the voice composer detects an
	// end-of-transmission boundary (a P25 terminator, a DMR voice
	// terminator, an FM squelch gap, …) within an active call and the
	// system is configured to split recordings per transmission
	// (trunking.voice_call_grouping = "transmission"). The recorder
	// finalizes the current WAV/.raw (and emits KindCallComplete for it)
	// and rolls to a fresh file for the next over — without ending the
	// engine call, so a same-talkgroup re-key that doesn't re-grant is
	// still captured. Payload is trunking.CallSegment. No-op in
	// "conversation" grouping (the composer never emits it).
	KindCallSegment Kind = "call.segment"
	KindGrant       Kind = "grant"
	// KindGrantUnserved fires when a grant was accepted (passed lockout /
	// scan / encryption gates) but no free voice device could be bound to
	// follow it — the "no voice device available for grant" overload moment.
	// Payload is trunking.Grant. Consumed by the event-driven IQ auto-recorder
	// (baseband.auto_record on_no_voice_device) to capture the raw IQ of the
	// carrier at the instant the system was too busy to follow a call.
	KindGrantUnserved Kind = "grant.unserved"
	KindToneAlert     Kind = "tone.alert"
	KindDecodeError   Kind = "decode.error"
	KindError         Kind = "error"
	// Scanner subsystem (internal/scanner/cchunt):
	//   KindHuntProgress fires once per CC candidate the hunter
	//     tries — payload identifies which system + frequency +
	//     position in the candidate list, so the TUI can render
	//     "trying 851.012500 MHz (2/3)".
	//   KindHuntFailed fires when a system exhausts its CC list
	//     without locking; payload carries the next backoff window
	//     so operators can see "retry in 5 s".
	KindHuntProgress Kind = "cchunt.progress"
	KindHuntFailed   Kind = "cchunt.failed"
	// Live system-discovery ("hunt") events, published by the daemon's hunt
	// Manager (internal/hunt). Distinct from the cchunt.* control-channel
	// hunter above: these track a blind discovery run (spectrum sweep →
	// identify → map). Payloads are hunt package types carried as `any`.
	//   KindHuntLiveProgress fires as the sweep/identify phases advance.
	//   KindHuntLiveCandidate fires once per candidate carrier mapped.
	//   KindHuntLiveDone fires when a run finishes (or is stopped).
	KindHuntLiveProgress  Kind = "hunt.progress"
	KindHuntLiveCandidate Kind = "hunt.candidate"
	KindHuntLiveDone      Kind = "hunt.done"
	// KindAffiliation fires when a radio unit affiliates with a
	// talkgroup. P25 control-channel publishes one per Group
	// Affiliation Response TSBK (opcode 0x28); the payload identifies
	// the source unit, the group it's joining, and the response code
	// (accepted / denied / refused / failed) from the system. Useful
	// downstream as a "who is listening where" feed for telemetry
	// dashboards.
	//
	// KindUnitRegistration fires when a radio registers (or
	// deregisters) on a site. P25 control-channel publishes one per
	// Unit Registration Response TSBK (opcode 0x2C); the payload
	// identifies the source unit, the WACN + System ID it's
	// registering on, and the response code. Useful as a "which
	// radios are on which site" feed.
	//
	// KindUnitToUnitRequest fires when the system asks a radio whether it
	// will answer a private (unit-to-unit) call. P25 control-channel
	// publishes one per Unit-to-Unit Answer Request TSBK (opcode 0x05); the
	// payload identifies the calling (source) and called (target) units. No
	// channel is granted yet — it's the call-setup handshake. Useful as a
	// "who is calling whom" feed alongside affiliations.
	KindAffiliation       Kind = "affiliation"
	KindUnitRegistration  Kind = "registration"
	KindUnitToUnitRequest Kind = "unit.request"
	// KindAudioState fires when an operator changes the live-audio
	// cockpit — volume, mute, or recording-gate. The payload is the
	// new state (the same shape as GET /api/v1/audio). Subscribers
	// can re-render volume sliders / mute indicators instantly
	// instead of waiting for the next 3 s poll tick. Emitted by
	// the HTTP API's PATCH /api/v1/audio handler.
	KindAudioState Kind = "audio.state"
	// KindPatch fires when a trunked system announces (or cancels) a
	// patch / dynamic-regroup — a super-group that merges several
	// talkgroups onto one channel. P25 Phase 2 publishes one per
	// Motorola group-regroup or Harris regroup MAC PDU. The payload
	// (trunking.Patch) carries the super-group, its member talkgroups,
	// and whether the patch is being added or removed.
	KindPatch Kind = "patch"
	// KindTalkerAlias fires when a radio's display name (its "talker
	// alias") has been fully reassembled from the multi-fragment vendor
	// MAC PDUs that carry it. P25 Phase 2 publishes one per completed
	// alias. The payload (trunking.TalkerAlias) is keyed by source unit
	// so a consumer can associate it with the active call.
	KindTalkerAlias Kind = "talker.alias"
	// KindLocation fires when a subscriber unit reports a geographic
	// position over the air — P25 Motorola Unit GPS, P25 L3Harris
	// Talker GPS, or DMR LRRP. The payload (trunking.Location) carries
	// the reporting radio, the lat/lon fix, and optional speed/heading.
	// The storage layer persists it; the API + web map surface it.
	KindLocation Kind = "location"
	// KindCallEncryption fires when the voice composer recovers an
	// Encryption Sync from the in-call signalling — e.g. a P25 Phase 1
	// LDU2 Encryption Sync, where ALGID/KID are only available after
	// the call has started (the grant TSBK carries only the encrypted
	// bit). The engine subscribes, backfills the bound ActiveCall's
	// Grant so downstream consumers (storage, SSE, TUI) see the
	// real algorithm/key on CallEnd, and republishes the event with
	// the call's system / protocol / group context so live listeners
	// (SSE, TUI) can patch the active row in flight. P25 Phase 2 and
	// any protocol whose grant carries ALGID/KID at grant time does
	// not need this event — the values are already on the Grant.
	KindCallEncryption Kind = "call.encryption"
	// KindCallSourceUpdate fires when the voice composer recovers the
	// source radio ID + encryption state for an active call from the
	// traffic-channel signalling — e.g. a P25 Phase 2
	// GROUP_VOICE_CHANNEL_USER PDU, where the CC grant arrived in a
	// compressed form without SOURCE_ID or SVC_OPTIONS. The engine
	// subscribes, backfills the bound ActiveCall's Grant
	// (SourceID + Encrypted) so downstream consumers (storage, SSE,
	// TUI, affiliation tracker) see the real source on CallEnd and
	// during the live call. Protocols whose grant always carries
	// source ID + encrypted state at grant time do not need this
	// event — the values are already on the Grant.
	KindCallSourceUpdate Kind = "call.source"
	// KindCallRelease fires when a protocol control-channel decoder sees an
	// explicit call-teardown message (e.g. a TETRA CMCE D-RELEASE) rather than
	// inferring the end from a silence timeout. Payload is a
	// trunking.CallRelease keyed by (System, GroupID); the engine ends the
	// matching active call at once instead of waiting out the hangtime/no-voice
	// timers.
	KindCallRelease Kind = "call.release"
	// KindCallTalker fires when a control-channel decoder resolves the current
	// transmitting party of an active call (e.g. a TETRA CMCE D-TX-GRANTED
	// carrying the transmitting party SSI). Payload is a trunking.CallTalker
	// keyed by (System, GroupID); the engine backfills the bound call's source
	// via the same path as KindCallSourceUpdate.
	KindCallTalker Kind = "call.talker"
	// KindSiteUpdate fires when the P25 control-channel decoder parses
	// an RFSS Status Broadcast (TSBK 0x3A), naming the site it is
	// camped on and the control-channel frequency it was heard on.
	// Payload is a trunking.SiteUpdate. The SiteTracker subscribes and
	// accumulates the discovered sites behind GET /api/v1/sites so
	// downstream tooling can map a grant's RFSS/Site to a human
	// site name (issue #698).
	KindSiteUpdate Kind = "site.update"
	// KindBookmarkCreated / KindBookmarkUpdated / KindBookmarkDeleted
	// fire whenever the bookmarks store mutates. Payload is a
	// storage.Bookmark (or {ID} for deletes). Surfaced over SSE / WS
	// so the web SPA + TUI can refresh their bookmark list without
	// polling.
	KindBookmarkCreated Kind = "bookmark.created"
	KindBookmarkUpdated Kind = "bookmark.updated"
	KindBookmarkDeleted Kind = "bookmark.deleted"
	// KindLabelUpserted / KindLabelDeleted fire when an operator names a
	// radio or a talkgroup from a UI. Payload is a storage.Label. Surfaced
	// over SSE / WS so other open consoles pick the name up without a
	// reload — a label is usually applied while watching live traffic.
	KindLabelUpserted Kind = "label.upserted"
	KindLabelDeleted  Kind = "label.deleted"
	// KindPagerMessage fires when the POCSAG decoder finishes
	// reassembling one page (address codeword + 0..N message
	// codewords, terminated by an idle or next address). Payload
	// is a storage.PagerMessage carrying RIC + function +
	// decoded text + per-page bit-error count. Surfaced over SSE /
	// WS for the live pager panel.
	KindPagerMessage Kind = "pager.message"
	// KindAPRSPacket fires when the APRS decoder finishes one
	// frame off the air — UI frame decoded, info field parsed
	// into the appropriate APRS sub-type (position, message,
	// status, bulletin, status, etc.). Payload is a
	// storage.APRSPacket carrying source + destination + path +
	// decoded sub-payload. Surfaced over SSE / WS for the live
	// APRS panel.
	KindAPRSPacket Kind = "aprs.packet"
	// KindAISMessage fires when the AIS decoder finishes one
	// message off the marine VHF channels (161.975 / 162.025 MHz).
	// One bus event per parsed AIS message (positions, static +
	// voyage data, base-station reports). Payload is a
	// storage.AISMessage carrying MMSI + message type + decoded
	// position + speed + course + vessel name (where available).
	// Surfaced over SSE / WS for the live AIS panel.
	KindAISMessage Kind = "ais.message"
	// KindDSCMessage fires when the DSC decoder finishes one
	// sequence off the marine VHF channel-70 (156.525 MHz) DSC
	// channel or one of the HF DSC channels. Payload is a
	// storage.DSCMessage carrying source / target MMSI + format
	// + category + (for distress alerts) position + nature.
	// Surfaced over SSE / WS for the live DSC panel.
	KindDSCMessage Kind = "dsc.message"
	// KindAircraftReport fires when the ADS-B decoder finishes
	// one Mode-S frame off the 1090 MHz aviation transponder
	// channel. Payload is a storage.AircraftReport carrying the
	// 24-bit ICAO address plus the message-kind-specific fields
	// (identification → callsign + category, airborne position
	// → CPR-decoded lat/lon + altitude, velocity → ground speed
	// + track + vertical rate). Surfaced over SSE / WS for the
	// live ADS-B panel.
	KindAircraftReport Kind = "adsb.aircraft"

	// KindMDC1200Message fires when the MDC1200 decoder completes one
	// signaling burst off a conventional analog FM voice channel.
	// Payload is a storage.MDC1200Message carrying the transmitting
	// radio's unit ID plus the operation (PTT ID, emergency, status,
	// radio check, ...) and the CRC-valid flag. Surfaced over SSE / WS
	// for the live MDC1200 panel.
	KindMDC1200Message Kind = "mdc1200.message"

	// KindM17LinkSetup fires when the M17 decoder reassembles a Link
	// Setup Frame (via the stream-frame LICH path). Payload is a
	// storage.M17LinkSetup carrying source / destination callsigns, the
	// mode (voice / data / packet), channel-access number, and the
	// CRC-valid flag. Surfaced over SSE / WS for the live M17 panel.
	KindM17LinkSetup Kind = "m17.linksetup"

	// KindLoRaFrame fires when the LoRa decoder recovers one PHY frame off
	// a configured sub-channel. Payload is a storage.LoRaFrame carrying the
	// spreading factor, coding rate, bandwidth, RSSI/SNR, CRC-valid flag and
	// the (de-whitened) payload bytes, plus any LoRaWAN MAC fields decoded
	// from it. Surfaced over SSE / WS for the live LoRa panel.
	KindLoRaFrame Kind = "lora.frame"

	// KindDMRGrantObserved fires for every DMR Tier III voice-grant CSBK
	// the control channel decodes, published BEFORE the LCN is resolved to
	// a downlink frequency. It is emitted in addition to (not instead of)
	// KindGrant on success / the no-bandplan KindDecodeError on failure, so
	// the DMR LCN autoconfig learner (internal/scanner/dmrlcn) can observe
	// the granted LCN even when no band plan is configured yet — which is
	// exactly the case it is trying to fix. Payload is DMRGrantObserved.
	KindDMRGrantObserved Kind = "dmr.grant.observed"

	// KindChannelPower fires once per diagnostics window (iqpower.Window,
	// ~1 s) for each wideband channel that decoded at least one frame that
	// window — the wideband engine (internal/scanner/widebandt2) gates it
	// on per-protocol decode activity so idle / off-band channels stay
	// silent. Payload is ChannelPower, carrying the window's mean IQ power
	// and a LowPower flag. The optional power log (internal/log.PowerLog)
	// subscribes to persist a per-channel signal-level record; by default
	// it writes only LowPower windows (the "decoding but weak" diagnostic).
	KindChannelPower Kind = "channel.power"

	// KindDMRBandPlanLearned fires once the DMR LCN autoconfig learner has
	// fit a band plan for a system from observed (LCN, frequency) pairs.
	// The learner has already hot-swapped the resolver into the running
	// control channel by the time this is published; subscribers use it for
	// logging, operator surfacing, and optional config writeback. Payload is
	// DMRBandPlanLearned.
	KindDMRBandPlanLearned Kind = "dmr.bandplan.learned"
)

// Stage names a particular FEC / parser checkpoint inside a protocol
// decoder. Stages are used as Prometheus label values, so the wire
// shapes below are part of the Stage's public contract — extend the
// const block, don't rename existing entries.
type Stage string

// Known decode stages. Add new ones here and reference them from the
// publishing protocol package; the events bus itself stays neutral.
const (
	StageNIDBCH          Stage = "nid-bch"          // P25 Phase 1 NID BCH(63,16,11)
	StageTSBKTrellis     Stage = "tsbk-trellis"     // P25 Phase 1 TSBK ½-rate trellis
	StageTSBKCRC         Stage = "tsbk-crc"         // P25 Phase 1 TSBK CRC trailer
	StageNoBandPlan      Stage = "no-bandplan"      // Voice grant arrived for an unknown channel ID / LCN
	StageSlotTypeHamming Stage = "slottype-hamming" // DMR slot-type Hamming(20,8)
	StageVoiceHeaderBPTC Stage = "voiceheader-bptc" // DMR Tier II Voice LC Header BPTC(196,96)
	StageVoiceHeaderRS   Stage = "voiceheader-rs"   // DMR Tier II Voice LC Header RS(12,9,4)
	StageSACCHTrellis    Stage = "sacch-trellis"    // NXDN SACCH ½-rate trellis
)

// DecodeError is the payload published with KindDecodeError. Protocol
// packages publish this when an FEC primitive returns errCount == -1
// (or a parser short-circuits on a structural failure) so the metrics
// collector can increment gophertrunk_decode_errors_total without
// each package having to hold a *metrics.Metrics handle.
type DecodeError struct {
	Protocol string
	Stage    Stage
}

// DMRGrantObserved is the payload published with KindDMRGrantObserved.
// It carries the raw fields of a DMR Tier III voice-grant CSBK before
// the LCN is resolved to a frequency, so the LCN autoconfig learner can
// correlate the granted LCN with the RF carrier that subsequently keys
// up. Timeslot uses the raw CSBK encoding (0 = TS1, 1 = TS2).
type DMRGrantObserved struct {
	System    string
	ColorCode uint8
	LCN       uint16
	Timeslot  uint8
	GroupID   uint32
	SourceID  uint32
	CCFreqHz  uint32 // control-channel frequency that decoded this grant
	At        time.Time
}

// DMRBandPlanLearned is the payload published with KindDMRBandPlanLearned.
// A linear plan sets BaseHz/SpacingHz/Offset with a nil Table; an
// irregular plan leaves the linear fields zero and populates Table. The
// payload mirrors trunking.DMRBandPlan with primitive fields so the
// events package stays free of a trunking import (the learner translates
// its trunking-shaped fit result into this on publish).
type DMRBandPlanLearned struct {
	System     string
	BaseHz     uint32
	SpacingHz  uint32
	Offset     int8
	Table      []DMRBandPlanLCN // non-nil only for an irregular (table) plan
	NumPairs   int
	Confidence float64
	ResidualHz uint32
}

// DMRBandPlanLCN is one explicit LCN→downlink-frequency entry in a
// learned irregular (table) band plan.
type DMRBandPlanLCN struct {
	LCN    uint16
	FreqHz uint32
}

// ChannelPower is the payload published with KindChannelPower. It is a
// per-window snapshot of one wideband channel's signal level, emitted
// only for windows in which the channel decoded at least one frame.
// Defined here (primitive fields only) so both the publisher
// (internal/scanner/widebandt2) and the subscriber (internal/log) can
// share it without an import cycle through the trunking package.
type ChannelPower struct {
	System    string
	Protocol  string // "dmr-tier2", "dmr-tier3", "p25-phase1", "p25-phase2"
	FreqHz    uint32
	PowerDbFS float64
	Decoded   uint64 // frames decoded this window (per-window delta)
	LowPower  bool   // PowerDbFS < iqpower.LowPowerThresholdDbFS
	At        time.Time
}

type Event struct {
	Kind      Kind
	Timestamp time.Time
	Payload   any
}

type Bus struct {
	mu     sync.RWMutex
	subs   map[uint64]chan Event
	nextID atomic.Uint64
	buffer int
	closed bool
}

func NewBus(buffer int) *Bus {
	if buffer <= 0 {
		buffer = 64
	}
	return &Bus{subs: make(map[uint64]chan Event), buffer: buffer}
}

type Subscription struct {
	id uint64
	C  <-chan Event
	b  *Bus
}

func (b *Bus) Subscribe() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		ch := make(chan Event)
		close(ch)
		return &Subscription{C: ch}
	}
	id := b.nextID.Add(1)
	ch := make(chan Event, b.buffer)
	b.subs[id] = ch
	return &Subscription{id: id, C: ch, b: b}
}

func (s *Subscription) Close() {
	if s.b == nil {
		return
	}
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	if ch, ok := s.b.subs[s.id]; ok {
		delete(s.b.subs, s.id)
		close(ch)
	}
	s.b = nil
}

// Publish delivers e to every subscriber. Slow subscribers drop the event
// rather than blocking the publisher; we count drops via the returned int.
func (b *Bus) Publish(e Event) int {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	dropped := 0
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			dropped++
		}
	}
	return dropped
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, ch := range b.subs {
		close(ch)
		delete(b.subs, id)
	}
}
