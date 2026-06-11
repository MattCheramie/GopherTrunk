package tier3

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyAccumulation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := New(Options{Bus: bus, FrequencyHz: 851_000_000})

	// C_BCAST Gen_Site_Params (anncd_type 7 → payload[0]=0x38):
	// SystemID=0x1234, RFSS=5, Site=7.
	c.handleCSBK(3, CSBK{Opcode: OpBcast, Payload: [8]byte{0x38, 0x12, 0x34, 0x05, 0x07}})
	// Two adjacent sites via C_BCAST Adjacent_Site (anncd_type 6 →
	// payload[0]=0x30); the second repeats site 3 → de-duped.
	c.handleCSBK(3, CSBK{Opcode: OpBcast, Payload: [8]byte{0x30, 0x12, 0x34, 0x00, 0x03, 0x00, 0x09, 0x10}})
	c.handleCSBK(3, CSBK{Opcode: OpBcast, Payload: [8]byte{0x30, 0x12, 0x34, 0x00, 0x04, 0x00, 0x0A, 0x10}})
	c.handleCSBK(3, CSBK{Opcode: OpBcast, Payload: [8]byte{0x30, 0x12, 0x34, 0x00, 0x03, 0x00, 0x09, 0x10}})

	topo := c.Topology()
	if topo.SystemID != 0x1234 {
		t.Errorf("SystemID = %X, want 1234", topo.SystemID)
	}
	if topo.RFSS != 5 || topo.Site != 7 {
		t.Errorf("RFSS/Site = %d/%d, want 5/7", topo.RFSS, topo.Site)
	}
	if topo.ColorCode != 3 {
		t.Errorf("ColorCode = %d, want 3", topo.ColorCode)
	}
	if len(topo.Neighbors) != 2 {
		t.Fatalf("neighbors = %d, want 2 (deduped): %+v", len(topo.Neighbors), topo.Neighbors)
	}
	if topo.Neighbors[0].SiteID != 3 || topo.Neighbors[0].LCN != 9 {
		t.Errorf("neighbor[0] = %+v, want site 3 LCN 9", topo.Neighbors[0])
	}
}

func TestTopologyIdentityFirstWins(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := New(Options{Bus: bus})
	// Aloha sets SystemID + ColorCode first; a later SysInfo with SystemID 0
	// must not clobber the identity.
	c.handleCSBK(7, CSBK{Opcode: OpAloha, Payload: [8]byte{0x00, 0x00, 0xAB, 0xCD}})
	if got := c.Topology(); got.SystemID != 0xABCD || got.ColorCode != 7 {
		t.Errorf("after Aloha: %+v, want SystemID ABCD ColorCode 7", got)
	}
}
