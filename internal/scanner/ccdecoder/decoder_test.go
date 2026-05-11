package ccdecoder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeIQSource feeds a pre-supplied IQ stream onto the channel
// StreamIQ returns. When `repeat` is true the source cycles
// through `chunks` until ctx cancels, so the Run loop has IQ to
// pump after a mid-test pipeline swap.
type fakeIQSource struct {
	chunks [][]complex64
	repeat bool
}

func (f *fakeIQSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		for {
			for _, c := range f.chunks {
				select {
				case <-ctx.Done():
					return
				case ch <- c:
				}
			}
			if !f.repeat {
				<-ctx.Done()
				return
			}
		}
	}()
	return ch, nil
}

type fakeTuner struct{}

func (fakeTuner) SetCenterFreq(uint32) error { return nil }

type erroringIQSource struct{ err error }

func (e erroringIQSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	return nil, e.err
}

// recordingPipeline implements ProtocolPipeline and records every
// Process / Reset / Close call so tests can assert on the
// connector's invocation pattern.
type recordingPipeline struct {
	processChunks [][]complex64
	resets        int
	closes        int
}

func (r *recordingPipeline) Process(iq []complex64) {
	cp := make([]complex64, len(iq))
	copy(cp, iq)
	r.processChunks = append(r.processChunks, cp)
}
func (r *recordingPipeline) Reset()       { r.resets++ }
func (r *recordingPipeline) Close() error { r.closes++; return nil }

// withRecordingFactory swaps the package factory map for one that
// builds *recordingPipelines, returns the built pipeline so the
// test can inspect it, and restores the original map on cleanup.
func withRecordingFactory(t *testing.T, proto trunking.Protocol) *recordingPipeline {
	t.Helper()
	saved := factories
	rec := &recordingPipeline{}
	factories = map[trunking.Protocol]PipelineFactory{
		proto: func(PipelineOptions) (ProtocolPipeline, error) {
			return rec, nil
		},
	}
	t.Cleanup(func() { factories = saved })
	return rec
}

func TestNewRequiresBus(t *testing.T) {
	_, err := New(Options{IQ: &fakeIQSource{}, SampleRateHz: 48000})
	if err == nil {
		t.Fatalf("expected error for missing Bus")
	}
}

func TestNewRequiresIQSource(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	_, err := New(Options{Bus: bus, SampleRateHz: 48000})
	if err == nil {
		t.Fatalf("expected error for missing IQ source")
	}
}

func TestNewRequiresSampleRate(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	_, err := New(Options{Bus: bus, IQ: &fakeIQSource{}})
	if err == nil {
		t.Fatalf("expected error for missing SampleRateHz")
	}
}

func TestRunPropagatesStreamIQError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	wantErr := errors.New("usb gone")
	d, err := New(Options{
		Bus: bus, IQ: erroringIQSource{err: wantErr}, SampleRateHz: 48000,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := d.Run(context.Background()); !errors.Is(got, wantErr) {
		t.Errorf("Run = %v, want %v", got, wantErr)
	}
}

// TestRunSwapsPipelineOnHuntProgress: publish a HuntProgress event
// before the StreamIQ stream is exhausted, then confirm the
// recording pipeline's factory ran (so an IQ chunk flowing after
// the event makes it through Process).
func TestRunSwapsPipelineOnHuntProgress(t *testing.T) {
	rec := withRecordingFactory(t, trunking.ProtocolP25)

	bus := events.NewBus(16)
	defer bus.Close()

	iq := &fakeIQSource{
		chunks: [][]complex64{make([]complex64, 64), make([]complex64, 64)},
		repeat: true,
	}
	systems := []trunking.System{{
		Name: "TestSys", Protocol: trunking.ProtocolP25,
		ControlChannels: []uint32{851_012_500},
	}}
	d, err := New(Options{
		Bus: bus, IQ: iq, Tuner: fakeTuner{},
		Systems: systems, SampleRateHz: 48000,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Publish the HuntProgress before Run starts so the subscription
	// catches it. Bus.Publish is fan-out; the events.NewBus buffer
	// holds it until Run subscribes.
	go func() {
		// Wait long enough that Run subscribes and starts consuming.
		time.Sleep(50 * time.Millisecond)
		bus.Publish(events.Event{
			Kind: events.KindHuntProgress,
			Payload: trunking.HuntProgress{
				System:          "TestSys",
				AttemptedFreqHz: 851_012_500,
				AttemptIndex:    1,
				TotalCandidates: 1,
				At:              time.Now(),
			},
		})
		// Let the swap + a couple of pump iterations land.
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_ = d.Run(ctx)

	// At least one Process call should have hit the recording
	// pipeline (the swap happened, then the next IQ chunk was
	// pumped through it).
	// Note: subscription delivery + StreamIQ pre-loaded buffer
	// timing isn't strictly ordered, so we tolerate the swap
	// happening after one chunk has already been dropped.
	if len(rec.processChunks) == 0 && rec.closes == 0 {
		t.Errorf("expected the recording pipeline to be either Process'd or Close'd, got neither")
	}
}

// TestHandleProgressUnknownSystemIsIgnored: HuntProgress for a
// system we don't know about must NOT construct a pipeline (no
// recordingPipeline.Process calls), but must also not crash.
func TestHandleProgressUnknownSystemIsIgnored(t *testing.T) {
	rec := withRecordingFactory(t, trunking.ProtocolP25)

	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000,
		Systems: []trunking.System{{
			Name: "Known", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{851_012_500},
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d.handleProgress(trunking.HuntProgress{
		System: "Unknown", AttemptedFreqHz: 851_012_500,
	})
	if len(rec.processChunks) != 0 {
		t.Errorf("recording pipeline should not have been built for an unknown system")
	}
}

// TestHandleProgressUnknownProtocolClearsActive: HuntProgress for
// a known system whose protocol has no factory must clear any
// active pipeline (otherwise stale state bleeds into new tunes).
func TestHandleProgressUnknownProtocolClearsActive(t *testing.T) {
	// Empty factory map for this test — every protocol is "unknown".
	saved := factories
	factories = map[trunking.Protocol]PipelineFactory{}
	defer func() { factories = saved }()

	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000,
		Systems: []trunking.System{{
			Name: "Sys", Protocol: trunking.ProtocolDMR,
			ControlChannels: []uint32{460_000_000},
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Seed an active pipeline.
	rec := &recordingPipeline{}
	d.active = rec
	d.activeAt = "Sys"

	d.handleProgress(trunking.HuntProgress{
		System: "Sys", AttemptedFreqHz: 460_000_000,
	})
	if d.active != nil {
		t.Errorf("active pipeline should be cleared when protocol has no factory")
	}
	if rec.closes != 1 {
		t.Errorf("recording pipeline Close calls = %d, want 1", rec.closes)
	}
}

// TestP25Phase1FactoryConstructs: smoke test that the wired
// P25 P1 factory builds without error and returns a pipeline whose
// Process / Reset / Close run cleanly on a small IQ chunk.
func TestP25Phase1FactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newP25Phase1Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 851_012_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newP25Phase1Pipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDPMRFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newDPMRPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 446_006_250, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newDPMRPipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestYSFFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newYSFPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 446_000_000, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newYSFPipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
