package tetra

import (
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// bitsOf turns a "0101…" string (spaces ignored) into the one-bit-per-byte
// slice ParseLLC hands to the L3 parsers.
func bitsOf(t *testing.T, s string) []byte {
	t.Helper()
	s = strings.ReplaceAll(s, " ", "")
	out := make([]byte, len(s))
	for i, ch := range s {
		switch ch {
		case '0':
		case '1':
			out[i] = 1
		default:
			t.Fatalf("bad bit char %q", ch)
		}
	}
	return out
}

// TestParseDNwrkBroadcastLiteral pins the D-NWRK-BROADCAST layout with a
// LITERAL bit vector hand-assembled from the field order both independent
// reference decoders implement (osmo-tetra-sq5bpf parse_d_nwrk_broadcast /
// parse_nci_ca, tetra-kit processDNwrkBroadcast) — not from this package's own
// parser, so a field-order or width slip here fails rather than round-trips
// (the #764/#771 self-consistent trap).
func TestParseDNwrkBroadcastLiteral(t *testing.T) {
	// Serving cell: re-select params 0xAB54, service level 2 (10), O=1,
	// network time absent (P=0), neighbours present (P=1), count 2 (010).
	pdu := "101" + "010" + // PD=MLE, type=D-NWRK-BROADCAST
		"1010101101010100" + // cell re-select parameters = 0xAB54
		"10" + // cell service level = 2
		"1" + // O-bit
		"0" + // P: TETRA network time absent
		"1" + "010" // P: neighbour cells, count=2
	// Cell 1: id=5 (00101), reselect types=3 (11), synced=1, service=1 (01),
	// carrier=2716 (101010011100), O=1, then P-gated: extension present
	// (band=4 0100, offset=3 11, duplex=6 110, reverse=0), MCC=250
	// (0011111010), MNC=13 (00000000001101), LA=4001 (00111110100001),
	// max tx power=5 (101), min rx level=10 (1010), subscriber class=0x8001,
	// BS service details=0x5A5 (010110100101), timeshare=0x15 (10101),
	// TDMA frame offset=51 (110011).
	pdu += "00101" + "11" + "1" + "01" + "101010011100" + "1" +
		"1" + "0100" + "11" + "110" + "0" +
		"1" + "0011111010" +
		"1" + "00000000001101" +
		"1" + "00111110100001" +
		"1" + "101" +
		"1" + "1010" +
		"1" + "1000000000000001" +
		"1" + "010110100101" +
		"1" + "10101" +
		"1" + "110011"
	// Cell 2: id=9 (01001), reselect types=0, synced=0, service=3 (11),
	// carrier=2720 (101010100000), O=0 — no optionals at all.
	pdu += "01001" + "00" + "0" + "11" + "101010100000" + "0"

	nb, ok := ParseDNwrkBroadcast(bitsOf(t, pdu))
	if !ok {
		t.Fatal("ParseDNwrkBroadcast returned ok=false on a valid PDU")
	}
	if nb.CellReselectParams != 0xAB54 || nb.CellServiceLevel != 2 {
		t.Errorf("serving fields = %#x/%d, want 0xAB54/2", nb.CellReselectParams, nb.CellServiceLevel)
	}
	if len(nb.Neighbours) != 2 {
		t.Fatalf("decoded %d neighbours, want 2", len(nb.Neighbours))
	}
	c1 := nb.Neighbours[0]
	if c1.CellID != 5 || c1.ReselectTypes != 3 || !c1.Synchronized || c1.CellServiceLevel != 1 || c1.MainCarrier != 2716 {
		t.Errorf("cell 1 mandatory fields = %+v, want id=5 types=3 synced sl=1 carrier=2716", c1)
	}
	if !c1.HasExtension || c1.FreqBand != 4 || c1.Offset != 3 || c1.DuplexSpacing != 6 || c1.ReverseOper {
		t.Errorf("cell 1 extension = %+v, want band=4 offset=3 duplex=6 reverse=false", c1)
	}
	if !c1.HasMCC || c1.MCC != 250 || !c1.HasMNC || c1.MNC != 13 || !c1.HasLA || c1.LA != 4001 {
		t.Errorf("cell 1 identity = %+v, want MCC=250 MNC=13 LA=4001", c1)
	}
	if !c1.HasMaxTxPower || c1.MaxTxPower != 5 || !c1.HasMinRxLevel || c1.MinRxLevel != 10 {
		t.Errorf("cell 1 power/access = %+v, want tx=5 rx=10", c1)
	}
	if !c1.HasSubscriberClass || c1.SubscriberClass != 0x8001 ||
		!c1.HasServiceDetails || c1.ServiceDetails != 0x5A5 {
		t.Errorf("cell 1 class/services = %+v, want subscriber=0x8001 bs=0x5a5", c1)
	}
	if !c1.HasTimeshare || c1.Timeshare != 0x15 || !c1.HasFrameOffset || c1.FrameOffset != 51 {
		t.Errorf("cell 1 timeshare/offset = %+v, want timeshare=0x15 offset=51", c1)
	}
	c2 := nb.Neighbours[1]
	if c2.CellID != 9 || c2.Synchronized || c2.CellServiceLevel != 3 || c2.MainCarrier != 2720 {
		t.Errorf("cell 2 = %+v, want id=9 unsynced sl=3 carrier=2720", c2)
	}
	if c2.HasExtension || c2.HasMCC || c2.HasMNC || c2.HasLA || c2.HasMaxTxPower ||
		c2.HasMinRxLevel || c2.HasSubscriberClass || c2.HasServiceDetails ||
		c2.HasTimeshare || c2.HasFrameOffset {
		t.Errorf("cell 2 carries optionals it did not broadcast: %+v", c2)
	}
}

// TestParseCMCERestoreAckLiteral: an MLE D-RESTORE-ACK wraps a CMCE SDU
// (5-bit CMCE type onward) — the framing tetra-kit's serviceMleSubsystem
// implements by forwarding the payload after the 3-bit MLE PDU type straight
// to its CMCE layer. The literal is a D-CONNECT for call id 0x1234.
func TestParseCMCERestoreAckLiteral(t *testing.T) {
	pdu := "101" + "100" + // PD=MLE, type=D-RESTORE-ACK
		"00010" + // CMCE type = D-CONNECT
		"01001000110100" + // call identifier = 0x1234
		"0000" + "0" + "0" + "00" + "0" + "0" + // timeout/hook/simplex/grant/perm/ownership
		"0" // O-bit: no optional elements
	msg, ok := ParseCMCERestoreAck(bitsOf(t, pdu))
	if !ok {
		t.Fatal("ParseCMCERestoreAck returned ok=false on a valid PDU")
	}
	if msg.Type != CMCETypeDConnect || msg.CallIdentifier != 0x1234 {
		t.Errorf("restored CMCE = %+v, want D-CONNECT call 0x1234", msg)
	}
	// A D-NWRK-BROADCAST is not a restore-ack; nor is a direct CMCE TL-SDU.
	if _, ok := ParseCMCERestoreAck(bitsOf(t, "101"+"010"+"1010101101010100"+"10"+"0")); ok {
		t.Error("ParseCMCERestoreAck accepted a D-NWRK-BROADCAST")
	}
	if _, ok := ParseCMCERestoreAck(bitsOf(t, "010"+"100"+"00010"+"01001000110100"+"000000000"+"0")); ok {
		t.Error("ParseCMCERestoreAck accepted a CMCE TL-SDU")
	}
}

// TestParseDNwrkBroadcastRejects covers the not-this-PDU and truncation exits.
func TestParseDNwrkBroadcastRejects(t *testing.T) {
	cases := map[string]string{
		// CMCE protocol discriminator (010), not MLE.
		"non-MLE PD": "010" + "010" + "1010101101010100" + "10" + "0",
		// MLE but a different PDU type (D-NEW-CELL, 000).
		"other MLE type": "101" + "000" + "1010101101010100" + "10" + "0",
		// Neighbour count promises a cell the bits cannot hold.
		"truncated cell": "101" + "010" + "1010101101010100" + "10" + "1" + "0" + "1" + "001" + "00101",
	}
	for name, s := range cases {
		if _, ok := ParseDNwrkBroadcast(bitsOf(t, s)); ok {
			t.Errorf("%s: ParseDNwrkBroadcast accepted", name)
		}
	}
}

// TestLearnNeighbourCellsFillsTopology: a decoded D-NWRK-BROADCAST lands in the
// control channel's TopologySnapshot as Neighbors (the systems report's
// "Neighbor sites"), publishes a site update, and the steady-state rebroadcast
// of unchanged content publishes nothing further.
func TestLearnNeighbourCellsFillsTopology(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Sys", FrequencyHz: 467_912_500})
	// Serving SYSINFO so a no-extension neighbour resolves in the serving band.
	cc.learnSysInfo(SysInfo{MainCarrier: 2716, FreqBand: 4, Offset: 3, DuplexSpacing: 6})

	nb := DNwrkBroadcast{
		Neighbours: []NeighbourCell{
			{CellID: 7, MainCarrier: 2720, Synchronized: true, HasMCC: true, MCC: 250, HasMNC: true, MNC: 13, HasLA: true, LA: 4001},
		},
	}
	// One sighting is NOT enough: a corrupted-but-CRC-passing TL-SDU can parse
	// as a plausible broadcast, so a cell surfaces only after the same content
	// decodes twice (the real broadcast repeats every few seconds).
	cc.learnNeighbourCells(nb, nil)
	if got := cc.TopologySnapshot().Neighbors; len(got) != 0 {
		t.Fatalf("a single sighting surfaced %d neighbours, want 0 (confirm-twice)", len(got))
	}
	cc.learnNeighbourCells(nb, nil)

	snap := cc.TopologySnapshot()
	if len(snap.Neighbors) != 1 {
		t.Fatalf("TopologySnapshot has %d neighbours, want 1", len(snap.Neighbors))
	}
	n := snap.Neighbors[0]
	if n.Site != 7 || n.ChannelNumber != 2720 {
		t.Errorf("neighbour = %+v, want Site=7 ChannelNumber=2720", n)
	}
	// No extension ⇒ serving band 4, offset +12.5 kHz: 400 MHz + 2720*25 kHz
	// + 12.5 kHz = 468.0125 MHz; uplink = -10 MHz (band 4 duplex).
	if n.FrequencyHz != 468_012_500 {
		t.Errorf("neighbour downlink = %d, want 468012500", n.FrequencyHz)
	}
	if n.UplinkHz != 458_012_500 {
		t.Errorf("neighbour uplink = %d, want 458012500", n.UplinkHz)
	}
	if want := "synced,mcc=250,mnc=13,la=4001"; n.StatusFlags != want {
		t.Errorf("StatusFlags = %q, want %q", n.StatusFlags, want)
	}

	countSiteUpdates := func() int {
		got := 0
		for {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindSiteUpdate {
					got++
				}
			default:
				return got
			}
		}
	}
	if n := countSiteUpdates(); n == 0 {
		t.Error("no KindSiteUpdate published for a newly-learned neighbour")
	}
	// Unchanged rebroadcast: no further publish, no further log.
	cc.learnNeighbourCells(nb, nil)
	if n := countSiteUpdates(); n != 0 {
		t.Errorf("unchanged rebroadcast published %d site updates, want 0", n)
	}
}

// TestLearnNeighbourCellsRejectsImplausibleCells: an implausible cell — a
// ≥1 GHz frequency-band extension (an operator's field report showed a
// confirmed 1.5 GHz "neighbour") or a zero main carrier — invalidates the
// ENTIRE broadcast it arrived in, plausible-looking siblings included. The
// list is parsed sequentially, so an impossible cell proves the bit alignment
// was lost and every sibling from the same bits is suspect; deterministic
// corruption repeats bit-identically, so confirm-twice cannot catch those
// siblings (the 4 Sep field log: a "cell 0, carrier 80, synced" phantom
// 65 MHz off-network confirmed twice alongside a band-13 implausible sibling
// in the same broadcasts and surfaced in the web UI).
func TestLearnNeighbourCellsRejectsImplausibleCells(t *testing.T) {
	cc := New(Options{SystemName: "Sys", FrequencyHz: 467_912_500})
	cc.learnSysInfo(SysInfo{MainCarrier: 2716, FreqBand: 4, Offset: 3, DuplexSpacing: 6})

	nb := DNwrkBroadcast{
		Neighbours: []NeighbourCell{
			{CellID: 2, MainCarrier: 698, HasExtension: true, FreqBand: 13}, // impossible band
			{CellID: 0, MainCarrier: 80, Synchronized: true},                // plausible-looking sibling (the field log's phantom)
		},
	}
	cc.learnNeighbourCells(nb, nil)
	cc.learnNeighbourCells(nb, nil)

	if got := cc.TopologySnapshot().Neighbors; len(got) != 0 {
		t.Fatalf("surfaced neighbours = %+v, want none — an implausible cell distrusts the whole broadcast", got)
	}

	// A fully-plausible broadcast is unaffected by the earlier rejection.
	good := DNwrkBroadcast{
		Neighbours: []NeighbourCell{{CellID: 4, MainCarrier: 2720, Synchronized: true}},
	}
	cc.learnNeighbourCells(good, nil)
	cc.learnNeighbourCells(good, nil)
	got := cc.TopologySnapshot().Neighbors
	if len(got) != 1 || got[0].Site != 4 {
		t.Fatalf("surfaced neighbours = %+v, want only the plausible cell 4", got)
	}
}

// TestNeighbourStatusFlagsRendersStatuses: the StatusFlags string carries the
// advertised load and the raw status fields when present, and never renders an
// absent optional as a zero.
func TestNeighbourStatusFlagsRendersStatuses(t *testing.T) {
	cell := NeighbourCell{
		Synchronized:      true,
		CellServiceLevel:  2,
		HasLA:             true,
		LA:                1031,
		HasServiceDetails: true,
		ServiceDetails:    0x5A5,
		HasTimeshare:      true,
		Timeshare:         0x15,
	}
	if got, want := neighbourStatusFlags(cell), "synced,load=medium,la=1031,bs_svc=0x5a5[dereg,min-mode,migration,voice,sndcp,adv-link],timeshare=0x15"; got != want {
		t.Errorf("StatusFlags = %q, want %q", got, want)
	}
	bare := NeighbourCell{}
	if got, want := neighbourStatusFlags(bare), "unsynced"; got != want {
		t.Errorf("bare StatusFlags = %q, want %q (absent fields must be omitted, not zero)", got, want)
	}
}

// TestParseDNwrkBroadcastRejectsSplicedFieldCapture pins the trailing-residue
// gate with a LITERAL TL-SDU from the 4 Sep 2026 field log (tl_sdu_hex, 17:02
// broadcast): a fragment reassembly that spliced two transmissions of the
// serving cell's rotating neighbour broadcast. The first cells parse perfectly
// (they are the real cells 9/10 with their true carriers and LAs) and the list
// "completes", but 132 bits of the second transmission trail the last cell — a
// genuine broadcast ends within a fill run (< 8 bits) of its final cell,
// because the MAC bounds the TM-SDU to the PDU's own length. The old parser
// accepted this and the misaligned later cells became phantom neighbour sites
// that repeated bit-identically past the confirm-twice dedup. Fails against
// the old parser (it returned ok=true with 7 "cells").
func TestParseDNwrkBroadcastRejectsSplicedFieldCapture(t *testing.T) {
	spliced := hexToBits(t,
		"ab1ba8e8a2ce0035fffd20ac2d30220380a0557680110200582baf4520ac2d3022"+
			"0380a0557680110200582baf40087fa030153fa60444100a0ab3d00110a40302bb340080")[:552]
	if _, ok := ParseDNwrkBroadcast(spliced); ok {
		t.Error("ParseDNwrkBroadcast accepted a spliced broadcast with 132 bits of trailing residue")
	}
}

// TestParseDNwrkBroadcastTrailingResidue: the gate's boundary. A genuine
// TL-SDU may end with up to 7 bits of MAC fill after the last cell (octet
// granularity of the length indication); 8 or more residual bits is proof of a
// misframed/spliced PDU.
func TestParseDNwrkBroadcastTrailingResidue(t *testing.T) {
	// Header + one cell (id=9, no optionals), ending flush.
	base := "101" + "010" + "1010101101010100" + "10" + "1" + "0" + "1" + "001" +
		"01001" + "00" + "0" + "11" + "101010100000" + "0"
	if _, ok := ParseDNwrkBroadcast(bitsOf(t, base)); !ok {
		t.Fatal("flush-ending broadcast rejected")
	}
	if nb, ok := ParseDNwrkBroadcast(bitsOf(t, base+"0000000")); !ok || len(nb.Neighbours) != 1 {
		t.Error("broadcast with 7 fill bits after the list rejected")
	}
	if _, ok := ParseDNwrkBroadcast(bitsOf(t, base+"00000000")); ok {
		t.Error("broadcast with 8 residual bits after the list accepted")
	}
}

// TestLearnNeighbourCellsExpiresUnadvertisedCells: a cell absent from every
// accepted broadcast for neighbourExpiry leaves the surfaced set (and the
// topology republishes), so a long session's "Neighbor sites" tracks the
// network's LIVE neighbour list — the 4-5 Sep 10-hour field report accumulated
// phantom sites for the daemon's whole lifetime because nothing ever pruned.
// Expiry is measured only across accepted broadcasts: a CC outage (no
// broadcasts at all) must not expire anything.
func TestLearnNeighbourCellsExpiresUnadvertisedCells(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cc := New(Options{SystemName: "Sys", FrequencyHz: 467_912_500,
		Now: func() time.Time { return now }})
	cc.learnSysInfo(SysInfo{MainCarrier: 2716, FreqBand: 4, Offset: 3, DuplexSpacing: 6})

	cellA := NeighbourCell{CellID: 7, MainCarrier: 2720, Synchronized: true}
	cellB := NeighbourCell{CellID: 9, MainCarrier: 2754}
	both := DNwrkBroadcast{Neighbours: []NeighbourCell{cellA, cellB}}
	onlyA := DNwrkBroadcast{Neighbours: []NeighbourCell{cellA}}

	cc.learnNeighbourCells(both, nil)
	cc.learnNeighbourCells(both, nil) // confirm-twice
	if got := len(cc.TopologySnapshot().Neighbors); got != 2 {
		t.Fatalf("surfaced %d neighbours, want 2", got)
	}

	// B keeps rotating out of the broadcast but the window has not elapsed yet:
	// it must survive.
	now = now.Add(neighbourExpiry / 2)
	cc.learnNeighbourCells(onlyA, nil)
	if got := len(cc.TopologySnapshot().Neighbors); got != 2 {
		t.Fatalf("cell expired before neighbourExpiry: surfaced %d, want 2", got)
	}

	// Past the window with B still absent: B is pruned, A (just re-advertised)
	// stays.
	now = now.Add(neighbourExpiry/2 + time.Minute)
	cc.learnNeighbourCells(onlyA, nil)
	snap := cc.TopologySnapshot()
	if len(snap.Neighbors) != 1 || snap.Neighbors[0].Site != 7 {
		t.Fatalf("after expiry snapshot = %+v, want only cell 7", snap.Neighbors)
	}
}
