package tetra

// TETRA LLC (Logical Link Control) sublayer — the layer that sits between the
// MAC and the MLE, per ETSI EN 300 392-2 V3.8.1 §21.2. The MAC delivers a
// TM-SDU (via the TMA-SAP) that is an *LLC PDU*, not the MLE PDU directly: a
// basic-link header (LLC PDU type + sequence numbers, and an optional 32-bit
// FCS trailer) wraps the TL-SDU, and the TL-SDU is what carries the 3-bit MLE
// protocol discriminator + CMCE/MM/… PDU. Skipping this layer — feeding the raw
// MAC TM-SDU straight into ParseCMCE — reads every L3 field at the wrong bit
// offset (the basic-link header is 4–6 bits wide), which corrupts the decoded
// ISSI/GSSI. osmo-tetra strips it in tetra_llc_pdu_parse() before
// tetra_mle_decode(); tetra-kit does the same in llc.cc. We handle the basic
// link (the sublayer used for CMCE call-control signalling on the MCCH); the
// advanced link (AL-*, used for packet data / SNDCP) is not a CMCE carrier and
// is reported as not-basic-link.

// llcPDUType is the 4-bit LLC PDU type (§21.2.1, Table 21.1). Only the basic
// link values (0..7) are handled here; 8..15 are advanced-link / supplementary
// / layer-2-signalling PDUs that do not carry CMCE call control.
type llcPDUType uint8

const (
	llcBLADATA    llcPDUType = 0x0 // BL-ADATA without FCS
	llcBLDATA     llcPDUType = 0x1 // BL-DATA without FCS
	llcBLUDATA    llcPDUType = 0x2 // BL-UDATA without FCS
	llcBLACK      llcPDUType = 0x3 // BL-ACK without FCS
	llcBLADATAFCS llcPDUType = 0x4 // BL-ADATA with FCS
	llcBLDATAFCS  llcPDUType = 0x5 // BL-DATA with FCS
	llcBLUDATAFCS llcPDUType = 0x6 // BL-UDATA with FCS
	llcBLACKFCS   llcPDUType = 0x7 // BL-ACK with FCS
)

// llcFCSBits is the width of the Frame Check Sequence carried by the "with FCS"
// basic-link PDUs (Tables 21.5/21.7/21.9/21.11), stripped from the tail of the
// TL-SDU before it is handed up.
const llcFCSBits = 32

// ParseLLC strips the basic-link LLC header (and any trailing FCS) from a MAC
// TM-SDU and returns the TL-SDU (the MLE PDU) as a one-bit-per-byte slice, ready
// for ParseCMCE / PDUFromBits. It reports ok=false when the PDU is not a
// basic-link PDU (advanced link, supplementary, or layer-2 signalling — none of
// which carry CMCE call control), or when the slice is too short to hold the
// header.
//
// Header widths (§21.2.2, Tables 21.4/21.6/21.8/21.10):
//
//	BL-ADATA: type(4) + N(R)(1) + N(S)(1)   = 6 bits
//	BL-DATA:  type(4) + N(S)(1)             = 5 bits
//	BL-ACK:   type(4) + N(R)(1)             = 5 bits
//	BL-UDATA: type(4)                       = 4 bits
//
// The "with FCS" variants carry the same header and an extra 32-bit FCS at the
// end of the PDU (after the TL-SDU), which is excluded from the returned slice.
func ParseLLC(bits []byte) (tlSDU []byte, ok bool) {
	if len(bits) < 4 {
		return nil, false
	}
	r := &bitReader{bits: bits}
	t := llcPDUType(r.u(4))

	hasFCS := false
	switch t {
	case llcBLADATA:
		r.u(2) // N(R) + N(S)
	case llcBLDATA:
		r.u(1) // N(S)
	case llcBLACK:
		r.u(1) // N(R)
	case llcBLUDATA:
		// header is the 4-bit type only
	case llcBLADATAFCS:
		r.u(2)
		hasFCS = true
	case llcBLDATAFCS, llcBLACKFCS:
		r.u(1)
		hasFCS = true
	case llcBLUDATAFCS:
		hasFCS = true
	default:
		// Advanced-link / supplementary / L2-signalling PDU — not a CMCE carrier.
		return nil, false
	}

	end := len(bits)
	if hasFCS {
		end -= llcFCSBits
	}
	if r.pos >= end {
		// Header consumed everything (or the FCS overlaps the header): no TL-SDU.
		// BL-ACK legitimately carries no TL-SDU; treat as an empty basic-link PDU.
		return nil, false
	}
	return bits[r.pos:end], true
}
