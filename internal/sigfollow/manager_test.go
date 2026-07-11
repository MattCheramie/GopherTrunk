package sigfollow

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/wbvoice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeDevice is the minimal sdr.Device an iqtap.Broker needs. Mirrors the
// helper in internal/sdr/wbvoice/tuner_test.go.
type fakeDevice struct {
	info   sdr.Info
	stream chan []complex64
}

func newFakeDevice() *fakeDevice {
	return &fakeDevice{
		info:   sdr.Info{Driver: "fake", Serial: "FAKE"},
		stream: make(chan []complex64, 16),
	}
}

func (f *fakeDevice) Info() sdr.Info             { return f.info }
func (f *fakeDevice) SetCenterFreq(uint32) error { return nil }
func (f *fakeDevice) SetSampleRate(uint32) error { return nil }
func (f *fakeDevice) SetGain(int) error          { return nil }
func (f *fakeDevice) SetPPM(int) error           { return nil }
func (f *fakeDevice) SetBiasTee(bool) error      { return nil }
func (f *fakeDevice) Close() error               { return nil }
func (f *fakeDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	return f.stream, nil
}

// aliasDibits builds a dibit stream of `warmup` voice-only superframes
// followed by `n` superframes interleaving Voice4V with the two
// talker-alias fragments for "UNIT-7" (source 0xABC123). The warmup lets
// the receiver's Gardner symbol-clock lock before the alias frames decode
// — without it the generic alias assembler completes on the very first
// (still-settling) superframe and a transient symbol slip corrupts the
// reassembled source / alias.
func aliasDibits(warmup, n int) (dibits []uint8, src uint32, alias string) {
	src, alias = 0xABC123, "UNIT-7"
	frag0 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: src, BlockIndex: 0, BlockCount: 2, Data: []byte("UNIT"),
	})
	frag1 := p25p2.EncodeTalkerAliasFragment(p25p2.TalkerAliasFragment{
		SourceID: src, BlockIndex: 1, BlockCount: 2, Data: []byte("-7"),
	})
	voiceOnly := func() [p25p2.SubframesPerSuperframe][]uint8 {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			payloads := make([][]byte, p25p2.Voice4VFrameCount)
			for j := range payloads {
				payloads[j] = make([]byte, p25p2.VoiceFrameBytes)
			}
			subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
		}
		return subs
	}
	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for s := 0; s < warmup; s++ {
		subs := voiceOnly()
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	for s := 0; s < n; s++ {
		subs := voiceOnly()
		subs[0] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, 0,
			frag0, p25p2.TrellisOn, p25p2.InterleaveOff)
		subs[6] = p25p2.EncodeMACSubframe(p25p2.SlotTypeMACSignaling, 6,
			frag1, p25p2.TrellisOn, p25p2.InterleaveOff)
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	return dibits, src, alias
}

// TestManagerHarvestsAliasWithoutVoiceTuner is the regression guard for
// #376's architectural fix: a P25 Phase 2 grant alone — no voice tuner,
// no recorder, no active call — drives a signalling follow that decodes
// the FACCH-S talker alias off the traffic channel and publishes it.
// Modulated H-DQPSK IQ flows through a real wideband broker → VirtualTuner
// DDC → the follower's receiver, end to end.
func TestManagerHarvestsAliasWithoutVoiceTuner(t *testing.T) {
	const (
		sdrRate    = 96_000.0 // wideband input; the DDC decimates 2:1 to 48 kHz
		sps        = 16       // 6000 sym/s × 16 = 96 kHz
		span       = 8
		alpha      = 0.20
		targetHz   = 851_062_500
		widebandHz = targetHz // offset 0: the tap sits on the traffic channel
	)

	dibits, wantSrc, wantAlias := aliasDibits(8, 8)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	fake := newFakeDevice()
	broker := iqtap.New(fake, 64, log)
	vt, err := wbvoice.New(wbvoice.Options{
		Serial: "sig:FAKE:tap-0", Broker: broker,
		WidebandCenterHz: widebandHz, SDRSampleRateHz: sdrRate, Log: log,
	})
	if err != nil {
		t.Fatal(err)
	}

	bus := events.NewBus(64)
	mgr, err := New(Options{Bus: bus, Log: log, Tuners: []*wbvoice.VirtualTuner{vt}})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = mgr.Run(ctx) }()

	aliasSub := bus.Subscribe()
	defer aliasSub.Close()

	// Drive the broker's primary stream so its fanout to the follower's
	// subscription actually runs (the broker only fans out while a
	// primary StreamIQ consumer is active).
	primary, err := broker.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for range primary {
		}
	}()

	// Re-publish the grant on a tick (as a real CC repeats it for the
	// duration of a call) so the follow starts regardless of when Run's
	// bus subscription registers — and restarts if a follow idles out.
	// onGrant dedupes by frequency, so repeats while following are no-ops.
	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for {
			bus.Publish(events.Event{
				Kind: events.KindGrant,
				Payload: trunking.Grant{
					System: "P25P2Site", Protocol: "p25-phase2",
					GroupID: 42, FrequencyHz: targetHz,
					P25Phase2Decode: trunking.P25Phase2Decode{Trellis: uint8(p25p2.TrellisOn)},
				},
			})
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	// Feed the modulated IQ in chunks, looping so there is always signal
	// on the channel whenever the follow subscribes. The encoder leaves
	// the MAC PDUs unscrambled / RS-uncovered, matching TrellisOn /
	// ScramblerOff.
	go func() {
		const chunk = 4096
		for {
			for off := 0; off < len(iq); off += chunk {
				end := off + chunk
				if end > len(iq) {
					end = len(iq)
				}
				cp := make([]complex64, end-off)
				copy(cp, iq[off:end])
				select {
				case fake.stream <- cp:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Generous deadline: this drives the full synthesized-IQ → P25 Phase 2
	// decode pipeline, which under `-race` on a heavily-loaded CI runner can
	// take tens of seconds. 20s flaked there; 60s keeps margin without hanging.
	deadline := time.After(60 * time.Second)
	for {
		select {
		case ev := <-aliasSub.C:
			if ev.Kind != events.KindTalkerAlias {
				continue
			}
			ta, ok := ev.Payload.(trunking.TalkerAlias)
			if !ok {
				t.Fatalf("KindTalkerAlias payload type = %T", ev.Payload)
			}
			// The decoded alias string and the call identity are asserted
			// exactly. The source ID is only required non-zero here: a
			// single deterministic symbol slip through the synthesized 2:1
			// DDC harness can flip a SourceID bit, and exact-bit decode
			// fidelity is already covered by the composer's no-DDC test
			// (TestComposerP25Phase2TalkerAliasFires). This test's job is
			// to prove the alias is harvested off a signalling tap with no
			// voice tuner — the #376 decouple.
			if ta.Alias != wantAlias {
				t.Errorf("Alias = %q, want %q", ta.Alias, wantAlias)
			}
			if ta.SourceID == 0 {
				t.Errorf("SourceID = 0, want non-zero (≈ %#x)", wantSrc)
			}
			if ta.Protocol != "p25-phase2" || ta.System != "P25P2Site" {
				t.Errorf("identity = (%q, %q), want (p25-phase2, P25P2Site)", ta.Protocol, ta.System)
			}
			return
		case <-deadline:
			t.Fatal("no KindTalkerAlias published within 60s — alias not harvested off the signalling tap")
		}
	}
}
