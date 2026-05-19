package trunking

import "time"

// AffiliationResponse encodes the P25 Group Affiliation Response value
// (TIA-102.AABF Table 7-37). The integer values are wire constants — do
// not renumber.
type AffiliationResponse uint8

const (
	AffiliationAccepted AffiliationResponse = 0
	AffiliationFailed   AffiliationResponse = 1
	AffiliationDenied   AffiliationResponse = 2
	AffiliationRefused  AffiliationResponse = 3
)

func (r AffiliationResponse) String() string {
	switch r {
	case AffiliationAccepted:
		return "accepted"
	case AffiliationFailed:
		return "failed"
	case AffiliationDenied:
		return "denied"
	case AffiliationRefused:
		return "refused"
	default:
		return "unknown"
	}
}

// Affiliation is published on the events bus when a radio unit
// affiliates with a talkgroup. Emitted by P25 control-channel decoders
// on opcode 0x28 (Group Affiliation Response).
type Affiliation struct {
	System            string              // System name, matches trunking.System.Name
	Protocol          string              // "p25" / "dmr" / "nxdn"
	SourceID          uint32              // radio unit being affiliated
	GroupID           uint32              // talkgroup the unit is joining
	AnnouncementGroup uint32              // optional announcement-group association (0 if unused)
	Response          AffiliationResponse // accepted / failed / denied / refused
	At                time.Time
}

// RegistrationResponse encodes the P25 Unit Registration Response value
// (TIA-102.AABF Table 7-43). The integer values are wire constants — do
// not renumber.
type RegistrationResponse uint8

const (
	RegistrationAccepted RegistrationResponse = 0
	RegistrationFailed   RegistrationResponse = 1
	RegistrationDenied   RegistrationResponse = 2
	RegistrationRefused  RegistrationResponse = 3
)

func (r RegistrationResponse) String() string {
	switch r {
	case RegistrationAccepted:
		return "accepted"
	case RegistrationFailed:
		return "failed"
	case RegistrationDenied:
		return "denied"
	case RegistrationRefused:
		return "refused"
	default:
		return "unknown"
	}
}

// UnitRegistration is published on the events bus when a radio unit
// registers (or attempts to register) on a site. Emitted by P25
// control-channel decoders on opcode 0x2C (Unit Registration Response).
type UnitRegistration struct {
	System   string               // System name, matches trunking.System.Name
	Protocol string               // "p25" / "dmr" / "nxdn"
	SourceID uint32               // radio unit's WUID
	WACN     uint32               // 20-bit Wide Area Communications Network ID
	SystemID uint16               // 12-bit system identifier
	Response RegistrationResponse // accepted / failed / denied / refused
	At       time.Time
}
