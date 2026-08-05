package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildTerminatorBurst builds a full Terminator-with-LC burst carrying flc
// (RS(12,9) terminator-seeded, BPTC, DTTerminatorWithLC slot type, BS-Data
// sync) — the burst the followed traffic channel sends at end-of-call.
func buildTerminatorBurst(t *testing.T, flc dmr.FLC, colorCode uint8) []uint8 {
	t.Helper()
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedTerminatorLC[i]
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
	}
	payloadDibits := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slotDibits := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTTerminatorWithLC}))
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)
	return burst
}

// TestComposerDMRReleasesOnTerminator drives the full DMR voice chain with
// a stream that carries a group call (group-voice LCs) and then a
// Terminator-with-LC for that group, and confirms the chain publishes a
// KindCallRelease at once rather than waiting out the hangtime.
func TestComposerDMRReleasesOnTerminator(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	const wantTG = uint32(0x64)

	gv := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: wantTG, SrcAddr: 0x100200})
	var lcInfos [][]byte
	for i := 0; i < 3; i++ {
		lcInfos = append(lcInfos, gv)
	}
	dibits := buildVoiceStreamWithLCInfos(t, lcInfos)
	// Append the terminator for this group, then trailing filler so it buffers.
	dibits = append(dibits, buildTerminatorBurst(t, dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: wantTG, SrcAddr: 0x100200}, 1)...)
	dibits = append(dibits, make([]uint8, 200)...)
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

	deadline := time.After(8 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("no KindCallRelease published on the Terminator")
		case ev := <-sub.C:
			if ev.Kind != events.KindCallRelease {
				continue
			}
			r, ok := ev.Payload.(trunking.CallRelease)
			if !ok {
				continue
			}
			if r.System != "DMRSite" || r.GroupID != wantTG {
				t.Fatalf("release identity = %s/%X, want DMRSite/%X", r.System, r.GroupID, wantTG)
			}
			return
		}
	}
}
