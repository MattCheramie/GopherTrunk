package tetra

// TETRA lower-MAC PDU parsing (ETSI EN 300 392-2 §21). The physical layer
// recovers type-1 bits for a logical channel (SCH/F, SCH/HD); those bits are a
// MAC PDU, not a Layer-3 PDU. The MAC PDU frames the TM-SDU (the LLC/L3 PDU) and
// — for MAC-RESOURCE — carries the channel-allocation element that assigns a
// call's traffic carrier + timeslot. A CMCE voice grant is therefore reached
// only by parsing the MAC header, extracting the TM-SDU, and (for the physical
// resource) reading the MAC channel-allocation element.
//
// Field widths follow EN 300 392-2 §21.4.3.1 (MAC-RESOURCE) and §21.5.2
// (channel allocation), cross-checked against osmo-tetra's decoder. Bits are
// carried one-per-byte, MSB first — the shape DecodeSCHF / DecodeSCHHD return.

// MACPDUType is the 2-bit MAC PDU type that opens every downlink MAC PDU
// (§21.4.1).
type MACPDUType uint8

const (
	MACPDUResource  MACPDUType = 0 // MAC-RESOURCE — carries a TM-SDU + optional grant
	MACPDUFragEnd   MACPDUType = 1 // MAC-FRAG (subtype 0) / MAC-END (subtype 1)
	MACPDUBroadcast MACPDUType = 2 // SYSINFO / ACCESS-DEFINE
	MACPDUSuppl     MACPDUType = 3 // MAC-U-SIGNAL / supplementary
)

// MAC-FRAG/END subtype (the bit following the 2-bit MAC PDU type when
// MACPDUType == MACFragEnd).
const (
	macFrag uint8 = 0
	macEnd  uint8 = 1
)

// Address types (§21.4.3.1). addrLenBits gives the address element width for
// each; NULL carries no address (and the PDU no TM-SDU).
const (
	addrNull     uint8 = 0
	addrSSI      uint8 = 1
	addrEventLbl uint8 = 2
	addrUSSI     uint8 = 3
	addrSMI      uint8 = 4
	addrSSIEvent uint8 = 5
	addrSSIUsage uint8 = 6
	addrSMIEvent uint8 = 7
)

var addrLenBits = [8]int{
	addrNull:     0,
	addrSSI:      24,
	addrEventLbl: 10,
	addrUSSI:     24,
	addrSMI:      24,
	addrSSIEvent: 34, // 24 + 10
	addrSSIUsage: 30, // 24 + 6
	addrSMIEvent: 34, // 24 + 10
}

// macLenStartFragment / macLenSecondStolen are the reserved length-indication
// values (§21.4.3.1): a start fragment continues the TM-SDU in a following
// MAC-FRAG/MAC-END; the second-stolen marker flags a stolen half-slot.
const (
	macLenStartFragment uint8 = 0x3f
	macLenSecondStolen  uint8 = 0x3e
)

// bitReader walks a one-bit-per-byte slice MSB-first.
type bitReader struct {
	bits []byte
	pos  int
}

func (r *bitReader) remaining() int { return len(r.bits) - r.pos }

// u reads n bits as an unsigned integer, MSB first. A read past the end
// returns whatever bits are available shifted into place and marks the reader
// exhausted by advancing pos past the end.
func (r *bitReader) u(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v <<= 1
		if r.pos < len(r.bits) {
			v |= uint32(r.bits[r.pos] & 1)
		}
		r.pos++
	}
	return v
}

func (r *bitReader) bit() uint8 { return uint8(r.u(1)) }

// MACAddress is the addressed party of a MAC-RESOURCE PDU (§21.4.3.1).
type MACAddress struct {
	Type        uint8
	SSI         uint32 // 24-bit short subscriber identity (individual or group)
	EventLabel  uint16 // 10-bit
	UsageMarker uint8  // 6-bit (addrSSIUsage)
}

// ChannelAllocation is the MAC channel-allocation element (§21.5.2): the
// physical traffic resource assigned to a call. CarrierNumber resolves to Hz
// via the band plan; Timeslot is the assigned downlink slot.
type ChannelAllocation struct {
	AllocationType uint8
	Timeslot       uint8 // 4-bit assigned-timeslot field
	ULDL           uint8 // 2-bit up/downlink assignment
	CLCHPermission bool
	CellChange     bool
	CarrierNumber  uint16 // 12-bit main carrier number
	ExtCarrier     bool
	FreqBand       uint8 // 4-bit (ext carrier)
	FreqOffset     uint8 // 2-bit (ext carrier)
	DuplexSpacing  uint8 // 3-bit (ext carrier)
	ReverseOper    bool  // (ext carrier)
	MonitPattern   uint8 // 2-bit
}

// MACResource is a decoded downlink MAC-RESOURCE PDU. TMSDUBitOffset is the bit
// index (into the type-1 bit slice passed to ParseMACResource) at which the
// TM-SDU (Layer-3 PDU) begins.
type MACResource struct {
	FillBits       bool
	GrantPosition  bool
	EncryptionMode uint8
	Encrypted      bool
	RandomAccess   bool
	LengthInd      uint8
	NullPDU        bool // address type NULL — no TM-SDU
	StartFragment  bool // TM-SDU continues in a following MAC-FRAG/MAC-END
	Address        MACAddress
	SlotGranting   bool
	ChanAlloc      *ChannelAllocation // nil when no allocation present
	TMSDUBitOffset int
}

// parseChanAlloc reads the channel-allocation element at the reader's cursor
// (§21.5.2). Mirrors osmo-tetra's macpdu_decode_chan_alloc, including the
// extended-carrier and augmented (ul_dl==0) branches so the cursor lands
// correctly past the element.
func parseChanAlloc(r *bitReader) ChannelAllocation {
	ca := ChannelAllocation{
		AllocationType: uint8(r.u(2)),
		Timeslot:       uint8(r.u(4)),
		ULDL:           uint8(r.u(2)),
		CLCHPermission: r.bit() == 1,
		CellChange:     r.bit() == 1,
		CarrierNumber:  uint16(r.u(12)),
	}
	if r.bit() == 1 {
		ca.ExtCarrier = true
		ca.FreqBand = uint8(r.u(4))
		ca.FreqOffset = uint8(r.u(2))
		ca.DuplexSpacing = uint8(r.u(3))
		ca.ReverseOper = r.bit() == 1
	}
	ca.MonitPattern = uint8(r.u(2))
	if ca.MonitPattern == 0 {
		r.u(2) // monitoring pattern for frame 18
	}
	if ca.ULDL == 0 {
		// Augmented channel-allocation element (§21.5.2a).
		r.u(2) // up/downlink assigned
		r.u(3) // bandwidth
		r.u(3) // modulation
		r.u(3) // max uplink QAM modulation
		r.u(3) // reserved
		r.u(3) // conforming channel status
		r.u(4) // BS link imbalance
		r.u(5) // BS transmit power relative to main carrier
		nap := r.u(2)
		if nap == 1 {
			r.u(11) // napping information (§21.5.2c)
		}
		r.u(4) // reserved
		if r.bit() == 1 {
			r.u(16)
		}
		if r.bit() == 1 {
			r.u(16)
		}
		r.u(1)
	}
	return ca
}

// ParseMACResource decodes a downlink MAC-RESOURCE PDU from type-1 bits
// (one-per-byte, MSB first). Returns (zero, false) if the leading MAC PDU type
// is not MAC-RESOURCE or the header is truncated. A NULL-address PDU returns
// with NullPDU set and no TM-SDU. isDecrypted lets the caller assert a payload
// has already been deciphered so a channel-allocation element behind an
// encryption flag is still parsed.
func ParseMACResource(bits []byte, isDecrypted bool) (MACResource, bool) {
	if len(bits) < 16 {
		return MACResource{}, false
	}
	r := &bitReader{bits: bits}
	if MACPDUType(r.u(2)) != MACPDUResource {
		return MACResource{}, false
	}
	m := MACResource{
		FillBits:      r.bit() == 1,
		GrantPosition: r.bit() == 1,
	}
	m.EncryptionMode = uint8(r.u(2))
	m.Encrypted = m.EncryptionMode > 0 && !isDecrypted
	m.RandomAccess = r.bit() == 1
	m.LengthInd = uint8(r.u(6))
	m.StartFragment = m.LengthInd == macLenStartFragment
	m.Address.Type = uint8(r.u(3))
	if m.Address.Type == addrNull {
		m.NullPDU = true
		m.TMSDUBitOffset = r.pos
		return m, true
	}
	// Read the address fields, then advance the cursor by the type's full width.
	addrStart := r.pos
	switch m.Address.Type {
	case addrSSI, addrUSSI, addrSMI:
		m.Address.SSI = r.u(24)
	case addrEventLbl:
		m.Address.EventLabel = uint16(r.u(10))
	case addrSSIEvent, addrSMIEvent:
		m.Address.SSI = r.u(24)
		m.Address.EventLabel = uint16(r.u(10))
	case addrSSIUsage:
		m.Address.SSI = r.u(24)
		m.Address.UsageMarker = uint8(r.u(6))
	default:
		return MACResource{}, false
	}
	r.pos = addrStart + addrLenBits[m.Address.Type]

	if r.bit() == 1 { // power control present
		r.u(4)
	}
	if r.bit() == 1 { // slot granting present
		m.SlotGranting = true
		r.u(4) // number of slots granted
		r.u(4) // delay
	}
	if r.bit() == 1 && !m.Encrypted { // channel allocation present
		ca := parseChanAlloc(r)
		m.ChanAlloc = &ca
	}
	m.TMSDUBitOffset = r.pos
	return m, true
}

// tmSDU returns the TM-SDU (Layer-3 PDU) bits following the MAC header, as a
// one-bit-per-byte slice. Returns nil when the offset runs past the block or
// the PDU carries no TM-SDU (null / start-fragment-with-nothing).
func (m MACResource) tmSDU(bits []byte) []byte {
	if m.NullPDU || m.TMSDUBitOffset >= len(bits) {
		return nil
	}
	return bits[m.TMSDUBitOffset:]
}

// macFragmentSubtype reports the MAC-FRAG/MAC-END subtype for a PDU whose
// leading MAC PDU type is MACFragEnd.
func macFragmentSubtype(bits []byte) (uint8, bool) {
	if len(bits) < 3 {
		return 0, false
	}
	r := &bitReader{bits: bits}
	if MACPDUType(r.u(2)) != MACPDUFragEnd {
		return 0, false
	}
	return r.bit(), true
}

// macFragmentPayload returns the fragment payload bits of a MAC-FRAG/MAC-END
// PDU. MAC-FRAG (§21.4.3.3): 2-bit type + 1-bit subtype, then the fragment.
// MAC-END additionally carries a fill-bit flag + length indication before the
// fragment (§21.4.3.4).
func macFragmentPayload(bits []byte) []byte {
	sub, ok := macFragmentSubtype(bits)
	if !ok {
		return nil
	}
	r := &bitReader{bits: bits}
	r.u(3) // MAC PDU type + subtype
	if sub == macEnd {
		r.bit() // fill-bit indication
		r.u(6)  // length indication
	}
	if r.pos >= len(bits) {
		return nil
	}
	return bits[r.pos:]
}
