package tetra

// Bit-accurate CMCE (Circuit-Mode Control Entity) downlink PDU parsing, per
// ETSI EN 300 392-2 V3.8.1 §14. This replaces the byte-aligned ParsePDU +
// AsVoiceGrant/AsRelease guesswork: a real control-channel TM-SDU is
// bit-packed, not byte-aligned, so the L3 PDU must be read one field at a time
// with the MSB-first bitReader (the same reader mac.go uses for MAC-RESOURCE).
//
// The bits handed to ParseCMCE are the TL-SDU — the MLE service PDU — recovered
// by stripping the LLC basic-link header from the MAC TM-SDU (see llc.go /
// ParseLLC). It opens with a 3-bit MLE protocol discriminator (Table 18.1)
// followed by the SDU. When the discriminator selects CMCE, the SDU opens with a
// 5-bit CMCE PDU type
// (Table 14.66) and the type's information elements. Type-1 (mandatory) elements
// are always present in a fixed order; the optional type-2 elements that follow
// are introduced by a single O-bit and then, one per element in table order, a
// presence (P-)bit — see §14 / the general PDU layout (the "Optional bit" +
// "Presence bit" scheme). We walk only as far as the type-2 elements we need
// (the group/temporary address and the calling/transmitting party SSI); the
// type-3/4 (M-bit) chain that would follow is never entered, so trailing
// SS/facility elements can never mis-frame the fields we read.

// CMCEType is the 5-bit downlink CMCE PDU type (EN 300 392-2 Table 14.66).
type CMCEType uint8

const (
	CMCETypeDConnect   CMCEType = 0x02 // D-CONNECT
	CMCETypeDRelease   CMCEType = 0x06 // D-RELEASE
	CMCETypeDSetup     CMCEType = 0x07 // D-SETUP
	CMCETypeDTxCeased  CMCEType = 0x09 // D-TX CEASED
	CMCETypeDTxGranted CMCEType = 0x0B // D-TX GRANTED
)

// cmceMLEPD is the 3-bit MLE protocol discriminator value that selects the CMCE
// SDU (Table 18.1: 010 = CMCE; 001 = MM, 100 = SNDCP, 101 = MLE).
const cmceMLEPD uint32 = 0x2

// callPriorityEmergency is the 4-bit Call priority value that marks an emergency
// call: 1111 = "Pre-emptive priority 4 (Emergency)" (Table 14.46).
const callPriorityEmergency uint8 = 0xF

// CMCEMessage is a decoded downlink CMCE PDU, tagged by Type. Absent optional
// fields are left zero: GroupSSI/PartySSI == 0 means the element was not present,
// and HasPriority == false means no Call priority element was decoded.
type CMCEMessage struct {
	Type           CMCEType
	CallIdentifier uint16 // 14-bit

	HasPriority bool
	Priority    uint8 // 4-bit Call priority (Table 14.46)
	Emergency   bool  // Priority == 1111 (Pre-emptive priority 4 / Emergency)

	GroupSSI uint32 // Temporary address (24-bit GSSI); 0 = absent
	PartySSI uint32 // Calling / Transmitting party SSI (24-bit); 0 = absent

	DisconnectCause uint8 // D-RELEASE only (5-bit)
}

// ParseCMCE decodes a downlink CMCE PDU from the TM-SDU bit slice
// (one-bit-per-byte, MSB first) returned by MACResource.tmSDU — i.e. BEFORE any
// byte packing. It reads the 3-bit MLE protocol discriminator (must be CMCE),
// the 5-bit CMCE PDU type, the mandatory type-1 fields, then the optional type-2
// chain far enough to reach the Temporary address (GSSI) and the
// calling/transmitting party SSI. Returns (zero, false) when the MLE
// discriminator is not CMCE, the PDU type is not one we model, or the slice is
// truncated inside a mandatory field.
func ParseCMCE(bits []byte) (CMCEMessage, bool) {
	r := &bitReader{bits: bits}
	// MLE protocol discriminator (3) + CMCE PDU type (5) = the opening byte.
	if r.remaining() < 8 {
		return CMCEMessage{}, false
	}
	if r.u(3) != cmceMLEPD {
		return CMCEMessage{}, false
	}
	msg := CMCEMessage{Type: CMCEType(r.u(5))}

	switch msg.Type {
	case CMCETypeDSetup:
		// CallId(14) CallTimeout(4) Hook(1) Simplex/duplex(1) BasicSvc(8)
		// TxGrant(2) TxReqPerm(1) CallPriority(4) — all mandatory (Table 14.15).
		if r.remaining() < 14+4+1+1+8+2+1+4 {
			return CMCEMessage{}, false
		}
		msg.CallIdentifier = uint16(r.u(14))
		r.u(4) // call time-out
		r.u(1) // hook method
		r.u(1) // simplex/duplex
		r.u(8) // basic service information
		r.u(2) // transmission grant
		r.u(1) // transmission request permission
		msg.setPriority(uint8(r.u(4)))
		// Optional type-2 chain: Notification indicator(6), Temporary address(24),
		// Calling party type identifier(2) + conditional Calling party SSI.
		if optElementsPresent(r) {
			optSkip(r, 6)              // notification indicator
			msg.GroupSSI = optU(r, 24) // temporary address (GSSI)
			msg.readPartySSI(r)        // calling party type id + conditional SSI
		}

	case CMCETypeDConnect:
		// CallId(14) CallTimeout(4) Hook(1) Simplex/duplex(1) TxGrant(2)
		// TxReqPerm(1) CallOwnership(1) — mandatory (Table 14.7).
		if r.remaining() < 14+4+1+1+2+1+1 {
			return CMCEMessage{}, false
		}
		msg.CallIdentifier = uint16(r.u(14))
		r.u(4) // call time-out
		r.u(1) // hook method
		r.u(1) // simplex/duplex
		r.u(2) // transmission grant
		r.u(1) // transmission request permission
		r.u(1) // call ownership
		// Optional type-2 chain: Call priority(4), Basic service(8),
		// Temporary address(24).
		if optElementsPresent(r) {
			if v, ok := optField(r, 4); ok {
				msg.setPriority(uint8(v))
			}
			optSkip(r, 8)              // basic service information
			msg.GroupSSI = optU(r, 24) // temporary address (GSSI)
		}

	case CMCETypeDRelease:
		// CallId(14) DisconnectCause(5) — mandatory (Table 14.12).
		if r.remaining() < 14+5 {
			return CMCEMessage{}, false
		}
		msg.CallIdentifier = uint16(r.u(14))
		msg.DisconnectCause = uint8(r.u(5))

	case CMCETypeDTxCeased:
		// CallId(14) TxReqPerm(1) — mandatory (Table 14.16).
		if r.remaining() < 14+1 {
			return CMCEMessage{}, false
		}
		msg.CallIdentifier = uint16(r.u(14))
		r.u(1) // transmission request permission

	case CMCETypeDTxGranted:
		// CallId(14) TxGrant(2) TxReqPerm(1) EncControl(1) Reserved(1) —
		// mandatory (Table 14.18).
		if r.remaining() < 14+2+1+1+1 {
			return CMCEMessage{}, false
		}
		msg.CallIdentifier = uint16(r.u(14))
		r.u(2) // transmission grant
		r.u(1) // transmission request permission
		r.u(1) // encryption control
		r.u(1) // reserved
		// Optional type-2: Notification indicator(6), Transmitting party type
		// identifier(2) + conditional Transmitting party SSI.
		if optElementsPresent(r) {
			optSkip(r, 6)       // notification indicator
			msg.readPartySSI(r) // transmitting party type id + conditional SSI
		}

	default:
		return CMCEMessage{}, false
	}
	return msg, true
}

// setPriority records the 4-bit Call priority and derives the emergency flag.
func (m *CMCEMessage) setPriority(p uint8) {
	m.HasPriority = true
	m.Priority = p & 0xF
	m.Emergency = m.Priority == callPriorityEmergency
}

// readPartySSI reads a "party type identifier" type-2 element and its
// conditional address SSI (Calling party in D-SETUP, Transmitting party in
// D-TX-GRANTED). The identifier is a type-2 element (P-bit gated); when present
// its 2-bit value conditions the SSI: 1 ⇒ SSI(24); 2 ⇒ SSI(24) + extension(24,
// skipped). 0 ⇒ no SSI.
func (m *CMCEMessage) readPartySSI(r *bitReader) {
	present, ok := optPBit(r)
	if !ok || !present {
		return
	}
	if r.remaining() < 2 {
		return
	}
	cpti := r.u(2)
	// CPTI 1 ⇒ SSI(24); CPTI 2 ⇒ SSI(24) + extension(24). The SSI and extension
	// are conditional elements (gated by the CPTI value, no P-bit of their own).
	if cpti == 1 || cpti == 2 {
		if r.remaining() < 24 {
			return
		}
		m.PartySSI = r.u(24)
	}
	if cpti == 2 && r.remaining() >= 24 {
		r.u(24) // calling/transmitting party extension (skipped)
	}
}

// optElementsPresent reads the O-bit that introduces the optional type-2 /
// type-3-4 element chain. Returns false when absent (no bits left, or O == 0).
func optElementsPresent(r *bitReader) bool {
	if r.remaining() < 1 {
		return false
	}
	return r.bit() == 1
}

// optPBit reads a type-2 element presence bit. ok is false when the stream is
// exhausted (the chain simply ends — remaining type-2 elements are absent).
func optPBit(r *bitReader) (present bool, ok bool) {
	if r.remaining() < 1 {
		return false, false
	}
	return r.bit() == 1, true
}

// optField reads a fixed-width type-2 element: its P-bit, then n bits of content
// if present. Returns (value, true) only when the element is present and fully
// available.
func optField(r *bitReader, n int) (uint32, bool) {
	present, ok := optPBit(r)
	if !ok || !present {
		return 0, false
	}
	if r.remaining() < n {
		return 0, false
	}
	return r.u(n), true
}

// optU is optField that returns just the value (0 when absent).
func optU(r *bitReader, n int) uint32 {
	v, _ := optField(r, n)
	return v
}

// optSkip advances past a fixed-width type-2 element (P-bit + n content bits when
// present), discarding its value.
func optSkip(r *bitReader, n int) {
	optField(r, n)
}
