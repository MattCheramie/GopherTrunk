package afsk

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/narrowband"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// aprsDeviationHz is the FM peak deviation of the synthetic packets:
// ±3 kHz, a narrowband FM voice radio's data deviation. The modulator's
// default (0.5 rad/sample) would be ±191 kHz at an SDR rate.
const aprsDeviationHz = 3_000

// aprsTestIQ modulates one APRS position packet (16 preamble flags, 8
// trailing flags so the closing flag clears the receiver's filter and
// timing-loop group delay) as Bell-202 AFSK FM at rateHz.
func aprsTestIQ(rateHz float64) []complex64 {
	info := []byte("!4903.50N/07201.75W-Test")
	body := buildAX25Body("APRS", 0, "W1AW", 9, info)
	bits := wrapHDLCBits(body, 16)
	for i := 0; i < 8; i++ { // trailing flags
		for b := 0; b < 8; b++ {
			bits = append(bits, (0x7E>>b)&1)
		}
	}
	return demod.ModulateFFSKDeviation(nrziEncode(bits), rateHz, BaudHz, MarkHz, SpaceHz, aprsDeviationHz)
}

// addNoise adds complex white Gaussian noise of per-component sigma.
func addNoise(iq []complex64, sigma float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, len(iq))
	for i, x := range iq {
		out[i] = x + complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
	return out
}

// countPackets runs iq through r and returns how many FCS-valid packets
// from W1AW-9 reach the bus.
func countPackets(t *testing.T, r *Receiver, sub *events.Subscription, iq []complex64) int {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 4)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	// Live-sized chunks (an RTL-SDR delivers 16 KiB = 8192 samples).
	for off := 0; off < len(iq); off += 8192 {
		end := off + 8192
		if end > len(iq) {
			end = len(iq)
		}
		in <- iq[off:end]
	}
	close(in)
	<-done
	n := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindAPRSPacket {
				continue
			}
			if pkt, ok := ev.Payload.(storage.APRSPacket); ok && pkt.FCSOK && pkt.Src == "W1AW-9" {
				n++
			}
		case <-time.After(200 * time.Millisecond):
			return n
		}
	}
}

// TestReceiverDecodesAtSDRRateThroughFrontEnd is the #1316 regression.
// The packet is modulated at the RTL-SDR's 2.4 MS/s with white noise
// across the whole 2.4 MHz at a level that leaves the 16 kHz channel at
// about +15 dB SNR (the same signal is ~−7 dB against the full-band
// noise). The receiver fed the raw SDR stream — the daemon's wiring
// before this fix — decodes nothing; behind the narrowband front end it
// decodes the packet.
func TestReceiverDecodesAtSDRRateThroughFrontEnd(t *testing.T) {
	const rate = 2_400_000.0
	clean := aprsTestIQ(rate)
	// Noise power per complex sample = 2·sigma². In-channel fraction of
	// that is 16 kHz / 2.4 MHz. For a unit-amplitude carrier, an
	// in-channel SNR of 15 dB needs 2·sigma²·(16e3/2.4e6) = 10^-1.5.
	sigma := math.Sqrt(math.Pow(10, -1.5) / 2 / (16e3 / rate))
	noisy := addNoise(clean, sigma, 1316)
	t.Logf("sigma=%.2f (full-band SNR %.1f dB, in-channel ~15 dB)", sigma, -10*math.Log10(2*sigma*sigma))

	// Raw: the receiver on the full-rate stream.
	bus, sub := newTestBus(t)
	raw, err := New(Options{InputRateHz: uint32(rate), Bus: bus})
	if err != nil {
		t.Fatal(err)
	}
	gotRaw := countPackets(t, raw, sub, noisy)

	// Front end: decimate/filter to 48 kHz, then the same receiver built
	// for that rate.
	fe := narrowband.New(rate, 0, 0)
	bus2, sub2 := newTestBus(t)
	filtered, err := New(Options{InputRateHz: fe.OutRate(), Bus: bus2})
	if err != nil {
		t.Fatal(err)
	}
	gotFE := countPackets(t, filtered, sub2, fe.Process(noisy))

	t.Logf("raw 2.4 MS/s: %d packets; through front end (%d Hz): %d packets", gotRaw, fe.OutRate(), gotFE)
	if gotFE < 1 {
		t.Errorf("front-ended receiver decoded %d packets, want ≥ 1", gotFE)
	}
	if gotRaw >= 1 {
		t.Errorf("raw full-rate receiver decoded %d packets at −7 dB full-band SNR; the regression fixture no longer fails first", gotRaw)
	}
}

// TestReceiverDecodesCleanPacketThroughFrontEnd: the no-harm half — a
// clean packet at 2.4 MS/s decodes through the front end.
func TestReceiverDecodesCleanPacketThroughFrontEnd(t *testing.T) {
	const rate = 2_400_000.0
	fe := narrowband.New(rate, 0, 0)
	bus, sub := newTestBus(t)
	r, err := New(Options{InputRateHz: fe.OutRate(), Bus: bus})
	if err != nil {
		t.Fatal(err)
	}
	if got := countPackets(t, r, sub, fe.Process(aprsTestIQ(rate))); got != 1 {
		t.Errorf("clean packet through front end: %d packets, want 1", got)
	}
}
