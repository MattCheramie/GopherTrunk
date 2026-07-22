package tetra

import (
	"encoding/hex"
	"testing"
)

// hexToBits expands packed MSB-first bytes into a one-bit-per-byte slice — the
// type-1 bit shape DecodeSCHF / DecodeSCHHD emit and ParseMACResource consumes.
func hexToBits(t *testing.T, s string) []byte {
	t.Helper()
	packed, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	bits := make([]byte, len(packed)*8)
	for i := range bits {
		if packed[i>>3]&(1<<uint(7-(i&7))) != 0 {
			bits[i] = 1
		}
	}
	return bits
}

// TestParseMACResourceRealGrant parses a real MAC-RESOURCE PDU recovered from
// the 467.913 MHz SCBS capture (issue: TETRA voice grants never decoded because
// the MAC header was fed straight to the L3 parser). This SCH/F block carries a
// channel-allocation element granting a call onto the cell's own carrier 2716
// (= 467.913 MHz), timeslot 1, for individual SSI 1005356 — one of the two
// calls in the capture. The bytes are the exact type-1 payload the receiver
// recovered (burst at dibit L=270183).
func TestParseMACResourceRealGrant(t *testing.T) {
	const pdu = "20760f572c3c83d538c90a1fe305009e0f92813c83d538c91e1fe301304a1eae5880"
	bits := hexToBits(t, pdu)

	m, ok := ParseMACResource(bits, false)
	if !ok {
		t.Fatal("ParseMACResource returned !ok on a real MAC-RESOURCE PDU")
	}
	if m.NullPDU {
		t.Fatal("real grant PDU wrongly parsed as a NULL PDU")
	}
	if m.Address.SSI != 1005356 {
		t.Errorf("Address.SSI = %d, want 1005356", m.Address.SSI)
	}
	if m.ChanAlloc == nil {
		t.Fatal("ChanAlloc = nil, want a channel-allocation grant")
	}
	if m.ChanAlloc.CarrierNumber != 2716 {
		t.Errorf("grant carrier = %d, want 2716 (467.913 MHz, the cell's own carrier)", m.ChanAlloc.CarrierNumber)
	}
	if m.ChanAlloc.Timeslot != 1 {
		t.Errorf("grant timeslot = %d, want 1", m.ChanAlloc.Timeslot)
	}
}

// TestParseMACResourceRejectsNonResource ensures a MAC broadcast PDU (SYSINFO,
// leading MAC PDU type 10) is not mistaken for a MAC-RESOURCE.
func TestParseMACResourceRejectsNonResource(t *testing.T) {
	// SYSINFO broadcast from the same capture; main carrier 2716 sits in the
	// 12 bits after the 2-bit MAC type + 2-bit broadcast type.
	const sysinfo = "8a9c4c0e928eec8bd0c0041cffffd700"
	bits := hexToBits(t, sysinfo)
	if MACPDUType(bits[0]<<1|bits[1]) != MACPDUBroadcast {
		t.Fatalf("leading MAC PDU type = %d, want broadcast (2)", bits[0]<<1|bits[1])
	}
	if _, ok := ParseMACResource(bits, false); ok {
		t.Error("ParseMACResource accepted a MAC broadcast PDU")
	}
	// The cell's own main carrier decodes to 2716.
	r := &bitReader{bits: bits}
	r.u(2) // MAC PDU type
	if r.u(2) != 0 {
		t.Fatal("broadcast subtype != SYSINFO")
	}
	if mc := uint16(r.u(12)); mc != 2716 {
		t.Errorf("SYSINFO main carrier = %d, want 2716", mc)
	}
}

// TestSysInfoMainCarrier decodes the cell's own carrier number from a real
// SYSINFO broadcast recovered from the 467.913 MHz capture — carrier 2716,
// which resolves to 467.913 MHz on the band plan.
func TestSysInfoMainCarrier(t *testing.T) {
	bits := hexToBits(t, "8a9c4c0e928eec8bd0c0041cffffd700")
	mc, ok := SysInfoMainCarrier(bits)
	if !ok {
		t.Fatal("SysInfoMainCarrier returned !ok on a real SYSINFO broadcast")
	}
	if mc != 2716 {
		t.Errorf("main carrier = %d, want 2716", mc)
	}
	// A MAC-RESOURCE PDU is not a SYSINFO broadcast.
	if _, ok := SysInfoMainCarrier(hexToBits(t, "20760f572c3c83d538c90a1fe305009e")); ok {
		t.Error("SysInfoMainCarrier accepted a non-broadcast PDU")
	}
}
