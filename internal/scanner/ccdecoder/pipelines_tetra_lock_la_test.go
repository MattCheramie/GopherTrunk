package ccdecoder

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestTETRACCRealAirLockCarriesLocationArea pins the #1321 location-area fix on
// real air, through the production pipeline. The fixture is a 2 s slice of a
// live 467.9125 MHz control channel (MCC 250 / MNC 13) whose SYSINFO
// broadcasts location area 1052. The sync burst that locks the channel carries
// no location area, and the D-MLE-SYSINFO one was logged but never reached the
// lock identity, so the topology snapshot (the Systems panel) and every lock
// line read location area 0 on any real cell.
func TestTETRACCRealAirLockCarriesLocationArea(t *testing.T) {
	iq := loadCS16(t, marginalCCFixture)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(64)
	t.Cleanup(bus.Close)
	pp, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "latest", FrequencyHz: 467_912_500,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "latest", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{467_912_500},
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p := pp.(*tetraPipeline)

	const chunk = 4096
	for pos := 0; pos < len(iq); pos += chunk {
		end := pos + chunk
		if end > len(iq) {
			end = len(iq)
		}
		p.Process(iq[pos:end])
	}

	topo := p.cc.Topology()
	if topo.MCC != 250 || topo.MNC != 13 {
		t.Fatalf("fixture did not lock to MCC 250 / MNC 13: got %d / %d", topo.MCC, topo.MNC)
	}
	if topo.LocationArea != 1052 {
		t.Errorf("topology location area = %d, want 1052 (the cell's SYSINFO location area)", topo.LocationArea)
	}
	if snap := p.cc.TopologySnapshot(); snap.LocationArea != 1052 {
		t.Errorf("topology snapshot location area = %d, want 1052", snap.LocationArea)
	}
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, `msg="tetra cc locked"`) && strings.Contains(l, "la=0") {
			t.Errorf("lock line claims location area 0 on a cell that broadcasts 1052: %s", l)
		}
	}
	if !strings.Contains(buf.String(), `msg="tetra cc location area learned"`) {
		t.Errorf("no line announced the learned location area:\n%s", buf.String())
	}
}
