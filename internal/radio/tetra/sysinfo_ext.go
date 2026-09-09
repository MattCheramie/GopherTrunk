package tetra

import (
	"fmt"
	"strings"
)

// Extended SYSINFO decoding (EN 300 392-2 §21.4.4.1, Table 21.65 / "table 333").
//
// ParseSysInfo (mac.go) reads only the five frequency parameters; the rest of
// the 124-bit SYSINFO block carries the cell's access parameters, the common
// secondary-control-channel (SCCH) allocation, and — as its TM-SDU — the
// D-MLE-SYSINFO element (EN 300 392-2 §18): Location Area, subscriber class and
// BS service details. The bit layout here is pinned against BOTH independent
// proven decoders (the #764/#771 discipline): osmo-tetra-sq5bpf
// macpdu_decode_sysinfo/decode_d_mle_sysinfo (tetra_mac_pdu.c — D-MLE-SYSINFO at
// the fixed offset 124-42) and tetra-kit Mac::pduProcessSysinfo (decoder/mac/
// mac.cc — same walk, TM-SDU = 42 bits at position 82):
//
//	MAC PDU type (2) + broadcast type (2)
//	main carrier              12   \
//	frequency band             4    |
//	offset                     2    | ParseSysInfo's five (mac.go)
//	duplex spacing             3    |
//	reverse operation          1   /
//	number of common SCCH      2
//	MS_TXPWR_MAX_CELL          3
//	RXLEV_ACCESS_MIN           4
//	ACCESS_PARAMETER           4
//	RADIO_DOWNLINK_TIMEOUT     4
//	hyperframe / CCK flag      1
//	hyperframe number OR CCK id 16
//	optional field flag        2
//	optional field value      20   (always present; which of the four the flag says)
//	TM-SDU: D-MLE-SYSINFO     42 = LA (14) + subscriber class (16) + BS service details (12)

// sysInfoExtBits is the full SYSINFO block length: the extended fields are
// decodable only when the whole 124-bit BNCH half-slot block is present.
const sysInfoExtBits = 124

// SysInfoExt is the extended (non-frequency) content of a SYSINFO broadcast.
// Raw field values are stored as broadcast; the handful of spec-derived unit
// conversions (dBm mappings) live in the render helpers, clearly labelled.
type SysInfoExt struct {
	// SCCHInUse is the 2-bit "number of common secondary control channels in
	// use": 0 = MCCH only, n = n common SCCH allocated on timeslots 2..(n+1)
	// of the main carrier (§21.4.4.1).
	SCCHInUse uint8
	// MSTxPwrMaxCell is the raw 3-bit MS_TXPWR_MAX_CELL field.
	MSTxPwrMaxCell uint8
	// RxLevAccessMin is the raw 4-bit RXLEV_ACCESS_MIN field.
	RxLevAccessMin uint8
	// AccessParameter is the raw 4-bit ACCESS_PARAMETER field.
	AccessParameter uint8
	// RadioDLTimeout is the raw 4-bit RADIO_DOWNLINK_TIMEOUT field.
	RadioDLTimeout uint8
	// CCKValid says the 16-bit CounterOrCCK field carries the common cipher key
	// identifier / static cipher key version number; false ⇒ it carries the
	// cyclic hyperframe count instead (osmo-tetra's cck_valid_no_hf flag —
	// flag set selects the CCK id).
	CCKValid     bool
	CounterOrCCK uint16
	// OptionalFieldFlag selects which 20-bit optional field follows (§21.4.4.1):
	// 0 = even multiframe definition for TS mode, 1 = odd multiframe definition,
	// 2 = default definition for access code A, 3 = extended services broadcast.
	// OptionalField is its raw 20-bit value.
	OptionalFieldFlag uint8
	OptionalField     uint32

	// D-MLE-SYSINFO (the SYSINFO TM-SDU, EN 300 392-2 §18).
	LocationArea     uint16 // 14-bit LA
	SubscriberClass  uint16 // 16-bit allowed subscriber classes bitmap
	BSServiceDetails uint16 // 12-bit BS service details bitmap (§18.5.3)
}

// ParseSysInfoExtended decodes the extended SYSINFO content from a full
// 124-bit MAC broadcast SYSINFO block (type-1 bits, one-per-byte MSB first).
// Returns (zero, false) when the PDU is not a SYSINFO broadcast or the block is
// too short to carry the extended fields — ParseSysInfo still decodes the
// frequency parameters from such a block.
func ParseSysInfoExtended(bits []byte) (SysInfoExt, bool) {
	if len(bits) < sysInfoExtBits {
		return SysInfoExt{}, false
	}
	r := &bitReader{bits: bits}
	if MACPDUType(r.u(2)) != MACPDUBroadcast || r.u(2) != uint32(TETRAMACBcastSysInfo) {
		return SysInfoExt{}, false
	}
	r.u(12 + 4 + 2 + 3 + 1) // the five frequency parameters (ParseSysInfo's)
	ext := SysInfoExt{
		SCCHInUse:       uint8(r.u(2)),
		MSTxPwrMaxCell:  uint8(r.u(3)),
		RxLevAccessMin:  uint8(r.u(4)),
		AccessParameter: uint8(r.u(4)),
		RadioDLTimeout:  uint8(r.u(4)),
	}
	ext.CCKValid = r.bit() == 1
	ext.CounterOrCCK = uint16(r.u(16))
	ext.OptionalFieldFlag = uint8(r.u(2))
	ext.OptionalField = r.u(20)
	ext.LocationArea = uint16(r.u(14))
	ext.SubscriberClass = uint16(r.u(16))
	ext.BSServiceDetails = uint16(r.u(12))
	return ext, true
}

// sameCellParams reports whether two extended SYSINFO decodes carry the same
// cell parameters, ignoring the hyperframe counter (it increments every
// multiframe cycle) AND the 2-bit-flag-selected optional field (the BS rotates
// which of the four 20-bit definitions each broadcast carries) — including
// either would defeat the change-gated logging.
func (e SysInfoExt) sameCellParams(o SysInfoExt) bool {
	if !e.CCKValid && !o.CCKValid {
		e.CounterOrCCK, o.CounterOrCCK = 0, 0
	}
	e.OptionalFieldFlag, o.OptionalFieldFlag = 0, 0
	e.OptionalField, o.OptionalField = 0, 0
	return e == o
}

// SCCHTimeslots renders the common-SCCH allocation as the timeslots it
// occupies on the main carrier ("" when none — MCCH-only operation).
func (e SysInfoExt) SCCHTimeslots() string {
	switch e.SCCHInUse {
	case 1:
		return "TS2"
	case 2:
		return "TS2-3"
	case 3:
		return "TS2-4"
	}
	return ""
}

// MSTxPwrMaxCellDBm converts MS_TXPWR_MAX_CELL to dBm: 15 + (n-1)·5, the
// coding tetra-kit implements for the same element in the neighbour-cell
// context (§18.5.13 — mle_elements.cc parseNeighbourCellInformation). Returns
// (0, false) for the reserved value 0.
func (e SysInfoExt) MSTxPwrMaxCellDBm() (int, bool) {
	if e.MSTxPwrMaxCell == 0 {
		return 0, false
	}
	return 15 + (int(e.MSTxPwrMaxCell)-1)*5, true
}

// RxLevAccessMinDBm converts RXLEV_ACCESS_MIN to dBm: n·5 − 125, the coding
// tetra-kit implements for the same element in the neighbour-cell context
// (§18.5.14 — mle_elements.cc parseNeighbourCellInformation).
func (e SysInfoExt) RxLevAccessMinDBm() int {
	return int(e.RxLevAccessMin)*5 - 125
}

// bsServiceDetailNames maps the 12 BS-service-details bits, MSB first, to the
// service each advertises — the naming tetra-kit itemises in
// parseBsServiceDetails (decoder/mle/mle_elements.cc, §18.5.3). Bit 3 (MSB
// numbering; "reserved") is omitted from rendering.
var bsServiceDetailNames = [12]string{
	"reg",       // registration mandatory
	"dereg",     // de-registration requested
	"pri-cell",  // priority cell
	"min-mode",  // minimum mode service supported
	"migration", // migration supported
	"sys-wide",  // system wide services supported
	"voice",     // TETRA voice service supported
	"data",      // circuit mode data service supported
	"",          // reserved
	"sndcp",     // SNDCP service available
	"air-enc",   // air interface encryption service available
	"adv-link",  // advanced link supported
}

// BSServiceDetailsString renders a 12-bit BS service details bitmap as the
// short names of the set bits (MSB first), e.g. "reg,voice,data,air-enc".
// Returns "" for an all-zero bitmap.
func BSServiceDetailsString(v uint16) string {
	parts := make([]string, 0, 12)
	for i, name := range bsServiceDetailNames {
		if name == "" {
			continue
		}
		if v&(1<<(11-i)) != 0 {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ",")
}

// sysInfoOptionalFieldName names which 20-bit optional field the 2-bit flag
// selected (§21.4.4.1).
func sysInfoOptionalFieldName(flag uint8) string {
	switch flag & 0x3 {
	case 0:
		return "even_multiframe"
	case 1:
		return "odd_multiframe"
	case 2:
		return "access_code_a"
	default:
		return "extended_services"
	}
}

// String renders the extended parameters compactly for the change-gated log.
func (e SysInfoExt) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "scch=%d", e.SCCHInUse)
	if ts := e.SCCHTimeslots(); ts != "" {
		fmt.Fprintf(&b, "(%s)", ts)
	}
	if dbm, ok := e.MSTxPwrMaxCellDBm(); ok {
		fmt.Fprintf(&b, " ms_txpwr_max=%ddBm", dbm)
	}
	fmt.Fprintf(&b, " rxlev_access_min=%ddBm access_parameter=%d radio_dl_timeout=%d",
		e.RxLevAccessMinDBm(), e.AccessParameter, e.RadioDLTimeout)
	if e.CCKValid {
		fmt.Fprintf(&b, " cck_id=0x%04x", e.CounterOrCCK)
	}
	fmt.Fprintf(&b, " la=%d subscriber_class=0x%04x bs_service=0x%03x", e.LocationArea, e.SubscriberClass, e.BSServiceDetails)
	if s := BSServiceDetailsString(e.BSServiceDetails); s != "" {
		fmt.Fprintf(&b, "[%s]", s)
	}
	return b.String()
}
