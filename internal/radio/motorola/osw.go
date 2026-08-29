package motorola

// OSW is one Outbound Status Word — the 27-bit information block
// carried by every Motorola Type II / SmartZone control-channel
// frame after sync, deinterleave, convolutional-parity ECC and the
// CRC-10 (see frame.go):
//
//	Address  16 bits — talkgroup (with low-nibble status flags) or
//	                   radio ID, depending on the surrounding
//	                   OSW sequence
//	Group     1 bit  — group (true) vs individual (false) address
//	Command  10 bits — a voice-channel number when it falls inside
//	                   the system's band plan, otherwise a control
//	                   command (idle, system ID, extended function…)
//
// Field semantics follow OP25's rx_smartnet / trunk-recorder's
// SmartnetParser: a single OSW is not self-describing — grants,
// system-ID broadcasts and extended functions span one to three
// consecutive OSWs, sequenced by the ControlChannel state machine
// (control.go).
type OSW struct {
	Address uint16
	Group   bool
	Command uint16
}

// Talkgroup returns the talkgroup ID with the low-nibble status
// flags stripped, the form scanners and RadioReference list. Only
// meaningful on group-addressed OSWs.
func (o OSW) Talkgroup() uint16 { return o.Address & 0xFFF0 }

// Encrypted reports the encrypted-call status flag carried in the
// low nibble of a group address.
func (o OSW) Encrypted() bool { return o.Address&0x8 != 0 }

// Emergency reports whether the group address's low-nibble option
// field flags an emergency call (options 2, 4 and 5).
func (o OSW) Emergency() bool {
	opt := o.Address & 0x7
	return opt == 2 || opt == 4 || opt == 5
}

// Control-command values carried in OSW.Command when it is not a
// channel number. From trunk-recorder's SmartnetParser.
const (
	// CmdIdle is the background idle / heartbeat OSW.
	CmdIdle uint16 = 0x2F8
	// CmdGroupBusy queues a group call when no channel is free.
	CmdGroupBusy uint16 = 0x300
	// CmdEmergencyBusy queues an emergency call.
	CmdEmergencyBusy uint16 = 0x303
	// CmdFirstNormal opens the two/three-OSW sequences: system ID +
	// control-channel broadcast, and analog group voice grants (the
	// first OSW carries the source radio ID).
	CmdFirstNormal uint16 = 0x308
	// CmdFirstAlternate opens the alternate two/three-OSW sequences:
	// system ID + alternate/adjacent control-channel broadcasts and
	// extended functions.
	CmdFirstAlternate uint16 = 0x30B
)

// IsIdle reports whether this OSW is the idle / heartbeat command.
func (o OSW) IsIdle() bool { return o.Command == CmdIdle }
