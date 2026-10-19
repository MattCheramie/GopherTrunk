package composer

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p2VoicePayload builds a deterministic, unique VoiceFrameBytes-long
// AMBE+2 frame. The 7 padding bits (low bits of the last byte) are
// cleared so an encode→extract round-trip is bit-exact.
func p25p2VoicePayload(seed int) []byte {
	out := make([]byte, p25p2.VoiceFrameBytes)
	x := uint32(seed)*2654435761 + 12345
	for i := range out {
		x = x*1664525 + 1013904223
		out[i] = byte(x >> 24)
	}
	out[p25p2.VoiceFrameBytes-1] &= 0x80
	return out
}

// p25p2TestSeed is the PN44 seed the synthetic air-format streams are
// scrambled with; the test grants carry it, as a live grant carries the
// seed the control channel derives from WACN / System ID / NAC.
var p25p2TestSeed = framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x161)

// buildP25P2VoiceStream assembles n superframes in the real TDMA air format
// with voice on logical channel 0 only, preceded by a clock-settling
// lead-in. want holds every AMBE+2 payload it carries, in order.
func buildP25P2VoiceStream(n int) (dibits []uint8, want [][]byte) {
	return buildP25P2VoiceStreamSeed(n, 0)
}

// buildP25P2VoiceStreamSeed is buildP25P2VoiceStream with a payload seed
// offset, so a test can build two streams with DISTINCT AMBE+2 payloads (a
// wanted call and an adjacent-channel neighbour) and tell them apart.
func buildP25P2VoiceStreamSeed(n, seed0 int) (dibits []uint8, want [][]byte) {
	dibits, both := buildP25P2AirVoiceStream(n, [2]int{seed0, -1})
	return dibits, both[0]
}

// buildP25P2AirVoiceStream assembles n superframes in the real TDMA air
// format (p25p2.EncodeAirPacket, scrambled with p25p2TestSeed). Logical
// channel l carries a call when seeds[l] >= 0 — VOICE-4 bursts with
// payloads numbered from seeds[l], a VOICE-2 every sixth — and idle
// SACCH otherwise. want[l] holds channel l's payloads in order. Two calls
// on one carrier is the case #1273's slot filter separates.
func buildP25P2AirVoiceStream(n int, seeds [2]int) (dibits []uint8, want [2][][]byte) {
	masks := p25p2.NewAirMasks(p25p2TestSeed)
	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4) // never matches the outbound sync
	}
	next := seeds
	var burst [2]int
	for s := 0; s < n; s++ {
		for slot := 0; slot < p25p2.AirSlotsPerSuperframe; slot++ {
			l := p25p2.AirSlotLogicalChannel(slot)
			duid, voice := p25p2.DUIDSACCHScrambled, [][]byte(nil)
			if seeds[l] >= 0 {
				duid = p25p2.DUIDVoice4
				if burst[l]%6 == 5 {
					duid = p25p2.DUIDVoice2
				}
				burst[l]++
				for j := 0; j < duid.VoiceFrames(); j++ {
					p := p25p2VoicePayload(next[l])
					next[l]++
					voice = append(voice, p)
					want[l] = append(want[l], p)
				}
			}
			pkt, err := p25p2.EncodeAirPacket(slot, duid, voice, masks, uint8(s&3))
			if err != nil {
				panic(err)
			}
			dibits = append(dibits, pkt...)
		}
	}
	return dibits, want
}

// p25p2FramesPerSuperframe is one logical channel's voice frames per
// superframe: six bursts, five VOICE-4 and one VOICE-2 on average.
const p25p2FramesPerSuperframe = 5*4 + 2

// bestAlignmentMatches returns the largest number of bit-exact frame
// matches between got and want over any stream alignment — the metric
// for "did the live IQ → voice chain recover the modulated payloads".
func bestAlignmentMatches(got, want [][]byte) int {
	best := 0
	for g := -len(want); g < len(got); g++ {
		n := 0
		for i := range got {
			w := i - g
			if w < 0 || w >= len(want) {
				continue
			}
			if bytes.Equal(got[i], want[w]) {
				n++
			}
		}
		if n > best {
			best = n
		}
	}
	return best
}

// TestComposerP25Phase2VoiceChainExtractsRawFrames drives the full
// composer Phase 2 voice path — modulated H-DQPSK IQ → Phase 2 receiver
// → air framer → DUID / logical-channel routing → AMBE+2 FEC → recorder
// .raw sidecar — and confirms the recovered frames round-trip to the
// modulated payloads.
func TestComposerP25Phase2VoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 8
	)
	framesPerSuperframe := p25p2FramesPerSuperframe

	dibits, want := buildP25P2VoiceStream(superframes)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	src := newFakeSource()
	bus := events.NewBus(8)
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
				System: "P25P2Site", Protocol: "p25-phase2",
				GroupID: 42, FrequencyHz: 851_062_500, Timeslot: 1,
				P25Phase2Decode: trunking.P25Phase2Decode{Seed: p25p2TestSeed},
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= framesPerSuperframe
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != p25p2.VoiceFrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), p25p2.VoiceFrameBytes)
		}
	}

	// At least one full superframe's worth of AMBE+2 frames must
	// round-trip to its modulated payload, proving the receiver →
	// superframe decoder → ISCH routing → voice-FEC chain is wired
	// correctly end-to-end through live IQ.
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerSuperframe {
		t.Errorf("only %d/%d frames round-tripped; want at least one superframe (%d)",
			matches, len(got), framesPerSuperframe)
	}

	// The chain keeps the call alive via Engine.Touch.
	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}

// The real Motorola talker alias from issue #376 (Victorian MMR, RID
// 200062): HEADER 0x91 + two DATA 0x95 messages. SDRTrunk decodes this
// radio's alias as "CRIO 0062".
var (
	p25p2AliasHeaderMsg = []byte{0x91, 0x90, 0x11, 0x4E, 0xF0, 0x02, 0x01, 0x00, 0x06,
		0xBE, 0xE0, 0x01, 0x64, 0x03, 0x0D, 0x7E, 0x24}
	p25p2AliasData1Msg = []byte{0x95, 0x90, 0x11, 0x01, 0x04,
		0x4F, 0x6F, 0xF2, 0xFA, 0x9A, 0xC3, 0xEC, 0x34, 0x43, 0x2F, 0xA6, 0x3C}
	p25p2AliasData2Msg = []byte{0x95, 0x90, 0x11, 0x02, 0x0C,
		0x81, 0xC3, 0xC5, 0xD9, 0x6A, 0x96, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

// p25p2MACStream assembles n superframes in the air format with voice on
// logical channel 0 and, in each superframe, the MAC PDUs mac() places on
// timeslots of that channel (scrambled with p25p2TestSeed). Idle slots of
// the other channel carry a scrambled SACCH filler.
func p25p2MACStream(t *testing.T, n int, mac func(slot int) *p25p2.AirSlotContent) []uint8 {
	t.Helper()
	masks := p25p2.NewAirMasks(p25p2TestSeed)
	dibits := make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	k := 0
	for s := 0; s <= n; s++ { // one trailing superframe flushes the last bursts
		var sf [p25p2.AirSlotsPerSuperframe]p25p2.AirSlotContent
		for slot := range sf {
			if p25p2.AirSlotLogicalChannel(slot) != 0 {
				sf[slot] = p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHScrambled}
				continue
			}
			if s < n {
				if m := mac(slot); m != nil {
					sf[slot] = *m
					continue
				}
			}
			var frames [][]byte
			for j := 0; j < 4; j++ {
				frames = append(frames, p25p2VoicePayload(k))
				k++
			}
			sf[slot] = p25p2.AirSlotContent{DUID: p25p2.DUIDVoice4, Voice: frames}
		}
		d, err := p25p2.EncodeAirSuperframe(sf, masks, uint8(s&3))
		if err != nil {
			t.Fatal(err)
		}
		dibits = append(dibits, d...)
	}
	return dibits
}

// p25p2FACCH carries one MAC message in a scrambled FACCH MAC_HANGTIME PDU.
func p25p2FACCH(t *testing.T, msg []byte) *p25p2.AirSlotContent {
	t.Helper()
	pdu, err := p25p2.AirMACPDUBytes(p25p2.AirMACHangtime, 0, 18, msg)
	if err != nil {
		t.Fatal(err)
	}
	return &p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: pdu}
}

// buildP25P2AliasStream carries the real #376 alias on FACCH-S among voice
// on logical channel 0: header, then both data blocks, in every superframe.
func buildP25P2AliasStream(t *testing.T, n int) (dibits []uint8, aliasSrc uint32, aliasText string) {
	dibits = p25p2MACStream(t, n, func(slot int) *p25p2.AirSlotContent {
		switch slot {
		case 0:
			return p25p2FACCH(t, p25p2AliasHeaderMsg)
		case 4:
			return p25p2FACCH(t, p25p2AliasData1Msg)
		case 8:
			return p25p2FACCH(t, p25p2AliasData2Msg)
		}
		return nil
	})
	return dibits, 200062, "CRIO 0062"
}

// TestComposerP25Phase2TalkerAliasFires drives the full composer Phase 2
// voice path with a synthesized superframe that interleaves Voice4V
// voice with FACCH-S bursts carrying the real Motorola alias — the on-air
// shape #376's MMR field report sees — and confirms the chain publishes a
// trunking.TalkerAlias event with the decoded "CRIO 0062". This is the regression guard for the issue's core
// claim: aliases ride the voice-channel MAC dispatch, not the CC.
func TestComposerP25Phase2TalkerAliasFires(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 6
	)

	dibits, wantSrc, wantAlias := buildP25P2AliasStream(t, superframes)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	src := newFakeSource()
	bus := events.NewBus(64)
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

	// Subscribe AFTER composer construction so the composer's
	// subscriber drains its CallStart first; this subscriber only sees
	// the TalkerAlias published by the chain.
	aliasSub := bus.Subscribe()
	defer aliasSub.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P2Site", Protocol: "p25-phase2",
				GroupID: 42, FrequencyHz: 851_062_500,
				P25Phase2Decode: trunking.P25Phase2Decode{Seed: p25p2TestSeed},
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	deadline := time.After(6 * time.Second)
	for {
		select {
		case ev := <-aliasSub.C:
			if ev.Kind != events.KindTalkerAlias {
				continue
			}
			ta, ok := ev.Payload.(trunking.TalkerAlias)
			if !ok {
				t.Fatalf("KindTalkerAlias payload type = %T, want trunking.TalkerAlias", ev.Payload)
			}
			if ta.SourceID != wantSrc {
				t.Errorf("TalkerAlias.SourceID = %#x, want %#x", ta.SourceID, wantSrc)
			}
			if ta.Alias != wantAlias {
				t.Errorf("TalkerAlias.Alias = %q, want %q", ta.Alias, wantAlias)
			}
			if ta.Protocol != "p25-phase2" {
				t.Errorf("TalkerAlias.Protocol = %q, want p25-phase2", ta.Protocol)
			}
			if ta.System != "P25P2Site" {
				t.Errorf("TalkerAlias.System = %q, want P25P2Site", ta.System)
			}
			return
		case <-deadline:
			t.Fatal("no KindTalkerAlias event published within 6s")
		}
	}
}

// buildP25P2MetadataStream carries the in-call MAC PDUs the MMR field test
// surfaced as missing from the composer dispatch, all on air-format
// signalling bursts of the call's logical channel: the key-up MAC_PTT
// (MI / ALGID / KID and the source), a SACCH GROUP_VOICE_CHANNEL_USER, and
// the real Motorola alias on FACCH-S.
func buildP25P2MetadataStream(t *testing.T, n int) (dibits []uint8, wantSrc uint32, wantAlias string, wantAlgID uint8, wantKeyID uint16) {
	t.Helper()
	wantSrc = 200062 // the alias's own RID, so every event names one radio
	wantAlias = "CRIO 0062"
	wantAlgID = 0x84 // AES-256
	wantKeyID = 0x1234
	ptt := p25p2.AirMACPTTBytes(p25p2.MACPTT{
		Encryption: p25p2.EncryptionSync{AlgorithmID: wantAlgID, KeyID: wantKeyID,
			MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		Source: wantSrc, Group: 0x4EEA,
	})
	gvcu := []byte{0x01, 0x40, 0x4E, 0xEA, byte(wantSrc >> 16), byte(wantSrc >> 8), byte(wantSrc)}
	sacch, err := p25p2.AirMACPDUBytes(p25p2.AirMACActive, 0, 21, gvcu)
	if err != nil {
		t.Fatal(err)
	}
	dibits = p25p2MACStream(t, n, func(slot int) *p25p2.AirSlotContent {
		switch slot {
		case 0:
			return &p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: ptt}
		case 2:
			return &p25p2.AirSlotContent{DUID: p25p2.DUIDSACCHScrambled, MAC: sacch}
		case 4:
			return p25p2FACCH(t, p25p2AliasHeaderMsg)
		case 6:
			return p25p2FACCH(t, p25p2AliasData1Msg)
		case 8:
			return p25p2FACCH(t, p25p2AliasData2Msg)
		}
		return nil
	})
	return dibits, wantSrc, wantAlias, wantAlgID, wantKeyID
}

// TestComposerP25Phase2InCallMetadataFires drives the in-call MAC PDUs
// the composer dispatches (MAC_PTT, Group Voice Channel User, Motorola
// talker alias) through the full Phase 2 voice chain
// and asserts the corresponding bus events fire. This is the
// regression guard for #376's MMR follow-up: source RID +
// encryption state + alias must all surface from traffic-channel MAC
// PDUs when the CC grant is compressed (src=0, enc=false).
func TestComposerP25Phase2InCallMetadataFires(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 6
	)

	dibits, wantSrc, wantAlias, wantAlgID, wantKeyID := buildP25P2MetadataStream(t, superframes)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	src := newFakeSource()
	bus := events.NewBus(128)
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

	evSub := bus.Subscribe()
	defer evSub.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "MMR", Protocol: "p25-phase2",
				GroupID: 20202, FrequencyHz: 421_387_500,
				P25Phase2Decode: trunking.P25Phase2Decode{Seed: p25p2TestSeed},
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	deadline := time.After(6 * time.Second)
	var (
		gotSource, gotEnc, gotAlias bool
	)
	for !(gotSource && gotEnc && gotAlias) {
		select {
		case ev := <-evSub.C:
			switch ev.Kind {
			case events.KindCallSourceUpdate:
				s, ok := ev.Payload.(trunking.CallSourceUpdate)
				if !ok {
					t.Fatalf("KindCallSourceUpdate payload type = %T", ev.Payload)
				}
				if s.SourceID != wantSrc {
					t.Errorf("CallSourceUpdate.SourceID = %#x, want %#x", s.SourceID, wantSrc)
				}
				if !s.Encrypted {
					t.Errorf("CallSourceUpdate.Encrypted = false, want true")
				}
				if s.DeviceSerial != "VOICE-1" {
					t.Errorf("CallSourceUpdate.DeviceSerial = %q, want VOICE-1", s.DeviceSerial)
				}
				gotSource = true
			case events.KindCallEncryption:
				e, ok := ev.Payload.(trunking.CallEncryption)
				if !ok {
					t.Fatalf("KindCallEncryption payload type = %T", ev.Payload)
				}
				if e.AlgorithmID != wantAlgID || e.KeyID != wantKeyID {
					t.Errorf("CallEncryption alg/key = 0x%X/0x%X, want 0x%X/0x%X",
						e.AlgorithmID, e.KeyID, wantAlgID, wantKeyID)
				}
				if e.DeviceSerial != "VOICE-1" {
					t.Errorf("CallEncryption.DeviceSerial = %q, want VOICE-1", e.DeviceSerial)
				}
				gotEnc = true
			case events.KindTalkerAlias:
				a, ok := ev.Payload.(trunking.TalkerAlias)
				if !ok {
					t.Fatalf("KindTalkerAlias payload type = %T", ev.Payload)
				}
				if a.SourceID != wantSrc || a.Alias != wantAlias {
					t.Errorf("TalkerAlias = (%#x, %q), want (%#x, %q)",
						a.SourceID, a.Alias, wantSrc, wantAlias)
				}
				gotAlias = true
			}
		case <-deadline:
			t.Fatalf("missing events after 6s: source=%v enc=%v alias=%v",
				gotSource, gotEnc, gotAlias)
		}
	}
}

// TestComposerP25Phase2DropsOutOfSetEncryptionSync pins the #924 validity
// gate: a MAC Encryption Sync whose Algorithm ID is not a registered TIA-102
// value is a bit-error mis-decode and must not be surfaced. Before the gate
// both syncs published, so the completed-call webhook carried a plausible
// algid+key that downstream couldn't tell from a real one.
func TestComposerP25Phase2DropsOutOfSetEncryptionSync(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := &Composer{bus: bus, log: discardLogger()}

	// Out-of-set algid (bit-error smear) — must be dropped.
	c.publishP25Phase2CallEncryption("VOICE-1", p25p2.EncryptionSync{AlgorithmID: 0x37, KeyID: 0xABCD})
	// Valid AES-256 — must publish.
	c.publishP25Phase2CallEncryption("VOICE-1", p25p2.EncryptionSync{AlgorithmID: 0x84, KeyID: 0x1234})

	var got []trunking.CallEncryption
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCallEncryption {
				continue
			}
			ce, ok := ev.Payload.(trunking.CallEncryption)
			if !ok {
				t.Fatalf("KindCallEncryption payload type = %T", ev.Payload)
			}
			got = append(got, ce)
		case <-deadline:
			if len(got) != 1 {
				t.Fatalf("published %d CallEncryption events, want 1 — the out-of-set algid must be dropped: %+v", len(got), got)
			}
			if got[0].AlgorithmID != 0x84 || got[0].KeyID != 0x1234 {
				t.Fatalf("surfaced alg/key = 0x%X/0x%X, want 0x84/0x1234", got[0].AlgorithmID, got[0].KeyID)
			}
			return
		}
	}
}

// TestComposerP25Phase2VoiceChainWarnsWithoutSeed confirms the chain-entry
// diagnostic warns when the grant carries no scrambler seed: without the
// site identity neither the voice frames nor the scrambled SACCH/FACCH can
// be descrambled, and live decode silently yields nothing (#451).
func TestComposerP25Phase2VoiceChainWarnsWithoutSeed(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c, err := New(Options{
		Bus:           events.NewBus(8),
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  48_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Log:           logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// A closed IQ channel makes the chain log its entry diagnostics and
	// then return immediately (no IQ to process).
	iqCh := make(chan []complex64)
	close(iqCh)
	done := make(chan struct{})
	c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "TestSys",
		p25p2.MACDecodeConfig{}, 0, iqCh, 48_000, done)
	<-done

	if out := buf.String(); !strings.Contains(out, "has no scrambler seed") {
		t.Fatalf("expected missing-seed warning, got:\n%s", out)
	}
}

// TestComposerP25Phase2CallCensusAlwaysFires is the #813 regression guard:
// the end-of-call MAC census must log exactly once even when the chain
// frames zero bursts (the silent-failure case the field test hit,
// where the success-only "composer: p25p2 mac pdu" line never appears).
// A closed IQ channel drives that zero-burst path.
func TestComposerP25Phase2CallCensusAlwaysFires(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c, err := New(Options{
		Bus:           events.NewBus(8),
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  48_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Log:           logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Clean live-decode config so no entry warnings pollute the buffer; the
	// closed channel returns the chain before any IQ is processed.
	iqCh := make(chan []complex64)
	close(iqCh)
	done := make(chan struct{})
	c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "TestSys",
		p25p2.MACDecodeConfig{Seed: 12345}, 0, iqCh, 48_000, done)
	<-done

	out := buf.String()
	if !strings.Contains(out, "composer: p25p2 call census") {
		t.Fatalf("expected unconditional call-census line, got:\n%s", out)
	}
	if !strings.Contains(out, "air_bursts=0") {
		t.Fatalf("census must report air_bursts=0 for a zero-IQ call, got:\n%s", out)
	}
}

// TestComposerP25Phase2CallCensusCountsMAC confirms the census counters
// and DUID histogram are wired correctly: when real MAC signalling rides
// the call's channel, the census reports a non-zero mac_pdus count and a
// bucket for the signalling DUID carried (#813).
func TestComposerP25Phase2CallCensusCountsMAC(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 8
		span       = 8
		alpha      = 0.20
	)
	dibits, _, _, _, _ := buildP25P2MetadataStream(t, 6)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	c, err := New(Options{
		Bus:           events.NewBus(128),
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Log:           logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Buffer + close so the synchronous chain drains the IQ then exits via
	// the closed-channel census path.
	iqCh := make(chan []complex64, 1)
	iqCh <- iq
	close(iqCh)
	done := make(chan struct{})
	c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "MMR",
		p25p2.MACDecodeConfig{Seed: p25p2TestSeed}, 0, iqCh, sampleRate, done)
	<-done

	out := buf.String()
	if !strings.Contains(out, "composer: p25p2 call census") {
		t.Fatalf("expected call-census line, got:\n%s", out)
	}
	if strings.Contains(out, "mac_pdus=0") {
		t.Fatalf("census reported mac_pdus=0 despite MAC signalling in the stream, got:\n%s", out)
	}
	if !strings.Contains(out, "duid_"+p25p2.DUIDFACCHScrambled.String()+"=") {
		t.Fatalf("census missing duid_%s bucket, got:\n%s", p25p2.DUIDFACCHScrambled, out)
	}
}

// TestP25P2VoiceFrontEndRejectsAdjacentChannel pins the channel selectivity of
// the wideband Phase 2 voice tap's front end: an adjacent P25 channel centred
// at ±12.5 kHz must be strongly rejected. The old pass-through front end
// (decim==1) applied NO filter, so a neighbour reached the linear H-DQPSK
// receiver at full amplitude, pumping its AGC and degrading carrier recovery.
// Fails first: pass-through leaves the neighbour at ~0 dB.
func TestP25P2VoiceFrontEndRejectsAdjacentChannel(t *testing.T) {
	const (
		fs        = float64(p25p2VoiceIntermediateHz) // 48 kHz wideband tap output → decim==1
		bw        = uint32(12_500)
		adjOffset = 12_500.0
		nSamples  = 24_000
	)
	respAt := func(offsetHz float64) float64 {
		fe := newP25P2VoiceFrontEnd(fs, bw)
		in := make([]complex64, nSamples)
		for i := range in {
			th := 2 * math.Pi * offsetHz * float64(i) / fs
			in[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
		}
		out := fe.Process(nil, in)
		var peak float64
		for _, s := range out[len(out)/2:] {
			if m := math.Hypot(float64(real(s)), float64(imag(s))); m > peak {
				peak = m
			}
		}
		return peak
	}
	wanted := respAt(0)
	neighbour := respAt(adjOffset)
	if wanted < 0.5 {
		t.Fatalf("wanted channel (DC) attenuated to %.3f; front end should pass it flat", wanted)
	}
	rejectionDB := 20 * math.Log10(neighbour/wanted)
	if rejectionDB > -40 {
		t.Errorf("adjacent channel (+%.0f Hz) rejection = %.1f dB; want <= -40 dB "+
			"(a ±12.5 kHz neighbour leaks into the Phase 2 voice tap and degrades the demod)",
			adjOffset, rejectionDB)
	}
}

// TestComposerP25Phase2VoiceChainSurvivesAdjacentChannel is the end-to-end
// regression: a wanted Phase 2 call at DC with a STRONGER adjacent call at
// +12.5 kHz present. Without a channel-select filter the neighbour corrupts the
// linear demod (AGC pumping, carrier-recovery degradation, raised EVM) and the
// wanted call's frames are lost. With the filter the neighbour is rejected and
// the wanted call decodes. Fails first. (Unlike the FM Phase 1 path the linear
// demod does not cleanly "capture" the neighbour, so we assert only that the
// wanted call survives, not that the neighbour decodes.)
func TestComposerP25Phase2VoiceChainSurvivesAdjacentChannel(t *testing.T) {
	const (
		sampleRate  = float64(p25p2VoiceIntermediateHz) // 48 kHz: wideband tap, decim==1
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 8
		adjOffset   = 12_500.0
	)
	framesPerSuperframe := p25p2FramesPerSuperframe

	dibitsW, wantW := buildP25P2VoiceStreamSeed(superframes, 0)
	iq := demod.ModulateHDQPSKSpec(dibitsW, sps, span, alpha)

	dibitsN, _ := buildP25P2VoiceStreamSeed(superframes, 500_000)
	iqN := demod.ModulateHDQPSKSpec(dibitsN, sps, span, alpha)
	for i := range iq {
		if i >= len(iqN) {
			break
		}
		th := 2 * math.Pi * adjOffset * float64(i) / sampleRate
		rot := complex(float32(math.Cos(th)), float32(math.Sin(th)))
		iq[i] += 2 * iqN[i] * rot
	}

	src := newFakeSource()
	bus := events.NewBus(8)
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
				System: "P25P2Site", Protocol: "p25-phase2",
				GroupID: 42, FrequencyHz: 851_062_500, Timeslot: 1,
				P25Phase2Decode: trunking.P25Phase2Decode{Seed: p25p2TestSeed},
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= framesPerSuperframe
	})
	got := sink.rawFrames("VOICE-1")

	matchW := bestAlignmentMatches(got, wantW)
	if matchW < framesPerSuperframe {
		t.Errorf("wanted call recovered only %d frames with a strong +%.0f Hz neighbour present; "+
			"want >= %d (adjacent channel leaked into the tap and degraded the demod)",
			matchW, adjOffset, framesPerSuperframe)
	}
}

// runP25P2Grant drives one Phase 2 grant on a modulated stream through the
// composer and returns the recorded raw frames once at least min arrive
// (or the wait expires).
func runP25P2Grant(t *testing.T, dibits []uint8, timeslot uint8, min int) [][]byte {
	t.Helper()
	const sampleRate = 48_000.0
	iq := demod.ModulateHDQPSKSpec(dibits, 8, 8, 0.20)
	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
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
				System: "P25P2Site", Protocol: "p25-phase2",
				GroupID: 42, FrequencyHz: 851_062_500, Timeslot: timeslot,
				P25Phase2Decode: trunking.P25Phase2Decode{Seed: p25p2TestSeed},
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && len(sink.rawFrames("VOICE-1")) < min {
		time.Sleep(20 * time.Millisecond)
	}
	// Let the tail of the stream drain so a wrongly recorded second
	// channel would show up too.
	time.Sleep(300 * time.Millisecond)
	return sink.rawFrames("VOICE-1")
}

// TestComposerP25Phase2RecordsOnlyGrantedTimeslot is the #1273 regression:
// a Phase 2 carrier holds two calls, one per TDMA logical channel, and the
// voice chain recorded every voice burst on the carrier into the one call
// that had been granted. Each grant must record its own channel only.
func TestComposerP25Phase2RecordsOnlyGrantedTimeslot(t *testing.T) {
	const superframes = 8
	dibits, want := buildP25P2AirVoiceStream(superframes, [2]int{0, 900_000})
	min := 2 * p25p2FramesPerSuperframe
	for _, tc := range []struct {
		timeslot   uint8
		own, other int
	}{
		{timeslot: 1, own: 0, other: 1},
		{timeslot: 2, own: 1, other: 0},
	} {
		got := runP25P2Grant(t, dibits, tc.timeslot, min)
		if n := bestAlignmentMatches(got, want[tc.own]); n < min {
			t.Errorf("timeslot %d: recovered %d of its own call's frames, want >= %d", tc.timeslot, n, min)
		}
		if n := bestAlignmentMatches(got, want[tc.other]); n != 0 {
			t.Errorf("timeslot %d: recorded %d frames of the other logical channel's call", tc.timeslot, n)
		}
	}
}

// A grant that names no timeslot binds to the first logical channel that
// carries voice instead of recording both.
func TestComposerP25Phase2UnknownTimeslotBindsOneChannel(t *testing.T) {
	dibits, want := buildP25P2AirVoiceStream(8, [2]int{0, 900_000})
	got := runP25P2Grant(t, dibits, 0, 2*p25p2FramesPerSuperframe)
	a, b := bestAlignmentMatches(got, want[0]), bestAlignmentMatches(got, want[1])
	if a == 0 && b == 0 {
		t.Fatal("no voice recorded")
	}
	if a != 0 && b != 0 {
		t.Errorf("recorded both logical channels (%d and %d frames)", a, b)
	}
}
