package composer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// amToneIQ is 2.4 MS/s IQ of carrierAmp·(1 + m·cos(2π·1 kHz·t)), plus, when
// neighbour is set, an equal-strength AM carrier 8.33 kHz up (an air-band
// neighbour) modulated with a 2.2 kHz tone that must not reach the audio.
func amToneIQ(n int, carrierAmp float64, neighbour bool) []complex64 {
	const rate = 2_400_000.0
	out := make([]complex64, n)
	for i := range out {
		t := float64(i) / rate
		want := carrierAmp * (1 + 0.6*math.Cos(2*math.Pi*1000*t))
		nb := 0.0
		if neighbour {
			nb = carrierAmp * (1 + 0.6*math.Cos(2*math.Pi*2200*t))
		}
		ph := 2 * math.Pi * 8330 * t
		out[i] = complex(float32(want+nb*math.Cos(ph)), float32(nb*math.Sin(ph)))
	}
	return out
}

// toneAmp is the amplitude of the hz component of pcm at 8 kHz.
func toneAmp(pcm []int16, hz float64) float64 {
	var re, im float64
	for i, v := range pcm {
		ph := 2 * math.Pi * hz * float64(i) / 8000
		re += float64(v) * math.Cos(ph)
		im += float64(v) * math.Sin(ph)
	}
	return 2 * math.Hypot(re, im) / float64(len(pcm))
}

// recordAnalog runs one conventional call with protocol proto over iq and
// returns the recorded PCM (the first 0.25 s dropped: filter settle).
func recordAnalog(t *testing.T, proto string, iq []complex64) []int16 {
	t.Helper()
	return recordAnalogOpts(t, proto, iq, nil)
}

// recordAnalogOpts is recordAnalog with a hook to adjust the composer options.
func recordAnalogOpts(t *testing.T, proto string, iq []complex64, mod func(*Options)) []int16 {
	t.Helper()
	bus := events.NewBus(8)
	defer bus.Close()
	src := newFakeSource()
	sink := &recordingSink{}
	opts := Options{
		Bus: bus, Devices: &fakeDevices{src: map[string]IQSource{"CONV": src}},
		Sink: sink, Engine: &fakeEngine{},
		IQSampleRate: 2_400_000, PCMSampleRate: 8000, TouchInterval: 30 * time.Millisecond,
		// Uniformly-timed PCM so a coherent tone measurement is meaningful
		// (the default naive decimation restarts its phase every chunk).
		AudioResampler: AudioResamplerConfig{Enabled: true},
	}
	if mod != nil {
		mod(&opts)
	}
	c, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); c.Close() }()
	go c.Run(ctx)
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant:        trunking.Grant{System: "scanner", Protocol: proto, GroupID: 0x80000000, FrequencyHz: 118_700_000},
		DeviceSerial: "CONV", StartedAt: time.Now().UTC(),
	}})
	var ch chan []complex64
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		if len(src.chs) > 0 {
			ch = src.chs[0]
		}
		return ch != nil
	})
	for i := 0; i < len(iq); i += 4096 {
		ch <- iq[i:min(i+4096, len(iq))]
	}
	want := len(iq) / 300 * 9 / 10 // 2.4 MS/s → 8 kHz, minus pipeline latency
	// A second of IQ per second of wait on top of the base 3 s, so a
	// multi-second capture replay (am_capture_replay_test.go) fits too.
	waitFor(t, 3*time.Second+time.Duration(len(iq))*time.Second/2_400_000, func() bool { return sink.total("CONV") >= want })
	pcm := sink.pcmCopy("CONV")
	return pcm[2000:]
}

// TestComposerAMChainRecoversAirBandAudio is the #1219 voice pin: an
// "am-conv" call records the transmitter's modulation — the 1 kHz tone at a
// level set by its modulation depth, independent of the carrier's strength —
// with the 8.33 kHz neighbour's 2.2 kHz tone rejected. The same IQ through
// the FM discriminator is (correctly) silence: an AM carrier has no
// frequency modulation, so before this change the air band recorded
// nothing even where a channel could be tuned.
func TestComposerAMChainRecoversAirBandAudio(t *testing.T) {
	const n = 2_400_000 // 1 s
	var levels []float64
	for _, amp := range []float64{0.002, 0.2} {
		pcm := recordAnalog(t, "am-conv", amToneIQ(n, amp, true))
		want, neighbour := toneAmp(pcm, 1000), toneAmp(pcm, 2200)
		t.Logf("carrier %g: 1 kHz tone %.0f, neighbour's 2.2 kHz tone %.1f", amp, want, neighbour)
		// 0.6 depth × amAudioScale × the chain's ×10 000 PCM scale.
		if want < 0.6*amAudioScale*10_000*0.8 || want > 0.6*amAudioScale*10_000*1.2 {
			t.Errorf("carrier %g: 1 kHz tone at %.0f, want ~%.0f", amp, want, 0.6*amAudioScale*10_000)
		}
		if neighbour > want/100 {
			t.Errorf("carrier %g: neighbour's 2.2 kHz tone at %.0f (%.1f dB under the wanted tone), want > 40 dB under",
				amp, neighbour, 20*math.Log10(want/neighbour))
		}
		levels = append(levels, want)
	}
	if r := levels[1] / levels[0]; r < 0.9 || r > 1.1 {
		t.Errorf("tone level changed %.2fx for a 40 dB stronger carrier, want the same level (carrier-referenced)", r)
	}

	// Without the neighbour (two carriers beat in a discriminator, and
	// the beat carries their envelopes): a lone AM carrier has no FM.
	fm := recordAnalog(t, "fm-conv", amToneIQ(n, 0.2, false))
	if got := toneAmp(fm, 1000); got > levels[1]/10 {
		t.Errorf("fixture: the FM chain recovered the AM tone at %.0f — the test would not distinguish the chains", got)
	}
}

// mistunedAMIQ is 2.4 MS/s IQ of an AM carrier offHz from the tuned
// frequency, modulated 30 % each by a 1 kHz and a 2 kHz tone.
func mistunedAMIQ(n int, offHz float64) []complex64 {
	const rate = 2_400_000.0
	out := make([]complex64, n)
	for i := range out {
		t := float64(i) / rate
		env := 0.1 * (1 + 0.3*math.Cos(2*math.Pi*1000*t) + 0.3*math.Cos(2*math.Pi*2000*t))
		ph := 2 * math.Pi * offHz * t
		out[i] = complex(float32(env*math.Cos(ph)), float32(env*math.Sin(ph)))
	}
	return out
}

// TestComposerAMChainTracksMistunedCarrier pins the #1219 on-air finding:
// the reporter's captures (tuned 123.453 MHz) carry the carrier at −3315 Hz,
// which leaves the far sideband outside the ±4.5 kHz channel filter centred
// on the TUNED frequency — the 2 kHz tone then records at roughly half its
// level. The chain must find the carrier and centre the filter on it, so a
// few kHz of tuning / ppm error records the same audio as a centred carrier.
func TestComposerAMChainTracksMistunedCarrier(t *testing.T) {
	const n = 2_400_000 // 1 s
	want := 0.3 * amAudioScale * 10_000
	for _, off := range []float64{0, -3315, 3315} {
		pcm := recordAnalog(t, "am-conv", mistunedAMIQ(n, off))
		lo, hi := toneAmp(pcm, 1000), toneAmp(pcm, 2000)
		t.Logf("carrier %+.0f Hz: 1 kHz tone %.0f, 2 kHz tone %.0f (want ~%.0f each)", off, lo, hi, want)
		for _, got := range []float64{lo, hi} {
			if got < want*0.8 || got > want*1.2 {
				t.Errorf("carrier %+.0f Hz off the tuned frequency: tone at %.0f, want ~%.0f (a sideband is being cut)", off, got, want)
			}
		}
	}
}

func TestAMCarrierAFCHoldsOnNoise(t *testing.T) {
	a := newAMCarrierAFC(48_000)
	x := make([]complex64, 48_000)
	s := uint64(1)
	for i := range x {
		s = s*6364136223846793005 + 1442695040888963407
		re := float32(int32(s>>32)) / (1 << 31)
		s = s*6364136223846793005 + 1442695040888963407
		im := float32(int32(s>>32)) / (1 << 31)
		x[i] = complex(re, im)
	}
	a.Process(nil, x)
	if off, locked := a.OffsetHz(); locked {
		t.Fatalf("AFC locked to %+.0f Hz on noise alone, want it to hold (no carrier line)", off)
	}
}

func TestClassifyAMConv(t *testing.T) {
	if k := classifyVoiceKind(trunking.CallStart{Grant: trunking.Grant{Protocol: "am-conv"}}); k != voiceKindAM {
		t.Fatalf("am-conv classified %v, want voiceKindAM", k)
	}
}
