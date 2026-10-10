package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerP25ConventionalCallIsNotTalkgroupGated pins issue #1239's P25
// conventional channels. The conventional scanner grants a call under the
// scan-list channel's SYNTHETIC GroupID (0x80000000|index, or an explicit
// talkgroup_id), which no on-air link control ever carries. The P25 Phase 1
// chain gated every LDU on the LDU1 talkgroup matching the grant's GroupID,
// so on a conventional channel every frame read as a foreign talkgroup: the
// recording got almost nothing and the call was ended as "another talkgroup
// took the frequency". A Conventional grant must record whatever talkgroup
// the radio keys up on.
func TestComposerP25ConventionalCallIsNotTalkgroupGated(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 12
	)
	dibits := buildP25P1VoiceStreamWithLC(t, ldus, phase1.LinkControl{
		LCFormat:    phase1.LCOGroupVoiceChannelUser,
		TalkgroupID: 1, // the radio's own talkgroup, unrelated to the scan list
		SourceID:    0x1234,
	})
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	sink := &recordingSink{}
	bus := events.NewBus(64)
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
				System: "Conventional", Protocol: "p25", Conventional: true,
				GroupID: 0x80000002, GroupLabel: "VCALL10", FrequencyHz: 155_752_500,
			},
			DeviceSerial: "CONV-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// Acquisition costs the chain a few LDUs at each end of this short
	// synthetic stream (8 of 12 decode); gated, it records none at all.
	const want = ldus / 2 * phase1.LDUVoiceSubframeCount
	rawFrames := func() int {
		sink.mu.Lock()
		defer sink.mu.Unlock()
		return len(sink.raw["CONV-1"])
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) && rawFrames() < want {
		time.Sleep(20 * time.Millisecond)
	}
	if got := rawFrames(); got < want {
		t.Fatalf("recorded %d IMBE frames of a %d-LDU conventional P25 call, want >= %d — audio gated on the synthetic GroupID?",
			got, ldus, want)
	}
}

// Trunked grants keep gating on their talkgroup; only conventional ones
// turn it off.
func TestVoiceGateTalkgroup(t *testing.T) {
	if got := voiceGateTalkgroup(trunking.Grant{GroupID: 356}); got != 356 {
		t.Errorf("trunked grant gates on %d, want 356", got)
	}
	if got := voiceGateTalkgroup(trunking.Grant{GroupID: 0x80000002, Conventional: true}); got != 0 {
		t.Errorf("conventional grant gates on %#x, want 0 (gating off)", got)
	}
}
