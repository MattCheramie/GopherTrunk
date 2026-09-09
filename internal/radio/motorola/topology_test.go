package motorola

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyAccumulation(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	c := New(Options{Bus: bus, FrequencyHz: 854_562_500})

	// System ID + alternate CC (channel 0x090), then an adjacent
	// site's CC (channel 0x0A0), then a duplicate of the first.
	feed := []OSW{
		{Address: 0x2A2A, Command: CmdFirstAlternate},
		{Address: 0x6000 | 0x090, Command: 0x2F8},
		{Address: 0x2A2A, Command: CmdFirstAlternate},
		{Address: 0x6000 | 0x0A0, Group: true, Command: 0x2F8},
		{Address: 0x2A2A, Command: CmdFirstAlternate},
		{Address: 0x6000 | 0x090, Command: 0x2F8},
		{Address: 0x02F8, Command: CmdIdle},
		{Address: 0x02F8, Command: CmdIdle},
	}
	for _, o := range feed {
		c.Ingest(o)
	}

	topo := c.Topology()
	if topo.SystemID != 0x2A2A {
		t.Errorf("SystemID = %#x, want 0x2A2A", topo.SystemID)
	}
	if len(topo.Neighbors) != 2 {
		t.Fatalf("neighbors = %d, want 2 (deduped): %+v", len(topo.Neighbors), topo.Neighbors)
	}
	if topo.Neighbors[0].LCN != 0x090 || topo.Neighbors[0].Adjacent {
		t.Errorf("neighbor[0] = %+v, want LCN 0x090 alternate", topo.Neighbors[0])
	}
	if topo.Neighbors[1].LCN != 0x0A0 || !topo.Neighbors[1].Adjacent {
		t.Errorf("neighbor[1] = %+v, want LCN 0x0A0 adjacent", topo.Neighbors[1])
	}
}
