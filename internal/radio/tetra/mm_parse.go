package tetra

// MM (Mobility Management) downlink PDU parsing — specifically D-ATTACH/DETACH
// GROUP IDENTITY (EN 300 392-2 §16.9.2.1), the SwMI-initiated group membership
// command: the network telling an MS which talkgroups it is attached to (or
// detached from). Decoding it gives the scanner a live ISSI→GSSI affiliation
// feed — the TETRA analogue of the P25 Group Affiliation Response.
//
// The bit layout is pinned against tetra-kit (decoder/mm/mm.cc
// parseDAttachDetachGroupIdentity + mm_elements.cc parseType34Elements /
// parseGroupIdentityDownlink / parseGroupIdentityAttachment /
// parseAddressExtension), the independent proven decoder that parses these
// PDUs in full; osmo-tetra-sq5bpf corroborates the PDU-type numbering
// (tetra_mm_pdu.h) but does not parse the contents. Per the #764/#771
// discipline nothing here is a private reading of the spec alone.
//
//	PD (3, = 001 MM) + MM PDU type (4, = 1010 D-ATTACH/DETACH GROUP IDENTITY)
//	group identity report request      1
//	group identity ack requested       1
//	group identity attach/detach mode  1  (0 = amendment, 1 = detach all + attach)
//	O-bit                              1
//	  type 3/4 elements, each: M-bit(1) + element id(4) + length indicator(11)
//	  + value(length bits). Element id 0111 = Group identity downlink (type 4):
//	  number of elements(6), then per element (§16.10.22):
//	    attach/detach type indicator   1
//	      0 ⇒ attachment (§16.10.19): lifetime(2) + class of usage(3)
//	      1 ⇒ detachment: detachment downlink(2)
//	    group identity address type    2  (§16.10.15)
//	      00 ⇒ GSSI(24); 01 ⇒ GSSI(24)+ext(MCC 10 + MNC 14)
//	      10 ⇒ (V)GSSI(24); 11 ⇒ GSSI(24)+ext(24)+(V)GSSI(24)

// mmPDMM is the 3-bit protocol discriminator that opens a TL-SDU carrying an
// MM-protocol PDU (EN 300 392-2 Table 18.1 / osmo-tetra TMLE_PDISC_MM = 1).
const mmPDMM uint32 = 0x1

// MMPDUType is the 4-bit downlink MM PDU type (§16.10.39; values pinned by
// osmo-tetra tetra_mm_pdu.h and tetra-kit mm.cc — both agree).
type MMPDUType uint8

const (
	MMDOtar                         MMPDUType = 0x0
	MMDAuthentication               MMPDUType = 0x1
	MMDCKChangeDemand               MMPDUType = 0x2
	MMDDisable                      MMPDUType = 0x3
	MMDEnable                       MMPDUType = 0x4
	MMDLocationUpdateAccept         MMPDUType = 0x5
	MMDLocationUpdateCommand        MMPDUType = 0x6
	MMDLocationUpdateReject         MMPDUType = 0x7
	MMDLocationUpdateProceeding     MMPDUType = 0x9
	MMDAttachDetachGroupIdentity    MMPDUType = 0xA
	MMDAttachDetachGroupIdentityAck MMPDUType = 0xB
	MMDMMStatus                     MMPDUType = 0xC
	MMDPDUNotSupported              MMPDUType = 0xF
)

func (t MMPDUType) String() string {
	switch t {
	case MMDOtar:
		return "D-OTAR"
	case MMDAuthentication:
		return "D-AUTHENTICATION"
	case MMDCKChangeDemand:
		return "D-CK-CHANGE-DEMAND"
	case MMDDisable:
		return "D-DISABLE"
	case MMDEnable:
		return "D-ENABLE"
	case MMDLocationUpdateAccept:
		return "D-LOCATION-UPDATE-ACCEPT"
	case MMDLocationUpdateCommand:
		return "D-LOCATION-UPDATE-COMMAND"
	case MMDLocationUpdateReject:
		return "D-LOCATION-UPDATE-REJECT"
	case MMDLocationUpdateProceeding:
		return "D-LOCATION-UPDATE-PROCEEDING"
	case MMDAttachDetachGroupIdentity:
		return "D-ATTACH/DETACH-GROUP-IDENTITY"
	case MMDAttachDetachGroupIdentityAck:
		return "D-ATTACH/DETACH-GROUP-IDENTITY-ACK"
	case MMDMMStatus:
		return "D-MM-STATUS"
	case MMDPDUNotSupported:
		return "MM-PDU-NOT-SUPPORTED"
	}
	return "MM-RESERVED"
}

// mmTypeGroupIdentityDownlink is the type-3/4 element identifier for the Group
// identity downlink element (§16.10.51 Table 16.89 / tetra-kit
// parseType34Elements case 0b0111).
const mmTypeGroupIdentityDownlink uint32 = 0x7

// GroupIdentityDownlink is one decoded Group identity downlink element
// (§16.10.22): a single group attachment or detachment for the addressed MS.
// Raw field values are kept as broadcast (no spec-derived value mappings).
type GroupIdentityDownlink struct {
	Detach bool
	// DetachReason is the raw 2-bit "group identity detachment downlink" value
	// (detachments only).
	DetachReason uint8
	// Lifetime is the raw 2-bit group identity attachment lifetime;
	// ClassOfUsage the raw 3-bit class of usage (attachments only).
	Lifetime     uint8
	ClassOfUsage uint8

	GSSI uint32 // 24-bit group SSI (or the (V)GSSI for address type 10)
	// Optional address extension (MCC + MNC) when the address type carries one.
	HasExtension bool
	MCC          uint16 // 10-bit
	MNC          uint16 // 14-bit
	// VGSSI is the visitor GSSI when address type 11 carried both.
	HasVGSSI bool
	VGSSI    uint32
}

// DAttachDetachGroupIdentity is the decoded D-ATTACH/DETACH GROUP IDENTITY PDU.
type DAttachDetachGroupIdentity struct {
	ReportRequest bool // group identity report request
	AckRequested  bool // group identity acknowledgement requested
	// DetachAllAttach is the attach/detach mode: true = "detach all and attach"
	// (the listed groups replace every prior attachment), false = amendment.
	DetachAllAttach bool
	Groups          []GroupIdentityDownlink
}

// ParseMMPDUType reports the MM PDU type of a TL-SDU, or false when the TL-SDU
// does not carry an MM-protocol PDU.
func ParseMMPDUType(tl []byte) (MMPDUType, bool) {
	r := &bitReader{bits: tl}
	if r.remaining() < 3+4 {
		return 0, false
	}
	if r.u(3) != mmPDMM {
		return 0, false
	}
	return MMPDUType(r.u(4)), true
}

// ParseDAttachDetachGroupIdentity decodes a TL-SDU as an MM D-ATTACH/DETACH
// GROUP IDENTITY. Returns (zero, false) when the TL-SDU is not one. A PDU that
// carries no Group identity downlink element (or whose type-3/4 chain is
// truncated before one) still decodes with an empty Groups slice — the header
// flags alone are valid content.
func ParseDAttachDetachGroupIdentity(tl []byte) (DAttachDetachGroupIdentity, bool) {
	r := &bitReader{bits: tl}
	if r.remaining() < 3+4+1+1+1+1 {
		return DAttachDetachGroupIdentity{}, false
	}
	if r.u(3) != mmPDMM || MMPDUType(r.u(4)) != MMDAttachDetachGroupIdentity {
		return DAttachDetachGroupIdentity{}, false
	}
	out := DAttachDetachGroupIdentity{
		ReportRequest:   r.bit() == 1,
		AckRequested:    r.bit() == 1,
		DetachAllAttach: r.bit() == 1,
	}
	if r.bit() == 0 { // O-bit: no optional elements
		return out, true
	}
	out.Groups = parseMMType34GroupIdentities(r)
	return out, true
}

// parseMMType34GroupIdentities walks an MM type-3/4 element chain (annex E.1.1:
// M-bit + 4-bit element identifier + 11-bit length indicator + value), decoding
// every Group identity downlink element and skipping the others by their own
// length indicator — so an unmodelled element (security downlink, OTAR, …)
// never desyncs the walk. A length that overruns the remaining bits ends the
// walk (truncated tail), returning what decoded cleanly before it.
func parseMMType34GroupIdentities(r *bitReader) []GroupIdentityDownlink {
	var out []GroupIdentityDownlink
	for r.remaining() >= 1 && r.bit() == 1 { // M-bit: another element follows
		if r.remaining() < 4+11 {
			return out
		}
		id := r.u(4)
		length := int(r.u(11))
		if length == 0 || length > r.remaining() {
			return out
		}
		body := &bitReader{bits: r.bits[r.pos : r.pos+length]}
		r.pos += length
		if id != mmTypeGroupIdentityDownlink {
			continue
		}
		// Type-4 element value: number of repeated elements (6), then each
		// Group identity downlink (tetra-kit parseType34Elements → the
		// per-element parse in parseGroupIdentityDownlink).
		if body.remaining() < 6 {
			continue
		}
		n := int(body.u(6))
		for i := 0; i < n; i++ {
			g, ok := parseGroupIdentityDownlink(body)
			if !ok {
				break // truncated inside the element: keep what decoded cleanly
			}
			out = append(out, g)
		}
	}
	return out
}

// parseGroupIdentityDownlink reads one Group identity downlink (§16.10.22) from
// r. Returns false when the element is truncated mid-field.
func parseGroupIdentityDownlink(r *bitReader) (GroupIdentityDownlink, bool) {
	var g GroupIdentityDownlink
	if r.remaining() < 1 {
		return g, false
	}
	g.Detach = r.bit() == 1
	if g.Detach {
		if r.remaining() < 2 {
			return g, false
		}
		g.DetachReason = uint8(r.u(2))
	} else {
		// Group identity attachment (§16.10.19): lifetime (2) + class of usage (3).
		if r.remaining() < 2+3 {
			return g, false
		}
		g.Lifetime = uint8(r.u(2))
		g.ClassOfUsage = uint8(r.u(3))
	}
	if r.remaining() < 2 {
		return g, false
	}
	addrType := r.u(2)
	if r.remaining() < 24 {
		return g, false
	}
	g.GSSI = r.u(24)
	switch addrType {
	case 1, 3: // GSSI + address extension (MCC 10 + MNC 14, §16.10.1)
		if r.remaining() < 24 {
			return g, false
		}
		g.HasExtension = true
		g.MCC = uint16(r.u(10))
		g.MNC = uint16(r.u(14))
		if addrType == 3 { // + (V)GSSI
			if r.remaining() < 24 {
				return g, false
			}
			g.HasVGSSI = true
			g.VGSSI = r.u(24)
		}
	}
	return g, true
}
