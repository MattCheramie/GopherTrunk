package main

import (
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// TestTETRANeighbourReportReplay decodes one branch of a pre-combine diversity
// capture (GT_DIVERSITY_CAPTURE, like TestDiversityCombinerReplay) through the
// shared TETRA control-channel path and reports the D-NWRK-BROADCAST neighbour
// cells that land in the TopologySnapshot — the on-air verification harness
// for the MLE neighbour decode (mle_parse.go): run it on a real capture and
// check the printed carriers/identities against the network's known layout.
// GT_DIVERSITY_TUNE_HZ offsets the DDC like the combiner harness.
func TestTETRANeighbourReportReplay(t *testing.T) {
	_, br1, meta := loadDiversityCapture(t)
	tuneHz := 0.0
	if s := os.Getenv("GT_DIVERSITY_TUNE_HZ"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("GT_DIVERSITY_TUNE_HZ: %v", err)
		}
		tuneHz = v
	}

	ddc := ccdecoder.NewDownconverterWithOffset(meta.SampleRateHz, 144_000, tuneHz)
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	cc := tetra.New(tetra.Options{SystemName: "nbreplay", Bus: bus, FrequencyHz: meta.CenterFreqHz})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	ch, _ := tetra.ParseChannelType("")
	cc.SetExpectedChannel(ch)
	cc.SetColourCode(0)

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        ddc.OutRateHz(),
		DibitSink:           func(d []uint8, base int) { cc.Process(d, base) },
		SoftSink:            func(diffs []complex64, base int) { cc.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true,
	})
	const chunk = 65536
	var scratch []complex64
	for pos := 0; pos < len(br1); pos += chunk {
		end := min(pos+chunk, len(br1))
		dec := ddc.Process(scratch, br1[pos:end])
		if len(dec) > 0 {
			rx.Process(dec)
		}
		scratch = dec[:0]
	}

	snap := cc.TopologySnapshot()
	topo := cc.Topology()
	t.Logf("serving cell: mcc=%d mnc=%d la=%d main_carrier=%d downlink_hz=%d",
		topo.MCC, topo.MNC, topo.LocationArea, topo.MainCarrier, topo.DownlinkHz)
	t.Logf("neighbour cells decoded: %d", len(snap.Neighbors))
	for _, n := range snap.Neighbors {
		t.Logf("  cell=%d carrier=%d downlink_hz=%d uplink_hz=%d flags=%q",
			n.Site, n.ChannelNumber, n.FrequencyHz, n.UplinkHz, n.StatusFlags)
	}
}
