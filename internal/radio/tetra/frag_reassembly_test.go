package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// macResourceStartFragment builds a MAC-RESOURCE whose length indication is the
// reserved start-fragment marker (0x3f, §21.4.3.1): the MAC header — including
// the channel-allocation element — is complete, but the TM-SDU is truncated and
// continues in a following MAC-FRAG/MAC-END. It is macResourceGrant with the
// length field swapped for the marker.
func macResourceStartFragment(ssi uint32, carrier uint16, timeslot uint8, partial []byte) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUResource), 2)      // MAC PDU type
	w.u(0, 1)                           // fill bit
	w.u(0, 1)                           // grant position
	w.u(0, 2)                           // encryption mode (0 = clear)
	w.u(0, 1)                           // random access
	w.u(uint64(macLenStartFragment), 6) // length indication = start-fragment marker
	w.u(uint64(addrSSI), 3)             // address type = SSI
	w.u(uint64(ssi), 24)                // SSI
	w.u(0, 1)                           // power control absent
	w.u(0, 1)                           // slot granting absent
	w.u(1, 1)                           // channel allocation present
	w.u(0, 2)                           // allocation type
	w.u(uint64(timeslot), 4)            // assigned timeslot
	w.u(1, 2)                           // up/downlink (nonzero → skip augmented branch)
	w.u(0, 1)                           // CLCH permission
	w.u(0, 1)                           // cell change
	w.u(uint64(carrier), 12)            // main carrier number
	w.u(0, 1)                           // extended carrier absent
	w.u(1, 2)                           // monitoring pattern (nonzero → no frame-18 field)
	return append(w.bits, partial...)
}

// macEndPDU wraps continuation bits in a MAC-END PDU (§21.4.3.4) with the
// on-air header layout (verified against osmo-tetra rx_macend and a real
// capture): MAC PDU type, subtype, fill-bit indication, 6-bit length
// indication, slot-granting flag, channel-allocation flag, then the fragment.
// The flags used to be missing here AND in macFragmentPayload — the encoder
// and decoder drifted together (the #764/#771 self-consistent trap), so the
// round-trip stayed green while every real reassembled TM-SDU carried stray
// bits at the fragment seam.
func macEndPDU(payload []byte) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUFragEnd), 2)
	w.u(uint64(macEnd), 1)
	w.u(0, 1) // fill-bit indication
	w.u(0, 6) // length indication
	w.u(0, 1) // slot-granting flag (absent)
	w.u(0, 1) // channel-allocation flag (absent)
	return append(w.bits, payload...)
}

// macFragPDU wraps continuation bits in a MAC-FRAG PDU (§21.4.3.3): MAC PDU
// type, subtype, fill-bit indication, then the fragment. The fill-bit
// indication is the one bit the old macFragmentPayload did not strip — every
// reassembled TM-SDU gained a stray bit at this seam on real air.
func macFragPDU(payload []byte) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUFragEnd), 2)
	w.u(uint64(macFrag), 1)
	w.u(0, 1) // fill-bit indication
	return append(w.bits, payload...)
}

// cmceSetupEmergencyParty builds a D-SETUP TM-SDU (LLC BL-UDATA + CMCE) marked
// emergency and carrying a calling-party SSI, so a published grant enriches to
// Emergency=true with SourceSSI=party. Long enough to be a realistic fragmented
// TM-SDU.
func cmceSetupEmergencyParty(callID uint16, party uint32) []byte {
	c := &cmceBitWriter{}
	c.header(CMCETypeDSetup)
	c.u(uint64(callID), 14)               // call identifier
	c.u(0, 4+1+1+8+2+1)                   // through transmission-request permission
	c.u(uint64(callPriorityEmergency), 4) // call priority = emergency
	c.u(1, 1)                             // O-bit: optional elements present
	c.opt(false, 0, 6)                    // notification indicator absent
	c.opt(false, 0, 24)                   // temporary address (GSSI) absent
	c.pbit(true)                          // calling party type identifier present
	c.u(1, 2)                             // CPTI = 1 (SSI present)
	c.u(uint64(party), 24)                // calling party SSI
	return append(bl(llcBLUDATA, 0), c.bits...)
}

func collectGrants(sub *events.Subscription) []trunking.Grant {
	var grants []trunking.Grant
	for {
		select {
		case ev := <-sub.C:
			if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
				grants = append(grants, g)
			}
		default:
			return grants
		}
	}
}

// TestIngestMACReassemblesFragmentedTMSDU is the MAC-FRAG/MAC-END regression. A
// TM-SDU too large for one MAC block arrives as a start-fragment MAC-RESOURCE
// (carrying the complete channel allocation) plus a MAC-END carrying the rest of
// the L3 CMCE PDU. The reassembled D-SETUP must enrich the grant with the same
// Emergency flag + calling-party SourceID that the identical unfragmented TM-SDU
// produces. Before reassembly the MAC-END was dropped, so the fragmented case
// only ever published the un-enriched start-fragment grant.
func TestIngestMACReassemblesFragmentedTMSDU(t *testing.T) {
	const (
		gssi     = 0x0F5670
		party    = 0x0123AB
		callID   = 0x1234
		carrier  = 2716
		timeslot = 1
	)
	full := cmceSetupEmergencyParty(callID, party)

	// Reference: the same TM-SDU delivered in one block.
	wantG, wantSrc, wantEmerg := ingestOneGrant(t, func(cc *ControlChannel) {
		cc.ingestMAC(macResourceGrant(gssi, carrier, timeslot, full))
	})
	if wantG != gssi || wantSrc != party || !wantEmerg {
		t.Fatalf("reference grant wrong: group=%#x src=%#x emergency=%v", wantG, wantSrc, wantEmerg)
	}

	// Split the TM-SDU early (inside the CMCE mandatory block, so the truncated
	// start fragment cannot itself parse a CMCE PDU) and deliver it fragmented.
	split := 24 // bits: past the LLC header, well short of the D-SETUP mandatory fields
	if split >= len(full) {
		t.Fatal("test TM-SDU shorter than the split point")
	}
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	cc.ingestMAC(macResourceStartFragment(gssi, carrier, timeslot, full[:split]))
	cc.ingestMAC(macEndPDU(full[split:]))

	grants := collectGrants(sub)
	if len(grants) == 0 {
		t.Fatal("no grant published for a fragmented D-SETUP")
	}
	var enriched *trunking.Grant
	for i := range grants {
		if grants[i].SourceID == party && grants[i].Emergency {
			enriched = &grants[i]
		}
	}
	if enriched == nil {
		t.Fatalf("fragmented grant not enriched from the reassembled MAC-END: got %+v (want group=%#x src=%#x emergency=true)",
			grants, uint32(gssi), uint32(party))
	}
	if enriched.GroupID != gssi {
		t.Errorf("reassembled grant GroupID = %#x, want %#x", enriched.GroupID, uint32(gssi))
	}
}

// TestIngestMACReassemblesThreeBlockTMSDU pins the MAC-FRAG seam: a TM-SDU
// spanning start-fragment + MAC-FRAG + MAC-END must reassemble to the same
// enriched grant as the unfragmented delivery. The MAC-FRAG header's fill-bit
// indication is what the old reader failed to strip — with it unstripped, the
// mid-PDU seam gains a stray bit and the CMCE fields after it decode as
// garbage, so this test fails against the old macFragmentPayload.
func TestIngestMACReassemblesThreeBlockTMSDU(t *testing.T) {
	const (
		gssi     = 0x0F5670
		party    = 0x0123AB
		callID   = 0x1234
		carrier  = 2716
		timeslot = 1
	)
	full := cmceSetupEmergencyParty(callID, party)
	splitA, splitB := 24, 40 // start | FRAG | END, both cuts inside the CMCE mandatory block
	if splitB >= len(full) {
		t.Fatal("test TM-SDU shorter than the split points")
	}

	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	cc.ingestMAC(macResourceStartFragment(gssi, carrier, timeslot, full[:splitA]))
	cc.ingestMAC(macFragPDU(full[splitA:splitB]))
	cc.ingestMAC(macEndPDU(full[splitB:]))

	grants := collectGrants(sub)
	var enriched *trunking.Grant
	for i := range grants {
		if grants[i].SourceID == party && grants[i].Emergency {
			enriched = &grants[i]
		}
	}
	if enriched == nil {
		t.Fatalf("three-block reassembly did not produce the enriched grant: got %+v (want src=%#x emergency=true)",
			grants, uint32(party))
	}
	if enriched.GroupID != gssi {
		t.Errorf("reassembled grant GroupID = %#x, want %#x", enriched.GroupID, uint32(gssi))
	}
}

// ingestOneGrant runs feed against a fresh ControlChannel and returns the fields
// of the last grant it published.
func ingestOneGrant(t *testing.T, feed func(*ControlChannel)) (group, src uint32, emergency bool) {
	t.Helper()
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	feed(cc)
	grants := collectGrants(sub)
	if len(grants) == 0 {
		t.Fatal("no grant published")
	}
	last := grants[len(grants)-1]
	return last.GroupID, last.SourceID, last.Emergency
}
