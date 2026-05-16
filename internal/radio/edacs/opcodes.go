package edacs

import "fmt"

// Command is the 4-bit operation field carried in CCW.Command. The
// EDACS standard reserves the full 4-bit space; this enum covers the
// subset most useful for trunking follow-along.
type Command uint8

const (
	CmdIdle            Command = 0x0
	CmdGroupVoiceGrant Command = 0x1
	CmdProVoiceGrant   Command = 0x2 // EDACS ProVoice (digital)
	CmdIndividualCall  Command = 0x3
	CmdDataGrant       Command = 0x4
	CmdSystemID        Command = 0x5
	CmdAdjacentSite    Command = 0x6
	CmdEmergency       Command = 0x7
	CmdAffiliation     Command = 0x8
	CmdEncryption      Command = 0x9
	CmdReserved        Command = 0xF
)

// IsKnown reports whether the Command value is one of the
// documented opcodes the state machine knows how to handle.
// Strict-mode operators use this to reject CCWs whose Command
// field falls outside the recognised set (a strong signal of
// bit errors or a misaligned codeword).
func (c Command) IsKnown() bool {
	switch c {
	case CmdIdle, CmdGroupVoiceGrant, CmdProVoiceGrant,
		CmdIndividualCall, CmdDataGrant, CmdSystemID,
		CmdAdjacentSite, CmdEmergency, CmdAffiliation,
		CmdEncryption, CmdReserved:
		return true
	}
	return false
}

func (c Command) String() string {
	switch c {
	case CmdIdle:
		return "Idle"
	case CmdGroupVoiceGrant:
		return "GroupVoiceGrant"
	case CmdProVoiceGrant:
		return "ProVoiceGrant"
	case CmdIndividualCall:
		return "IndividualCall"
	case CmdDataGrant:
		return "DataGrant"
	case CmdSystemID:
		return "SystemID"
	case CmdAdjacentSite:
		return "AdjacentSite"
	case CmdEmergency:
		return "Emergency"
	case CmdAffiliation:
		return "Affiliation"
	case CmdEncryption:
		return "Encryption"
	default:
		return fmt.Sprintf("Command(%X)", uint8(c))
	}
}

// IsIdle reports whether the CCW is the channel-idle / heartbeat
// command.
func (c CCW) IsIdle() bool { return c.Command == CmdIdle }

// IsEncrypted reads bit 0 of the Status nibble — EDACS reserves the
// low Status bit as the encryption flag for voice grants.
func (c CCW) IsEncrypted() bool { return c.Status&0x1 != 0 }

// IsEmergency reads bit 1 of the Status nibble — EDACS sets it for
// emergency calls layered onto a regular voice grant.
func (c CCW) IsEmergency() bool { return c.Status&0x2 != 0 }

// GroupVoiceGrant is the high-level shape of an EDACS voice grant.
// Address holds the talkgroup; LCN references an entry in the
// operator-supplied band plan.
type GroupVoiceGrant struct {
	GroupAddress uint16
	LCN          uint8
	Encrypted    bool
	Emergency    bool
	ProVoice     bool // true when the grant came from a CmdProVoiceGrant
}

// AsGroupVoiceGrant returns the structured grant if the CCW's command
// is one of the voice-grant variants, otherwise (zero, false).
func (c CCW) AsGroupVoiceGrant() (GroupVoiceGrant, bool) {
	switch c.Command {
	case CmdGroupVoiceGrant, CmdProVoiceGrant:
	default:
		return GroupVoiceGrant{}, false
	}
	return GroupVoiceGrant{
		GroupAddress: c.Address,
		LCN:          c.LCN,
		Encrypted:    c.IsEncrypted(),
		Emergency:    c.IsEmergency(),
		ProVoice:     c.Command == CmdProVoiceGrant,
	}, true
}

// SystemID is the opaque identifier carried by CmdSystemID
// announcements. Address holds the system identifier; the Aux field
// carries an EDACS site / network identifier subfield.
type SystemID struct {
	ID  uint16
	Aux uint16
}

// AsSystemID returns the structured system identifier if this is a
// CmdSystemID CCW, otherwise (zero, false).
func (c CCW) AsSystemID() (SystemID, bool) {
	if c.Command != CmdSystemID {
		return SystemID{}, false
	}
	return SystemID{ID: c.Address, Aux: c.Aux}, true
}

// AdjacentSite is a neighbour-site announcement. Address holds the
// adjacent site ID; LCN is that site's control-channel index.
type AdjacentSite struct {
	SiteID uint16
	LCN    uint8
}

// AsAdjacentSite returns the structured adjacent-site descriptor if
// this is a CmdAdjacentSite CCW.
func (c CCW) AsAdjacentSite() (AdjacentSite, bool) {
	if c.Command != CmdAdjacentSite {
		return AdjacentSite{}, false
	}
	return AdjacentSite{SiteID: c.Address, LCN: c.LCN}, true
}
