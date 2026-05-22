package api

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

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

// GrantDTO mirrors trunking.Grant.
type GrantDTO struct {
	System        string `json:"system"`
	Protocol      string `json:"protocol"`
	GroupID       uint32 `json:"group_id"`
	SourceID      uint32 `json:"source_id"`
	FrequencyHz   uint32 `json:"frequency_hz"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	Encrypted     bool   `json:"encrypted,omitempty"`
	Emergency     bool   `json:"emergency,omitempty"`
	DataCall      bool   `json:"data_call,omitempty"`
}

func grantToDTO(g trunking.Grant) GrantDTO {
	return GrantDTO{
		System: g.System, Protocol: g.Protocol,
		GroupID: g.GroupID, SourceID: g.SourceID,
		FrequencyHz: g.FrequencyHz,
		ChannelID:   g.ChannelID, ChannelNumber: g.ChannelNum,
		Encrypted: g.Encrypted, Emergency: g.Emergency,
		DataCall: g.DataCall,
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
