package edacs

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyAccumulation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := New(Options{Bus: bus, FrequencyHz: 851_000_000})

	c.Ingest(CCW{Command: CmdSystemID, Address: 0x2A})
	c.Ingest(CCW{Command: CmdAdjacentSite, Address: 11, LCN: 4})
	c.Ingest(CCW{Command: CmdAdjacentSite, Address: 12, LCN: 5})
	c.Ingest(CCW{Command: CmdAdjacentSite, Address: 11, LCN: 4}) // dup → deduped

	topo := c.Topology()
	if topo.SystemID != 0x2A {
		t.Errorf("SystemID = %X, want 2A", topo.SystemID)
	}
	if len(topo.Neighbors) != 2 {
		t.Fatalf("neighbors = %d, want 2: %+v", len(topo.Neighbors), topo.Neighbors)
	}
	if topo.Neighbors[0].SiteID != 11 || topo.Neighbors[0].LCN != 4 {
		t.Errorf("neighbor[0] = %+v, want site 11 LCN 4", topo.Neighbors[0])
	}
}
