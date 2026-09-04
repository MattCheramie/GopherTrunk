package tetra

import "encoding/hex"

// MLE (Mobile Link Entity) downlink PDU parsing — specifically D-NWRK-BROADCAST
// (EN 300 392-2 §18.4.1.4.1), the broadcast that advertises the serving cell's
// re-selection parameters and its NEIGHBOUR CELLS: per-cell main carrier (and
// optionally the full band/offset/duplex extension, MCC/MNC and Location Area).
// It is the TETRA analogue of the P25 adjacent-status broadcast, and what fills
// the "Neighbor sites" half of the systems report that used to be empty on a
// TETRA rig.
//
// The bit layout is pinned against two independent proven decoders rather than
// a private reading of the spec (the #764/#771 self-consistent-synthetic
// discipline): osmo-tetra-sq5bpf's parse_d_nwrk_broadcast/parse_nci_ca
// (tetra_upper_mac.c) and tetra-kit's processDNwrkBroadcast/
// parseNeighbourCellInformation (decoder/mle/mle.cc). Both agree:
//
//	MLE PD (3, = 101 MLE) + PDU type (3, = 010 D-NWRK-BROADCAST)
//	cell re-select parameters  16
//	cell service level          2
//	O-bit                       1
//	  P: TETRA network time    48  (24 UTC + 1 sign + 6 offset + 6 year + 11 rsvd)
//	  P: number of neighbours   3, then per cell:
//	     cell identifier              5
//	     cell reselection types       2
//	     neighbour cell synchronized  1
//	     cell service level           2
//	     main carrier number         12
//	     O-bit                        1
//	       P: main carrier extension 10 (band 4 + offset 2 + duplex 3 + reverse 1)
//	       P: MCC                    10
//	       P: MNC                    14
//	       P: LA                     14
//	       P: max MS tx power         3
//	       P: min RX access level     4
//	       P: subscriber class       16
//	       P: BS service details     12
//	       P: timeshare + security    5
//	       P: TDMA frame offset       6

// mlePDMLE is the 3-bit protocol discriminator that opens a TL-SDU carrying an
// MLE-protocol PDU (EN 300 392-2 Table 18.1: 101 = MLE; 010 = CMCE, see
// cmceMLEPD).
const mlePDMLE uint32 = 0x5

// packBitsHex packs a one-bit-per-byte slice into MSB-first hex (the inverse of
// the test helper hexToBits), zero-padding the final nibble group. Used to dump
// a rejected TL-SDU into the log so a mis-framed layout can be pinned from a
// field report alone.
func packBitsHex(bits []byte) string {
	if len(bits) == 0 {
		return ""
	}
	packed := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b != 0 {
			packed[i/8] |= 0x80 >> (i % 8)
		}
	}
	return hex.EncodeToString(packed)
}

// Downlink MLE-subsystem PDU types (EN 300 392-2 §18.5.20; the numbering is
// pinned by BOTH osmo-tetra-sq5bpf tetra_mle_pdu.h (tetra_mle_pdu_type_d) and
// tetra-kit mle.cc serviceMleSubsystem — they agree on all seven values).
const (
	mlePDUTypeDNewCell          uint32 = 0x0 // D-NEW-CELL
	mlePDUTypeDPrepareFail      uint32 = 0x1 // D-PREPARE-FAIL
	mlePDUTypeDNwrkBroadcast    uint32 = 0x2 // D-NWRK-BROADCAST
	mlePDUTypeDNwrkBroadcastExt uint32 = 0x3 // D-NWRK-BROADCAST-EXTENSION
	mlePDUTypeDRestoreAck       uint32 = 0x4 // D-RESTORE-ACK (wraps a CMCE SDU)
	mlePDUTypeDRestoreFail      uint32 = 0x5 // D-RESTORE-FAIL
	mlePDUTypeDChannelResponse  uint32 = 0x6 // D-CHANNEL-RESPONSE
)

// mleSubsystemPDUName names a downlink MLE-subsystem PDU type for logging.
func mleSubsystemPDUName(t uint32) string {
	switch t {
	case mlePDUTypeDNewCell:
		return "D-NEW-CELL"
	case mlePDUTypeDPrepareFail:
		return "D-PREPARE-FAIL"
	case mlePDUTypeDNwrkBroadcast:
		return "D-NWRK-BROADCAST"
	case mlePDUTypeDNwrkBroadcastExt:
		return "D-NWRK-BROADCAST-EXTENSION"
	case mlePDUTypeDRestoreAck:
		return "D-RESTORE-ACK"
	case mlePDUTypeDRestoreFail:
		return "D-RESTORE-FAIL"
	case mlePDUTypeDChannelResponse:
		return "D-CHANNEL-RESPONSE"
	}
	return "MLE-RESERVED"
}

// ParseMLESubsystemType reports the MLE-subsystem PDU type of a TL-SDU, or
// false when the TL-SDU does not carry an MLE-subsystem PDU.
func ParseMLESubsystemType(tl []byte) (uint32, bool) {
	r := &bitReader{bits: tl}
	if r.remaining() < 3+3 {
		return 0, false
	}
	if r.u(3) != mlePDMLE {
		return 0, false
	}
	return r.u(3), true
}

// ParseCMCERestoreAck decodes the CMCE SDU an MLE D-RESTORE-ACK carries — the
// call-restoration signalling sent to an MS that re-selected onto this cell
// mid-call. tetra-kit's serviceMleSubsystem (mle.cc, case 0b100) forwards the
// payload after the 3-bit MLE PDU type straight to its CMCE layer, which reads
// a 5-bit CMCE PDU type first — exactly parseCMCEBody's framing. Returns
// (zero, false) when the TL-SDU is not a D-RESTORE-ACK or the wrapped SDU is
// not a modelled CMCE PDU.
func ParseCMCERestoreAck(tl []byte) (CMCEMessage, bool) {
	r := &bitReader{bits: tl}
	if r.remaining() < 3+3 {
		return CMCEMessage{}, false
	}
	if r.u(3) != mlePDMLE || r.u(3) != mlePDUTypeDRestoreAck {
		return CMCEMessage{}, false
	}
	return parseCMCEBody(r)
}

// NeighbourCell is one neighbour advertised by D-NWRK-BROADCAST. MainCarrier
// is always present; the frequency extension (band/offset/duplex) and the
// MCC/MNC/LA identity are optional on air — Has* reports what the broadcast
// actually carried.
type NeighbourCell struct {
	CellID           uint8  // 5-bit cell identifier
	ReselectTypes    uint8  // 2-bit cell re-selection types supported
	Synchronized     bool   // neighbour cell synchronized flag
	CellServiceLevel uint8  // 2-bit cell service level (load)
	MainCarrier      uint16 // 12-bit main carrier number

	// Optional main-carrier extension: without it the neighbour's carrier is
	// interpreted in the SERVING cell's frequency band (the spec's default).
	HasExtension  bool
	FreqBand      uint8 // 4-bit frequency band
	Offset        uint8 // 2-bit offset field (same coding as SYSINFO)
	DuplexSpacing uint8 // 3-bit duplex spacing selector
	ReverseOper   bool  // reverse operation

	HasMCC bool
	MCC    uint16 // 10-bit
	HasMNC bool
	MNC    uint16 // 14-bit
	HasLA  bool
	LA     uint16 // 14-bit location area

	// Remaining optional per-cell status elements (§18.5.17), surfaced as RAW
	// field values. BS service details additionally renders named bits via
	// BSServiceDetailsString (sysinfo_ext.go): an earlier note here claimed no
	// reference decoder itemises them, but tetra-kit's parseBsServiceDetails
	// (decoder/mle/mle_elements.cc) names all 12 (registration,
	// de-registration, priority cell, minimum mode, migration, system wide
	// services, voice, circuit data, reserved, SNDCP, air encryption, advanced
	// link — MSB first), which is the independent corroboration the CommsType
	// discipline requires. The other raw fields (subscriber class, timeshare)
	// stay un-named — still no decoder itemises those.
	HasMaxTxPower      bool
	MaxTxPower         uint8 // 3-bit maximum MS transmit power
	HasMinRxLevel      bool
	MinRxLevel         uint8 // 4-bit minimum RX access level
	HasSubscriberClass bool
	SubscriberClass    uint16 // 16-bit allowed subscriber classes bitmap
	HasServiceDetails  bool
	ServiceDetails     uint16 // 12-bit BS service details bitmap
	HasTimeshare       bool
	Timeshare          uint8 // 5-bit timeshare cell / security parameters
	HasFrameOffset     bool
	FrameOffset        uint8 // 6-bit TDMA frame offset
}

// CellLoadName renders a 2-bit cell service level as the load it advertises
// (§18.5.5): 0 = unknown, 1 = low, 2 = medium, 3 = high.
func CellLoadName(level uint8) string {
	switch level & 0x3 {
	case 1:
		return "low"
	case 2:
		return "medium"
	case 3:
		return "high"
	}
	return "unknown"
}

// DNwrkBroadcast is the decoded D-NWRK-BROADCAST PDU.
type DNwrkBroadcast struct {
	CellReselectParams uint16 // 16-bit serving-cell re-selection parameters
	CellServiceLevel   uint8  // 2-bit serving-cell service level
	Neighbours         []NeighbourCell
}

// ParseDNwrkBroadcast decodes a TL-SDU (one-bit-per-byte, as returned by
// ParseLLC) as an MLE D-NWRK-BROADCAST. Returns (zero, false) when the TL-SDU
// is not one, or is too short for the fields its own flags promise (a truncated
// neighbour list decodes no cells rather than garbage).
func ParseDNwrkBroadcast(tl []byte) (DNwrkBroadcast, bool) {
	r := &bitReader{bits: tl}
	if r.remaining() < 3+3+16+2+1 {
		return DNwrkBroadcast{}, false
	}
	if r.u(3) != mlePDMLE || r.u(3) != mlePDUTypeDNwrkBroadcast {
		return DNwrkBroadcast{}, false
	}
	out := DNwrkBroadcast{
		CellReselectParams: uint16(r.u(16)),
		CellServiceLevel:   uint8(r.u(2)),
	}
	if r.bit() == 0 { // O-bit: no optional elements at all
		return out, true
	}
	if r.remaining() < 1 {
		return DNwrkBroadcast{}, false
	}
	if r.bit() == 1 { // P: TETRA network time (48 bits; not surfaced)
		if r.remaining() < 48 {
			return DNwrkBroadcast{}, false
		}
		r.u(32)
		r.u(16)
	}
	if r.remaining() < 1 {
		return DNwrkBroadcast{}, false
	}
	if r.bit() == 1 { // P: number of neighbour cells
		if r.remaining() < 3 {
			return DNwrkBroadcast{}, false
		}
		n := int(r.u(3))
		for i := 0; i < n; i++ {
			cell, ok := parseNeighbourCell(r)
			if !ok {
				// A truncated/misframed tail invalidates the whole list — a
				// partially-read neighbour would carry misaligned field values.
				return DNwrkBroadcast{}, false
			}
			out.Neighbours = append(out.Neighbours, cell)
		}
	}
	return out, true
}

// parseNeighbourCell reads one Neighbour cell information element (§18.5.17)
// from r. Every optional field is decoded with a Has* presence flag, so
// consumers can distinguish "absent" from a genuine zero.
func parseNeighbourCell(r *bitReader) (NeighbourCell, bool) {
	if r.remaining() < 5+2+1+2+12+1 {
		return NeighbourCell{}, false
	}
	cell := NeighbourCell{
		CellID:           uint8(r.u(5)),
		ReselectTypes:    uint8(r.u(2)),
		Synchronized:     r.bit() == 1,
		CellServiceLevel: uint8(r.u(2)),
		MainCarrier:      uint16(r.u(12)),
	}
	if r.bit() == 0 { // O-bit: no optional fields for this cell
		return cell, true
	}
	// P-bit-gated optionals, in broadcast order. Each read is preceded by a
	// remaining() check so a truncated element reports failure instead of
	// zero-filled fields.
	type opt struct {
		bits int
		read func()
	}
	opts := []opt{
		{10, func() {
			cell.HasExtension = true
			cell.FreqBand = uint8(r.u(4))
			cell.Offset = uint8(r.u(2))
			cell.DuplexSpacing = uint8(r.u(3))
			cell.ReverseOper = r.bit() == 1
		}},
		{10, func() { cell.HasMCC = true; cell.MCC = uint16(r.u(10)) }},
		{14, func() { cell.HasMNC = true; cell.MNC = uint16(r.u(14)) }},
		{14, func() { cell.HasLA = true; cell.LA = uint16(r.u(14)) }},
		{3, func() { cell.HasMaxTxPower = true; cell.MaxTxPower = uint8(r.u(3)) }},
		{4, func() { cell.HasMinRxLevel = true; cell.MinRxLevel = uint8(r.u(4)) }},
		{16, func() { cell.HasSubscriberClass = true; cell.SubscriberClass = uint16(r.u(16)) }},
		{12, func() { cell.HasServiceDetails = true; cell.ServiceDetails = uint16(r.u(12)) }},
		{5, func() { cell.HasTimeshare = true; cell.Timeshare = uint8(r.u(5)) }},
		{6, func() { cell.HasFrameOffset = true; cell.FrameOffset = uint8(r.u(6)) }},
	}
	for _, o := range opts {
		if r.remaining() < 1 {
			// PDU ends exactly at a P-bit position: the MAC/LLC layer strips
			// trailing fill, so absent trailing P-bits read as "not present".
			// (Mid-list truncation is still caught: the next cell's mandatory
			// fields won't fit and the caller invalidates the whole list.)
			return cell, true
		}
		if r.bit() == 0 {
			continue
		}
		if r.remaining() < o.bits {
			return NeighbourCell{}, false
		}
		o.read()
	}
	return cell, true
}
