package tetra

import (
	"strings"
	"testing"

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
	// remaining six optionals absent.
	pdu += "00101" + "11" + "1" + "01" + "101010011100" + "1" +
		"1" + "0100" + "11" + "110" + "0" +
		"1" + "0011111010" +
		"1" + "00000000001101" +
		"1" + "00111110100001" +
		"0" + "0" + "0" + "0" + "0" + "0"
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
	c2 := nb.Neighbours[1]
	if c2.CellID != 9 || c2.Synchronized || c2.CellServiceLevel != 3 || c2.MainCarrier != 2720 {
		t.Errorf("cell 2 = %+v, want id=9 unsynced sl=3 carrier=2720", c2)
	}
	if c2.HasExtension || c2.HasMCC || c2.HasMNC || c2.HasLA {
		t.Errorf("cell 2 carries optionals it did not broadcast: %+v", c2)
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
	cc.learnNeighbourCells(nb)
	if got := cc.TopologySnapshot().Neighbors; len(got) != 0 {
		t.Fatalf("a single sighting surfaced %d neighbours, want 0 (confirm-twice)", len(got))
	}
	cc.learnNeighbourCells(nb)

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
	cc.learnNeighbourCells(nb)
	if n := countSiteUpdates(); n != 0 {
		t.Errorf("unchanged rebroadcast published %d site updates, want 0", n)
	}
}
