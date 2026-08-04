package composer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerDMRPublishesLocation drives the full DMR voice chain with a
// stream that carries a group-voice LC (establishing the call's source
// radio) then GPS Info embedded LCs, and confirms the chain publishes a
// KindLocation event bound to that radio.
func TestComposerDMRPublishesLocation(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	const (
		wantLat = -37.8136
		wantLon = 144.9631
		wantSrc = uint32(90210)
		wantTG  = uint32(7)
	)

	gv := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: wantTG, SrcAddr: wantSrc})
	gps := dmr.AssembleGPSInfo(dmr.GPSInfo{Latitude: wantLat, Longitude: wantLon})

	var lcInfos [][]byte
	for cycle := 0; cycle < 6; cycle++ {
		lcInfos = append(lcInfos, gv, gps)
	}
	dibits := buildVoiceStreamWithLCInfos(t, lcInfos)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "DMRSite", Protocol: "dmr-tier3",
				GroupID: wantTG, FrequencyHz: 460_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	const tol = 360.0 / (1 << 25)
	deadline := time.After(8 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("no KindLocation published within the deadline")
		case ev := <-sub.C:
			if ev.Kind != events.KindLocation {
				continue
			}
			loc, ok := ev.Payload.(trunking.Location)
			if !ok {
				continue
			}
			if math.Abs(loc.Latitude-wantLat) > tol || math.Abs(loc.Longitude-wantLon) > tol {
				continue // a warmup/garbled partial — keep waiting for the clean fix
			}
			if loc.Protocol != "dmr" || loc.System != "DMRSite" ||
				loc.RadioID != wantSrc || loc.Talkgroup != wantTG {
				t.Fatalf("location metadata wrong: proto=%q sys=%q rid=%d tg=%d",
					loc.Protocol, loc.System, loc.RadioID, loc.Talkgroup)
			}
			return
		}
	}
}
