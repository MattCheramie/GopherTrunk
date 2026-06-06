package siglab

import (
	"testing"

	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestP25TopologyMapping(t *testing.T) {
	net := p25phase1.NetworkConfig{
		WACN:     0xBEE99,
		SystemID: 0x49A,
		RFSS:     1,
		Site:     2,
		LRA:      0x0C,
		Secondary: []p25phase1.Channel{
			{ChannelID: 1, ChannelNumber: 100},
		},
		Neighbors: []p25phase1.NeighborSite{
			{RFSS: 1, Site: 3, ChannelID: 1, ChannelNumber: 200},
			{RFSS: 1, Site: 4, ChannelID: 1, ChannelNumber: 300},
		},
	}
	plan := []p25phase1.IdentifierUpdate{
		{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500, BandwidthHz: 12_500, TxOffsetHz: -45_000_000},
	}

	got := p25Topology(net, plan)
	if got.WACN != 0xBEE99 || got.SystemID != 0x49A {
		t.Errorf("identity = WACN %X SYSID %X, want BEE99/49A", got.WACN, got.SystemID)
	}
	if got.RFSS != 1 || got.Site != 2 || got.LRA != 0x0C {
		t.Errorf("RFSS/Site/LRA = %d/%d/%d, want 1/2/12", got.RFSS, got.Site, got.LRA)
	}
	if len(got.Secondary) != 1 || got.Secondary[0].ChannelNumber != 100 {
		t.Errorf("secondary = %+v, want one channel 100", got.Secondary)
	}
	if len(got.Neighbors) != 2 || got.Neighbors[0].Site != 3 || got.Neighbors[1].Site != 4 {
		t.Errorf("neighbors = %+v, want sites 3,4", got.Neighbors)
	}
	if len(got.BandPlan) != 1 || got.BandPlan[0].BaseHz != 851_000_000 || got.BandPlan[0].SpacingHz != 12_500 {
		t.Errorf("band plan = %+v, want base 851M spacing 12.5k", got.BandPlan)
	}
	if got.Empty() {
		t.Error("snapshot reported Empty despite populated fields")
	}
}

func TestTopologySnapshotEmpty(t *testing.T) {
	if !(&TopologySnapshot{}).Empty() {
		t.Error("zero TopologySnapshot should be Empty")
	}
	if (*TopologySnapshot)(nil).Empty() != true {
		t.Error("nil TopologySnapshot should be Empty")
	}
	if (&TopologySnapshot{Neighbors: []NeighborRef{{Site: 1}}}).Empty() {
		t.Error("snapshot with a neighbor should not be Empty")
	}
}

// TestIdentifyReaderMatchesIdentify confirms the in-memory IdentifyReader
// produces the same winner as the file-based Identify over the same capture —
// the parity guarantee the live sweep relies on.
func TestIdentifyReaderMatchesIdentify(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	cfg := IdentifyConfig{SampleRateHz: meta.SampleRateHz, Format: FormatF32}
	fromIQ, err := IdentifyIQ(iq, "p25.iq", cfg)
	if err != nil {
		t.Fatalf("IdentifyIQ: %v", err)
	}
	if fromIQ.Winner != trunking.ProtocolP25.String() {
		t.Errorf("IdentifyIQ winner = %q, want p25", fromIQ.Winner)
	}
	if fromIQ.Inconclusive {
		t.Errorf("IdentifyIQ inconclusive (conf %.2f)", fromIQ.Confidence)
	}

	// Same buffer through a file → Identify should agree on the winner.
	dir := t.TempDir()
	path := dir + "/p25.cfile"
	if err := WriteCapture(path, iq, FormatF32); err != nil {
		t.Fatalf("WriteCapture: %v", err)
	}
	fromFile, err := Identify(path, cfg)
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if fromIQ.Winner != fromFile.Winner {
		t.Errorf("winner mismatch: IdentifyIQ=%q Identify=%q", fromIQ.Winner, fromFile.Winner)
	}
	if len(fromIQ.Candidates) != len(fromFile.Candidates) {
		t.Errorf("candidate count mismatch: IQ=%d file=%d", len(fromIQ.Candidates), len(fromFile.Candidates))
	}
}
