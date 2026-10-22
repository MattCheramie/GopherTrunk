package main

import (
	"math"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// pcmRecorder is a live sink that keeps what it is handed, per serial.
type pcmRecorder struct {
	mu  sync.Mutex
	pcm map[string][]int16
}

func newPCMRecorder() *pcmRecorder { return &pcmRecorder{pcm: map[string][]int16{}} }

func (r *pcmRecorder) WritePCM(serial string, samples []int16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pcm[serial] = append(r.pcm[serial], samples...)
	return nil
}

func (r *pcmRecorder) get(serial string) []int16 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int16(nil), r.pcm[serial]...)
}

// tone returns n samples of a sine at f Hz sampled at rate.
func tone(f, rate float64, n int) []int16 {
	pcm := make([]int16, n)
	for i := range pcm {
		pcm[i] = int16(8000 * math.Sin(2*math.Pi*f*float64(i)/rate))
	}
	return pcm
}

// toneHz estimates the frequency of a sine sampled at rate from its zero
// crossings, skipping the resampler's start-up transient.
func toneHz(pcm []int16, rate float64) float64 {
	skip := len(pcm) / 10
	crossings := 0
	for i := skip + 1; i < len(pcm); i++ {
		if (pcm[i-1] < 0) != (pcm[i] < 0) {
			crossings++
		}
	}
	return float64(crossings) / 2 / (float64(len(pcm)-skip-1) / rate)
}

// feedPCM writes pcm to sink in 160-sample (20 ms at 8 kHz) chunks, as the
// recorder's decoded tap does per vocoder frame.
func feedPCM(t *testing.T, sink composer.PCMSink, serial string, pcm []int16, chunk int) {
	t.Helper()
	for i := 0; i < len(pcm); i += chunk {
		if err := sink.WritePCM(serial, pcm[i:min(i+chunk, len(pcm))]); err != nil {
			t.Fatal(err)
		}
	}
}

// #1242: decoded digital voice is always 8 kHz, but the live stream runs at
// the player's rate (audio.sample_rate, else recordings.sample_rate), which
// its WAV header announces. With 48000 configured, the decoded tap handed
// the stream 8 kHz samples under a 48 kHz header, so a call played six times
// fast — a whistle. The stream must receive the call at its own rate: the
// same tone, six times the samples. Analog PCM, which the composer already
// produces at recordings.sample_rate, must pass through untouched, and the
// tone-out detector (tuned to recordings.sample_rate) must get digital PCM at
// that rate too.
func TestLiveSinksDeliverEachConsumerItsRate(t *testing.T) {
	cfg := config.Default()
	cfg.Recordings.SampleRate = 48000
	lr := liveRatesFor(cfg)
	if lr.analogHz != 48000 || lr.liveHz != 48000 {
		t.Fatalf("rates %+v, want analog 48000 live 48000", lr)
	}
	dec, toneSink, player, stream := newPCMRecorder(), newPCMRecorder(), newPCMRecorder(), newPCMRecorder()
	analog, decoded := wireLiveSinks(lr, dec, toneSink, player, stream, nil)

	// A digital call: one second of a 1 kHz tone as the vocoder emits it.
	feedPCM(t, fanoutSink(decoded), "VOICE-1", tone(1000, 8000, 8000), 160)
	got := stream.get("VOICE-1")
	if n := len(got); n < 47000 || n > 48000 {
		t.Errorf("stream got %d samples of a 1 s digital call, want ~48000 (one second at its 48 kHz header rate)", n)
	}
	if f := toneHz(got, float64(lr.liveHz)); math.Abs(f-1000) > 20 {
		t.Errorf("a 1 kHz digital tone plays at %.0f Hz on the live stream", f)
	}
	if f := toneHz(toneSink.get("VOICE-1"), float64(lr.analogHz)); math.Abs(f-1000) > 20 {
		t.Errorf("tone-out hears a 1 kHz digital tone at %.0f Hz", f)
	}
	if len(dec.get("VOICE-1")) != 0 || len(player.get("VOICE-1")) != 0 {
		t.Error("decoded PCM reached the recorder or the host player; neither is on the tap")
	}

	// An analog call at recordings.sample_rate: already the live rate.
	fm := tone(1000, 48000, 48000)
	feedPCM(t, fanoutSink(analog), "FM-1", fm, 4800)
	if got := stream.get("FM-1"); len(got) != len(fm) || got[100] != fm[100] {
		t.Errorf("analog PCM changed on its way to the stream (%d samples, want %d unchanged)", len(got), len(fm))
	}
}

// With audio.sample_rate set apart from recordings.sample_rate, analog PCM
// (at recordings.sample_rate) is converted for the host player and the
// stream, which both run at audio.sample_rate.
func TestLiveSinksConvertAnalogToAudioRate(t *testing.T) {
	cfg := config.Default()
	cfg.Recordings.SampleRate = 16000
	cfg.Audio.SampleRate = 48000
	lr := liveRatesFor(cfg)
	dec, player, stream := newPCMRecorder(), newPCMRecorder(), newPCMRecorder()
	analog, _ := wireLiveSinks(lr, dec, nil, player, stream, nil)
	feedPCM(t, fanoutSink(analog), "FM-1", tone(1000, 16000, 16000), 1600)
	for name, r := range map[string]*pcmRecorder{"player": player, "stream": stream} {
		got := r.get("FM-1")
		if f := toneHz(got, 48000); math.Abs(f-1000) > 20 || len(got) < 47000 {
			t.Errorf("%s: %d samples, tone at %.0f Hz at 48 kHz; want ~48000 samples at 1000 Hz", name, len(got), f)
		}
	}
	if got := dec.get("FM-1"); len(got) != 16000 {
		t.Errorf("the recorder got %d analog samples, want the 16000 it records at", len(got))
	}
}

// The default (8000 everywhere) wires the original sinks with no resampler
// in the way, so default installs are byte-identical to before.
func TestLiveSinksDefaultIsPassThrough(t *testing.T) {
	lr := liveRatesFor(config.Default())
	if lr.analogHz != 8000 || lr.liveHz != 8000 {
		t.Fatalf("default rates %+v, want 8000/8000", lr)
	}
	stream := newPCMRecorder()
	analog, decoded := wireLiveSinks(lr, newPCMRecorder(), nil, nil, stream, nil)
	if analog[len(analog)-1] != composer.PCMSink(stream) || decoded[0] != composer.PCMSink(stream) {
		t.Error("default wiring put a resampler in front of the stream")
	}
}

// A reused voice tap (new CallID on the same serial) starts a fresh
// resampler, so the previous call's filter tail cannot open the next call.
func TestRateSinkResetsOnNewCall(t *testing.T) {
	stream := newPCMRecorder()
	s := withRate(stream, 8000, 48000).(*rateSink)
	_ = s.WritePCMForCall("V", 1, tone(1000, 8000, 800))
	before := len(stream.get("V"))
	_ = s.WritePCMForCall("V", 2, make([]int16, 160))
	tail := stream.get("V")[before:]
	for i, v := range tail {
		if v != 0 {
			t.Fatalf("sample %d of call 2 = %d: call 1's audio bled into it", i, v)
		}
	}
}
