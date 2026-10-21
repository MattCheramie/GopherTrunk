package ccdecoder

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
)

// TestNXDNPipelineLocksOnRealAirControlChannel: the production NXDN control
// channel pipeline, default system config, on one second of the #1261
// reporter's NXDN96 control channel (453.775 MHz, which sits ≈ -650 Hz off
// as captured and was drifting). It must lock on the site the channel
// announces, and keep doing so with the carrier a further ±1 kHz away: the
// reporter's dongle walked 34 Hz in the capture's 8.7 s alone.
func TestNXDNPipelineLocksOnRealAirControlChannel(t *testing.T) {
	raw, err := os.ReadFile("../../radio/nxdn/testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16")
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		iq[i] = complex(float32(int16(binary.LittleEndian.Uint16(raw[4*i:])))/32768,
			float32(int16(binary.LittleEndian.Uint16(raw[4*i+2:])))/32768)
	}
	for _, shiftHz := range []float64{0, -1000, +1000} {
		bus := events.NewBus(64)
		sub := bus.Subscribe()
		p, err := newNXDNPipeline(PipelineOptions{
			Bus: bus, SystemName: "Site", FrequencyHz: 453_775_000, SampleRateHz: 48_000,
		})
		if err != nil {
			t.Fatal(err)
		}
		in := make([]complex64, len(iq))
		for i := range iq {
			ph := 2 * math.Pi * shiftHz * float64(i) / 48_000
			in[i] = iq[i] * complex64(complex(math.Cos(ph), math.Sin(ph)))
		}
		for i := 0; i < len(in); i += 4096 {
			p.Process(in[i:min(i+4096, len(in))])
		}
		frames := p.(*nxdnPipeline).cc.DecodedFrames()
		var lock *nxdn.LockState
		deadline := time.After(time.Second)
	drain:
		for lock == nil {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLocked {
					ls := ev.Payload.(nxdn.LockState)
					lock = &ls
				}
			case <-deadline:
				break drain
			}
		}
		t.Logf("shift %+.0f Hz: %d CAC frames decoded, lock %+v", shiftHz, frames, lock)
		if lock == nil || lock.SystemID != 85 || lock.SiteID != 1 {
			t.Errorf("shift %+.0f Hz: lock %+v, want system 85 site 1", shiftHz, lock)
		}
		if frames < 15 {
			t.Errorf("shift %+.0f Hz: %d CAC frames decoded, want ≥ 15 of the slice's ~20", shiftHz, frames)
		}
		sub.Close()
		bus.Close()
	}
}
