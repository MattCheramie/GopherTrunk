package main

import (
	"context"
	"log/slog"
	"math"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// liveLoudnessSink is an optional composer.PCMSink decorator that runs a
// real-time envelope-follower AGC over decoded digital-voice PCM before
// forwarding it to the live network stream (the audio publisher). It gives the
// live stream consistent loudness so it tracks the loudness-normalized on-disk
// recordings instead of arriving raw and quieter/inconsistent (the third gap in
// the live-vs-recorded parity work).
//
// It is gated behind audio.live_loudness (default off) and wraps ONLY the
// digital decoded-PCM tap, so:
//   - the on-disk WAV is untouched (the recorder writes from its own decode and
//     applies its own per-call EBU R128 normalization) — no double-processing;
//   - analog FM live audio is untouched (its loudness is shaped upstream by the
//     composer's own optional audio AGC), preserving analog live==recorded.
//
// R128 *integrated* loudness needs the whole call, so this can't reproduce the
// recorder's offline result bit-for-bit; the envelope AGC is the real-time
// approximation that matches what the listener actually perceives.
type liveLoudnessSink struct {
	next       composer.PCMSink
	sampleRate float64
	log        *slog.Logger

	bus    *events.Bus
	busSub *events.Subscription

	mu   sync.Mutex
	agcs map[string]*serialAGC

	runDone   chan struct{}
	closeOnce sync.Once
}

// serialAGC pins one AudioAGC (and its scratch buffer) to a device serial.
// AudioAGC.Process is not concurrency-safe, so each entry carries its own lock;
// a given serial is normally driven by a single decode goroutine, but the lock
// keeps a CallStart-triggered Reset from racing an in-flight WritePCM.
type serialAGC struct {
	mu      sync.Mutex
	agc     *dsp.AudioAGC
	scratch []float32
	out     []int16
}

// newLiveLoudnessSink builds the decorator. sampleRate is the PCM rate of the
// decoded digital audio (8 kHz today). next is the downstream sink (the audio
// publisher). The bus subscription is taken at construction so CallStart events
// fired before Run begins still reset the right serial.
func newLiveLoudnessSink(next composer.PCMSink, sampleRate float64, bus *events.Bus, log *slog.Logger) *liveLoudnessSink {
	if log == nil {
		log = slog.Default()
	}
	return &liveLoudnessSink{
		next:       next,
		sampleRate: sampleRate,
		log:        log,
		bus:        bus,
		busSub:     bus.Subscribe(),
		agcs:       make(map[string]*serialAGC),
		runDone:    make(chan struct{}),
	}
}

// entry returns the per-serial AGC, creating it on first use.
func (s *liveLoudnessSink) entry(serial string) *serialAGC {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.agcs[serial]
	if e == nil {
		e = &serialAGC{
			agc: dsp.NewAudioAGC(dsp.AudioAGCConfig{SampleRate: s.sampleRate}),
		}
		s.agcs[serial] = e
	}
	return e
}

// WritePCM levels the samples for deviceSerial and forwards them downstream.
// The input slice is never mutated; a fresh int16 buffer is handed to next.
func (s *liveLoudnessSink) WritePCM(deviceSerial string, samples []int16) error {
	if len(samples) == 0 {
		return s.next.WritePCM(deviceSerial, samples)
	}
	e := s.entry(deviceSerial)

	e.mu.Lock()
	if cap(e.scratch) < len(samples) {
		e.scratch = make([]float32, len(samples))
		e.out = make([]int16, len(samples))
	}
	in := e.scratch[:len(samples)]
	for i, v := range samples {
		in[i] = float32(v) / 32768
	}
	in = e.agc.Process(in, in)
	out := e.out[:len(samples)]
	for i, v := range in {
		out[i] = clampToInt16(v)
	}
	e.mu.Unlock()

	return s.next.WritePCM(deviceSerial, out)
}

// Run drains the bus, resetting a serial's AGC envelope at CallStart (so a loud
// prior call doesn't under-amplify the next call's onset through the slow
// release) and dropping it at CallEnd to bound the map. Returns ctx.Err() on
// shutdown. Spawned on a goroutine by the daemon like other long-lived parts.
func (s *liveLoudnessSink) Run(ctx context.Context) error {
	defer close(s.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-s.busSub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					if e := s.lookup(cs.DeviceSerial); e != nil {
						e.mu.Lock()
						e.agc.Reset()
						e.mu.Unlock()
					}
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					s.mu.Lock()
					delete(s.agcs, ce.DeviceSerial)
					s.mu.Unlock()
				}
			}
		}
	}
}

// lookup returns the existing entry for a serial without creating one.
func (s *liveLoudnessSink) lookup(serial string) *serialAGC {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agcs[serial]
}

// Close releases the bus subscription. Safe to call multiple times.
func (s *liveLoudnessSink) Close() error {
	s.closeOnce.Do(func() {
		if s.busSub != nil {
			s.busSub.Close()
		}
	})
	return nil
}

// clampToInt16 converts a float sample in roughly [-1, 1] to int16, saturating
// rather than wrapping on overshoot so AGC transients don't produce harsh
// wrap-around clicks.
func clampToInt16(v float32) int16 {
	scaled := math.Round(float64(v) * 32767)
	if scaled > math.MaxInt16 {
		return math.MaxInt16
	}
	if scaled < math.MinInt16 {
		return math.MinInt16
	}
	return int16(scaled)
}
