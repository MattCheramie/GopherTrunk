package dpmr

import "fmt"

// Mode 3 beacon messages and SYScast fields (ETSI TS 102 658 V2.6.1 clause
// 5.2 and 5.5.32). Every field is read MSB first from the 72 decoded bits,
// in the order the clause 5.2 tables list them (dsdcc reads a traffic
// channel header the same way).

// Message_Type values (table 5.3). The four bits that open every message.
const (
	MTCommunicationsStart = 0x0
	MTConnectionRequest   = 0x1
	MTDisconnectRequest   = 0x2
	MTAck                 = 0x3
	MTMaintenance         = 0x4
	MTSystemRequest       = 0x5
	MTSystemDelivery      = 0x6
	MTStatusResponse      = 0x7
	MTStatusRequest       = 0x8
	MTBSCommand           = 0x9
	MTBSAccess            = 0xA
	MTBroadcast           = 0xB
	MTAhoy                = 0xC // B_AHOY downlink, B_RAND uplink
	MTUDTHeader           = 0xE
	MTUDTAppended         = 0xF
)

// Broadcast kinds: the MI_TYPE field (table 5.29) at bits 61..63 of every
// broadcast message.
const (
	BroadcastAloha        = 0
	BroadcastAnnouncement = 1
	BroadcastMove         = 2
	BroadcastGotoChannel  = 3
	BroadcastAmbience     = 4
)

// BeaconMessage is the 72 information bits of one decoded message field.
type BeaconMessage struct {
	Bits [72]uint8
}

func (m BeaconMessage) field(from, n int) uint32 {
	var v uint32
	for _, b := range m.Bits[from : from+n] {
		v = v<<1 | uint32(b)
	}
	return v
}

// Type is the Message_Type (MT*).
func (m BeaconMessage) Type() uint8 { return uint8(m.field(0, 4)) }

// BroadcastKind is the MI_TYPE of a broadcast message (Broadcast*), or -1
// for any other message.
func (m BeaconMessage) BroadcastKind() int {
	if m.Type() != MTBroadcast {
		return -1
	}
	return int(m.field(61, 3))
}

// Name is a short label for logs.
func (m BeaconMessage) Name() string {
	switch m.Type() {
	case MTBroadcast:
		switch m.BroadcastKind() {
		case BroadcastAloha:
			return "B_ALOHA"
		case BroadcastAnnouncement:
			return "B_BCAST"
		case BroadcastMove:
			return "B_MOVE"
		case BroadcastGotoChannel:
			return "B_GTC"
		case BroadcastAmbience:
			return "AMBIENCE"
		}
		return fmt.Sprintf("BROADCAST(%d)", m.BroadcastKind())
	case MTAhoy:
		return "B_AHOY"
	case MTAck:
		return "B_ACK"
	case MTUDTHeader:
		return "B_UDT_HEADER"
	case MTUDTAppended:
		return "B_UDT_DATA"
	}
	return fmt.Sprintf("MT(%X)", m.Type())
}

// Aloha is a Broadcast Aloha message (table 5.30). The reserved field
// before INFILL is nine bits (its value column reads "9 x 0"; with the ten
// its length column prints, the table adds up to 73 bits).
type Aloha struct {
	MSID         uint32 // the MS addressed, or DUMMYI for all
	SystemCode   uint16 // 14-bit System Identity Code of the radio site
	Infill       bool
	Mask         uint8
	NRandWait    uint8
	Service      uint8 // Service_Function
	Backoff      uint8
	Active       bool // the radio site is connected to the network
	Registration bool // MS must register before access
}

// Aloha parses a Broadcast Aloha message.
func (m BeaconMessage) Aloha() (Aloha, bool) {
	if m.BroadcastKind() != BroadcastAloha {
		return Aloha{}, false
	}
	return Aloha{
		MSID:         m.field(4, 24),
		SystemCode:   uint16(m.field(28, 14)),
		Infill:       m.Bits[51] == 1,
		Mask:         uint8(m.field(52, 5)),
		NRandWait:    uint8(m.field(57, 4)),
		Service:      uint8(m.field(64, 2)),
		Backoff:      uint8(m.field(66, 4)),
		Active:       m.Bits[70] == 1,
		Registration: m.Bits[71] == 1,
	}, true
}

// GotoChannel is a Goto Channel broadcast (table 5.38): the beacon sends
// the called party, or a talkgroup, to a traffic channel. The calling
// party follows in the SYScast2 and SYScast3 of the same frameset.
type GotoChannel struct {
	Called uint32 // MS ID, talkgroup or gateway
	// TxField and RxField are the MS transmit and receive frequencies as
	// coded: BAND + SEP x field (clause 5.5.15).
	TxField, RxField uint16
	Mode             uint8 // communication mode (AirMode*, table 5.55)
	Sep5k            bool  // channel separation 5 kHz (else 3.125 kHz)
	Band             uint8
	Emergency        bool
	Broadcast        bool
}

// GotoChannel parses a Goto Channel broadcast.
func (m BeaconMessage) GotoChannel() (GotoChannel, bool) {
	if m.BroadcastKind() != BroadcastGotoChannel {
		return GotoChannel{}, false
	}
	return GotoChannel{
		Called:    m.field(4, 24),
		TxField:   uint16(m.field(28, 15)),
		RxField:   uint16(m.field(43, 9)<<6 | m.field(55, 6)),
		Mode:      uint8(m.field(52, 3)),
		Sep5k:     m.Bits[64] == 1,
		Band:      uint8(m.field(65, 4)),
		Emergency: m.Bits[69] == 1,
		Broadcast: m.Bits[70] == 1,
	}, true
}

// Voice reports whether the communication mode is a voice call.
func (g GotoChannel) Voice() bool {
	return g.Mode == AirModeVoice || g.Mode == AirModeVoiceSlowData || g.Mode == AirModeVoiceData
}

// RxHz is the frequency the MS receives on: the traffic channel's
// downlink, which a scanner follows. ok is false for the reserved and the
// custom band, which the standard leaves to the operator.
func (g GotoChannel) RxHz() (hz uint32, ok bool) { return frequencyHz(g.Band, g.Sep5k, g.RxField) }

// TxHz is the frequency the MS transmits on (the uplink).
func (g GotoChannel) TxHz() (hz uint32, ok bool) { return frequencyHz(g.Band, g.Sep5k, g.TxField) }

// bandBaseHz is the lower edge of each BAND (table 5.63).
var bandBaseHz = [14]uint32{
	0, 80e6, 160e6, 240e6, 320e6, 400e6, 480e6,
	560e6, 640e6, 700e6, 750e6, 800e6, 850e6, 900e6,
}

// Band codes the standard does not map to a frequency.
const (
	BandReserved = 14
	BandCustom   = 15
)

func frequencyHz(band uint8, sep5k bool, n uint16) (uint32, bool) {
	if int(band) >= len(bandBaseHz) {
		return 0, false
	}
	step := uint32(3125)
	if sep5k {
		step = 5000
	}
	return bandBaseHz[band] + step*uint32(n), true
}

// SYScast types: the SYTYPE field of SYScast2 and SYScast3 (table 5.100).
const (
	SYTypeCallTimer    = 0
	SYTypeLineTimer    = 1
	SYTypeRealTime     = 2
	SYTypeCallingMSB   = 3 // calling party address, most significant 10 bits
	SYTypeCallingLSB   = 4 // calling party address, least significant 14 bits
	SYTypeFrameCounter = 7
)

// SYScast1 is the first field of a SYScast frame (table 5.99).
type SYScast1 struct {
	Registration bool
	SystemCode   uint16
}

// ParseSYScast1 reads a decoded SYScast1 field.
func ParseSYScast1(bits uint32) SYScast1 {
	return SYScast1{Registration: bits>>16&1 == 1, SystemCode: uint16(bits & 0x3FFF)}
}

// SYScastType and SYScastData split a SYScast2 or SYScast3 field.
func SYScastType(bits uint32) uint8  { return uint8(bits >> 14 & 7) }
func SYScastData(bits uint32) uint16 { return uint16(bits & 0x3FFF) }

// CallingParty assembles the calling party address a beacon sends in the
// SYScast2 and SYScast3 after a Goto Channel (clause 5.5.32.2.5).
func CallingParty(sy2, sy3 uint32) (id uint32, ok bool) {
	if SYScastType(sy2) != SYTypeCallingMSB || SYScastType(sy3) != SYTypeCallingLSB {
		return 0, false
	}
	return uint32(SYScastData(sy2)&0x3FF)<<14 | uint32(SYScastData(sy3)), true
}

// Address reverses the B2 mapping of annex A.1.2.1.1.6.1: a 24-bit air
// interface address to its seven dialled characters. K1..K3 are decimal,
// K4..K7 base 11 with "*" as 10. ok is false above the dialable range
// ("999****"), where the gateway and special addresses sit.
func Address(id uint32) (digits string, ok bool) {
	if id > 0xDF6767 {
		return "", false
	}
	w := [7]uint32{1464100, 146410, 14641, 1331, 121, 11, 1}
	var b [7]byte
	for i, wt := range w {
		d := id / wt
		id %= wt
		switch {
		case d == 10 && i >= 3:
			b[i] = '*'
		case d <= 9:
			b[i] = byte('0' + d)
		default:
			return "", false
		}
	}
	return string(b[:]), true
}

// IsWildcardTalkgroup reports whether an address names a talkgroup by
// wildcard (a "*" in K4..K7, clause A.1.3.3.5.2). An all-numeric address
// may still be a talkgroup a radio has in its own table; the air
// interface does not say.
func IsWildcardTalkgroup(id uint32) bool {
	d, ok := Address(id)
	if !ok {
		return false
	}
	for _, c := range d[3:] {
		if c == '*' {
			return true
		}
	}
	return false
}

// AddressID is the B2 mapping itself, dialled characters to the 24-bit
// address. ok is false for a string that is not seven characters of the
// allowed form.
func AddressID(digits string) (id uint32, ok bool) {
	if len(digits) != 7 {
		return 0, false
	}
	w := [7]uint32{1464100, 146410, 14641, 1331, 121, 11, 1}
	for i := 0; i < 7; i++ {
		c := digits[i]
		var d uint32
		switch {
		case c >= '0' && c <= '9':
			d = uint32(c - '0')
		case c == '*' && i >= 3:
			d = 10
		default:
			return 0, false
		}
		id += d * w[i]
	}
	return id, true
}

// Encoders, for test signals and siglab's synthesiser. Each writes the
// fields where the clause 5.2 table puts them.

func (m *BeaconMessage) put(from, n int, v uint32) {
	for i := 0; i < n; i++ {
		m.Bits[from+i] = uint8(v>>uint(n-1-i)) & 1
	}
}

// EncodeAloha builds a Broadcast Aloha message.
func EncodeAloha(a Aloha) BeaconMessage {
	var m BeaconMessage
	m.put(0, 4, MTBroadcast)
	m.put(4, 24, a.MSID)
	m.put(28, 14, uint32(a.SystemCode))
	if a.Infill {
		m.Bits[51] = 1
	}
	m.put(52, 5, uint32(a.Mask))
	m.put(57, 4, uint32(a.NRandWait))
	m.put(61, 3, BroadcastAloha)
	m.put(64, 2, uint32(a.Service))
	m.put(66, 4, uint32(a.Backoff))
	if a.Active {
		m.Bits[70] = 1
	}
	if a.Registration {
		m.Bits[71] = 1
	}
	return m
}

// EncodeGotoChannel builds a Goto Channel broadcast.
func EncodeGotoChannel(g GotoChannel) BeaconMessage {
	var m BeaconMessage
	m.put(0, 4, MTBroadcast)
	m.put(4, 24, g.Called)
	m.put(28, 15, uint32(g.TxField))
	m.put(43, 9, uint32(g.RxField>>6))
	m.put(52, 3, uint32(g.Mode))
	m.put(55, 6, uint32(g.RxField&0x3F))
	m.put(61, 3, BroadcastGotoChannel)
	if g.Sep5k {
		m.Bits[64] = 1
	}
	m.put(65, 4, uint32(g.Band))
	if g.Emergency {
		m.Bits[69] = 1
	}
	if g.Broadcast {
		m.Bits[70] = 1
	}
	return m
}

// EncodeSYScast1 and EncodeSYScast2 build SYScast fields.
func EncodeSYScast1(s SYScast1) uint32 {
	v := uint32(s.SystemCode & 0x3FFF)
	if s.Registration {
		v |= 1 << 16
	}
	return v
}

func EncodeSYScast2(sytype uint8, data uint16) uint32 {
	return uint32(sytype&7)<<14 | uint32(data&0x3FFF)
}

// EncodeCallingParty returns the SYScast2 and SYScast3 that carry a
// calling party address.
func EncodeCallingParty(id uint32) (sy2, sy3 uint32) {
	return EncodeSYScast2(SYTypeCallingMSB, uint16(id>>14&0x3FF)), EncodeSYScast2(SYTypeCallingLSB, uint16(id&0x3FFF))
}

// FrequencyFields is the inverse of RxHz / TxHz: the band, separation and
// field that code hz, choosing the lowest band that reaches it. ok is
// false when no band codes it exactly.
func FrequencyFields(hz uint32) (band uint8, sep5k bool, field uint16, ok bool) {
	for b := range bandBaseHz {
		if hz < bandBaseHz[b] {
			break
		}
		off := hz - bandBaseHz[b]
		for _, s := range []struct {
			step uint32
			five bool
		}{{3125, false}, {5000, true}} {
			if off%s.step == 0 && off/s.step < 1<<15 {
				return uint8(b), s.five, uint16(off / s.step), true
			}
		}
	}
	return 0, false, 0, false
}
