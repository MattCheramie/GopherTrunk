package nxdn

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestTopologyFromSiteInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	c := NewControlChannel(bus, nil, 851_000_000, Rate9600)

	// SITE_INFO payload: LocationID=p[0:2], SiteID=p[2:4], SystemID=p[4:6].
	var p [8]byte
	p[0], p[1] = 0x00, 0x2A // LocationID 42
	p[2], p[3] = 0x00, 0x05 // SiteID 5
	p[4], p[5] = 0x12, 0x34 // SystemID 0x1234
	c.IngestFrame(LICH{RFCh: RFChControl, ParityOK: true}, &CACMessage{Type: RCCHSITEINFO, Payload: p})

	topo := c.Topology()
	if topo.SystemID != 0x1234 || topo.SiteID != 5 || topo.LocationID != 42 {
		t.Errorf("topology = %+v, want SystemID 1234 Site 5 Location 42", topo)
	}
}
