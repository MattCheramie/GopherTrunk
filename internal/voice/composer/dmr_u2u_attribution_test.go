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

// TestComposerDMRAttributesAliasOnUnitToUnitCall confirms the voice chain
// learns the call's source radio from a unit-to-unit (private) voice LC —
// not just a group LC — so talker-alias metadata is attributed on private
// calls. Regression for the source-tracking half of the unit-to-unit work.
func TestComposerDMRAttributesAliasOnUnitToUnitCall(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	const wantAlias = "Truck 349"
	const wantSrc = uint32(77000)

	// A private call: the Voice LC destination is a called subscriber; its
	// source is the radio the alias names.
	u2u := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOUnitToUnitVoice, DstAddr: 0x00ABCD, SrcAddr: wantSrc})
	aliasFrags := dmr.AssembleTalkerAliasFragments(dmr.TalkerAliasFormatUTF8, wantAlias)

	var lcInfos [][]byte
	for cycle := 0; cycle < 6; cycle++ {
		lcInfos = append(lcInfos, u2u)
		lcInfos = append(lcInfos, aliasFrags...)
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
				GroupID: 0x00ABCD, Individual: true, FrequencyHz: 460_000_000,
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
			t.Fatalf("no KindTalkerAlias attributed to the private call's source %d", wantSrc)
		case ev := <-sub.C:
			if ev.Kind != events.KindTalkerAlias {
				continue
			}
			ta, ok := ev.Payload.(trunking.TalkerAlias)
			if !ok || ta.Alias != wantAlias {
				continue
			}
			if ta.SourceID != wantSrc {
				t.Fatalf("alias SourceID = %d, want %d (from unit-to-unit LC)", ta.SourceID, wantSrc)
			}
			return
		}
	}
}
