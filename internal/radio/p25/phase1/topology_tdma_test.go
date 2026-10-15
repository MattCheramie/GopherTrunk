package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// The topology snapshot is what the network report and the hunt resolve
// (id, number) pairs from, so it must carry the IDEN_UP_TDMA slot count, or
// every consumer computes carrier n instead of n/slots (issue #1273).
func TestTopologySnapshotCarriesTDMASlotsPerCarrier(t *testing.T) {
	ident := TSBK{LB: true, Opcode: OpIdentifierUpdateTDMA}
	ident.Payload = AssembleIdentifierUpdateTDMA(IdentifierUpdate{
		ChannelID: 3, BandwidthHz: 12_500, SpacingHz: 12_500, BaseHz: 762_006_250,
		AccessTDMA: true, // channel type 3: 2 slots per carrier
	})
	bus := events.NewBus(16)
	defer bus.Close()
	cc := New(Options{
		Bus:         bus,
		SystemName:  "MOSWIN",
		FrequencyHz: 161_912_500,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	cc.Process(buildLockedStreamWithTSBK(10, 0x452, DUIDTrunkingSignaling, ident), 0)

	snap := cc.TopologySnapshot()
	if snap == nil || len(snap.BandPlan) != 1 {
		t.Fatalf("no band plan in the topology snapshot: %+v", snap)
	}
	b := snap.BandPlan[0]
	if b.SlotsPerCarrier != 2 {
		t.Fatalf("snapshot slot SlotsPerCarrier = %d, want 2 (AccessTDMA=%v)", b.SlotsPerCarrier, b.AccessTDMA)
	}
	if got, want := b.ChannelHz(11), uint32(762_006_250+5*12_500); got != want {
		t.Errorf("ChannelHz(11) = %d, want %d", got, want)
	}
}
