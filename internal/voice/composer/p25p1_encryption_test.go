package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// buildP25P1EncryptedLDU2Stream emits a clock-settling lead-in followed
// by `ldus` LDU2 frames, each carrying a fresh IMBE voice subframe set
// and the same Encryption Sync (Algorithm ID + Key ID). Used to drive
// the composer's encryption-sync publish path.
func buildP25P1EncryptedLDU2Stream(t *testing.T, ldus int, es phase1.EncryptionSync) []uint8 {
	t.Helper()
	dibits := make([]uint8, 400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
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
		lces := phase1.AssembleEncryptionSync(es)
		var lsd [phase1.LDULSDBlockCount][]byte
		ldu, err := phase1.AssembleLDU(0x123, phase1.DUIDLogicalLink2, voice, lces, lsd)
		if err != nil {
			t.Fatalf("AssembleLDU LDU2: %v", err)
		}
		dibits = append(dibits, framing.BitsToDibits(ldu)...)
	}
	return dibits
}

// TestComposerP25Phase1PublishesEncryptionSync wires up a Phase 1 voice
// chain, feeds it an LDU2 stream with a recognisable Encryption Sync
// (AES-256 / key 0x1234), and asserts the composer publishes a
// KindCallEncryption event with the matching ALGID/KID + device serial.
// This is the LDU2 → engine backfill path that closes issue #353 for
// Phase 1 calls (Phase 2 already gets ALGID/KID on the grant TSBK).
func TestComposerP25Phase1PublishesEncryptionSync(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 6
	)
	es := phase1.EncryptionSync{
		MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		AlgorithmID:      0x84,
		KeyID:            0x1234,
	}
	dibits := buildP25P1EncryptedLDU2Stream(t, ldus, es)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(16)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-ENC": src}},
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

	// Subscribe BEFORE the CallStart so we don't race the chain
	// goroutine spinning up.
	encSub := bus.Subscribe()
	defer encSub.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "MMR", Protocol: "p25",
				GroupID: 4321, FrequencyHz: 851_000_000,
				Encrypted: true,
			},
			DeviceSerial: "VOICE-ENC",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// Drain events until a KindCallEncryption from the composer (with
	// our device serial) lands. Other events on the bus (CallStart
	// echoes, etc.) are ignored.
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case ev, ok := <-encSub.C:
			if !ok {
				t.Fatal("bus closed before encryption sync arrived")
			}
			if ev.Kind != events.KindCallEncryption {
				continue
			}
			ce, ok := ev.Payload.(trunking.CallEncryption)
			if !ok {
				t.Fatalf("CallEncryption payload type = %T", ev.Payload)
			}
			if ce.DeviceSerial != "VOICE-ENC" {
				continue
			}
			if ce.AlgorithmID != es.AlgorithmID || ce.KeyID != es.KeyID {
				t.Errorf("CallEncryption alg/key = 0x%X/0x%X, want 0x%X/0x%X",
					ce.AlgorithmID, ce.KeyID, es.AlgorithmID, es.KeyID)
			}
			// MessageIndicator must round-trip too — Phase 2 OFB
			// resync relies on it eventually.
			if ce.MessageIndicator != es.MessageIndicator {
				t.Errorf("CallEncryption MI = %v, want %v",
					ce.MessageIndicator, es.MessageIndicator)
			}
			return
		case <-deadline.C:
			t.Fatal("never received KindCallEncryption from composer")
		}
	}
}
