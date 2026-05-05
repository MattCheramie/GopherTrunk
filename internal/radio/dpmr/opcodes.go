package dpmr

import "fmt"

// MessageType is the 5-bit Message Type field that opens every CSBK.
// Values follow ETSI TS 102 658 §6.5.2; only the subset relevant to
// trunked-grant follow-along is enumerated below.
type MessageType uint8

const (
	MsgUnknown                    MessageType = 0x00
	MsgRegistrationRequest        MessageType = 0x01
	MsgRegistrationResponse       MessageType = 0x02
	MsgVoiceServiceAllocation     MessageType = 0x03 // group voice channel grant
	MsgIndividualVoiceAllocation  MessageType = 0x04 // unit-to-unit voice grant
	MsgDataServiceAllocation      MessageType = 0x05
	MsgServiceRequest             MessageType = 0x06
	MsgStandingServiceStatus      MessageType = 0x07 // periodic site broadcast
	MsgRelease                    MessageType = 0x0F
	MsgIdle                       MessageType = 0x1F
)

// String returns a stable human-readable label for log output.
func (m MessageType) String() string {
	switch m {
	case MsgRegistrationRequest:
		return "RegistrationRequest"
	case MsgRegistrationResponse:
		return "RegistrationResponse"
	case MsgVoiceServiceAllocation:
		return "VoiceServiceAllocation"
	case MsgIndividualVoiceAllocation:
		return "IndividualVoiceAllocation"
	case MsgDataServiceAllocation:
		return "DataServiceAllocation"
	case MsgServiceRequest:
		return "ServiceRequest"
	case MsgStandingServiceStatus:
		return "StandingServiceStatus"
	case MsgRelease:
		return "Release"
	case MsgIdle:
		return "Idle"
	default:
		return fmt.Sprintf("MessageType(%02X)", uint8(m))
	}
}

// VoiceGrant is the structured shape of a voice-allocation CSBK
// (group or individual). The Extra field carries the channel number
// the calling/called subscriber should retune to.
type VoiceGrant struct {
	SourceID  uint32 // 24-bit calling subscriber
	DestID    uint32 // 24-bit destination (group or unit)
	Channel   uint16 // physical channel number in the band-plan
	Group     bool   // true when the call is to a group
	Emergency bool
	Encrypted bool
}

// AsVoiceGrant returns the structured grant if the CSBK message type
// is a voice-allocation variant, otherwise (zero, false).
func (c CSBK) AsVoiceGrant() (VoiceGrant, bool) {
	switch c.Type {
	case MsgVoiceServiceAllocation, MsgIndividualVoiceAllocation:
	default:
		return VoiceGrant{}, false
	}
	return VoiceGrant{
		SourceID:  c.SourceID,
		DestID:    c.DestID,
		Channel:   c.Extra,
		Group:     c.IsGroup() || c.Type == MsgVoiceServiceAllocation,
		Emergency: c.IsEmergency(),
		Encrypted: c.IsEncrypted(),
	}, true
}

// SiteBroadcast is the structured shape of a StandingServiceStatus
// CSBK — used by the state machine to declare the control channel
// "locked" and learn the system's identifier.
type SiteBroadcast struct {
	SystemID uint32 // packed (DestID) — system / fleet identifier
	Status   uint16 // Extra — opaque site status word
}

// AsSiteBroadcast returns the structured broadcast if the CSBK type
// is StandingServiceStatus, otherwise (zero, false).
func (c CSBK) AsSiteBroadcast() (SiteBroadcast, bool) {
	if c.Type != MsgStandingServiceStatus {
		return SiteBroadcast{}, false
	}
	return SiteBroadcast{
		SystemID: c.DestID,
		Status:   c.Extra,
	}, true
}

// IsIdle reports whether the CSBK is an idle / release filler the
// state machine should silently absorb.
func (c CSBK) IsIdle() bool {
	switch c.Type {
	case MsgIdle, MsgRelease:
		return true
	}
	return false
}
