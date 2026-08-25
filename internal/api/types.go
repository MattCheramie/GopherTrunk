package api

import (
	"sort"
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
	// GainRecommendations / GainNote carry the result of an -auto-gain run.
	GainRecommendations []hunt.GainRecommendation `json:"gain_recommendations,omitempty"`
	GainNote            string                    `json:"gain_note,omitempty"`
}

// HuntCaptureRequest records one signal from the survey inventory and routes it
// to deeper analysis — the daemon counterpart of the CLI's -survey-capture.
type HuntCaptureRequest struct {
	FreqHz  uint32  `json:"freq_hz"`           // signal centre frequency to record
	Seconds float64 `json:"seconds,omitempty"` // capture length (0 ⇒ a default)
	Target  string  `json:"target,omitempty"`  // "siglab" (default) | "cryptolab"
}

// HuntCaptureResult is what the daemon recorded and where it routed it.
type HuntCaptureResult struct {
	Path         string  `json:"path"`                    // the written .cfile
	MetadataPath string  `json:"metadata_path,omitempty"` // the sidecar
	SampleRateHz float64 `json:"sample_rate_hz"`
	CenterHz     uint32  `json:"center_hz"`
	Samples      int     `json:"samples"`
	// Identify is a short SigLab identify summary (target=siglab), e.g.
	// "p25 (0.82)" or "inconclusive".
	Identify string `json:"identify,omitempty"`
	// FramesPath is the cryptolab `ks` frames file (target=cryptolab), empty
	// otherwise.
	FramesPath string `json:"frames_path,omitempty"`
	// Note carries an advisory (e.g. a wideband signal recorded as a slice).
	Note string `json:"note,omitempty"`
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
	Serial           string    `json:"serial,omitempty"`
	Bands            []string  `json:"bands,omitempty"`      // "low:high" MHz
	Candidates       []float64 `json:"candidates,omitempty"` // MHz
	NoSweep          bool      `json:"no_sweep,omitempty"`
	ConfirmBorrow    bool      `json:"confirm_borrow,omitempty"`    // authorize borrowing an SDR with live wideband decode (suspends it for the run)
	Survey           bool      `json:"survey,omitempty"`            // classify+decode every carrier, not just trunking CCs
	ClassifyOnly     bool      `json:"classify_only,omitempty"`     // survey: classify, skip decoding
	DetectWideband   bool      `json:"detect_wideband,omitempty"`   // survey: detect+stitch signals wider than a channel (cellular/WiFi/OFDM)
	DetectEncryption bool      `json:"detect_encryption,omitempty"` // survey: entropy-triage unknown digital carriers for encryption
	PersistSurvey    bool      `json:"persist_survey,omitempty"`    // survey: stream carriers to NDJSON (crash-safe, resumable)
	Resume           bool      `json:"resume,omitempty"`            // survey: skip frequencies already in the NDJSON file
	AutoGain         bool      `json:"auto_gain,omitempty"`         // post-run: sweep gains on locked CCs, recommend the best
	AutoGainSet      string    `json:"auto_gain_set,omitempty"`     // optional comma-separated gain ladder in dB (e.g. "0,8.7,16.6")
	MaxDwellSeconds  float64   `json:"max_dwell_seconds,omitempty"`
	MonitorSeconds   float64   `json:"monitor_seconds,omitempty"` // >0: stream a locked CC in real time for up to this long (converge-and-stop), instead of buffering the dwell
	Protocol         string    `json:"protocol,omitempty"`
	DwellSeconds     float64   `json:"dwell_seconds,omitempty"`
	SweepDwellMs     int       `json:"sweep_dwell_ms,omitempty"`
	PeakThresholdDb  float64   `json:"peak_threshold_db,omitempty"`
	MinSpacingHz     uint32    `json:"min_spacing_hz,omitempty"`
	FFTSize          int       `json:"fft_size,omitempty"`
	MinConfidence    float64   `json:"min_confidence,omitempty"`
	Name             string    `json:"name,omitempty"`
	State            string    `json:"state,omitempty"`
	County           string    `json:"county,omitempty"`
	Location         string    `json:"location,omitempty"`
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
	// Hex renderings of the identity numbers (P25 field convention),
	// alongside the decimal fields above. Empty when the value is unknown.
	WACNHex     string `json:"wacn_hex,omitempty"`
	SystemIDHex string `json:"system_id_hex,omitempty"`
	RFSSHex     string `json:"rfss_hex,omitempty"`
	SiteHex     string `json:"site_hex,omitempty"`

	// TETRA network identity decoded live from the control channel
	// (MLE-SYSINFO / D-NWRK-BROADCAST), overlaid from the SiteTracker topology —
	// the TETRA analogue of WACN/SystemID/RFSS/Site. TETRA has no WACN/RFSS/Site
	// concept, so those P25 fields stay zero for a TETRA system; the UI shows
	// these instead. MCC+MNC form the Mobile Network Identity (MNI). Zero until
	// the CC locks and broadcasts identity; HasTETRAIdentity distinguishes a
	// decoded value from "not yet" (colour code / LA can legitimately be 0).
	TETRAMCC               uint16 `json:"tetra_mcc,omitempty"`
	TETRAMNC               uint16 `json:"tetra_mnc,omitempty"`
	TETRALocationArea      uint16 `json:"tetra_location_area,omitempty"`
	TETRADecodedColourCode uint8  `json:"tetra_decoded_colour_code,omitempty"`
	HasTETRAIdentity       bool   `json:"has_tetra_identity,omitempty"`

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
	P25Phase2SoftDecision  string  `json:"p25_phase2_soft_decision,omitempty"`
	P25Phase2Equalizer     string  `json:"p25_phase2_equalizer,omitempty"`
	P25Phase2DCBlock       string  `json:"p25_phase2_dc_block,omitempty"`
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

	// Neighbors are the adjacent (neighbour) sites this system's control
	// channel advertised over the air (P25 Adjacent Site Status Broadcast,
	// opcode 0x3C), decoded live and overlaid from the SiteTracker's topology
	// snapshot. Empty when none have been heard yet. Surfaced so the Systems
	// panels can show them the way SDRtrunk's "Neighbor Sites" view does,
	// instead of only the system drill-in report.
	Neighbors []NeighborDTO `json:"neighbors,omitempty"`

	// FrequencyBands is the decoded P25 IDEN_UP frequency-band table (channel
	// ID → base/spacing/offset), overlaid live from the same topology snapshot
	// as Neighbors. Surfaced so the Systems panel can show the band plan the way
	// SDRtrunk's "Details" tab does, and so a neighbour's channel_id/
	// channel_number can be mapped back to an absolute frequency. Empty until an
	// IDEN_UP has been heard. (#814)
	FrequencyBands []BandPlanSlotDTO `json:"frequency_bands,omitempty"`
}

// NeighborDTO is one adjacent site advertised by a system's control channel,
// with band-plan-resolved downlink/uplink frequencies (zero when the band plan
// was not yet known). The hex renderings follow the P25 field convention,
// alongside the decimal fields.
type NeighborDTO struct {
	RFSS          uint8  `json:"rfss"`
	Site          uint8  `json:"site"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	DownlinkHz    uint32 `json:"downlink_hz,omitempty"`
	UplinkHz      uint32 `json:"uplink_hz,omitempty"`
	RFSSHex       string `json:"rfss_hex,omitempty"`
	SiteHex       string `json:"site_hex,omitempty"`
}

// neighborsFromTopology converts a live topology snapshot's neighbours into
// DTOs, reusing trunking.ReportFromTopology so downlink/uplink resolution
// matches the network-configuration report and the decoded-message log exactly.
// Sorted by (RFSS, Site) for stable output. Returns nil when there are none.
func neighborsFromTopology(snap *trunking.TopologySnapshot) []NeighborDTO {
	if snap == nil {
		return nil
	}
	r := trunking.ReportFromTopology(snap)
	if len(r.Sites) == 0 || len(r.Sites[0].Neighbors) == 0 {
		return nil
	}
	out := make([]NeighborDTO, 0, len(r.Sites[0].Neighbors))
	for _, n := range r.Sites[0].Neighbors {
		out = append(out, NeighborDTO{
			RFSS:          n.RFSS,
			Site:          n.Site,
			ChannelID:     n.Channel.ChannelID,
			ChannelNumber: n.Channel.ChannelNumber,
			DownlinkHz:    n.Channel.DownlinkHz,
			UplinkHz:      n.Channel.UplinkHz,
			RFSSHex:       trunking.IDHex(uint64(n.RFSS)),
			SiteHex:       trunking.IDHex(uint64(n.Site)),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RFSS != out[j].RFSS {
			return out[i].RFSS < out[j].RFSS
		}
		return out[i].Site < out[j].Site
	})
	return out
}

// BandPlanSlotDTO is one P25 IDEN_UP frequency-band entry, mirroring
// trunking.TopoBandPlanSlot / trunking.ReportBand. It maps a channel ID to a
// downlink base frequency, channel spacing, and (signed) transmit offset — the
// table SDRtrunk surfaces in its "Details" tab and the data needed to map a
// neighbour's channel_id/channel_number back to an absolute frequency.
type BandPlanSlotDTO struct {
	ChannelID   uint8  `json:"channel_id"`
	BaseHz      uint64 `json:"base_hz"`
	SpacingHz   uint32 `json:"spacing_hz"`
	BandwidthHz uint32 `json:"bandwidth_hz,omitempty"`
	TxOffsetHz  int64  `json:"tx_offset_hz,omitempty"`
	AccessTDMA  bool   `json:"access_tdma,omitempty"`
}

// bandPlanFromTopology converts a live topology snapshot's band plan into DTOs,
// reusing trunking.ReportFromTopology so the values match the
// network-configuration report and the decoded-message log exactly. Sorted by
// channel ID for stable output. Returns nil when there are none.
func bandPlanFromTopology(snap *trunking.TopologySnapshot) []BandPlanSlotDTO {
	if snap == nil {
		return nil
	}
	r := trunking.ReportFromTopology(snap)
	if len(r.Bands) == 0 {
		return nil
	}
	out := make([]BandPlanSlotDTO, 0, len(r.Bands))
	for _, b := range r.Bands {
		out = append(out, BandPlanSlotDTO{
			ChannelID:   b.ChannelID,
			BaseHz:      b.BaseHz,
			SpacingHz:   b.SpacingHz,
			BandwidthHz: b.BandwidthHz,
			TxOffsetHz:  b.TxOffsetHz,
			AccessTDMA:  b.AccessTDMA,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChannelID < out[j].ChannelID })
	return out
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
		P25Phase2SoftDecision:  s.P25Phase2SoftDecision,
		P25Phase2Equalizer:     s.P25Phase2Equalizer,
		P25Phase2DCBlock:       s.P25Phase2DCBlock,
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
	// Discovered marks an auto-learned talkgroup (Tag == "Discovered") so the
	// UI can badge it and offer a "hide auto-discovered" filter — the phantom
	// radio-ID entries the operator wants collapsed.
	Discovered bool `json:"discovered"`
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
		Discovered:  tg.Tag == trunking.DiscoveredTag,
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
	// TalkerAliasUnreliable flags an over-the-air alias whose decode
	// passed CRC but held non-ASCII-printable characters (#711); a
	// client should render it as suspect.
	TalkerAliasUnreliable bool      `json:"talker_alias_unreliable,omitempty"`
	CallCount             uint64    `json:"call_count,omitempty"`
	FirstSeen             time.Time `json:"first_seen,omitempty"`
	LastSeen              time.Time `json:"last_seen,omitempty"`
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

// mergeRIDSummary applies the durable, call-log-derived aggregate fields to the
// DTO. If dto is nil a fresh, non-Configured DTO is returned for the persisted
// row (a radio seen over the air but not in the static catalogue). Applied
// BENEATH mergeRIDLive: the live tracker, when it still holds the radio, wins for
// the most-recent columns; this fills them in for radios the tracker has swept.
func mergeRIDSummary(dto *RIDDTO, s RIDSummary) *RIDDTO {
	if dto == nil {
		dto = &RIDDTO{ID: s.SourceID, Watch: true}
	}
	dto.System = s.System
	dto.Protocol = s.Protocol
	if s.LastTalkgroup != 0 {
		dto.LastTalkgroup = s.LastTalkgroup
	}
	if s.SourceAlpha != "" && dto.TalkerAlias == "" {
		dto.TalkerAlias = s.SourceAlpha
	}
	dto.CallCount = s.CallCount
	dto.FirstSeen = s.FirstSeen
	dto.LastSeen = s.LastSeen
	return dto
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
	dto.TalkerAliasUnreliable = u.TalkerAliasUnreliable
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
	// RFSSID / SiteID name the P25 site whose control channel granted
	// the call, so consumers can label metrics by site instead of the
	// opaque channel_id (issue #698). Zero (and omitted) on non-P25
	// grants and until the site's RFSS Status TSBK has been decoded.
	RFSSID uint8 `json:"rfss_id,omitempty"`
	SiteID uint8 `json:"site_id,omitempty"`
	// NAC is the control channel's Network Access Code (Phase 2: the
	// NSB Color Code). Coarse — not unique per site — so consumers
	// label by (rfss_id, site_id). Zero/omitted on non-P25 grants.
	NAC uint16 `json:"nac,omitempty"`
	// Timeslot is the 1-based TDMA slot (0 = n/a, 1 = TS1, 2 = TS2).
	// Non-zero only for slotted protocols (DMR Tier III); identifies
	// which of a carrier's two calls this is.
	Timeslot  uint8 `json:"timeslot,omitempty"`
	Encrypted bool  `json:"encrypted,omitempty"`
	Emergency bool  `json:"emergency,omitempty"`
	DataCall  bool  `json:"data_call,omitempty"`
	// Individual marks a grant whose group_id is NOT a talkgroup but an
	// individual destination — a unit-to-unit target radio, an interconnect
	// target, or a TETRA subscriber ISSI — so a consumer can render the radio ID
	// as a unit rather than a phantom talkgroup. Omitted (false) for group calls.
	Individual bool `json:"individual,omitempty"`
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
		RFSSID: g.RFSSID, SiteID: g.SiteID,
		NAC:       g.NAC,
		Timeslot:  g.Timeslot,
		Encrypted: g.Encrypted, Emergency: g.Emergency,
		DataCall:    g.DataCall,
		Individual:  g.Individual,
		AlgorithmID: g.AlgorithmID, KeyID: g.KeyID,
	}
}

// SiteDTO is one row of GET /api/v1/sites: a P25 site GT has either
// discovered from the control channel, named in the system config, or
// both (issue #698). Name is empty when the operator has not labelled
// the site; ControlChannelHz is zero for a configured site GT has not
// yet heard over the air.
type SiteDTO struct {
	System           string `json:"system"`
	RFSSID           uint8  `json:"rfss_id"`
	SiteID           uint8  `json:"site_id"`
	ControlChannelHz uint32 `json:"control_channel_hz,omitempty"`
	// ControlChannelCarrierOffsetHz is the demod's measured carrier offset (Hz)
	// of the locked control carrier relative to the tuned centre. A large value
	// flags a control lock sitting well off the configured frequency — e.g. an
	// adjacent site bleeding through at 12.5 kHz spacing (issue #815).
	ControlChannelCarrierOffsetHz int32 `json:"control_channel_carrier_offset_hz,omitempty"`
	// ControlChannelTSBKErrorRate is the cumulative percentage of TSBK blocks on
	// this site's control channel that failed Viterbi/CRC — a decode-quality
	// signal independent of carrier lock, since a well-locked carrier at range
	// can still show a high rate (issue #858). It is a frame-error rate, not a
	// pre-FEC bit-error rate. A pointer so a genuine 0.0 (perfect decode) is
	// distinguishable from "no data" (nil): nil until a TSBK has been attempted,
	// and on non-TSBK Phase 2 (TDMA) paths. ControlChannelTSBKCount is the total
	// attempts the rate was measured over (a confidence weight).
	ControlChannelTSBKErrorRate *float64 `json:"control_channel_tsbk_error_rate,omitempty"`
	ControlChannelTSBKCount     int64    `json:"control_channel_tsbk_count,omitempty"`
	// ControlChannelDecodeQuality buckets the error rate into clean / marginal /
	// poor (SDRTrunk-style), for consumers that want a category rather than a
	// number. Empty when there is no TSBK data yet. See decodeQuality.
	ControlChannelDecodeQuality string `json:"control_channel_decode_quality,omitempty"`
	Name                        string `json:"name"`
	// Latitude / Longitude are the operator-configured (or RadioReference-
	// imported) site position in decimal degrees, merged from the matching
	// ConfiguredSite. Omitted when unset, so the web console plots only the
	// sites that have a known location.
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	WACN      uint32  `json:"wacn,omitempty"`
	SystemID  uint16  `json:"system_id,omitempty"`
	// Hex renderings of the identity numbers, alongside the decimal fields.
	WACNHex     string `json:"wacn_hex,omitempty"`
	SystemIDHex string `json:"system_id_hex,omitempty"`
	RFSSIDHex   string `json:"rfss_id_hex,omitempty"`
	SiteIDHex   string `json:"site_id_hex,omitempty"`
	// Neighbors are the adjacent sites this site advertised over the air (P25
	// Adjacent Site Status Broadcast, opcode 0x3C), overlaid from the live
	// topology snapshot. Each carries the neighbour's RFSS/Site and its
	// band-plan-resolved control-channel downlink (and uplink) frequency, so a
	// consumer can backfill the wider network's control-channel map from a
	// single camped site without decoding every site directly. Only the camped
	// site of a system carries them (that is the site that broadcast the list);
	// other rows leave it empty (issue #864).
	Neighbors []NeighborDTO `json:"neighbors,omitempty"`
}

// TSBK frame-error-rate thresholds (percent) that bucket a control channel's
// decode quality for ControlChannelDecodeQuality. Starting values chosen to be
// legible as percentages; tune as field data accumulates (issue #858).
const (
	tsbkCleanMaxPct    = 1.0 // ≤ this → "clean"
	tsbkMarginalMaxPct = 5.0 // ≤ this → "marginal"; above → "poor"
)

// decodeQuality buckets a TSBK frame-error rate (percent) into an SDRTrunk-style
// quality category. Only meaningful when at least one TSBK has been attempted;
// callers gate on the attempt count before calling.
func decodeQuality(ratePct float64) string {
	switch {
	case ratePct <= tsbkCleanMaxPct:
		return "clean"
	case ratePct <= tsbkMarginalMaxPct:
		return "marginal"
	default:
		return "poor"
	}
}

// fillIdentityHex derives the *_hex fields from the decimal identity fields.
// Call after any overlay that mutates the numeric fields so the two agree.
func (d *SystemDTO) fillIdentityHex() {
	d.WACNHex = trunking.IDHex(uint64(d.WACN))
	d.SystemIDHex = trunking.IDHex(uint64(d.SystemID))
	d.RFSSHex = trunking.IDHex(uint64(d.RFSS))
	d.SiteHex = trunking.IDHex(uint64(d.Site))
}

// fillIdentityHex derives the *_hex fields from the decimal identity fields.
func (d *SiteDTO) fillIdentityHex() {
	d.WACNHex = trunking.IDHex(uint64(d.WACN))
	d.SystemIDHex = trunking.IDHex(uint64(d.SystemID))
	d.RFSSIDHex = trunking.IDHex(uint64(d.RFSSID))
	d.SiteIDHex = trunking.IDHex(uint64(d.SiteID))
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
	// RFSSID / SiteID name the radio's serving site (a genuine
	// RID→site fix, unlike grant-site) and NAC is the coarse control
	// channel access code (issue #698). Zero/omitted on non-P25
	// affiliations and until the site's status broadcast has decoded.
	RFSSID uint8  `json:"rfss_id,omitempty"`
	SiteID uint8  `json:"site_id,omitempty"`
	NAC    uint16 `json:"nac,omitempty"`
}

func affiliationToDTO(a trunking.Affiliation) AffiliationDTO {
	return AffiliationDTO{
		System: a.System, Protocol: a.Protocol,
		SourceID:          a.SourceID,
		GroupID:           a.GroupID,
		AnnouncementGroup: a.AnnouncementGroup,
		Response:          a.Response.String(),
		RFSSID:            a.RFSSID,
		SiteID:            a.SiteID,
		NAC:               a.NAC,
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
	// RFSSID / SiteID name the radio's serving site (a genuine
	// RID→site fix) and NAC is the coarse control channel access code
	// (issue #698). Zero/omitted on non-P25 registrations and until the
	// site's status broadcast has decoded.
	RFSSID uint8  `json:"rfss_id,omitempty"`
	SiteID uint8  `json:"site_id,omitempty"`
	NAC    uint16 `json:"nac,omitempty"`
}

func unitRegistrationToDTO(u trunking.UnitRegistration) UnitRegistrationDTO {
	return UnitRegistrationDTO{
		System: u.System, Protocol: u.Protocol,
		SourceID: u.SourceID,
		WACN:     u.WACN,
		SystemID: u.SystemID,
		Response: u.Response.String(),
		RFSSID:   u.RFSSID,
		SiteID:   u.SiteID,
		NAC:      u.NAC,
	}
}

// UnitToUnitRequestDTO mirrors trunking.UnitToUnitRequest.
type UnitToUnitRequestDTO struct {
	System         string `json:"system"`
	Protocol       string `json:"protocol"`
	SourceID       uint32 `json:"source_id"`
	TargetID       uint32 `json:"target_id"`
	ServiceOptions uint8  `json:"service_options,omitempty"`
}

func unitToUnitRequestToDTO(u trunking.UnitToUnitRequest) UnitToUnitRequestDTO {
	return UnitToUnitRequestDTO{
		System: u.System, Protocol: u.Protocol,
		SourceID:       u.SourceID,
		TargetID:       u.TargetID,
		ServiceOptions: u.ServiceOptions,
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
	// Following is true when a voice tuner is actively decoding this call.
	// False for calls the control channel has announced but no tuner is
	// following (every voice device busy) — surfaced so the operator sees all
	// talkgroups up on the system, not only the ones being decoded. An
	// unfollowed call has an empty DeviceSerial.
	Following bool `json:"following"`
	// UnfollowedReason says why no tuner is following this call (issue #356):
	// "no voice SDR configured", "voice frequency outside every voice
	// device's tuning window", or "all voice tuners busy with higher-priority
	// calls". Omitted for followed calls, and empty while a grant is still
	// being allocated.
	UnfollowedReason string `json:"unfollowed_reason,omitempty"`
}

func activeCallToDTO(ac *trunking.ActiveCall, following bool) ActiveCallDTO {
	serial := ""
	if ac.Device != nil {
		serial = ac.Device.Serial
	}
	dto := ActiveCallDTO{
		Grant:        grantToDTO(ac.Grant),
		Talkgroup:    talkgroupToDTO(ac.Talkgroup),
		DeviceSerial: serial,
		StartedAt:    ac.StartedAt,
		LastHeardAt:  ac.LastHeardAt,
		Following:    following,
	}
	if !following {
		dto.UnfollowedReason = ac.UnfollowedReason
	}
	return dto
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
	// DurationMs is the call's length in milliseconds (ended − started). It
	// mirrors the completed-call webhook's duration_ms so an SSE/WS-only
	// consumer (a Prometheus exporter, a Grafana feed) can read a call's
	// duration off the completion event without pairing it back to the
	// call.start and subtracting timestamps itself (issue #268).
	DurationMs int64  `json:"duration_ms"`
	Reason     string `json:"reason"`
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
	// Duration only when both timestamps are sane; a watchdog/shutdown teardown
	// can leave StartedAt zero, and clock skew must not surface a negative.
	var durMs int64
	if !ce.StartedAt.IsZero() && ce.EndedAt.After(ce.StartedAt) {
		durMs = ce.EndedAt.Sub(ce.StartedAt).Milliseconds()
	}
	return CallEndDTO{
		Grant:        grantToDTO(ce.Grant),
		Talkgroup:    talkgroupToDTO(ce.Talkgroup),
		DeviceSerial: ce.DeviceSerial,
		StartedAt:    ce.StartedAt,
		EndedAt:      ce.EndedAt,
		DurationMs:   durMs,
		Reason:       ce.Reason.String(),
	}
}
