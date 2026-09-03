package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Continuity regressions for the MAC fragment reassembly. Fragments of one
// TM-SDU continue in the following signalling opportunities of the same channel
// (§23.4.2); the reassembly used to splice a MAC-FRAG/MAC-END onto whatever
// start fragment was in flight, however long ago it was stashed and however
// many undecoded slots sat in between. Around a lost block that manufactures a
// plausible corrupt L3 PDU — and because broadcast content and fragmentation
// split points repeat, the SAME corrupt reassembly repeats bit-identically,
// defeating the confirm-twice dedup on D-NWRK-BROADCAST neighbour cells
// (osmo-tetra-sq5bpf ages fragment slots out for the same reason).

// ingestMACAt stamps the slot stream position the NCDB path normally stamps
// (decodeDownlinkSlot) and ingests the block at it.
func ingestMACAt(cc *ControlChannel, pos int, block []byte) {
	cc.mu.Lock()
	cc.curSlotPos = pos
	cc.mu.Unlock()
	cc.ingestMAC(block)
}

// enrichedGrantSeen reports whether the bus carried a grant enriched from a
// reassembled D-SETUP (SourceID + Emergency only a completed reassembly has).
func enrichedGrantSeen(sub *events.Subscription, party uint32) bool {
	for _, g := range collectGrants(sub) {
		if g.SourceID == party && g.Emergency {
			return true
		}
	}
	return false
}

// TestFragmentReassemblyRefusesStaleContinuation: a MAC-END arriving far
// beyond fragMaxGapDibits after the start fragment belongs to a LATER
// fragmented PDU whose own start was lost — it must not splice onto the stale
// chain. A stream-adjacent MAC-END (one TDMA frame later) must still
// reassemble. Fails against the old takeFragment, which spliced at any
// distance.
func TestFragmentReassemblyRefusesStaleContinuation(t *testing.T) {
	const (
		gssi     = 0x0F5670
		party    = 0x0123AB
		carrier  = 2716
		timeslot = 1
	)
	full := cmceSetupEmergencyParty(0x1234, party)
	split := 24

	run := func(endPos int) bool {
		bus := events.NewBus(32)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()
		cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
		ingestMACAt(cc, 1020, macResourceStartFragment(gssi, carrier, timeslot, full[:split]))
		ingestMACAt(cc, endPos, macEndPDU(full[split:]))
		return enrichedGrantSeen(sub, party)
	}

	if run(1020 + fragMaxGapDibits + 1) {
		t.Error("a MAC-END far past fragMaxGapDibits was spliced onto the stale start fragment")
	}
	if !run(1020 + 1020) { // the next TDMA frame's MCCH slot — the normal case
		t.Error("a stream-adjacent MAC-END no longer reassembles (continuity window too tight)")
	}
	// A backwards position is a stream discontinuity (resync baseline jump).
	if run(0) {
		t.Error("a MAC-END at an earlier stream position was spliced onto the chain")
	}
}

// TestFragmentReassemblyAbandonedOnUndecodedControlSlot drives the real NCDB
// slot path: a start fragment is in flight when a control slot (valid AACH,
// common control) yields no CRC-clean block — a lost signalling block, quite
// possibly the chain's continuation. The chain must be abandoned so the next
// MAC-END cannot splice around the gap. Fails against the old code, which
// completed the splice and published the corrupt-chain grant.
func TestFragmentReassemblyAbandonedOnUndecodedControlSlot(t *testing.T) {
	const (
		gssi     = 0x0F5670
		party    = 0x0123AB
		carrier  = 2716
		timeslot = 1
		colour   = 0
	)
	full := cmceSetupEmergencyParty(0x1234, party)
	split := 24

	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})

	ingestMACAt(cc, 0, macResourceStartFragment(gssi, carrier, timeslot, full[:split]))

	// An AACH that decodes (header 0 ⇒ downlink common control) over BKN halves
	// that do not: the SCHPDUsFail shape — a lost control block mid-reassembly.
	aachType5 := EncodeAACH(make([]byte, 14), colour)
	aachDibits := TetraBitsToDibits(aachType5)
	if len(aachDibits) != 15 {
		t.Fatalf("AACH encoded to %d dibits, want 15", len(aachDibits))
	}
	garbage := make([]uint8, ndbBlockDibits)
	for i := range garbage {
		garbage[i] = uint8((i*7 + 3) % 4)
	}
	cc.decodeDownlinkSlot(1020, garbage, aachDibits[:ndbAACH1Len], aachDibits[ndbAACH1Len:], garbage, nil, nil)

	ingestMACAt(cc, 2040, macEndPDU(full[split:]))

	if enrichedGrantSeen(sub, party) {
		t.Error("reassembly spliced across an undecoded control slot and published the corrupt-chain grant")
	}
}
