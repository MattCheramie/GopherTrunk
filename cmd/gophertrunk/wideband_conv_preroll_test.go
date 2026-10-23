package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"math/cmplx"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// voiceChainEngine is a conventional.Engine that does what the composer does
// when a call starts: it subscribes to the call's IQ source (StreamIQ) and
// collects what the voice chain would hear until the call ends.
type voiceChainEngine struct {
	src *widebandt2.ChannelVoiceSource

	mu     sync.Mutex
	iq     []complex64
	calls  int
	ended  bool
	cancel context.CancelFunc
	done   chan struct{}
}

func (e *voiceChainEngine) HandleSyntheticCall(trunking.Grant, string) {
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := e.src.StreamIQ(ctx)
	if err != nil {
		cancel()
		return
	}
	e.mu.Lock()
	e.calls++
	e.cancel = cancel
	e.done = make(chan struct{})
	done := e.done
	e.mu.Unlock()
	go func() {
		defer close(done)
		for c := range ch {
			e.mu.Lock()
			e.iq = append(e.iq, c...)
			e.mu.Unlock()
		}
	}()
}

func (e *voiceChainEngine) EndSyntheticCall(string, trunking.EndReason) bool {
	e.mu.Lock()
	cancel, done := e.cancel, e.done
	e.ended = true
	e.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	return true
}

func (e *voiceChainEngine) Touch(string) {}

// carrierSpan reports, for IQ at 48 kHz, how many seconds of carrier it holds
// and how many seconds of noise precede the first carrier block (10 ms
// blocks; the carrier sits ~30 dB over the noise, so the split is the
// midpoint in dB).
func carrierSpan(iq []complex64) (carrier, leadingNoise float64) {
	const block = 480
	first := -1
	n := 0
	for i := 0; i+block <= len(iq); i += block {
		var p float64
		for _, v := range iq[i : i+block] {
			p += float64(real(v)*real(v) + imag(v)*imag(v))
		}
		if 10*math.Log10(p/block+1e-20) > -30 { // carrier ≈ -11 dBFS, noise ≈ -40
			n++
			if first < 0 {
				first = i / block
			}
		}
	}
	if first < 0 {
		return 0, float64(len(iq)) / 48_000
	}
	return float64(n*block) / 48_000, float64(first*block) / 48_000
}

// TestWidebandConvVoiceChainStartsAtTheCarrier is issue #1288's "tone
// detection must not discard initial audio": a CTCSS-gated wideband
// conventional channel declares its call only once the tone gate confirms,
// up to about half a second after keyup, and the voice chain subscribed only
// then, so every recording began that far into the transmission. The
// scanner now marks where the carrier came up and the chain starts there.
// Fail-first: the old wiring gives the chain 0.96 s of the 1.2 s
// transmission (the 240 ms the CTCSS gate took is gone). An untoned channel,
// whose call starts on the first above-squelch chunk, must not gain a
// lead-in of noise either.
func TestWidebandConvVoiceChainStartsAtTheCarrier(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ctcss float64
	}{
		{"ctcss", 100},
		{"untoned", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				rate     = 960_000.0
				centerHz = 460_500_000
				chunk    = 9_600 // 10 ms
				seconds  = 2.4
				on, off  = 0.5, 1.7
			)
			chHz := uint32(centerHz - 200_000)
			iq := synthToneGated(rate, seconds, -200_000, 1000, tc.ctcss, on, off)
			var chunks [][]complex64
			for i := 0; i+chunk <= len(iq); i += chunk {
				chunks = append(chunks, iq[i:i+chunk])
			}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			bus := events.NewBus(64)
			defer bus.Close()
			eng, err := widebandt2.New(widebandt2.Options{
				Log: log, Bus: bus,
				Device:       &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
				SampleRateHz: uint32(rate), CenterFreqHz: centerHz,
				Channels: []widebandt2.ChannelConfig{{FrequencyHz: chHz, Conventional: true}},
				Serial:   "WB1",
			})
			if err != nil {
				t.Fatal(err)
			}
			src := widebandt2.NewChannelVoiceSource(func() *widebandt2.Engine { return eng }, chHz, wbConvSerial("WB1", chHz))
			cfgCh := config.ConvChannelConfig{Label: "A", FrequencyHz: chHz, Mode: "nfm", SquelchDbFS: -30, HangtimeMs: 250}
			if tc.ctcss > 0 {
				cfgCh.Tone = config.ConvToneConfig{Mode: "ctcss", CTCSSHz: tc.ctcss}
			}
			chain := &voiceChainEngine{src: src}
			cs, err := newWidebandConvScanner(src, convChannelFromConfig(cfgCh, config.WidebandConvPositionalBase),
				chain, discardRecorder{}, nil, log)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = cs.Run(ctx) }()
			time.Sleep(50 * time.Millisecond)
			go func() { _ = eng.Run(ctx) }()

			deadline := time.Now().Add(15 * time.Second)
			for {
				chain.mu.Lock()
				ended := chain.ended
				chain.mu.Unlock()
				if ended || time.Now().After(deadline) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			cancel()

			chain.mu.Lock()
			defer chain.mu.Unlock()
			if chain.calls != 1 || !chain.ended {
				t.Fatalf("calls=%d ended=%v, want one call that ended", chain.calls, chain.ended)
			}
			carrier, lead := carrierSpan(chain.iq)
			t.Logf("voice chain heard %.2f s of a %.2f s transmission, after %.3f s of noise", carrier, off-on, lead)
			if carrier < 0.95*(off-on) {
				t.Errorf("voice chain heard %.2f s of the %.2f s transmission; its start was lost", carrier, off-on)
			}
			if lead > 0.03 {
				t.Errorf("voice chain starts with %.3f s of noise before the carrier", lead)
			}
		})
	}
}

// synthToneGated is one NFM transmitter at offsetHz, keyed from on to off: a
// voiceHz tone at 3 kHz deviation plus, when ctcssHz > 0, a sub-audible tone
// at 500 Hz deviation, over a little receiver noise.
func synthToneGated(rate, seconds, offsetHz, voiceHz, ctcssHz, on, off float64) []complex64 {
	const amp = 0.3
	n := int(rate * seconds)
	out := make([]complex64, n)
	rng := rand.New(rand.NewSource(2))
	var phase float64
	for i := range out {
		t := float64(i) / rate
		inst := offsetHz + 3000*math.Sin(2*math.Pi*voiceHz*t)
		if ctcssHz > 0 {
			inst += 500 * math.Sin(2*math.Pi*ctcssHz*t)
		}
		phase += 2 * math.Pi * inst / rate
		var s complex128
		if t >= on && t < off {
			s = cmplx.Rect(amp, phase)
		}
		s += complex(rng.NormFloat64()*0.01, rng.NormFloat64()*0.01)
		out[i] = complex64(s)
	}
	return out
}
