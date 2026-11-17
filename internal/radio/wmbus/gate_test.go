package wmbus

import (
	"math"
	"math/rand"
	"testing"
)

// gateNoise is n samples of white noise, rms per axis.
func gateNoise(rng *rand.Rand, n int, rms float64) []complex64 {
	iq := make([]complex64, n)
	for i := range iq {
		iq[i] = complex(float32(rng.NormFloat64()*rms), float32(rng.NormFloat64()*rms))
	}
	return iq
}

// gateTelegram is one C mode telegram of random content, 50 kHz off centre
// as the issue #1256 tuner hears its meters, snrDB over the noise in the
// 200 kHz channel, with a millisecond of that noise either side.
func gateTelegram(rng *rand.Rand, snrDB float64, seed int64) (iq []complex64, data []byte, sigma float64) {
	const rate = 1_600_000
	sigma = math.Sqrt(1 / math.Pow(10, snrDB/10)) // see noisyCModeYield
	data = make([]byte, 60)
	rng.Read(data)
	data[0] = byte(len(data) - 1)
	iq = Synthesize(data, SynthOptions{
		Mode: ModeC1, SampleRateHz: rate, OffsetHz: -50_000, DeviationHz: 45_000,
		NoiseRMS: sigma, Seed: seed,
	})
	return iq, data, sigma
}

type gateFrame struct {
	data   string
	sample int64
}

// gateRun feeds iq through a receiver in RTL-sized chunks and returns the
// CRC-valid frames and the receiver's counters.
func gateRun(t *testing.T, iq []complex64, noGate bool) ([]gateFrame, ReceiverStats) {
	t.Helper()
	var frames []gateFrame
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: 1_600_000, NoBurstGate: noGate, OnFrame: func(f Frame, info FrameInfo) {
		if f.CRCOK && !info.Inverted {
			frames = append(frames, gateFrame{string(f.Data), info.Sample})
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < len(iq); j += 8192 {
		rx.Process(iq[j:min(j+8192, len(iq))])
	}
	rx.Flush()
	return frames, rx.Stats()
}

// TestBurstGateSkipsNoise: on a channel with nobody transmitting, which is
// what a live run hears nearly all the time, the demodulator stays off. It
// used to run on every sample (issue #1256: the reporter's phone dropped
// IQ whenever it slowed the process down).
func TestBurstGateSkipsNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const seconds = 5
	_, st := gateRun(t, gateNoise(rng, seconds*1_600_000, 0.05), false)
	// The first 20 ms are always demodulated, while the floor settles.
	if share := float64(st.Demodulated) / float64(st.Samples); share > 0.02 {
		t.Errorf("the demodulator ran on %.1f%% of %d s of noise, want under 2%%", 100*share, seconds)
	}
	if st.Bursts > 2*seconds {
		t.Errorf("noise opened the gate %d times in %d s", st.Bursts, seconds)
	}
	if st.Samples != seconds*1_600_000 {
		t.Errorf("Samples = %d, want every sample fed counted", st.Samples)
	}
}

// TestBurstGateDecodesWhatTheDemodulatorDoes: telegrams far apart in noise,
// at a level where most but not all decode. The gated receiver must deliver
// the same telegrams as the receiver that demodulates everything, at the
// same stream positions (the snippet saver and the drop log locate a
// telegram by its position), while demodulating a fraction of the IQ.
func TestBurstGateDecodesWhatTheDemodulatorDoes(t *testing.T) {
	rng := rand.New(rand.NewSource(1256))
	var iq []complex64
	sent := map[string]bool{}
	const telegrams = 30
	for i := 0; i < telegrams; i++ {
		tg, data, sigma := gateTelegram(rng, 8, int64(i+1))
		// 20 to 60 ms of noise between telegrams, so they start anywhere
		// in the gate's blocks and the gate closes in between.
		iq = append(iq, gateNoise(rng, 32_000+rng.Intn(64_000), sigma)...)
		iq = append(iq, tg...)
		sent[string(data)] = true
	}
	want, _ := gateRun(t, iq, true)
	got, st := gateRun(t, iq, false)
	if len(want) < telegrams*2/3 {
		t.Fatalf("the ungated receiver decoded %d of %d telegrams: the fixture is too weak to test with", len(want), telegrams)
	}
	gotAt := map[string]int64{}
	for _, f := range got {
		if !sent[f.data] {
			t.Errorf("gated: a CRC-valid frame that was never sent")
		}
		gotAt[f.data] = f.sample
	}
	lost := 0
	for _, f := range want {
		at, ok := gotAt[f.data]
		if !ok {
			lost++
			continue
		}
		// Within a chip: skipped input is counted in whole processed
		// samples.
		if d := at - f.sample; d < -16 || d > 16 {
			t.Errorf("a telegram ends at sample %d ungated and %d gated", f.sample, at)
		}
	}
	t.Logf("CRC-valid: %d ungated, %d gated, %d of the ungated ones missing; demodulated %.0f%% of the IQ, %d bursts",
		len(want), len(got), lost, 100*float64(st.Demodulated)/float64(st.Samples), st.Bursts)
	// The chip clock meets each preamble in a different state when the
	// noise before it was skipped, so a telegram on the edge can go either
	// way; anything more than that is the gate missing telegrams.
	if lost > 1 || len(got) < len(want)-1 {
		t.Errorf("gated: %d CRC-valid and %d of the ungated ones missing, ungated %d", len(got), lost, len(want))
	}
	if share := float64(st.Demodulated) / float64(st.Samples); share > 0.45 {
		t.Errorf("the demodulator ran on %.0f%% of a stream that is 16%% telegrams", 100*share)
	}
}

// TestBurstGateStaysOpenThroughAnotherTransmission: a transmitter elsewhere
// in the tuner's span (here 300 kHz up, 23 dB over the channel's noise, on
// for 0.6 s) raises the detector's power without touching the channel. The
// gate must then stay open, as the receiver ran before it, and not learn
// that level as its floor within the transmission: a weak telegram in the
// channel half a second in would be skipped.
func TestBurstGateStaysOpenThroughAnotherTransmission(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	tg, data, sigma := gateTelegram(rng, 9, 1)
	const rate = 1_600_000
	iq := gateNoise(rng, rate/2, sigma) // 0.5 s for the floor to settle
	start := len(iq)
	iq = append(iq, gateNoise(rng, rate/2, sigma)...)
	iq = append(iq, tg...)
	iq = append(iq, gateNoise(rng, rate/10-len(tg), sigma)...)
	end := len(iq)
	iq = append(iq, gateNoise(rng, rate/10, sigma)...)
	// The other transmitter: a carrier whose power is 200 times the noise
	// in 200 kHz (which is 2 sigma^2 / 8).
	amp := math.Sqrt(200 * 2 * sigma * sigma / 8)
	for i := start; i < end; i++ {
		ph := 2 * math.Pi * 300_000 / rate * float64(i)
		iq[i] += complex(float32(amp*math.Cos(ph)), float32(amp*math.Sin(ph)))
	}
	want, _ := gateRun(t, iq, true)
	if len(want) != 1 || want[0].data != string(data) {
		t.Fatalf("the ungated receiver does not decode the telegram beside the other transmitter: the fixture is wrong")
	}
	got, st := gateRun(t, iq, false)
	t.Logf("demodulated %.0f%% of the IQ, %d bursts, %d access codes", 100*float64(st.Demodulated)/float64(st.Samples), st.Bursts, st.Framer.Syncs)
	if len(got) != 1 || got[0].data != string(data) {
		t.Errorf("gated: %d CRC-valid frames, want the telegram sent 0.5 s into the other transmission (demodulated %.0f%% of the IQ)",
			len(got), 100*float64(st.Demodulated)/float64(st.Samples))
	}
}

// TestBurstGateFollowsALastingNoiseRise: the noise itself going up for good
// (a new place, a gain change) opens the gate, and within a few seconds the
// floor has followed and the demodulator is off again.
func TestBurstGateFollowsALastingNoiseRise(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	const rate = 1_600_000
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate})
	if err != nil {
		t.Fatal(err)
	}
	rx.Process(gateNoise(rng, rate, 0.02))
	before := rx.Stats().Demodulated
	rx.Process(gateNoise(rng, 8*rate, 0.04)) // +6 dB, for 8 s
	settled := rx.Stats().Demodulated
	rx.Process(gateNoise(rng, 2*rate, 0.04))
	after := rx.Stats().Demodulated
	if settled-before < rate {
		t.Errorf("a 6 dB rise in the noise demodulated only %d samples: the gate should open on it", settled-before)
	}
	if share := float64(after-settled) / float64(2*rate); share > 0.02 {
		t.Errorf("8 s after the noise rose the demodulator still runs on %.0f%% of it", 100*share)
	}
}
