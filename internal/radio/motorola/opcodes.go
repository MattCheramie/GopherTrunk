package motorola

import "fmt"

// Opcode is the 12-bit operation field carried in the upper 12 bits
// of the OSW.Command word (the lower 4 bits are an opcode-specific
// parameter — typically the LCN / channel index for voice grants).
//
// Constants reflect the SmartZone subset most useful for trunking
// follow-along; the full Motorola opcode space is much larger and
// vendor-extended. Add to the list as needed.
type Opcode uint16

const (
	OpUnknown                      Opcode = 0x000
	OpGroupVoiceChannelGrant       Opcode = 0x308
	OpGroupVoiceChannelGrantUpdate Opcode = 0x309
	OpPrivateCallGrant             Opcode = 0x30B
	OpAdjacentSiteStatus           Opcode = 0x31B
	OpSystemIDExtended             Opcode = 0x080
	OpDataChannelGrant             Opcode = 0x310
	OpAffiliationResponse          Opcode = 0x320
	OpIdle1                        Opcode = 0x28D
	OpIdle2                        Opcode = 0x290
	OpEncryption                   Opcode = 0x140
	OpEmergency                    Opcode = 0x300
)

func (o Opcode) String() string {
	switch o {
	case OpGroupVoiceChannelGrant:
		return "GroupVoiceChannelGrant"
	case OpGroupVoiceChannelGrantUpdate:
		return "GroupVoiceChannelGrantUpdate"
	case OpPrivateCallGrant:
		return "PrivateCallGrant"
	case OpAdjacentSiteStatus:
		return "AdjacentSiteStatus"
	case OpSystemIDExtended:
		return "SystemIDExtended"
	case OpDataChannelGrant:
		return "DataChannelGrant"
	case OpAffiliationResponse:
		return "AffiliationResponse"
	case OpIdle1, OpIdle2:
		return "Idle"
	case OpEncryption:
		return "Encryption"
	case OpEmergency:
		return "Emergency"
	default:
		return fmt.Sprintf("Opcode(%03X)", uint16(o))
	}
}

// Opcode returns the 12-bit operation field encoded in OSW.Command.
func (o OSW) Opcode() Opcode { return Opcode(o.Command >> 4) }

// IsKnown reports whether the Opcode value is one of the
// documented constants the state machine recognises. Strict-mode
// operators use this to reject OSWs whose Opcode field falls
// outside the recognised set (a strong signal of bit errors or a
// misaligned codeword pair).
func (o Opcode) IsKnown() bool {
	switch o {
	case OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate,
		OpPrivateCallGrant, OpAdjacentSiteStatus, OpSystemIDExtended,
		OpDataChannelGrant, OpAffiliationResponse,
		OpIdle1, OpIdle2, OpEncryption, OpEmergency:
		return true
	}
	return false
}

// LCN returns the 4-bit logical-channel-number nibble that voice and
// data grants pack into the bottom of OSW.Command.
func (o OSW) LCN() uint16 { return o.Command & 0x000F }

// IsIdle reports whether this OSW is one of the recognised
// channel-idle / heartbeat opcodes.
func (o OSW) IsIdle() bool {
	op := o.Opcode()
	return op == OpIdle1 || op == OpIdle2
}

// GroupVoiceChannelGrant is the high-level shape of a voice grant. The
// Address field of the OSW carries the talkgroup; the LCN nibble
// references an entry in the operator-supplied band plan.
type GroupVoiceChannelGrant struct {
	GroupAddress uint16
	LCN          uint16
}

// AsGroupVoiceChannelGrant returns the structured grant if the OSW's
// opcode is one of the voice-grant variants, otherwise (zero, false).
func (o OSW) AsGroupVoiceChannelGrant() (GroupVoiceChannelGrant, bool) {
	switch o.Opcode() {
	case OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate:
	default:
		return GroupVoiceChannelGrant{}, false
	}
	return GroupVoiceChannelGrant{
		GroupAddress: o.Address,
		LCN:          o.LCN(),
	}, true
}

// SystemID is the opaque identifier carried by OpSystemIDExtended
// announcements. Address fields hold the system identifier; the
// command field's high 12 bits are the opcode and low 4 are a
// per-system "system-ID type" nibble.
type SystemID struct {
	ID    uint16
	Class uint8 // low nibble of OSW.Command (system-ID variant)
}

// AsSystemID returns the structured system identifier if this is a
// SystemIDExtended OSW, otherwise (zero, false).
func (o OSW) AsSystemID() (SystemID, bool) {
	if o.Opcode() != OpSystemIDExtended {
		return SystemID{}, false
	}
	return SystemID{
		ID:    o.Address,
		Class: uint8(o.Command & 0x0F),
	}, true
}

// AdjacentSite is a neighbour-site announcement. Site ID is in the
// Address field; the LCN nibble references this site's control
// channel within the operator's band plan.
type AdjacentSite struct {
	SiteID uint16
	LCN    uint16
}

// AsAdjacentSite returns the structured adjacent-site descriptor if
// this is an OpAdjacentSiteStatus OSW.
func (o OSW) AsAdjacentSite() (AdjacentSite, bool) {
	if o.Opcode() != OpAdjacentSiteStatus {
		return AdjacentSite{}, false
	}
	return AdjacentSite{
		SiteID: o.Address,
		LCN:    o.LCN(),
	}, true
}
