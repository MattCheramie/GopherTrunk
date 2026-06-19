package ccdecoder

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/autotune"
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
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

// closingIQSource returns a channel that closes after dispatching
// `chunks` worth of IQ — regardless of ctx state. Models the SDR
// driver's behaviour when its USB reaper dies (issue #345): the
// channel closes, the decoder must surface ErrIQStreamClosed so the
// daemon's restart loop can re-open the SDR.
type closingIQSource struct {
	chunks [][]complex64
}

func (c *closingIQSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		for _, chunk := range c.chunks {
			select {
			case <-ctx.Done():
				return
			case ch <- chunk:
			}
		}
	}()
	return ch, nil
}

// TestRunReturnsErrIQStreamClosedOnStreamEOF pins the issue-#345
// surface: when the IQ stream closes while ctx is still live (the
// SDR's USB reaper died and the driver propagated EOF), Run must
// return ErrIQStreamClosed so the daemon's restart loop can fire.
// Before the fix, Run returned nil and the daemon swallowed the exit
// silently — the entire pipeline went idle at 0% CPU.
func TestRunReturnsErrIQStreamClosedOnStreamEOF(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	src := &closingIQSource{
		chunks: [][]complex64{make([]complex64, 4)},
	}
	d, err := New(Options{Bus: bus, IQ: src, SampleRateHz: 48000})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- d.Run(context.Background())
	}()
	select {
	case got := <-done:
		if !errors.Is(got, ErrIQStreamClosed) {
			t.Errorf("Run = %v, want ErrIQStreamClosed", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after stream EOF")
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
	got := d.Run(context.Background())
	// The underlying StreamIQ error must remain inspectable so callers
	// can still log the root cause.
	if !errors.Is(got, wantErr) {
		t.Errorf("Run = %v, want underlying %v preserved", got, wantErr)
	}
	// A non-context StreamIQ open failure must now classify as
	// ErrIQStreamClosed so the daemon's restart loop catches it and
	// either retries or escalates to a clean fatal. Issue #345.
	if !errors.Is(got, ErrIQStreamClosed) {
		t.Errorf("Run = %v, want errors.Is(..., ErrIQStreamClosed)", got)
	}
}

// TestRunPassesThroughContextCancelFromStreamIQ pins the inverse: a
// context-cancel error from StreamIQ must NOT be wrapped as
// ErrIQStreamClosed — shutdown is not a stream failure.
func TestRunPassesThroughContextCancelFromStreamIQ(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: erroringIQSource{err: context.Canceled}, SampleRateHz: 48000,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := d.Run(context.Background())
	if !errors.Is(got, context.Canceled) {
		t.Errorf("Run = %v, want context.Canceled", got)
	}
	if errors.Is(got, ErrIQStreamClosed) {
		t.Errorf("Run = %v, must not classify ctx cancel as ErrIQStreamClosed", got)
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
	setActiveSystem(d, "Sys")

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

// TestHandleProgressRepeatedSameFreqIsIdempotent pins the issue-#402
// live-acquisition fix: a HuntProgress that repeats the same (system,
// frequency) the active pipeline is already decoding must NOT rebuild
// the pipeline or flush the DDC. On a single-candidate system the
// supervisor re-attempts every dwell (~3s); a rebuild-on-every-attempt
// loop flushed the symbol clock + decimation filter each time and the
// live stream never acquired FSW ("no FSW hits in chunk" cycling with
// "pipeline configured"). Replay never exercised this because it never
// re-hunts.
func TestHandleProgressRepeatedSameFreqIsIdempotent(t *testing.T) {
	saved := factories
	builds := 0
	var rec *recordingPipeline
	factories = map[trunking.Protocol]PipelineFactory{
		trunking.ProtocolP25: func(PipelineOptions) (ProtocolPipeline, error) {
			builds++
			rec = &recordingPipeline{}
			return rec, nil
		},
	}
	t.Cleanup(func() { factories = saved })

	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000,
		Systems: []trunking.System{{
			Name: "Sys", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{851_012_500},
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	prog := trunking.HuntProgress{System: "Sys", AttemptedFreqHz: 851_012_500}
	d.handleProgress(prog) // first attempt: builds
	first := rec
	d.handleProgress(prog) // re-hunt same freq: must be a no-op
	d.handleProgress(prog)

	if builds != 1 {
		t.Errorf("pipeline builds = %d, want 1 (repeated same-freq retune must not rebuild)", builds)
	}
	if d.active != first {
		t.Errorf("active pipeline was swapped on a repeated same-freq retune")
	}
	if first.closes != 0 {
		t.Errorf("active pipeline Close calls = %d, want 0 (must not tear down on same-freq retune)", first.closes)
	}

	// A genuine retune to a different frequency must still rebuild.
	d.handleProgress(trunking.HuntProgress{System: "Sys", AttemptedFreqHz: 851_025_000})
	if builds != 2 {
		t.Errorf("pipeline builds = %d after a real retune, want 2", builds)
	}
	if first.closes != 1 {
		t.Errorf("previous pipeline Close calls = %d after a real retune, want 1", first.closes)
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

func TestTETRAFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 412_062_500, SampleRateHz: 144_000,
		// Smoke test only — set a placeholder colour code so the
		// connector doesn't warn about zero colour code on the
		// default non-BSCH channel under the new ChannelCodingOn
		// default.
		System: trunking.System{
			Name: "Smoke", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_062_500},
			TETRAColourCode: 0x1,
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p.Process(make([]complex64, 14400))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestTETRAFactoryEnablesChannelCodingFromSystem: when the
// trunking.System carries a non-zero TETRAColourCode, the factory
// must flip the underlying ControlChannel into ChannelCodingOn and
// apply the colour code + expected channel from config.
func TestTETRAFactoryEnablesChannelCodingFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 412_062_500,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_062_500},
			TETRAColourCode: 0x12345,
			TETRAChannel:    "sch/f",
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	tp := p.(*tetraPipeline)
	if got := tp.cc.ChannelCoding(); got != tetra.ChannelCodingOn {
		t.Errorf("ChannelCoding = %v, want ChannelCodingOn", got)
	}
	if got := tp.cc.ExpectedChannel(); got != tetra.ChannelSCHF {
		t.Errorf("ExpectedChannel = %v, want ChannelSCHF", got)
	}
	if got := tp.cc.ColourCode(); got != 0x12345 {
		t.Errorf("ColourCode = %#x, want 0x12345", got)
	}
}

// TestTETRAFactoryDefaultsKeepCodingOn: empty TETRAChannelCoding
// flips the connector to ChannelCodingOn (the new default — full
// ETSI EN 300 392-2 §8.3.1 chain) so live captures decode without
// per-system YAML tweaks.
func TestTETRAFactoryDefaultsKeepCodingOn(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 412_062_500,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_062_500},
			TETRAColourCode: 0x12345,
			// TETRAChannelCoding left empty → ChannelCodingOn.
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	tp := p.(*tetraPipeline)
	if got := tp.cc.ChannelCoding(); got != tetra.ChannelCodingOn {
		t.Errorf("ChannelCoding = %v, want ChannelCodingOn", got)
	}
}

// TestTETRAFactoryExplicitOffOptsOut: tetra_channel_coding=off opts
// out of the new ChannelCodingOn default for operators feeding
// pre-stripped DSD-FME / OP25 captures.
func TestTETRAFactoryExplicitOffOptsOut(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 412_062_500,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels:    []uint32{412_062_500},
			TETRAChannelCoding: "off",
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	tp := p.(*tetraPipeline)
	if got := tp.cc.ChannelCoding(); got != tetra.ChannelCodingOff {
		t.Errorf("ChannelCoding = %v, want ChannelCodingOff", got)
	}
}

func TestP25Phase2FactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newP25Phase2Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 851_062_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newP25Phase2Pipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDMRTier3FactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newDMRTier3Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 460_000_000, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newDMRTier3Pipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDMRTier2FactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newDMRTier2Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 460_500_000, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newDMRTier2Pipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestMPT1327FactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMPT1327Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 169_212_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newMPT1327Pipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestLTRFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newLTRPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 935_012_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newLTRPipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestLTRFactoryAppliesFCSAndManchesterFromSystem: when the
// trunking.System carries LTRFCSMode + LTRManchesterMode strings,
// the factory must parse them and apply them to the underlying
// ControlChannel.
func TestLTRFactoryAppliesFCSAndManchesterFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newLTRPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 935_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolLTR,
			ControlChannels:   []uint32{935_012_500},
			LTRFCSMode:        "on",
			LTRManchesterMode: "soft",
		},
	})
	if err != nil {
		t.Fatalf("newLTRPipeline: %v", err)
	}
	lp := p.(*ltrPipeline)
	if got := lp.cc.FCSMode(); got != ltr.FCSOn {
		t.Errorf("FCSMode = %v, want FCSOn", got)
	}
	if got := lp.cc.ManchesterMode(); got != ltr.ManchesterSoft {
		t.Errorf("ManchesterMode = %v, want ManchesterSoft", got)
	}
}

// TestLTRFactoryDefaultsKeepLiveOnAirModes: empty LTRFCSMode +
// LTRManchesterMode flip the connector to FCSOn + ManchesterSoft
// (the new defaults — matches the dominant on-air encoding).
func TestLTRFactoryDefaultsKeepLiveOnAirModes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newLTRPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 935_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolLTR,
			ControlChannels: []uint32{935_012_500},
			// LTRFCSMode + LTRManchesterMode left empty.
		},
	})
	if err != nil {
		t.Fatalf("newLTRPipeline: %v", err)
	}
	lp := p.(*ltrPipeline)
	if got := lp.cc.FCSMode(); got != ltr.FCSOn {
		t.Errorf("FCSMode = %v, want FCSOn", got)
	}
	if got := lp.cc.ManchesterMode(); got != ltr.ManchesterSoft {
		t.Errorf("ManchesterMode = %v, want ManchesterSoft", got)
	}
}

// TestLTRFactoryExplicitOffOptsOut: ltr_fcs_mode=off +
// ltr_manchester_mode=off opt out of the new on-defaults for
// operators feeding pre-stripped synthesized fixtures.
func TestLTRFactoryExplicitOffOptsOut(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newLTRPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 935_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolLTR,
			ControlChannels:   []uint32{935_012_500},
			LTRFCSMode:        "off",
			LTRManchesterMode: "off",
		},
	})
	if err != nil {
		t.Fatalf("newLTRPipeline: %v", err)
	}
	lp := p.(*ltrPipeline)
	if got := lp.cc.FCSMode(); got != ltr.FCSOff {
		t.Errorf("FCSMode = %v, want FCSOff", got)
	}
	if got := lp.cc.ManchesterMode(); got != ltr.ManchesterOff {
		t.Errorf("ManchesterMode = %v, want ManchesterOff", got)
	}
}

func TestMotorolaFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMotorolaPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 851_012_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newMotorolaPipeline: %v", err)
	}
	p.Process(make([]complex64, 4800))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestMotorolaFactoryAppliesBCHFromSystem: MotorolaBCHMode = "on"
// flips the OSW decoder into BCHOn.
func TestMotorolaFactoryAppliesBCHFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMotorolaPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolMotorola,
			ControlChannels: []uint32{851_012_500},
			MotorolaBCHMode: "on",
		},
	})
	if err != nil {
		t.Fatalf("newMotorolaPipeline: %v", err)
	}
	mp := p.(*motorolaPipeline)
	if got := mp.cc.BCHMode(); got != motorola.BCHOn {
		t.Errorf("BCHMode = %v, want BCHOn", got)
	}
}

// TestMotorolaFactoryDefaultsKeepBCHOn: empty MotorolaBCHMode flips
// the connector to BCHOn (the new default — dual 64-bit
// BCH(64, 16, 11) reassembly). Live captures always need the FEC
// layer, so default-on matches operator expectations.
func TestMotorolaFactoryDefaultsKeepBCHOn(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMotorolaPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolMotorola,
			ControlChannels: []uint32{851_012_500},
		},
	})
	if err != nil {
		t.Fatalf("newMotorolaPipeline: %v", err)
	}
	mp := p.(*motorolaPipeline)
	if got := mp.cc.BCHMode(); got != motorola.BCHOn {
		t.Errorf("BCHMode = %v, want BCHOn", got)
	}
}

// TestMotorolaFactoryExplicitOffOptsOut: motorola_bch_mode=off opts
// out of the new BCHOn default for pre-stripped fixtures.
func TestMotorolaFactoryExplicitOffOptsOut(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMotorolaPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_012_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolMotorola,
			ControlChannels: []uint32{851_012_500},
			MotorolaBCHMode: "off",
		},
	})
	if err != nil {
		t.Fatalf("newMotorolaPipeline: %v", err)
	}
	mp := p.(*motorolaPipeline)
	if got := mp.cc.BCHMode(); got != motorola.BCHOff {
		t.Errorf("BCHMode = %v, want BCHOff", got)
	}
}

func TestEDACSFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newEDACSPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 866_000_000, SampleRateHz: 96_000,
	})
	if err != nil {
		t.Fatalf("newEDACSPipeline: %v", err)
	}
	p.Process(make([]complex64, 9600))
	p.Reset()
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNXDNFactoryConstructs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newNXDNPipeline(PipelineOptions{
		Bus: bus, SystemName: "Smoke",
		FrequencyHz: 851_062_500, SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("newNXDNPipeline: %v", err)
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

// TestP25Phase2FactoryAppliesTrellisFromSystem: a populated
// trunking.System with P25Phase2TrellisMode = "on" must call
// SetTrellisMode(TrellisOn) on the underlying ControlChannel.
func TestP25Phase2FactoryAppliesTrellisFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newP25Phase2Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolP25Phase2,
			ControlChannels:      []uint32{851_062_500},
			P25Phase2TrellisMode: "on",
		},
	})
	if err != nil {
		t.Fatalf("newP25Phase2Pipeline: %v", err)
	}
	pp := p.(*p25Phase2Pipeline)
	if got := pp.cc.TrellisMode(); got != p25phase2.TrellisOn {
		t.Errorf("TrellisMode = %v, want TrellisOn", got)
	}
}

// TestP25Phase2FactoryDefaultsKeepTrellisOn: empty config string
// flips the connector to TrellisOn (the new default — full
// TIA-102.AABF trellis decode).
func TestP25Phase2FactoryDefaultsKeepTrellisOn(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newP25Phase2Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolP25Phase2,
			ControlChannels: []uint32{851_062_500},
		},
	})
	if err != nil {
		t.Fatalf("newP25Phase2Pipeline: %v", err)
	}
	pp := p.(*p25Phase2Pipeline)
	if got := pp.cc.TrellisMode(); got != p25phase2.TrellisOn {
		t.Errorf("TrellisMode = %v, want TrellisOn", got)
	}
}

// TestP25Phase2FactoryExplicitOffOptsOut: p25_phase2_trellis_mode=off
// opts out of the new TrellisOn default for pre-stripped MAC-PDU
// fixtures.
func TestP25Phase2FactoryExplicitOffOptsOut(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newP25Phase2Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolP25Phase2,
			ControlChannels:      []uint32{851_062_500},
			P25Phase2TrellisMode: "off",
		},
	})
	if err != nil {
		t.Fatalf("newP25Phase2Pipeline: %v", err)
	}
	pp := p.(*p25Phase2Pipeline)
	if got := pp.cc.TrellisMode(); got != p25phase2.TrellisOff {
		t.Errorf("TrellisMode = %v, want TrellisOff", got)
	}
}

// TestNXDNFactoryAppliesViterbiFromSystem: NXDNViterbiMode = "on"
// flips the underlying ControlChannel's CAC region into ViterbiOn.
func TestNXDNFactoryAppliesViterbiFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newNXDNPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolNXDN,
			ControlChannels: []uint32{851_062_500},
			NXDNViterbiMode: "on",
		},
	})
	if err != nil {
		t.Fatalf("newNXDNPipeline: %v", err)
	}
	np := p.(*nxdnPipeline)
	if got := np.cc.ViterbiMode(); got != nxdn.ViterbiOn {
		t.Errorf("ViterbiMode = %v, want ViterbiOn", got)
	}
}

// TestNXDNFactoryAppliesDeviationFromSystem: a non-zero
// NXDNDeviationHz on the system overrides the 1800 Hz spec default
// the slicer calibrates against.
func TestNXDNFactoryAppliesDeviationFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newNXDNPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolNXDN,
			ControlChannels: []uint32{851_062_500},
			NXDNDeviationHz: 2400.0,
		},
	})
	if err != nil {
		t.Fatalf("newNXDNPipeline: %v", err)
	}
	np := p.(*nxdnPipeline)
	if np.deviationHz != 2400.0 {
		t.Errorf("deviationHz = %v, want 2400.0", np.deviationHz)
	}
}

// TestNXDNFactoryDefaultsDeviation: a zero / unset NXDNDeviationHz
// falls back to the 1800 Hz spec default per the Common Air
// Interface.
func TestNXDNFactoryDefaultsDeviation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newNXDNPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 851_062_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolNXDN,
			ControlChannels: []uint32{851_062_500},
		},
	})
	if err != nil {
		t.Fatalf("newNXDNPipeline: %v", err)
	}
	np := p.(*nxdnPipeline)
	if np.deviationHz != 1800.0 {
		t.Errorf("deviationHz = %v, want 1800.0", np.deviationHz)
	}
}

// TestEDACSFactoryAppliesBCHFromSystem: EDACSBCHMode = "on" flips
// the CCW decoder into BCHOn.
func TestEDACSFactoryAppliesBCHFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newEDACSPipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 866_000_000,
		SampleRateHz: 96_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolEDACS,
			ControlChannels: []uint32{866_000_000},
			EDACSBCHMode:    "on",
		},
	})
	if err != nil {
		t.Fatalf("newEDACSPipeline: %v", err)
	}
	ep := p.(*edacsPipeline)
	if got := ep.cc.BCHMode(); got != edacs.BCHOn {
		t.Errorf("BCHMode = %v, want BCHOn", got)
	}
}

// TestMPT1327FactoryAppliesBCHFromSystem: MPT1327BCHMode = "on"
// flips the codeword decoder into BCHOn.
func TestMPT1327FactoryAppliesBCHFromSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	p, err := newMPT1327Pipeline(PipelineOptions{
		Bus: bus, SystemName: "Test", FrequencyHz: 169_212_500,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolMPT1327,
			ControlChannels: []uint32{169_212_500},
			MPT1327BCHMode:  "on",
		},
	})
	if err != nil {
		t.Fatalf("newMPT1327Pipeline: %v", err)
	}
	mp := p.(*mpt1327Pipeline)
	if got := mp.cc.BCHMode(); got != mpt1327.BCHOn {
		t.Errorf("BCHMode = %v, want BCHOn", got)
	}
}

// recordingPower captures every IQ-power gauge update + clear so
// tests can assert the decoder's pump-side observation pipeline
// without depending on the real Prometheus collector.
type recordingPower struct {
	sets         []powerSample
	cleared      []string
	dcSets       []dcSample
	dcCleared    []string
	clipSets     []clipSample
	clipCleared  []string
	decodeOverru int
}

type powerSample struct {
	system string
	dbfs   float64
}

type dcSample struct {
	system  string
	ratioDb float64
}

type clipSample struct {
	system string
	ratio  float64
}

func (r *recordingPower) RecordIQPowerDbFS(system string, dbfs float64) {
	r.sets = append(r.sets, powerSample{system, dbfs})
}
func (r *recordingPower) ClearIQPowerDbFS(system string) {
	r.cleared = append(r.cleared, system)
}
func (r *recordingPower) RecordIQDCRatioDb(system string, ratioDb float64) {
	r.dcSets = append(r.dcSets, dcSample{system, ratioDb})
}
func (r *recordingPower) ClearIQDCRatioDb(system string) {
	r.dcCleared = append(r.dcCleared, system)
}
func (r *recordingPower) RecordIQClipRatio(system string, ratio float64) {
	r.clipSets = append(r.clipSets, clipSample{system, ratio})
}
func (r *recordingPower) ClearIQClipRatio(system string) {
	r.clipCleared = append(r.clipCleared, system)
}
func (r *recordingPower) RecordDecodeOverrun() {
	r.decodeOverru++
}

// setActiveSystem stands in for a pipeline swap in tests: it sets both the
// mu-guarded activeAt (read by clearActive/IQHealth) and the atomic
// activeSystem gauge label that the forwarder's observeIQPower reads,
// exactly as handleProgress does in production (issue #402).
func setActiveSystem(d *Decoder, name string) {
	d.activeAt = name
	d.activeSystem.Store(&name)
}

// TestPumpRecordsIQPowerOnceWindowElapses feeds a known signal level
// through observeIQPower and checks the Metrics observer sees the
// expected dBFS once iqPowerWindow worth of samples have been folded in.
func TestPumpRecordsIQPowerOnceWindowElapses(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Pretend the swap has happened so the gauge has a system label.
	setActiveSystem(d, "TestSys")

	// 0.1 amplitude → |c|^2 = 0.02 mean → -16.99 dBFS, but we're
	// computing -16.99 dBFS = 10*log10(0.02). Use a chunk of 0.5
	// amplitude → |c|^2 = 0.5 → -3 dBFS for a value that survives
	// rounding clearly.
	chunk := make([]complex64, 128)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}

	// First observation primes pwWindowAt — no record yet.
	d.observeIQPower(chunk)
	if len(pwr.sets) != 0 {
		t.Fatalf("first observation should not record, got %v", pwr.sets)
	}

	// Force the window timer past iqPowerWindow.
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.observeIQPower(chunk)

	if len(pwr.sets) != 1 {
		t.Fatalf("after window, sets = %d, want 1", len(pwr.sets))
	}
	got := pwr.sets[0]
	if got.system != "TestSys" {
		t.Errorf("system = %q, want TestSys", got.system)
	}
	// |0.5+0.5i|^2 = 0.5 → 10*log10(0.5) = -3.01 dBFS
	if got.dbfs < -3.5 || got.dbfs > -2.5 {
		t.Errorf("dbfs = %v, want roughly -3", got.dbfs)
	}
	// A mid-scale chunk is nowhere near the ADC rail → no clipping.
	if len(pwr.clipSets) != 1 {
		t.Fatalf("after window, clipSets = %d, want 1", len(pwr.clipSets))
	}
	if r := pwr.clipSets[0].ratio; r != 0 {
		t.Errorf("clip_ratio = %v, want 0 for a mid-scale chunk", r)
	}
}

// TestPumpRecordsIQClipRatioWhenRailed feeds a window of rail-pinned
// samples and asserts the new clip-ratio gauge reports ~1.0 — the
// front-end overload failure mode behind issue #402 that the RMS
// iq_power_dbfs gauge averages away.
func TestPumpRecordsIQClipRatioWhenRailed(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")

	// Both components at full scale (the ADC rail) on every sample.
	chunk := make([]complex64, 128)
	for i := range chunk {
		chunk[i] = complex(1.0, 1.0)
	}

	d.observeIQPower(chunk) // primes pwWindowAt, no record yet
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.observeIQPower(chunk)

	if len(pwr.clipSets) != 1 {
		t.Fatalf("after window, clipSets = %d, want 1", len(pwr.clipSets))
	}
	got := pwr.clipSets[0]
	if got.system != "TestSys" {
		t.Errorf("system = %q, want TestSys", got.system)
	}
	if got.ratio < 0.999 {
		t.Errorf("clip_ratio = %v, want ~1.0 for a fully-railed window", got.ratio)
	}
}

// TestPumpRecordsIQDCRatioPureDC drives observeIQPower with a
// constant-IQ chunk (all energy in the DC bin) and asserts the new
// DC-ratio gauge reports ~0 dB — the smoking-gun shape of the RTL-SDR
// R820T2 DC-spike-on-channel failure mode behind issue #402.
func TestPumpRecordsIQDCRatioPureDC(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")

	chunk := make([]complex64, 128)
	for i := range chunk {
		chunk[i] = complex(0.3, 0.4) // constant ⇒ mean(IQ) = sample, |mean|² = |sample|²
	}

	d.observeIQPower(chunk)
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.observeIQPower(chunk)

	if len(pwr.dcSets) != 1 {
		t.Fatalf("dcSets = %d, want 1", len(pwr.dcSets))
	}
	got := pwr.dcSets[0]
	if got.system != "TestSys" {
		t.Errorf("system = %q, want TestSys", got.system)
	}
	// All sample energy is in the mean → DC power == total power → 0 dB.
	if got.ratioDb < -0.5 || got.ratioDb > 0.5 {
		t.Errorf("dc_ratio_db = %v, want ~0 (DC-dominated)", got.ratioDb)
	}
}

// TestPumpRecordsIQDCRatioCleanSignal drives the observer with a
// zero-mean signal (a half-rate complex sinusoid) and asserts the
// DC-ratio gauge reports a very negative value — the shape of a clean
// off-channel capture, which serves as the regression guard against
// false-positive DC alarms.
func TestPumpRecordsIQDCRatioCleanSignal(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")

	chunk := make([]complex64, 128)
	for i := range chunk {
		// Alternating ±(0.5+0.5i) — pure tone at fs/2, sample mean is 0.
		if i%2 == 0 {
			chunk[i] = complex(0.5, 0.5)
		} else {
			chunk[i] = complex(-0.5, -0.5)
		}
	}

	d.observeIQPower(chunk)
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.observeIQPower(chunk)

	if len(pwr.dcSets) != 1 {
		t.Fatalf("dcSets = %d, want 1", len(pwr.dcSets))
	}
	got := pwr.dcSets[0]
	// Mean(IQ) = 0 → DC power = 0 → DC dBFS clamped at the -120 dBFS
	// epsilon floor (10·log10(1e-12)). Total dBFS ≈ -3 (|0.5+0.5i|² = 0.5).
	// ratio = dc - total ≈ -117 dB. Assert well below the
	// iqDCRatioWarnDb threshold; the exact value is sensitive to the
	// epsilon floor and not interesting.
	if got.ratioDb > -60 {
		t.Errorf("dc_ratio_db = %v, want very negative (zero-mean signal)", got.ratioDb)
	}
}

// TestClearActiveClearsDCRatio asserts that tearing the active pipeline
// down drops the DC-ratio gauge series alongside the IQ-power gauge —
// so stale ratios don't linger after a retune to a different system.
func TestClearActiveClearsDCRatio(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")

	d.mu.Lock()
	d.clearActiveLocked()
	d.mu.Unlock()

	if len(pwr.cleared) != 1 || pwr.cleared[0] != "TestSys" {
		t.Errorf("cleared (power) = %v, want [TestSys]", pwr.cleared)
	}
	if len(pwr.dcCleared) != 1 || pwr.dcCleared[0] != "TestSys" {
		t.Errorf("cleared (dc) = %v, want [TestSys]", pwr.dcCleared)
	}
}

// buildP25CCDibits assembles a P25 Phase 1 control-channel dibit
// stream — a warmup pattern + `repeats` × (FSW + NID + trellis-
// encoded TSBK + idle gap) + trailer — mirroring the integration
// suite's fixture so the C4FM modulator + receiver chain can lock.
func buildP25CCDibits(nac uint16, repeats int) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, p25phase1.FrameSyncWord[:]...)
	nidBits := p25phase1.EncodeNIDBits(nac, p25phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	tsbk := p25phase1.AssembleTSBK(p25phase1.TSBK{
		LB: true, Opcode: p25phase1.OpRFSSStatusBroadcast,
	})
	frame = append(frame, p25phase1.EncodeTSBKChannel(tsbk)...)
	// Interleave the P25 status symbols a real transmitter inserts (one
	// per 36 dibits); the receiver must strip them to decode the NID.
	frame = p25phase1.InjectControlStatusSymbols(frame)

	out := make([]uint8, 0, 200+repeats*(len(frame)+50)+100)
	for i := 0; i < 200; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// TestDecoderLocksP25Phase1ThroughWidebandDDC is the regression
// proof for issue #275: a P25 Phase 1 control channel synthesized at
// the channelized 48 kHz rate, upsampled to a raw 2.048 MHz SDR
// stream, and fed through the real Decoder must still lock — the
// down-converter has to channelize the wideband IQ back to ~48 kHz
// before the production newP25Phase1Pipeline can correlate the FSW.
//
// Before the DDC was wired in, the receiver saw the full 2.048 MHz
// stream (≈427 samples per symbol) and never locked on-air.
func TestDecoderLocksP25Phase1ThroughWidebandDDC(t *testing.T) {
	const (
		nac          = 0x293
		controlFreq  = 851_000_000
		sdrRateHz    = 2_048_000.0
		narrowRateHz = 48_000.0
		deviationHz  = 1800.0
		frameRepeats = 30
	)

	// Synthesize the control channel at the narrowband 48 kHz rate,
	// then upsample (L/M = 128/3) to the raw SDR rate so the
	// Decoder's down-converter has a genuine wideband chunk to
	// channelize back down.
	dibits := buildP25CCDibits(nac, frameRepeats)
	narrow := demod.ModulateP25C4FM(dibits, narrowRateHz, deviationHz)
	wide := dsp.NewResampler(128, 3, 8, 8.0).Process(nil, narrow)

	bus := events.NewBus(256)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{},
		Systems: []trunking.System{{
			Name: "WBSys", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{controlFreq},
		}},
		SampleRateHz: sdrRateHz,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	sub := bus.Subscribe()
	defer sub.Close()

	// Build the real P25 Phase 1 pipeline (production factory). This
	// is what (re)builds the down-converter for the P25 channel rate.
	d.handleProgress(trunking.HuntProgress{
		System: "WBSys", AttemptedFreqHz: controlFreq,
	})
	if d.active == nil {
		t.Fatalf("handleProgress did not install a pipeline")
	}
	// The down-converter must hand the pipeline factory exactly
	// 48 kHz — the rate every P25 receiver is matched-filter-tuned
	// for.
	if d.pipelineRateHz != narrowRateHz {
		t.Fatalf("pipelineRateHz = %v, want %v", d.pipelineRateHz, narrowRateHz)
	}

	// Pump the wideband stream through in chunks the size of a real
	// RTL-SDR USB transfer (16 KiB = 8192 complex samples ≈ 19 P25
	// symbols after the down-converter), draining the bus after each
	// so a full subscriber buffer can't drop the cc.locked event.
	const chunk = 8_192
	var locked bool
	var lockState p25phase1.LockState
	for off := 0; off < len(wide) && !locked; off += chunk {
		end := off + chunk
		if end > len(wide) {
			end = len(wide)
		}
		d.pump(wide[off:end])
		for drained := false; !drained; {
			select {
			case ev := <-sub.C:
				if ev.Kind != events.KindCCLocked {
					continue
				}
				if ls, ok := ev.Payload.(p25phase1.LockState); ok {
					lockState = ls
					locked = true
				}
			default:
				drained = true
			}
		}
	}

	if !locked {
		t.Fatalf("no cc.locked event after pumping %d wideband samples through the DDC", len(wide))
	}
	if lockState.NAC != nac {
		t.Errorf("LockState.NAC = %#x, want %#x", lockState.NAC, nac)
	}
	if lockState.FrequencyHz != controlFreq {
		t.Errorf("LockState.FrequencyHz = %d, want %d", lockState.FrequencyHz, controlFreq)
	}
}

// TestDecoderLocksP25Phase1WithLOOffset is the live↔replay parity guard for
// issue #402: it proves the DC-spike-avoidance LO offset actually recovers a
// channel the daemon deliberately tuned OFF-centre, and — in the negative
// sub-case — that without the matching DDC offset the same off-centre channel
// never locks. This mirrors the production wiring: the daemon tunes the
// hardware LO `O` Hz below the logical channel (so the wanted channel sits at
// +O in the delivered baseband) and builds the DDC with Options.LOOffsetHz=O
// to mix it back to 0. Here we synthesize a wideband stream with the channel
// shifted up to +O and feed it through a real Decoder.
func TestDecoderLocksP25Phase1WithLOOffset(t *testing.T) {
	const (
		nac          = 0x293
		controlFreq  = 851_000_000
		sdrRateHz    = 2_048_000.0
		narrowRateHz = 48_000.0
		deviationHz  = 1800.0
		frameRepeats = 30
		offsetHz     = sdrRateHz / 4 // 512 kHz — the production auto value
	)

	// Centred wideband control channel, then frequency-shifted UP to +offsetHz
	// to simulate an LO tuned offsetHz below the channel. NCO.Mix applies
	// exp(-j2π·offsetHz·n/Fs), so NewNCO(-offsetHz) multiplies by
	// exp(+j2π·offsetHz·n/Fs) — i.e. shifts the centred channel up to +offsetHz.
	dibits := buildP25CCDibits(nac, frameRepeats)
	narrow := demod.ModulateP25C4FM(dibits, narrowRateHz, deviationHz)
	centred := dsp.NewResampler(128, 3, 8, 8.0).Process(nil, narrow)
	offChannel := dsp.NewNCO(-offsetHz, sdrRateHz).Mix(nil, centred)

	// pumpUntilLock builds a Decoder with the given LO offset, installs the real
	// P25 pipeline, pumps the off-channel stream, and reports whether it locked.
	pumpUntilLock := func(t *testing.T, loOffsetHz float64) (bool, p25phase1.LockState) {
		t.Helper()
		bus := events.NewBus(256)
		defer bus.Close()
		d, err := New(Options{
			Bus: bus, IQ: &fakeIQSource{},
			Systems: []trunking.System{{
				Name: "WBSys", Protocol: trunking.ProtocolP25,
				ControlChannels: []uint32{controlFreq},
			}},
			SampleRateHz: sdrRateHz,
			LOOffsetHz:   loOffsetHz,
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer d.Close()
		sub := bus.Subscribe()
		defer sub.Close()

		d.handleProgress(trunking.HuntProgress{System: "WBSys", AttemptedFreqHz: controlFreq})
		if d.active == nil {
			t.Fatalf("handleProgress did not install a pipeline")
		}

		const chunk = 8_192
		var lockState p25phase1.LockState
		locked := false
		for off := 0; off < len(offChannel) && !locked; off += chunk {
			end := off + chunk
			if end > len(offChannel) {
				end = len(offChannel)
			}
			d.pump(offChannel[off:end])
			for drained := false; !drained; {
				select {
				case ev := <-sub.C:
					if ev.Kind != events.KindCCLocked {
						continue
					}
					if ls, ok := ev.Payload.(p25phase1.LockState); ok {
						lockState = ls
						locked = true
					}
				default:
					drained = true
				}
			}
		}
		return locked, lockState
	}

	// Positive: with the matching DDC offset the off-channel signal is mixed
	// back to baseband and locks at the LOGICAL frequency (the offset is
	// invisible above the DDC).
	t.Run("offset_locks", func(t *testing.T) {
		locked, ls := pumpUntilLock(t, offsetHz)
		if !locked {
			t.Fatalf("no cc.locked with LOOffsetHz=%v — DC-spike-avoidance offset failed to recover the off-channel signal", offsetHz)
		}
		if ls.NAC != nac {
			t.Errorf("LockState.NAC = %#x, want %#x", ls.NAC, nac)
		}
		if ls.FrequencyHz != controlFreq {
			t.Errorf("LockState.FrequencyHz = %d, want %d (logical freq, offset must be invisible above the DDC)", ls.FrequencyHz, controlFreq)
		}
	})

	// Negative / regression guard: without the offset the channel stays at
	// +offsetHz, outside the 48 kHz channel the DDC keeps, and never locks.
	// This is what proves the offset — not luck — is what fixes issue #402.
	t.Run("no_offset_does_not_lock", func(t *testing.T) {
		locked, _ := pumpUntilLock(t, 0)
		if locked {
			t.Fatalf("locked with LOOffsetHz=0 on an off-channel signal — the negative control is invalid (the channel should be unrecoverable without the DDC offset)")
		}
	})
}

// TestDecoderAutotuneMeasuresAndCorrectsOffset proves the autotune feedback
// loop end-to-end on the live decode path: a P25 control channel synthesized
// with a known carrier-frequency error δ is recovered by the receiver, and the
// decoder's autotune tracker folds the receiver's residual AFC estimate into
// its running average with the SIGN that lets ensureDownconverterLocked cancel
// the error at the next acquisition (mix offset = loOffsetHz + average). If the
// receiver reported the offset with the opposite sign, the average would have
// the wrong sign and this test would catch it.
func TestDecoderAutotuneMeasuresAndCorrectsOffset(t *testing.T) {
	const (
		nac          = 0x293
		controlFreq  = 851_000_000
		sdrRateHz    = 2_048_000.0
		narrowRateHz = 48_000.0
		deviationHz  = 1800.0
		frameRepeats = 40
		carrierErrHz = 600.0 // the dongle's crystal error
	)

	// Centred control channel, then shifted UP to +carrierErrHz to emulate a
	// crystal error that leaves the wanted carrier carrierErrHz above DC in the
	// delivered baseband. NewNCO(-f).Mix multiplies by exp(+j2π·f·n/Fs), i.e.
	// shifts the centred channel up to +f — the same construction the LO-offset
	// test uses.
	dibits := buildP25CCDibits(nac, frameRepeats)
	narrow := demod.ModulateP25C4FM(dibits, narrowRateHz, deviationHz)
	centred := dsp.NewResampler(128, 3, 8, 8.0).Process(nil, narrow)
	offChannel := dsp.NewNCO(-carrierErrHz, sdrRateHz).Mix(nil, centred)

	bus := events.NewBus(256)
	defer bus.Close()
	mgr := autotune.NewManager("test-dongle", nil)
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{},
		Systems: []trunking.System{{
			Name: "ATSys", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{controlFreq},
		}},
		SampleRateHz: sdrRateHz,
		Autotune:     mgr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	d.handleProgress(trunking.HuntProgress{System: "ATSys", AttemptedFreqHz: controlFreq})
	if d.active == nil {
		t.Fatalf("handleProgress did not install a pipeline")
	}
	// With no measurements yet the DDC must be built with zero correction —
	// byte-identical to the non-autotune path.
	if d.autotuneApplied != 0 {
		t.Fatalf("autotuneApplied = %d before any measurement, want 0", d.autotuneApplied)
	}

	const chunk = 8_192
	locked := false
	for off := 0; off < len(offChannel); off += chunk {
		end := off + chunk
		if end > len(offChannel) {
			end = len(offChannel)
		}
		d.pump(offChannel[off:end])
		if !locked {
			for drained := false; !drained; {
				select {
				case ev := <-sub.C:
					if ev.Kind == events.KindCCLocked {
						// Simulate the Run loop's lock bookkeeping so
						// sampleAutotuneLocked starts folding measurements.
						d.locked.Store(true)
						locked = true
					}
				default:
					drained = true
				}
			}
		}
	}
	if !locked {
		t.Fatalf("control channel never locked with a %.0f Hz carrier error", carrierErrHz)
	}

	avg := mgr.AverageError()
	if avg == 0 {
		t.Fatalf("autotune recorded no measurement after lock")
	}
	// The measured average must approximate the true carrier error and carry
	// the sign the DDC correction expects: ensureDownconverterLocked mixes by
	// (loOffsetHz + avg), and the carrier sits at +carrierErrHz, so avg must be
	// ≈ +carrierErrHz for the next acquisition to pull it to DC.
	const tol = 200.0 // Hz — CoarseAFC estimate + symbol-rate rounding slack
	if diff := float64(avg) - carrierErrHz; diff < -tol || diff > tol {
		t.Fatalf("autotune average = %d Hz, want %.0f ± %.0f Hz (wrong sign means the DDC correction would worsen the error)", avg, carrierErrHz, tol)
	}
}

// TestClearActiveClearsIQPowerSeries: when the decoder swaps pipelines
// it must drop the gauge series for the system the previous pipeline
// owned, so stale dBFS doesn't outlive the active system.
func TestClearActiveClearsIQPowerSeries(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")
	d.clearActive()
	if len(pwr.cleared) != 1 || pwr.cleared[0] != "TestSys" {
		t.Errorf("cleared = %v, want [TestSys]", pwr.cleared)
	}
}

// TestDiagnoseIQ exercises every branch of the cchunt.failed cause
// classifier so the one-line hint stays in lock-step with the live
// pump's thresholds.
func TestDiagnoseIQ(t *testing.T) {
	cases := []struct {
		name       string
		diag       trunking.HuntDiagnostics
		haveWindow bool
		wantSubstr string
	}{
		{
			name:       "no pipeline",
			diag:       trunking.HuntDiagnostics{PipelineActive: false},
			haveWindow: true,
			wantSubstr: "no live decode pipeline",
		},
		{
			name:       "partial window",
			diag:       trunking.HuntDiagnostics{PipelineActive: true},
			haveWindow: false,
			wantSubstr: "too short to gauge",
		},
		{
			name:       "low power",
			diag:       trunking.HuntDiagnostics{PipelineActive: true, IQPowerDbFS: iqLowPowerThresholdDbFS - 1},
			haveWindow: true,
			wantSubstr: "IQ level very low",
		},
		{
			name:       "clipping",
			diag:       trunking.HuntDiagnostics{PipelineActive: true, IQPowerDbFS: -10, IQClipRatio: iqClipWarnRatio + 0.01},
			haveWindow: true,
			wantSubstr: "front-end overload",
		},
		{
			name:       "dc spike",
			diag:       trunking.HuntDiagnostics{PipelineActive: true, IQPowerDbFS: -10, IQDCRatioDb: iqDCRatioWarnDb + 1},
			haveWindow: true,
			wantSubstr: "DC spike",
		},
		{
			name:       "healthy but no lock",
			diag:       trunking.HuntDiagnostics{PipelineActive: true, IQPowerDbFS: -30, IQClipRatio: 0, IQDCRatioDb: -25},
			haveWindow: true,
			wantSubstr: "no control-channel lock",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnoseIQ(tc.diag, tc.haveWindow)
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("diagnoseIQ(%s) = %q, want substring %q", tc.name, got, tc.wantSubstr)
			}
		})
	}
}

// TestIQHealthUnknownSystem: with no observations for a system, IQHealth
// reports ok=false so the supervisor falls back to its "no IQ observed"
// diagnosis.
func TestIQHealthUnknownSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := d.IQHealth("Nope"); ok {
		t.Fatal("IQHealth(unknown) ok = true, want false")
	}
}

// TestIQHealthPartialSamples: once IQ has flowed for the active system
// but no full power window has elapsed, IQHealth reports the sample
// count with the "dwell too short" hint.
func TestIQHealthPartialSamples(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Stand up an active pipeline for "Sys" and fold one chunk of IQ
	// through the power observer without letting a window elapse.
	d.active = &recordingPipeline{}
	setActiveSystem(d, "Sys")
	d.observeIQPower(make([]complex64, 1024))

	diag, ok := d.IQHealth("Sys")
	if !ok {
		t.Fatal("IQHealth(Sys) ok = false, want true")
	}
	if !diag.IQObserved || diag.IQSamples != 1024 {
		t.Errorf("diag = %+v, want IQObserved with 1024 samples", diag)
	}
	if !diag.PipelineActive {
		t.Errorf("diag.PipelineActive = false, want true")
	}
	if !strings.Contains(diag.Diagnosis, "too short to gauge") {
		t.Errorf("diag.Diagnosis = %q, want 'too short to gauge'", diag.Diagnosis)
	}
}

// TestForwardIQDecouplesIngestFromDecode pins the issue-#402 live-path
// fix: the forwarder drains the SDR channel into the decode queue,
// preserves order, drops only when the queue is full (and counts each
// drop as a decode overrun, distinct from a driver underrun), and closes
// the queue when the source ends.
func TestForwardIQDecouplesIngestFromDecode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Pre-load every chunk and close the source. The forwarder's enqueue is
	// non-blocking (it drops on a full queue), so it runs to completion with
	// no consumer: the first two chunks land in the cap-2 queue and the next
	// three are dropped. Waiting for it to finish before draining keeps the
	// queued set and drop count deterministic (a concurrent drain would race
	// the last enqueue). Tag each chunk by its first sample to check order.
	in := make(chan []complex64, 5)
	out := make(chan []complex64, 2)
	mk := func(tag float32) []complex64 { return []complex64{complex(tag, 0)} }
	for i := 0; i < 5; i++ {
		in <- mk(float32(i))
	}
	close(in)

	done := make(chan struct{})
	go func() { d.forwardIQ(context.Background(), in, out); close(done) }()
	<-done // forwarder has processed all 5 chunks and closed out

	var got []float32
	for c := range out {
		got = append(got, real(c[0]))
	}
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("queued = %v, want [0 1] (first two, in order)", got)
	}
	if pwr.decodeOverru != 3 {
		t.Errorf("decode overruns = %d, want 3", pwr.decodeOverru)
	}
}

// TestForwardIQCopiesDriverBuffer pins the correctness fix for the
// interaction between the decode queue (issue #402) and the SDR driver's
// small reuse ring (issue #489): the forwarder must copy each chunk into a
// decoder-owned buffer before queueing it, so a later driver delivery that
// recycles the ring slot cannot overwrite a chunk still awaiting decode.
func TestForwardIQCopiesDriverBuffer(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	src := []complex64{complex(1, 2), complex(3, 4)}
	in := make(chan []complex64, 1)
	out := make(chan []complex64, 1)
	in <- src
	close(in)

	done := make(chan struct{})
	go func() { d.forwardIQ(context.Background(), in, out); close(done) }()
	<-done

	queued := <-out
	if len(queued) != len(src) {
		t.Fatalf("queued len = %d, want %d", len(queued), len(src))
	}
	if &queued[0] == &src[0] {
		t.Fatal("forwarder queued the driver's slice itself, not a copy — driver-ring reuse (#489) could corrupt IQ still queued for decode (#402)")
	}
	// Simulate the driver overwriting its ring slot after the buffer was
	// handed back; the queued copy must be unaffected.
	src[0] = complex(99, 99)
	if queued[0] != complex(1, 2) {
		t.Errorf("queued chunk changed when the source ring slot was overwritten: got %v, want (1+2i)", queued[0])
	}
}

// TestIQHealthConcurrentWithForwarder exercises observeIQPower (forwarder
// goroutine) and IQHealth (cchunt-supervisor goroutine) concurrently so
// the race detector pins the healthMu/activeSystem synchronisation that
// keeps the issue-#489 IQ-health snapshot safe now that observation runs
// off the decode goroutine (issue #402).
func TestIQHealthConcurrentWithForwarder(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pwr := &recordingPower{}
	d, err := New(Options{Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Metrics: pwr})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "Sys")
	d.active = &recordingPipeline{}
	// Force the first observed chunk to publish a window snapshot so the
	// lastHealth* write path runs concurrently with IQHealth's reads.
	d.pwWindowAt = time.Now().Add(-2 * iqPowerWindow)

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan []complex64, 8)
	out := make(chan []complex64, 8)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); d.forwardIQ(ctx, in, out) }()
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			select {
			case in <- make([]complex64, 256):
			case <-ctx.Done():
				return
			}
		}
	}()

	for i := 0; i < 2000; i++ {
		_, _ = d.IQHealth("Sys")
	}
	cancel()
	wg.Wait()
}

// TestPumpConjugatesWhenConfigured verifies the per-device spectrum-
// inversion option (Options.Conjugate / config iq_invert): pump must
// negate Q on every raw IQ sample before handing it to the pipeline, so
// a spectrum-inverted / I-Q-swapped front end decodes (issue #264 /
// #553). At SampleRateHz == the P25 channel rate the DDC is pass-through,
// so the recording pipeline sees exactly the conjugated input.
func TestPumpConjugatesWhenConfigured(t *testing.T) {
	rec := withRecordingFactory(t, trunking.ProtocolP25)
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, Tuner: fakeTuner{},
		SampleRateHz: DDCTargetRateHz, Conjugate: true,
		Systems: []trunking.System{{
			Name: "Sys", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{851_012_500},
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d.handleProgress(trunking.HuntProgress{System: "Sys", AttemptedFreqHz: 851_012_500})

	in := []complex64{complex(1, 2), complex(-3, 4), complex(0.5, -0.25)}
	d.pump(append([]complex64(nil), in...))

	if len(rec.processChunks) == 0 {
		t.Fatalf("recording pipeline received no chunk")
	}
	got := rec.processChunks[0]
	if len(got) != len(in) {
		t.Fatalf("chunk len = %d, want %d", len(got), len(in))
	}
	for i := range in {
		want := complex(real(in[i]), -imag(in[i]))
		if got[i] != want {
			t.Errorf("sample %d = %v, want conjugated %v", i, got[i], want)
		}
	}
}

// TestPumpNoConjugateByDefault: without Conjugate, pump must pass raw IQ
// through unchanged (pass-through DDC at the channel rate).
func TestPumpNoConjugateByDefault(t *testing.T) {
	rec := withRecordingFactory(t, trunking.ProtocolP25)
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{
		Bus: bus, IQ: &fakeIQSource{}, Tuner: fakeTuner{},
		SampleRateHz: DDCTargetRateHz,
		Systems: []trunking.System{{
			Name: "Sys", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{851_012_500},
		}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d.handleProgress(trunking.HuntProgress{System: "Sys", AttemptedFreqHz: 851_012_500})
	in := []complex64{complex(1, 2), complex(-3, 4)}
	d.pump(append([]complex64(nil), in...))
	if len(rec.processChunks) == 0 {
		t.Fatalf("recording pipeline received no chunk")
	}
	for i, got := range rec.processChunks[0] {
		if got != in[i] {
			t.Errorf("sample %d = %v, want unchanged %v", i, got, in[i])
		}
	}
}
