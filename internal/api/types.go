package api

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// HuntStatus is the JSON shape returned by GET /api/v1/hunt — the live
// system-discovery run state plus the discovered system map once available.
type HuntStatus struct {
	RunID      int    `json:"run_id"`
	State      string `json:"state"`
	Running    bool   `json:"running"`
	Mode       string `json:"mode,omitempty"` // "hunt" | "survey"
	Phase      string `json:"phase,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Sites      int    `json:"sites"`
	Talkgroups int    `json:"talkgroups"`
	SystemName string `json:"system_name,omitempty"`
	// Signals is the classified-carrier inventory of a survey run (empty for a
	// plain hunt) — the survey's primary result.
	Signals []hunt.DetectedSignal  `json:"signals,omitempty"`
	Error   string                 `json:"error,omitempty"`
	System  *hunt.DiscoveredSystem `json:"system,omitempty"`
	Reports []hunt.CaptureReport   `json:"reports,omitempty"`
}

// HuntRRReport is the GET /api/v1/hunt/radioreference response: the
// RadioReference cross-reference of a run's discovered system — ranked
// duplicate-system hints and a frequency/talkgroup diff vs the strongest match.
type HuntRRReport struct {
	Hints    []hunt.DuplicateHint `json:"hints,omitempty"`
	Diff     *hunt.RRDiff         `json:"diff,omitempty"`
	Compared int                  `json:"compared"` // existing RR systems compared against
}

// HuntStartRequest is the POST /api/v1/hunt/start body. Frequencies are in MHz
// for operator convenience. With Bands set the hunt sweeps; with Candidates +
// NoSweep it probes the listed control channels directly.
type HuntStartRequest struct {
	Serial          string    `json:"serial,omitempty"`
	Bands           []string  `json:"bands,omitempty"`      // "low:high" MHz
	Candidates      []float64 `json:"candidates,omitempty"` // MHz
	NoSweep         bool      `json:"no_sweep,omitempty"`
	Survey          bool      `json:"survey,omitempty"`        // classify+decode every carrier, not just trunking CCs
	ClassifyOnly    bool      `json:"classify_only,omitempty"` // survey: classify, skip decoding
	MaxDwellSeconds float64   `json:"max_dwell_seconds,omitempty"`
	Protocol        string    `json:"protocol,omitempty"`
	DwellSeconds    float64   `json:"dwell_seconds,omitempty"`
	SweepDwellMs    int       `json:"sweep_dwell_ms,omitempty"`
	PeakThresholdDb float64   `json:"peak_threshold_db,omitempty"`
	MinSpacingHz    uint32    `json:"min_spacing_hz,omitempty"`
	FFTSize         int       `json:"fft_size,omitempty"`
	MinConfidence   float64   `json:"min_confidence,omitempty"`
	Name            string    `json:"name,omitempty"`
	State           string    `json:"state,omitempty"`
	County          string    `json:"county,omitempty"`
	Location        string    `json:"location,omitempty"`
}

// EventDTO is the JSON envelope for every event streamed to clients.
// Kind matches the events.Kind constant; Payload is the kind-specific
// body (one of the *DTO types below). A separate envelope keeps the
// wire format easy to consume from JS / browser frontends.
type EventDTO struct {
	Kind      string    `json:"kind"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// SystemDTO mirrors trunking.System for JSON.
type SystemDTO struct {
	Name            string   `json:"name"`
	Protocol        string   `json:"protocol"`
	ControlChannels []uint32 `json:"control_channels"`
	WACN            uint32   `json:"wacn,omitempty"`
	SystemID        uint16   `json:"system_id,omitempty"`
	RFSS            uint8    `json:"rfss,omitempty"`
	Site            uint8    `json:"site,omitempty"`

	// Per-protocol FEC opt-out surface. Empty strings indicate the
	// new spec-correct default is active (channel coding / FEC on
	// for every protocol). Non-empty values that parse to "off" /
	// "false" / "0" opt the operator into the legacy raw-bit path
	// per-protocol. The TUI Settings panel renders these so operators
	// can verify their config landed; runtime mutation is a follow-up
	// (currently requires editing config.yaml + restarting the
	// daemon).
	TETRAColourCode        uint32  `json:"tetra_colour_code,omitempty"`
	TETRAChannel           string  `json:"tetra_channel,omitempty"`
	TETRAChannelCoding     string  `json:"tetra_channel_coding,omitempty"`
	LTRFCSMode             string  `json:"ltr_fcs_mode,omitempty"`
	LTRManchesterMode      string  `json:"ltr_manchester_mode,omitempty"`
	P25Phase1DemodMode     string  `json:"p25_phase1_demod_mode,omitempty"`
	P25Phase2TrellisMode   string  `json:"p25_phase2_trellis_mode,omitempty"`
	P25Phase2RSMode        string  `json:"p25_phase2_rs_mode,omitempty"`
	P25Phase2ScramblerMode string  `json:"p25_phase2_scrambler_mode,omitempty"`
	NXDNViterbiMode        string  `json:"nxdn_viterbi_mode,omitempty"`
	NXDNDeviationHz        float64 `json:"nxdn_deviation_hz,omitempty"`
	EDACSBCHMode           string  `json:"edacs_bch_mode,omitempty"`
	MPT1327BCHMode         string  `json:"mpt1327_bch_mode,omitempty"`
	MPT1327CWSCTolerance   string  `json:"mpt1327_cwsc_tolerance,omitempty"`
	MotorolaBCHMode        string  `json:"motorola_bch_mode,omitempty"`
	// DMRBandPlan surfaces the active DMR Tier III LCN→frequency plan
	// (operator-configured, or learned by the autoconfig learner and
	// written back to config) so the web Systems panel can show how voice
	// grants resolve. Nil when the system has no DMR band plan. (#638)
	DMRBandPlan *DMRBandPlanDTO `json:"dmr_band_plan,omitempty"`
}

// DMRBandPlanDTO mirrors trunking.DMRBandPlan: exactly one of Linear or
// Table is populated.
type DMRBandPlanDTO struct {
	Linear *DMRLinearBandPlanDTO `json:"linear,omitempty"`
	Table  []DMRBandPlanLCNDTO   `json:"table,omitempty"`
}

// DMRLinearBandPlanDTO mirrors trunking.DMRLinearBandPlan — a regular
// base+spacing grid (freq = base + (lcn-offset)*spacing).
type DMRLinearBandPlanDTO struct {
	BaseHz    uint32 `json:"base_hz"`
	SpacingHz uint32 `json:"spacing_hz"`
	Offset    int8   `json:"offset,omitempty"`
}

// DMRBandPlanLCNDTO is one explicit LCN→downlink-frequency entry.
type DMRBandPlanLCNDTO struct {
	LCN    uint16 `json:"lcn"`
	FreqHz uint32 `json:"freq_hz"`
}

func dmrBandPlanToDTO(p *trunking.DMRBandPlan) *DMRBandPlanDTO {
	if p == nil {
		return nil
	}
	out := &DMRBandPlanDTO{}
	if p.Linear != nil {
		out.Linear = &DMRLinearBandPlanDTO{
			BaseHz:    p.Linear.BaseHz,
			SpacingHz: p.Linear.SpacingHz,
			Offset:    p.Linear.Offset,
		}
	}
	for _, e := range p.Table {
		out.Table = append(out.Table, DMRBandPlanLCNDTO{LCN: e.LCN, FreqHz: e.FreqHz})
	}
	return out
}

func systemToDTO(s trunking.System) SystemDTO {
	return SystemDTO{
		Name:                   s.Name,
		Protocol:               s.Protocol.String(),
		ControlChannels:        append([]uint32(nil), s.ControlChannels...),
		WACN:                   s.WACN,
		SystemID:               s.SystemID,
		RFSS:                   s.RFSS,
		Site:                   s.Site,
		TETRAColourCode:        s.TETRAColourCode,
		TETRAChannel:           s.TETRAChannel,
		TETRAChannelCoding:     s.TETRAChannelCoding,
		LTRFCSMode:             s.LTRFCSMode,
		LTRManchesterMode:      s.LTRManchesterMode,
		P25Phase1DemodMode:     s.P25Phase1DemodMode,
		P25Phase2TrellisMode:   s.P25Phase2TrellisMode,
		P25Phase2RSMode:        s.P25Phase2RSMode,
		P25Phase2ScramblerMode: s.P25Phase2ScramblerMode,
		NXDNViterbiMode:        s.NXDNViterbiMode,
		NXDNDeviationHz:        s.NXDNDeviationHz,
		EDACSBCHMode:           s.EDACSBCHMode,
		MPT1327BCHMode:         s.MPT1327BCHMode,
		MPT1327CWSCTolerance:   s.MPT1327CWSCTolerance,
		MotorolaBCHMode:        s.MotorolaBCHMode,
		DMRBandPlan:            dmrBandPlanToDTO(s.DMRBandPlan),
	}
}

// TalkgroupDTO mirrors trunking.TalkGroup for JSON.
type TalkgroupDTO struct {
	ID          uint32 `json:"id"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Lockout     bool   `json:"lockout,omitempty"`
	Scan        bool   `json:"scan"`
	Stream      bool   `json:"stream"`
	Record      bool   `json:"record"`
	Mute        bool   `json:"mute"`
	Icon        string `json:"icon,omitempty"`
}

func talkgroupToDTO(tg *trunking.TalkGroup) *TalkgroupDTO {
	if tg == nil {
		return nil
	}
	return &TalkgroupDTO{
		ID:          tg.ID,
		AlphaTag:    tg.AlphaTag,
		Description: tg.Description,
		Tag:         tg.Tag,
		Group:       tg.Group,
		Mode:        tg.Mode,
		Priority:    tg.Priority,
		Lockout:     tg.Lockout,
		Scan:        tg.Scan,
		Stream:      tg.Stream,
		Record:      tg.Record,
		Mute:        tg.Mute,
		Icon:        tg.Icon,
	}
}

// RIDDTO mirrors trunking.RID plus the live affiliation-tracker
// fields (last_seen, last_talkgroup, talker_alias, call_count). When
// a row is purely live (no configured static RID), the configured
// fields are zero / empty and Live is true.
type RIDDTO struct {
	ID          uint32 `json:"id"`
	Alias       string `json:"alias,omitempty"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Lockout     bool   `json:"lockout,omitempty"`
	Watch       bool   `json:"watch"`
	Icon        string `json:"icon,omitempty"`

	// Configured is true when this row is backed by an entry in the
	// static RIDDB (rid_alias_file). Used by the UI to distinguish
	// known radios from RIDs only ever seen over the air.
	Configured bool `json:"configured"`

	// Live observation fields — empty/zero when the RID has not been
	// seen since the daemon started (or since the affiliation tracker
	// swept it).
	System        string    `json:"system,omitempty"`
	Protocol      string    `json:"protocol,omitempty"`
	LastTalkgroup uint32    `json:"last_talkgroup,omitempty"`
	TalkerAlias   string    `json:"talker_alias,omitempty"`
	TalkerAliasAt time.Time `json:"talker_alias_at,omitempty"`
	CallCount     uint64    `json:"call_count,omitempty"`
	FirstSeen     time.Time `json:"first_seen,omitempty"`
	LastSeen      time.Time `json:"last_seen,omitempty"`
}

func ridToDTO(r *trunking.RID) *RIDDTO {
	if r == nil {
		return nil
	}
	return &RIDDTO{
		ID:          r.ID,
		Alias:       r.Alias,
		Description: r.Description,
		Tag:         r.Tag,
		Group:       r.Group,
		Owner:       r.Owner,
		Priority:    r.Priority,
		Lockout:     r.Lockout,
		Watch:       r.Watch,
		Icon:        r.Icon,
		Configured:  true,
	}
}

// mergeRIDLive applies the live UnitActivity fields to the DTO. If
// dto is nil a fresh, non-Configured DTO is returned for the live row.
func mergeRIDLive(dto *RIDDTO, u trunking.UnitActivity) *RIDDTO {
	if dto == nil {
		dto = &RIDDTO{ID: u.RadioID, Watch: true}
	}
	dto.System = u.System
	dto.Protocol = u.Protocol
	dto.LastTalkgroup = u.Talkgroup
	dto.TalkerAlias = u.TalkerAlias
	dto.TalkerAliasAt = u.TalkerAliasAt
	dto.CallCount = u.CallCount
	dto.FirstSeen = u.FirstSeen
	dto.LastSeen = u.LastSeen
	return dto
}

// GrantDTO mirrors trunking.Grant.
type GrantDTO struct {
	System        string `json:"system"`
	Protocol      string `json:"protocol"`
	GroupID       uint32 `json:"group_id"`
	SourceID      uint32 `json:"source_id"`
	FrequencyHz   uint32 `json:"frequency_hz"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	// Timeslot is the 1-based TDMA slot (0 = n/a, 1 = TS1, 2 = TS2).
	// Non-zero only for slotted protocols (DMR Tier III); identifies
	// which of a carrier's two calls this is.
	Timeslot  uint8 `json:"timeslot,omitempty"`
	Encrypted bool  `json:"encrypted,omitempty"`
	Emergency bool  `json:"emergency,omitempty"`
	DataCall  bool  `json:"data_call,omitempty"`
	// AlgorithmID / KeyID surface the P25 encryption parameters
	// recovered from the in-call signalling. Zero when Encrypted is
	// false; also zero on a Phase 1 grant until the LDU2 Encryption
	// Sync has been parsed and the engine has backfilled the active
	// call (see KindCallEncryption).
	AlgorithmID uint8  `json:"algorithm_id,omitempty"`
	KeyID       uint16 `json:"key_id,omitempty"`
}

func grantToDTO(g trunking.Grant) GrantDTO {
	return GrantDTO{
		System: g.System, Protocol: g.Protocol,
		GroupID: g.GroupID, SourceID: g.SourceID,
		FrequencyHz: g.FrequencyHz,
		ChannelID:   g.ChannelID, ChannelNumber: g.ChannelNum,
		Timeslot:  g.Timeslot,
		Encrypted: g.Encrypted, Emergency: g.Emergency,
		DataCall:    g.DataCall,
		AlgorithmID: g.AlgorithmID, KeyID: g.KeyID,
	}
}

// CallEncryptionDTO mirrors trunking.CallEncryption for SSE / REST
// consumers. Subscribers patch the matching active-call row with the
// new ALGID/KID so the UI flips from "enc" to "enc 0x84 (AES-256)"
// the moment the LDU2 lands.
type CallEncryptionDTO struct {
	DeviceSerial string    `json:"device_serial"`
	System       string    `json:"system,omitempty"`
	Protocol     string    `json:"protocol,omitempty"`
	GroupID      uint32    `json:"group_id,omitempty"`
	AlgorithmID  uint8     `json:"algorithm_id"`
	KeyID        uint16    `json:"key_id"`
	At           time.Time `json:"at"`
}

func callEncryptionToDTO(c trunking.CallEncryption) CallEncryptionDTO {
	return CallEncryptionDTO{
		DeviceSerial: c.DeviceSerial,
		System:       c.System,
		Protocol:     c.Protocol,
		GroupID:      c.GroupID,
		AlgorithmID:  c.AlgorithmID,
		KeyID:        c.KeyID,
		At:           c.At,
	}
}

// AffiliationDTO mirrors trunking.Affiliation.
type AffiliationDTO struct {
	System            string `json:"system"`
	Protocol          string `json:"protocol"`
	SourceID          uint32 `json:"source_id"`
	GroupID           uint32 `json:"group_id"`
	AnnouncementGroup uint32 `json:"announcement_group,omitempty"`
	Response          string `json:"response"`
}

func affiliationToDTO(a trunking.Affiliation) AffiliationDTO {
	return AffiliationDTO{
		System: a.System, Protocol: a.Protocol,
		SourceID:          a.SourceID,
		GroupID:           a.GroupID,
		AnnouncementGroup: a.AnnouncementGroup,
		Response:          a.Response.String(),
	}
}

// UnitRegistrationDTO mirrors trunking.UnitRegistration.
type UnitRegistrationDTO struct {
	System   string `json:"system"`
	Protocol string `json:"protocol"`
	SourceID uint32 `json:"source_id"`
	WACN     uint32 `json:"wacn"`
	SystemID uint16 `json:"system_id"`
	Response string `json:"response"`
}

func unitRegistrationToDTO(u trunking.UnitRegistration) UnitRegistrationDTO {
	return UnitRegistrationDTO{
		System: u.System, Protocol: u.Protocol,
		SourceID: u.SourceID,
		WACN:     u.WACN,
		SystemID: u.SystemID,
		Response: u.Response.String(),
	}
}

// PatchDTO mirrors trunking.Patch for SSE / REST consumers. Add=true is
// a patch becoming active; Add=false is a cancel.
type PatchDTO struct {
	System     string    `json:"system"`
	Protocol   string    `json:"protocol"`
	SuperGroup uint32    `json:"super_group"`
	Members    []uint32  `json:"members"`
	Vendor     string    `json:"vendor,omitempty"`
	Add        bool      `json:"add"`
	At         time.Time `json:"at"`
}

func patchToDTO(p trunking.Patch) PatchDTO {
	return PatchDTO{
		System:     p.System,
		Protocol:   p.Protocol,
		SuperGroup: p.SuperGroup,
		Members:    append([]uint32(nil), p.Members...),
		Vendor:     p.Vendor,
		Add:        p.Add,
		At:         p.At,
	}
}

// DMRGrantObservedDTO mirrors events.DMRGrantObserved — a DMR Tier III
// voice-grant CSBK seen by the control channel before the LCN is resolved
// to a frequency. Surfaced so operators can watch the autoconfig learner's
// raw input in the CC Activity log. Timeslot uses the raw CSBK encoding
// (0 = TS1, 1 = TS2). (#638)
type DMRGrantObservedDTO struct {
	System    string    `json:"system"`
	ColorCode uint8     `json:"color_code"`
	LCN       uint16    `json:"lcn"`
	Timeslot  uint8     `json:"timeslot"`
	GroupID   uint32    `json:"group_id"`
	SourceID  uint32    `json:"source_id"`
	CCFreqHz  uint32    `json:"cc_freq_hz"`
	At        time.Time `json:"at"`
}

func dmrGrantObservedToDTO(g events.DMRGrantObserved) DMRGrantObservedDTO {
	return DMRGrantObservedDTO{
		System:    g.System,
		ColorCode: g.ColorCode,
		LCN:       g.LCN,
		Timeslot:  g.Timeslot,
		GroupID:   g.GroupID,
		SourceID:  g.SourceID,
		CCFreqHz:  g.CCFreqHz,
		At:        g.At,
	}
}

// DMRBandPlanLearnedDTO mirrors events.DMRBandPlanLearned — the result of
// the DMR Tier III LCN autoconfig learner fitting a band plan from observed
// (LCN, frequency) pairs. A linear plan sets BaseHz/SpacingHz/Offset with an
// empty Table; an irregular plan populates Table. (#638)
type DMRBandPlanLearnedDTO struct {
	System     string              `json:"system"`
	BaseHz     uint32              `json:"base_hz,omitempty"`
	SpacingHz  uint32              `json:"spacing_hz,omitempty"`
	Offset     int8                `json:"offset,omitempty"`
	Table      []DMRBandPlanLCNDTO `json:"table,omitempty"`
	NumPairs   int                 `json:"num_pairs"`
	Confidence float64             `json:"confidence"`
	ResidualHz uint32              `json:"residual_hz,omitempty"`
}

func dmrBandPlanLearnedToDTO(p events.DMRBandPlanLearned) DMRBandPlanLearnedDTO {
	dto := DMRBandPlanLearnedDTO{
		System:     p.System,
		BaseHz:     p.BaseHz,
		SpacingHz:  p.SpacingHz,
		Offset:     p.Offset,
		NumPairs:   p.NumPairs,
		Confidence: p.Confidence,
		ResidualHz: p.ResidualHz,
	}
	for _, e := range p.Table {
		dto.Table = append(dto.Table, DMRBandPlanLCNDTO{LCN: e.LCN, FreqHz: e.FreqHz})
	}
	return dto
}

// ActiveCallDTO mirrors trunking.ActiveCall for JSON.
type ActiveCallDTO struct {
	Grant        GrantDTO      `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string        `json:"device_serial"`
	StartedAt    time.Time     `json:"started_at"`
	LastHeardAt  time.Time     `json:"last_heard_at"`
}

func activeCallToDTO(ac *trunking.ActiveCall) ActiveCallDTO {
	return ActiveCallDTO{
		Grant:        grantToDTO(ac.Grant),
		Talkgroup:    talkgroupToDTO(ac.Talkgroup),
		DeviceSerial: ac.Device.Serial,
		StartedAt:    ac.StartedAt,
		LastHeardAt:  ac.LastHeardAt,
	}
}

// CallStartDTO / CallEndDTO mirror the trunking event payloads.
type CallStartDTO struct {
	Grant        GrantDTO      `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string        `json:"device_serial"`
	StartedAt    time.Time     `json:"started_at"`
}

type CallEndDTO struct {
	Grant        GrantDTO      `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string        `json:"device_serial"`
	StartedAt    time.Time     `json:"started_at"`
	EndedAt      time.Time     `json:"ended_at"`
	Reason       string        `json:"reason"`
}

func callStartToDTO(cs trunking.CallStart) CallStartDTO {
	return CallStartDTO{
		Grant:        grantToDTO(cs.Grant),
		Talkgroup:    talkgroupToDTO(cs.Talkgroup),
		DeviceSerial: cs.DeviceSerial,
		StartedAt:    cs.StartedAt,
	}
}

func callEndToDTO(ce trunking.CallEnd) CallEndDTO {
	return CallEndDTO{
		Grant:        grantToDTO(ce.Grant),
		Talkgroup:    talkgroupToDTO(ce.Talkgroup),
		DeviceSerial: ce.DeviceSerial,
		StartedAt:    ce.StartedAt,
		EndedAt:      ce.EndedAt,
		Reason:       ce.Reason.String(),
	}
}
