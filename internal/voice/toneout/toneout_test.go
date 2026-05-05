package toneout

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

const sampleRate = 8000

// genTone returns int16 PCM samples for a sine wave at freqHz lasting
// duration. Amplitude is 0.6 of full-scale to leave headroom.
func genTone(freqHz float64, duration time.Duration) []int16 {
	n := int(float64(sampleRate) * duration.Seconds())
	out := make([]int16, n)
	const amp = 0.6
	for i := range out {
		v := amp * math.Sin(2*math.Pi*freqHz*float64(i)/sampleRate)
		out[i] = int16(v * 32767)
	}
	return out
}

// genSilence returns n samples of silence.
func genSilence(duration time.Duration) []int16 {
	return make([]int16, int(float64(sampleRate)*duration.Seconds()))
}

// busSink collects events on a subscription so tests can assert kinds
// and payloads without racing the publisher.
func busSink(t *testing.T, bus *events.Bus) (collect func(int, time.Duration) []events.Event, close func()) {
	t.Helper()
	sub := bus.Subscribe()
	mu := sync.Mutex{}
	xs := []events.Event{}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case ev, ok := <-sub.C:
				if !ok {
					return
				}
				mu.Lock()
				xs = append(xs, ev)
				mu.Unlock()
			}
		}
	}()
	return func(want int, deadline time.Duration) []events.Event {
			end := time.Now().Add(deadline)
			for time.Now().Before(end) {
				mu.Lock()
				if len(xs) >= want {
					out := append([]events.Event(nil), xs...)
					mu.Unlock()
					return out
				}
				mu.Unlock()
				time.Sleep(5 * time.Millisecond)
			}
			mu.Lock()
			defer mu.Unlock()
			return append([]events.Event(nil), xs...)
		}, func() {
			close := struct{}{}
			_ = close
			sub.Close()
		}
}

func TestGoertzelMagnitudePeaksAtTarget(t *testing.T) {
	const target = 1000.0
	const amp = 0.6
	// At unit amplitude the normalised magnitude approaches 1; at
	// amplitude `amp` it scales quadratically (≈ amp² = 0.36). We
	// just need on-target to be substantially larger than off-target.
	g := NewGoertzel(target, sampleRate, 800)
	tones := genTone(target, 100*time.Millisecond)
	var onMag float64
	for _, s := range tones {
		if m, ok := g.Process(s); ok {
			onMag = m
		}
	}
	want := amp * amp * 0.7 // generous lower bound on amp² ≈ 0.36
	if onMag < want {
		t.Errorf("on-target magnitude = %f, want > %f", onMag, want)
	}

	// Off-target should produce far less power.
	g2 := NewGoertzel(2000.0, sampleRate, 800)
	var offMag float64
	for _, s := range tones {
		if m, ok := g2.Process(s); ok {
			offMag = m
		}
	}
	if offMag > 0.05 {
		t.Errorf("off-target magnitude = %f, want < 0.05", offMag)
	}
	if onMag/offMag < 5 {
		t.Errorf("on/off ratio = %f, want > 5", onMag/offMag)
	}
}

func TestDetectorMatchesSingleTone(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name:     "test-single",
			AlphaTag: "Test",
			Tones: []Tone{
				{FrequencyHz: 1000, MinDuration: 200 * time.Millisecond, MaxDuration: 1500 * time.Millisecond},
			},
		}},
		SampleRate: sampleRate,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 500 ms of 1 kHz tone, then silence so the trailing block detects
	// the tone-end and the matcher commits the match.
	d.WritePCM("VOICE-1", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))

	evs := collect(1, time.Second)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(evs), evs)
	}
	if evs[0].Kind != events.KindToneAlert {
		t.Errorf("kind = %s", evs[0].Kind)
	}
	a, ok := evs[0].Payload.(Alert)
	if !ok {
		t.Fatalf("payload type = %T", evs[0].Payload)
	}
	if a.Profile != "test-single" || a.DeviceSerial != "VOICE-1" {
		t.Errorf("alert = %+v", a)
	}
}

func TestDetectorMatchesTwoTone(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name: "QC2-station-1",
			Tones: []Tone{
				{FrequencyHz: 1000, MinDuration: 250 * time.Millisecond, MaxDuration: 1500 * time.Millisecond},
				{FrequencyHz: 1500, MinDuration: 250 * time.Millisecond, MaxDuration: 4000 * time.Millisecond},
			},
		}},
		SampleRate: sampleRate,
	})
	if err != nil {
		t.Fatal(err)
	}

	// A-tone for 700 ms, ~30 ms gap, B-tone for 2 s, then silence.
	d.WritePCM("VOICE-1", genTone(1000, 700*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(30*time.Millisecond))
	d.WritePCM("VOICE-1", genTone(1500, 2*time.Second))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))

	evs := collect(1, time.Second)
	if len(evs) != 1 {
		t.Fatalf("got %d events: %+v", len(evs), evs)
	}
	a := evs[0].Payload.(Alert)
	if got := a.FrequenciesHz; len(got) != 2 || got[0] != 1000 || got[1] != 1500 {
		t.Errorf("matched freqs = %v", got)
	}
}

func TestDetectorIgnoresWrongFrequency(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name:  "1k",
			Tones: []Tone{{FrequencyHz: 1000, MinDuration: 200 * time.Millisecond}},
		}},
		SampleRate: sampleRate,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 700 ms of 1500 Hz — wrong frequency for the profile.
	d.WritePCM("VOICE-1", genTone(1500, 700*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))

	evs := collect(0, 200*time.Millisecond)
	if len(evs) != 0 {
		t.Errorf("got %d events, want 0", len(evs))
	}
}

func TestDetectorIgnoresTooShortTone(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name: "long-only",
			Tones: []Tone{{FrequencyHz: 1000, MinDuration: 500 * time.Millisecond}},
		}},
		SampleRate: sampleRate,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Only 200 ms of 1 kHz — below MinDuration.
	d.WritePCM("VOICE-1", genTone(1000, 200*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))

	evs := collect(0, 200*time.Millisecond)
	if len(evs) != 0 {
		t.Errorf("got %d events, want 0", len(evs))
	}
}

func TestDetectorCooldownSuppressesRefires(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	now := time.Unix(1_000_000, 0)
	clock := &fakeClock{t: now}

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name:     "cool-1",
			Tones:    []Tone{{FrequencyHz: 1000, MinDuration: 200 * time.Millisecond}},
			Cooldown: time.Hour,
		}},
		SampleRate: sampleRate,
		Now:        clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	// First match.
	d.WritePCM("VOICE-1", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))
	first := collect(1, time.Second)
	if len(first) != 1 {
		t.Fatalf("first match: got %d events", len(first))
	}

	// Second tone within the cooldown window — should NOT fire.
	d.WritePCM("VOICE-1", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))
	second := collect(2, 200*time.Millisecond)
	if len(second) != 1 {
		t.Errorf("cooldown failed: got %d events, want 1", len(second))
	}

	// Advance the clock past cooldown, then re-run — should fire again.
	clock.t = clock.t.Add(2 * time.Hour)
	d.WritePCM("VOICE-1", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))
	third := collect(2, time.Second)
	if len(third) != 2 {
		t.Errorf("post-cooldown: got %d events, want 2", len(third))
	}
}

func TestDetectorPerDeviceIsolation(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	collect, closeSub := busSink(t, bus)
	defer closeSub()

	d, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{
			Name: "qc2",
			Tones: []Tone{
				{FrequencyHz: 1000, MinDuration: 250 * time.Millisecond},
				{FrequencyHz: 1500, MinDuration: 250 * time.Millisecond},
			},
		}},
		SampleRate: sampleRate,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Device VOICE-1 hears the A-tone but NOT the B-tone.
	d.WritePCM("VOICE-1", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-1", genSilence(150*time.Millisecond))
	// Device VOICE-2 hears a complete two-tone sequence.
	d.WritePCM("VOICE-2", genTone(1000, 500*time.Millisecond))
	d.WritePCM("VOICE-2", genSilence(20*time.Millisecond))
	d.WritePCM("VOICE-2", genTone(1500, 600*time.Millisecond))
	d.WritePCM("VOICE-2", genSilence(150*time.Millisecond))

	evs := collect(1, time.Second)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(evs), evs)
	}
	if a := evs[0].Payload.(Alert); a.DeviceSerial != "VOICE-2" {
		t.Errorf("alert serial = %q, want VOICE-2", a.DeviceSerial)
	}
}

func TestProfileValidate(t *testing.T) {
	cases := []struct {
		name string
		p    Profile
		ok   bool
	}{
		{"empty name", Profile{}, false},
		{"no tones", Profile{Name: "x"}, false},
		{"tone freq zero", Profile{Name: "x", Tones: []Tone{{FrequencyHz: 0, MinDuration: time.Second}}}, false},
		{"tone min_duration zero", Profile{Name: "x", Tones: []Tone{{FrequencyHz: 1000}}}, false},
		{"max < min", Profile{Name: "x", Tones: []Tone{{FrequencyHz: 1000, MinDuration: time.Second, MaxDuration: time.Millisecond}}}, false},
		{"ok", Profile{Name: "x", Tones: []Tone{{FrequencyHz: 1000, MinDuration: 500 * time.Millisecond}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			if (err == nil) != tc.ok {
				t.Errorf("Validate err = %v, ok = %v", err, tc.ok)
			}
		})
	}
}

func TestNewValidates(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Error("expected error for missing bus")
	}
	bus := events.NewBus(1)
	defer bus.Close()
	_, err := New(Options{
		Bus: bus,
		Profiles: []Profile{{Name: "bad"}}, // no tones
	})
	if err == nil {
		t.Error("expected validation error to propagate")
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }
