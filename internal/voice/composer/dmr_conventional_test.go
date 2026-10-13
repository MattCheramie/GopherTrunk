package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerDMRConventionalCallFollowsActiveSlot pins issue #1261's DMR
// scan-list channels. The scanner grants a conventional call under the
// channel's SYNTHETIC GroupID, which no embedded LC ever names. The DMR
// chain's slot router bound timeslots by LC talkgroup == GroupID, so on a
// conventional channel it marked both slots foreign and recorded nothing.
// A Conventional grant must record one slot's call — whichever carries
// voice first — without the other slot leaking in.
func TestComposerDMRConventionalCallFollowsActiveSlot(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 10
		span        = 8
		alpha       = 0.20
		deviation   = 1944.0
		superframes = 12
	)
	// Two concurrent calls on a repeater's two slots (talkgroups 1 and 2),
	// neither of them the scan list's synthetic ID.
	dibits, aInfos, bInfos := buildInterleavedVoiceStreamLC(t, superframes, 1, 2)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"CONV-1": src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
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
				System: "scanner", Protocol: "dmr-tier2", Conventional: true,
				GroupID: 0x80000003, GroupLabel: "DMR Rptr", FrequencyHz: 442_387_500,
				DMRInterleavedVoice: true,
			},
			DeviceSerial: "CONV-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) && len(sink.rawFrames("CONV-1")) < 4*dmrvoice.FramesPerSuperframe {
		time.Sleep(20 * time.Millisecond)
	}
	got := sink.rawFrames("CONV-1")
	got = got[:len(got)-len(got)%dmrvoice.FramesPerSuperframe]
	if len(got) == 0 {
		t.Fatal("no superframe of a conventional DMR call reached the sidecar — slots routed by the synthetic GroupID?")
	}
	inA, inB := matchesAnySuperframe(got, aInfos), matchesAnySuperframe(got, bInfos)
	if !inA && !inB {
		t.Fatal("recorded frames match neither slot's call")
	}
	if inA && inB {
		t.Error("both slots' calls were recorded into one conventional call")
	}
}
