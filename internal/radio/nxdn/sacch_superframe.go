package nxdn

// SACCH superframe (NXDN RTCH / RDCH, cross-checked against DSD-FME's
// nxdn_sacch / MMDVMHost's CNXDNSACCH field layout and pinned on the issue
// #1261 real-air captures). Each traffic frame's 32-bit SACCH information
// block is
//
//	structure 2 | RAN 6 | message 18 | CRC-6
//
// where the structure field counts the superframe DOWN — 3 on the first of
// four consecutive frames, then 2, 1, 0 — and the four 18-bit messages
// concatenate into one 72-bit layer-3 message, normally VCALL:
//
//	bits  0.. 1  reserved / "F1/F2"
//	bits  2.. 7  message type (0x01 VCALL)
//	bits  8..15  call control option
//	bits 16..18  call type
//	bits 19..23  voice call option
//	bits 24..39  source unit ID
//	bits 40..55  destination (group / unit) ID
//	bits 56..57  cipher type (0 clear, 1 scrambler, 2 DES, 3 AES)
//	bits 58..63  key ID
//	bits 64..71  (unused in VCALL)
//
// The superframe start is also where the voice scrambler reseeds its
// keystream (see VoiceCipherTracker).

// SACCHMessageBits is the message payload of one SACCH block.
const SACCHMessageBits = 18

// SACCHSuperframeBits is the layer-3 message four SACCH blocks carry.
const SACCHSuperframeBits = 4 * SACCHMessageBits

// SACCHStructureFirst is the structure field of a superframe's first frame.
const SACCHStructureFirst = 3

// SACCHHeader is the per-block header of a decoded SACCH information block.
type SACCHHeader struct {
	// Structure is the superframe countdown: 3 first … 0 last.
	Structure uint8
	// RAN is the Radio Access Number (colour code).
	RAN uint8
}

// ParseSACCHInfo splits a CRC-valid 32-bit SACCH information block (as
// DecodeSACCH returns it) into its header and 18 message bits.
func ParseSACCHInfo(info []byte) (SACCHHeader, []byte) {
	if len(info) < 8+SACCHMessageBits {
		return SACCHHeader{}, nil
	}
	h := SACCHHeader{Structure: info[0]&1<<1 | info[1]&1}
	for i := 2; i < 8; i++ {
		h.RAN = h.RAN<<1 | info[i]&1
	}
	msg := make([]byte, SACCHMessageBits)
	for i := range msg {
		msg[i] = info[8+i] & 1
	}
	return h, msg
}

// SACCHSuperframe assembles the 72-bit layer-3 message from four
// consecutive SACCH blocks (structure 3, 2, 1, 0). A block out of sequence
// restarts the assembly.
type SACCHSuperframe struct {
	bits []byte
	next int // structure field expected next; -1 = waiting for a 3
}

// Push adds one CRC-valid block. It returns the complete 72-bit message
// when the block was the superframe's last (structure 0) and the three
// before it were seen in order.
func (s *SACCHSuperframe) Push(h SACCHHeader, msg []byte) ([]byte, bool) {
	if len(msg) != SACCHMessageBits {
		return nil, false
	}
	if h.Structure == SACCHStructureFirst {
		s.bits = append(s.bits[:0], msg...)
		s.next = SACCHStructureFirst - 1
		return nil, false
	}
	if s.bits == nil || int(h.Structure) != s.next {
		s.Reset()
		return nil, false
	}
	s.bits = append(s.bits, msg...)
	s.next--
	if h.Structure != 0 {
		return nil, false
	}
	out := append([]byte(nil), s.bits...)
	s.Reset()
	return out, true
}

// Reset drops a partial superframe.
func (s *SACCHSuperframe) Reset() {
	s.bits = s.bits[:0]
	s.next = -1
}

// CipherType is the VCALL cipher field.
type CipherType uint8

const (
	CipherClear     CipherType = 0
	CipherScrambler CipherType = 1 // the 15-bit scrambler (Scrambler)
	CipherDES       CipherType = 2
	CipherAES       CipherType = 3
)

func (c CipherType) String() string {
	switch c {
	case CipherClear:
		return "clear"
	case CipherScrambler:
		return "scrambler"
	case CipherDES:
		return "des"
	case CipherAES:
		return "aes"
	}
	return "cipher?"
}

// VCALLMessage is the SACCH superframe's VCALL (message type 0x01): the
// call's parties and its cipher.
type VCALLMessage struct {
	CallControlOption uint8
	CallType          uint8
	VoiceCallOption   uint8
	Source            uint16
	Destination       uint16
	Cipher            CipherType
	KeyID             uint8
}

// Encrypted reports whether the call carries a cipher.
func (v VCALLMessage) Encrypted() bool { return v.Cipher != CipherClear }

// MessageTypeVCALL is the SACCH / CAC layer-3 VCALL message type.
const MessageTypeVCALL = 0x01

// SACCHMessageType returns the 6-bit message type of a 72-bit superframe.
func SACCHMessageType(sf []byte) uint8 {
	if len(sf) < 8 {
		return 0xFF
	}
	return uint8(bitsUint(sf[2:8]))
}

// ParseVCALLSuperframe decodes a 72-bit SACCH superframe as VCALL; false
// when it is some other message.
func ParseVCALLSuperframe(sf []byte) (VCALLMessage, bool) {
	if len(sf) != SACCHSuperframeBits || SACCHMessageType(sf) != MessageTypeVCALL {
		return VCALLMessage{}, false
	}
	return VCALLMessage{
		CallControlOption: uint8(bitsUint(sf[8:16])),
		CallType:          uint8(bitsUint(sf[16:19])),
		VoiceCallOption:   uint8(bitsUint(sf[19:24])),
		Source:            uint16(bitsUint(sf[24:40])),
		Destination:       uint16(bitsUint(sf[40:56])),
		Cipher:            CipherType(bitsUint(sf[56:58])),
		KeyID:             uint8(bitsUint(sf[58:64])),
	}, true
}

func bitsUint(bits []byte) uint64 {
	var v uint64
	for _, b := range bits {
		v = v<<1 | uint64(b&1)
	}
	return v
}
