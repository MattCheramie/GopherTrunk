package tetra

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

// mlePDUTypeDNwrkBroadcast is the 3-bit downlink MLE PDU type for
// D-NWRK-BROADCAST (osmo-tetra TMLE_PDUT_D_NWRK_BROADCAST = tetra-kit 0b010).
const mlePDUTypeDNwrkBroadcast uint32 = 0x2

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
// from r. Unsurfaced optional fields are length-skipped so the NEXT cell in the
// list stays aligned.
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
		{3, func() { r.u(3) }},   // max MS tx power
		{4, func() { r.u(4) }},   // min RX access level
		{16, func() { r.u(16) }}, // subscriber class
		{12, func() { r.u(12) }}, // BS service details
		{5, func() { r.u(5) }},   // timeshare cell / security parameters
		{6, func() { r.u(6) }},   // TDMA frame offset
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
