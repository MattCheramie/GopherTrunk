package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// amBurst is an AM transmitter: a carrier at offsetHz, amplitude-modulated
// by a tone at depth, keyed from on to off seconds.
type amBurst struct {
	offsetHz, toneHz, depth float64
	on, off                 float64
}

// synthAMWideband sums AM transmitters plus receiver noise.
func synthAMWideband(rate, seconds, noise float64, txs []amBurst) []complex64 {
	const amp = 0.2
	n := int(rate * seconds)
	out := make([]complex64, n)
	rng := rand.New(rand.NewSource(7))
	for i := range out {
		t := float64(i) / rate
		var re, im float64
		for _, tx := range txs {
			if t < tx.on || t >= tx.off {
				continue
			}
			a := amp * (1 + tx.depth*math.Sin(2*math.Pi*tx.toneHz*t))
			ph := 2 * math.Pi * tx.offsetHz * t
			re += a * math.Cos(ph)
			im += a * math.Sin(ph)
		}
		re += rng.NormFloat64() * noise
		im += rng.NormFloat64() * noise
		out[i] = complex(float32(re), float32(im))
	}
	return out
}

// envelopeToneRatioDB is the AM-demodulated (envelope) power at want over
// the power at other, in dB.
func envelopeToneRatioDB(iq []complex64, rate, want, other float64) float64 {
	env := make([]float64, len(iq))
	var mean float64
	for i, v := range iq {
		env[i] = math.Hypot(float64(real(v)), float64(imag(v)))
		mean += env[i]
	}
	mean /= float64(len(env))
	power := func(f float64) float64 {
		var re, im float64
		for i, v := range env {
			re += (v - mean) * math.Cos(2*math.Pi*f*float64(i)/rate)
			im += (v - mean) * math.Sin(2*math.Pi*f*float64(i)/rate)
		}
		return re*re + im*im
	}
	return 10 * math.Log10(power(want)/power(other))
}

// TestWidebandConventionalAMOverlappingCalls: AM channels on one wideband
// dongle (an air-band receiver's use case, issue #1288) each open their own
// call on their own carrier-to-noise squelch while the transmissions overlap,
// close after hangtime, and a channel with only noise on it never opens. The
// AM squelch runs on the 48 kHz tap exactly as on a scan-list channel.
func TestWidebandConventionalAMOverlappingCalls(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 127_000_000
		chunk    = 9_600 // 10 ms
		seconds  = 3.0
		hangtime = 250 * time.Millisecond
	)
	aHz, bHz, quietHz := uint32(centerHz-300_000), uint32(centerHz+250_000), uint32(centerHz-50_000)
	// Each a little off its channel, as a real transmitter is.
	txA := amBurst{offsetHz: -300_000 - 1200, toneHz: 1000, depth: 0.7, on: 0.5, off: 1.7}
	txB := amBurst{offsetHz: 250_000 + 800, toneHz: 1700, depth: 0.7, on: 0.9, off: 2.2}
	iq := synthAMWideband(rate, seconds, 0.01, []amBurst{txA, txB})
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	freqs := []uint32{aHz, bHz, quietHz}
	var chans []widebandt2.ChannelConfig
	for _, hz := range freqs {
		chans = append(chans, widebandt2.ChannelConfig{FrequencyHz: hz, Conventional: true, ConventionalAM: true})
	}
	eng, err := widebandt2.New(widebandt2.Options{
		Log: log, Bus: bus, Device: &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
		SampleRateHz: uint32(rate), CenterFreqHz: centerHz, Channels: chans, Serial: "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	for hz, tag := range eng.ChannelProtocolTags() {
		if tag != "am-conv" {
			t.Errorf("%d Hz tagged %q, want am-conv", hz, tag)
		}
	}
	get := func() *widebandt2.Engine { return eng }
	calls := &syntheticCallLog{start: time.Now(), calls: map[string]*loggedCall{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type tap struct {
		mu sync.Mutex
		iq []complex64
	}
	taps := map[uint32]*tap{}
	for _, hz := range []uint32{aHz, bHz} {
		ch, unsub, ok := eng.SubscribeChannelIQ(hz)
		if !ok {
			t.Fatalf("no tap on %d Hz", hz)
		}
		defer unsub()
		tp := &tap{}
		taps[hz] = tp
		go func() {
			for c := range ch {
				tp.mu.Lock()
				tp.iq = append(tp.iq, c...)
				tp.mu.Unlock()
			}
		}()
	}

	for i, hz := range freqs {
		cfgCh := config.ConvChannelConfig{Label: string(rune('A' + i)), FrequencyHz: hz, Mode: "am",
			HangtimeMs: int(hangtime / time.Millisecond)}
		src := widebandt2.NewChannelVoiceSource(get, hz, wbConvSerial("WB1", hz))
		cs, err := newWidebandConvScanner(src, convChannelFromConfig(cfgCh, config.WidebandConvPositionalBase|uint32(i)),
			calls, discardRecorder{}, nil, log)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = cs.Run(ctx) }()
	}
	time.Sleep(50 * time.Millisecond)
	streamStart := time.Since(calls.start)
	go func() { _ = eng.Run(ctx) }()

	// Wait for both calls to end (B's ends 2.45 s into the stream), so the
	// quiet channel has had seconds of noise to open on falsely.
	deadline := time.Now().Add(15 * time.Second)
	for {
		calls.mu.Lock()
		ended := 0
		for _, c := range calls.order {
			if c.ended && c.freqHz != quietHz {
				ended++
			}
		}
		calls.mu.Unlock()
		if ended >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	cancel()

	calls.mu.Lock()
	defer calls.mu.Unlock()
	byFreq := map[uint32]*loggedCall{}
	for _, c := range calls.order {
		if byFreq[c.freqHz] != nil {
			t.Errorf("more than one call on %d Hz: %+v", c.freqHz, calls.order)
		}
		byFreq[c.freqHz] = c
	}
	if c := byFreq[quietHz]; c != nil {
		t.Errorf("noise-only channel opened a call: %+v", c)
	}
	a, b := byFreq[aHz], byFreq[bHz]
	if a == nil || b == nil {
		t.Fatalf("calls %+v, want one on each of %d and %d", calls.order, aHz, bHz)
	}
	if a.serial == b.serial || a.groupID == b.groupID {
		t.Errorf("calls share an identity: A %s/%#x, B %s/%#x", a.serial, a.groupID, b.serial, b.groupID)
	}
	if !a.ended || !b.ended {
		t.Fatalf("calls did not end: A %+v, B %+v", a, b)
	}
	// Never before keyup or unkey. The stream is paced in real time but a
	// loaded runner can fall behind it, so later bounds are stated as order:
	// A keys 0.4 s before B and unkeys 0.5 s before B.
	for name, c := range map[string]struct {
		call *loggedCall
		tx   amBurst
	}{"A": {a, txA}, "B": {b, txB}} {
		on := streamStart + time.Duration(c.tx.on*float64(time.Second))
		off := streamStart + time.Duration(c.tx.off*float64(time.Second))
		if c.call.start < on-50*time.Millisecond {
			t.Errorf("%s opened at %v, before the transmitter keyed at %v", name, c.call.start, on)
		}
		if c.call.end < off {
			t.Errorf("%s ended at %v, before the transmitter unkeyed at %v", name, c.call.end, off)
		}
	}
	if !(a.start < b.start && a.end < b.end) {
		t.Errorf("calls out of order: A %v–%v, B %v–%v (A keys and unkeys first)", a.start, a.end, b.start, b.end)
	}
	if !(b.start < a.end && a.start < b.end) {
		t.Errorf("calls did not overlap: A %v–%v, B %v–%v", a.start, a.end, b.start, b.end)
	}

	// Cross-talk: during the overlap (1.0–1.6 s) each tap's envelope carries
	// its own transmitter's tone far above the other's.
	for hz, want := range map[uint32][2]float64{aHz: {txA.toneHz, txB.toneHz}, bHz: {txB.toneHz, txA.toneHz}} {
		tp := taps[hz]
		tp.mu.Lock()
		lo, hi := int(1.0*48_000), int(1.6*48_000)
		if len(tp.iq) < hi {
			tp.mu.Unlock()
			t.Fatalf("tap %d Hz delivered %d samples, want ≥ %d", hz, len(tp.iq), hi)
		}
		r := envelopeToneRatioDB(tp.iq[lo:hi], 48_000, want[0], want[1])
		tp.mu.Unlock()
		t.Logf("tap %d Hz: own tone %.1f dB above the other", hz, r)
		if r < 30 {
			t.Errorf("tap %d Hz: own tone only %.1f dB above the other channel's", hz, r)
		}
	}
}
