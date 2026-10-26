package ppm

import (
	"context"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/adsb"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

func noiseIQ(n int, sigma float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
	return out
}

// runIQ pushes iq through a fresh receiver in live-sized chunks and
// returns its stats plus every report it published.
func runIQ(t *testing.T, iq []complex64) (Stats, []storage.AircraftReport) {
	t.Helper()
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	r, err := New(Options{InputRateHz: SampleRateHz, Bus: bus})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 1)
	done := make(chan struct{})
	go func() { _ = r.Process(ctx, in); close(done) }()
	for off := 0; off < len(iq); off += 8192 {
		end := off + 8192
		if end > len(iq) {
			end = len(iq)
		}
		in <- iq[off:end]
	}
	close(in)
	<-done
	var reps []storage.AircraftReport
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindAircraftReport {
				reps = append(reps, ev.Payload.(storage.AircraftReport))
			}
		case <-time.After(100 * time.Millisecond):
			return r.Stats(), reps
		}
	}
}

// TestReceiverPublishesNoAircraftFromNoise is the #1316 regression at the
// receiver level: half a second of receiver noise trips the preamble
// detector over a thousand times, and before the address gate each
// detection was sliced into a DF 0/4/5 frame and published as an aircraft
// with a random ICAO (the reporter's 440 reports/s). Now none is.
func TestReceiverPublishesNoAircraftFromNoise(t *testing.T) {
	st, reps := runIQ(t, noiseIQ(SampleRateHz/2, 0.3, 7))
	if st.PreamblesSeen < 100 {
		t.Fatalf("fixture too quiet: only %d preambles detected on noise", st.PreamblesSeen)
	}
	if len(reps) != 0 || st.FramesEmitted != 0 {
		t.Errorf("noise produced %d aircraft reports (%d emitted); want 0. First: %+v", len(reps), st.FramesEmitted, reps[0])
	}
	t.Logf("noise: %d preambles detected, %d reports", st.PreamblesSeen, len(reps))
}

// TestReceiverAcceptsAddressParityFrameAfterSquitter: the DF 4 altitude
// reply of an aircraft whose DF 17 squitter was heard first is reported;
// the same DF 4 alone is not.
func TestReceiverAcceptsAddressParityFrameAfterSquitter(t *testing.T) {
	df17, _ := hex.DecodeString("8D4840D6202CC371C32CE0576098") // ICAO 4840D6
	// DF 4 with altitude code bytes 00 1B 7C and the parity overlaid with
	// 4840D6 (crc24 over the first 4 bytes XOR the ICAO).
	body := []byte{0x20, 0x00, 0x1B, 0x7C}
	crc := adsb.CRC24(body) ^ 0x4840D6
	df4 := append(body, byte(crc>>16), byte(crc>>8), byte(crc))

	quiet := func(n int) []complex64 { return make([]complex64, n) }
	alone := append(quiet(1000), modulatePPM(df4)...)
	alone = append(alone, quiet(1000)...)
	if _, reps := runIQ(t, alone); len(reps) != 0 {
		t.Fatalf("lone DF4 reported as %+v", reps)
	}
	both := append(quiet(1000), modulatePPM(df17)...)
	both = append(both, quiet(1000)...)
	both = append(both, modulatePPM(df4)...)
	both = append(both, quiet(1000)...)
	_, reps := runIQ(t, both)
	if len(reps) != 2 {
		t.Fatalf("DF17 then DF4: %d reports, want 2: %+v", len(reps), reps)
	}
	if reps[1].ICAOHex != "4840D6" || !reps[1].CRCValid || reps[1].Kind != "unknown" {
		t.Errorf("DF4 report = %+v; want ICAO 4840D6, crc_valid, kind unknown", reps[1])
	}
}
