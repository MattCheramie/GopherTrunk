package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// embeddedFragsFromInfo runs a 9-octet info block through the embedded-LC
// BPTC encoder and returns the four 32-bit fragments bursts B–E carry.
func embeddedFragsFromInfo(t *testing.T, info []byte) [4][]byte {
	t.Helper()
	lc := make([]byte, framing.EmbLCBits)
	for i := 0; i < framing.EmbLCBits; i++ {
		lc[i] = (info[i/8] >> uint(7-(i%8))) & 1
	}
	ch := framing.EncodeEmbeddedLC(lc)
	var frags [4][]byte
	for i := range frags {
		frags[i] = ch[i*framing.EmbeddedFragmentBits : (i+1)*framing.EmbeddedFragmentBits]
	}
	return frags
}

// buildVoiceStreamWithLCInfos builds a single-slot voice dibit stream in
// which superframe s embeds lcInfos[s] (a raw 9-octet embedded-LC info
// block) across bursts B–E; a nil entry embeds plain BS-Data filler.
func buildVoiceStreamWithLCInfos(t *testing.T, lcInfos [][]byte) []uint8 {
	t.Helper()
	lcss := func(b int) dmr.LCSS {
		switch b {
		case 1:
			return dmr.LCSSFirst
		case 4:
			return dmr.LCSSLast
		default:
			return dmr.LCSSCont
		}
	}
	dibits := make([]uint8, 240) // clock-settling lead
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for s, info := range lcInfos {
		var frags [4][]byte
		if info != nil {
			frags = embeddedFragsFromInfo(t, info)
		}
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			switch {
			case b == 0:
				sync = dmr.BSVoice.Dibits
			case b >= 1 && b <= 4 && info != nil:
				sync = embeddedSyncDibits(dmr.EMB{ColorCode: 1, LCSS: lcss(b)}, frags[b-1])
			}
			// Audio content is irrelevant to this test; any valid AMBE frames
			// keep the superframe decodable so the embedded LC is reached.
			frames := make([][]byte, dmrvoice.FramesPerBurst)
			for i := range frames {
				f, err := dmrvoice.EncodeAMBEFrame(mkInfo(s*100 + b*3 + i))
				if err != nil {
					t.Fatalf("EncodeAMBEFrame: %v", err)
				}
				frames[i] = f
			}
			dibits = append(dibits, voiceBurstDibits(frames, sync)...)
		}
	}
	return dibits
}

// TestComposerDMRPublishesTalkerAlias drives the full DMR voice chain with
// a stream that first carries a group-voice LC (establishing the call's
// source radio) and then the talker-alias header + block embedded LCs, and
// confirms the chain reassembles and publishes a KindTalkerAlias event.
func TestComposerDMRPublishesTalkerAlias(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	const wantAlias = "Truck 349"
	const wantSrc = uint32(4242)

	gv := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 7, SrcAddr: wantSrc})
	aliasFrags := dmr.AssembleTalkerAliasFragments(dmr.TalkerAliasFormatUTF8, wantAlias)

	// Repeat [group-voice, alias-header, alias-block…] several times so the
	// receiver's sync/cadence warmup can drop the first cycle and a later,
	// fully-aligned cycle still completes the alias.
	var lcInfos [][]byte
	for cycle := 0; cycle < 6; cycle++ {
		lcInfos = append(lcInfos, gv)
		for _, f := range aliasFrags {
			lcInfos = append(lcInfos, f)
		}
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
				GroupID: 7, FrequencyHz: 460_000_000,
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
			t.Fatalf("no KindTalkerAlias %q published within the deadline", wantAlias)
		case ev := <-sub.C:
			if ev.Kind != events.KindTalkerAlias {
				continue
			}
			ta, ok := ev.Payload.(trunking.TalkerAlias)
			if !ok {
				continue
			}
			if ta.Alias != wantAlias {
				continue // a warmup/garbled partial — keep waiting for the clean one
			}
			if ta.Protocol != "dmr" || ta.System != "DMRSite" || ta.SourceID != wantSrc {
				t.Fatalf("alias metadata wrong: proto=%q sys=%q src=%d",
					ta.Protocol, ta.System, ta.SourceID)
			}
			if ta.Unreliable {
				t.Fatalf("clean alias %q flagged unreliable", ta.Alias)
			}
			return
		}
	}
}
