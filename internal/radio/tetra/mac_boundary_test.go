package tetra

import (
	"bytes"
	"testing"
)

// These are the regressions for the MAC PDU boundary bugs behind the operator's
// "totally bogus entries in the neighbours list, like a 1.5 GHz one" report:
// tmSDU/macFragmentPayload used to hand everything to the block end to L3, so
// fill bits and any further multiplexed content after the PDU leaked into the
// TL-SDU. Most L3 parsers read a fixed prefix and never noticed; a parser with
// trailing presence bits — ParseDNwrkBroadcast's last neighbour cell — read the
// leaked bits as phantom optional fields. Because the leaked content repeats
// bit-identically across rebroadcasts, the corruption also defeated the
// confirm-twice dedup in learnNeighbourCells.

// macResourceSDU builds a MAC-RESOURCE block with an SSI address, no channel
// allocation, and the given TM-SDU — the shape a D-NWRK-BROADCAST arrives in.
// Length indication stamped like macResourceGrant.
func macResourceSDU(ssi uint32, tmsdu []byte) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUResource), 2) // MAC PDU type
	w.u(0, 1)                      // fill bit
	w.u(0, 1)                      // grant position
	w.u(0, 2)                      // encryption mode (0 = clear)
	w.u(0, 1)                      // random access
	w.u(0, 6)                      // length indication — stamped below
	w.u(uint64(addrSSI), 3)        // address type = SSI
	w.u(uint64(ssi), 24)           // SSI
	w.u(0, 1)                      // power control absent
	w.u(0, 1)                      // slot granting absent
	w.u(0, 1)                      // channel allocation absent
	return stampMACResourceLength(append(w.bits, tmsdu...))
}

// bitsPattern returns n bits of the given value repeated.
func bitsPattern(bit byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = bit
	}
	return out
}

// TestTMSDUHonoursLengthIndicationAndFill: the TM-SDU must end where the
// header's length indication says the PDU ends, minus the fill run the
// fill-bit indication flags — not at the block end. Fails against the old
// tmSDU, which returned tmsdu+fill+garbage.
func TestTMSDUHonoursLengthIndicationAndFill(t *testing.T) {
	tmsdu := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 1, 0, 1, 0}
	block := macResourceSDU(0x0ABCDE, tmsdu)

	// Fill-pad the PDU to the stamped octet boundary ('1' then '0's,
	// §23.4.3.2), setting the fill-bit indication, exactly as a transmitter
	// does. Then extend the block with garbage simulating further multiplexed
	// content after the PDU.
	pad := ((len(block) + 7) / 8 * 8) - len(block)
	if pad == 0 {
		tmsdu = append(tmsdu, 0, 1, 0)
		block = macResourceSDU(0x0ABCDE, tmsdu)
		pad = ((len(block) + 7) / 8 * 8) - len(block)
	}
	if pad == 0 {
		t.Fatal("could not arrange a fill-padded PDU")
	}
	block[2] = 1 // fill-bit indication
	block = append(block, 1)
	block = append(block, bitsPattern(0, pad-1)...)
	full := append(append([]byte{}, block...), bitsPattern(1, 48)...)

	m, ok := ParseMACResource(full, false)
	if !ok {
		t.Fatal("ParseMACResource rejected the block")
	}
	got := m.tmSDU(full)
	if !bytes.Equal(got, tmsdu) {
		t.Errorf("tmSDU returned %d bits, want the exact %d-bit TM-SDU\n got: %v\nwant: %v",
			len(got), len(tmsdu), got, tmsdu)
	}
}

// TestMACEndPayloadHonoursLengthAndFill: the MAC-END fragment payload must
// stop at the header's length indication and drop the flagged fill run.
func TestMACEndPayloadHonoursLengthAndFill(t *testing.T) {
	payload := []byte{1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 0, 0, 1, 0}
	pdu := macEndPDU(payload) // 12-bit header + payload; li written as 0
	pad := ((len(pdu) + 7) / 8 * 8) - len(pdu)
	if pad == 0 {
		payload = append(payload, 0, 1, 1)
		pdu = macEndPDU(payload)
		pad = ((len(pdu) + 7) / 8 * 8) - len(pdu)
	}
	if pad == 0 {
		t.Fatal("could not arrange a fill-padded MAC-END")
	}
	pdu[3] = 1 // fill-bit indication
	octets := (len(pdu) + pad) / 8
	for i := 0; i < 6; i++ { // length indication at bit offsets 4..9
		pdu[4+i] = byte((octets >> (5 - i)) & 1)
	}
	pdu = append(pdu, 1)
	pdu = append(pdu, bitsPattern(0, pad-1)...)
	full := append(append([]byte{}, pdu...), bitsPattern(1, 40)...)

	got := macFragmentPayload(full)
	if !bytes.Equal(got, payload) {
		t.Errorf("macFragmentPayload returned %d bits, want the exact %d-bit fragment\n got: %v\nwant: %v",
			len(got), len(payload), got, payload)
	}
}

// TestMACFragPayloadStripsFill: a MAC-FRAG with the fill-bit indication set
// must drop its trailing fill run.
func TestMACFragPayloadStripsFill(t *testing.T) {
	payload := []byte{0, 1, 1, 0, 1, 0, 0, 1, 1, 1}
	pdu := macFragPDU(payload)
	pdu[3] = 1 // fill-bit indication
	pdu = append(pdu, 1, 0, 0, 0)

	got := macFragmentPayload(pdu)
	if !bytes.Equal(got, payload) {
		t.Errorf("macFragmentPayload returned %v, want %v", got, payload)
	}
}

// dnwrkTLSDUTruncatedCell builds a D-NWRK-BROADCAST TL-SDU carrying ONE
// neighbour cell whose element ends mid-optional-list (LA is the last present
// field; the six remaining P-bits are omitted because the PDU ends there —
// the legal type-2 truncation real encoders use). Whatever follows the PDU in
// the MAC block must NOT be read as those P-bits.
func dnwrkTLSDUTruncatedCell() []byte {
	w := &cmceBitWriter{}
	w.u(0x5, 3)     // MLE PD
	w.u(0x2, 3)     // D-NWRK-BROADCAST
	w.u(0xAB54, 16) // cell re-select parameters
	w.u(0, 2)       // serving cell service level
	w.u(1, 1)       // O-bit
	w.u(0, 1)       // P: network time absent
	w.u(1, 1)       // P: neighbour cells present
	w.u(1, 3)       // count = 1
	w.u(9, 5)       // cell id
	w.u(0, 2)       // reselect types
	w.u(0, 1)       // not synchronized
	w.u(0, 2)       // cell service level
	w.u(2754, 12)   // main carrier
	w.u(1, 1)       // O-bit: optionals follow
	w.u(1, 1)       // P: extension present
	w.u(4, 4)       // band 4
	w.u(3, 2)       // offset
	w.u(6, 3)       // duplex spacing
	w.u(0, 1)       // reverse operation
	w.u(0, 1)       // P: MCC absent
	w.u(0, 1)       // P: MNC absent
	w.u(1, 1)       // P: LA present
	w.u(1031, 14)   // LA
	return w.bits   // PDU ends here — remaining six P-bits omitted
}

// TestNeighbourCellNoPhantomOptionalsFromBlockTail is the end-to-end
// regression: a MAC block whose D-NWRK-BROADCAST PDU is followed by non-fill
// content (further multiplexed bits) must decode the neighbour cell EXACTLY,
// with no phantom optional statuses read from the tail. Ingested twice because
// deterministic tail garbage repeats bit-identically — the confirm-twice gate
// does not catch it, which is the point of fixing the boundary instead.
func TestNeighbourCellNoPhantomOptionalsFromBlockTail(t *testing.T) {
	tmsdu := append(bl(llcBLUDATA, 0), dnwrkTLSDUTruncatedCell()...)
	block := macResourceSDU(0x0ABCDE, tmsdu)
	// The fixture is arranged to end exactly on the stamped octet boundary
	// (43-bit MAC header + 85-bit TM-SDU = 128 bits), so the garbage tail sits
	// DIRECTLY after the PDU with no intervening fill — the shape that made the
	// old block-end tmSDU read tail bits as the cell's omitted P-bits.
	if len(block)%8 != 0 {
		t.Fatalf("fixture PDU is %d bits — must be octet-aligned for the tail to abut the PDU", len(block))
	}
	// All-ones tail: under the old block-end tmSDU every omitted P-bit read 1
	// and the cell grew phantom max-tx-power/min-rx/subscriber-class/… fields.
	full := append(append([]byte{}, block...), bitsPattern(1, 60)...)

	cc := New(Options{SystemName: "Sys", FrequencyHz: 467_912_500})
	cc.ingestMAC(full)
	cc.ingestMAC(full)

	cc.mu.Lock()
	got, ok := cc.neighbours[9]
	cc.mu.Unlock()
	if !ok {
		t.Fatal("neighbour cell 9 did not surface after two identical sightings")
	}
	want := NeighbourCell{
		CellID: 9, MainCarrier: 2754,
		HasExtension: true, FreqBand: 4, Offset: 3, DuplexSpacing: 6,
		HasLA: true, LA: 1031,
	}
	if got != want {
		t.Errorf("neighbour cell decoded with phantom fields from the block tail:\n got: %+v\nwant: %+v", got, want)
	}
}
