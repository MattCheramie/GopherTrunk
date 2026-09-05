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

// fragTestChain stashes a start fragment at slot position 0 and returns the
// bus subscription plus the CMCE payload halves, so each abandon-hole test
// shares one setup: stash → the slot under test → MAC-END → did the corrupt
// chain publish?
func fragTestChain(t *testing.T) (*ControlChannel, *events.Subscription, []byte, int) {
	t.Helper()
	const (
		gssi     = 0x0F5670
		carrier  = 2716
		timeslot = 1
	)
	full := cmceSetupEmergencyParty(0x1234, 0x0123AB)
	split := 24
	bus := events.NewBus(32)
	t.Cleanup(bus.Close)
	sub := bus.Subscribe()
	t.Cleanup(sub.Close)
	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_913_000})
	ingestMACAt(cc, 0, macResourceStartFragment(gssi, carrier, timeslot, full[:split]))
	return cc, sub, full, split
}

// controlAACHDibits encodes an AACH whose header reads downlink common control
// and splits it into the two NCDB half spans.
func controlAACHDibits(t *testing.T, colour uint32) (aach1, aach2 []uint8) {
	t.Helper()
	dibits := TetraBitsToDibits(EncodeAACH(make([]byte, 14), colour))
	if len(dibits) != ndbAACH1Len+ndbAACH2Len {
		t.Fatalf("AACH encoded to %d dibits, want %d", len(dibits), ndbAACH1Len+ndbAACH2Len)
	}
	return dibits[:ndbAACH1Len], dibits[ndbAACH1Len:]
}

// garbageBlock is a BKN half that fails every SCH CRC.
func garbageBlock() []uint8 {
	g := make([]uint8, ndbBlockDibits)
	for i := range g {
		g[i] = uint8((i*7 + 3) % 4)
	}
	return g
}

// benignSCHHDHalf encodes a decodable SCH/HD half-slot block (a real SYSINFO
// broadcast block, harmless to ingest) at the given colour.
func benignSCHHDHalf(t *testing.T, colour uint32) []uint8 {
	t.Helper()
	type1 := hexToBits(t, "8a9c4c0e928eec8bd0c0041cffffd700")[:124]
	dibits := TetraBitsToDibits(EncodeSCHHD(type1, colour))
	if len(dibits) != ndbBlockDibits {
		t.Fatalf("SCH/HD encoded to %d dibits, want %d", len(dibits), ndbBlockDibits)
	}
	return dibits
}

// TestFragmentReassemblyAbandonedOnHalfSlotLoss: a control slot where one
// SCH/HD half decodes and the other fails CRC lost a signalling block — the
// old slot-level check (recovered == 0) counted it as recovered, kept the
// chain, and let the NEXT transmission's MAC-FRAG/MAC-END splice on (the 4-5
// Sep field mechanism behind the phantom D-NWRK-BROADCAST neighbour sites).
// Fails against the old code: the corrupt chain published its grant.
func TestFragmentReassemblyAbandonedOnHalfSlotLoss(t *testing.T) {
	const colour = 0
	cc, sub, full, split := fragTestChain(t)
	aach1, aach2 := controlAACHDibits(t, colour)
	cc.decodeDownlinkSlot(1020, benignSCHHDHalf(t, colour), aach1, aach2, garbageBlock(), nil, nil)
	ingestMACAt(cc, 2040, macEndPDU(full[split:]))
	if enrichedGrantSeen(sub, 0x0123AB) {
		t.Error("reassembly spliced across a half-slot signalling loss and published the corrupt-chain grant")
	}
}

// TestFragmentReassemblyAbandonedOnUnclassifiableSlot: a slot whose AACH does
// not decode is of unknown type — it may have been a control slot carrying the
// chain's continuation. DecodeAACH's maximum-likelihood search ALWAYS returns
// the nearest codeword, so a garbage AACH "decodes" — at a large Hamming
// distance — to random header bits: the old code took that as an authoritative
// classification, and whenever the coin-flip read "traffic" the chain survived
// the loss. Fails against the old code.
func TestFragmentReassemblyAbandonedOnUnclassifiableSlot(t *testing.T) {
	cc, sub, full, split := fragTestChain(t)
	g := garbageBlock()
	// Pick a garbage AACH whose nearest codeword is beyond the trust gate at
	// every rotation, so this slot is genuinely unclassifiable — and prove it,
	// so the test cannot silently exercise the trusted-control branch instead.
	// (The old code additionally had to read "traffic" from the garbage to
	// expose the hole; this seed does.)
	aach := make([]uint8, ndbAACH1Len+ndbAACH2Len)
	derot := func(d []uint8, rot uint8) []byte {
		return TetraDibitsToBits(rotateDibits(d, (4-rot)&3))
	}
	seedOK := false
	for seed := 1; seed <= 64 && !seedOK; seed++ {
		for i := range aach {
			aach[i] = uint8((i*seed + seed*seed + 1) % 4)
		}
		seedOK = true
		for r := uint8(0); r < 4; r++ {
			if _, errs := DecodeAACH(derot(aach, r), 0); errs >= 0 && errs <= aachClassifyMaxErrs {
				seedOK = false
				break
			}
		}
		if !seedOK {
			continue
		}
		// Old-code exposure: its first-rotation decode must coin-flip to
		// "traffic" for the hole to manifest (it abandoned on "control").
		if rec, errs := DecodeAACH(derot(aach, 0), 0); errs >= 0 {
			if parsed, ok := ParseAccessAssign(rec); ok && parsed.IsControlChannel() {
				seedOK = false
			}
		}
	}
	if !seedOK {
		t.Fatal("no garbage AACH beyond the classification trust gate found")
	}
	cc.decodeDownlinkSlot(1020, g, aach[:ndbAACH1Len], aach[ndbAACH1Len:], g, nil, nil)
	ingestMACAt(cc, 2040, macEndPDU(full[split:]))
	if enrichedGrantSeen(sub, 0x0123AB) {
		t.Error("reassembly spliced across an unclassifiable slot and published the corrupt-chain grant")
	}
}

// TestFragmentReassemblyRequiresOnGridContinuation: a MAC-END can only arrive
// in a real slot, and real slots sit on the 255-dibit grid anchored by the
// chain's own pieces. An off-grid position is a spurious correlator emit (the
// NCDB detector's tolerance-2 matches inside burst payloads, seen at +92/+163
// dibits on the 4 Sep field captures) — a continuation "arriving" there must
// not splice. Fails against the old adjacency check, which accepted any delta
// within the window. Also pins the window itself at two frames: the 4-5 Sep
// splices arrived 3+ frames after the stale chain's last piece, inside the old
// four-frame window.
func TestFragmentReassemblyRequiresOnGridContinuation(t *testing.T) {
	cc, sub, full, split := fragTestChain(t)
	ingestMACAt(cc, 1020+92, macEndPDU(full[split:])) // off-grid spurious emit
	if enrichedGrantSeen(sub, 0x0123AB) {
		t.Error("an off-grid MAC-END was spliced onto the chain")
	}

	cc2, sub2, full2, split2 := fragTestChain(t)
	ingestMACAt(cc2, 3*1020, macEndPDU(full2[split2:])) // three frames late
	if enrichedGrantSeen(sub2, 0x0123AB) {
		t.Error("a MAC-END three frames after the chain's last piece was spliced on")
	}

	cc3, sub3, full3, split3 := fragTestChain(t)
	ingestMACAt(cc3, 2*1020, macEndPDU(full3[split3:])) // frame-18 skip: legitimate
	if !enrichedGrantSeen(sub3, 0x0123AB) {
		t.Error("a MAC-END two frames later (frame-18 skip) no longer reassembles")
	}
}

// TestFragmentReassemblySurvivesDecodedSlots is the no-harm control for the
// abandon holes: fully decoded control slots between the start fragment and
// its MAC-END (both halves clean, at the legitimate 255/510-dibit emit
// spacing) must NOT abandon the chain — the reassembly still completes.
func TestFragmentReassemblySurvivesDecodedSlots(t *testing.T) {
	const colour = 0
	cc, sub, full, split := fragTestChain(t)
	aach1, aach2 := controlAACHDibits(t, colour)
	half := benignSCHHDHalf(t, colour)
	cc.decodeDownlinkSlot(255, half, aach1, aach2, half, nil, nil)
	cc.decodeDownlinkSlot(765, half, aach1, aach2, half, nil, nil) // 510: the frame-18 SB skip
	ingestMACAt(cc, 1020, macEndPDU(full[split:]))
	if !enrichedGrantSeen(sub, 0x0123AB) {
		t.Error("clean decoded slots mid-chain abandoned the reassembly; MAC-END no longer completes it")
	}
}
