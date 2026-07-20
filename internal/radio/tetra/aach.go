package tetra

// Access-assignment header values (ETSI EN 300 392-2 §21.4.7, mirrored by
// osmo-tetra's macpdu_decode_access_assign). The 2-bit header selects how
// the two 6-bit access-assignment fields are interpreted. Header 0 leaves
// the downlink as common control; the other three all carry the downlink
// usage marker in field 1.
const (
	accHdrDLCommonControl uint8 = 0 // DL common control / UL common
	// headers 1..3 (DLF1_ULCA / DLF1_ULAO / DLF1_ULF1) put the downlink
	// usage marker in field 1; field 2's meaning is uplink-only and not
	// needed to classify the downlink slot.
)

// Downlink usage marker values (ETSI EN 300 392-2, osmo-tetra
// dl_usage_names). The marker is what identifies, slot by slot, whether a
// TETRA downlink slot is currently carrying control signalling or traffic.
// It is the basis for following a Single Carrier Base Station running
// dynamic MCCH sharing, where any of the four TDMA slots can be acting as
// the control channel at a given moment rather than a fixed slot 1
// (issue #925).
const (
	DLUsageUnallocated     uint8 = 0
	DLUsageAssignedControl uint8 = 1
	DLUsageCommonControl   uint8 = 2
	DLUsageReserved        uint8 = 3
	// Values >= DLUsageTraffic are traffic; the marker value itself is the
	// usage marker identifying the call occupying the slot.
	DLUsageTraffic uint8 = 4
)

// AccessAssign is the decoded ACCESS-ASSIGN PDU carried by the AACH in the
// centre of every TETRA downlink slot (ETSI EN 300 392-2 §21.4.7). Its
// downlink usage marker is the per-slot control-vs-traffic indicator a
// dynamic-MCCH-sharing SCBS relies on (issue #925): the AACH is present in
// every slot regardless of what the slot carries, so decoding it frame by
// frame is how a receiver maps which slot is currently the control channel.
//
// Field2 carries uplink access information that is not needed to classify
// the downlink; it is retained raw for completeness.
type AccessAssign struct {
	Header uint8 // 2-bit access-assignment header
	Field1 uint8 // 6-bit
	Field2 uint8 // 6-bit
}

// DownlinkUsage returns the slot's downlink usage marker. Header 0 leaves
// the downlink as common control by definition; every other header carries
// the marker explicitly in field 1.
func (a AccessAssign) DownlinkUsage() uint8 {
	if a.Header == accHdrDLCommonControl {
		return DLUsageCommonControl
	}
	return a.Field1
}

// IsControlChannel reports whether the slot's downlink is currently
// carrying control signalling (assigned or common control) rather than
// traffic or nothing — the per-slot MCCH indicator for dynamic sharing.
func (a AccessAssign) IsControlChannel() bool {
	u := a.DownlinkUsage()
	return u == DLUsageAssignedControl || u == DLUsageCommonControl
}

// IsTraffic reports whether the slot's downlink is carrying a traffic
// channel (a call), identified by the usage marker value.
func (a AccessAssign) IsTraffic() bool {
	return a.DownlinkUsage() >= DLUsageTraffic
}

// ParseAccessAssign decodes the 14 recovered type-1 bits of an AACH (the
// DecodeAACH output — one bit per byte, MSB first) into the ACCESS-ASSIGN
// PDU. Returns (zero, false) when fewer than 14 bits are supplied.
//
// Note: the header interpretation here follows the normal-frame
// (frames 1..17) convention; frame 18 carries the broadcast block and can
// reinterpret the fields — tracking the frame number to handle that is a
// follow-up, and does not affect the normal-frame downlink usage marker.
func ParseAccessAssign(type1 []byte) (AccessAssign, bool) {
	if len(type1) < 14 {
		return AccessAssign{}, false
	}
	bit := func(i int) uint8 { return type1[i] & 1 }
	field6 := func(start int) uint8 {
		var v uint8
		for i := 0; i < 6; i++ {
			v = v<<1 | bit(start+i)
		}
		return v
	}
	return AccessAssign{
		Header: bit(0)<<1 | bit(1),
		Field1: field6(2),
		Field2: field6(8),
	}, true
}
