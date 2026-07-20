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

// buildP25P2VoiceStream assembles a dibit stream of n all-voice (4V)
// superframes preceded by a clock-settling lead-in. want holds every
// AMBE+2 payload it carries, in transmission order.
func buildP25P2VoiceStream(n int) (dibits []uint8, want [][]byte) {
	return buildP25P2VoiceStreamSeed(n, 0)
}

// buildP25P2VoiceStreamSeed is buildP25P2VoiceStream with a seed offset so a
// test can build two streams with DISTINCT AMBE+2 payloads (a wanted call and
// an adjacent-channel neighbour) and tell which one the chain decoded.
func buildP25P2VoiceStreamSeed(n, seed0 int) (dibits []uint8, want [][]byte) {
	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4) // never matches the outbound sync
	}
	frame := seed0
	for s := 0; s < n; s++ {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			payloads := make([][]byte, p25p2.Voice4VFrameCount)
			for j := range payloads {
				p := p25p2VoicePayload(frame)
				frame++
				payloads[j] = p
				want = append(want, p)
			}
			subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
		}
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	return dibits, want
}

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
// → superframe decoder → ISCH/SlotType routing → AMBE+2 FEC → recorder
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
	framesPerSuperframe := p25p2.SubframesPerSuperframe * p25p2.Voice4VFrameCount

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
				GroupID: 42, FrequencyHz: 851_062_500,
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

// buildP25P2AliasStream assembles n superframes that interleave Voice4V
// sub-frames with MAC sub-frames carrying the two talker-alias
// fragments needed to spell "UNIT-7" for source ID 0xABC123. The
// fragments occupy sub-frames 0 and 6 (one per timeslot) of every
// superframe, so even a partial chain decode catches them.
func buildP25P2AliasStream(n int) (dibits []uint8, aliasSrc uint32, aliasText string) {
	aliasSrc = 0xABC123
	aliasText = "UNIT-7"

	frag0 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: aliasSrc, BlockIndex: 0, BlockCount: 2, Data: []byte("UNIT"),
	})
	frag1 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: aliasSrc, BlockIndex: 1, BlockCount: 2, Data: []byte("-7"),
	})

	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for s := 0; s < n; s++ {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			switch i {
			case 0:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag0, p25p2.TrellisOn, p25p2.InterleaveOff)
			case 6:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag1, p25p2.TrellisOn, p25p2.InterleaveOff)
			default:
				payloads := make([][]byte, p25p2.Voice4VFrameCount)
				for j := range payloads {
					payloads[j] = p25p2VoicePayload(s*p25p2.SubframesPerSuperframe*p25p2.Voice4VFrameCount + i*p25p2.Voice4VFrameCount + j)
				}
				subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
			}
		}
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	return dibits, aliasSrc, aliasText
}

// TestComposerP25Phase2TalkerAliasFires drives the full composer Phase 2
// voice path with a synthesized superframe that interleaves Voice4V
// sub-frames with MAC sub-frames carrying talker-alias fragments —
// the on-air shape #376's MMR field report sees — and confirms the
// chain publishes a trunking.TalkerAlias event with the reassembled
// "UNIT-7" alias. This is the regression guard for the issue's core
// claim: aliases ride the voice-channel MAC dispatch, not the CC.
func TestComposerP25Phase2TalkerAliasFires(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 6
	)

	dibits, wantSrc, wantAlias := buildP25P2AliasStream(superframes)
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
				// The encoder above leaves the MAC PDUs unscrambled
				// + RS-uncovered (matching every existing Phase 2
				// fixture), so the composer's MAC decode needs the
				// same TrellisOn / ScramblerOff config.
				P25Phase2Decode: trunking.P25Phase2Decode{
					Trellis: uint8(p25p2.TrellisOn),
				},
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

// buildP25P2MetadataStream assembles n superframes carrying three
// in-call MAC PDUs the MMR field test surfaced as missing from the
// composer dispatch: GROUP_VOICE_CHANNEL_USER (source RID + svc opts),
// ENCRYPTION_SYNC (ALGID + KID), and a talker-alias fragment pair —
// interleaved with Voice4V sub-frames.
func buildP25P2MetadataStream(n int) (dibits []uint8, wantSrc uint32, wantAlias string, wantAlgID uint8, wantKeyID uint16) {
	wantSrc = 315203 // the reporter's MMR sample radio
	wantAlias = "UNIT-7"
	wantAlgID = 0x84 // AES-256
	wantKeyID = 0x1234

	// RS-valid outer parity, as a real over-the-air PDU carries: the
	// completed-call source_rid backfill only trusts an RS-verified
	// GROUP_VOICE_CHANNEL_USER (issue #915).
	userPDU := p25p2.EncodeMACPDURS(p25p2.EncodeGroupVoiceChannelUser(p25p2.GroupVoiceChannelUser{
		ServiceOptions: 0x40, // bit 6 = encrypted
		GroupAddress:   0x4EEA,
		SourceID:       wantSrc,
	}, false))
	encPDU := p25p2.EncodeEncryptionSync(p25p2.EncryptionSync{
		AlgorithmID:      wantAlgID,
		KeyID:            wantKeyID,
		MessageIndicator: [9]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
	})
	frag0 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: wantSrc, BlockIndex: 0, BlockCount: 2, Data: []byte("UNIT"),
	})
	frag1 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: wantSrc, BlockIndex: 1, BlockCount: 2, Data: []byte("-7"),
	})

	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for s := 0; s < n; s++ {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			switch i {
			case 0:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					userPDU, p25p2.TrellisOn, p25p2.InterleaveOff)
			case 3:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					encPDU, p25p2.TrellisOn, p25p2.InterleaveOff)
			case 6:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag0, p25p2.TrellisOn, p25p2.InterleaveOff)
			case 9:
				subs[i] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, uint8(i),
					frag1, p25p2.TrellisOn, p25p2.InterleaveOff)
			default:
				payloads := make([][]byte, p25p2.Voice4VFrameCount)
				for j := range payloads {
					payloads[j] = p25p2VoicePayload(s*p25p2.SubframesPerSuperframe*p25p2.Voice4VFrameCount + i*p25p2.Voice4VFrameCount + j)
				}
				subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
			}
		}
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	return dibits, wantSrc, wantAlias, wantAlgID, wantKeyID
}

// TestComposerP25Phase2InCallMetadataFires drives all three in-call
// MAC PDU transports the composer now dispatches (User Abbreviated,
// Encryption Sync, Talker Alias) through the full Phase 2 voice chain
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

	dibits, wantSrc, wantAlias, wantAlgID, wantKeyID := buildP25P2MetadataStream(superframes)
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
				P25Phase2Decode: trunking.P25Phase2Decode{
					Trellis: uint8(p25p2.TrellisOn),
				},
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

// TestComposerP25Phase2VoiceChainWarnsTrellisOff confirms the chain-entry
// diagnostic fires a warning when the grant carries TrellisOff — the
// inconsistent FEC profile (trellis off while RS/interleave/scrambler are
// on) that surfaced in a field report and silently fails live decode.
func TestComposerP25Phase2VoiceChainWarnsTrellisOff(t *testing.T) {
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
		p25p2.MACDecodeConfig{
			Trellis:    p25p2.TrellisOff,
			RS:         p25p2.RSOn,
			Interleave: p25p2.InterleaveOn,
			Scrambler:  p25p2.ScramblerProbe,
			Seed:       12345,
		}, iqCh, 48_000, done)
	<-done

	if out := buf.String(); !strings.Contains(out, "trellis is off") {
		t.Fatalf("expected trellis-off warning, got:\n%s", out)
	}
}

// TestComposerP25Phase2CallCensusAlwaysFires is the #813 regression guard:
// the end-of-call MAC census must log exactly once even when the chain
// locks zero superframes (the silent-failure case the field test hit,
// where the success-only "composer: p25p2 mac pdu" line never appears).
// A closed IQ channel drives that zero-superframe path.
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
		p25p2.MACDecodeConfig{
			Trellis:    p25p2.TrellisOn,
			RS:         p25p2.RSOn,
			Interleave: p25p2.InterleaveOn,
			Scrambler:  p25p2.ScramblerOn,
			Seed:       12345,
		}, iqCh, 48_000, done)
	<-done

	out := buf.String()
	if !strings.Contains(out, "composer: p25p2 call census") {
		t.Fatalf("expected unconditional call-census line, got:\n%s", out)
	}
	if !strings.Contains(out, "superframes=0") {
		t.Fatalf("census must report superframes=0 for a zero-IQ call, got:\n%s", out)
	}
}

// TestComposerP25Phase2CallCensusCountsMAC confirms the census counters
// and slot histogram are wired correctly: when real MAC signalling rides
// the superframes, the census reports a non-zero mac_pdus count and a
// per-slot bucket for the MAC slot type carried (#813).
func TestComposerP25Phase2CallCensusCountsMAC(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 8
		span       = 8
		alpha      = 0.20
	)
	dibits, _, _, _, _ := buildP25P2MetadataStream(6)
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

	// The fixture encodes MAC PDUs unscrambled + RS-uncovered (matching
	// every Phase 2 fixture), so decode needs the same TrellisOn /
	// ScramblerOff config. Buffer + close so the synchronous chain drains
	// the IQ then exits via the closed-channel census path.
	iqCh := make(chan []complex64, 1)
	iqCh <- iq
	close(iqCh)
	done := make(chan struct{})
	c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "MMR",
		p25p2.MACDecodeConfig{Trellis: p25p2.TrellisOn}, iqCh, sampleRate, done)
	<-done

	out := buf.String()
	if !strings.Contains(out, "composer: p25p2 call census") {
		t.Fatalf("expected call-census line, got:\n%s", out)
	}
	if strings.Contains(out, "mac_pdus=0") {
		t.Fatalf("census reported mac_pdus=0 despite MAC signalling in the stream, got:\n%s", out)
	}
	if !strings.Contains(out, "slot_"+p25p2.SlotTypeMACSignaling.String()+"=") {
		t.Fatalf("census missing slot_%s bucket, got:\n%s", p25p2.SlotTypeMACSignaling, out)
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
	framesPerSuperframe := p25p2.SubframesPerSuperframe * p25p2.Voice4VFrameCount

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
				GroupID: 42, FrequencyHz: 851_062_500,
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
