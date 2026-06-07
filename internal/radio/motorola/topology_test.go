package motorola

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyAccumulation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := New(Options{Bus: bus, FrequencyHz: 851_000_000})

	// System ID Extended (opcode 0x080 = Command>>4): ID 0x2A.
	c.Ingest(OSW{Address: 0x2A, Command: 0x0800})
	// Adjacent Site Status (opcode 0x31B): site/LCN in Address/LCN nibble.
	c.Ingest(OSW{Address: 11, Command: 0x31B4})
	c.Ingest(OSW{Address: 12, Command: 0x31B5})
	c.Ingest(OSW{Address: 11, Command: 0x31B4}) // dup → deduped

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
