package composer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/cryptocap"
)

type memCryptoSink struct {
	mu     sync.Mutex
	frames []cryptocap.Frame
}

func (m *memCryptoSink) WriteCryptoFrame(f cryptocap.Frame) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frames = append(m.frames, f)
}

func (m *memCryptoSink) snapshot() []cryptocap.Frame {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]cryptocap.Frame(nil), m.frames...)
}

// TestComposerP25Phase1CryptoCapture drives the Phase 1 voice chain with an
// encrypted LDU2 stream and asserts the crypto-frame sink receives frames
// carrying the LDU2's Message Indicator (IV), ALGID/KID, and a non-empty
// ciphertext (the encrypted voice frames). This is the Tier 2 decoder→cryptolab
// bridge feeding `cryptolab ks reuse/mtp`.
func TestComposerP25Phase1CryptoCapture(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 6
	)
	es := phase1.EncryptionSync{
		MessageIndicator: [9]byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
		AlgorithmID:      0x81, // DES-OFB — an additive stream cipher
		KeyID:            0x2222,
	}
	dibits := buildP25P1EncryptedLDU2Stream(t, ldus, es)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(16)
	sink := &recordingSink{}
	cap := &memCryptoSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-ENC": src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		CryptoSink:    cap,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "MMR", Protocol: "p25",
				GroupID: 4321, FrequencyHz: 851_000_000, Encrypted: true,
			},
			DeviceSerial: "VOICE-ENC",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 8*time.Second, func() bool { return len(cap.snapshot()) >= 1 })

	frames := cap.snapshot()
	f := frames[0]
	if f.Protocol != "p25-phase1" {
		t.Errorf("protocol = %q, want p25-phase1", f.Protocol)
	}
	if f.AlgID != es.AlgorithmID || f.KeyID != es.KeyID {
		t.Errorf("alg/key = 0x%X/0x%X, want 0x%X/0x%X", f.AlgID, f.KeyID, es.AlgorithmID, es.KeyID)
	}
	if string(f.MI) != string(es.MessageIndicator[:]) {
		t.Errorf("MI = %v, want %v", f.MI, es.MessageIndicator)
	}
	if len(f.CT) == 0 {
		t.Error("ciphertext is empty; expected concatenated encrypted voice frames")
	}
	if f.System != "MMR" {
		t.Errorf("system = %q, want MMR", f.System)
	}
}
