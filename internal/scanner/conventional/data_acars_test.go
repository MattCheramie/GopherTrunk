package conventional

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/acars"
	acarsrx "github.com/MattCheramie/GopherTrunk/internal/radio/acars/receiver"
)

// An ACARS block runs up to ~0.86 s, longer than the 500 ms FFSK hold, so a
// channel with the acars decoder holds its scan window up to
// acarsScanHoldMax (#1231).
func TestACARSDecoderHoldsScanWindowForALongBlock(t *testing.T) {
	dec := &fakeDataDecoder{busy: func(int) bool { return true }}
	s := newDataTestScanner(t, Channel{
		Label: "ACARS", FrequencyHz: 131_550_000, Mode: ModeAM, Decoders: []string{DecoderACARS},
	}, dec, nil)
	stream := make(chan []complex64) // never delivers: the window can only time out
	start := time.Now()
	if s.scanWindow(context.Background(), 0, s.channels[0], stream, nil) {
		t.Fatal("squelch broke with no IQ")
	}
	held := time.Since(start)
	if held < 30*time.Millisecond+acarsScanHoldMax-dataScanHoldStep {
		t.Fatalf("busy ACARS decoder: window took %v, want it held ~%v", held, acarsScanHoldMax)
	}
	if held > 30*time.Millisecond+acarsScanHoldMax+time.Second {
		t.Fatalf("busy ACARS decoder: window took %v — the hold must stay bounded", held)
	}
}

// TestScannerDecodesACARSOnAMChannel runs a synthetic ACARS transmission
// through the whole scanner at the SDR rate: the AM C/N squelch opens on
// the carrier, the data front end decimates to the channel rate, and the
// production ACARS receiver built by the factory frames the block.
func TestScannerDecodesACARSOnAMChannel(t *testing.T) {
	const (
		freq = 131_550_000
		rate = 2_400_000
	)
	block := acars.Block{Mode: '2', Address: "N123GT", Ack: 0x15, Label: "H1", BlockID: '5',
		Text: "M01AGT0042HELLO FROM THE SCANNER"}
	// Keyed carrier, the block, trailing carrier, with noise under it.
	lead := make([]complex64, rate/5)
	for i := range lead {
		lead[i] = 1
	}
	x := append(lead, acarsrx.SynthAMIQ(acars.EncodeBlock(block), rate, 0, 0.6)...)
	x = append(x, lead...)
	rng := rand.New(rand.NewSource(1))
	for i := range x {
		x[i] = x[i]*0.1 + complex64(complex(0.01*rng.NormFloat64(), 0.01*rng.NormFloat64()))
	}
	var chunks [][]complex64
	for i := 0; i < len(x); i += 16384 {
		chunks = append(chunks, x[i:min(i+16384, len(x))])
	}
	tuner := &fakeTuner{}
	var mu sync.Mutex
	var got []acars.Message
	s, err := New(Options{
		Tuner: tuner, IQ: &fakeIQ{chunks: map[uint32][][]complex64{freq: chunks}, tuner: tuner},
		Engine: &fakeEngine{}, Recorder: fakeRecorder{}, DeviceSerial: "CONV-ACARS",
		Channels: []Channel{{
			Label: "ACARS", FrequencyHz: freq, Mode: ModeAM,
			Hangtime: 5 * time.Second, Decoders: []string{DecoderACARS},
		}},
		SampleRateHz: rate,
		DataDecoders: func(ch Channel, kind string, rateHz float64) (DataDecoder, error) {
			return acarsrx.New(acarsrx.Options{InputRateHz: uint32(rateHz), OnMessage: func(m acars.Message) {
				mu.Lock()
				got = append(got, m)
				mu.Unlock()
			}})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		mu.Lock()
		for _, m := range got {
			if m.CRCOK && m.Address == "N123GT" && m.FlightID == "GT0042" {
				mu.Unlock()
				return
			}
		}
		mu.Unlock()
	}
	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("scanner framed %d blocks (%+v), want the N123GT block CRC-valid", len(got), got)
}
