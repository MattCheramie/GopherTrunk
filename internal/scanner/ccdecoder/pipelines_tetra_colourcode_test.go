package ccdecoder

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestTETRATopologyColourCodeMasksToSixBits pins the fix for the surfaced TETRA
// colour code. The control channel stores the 30-bit extended colour code
// (MCC<<20 | MNC<<6 | CC) as the scrambler seed; the ETSI colour code the user
// sees is the low 6 bits. The old display path masked 0xFF, dragging in two
// stray MNC bits — for the reporter's two sites that produced 108 and 77 instead
// of the real 0x2c (44) and 0x0d (13) reported by tetra-demodulator.
func TestTETRATopologyColourCodeMasksToSixBits(t *testing.T) {
	cases := []struct {
		name string
		ext  uint32
		want uint8
	}{
		{"site_467_913", 262144876, 0x2c}, // colour_ext from the reporter's log → CC 44
		{"site_469_875", 262144845, 0x0d}, // colour_ext from the reporter's log → CC 13
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(8)
			defer bus.Close()
			// A non-zero TETRAColourCode makes the factory apply it via
			// SetColourCode under the default ChannelCodingOn, so it lands in the
			// control channel exactly as a runtime BSCH decode would.
			p, err := newTETRAPipeline(PipelineOptions{
				Bus: bus, SystemName: "Test", FrequencyHz: 467_913_000,
				SampleRateHz: 144_000,
				System: trunking.System{
					Name: "Test", Protocol: trunking.ProtocolTETRA,
					ControlChannels: []uint32{467_913_000},
					TETRAColourCode: tc.ext,
				},
			})
			if err != nil {
				t.Fatalf("newTETRAPipeline: %v", err)
			}
			snap := p.(*tetraPipeline).TopologySnapshot()
			if snap.ColorCode != tc.want {
				t.Errorf("TopologySnapshot ColorCode = %d (%#x), want %d (%#x)",
					snap.ColorCode, snap.ColorCode, tc.want, tc.want)
			}
		})
	}
}
