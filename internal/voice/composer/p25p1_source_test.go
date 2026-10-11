package composer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// buildP25P1VoiceStreamWithLC is buildP25P1VoiceStream with every LDU1
// carrying the given Link Control word, as a real traffic channel does.
func buildP25P1VoiceStreamWithLC(t *testing.T, ldus int, lc phase1.LinkControl) []uint8 {
	t.Helper()
	dibits := make([]uint8, 400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	lces := phase1.AssembleLinkControl(lc)
	frame := 0
	for l := 0; l < ldus; l++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for s := range voice {
			onAir, err := imbe.EncodeFrameToChannel(p25p1VoiceInfo(frame))
			frame++
			if err != nil {
				t.Fatalf("EncodeFrameToChannel: %v", err)
			}
			voice[s] = onAir
		}
		var lsd [phase1.LDULSDBlockCount][]byte
		ldu, err := phase1.AssembleLDU(0x123, phase1.DUIDLogicalLink1, voice, lces, lsd)
		if err != nil {
			t.Fatalf("AssembleLDU: %v", err)
		}
		dibits = append(dibits, framing.BitsToDibits(ldu)...)
	}
	return dibits
}

// TestComposerP25Phase1BackfillsSourceFromLinkControl pins issue #1242's
// SRC=0 symptom: a call set up by a Group Voice Channel Update – Explicit
// (opcode 0x03) carries no source radio, so the grant arrives with
// SourceID 0. The radio that is actually keyed is named in the voice
// channel's LDU1 Link Control (LCO 0, Group Voice Channel User), and the
// chain must publish it as a KindCallSourceUpdate so the engine, call log
// and recorder pick it up — as the DMR and P25 Phase 2 chains already do.
// The Phase 1 chain parsed that source (for talker-alias tagging) but
// never published it, so every call on such a site logged SRC=0.
func TestComposerP25Phase1BackfillsSourceFromLinkControl(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		wantTG     = 356
		wantSrc    = uint32(0x1A2B3C)
	)
	dibits := buildP25P1VoiceStreamWithLC(t, 12, phase1.LinkControl{
		LCFormat:    phase1.LCOGroupVoiceChannelUser,
		TalkgroupID: wantTG,
		SourceID:    wantSrc,
	})
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
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
			// SourceID 0: the explicit update carries no source unit.
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: wantTG, FrequencyHz: 451_100_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	deadline := time.After(8 * time.Second)
	updates := 0
	for {
		select {
		case <-deadline:
			if updates == 0 {
				t.Fatal("no KindCallSourceUpdate published from the LDU1 link control")
			}
			return
		case ev := <-sub.C:
			if ev.Kind != events.KindCallSourceUpdate {
				continue
			}
			u, ok := ev.Payload.(trunking.CallSourceUpdate)
			if !ok {
				continue
			}
			if u.DeviceSerial != "VOICE-1" || u.SourceID != wantSrc {
				t.Fatalf("source update = serial %q src %d, want VOICE-1 / %d",
					u.DeviceSerial, u.SourceID, wantSrc)
			}
			updates++
			// Every LDU1 repeats the same LC; the chain must publish the
			// source once, not on every LDU1. Watch a little longer for a
			// duplicate before passing.
			if updates > 1 {
				t.Fatalf("source published %d times for one unchanged talker", updates)
			}
			deadline = time.After(1500 * time.Millisecond)
		}
	}
}

// TestVoiceIQDebugSidecarCarriesDecodedSource pins the #1242 follow-up: on
// a site that sets calls up with Group Voice Channel Update – Explicit the
// grant has no source radio, and the voice_iq_debug sidecar was written
// from the grant alone, so it read "source_id": "0" while the chain logged
// the keyed radio from LDU1 link control. The sidecar must carry the radio
// the chain decoded, the same one the call log gets.
func TestVoiceIQDebugSidecarCarriesDecodedSource(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		wantTG     = 301
		wantSrc    = uint32(2710)
	)
	dibits := buildP25P1VoiceStreamWithLC(t, 12, phase1.LinkControl{
		LCFormat:    phase1.LCOGroupVoiceChannelUser,
		TalkgroupID: wantTG,
		SourceID:    wantSrc,
	})
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	dir := t.TempDir()
	src := newFakeSource()
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		VoiceIQDebug:  VoiceIQDebugConfig{Enabled: true, Dir: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25-System", Protocol: "p25",
				GroupID: wantTG, FrequencyHz: 451_500_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// Wait until the chain has decoded the LC, then end the call so the
	// capture finalises its sidecar.
	deadline := time.After(8 * time.Second)
waitSource:
	for {
		select {
		case <-deadline:
			t.Fatal("no KindCallSourceUpdate published from the LDU1 link control")
		case ev := <-sub.C:
			if u, ok := ev.Payload.(trunking.CallSourceUpdate); ok && ev.Kind == events.KindCallSourceUpdate && u.SourceID == wantSrc {
				break waitSource
			}
		}
	}
	c.Close()

	files, _ := filepath.Glob(filepath.Join(dir, "*.metadata.json"))
	if len(files) != 1 {
		t.Fatalf("sidecars = %v, want exactly one", files)
	}
	var meta *siglab.Metadata
	waitFor(t, 2*time.Second, func() bool {
		m, err := siglab.LoadMetadata(files[0])
		if err != nil || m.System["samples"] == "" {
			return false
		}
		meta = m
		return true
	})
	if got := meta.System["source_id"]; got != "2710" {
		t.Errorf("sidecar source_id = %q, want 2710 (the radio the chain decoded from link control)", got)
	}
	if got := meta.System["source_ids"]; got != "2710" {
		t.Errorf("sidecar source_ids = %q, want 2710", got)
	}
}
