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
	TETRAColourCode      uint32 `json:"tetra_colour_code,omitempty"`
	TETRAChannel         string `json:"tetra_channel,omitempty"`
	TETRAChannelCoding   string `json:"tetra_channel_coding,omitempty"`
	LTRFCSMode           string `json:"ltr_fcs_mode,omitempty"`
	LTRManchesterMode    string `json:"ltr_manchester_mode,omitempty"`
	P25Phase2TrellisMode string `json:"p25_phase2_trellis_mode,omitempty"`
	P25Phase2RSMode      string `json:"p25_phase2_rs_mode,omitempty"`
	NXDNViterbiMode      string `json:"nxdn_viterbi_mode,omitempty"`
	EDACSBCHMode         string `json:"edacs_bch_mode,omitempty"`
	MPT1327BCHMode       string `json:"mpt1327_bch_mode,omitempty"`
	MotorolaBCHMode      string `json:"motorola_bch_mode,omitempty"`
}

func systemToDTO(s trunking.System) SystemDTO {
	return SystemDTO{
		Name:                 s.Name,
		Protocol:             s.Protocol.String(),
		ControlChannels:      append([]uint32(nil), s.ControlChannels...),
		WACN:                 s.WACN,
		SystemID:             s.SystemID,
		RFSS:                 s.RFSS,
		Site:                 s.Site,
		TETRAColourCode:      s.TETRAColourCode,
		TETRAChannel:         s.TETRAChannel,
		TETRAChannelCoding:   s.TETRAChannelCoding,
		LTRFCSMode:           s.LTRFCSMode,
		LTRManchesterMode:    s.LTRManchesterMode,
		P25Phase2TrellisMode: s.P25Phase2TrellisMode,
		P25Phase2RSMode:      s.P25Phase2RSMode,
		NXDNViterbiMode:      s.NXDNViterbiMode,
		EDACSBCHMode:         s.EDACSBCHMode,
		MPT1327BCHMode:       s.MPT1327BCHMode,
		MotorolaBCHMode:      s.MotorolaBCHMode,
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

// ActiveCallDTO mirrors trunking.ActiveCall for JSON.
type ActiveCallDTO struct {
	Grant        GrantDTO       `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string         `json:"device_serial"`
	StartedAt    time.Time      `json:"started_at"`
	LastHeardAt  time.Time      `json:"last_heard_at"`
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
	Grant        GrantDTO       `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string         `json:"device_serial"`
	StartedAt    time.Time      `json:"started_at"`
}

type CallEndDTO struct {
	Grant        GrantDTO       `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string         `json:"device_serial"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      time.Time      `json:"ended_at"`
	Reason       string         `json:"reason"`
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
