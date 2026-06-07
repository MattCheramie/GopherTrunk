package tetra

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyFromSystemBroadcast(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := New(Options{Bus: bus, FrequencyHz: 390_000_000})
	c.SetColourCode(7)

	// MLE SYSINFO payload: 10b MCC + 14b MNC + 14b LA. This encodes
	// MCC=3, MNC=5, LocationArea=64 (see AsSystemBroadcast bit layout).
	pdu := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: []byte{0x00, 0xC0, 0x05, 0x01, 0x00},
	}
	c.Ingest(pdu)

	topo := c.Topology()
	if topo.MCC != 3 || topo.MNC != 5 || topo.LocationArea != 64 {
		t.Errorf("topology = %+v, want MCC 3 MNC 5 LA 64", topo)
	}
	if topo.ColourCode != 7 {
		t.Errorf("ColourCode = %d, want 7", topo.ColourCode)
	}
}
