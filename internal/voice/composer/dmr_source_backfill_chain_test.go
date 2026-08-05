package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerDMRBackfillsServiceOptions confirms the DMR voice chain
// publishes a CallSourceUpdate carrying the source + encrypted / emergency
// / priority it decodes from the embedded voice LC — the metadata a Tier
// III channel-grant CSBK doesn't carry, so without this an encrypted /
// emergency Tier III call would be logged clear.
func TestComposerDMRBackfillsServiceOptions(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	const (
		wantTG  = uint32(0x64)
		wantSrc = uint32(0x100200)
	)
	// ServiceOptions 0xC5 = emergency(bit7) + encrypted(bit6) + priority 5.
	gv := dmr.AssembleFLC(dmr.FLC{
		FLCO: dmr.FLCOGroupVoiceUser, DstAddr: wantTG, SrcAddr: wantSrc, ServiceOptions: 0xC5,
	})
	var lcInfos [][]byte
	for i := 0; i < 4; i++ {
		lcInfos = append(lcInfos, gv)
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

	// A Tier III-style grant: no service options on the grant itself.
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

	deadline := time.After(8 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("no KindCallSourceUpdate with the decoded service options")
		case ev := <-sub.C:
			if ev.Kind != events.KindCallSourceUpdate {
				continue
			}
			u, ok := ev.Payload.(trunking.CallSourceUpdate)
			if !ok || u.DeviceSerial != "VOICE-1" {
				continue
			}
			if u.SourceID != wantSrc {
				continue // wait for the clean decode
			}
			if !u.Encrypted || !u.Emergency || u.Priority != 5 {
				t.Fatalf("service options not backfilled: enc=%v emer=%v prio=%d",
					u.Encrypted, u.Emergency, u.Priority)
			}
			return
		}
	}
}
