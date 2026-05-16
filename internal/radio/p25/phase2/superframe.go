package phase2

// Superframe / slot layout per TIA-102.BBAB §7. The Phase 2 traffic
// channel runs at 6000 sym/sec and is organised as a 360 ms
// superframe carrying 12 sub-frames, each 30 ms long, alternating
// between the two timeslots. Each sub-frame in turn is composed of
// either a 4-voice / 2-voice slot or a MAC slot, identified by the
// SlotType field that prefixes the sub-frame.
const (
	SuperframeMs           = 360
	SubframeMs             = 30
	SubframesPerSuperframe = 12
	TimeslotsPerSuperframe = 2
	SymbolsPerSecond       = 6000
	DibitsPerSubframe      = (SymbolsPerSecond * SubframeMs) / 1000 / 1 // 180
)

// SlotType is the 4-bit identifier that names what a sub-frame
// contains. The full enum is defined in TIA-102.BBAB Table 7.x; the
// subset below covers what the trunking layer cares about.
type SlotType uint8

const (
	SlotTypeUnknown      SlotType = 0x0
	SlotTypeVoice4V      SlotType = 0x1 // 4 voice frames
	SlotTypeVoice2V      SlotType = 0x2 // 2 voice frames + MAC
	SlotTypeMACPTT       SlotType = 0x3 // MAC PTT signalling
	SlotTypeMACEnd       SlotType = 0x4 // MAC end of transmission
	SlotTypeMACIdle      SlotType = 0x5 // MAC channel idle
	SlotTypeMACActive    SlotType = 0x6 // MAC active update
	SlotTypeMACHangtime  SlotType = 0x7 // MAC hang-time
	SlotTypeMACSignaling SlotType = 0x8
	SlotTypeMACEndCont   SlotType = 0x9 // MAC end + continuation
)

// String returns a stable human-readable label for log output.
func (s SlotType) String() string {
	switch s {
	case SlotTypeVoice4V:
		return "Voice4V"
	case SlotTypeVoice2V:
		return "Voice2V"
	case SlotTypeMACPTT:
		return "MAC_PTT"
	case SlotTypeMACEnd:
		return "MAC_END"
	case SlotTypeMACIdle:
		return "MAC_IDLE"
	case SlotTypeMACActive:
		return "MAC_ACTIVE"
	case SlotTypeMACHangtime:
		return "MAC_HANGTIME"
	case SlotTypeMACSignaling:
		return "MAC_SIGNALING"
	case SlotTypeMACEndCont:
		return "MAC_END_CONT"
	default:
		return "Unknown"
	}
}

// IsMAC reports whether the slot type carries a MAC PDU rather than
// raw voice frames.
func (s SlotType) IsMAC() bool {
	switch s {
	case SlotTypeMACPTT, SlotTypeMACEnd, SlotTypeMACIdle,
		SlotTypeMACActive, SlotTypeMACHangtime, SlotTypeMACSignaling,
		SlotTypeMACEndCont:
		return true
	}
	return false
}

// IsVoice reports whether the slot type carries voice frames.
func (s SlotType) IsVoice() bool {
	return s == SlotTypeVoice4V || s == SlotTypeVoice2V
}
