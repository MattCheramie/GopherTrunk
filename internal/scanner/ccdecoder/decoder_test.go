package ccdecoder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
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
	sets    []powerSample
	cleared []string
}

type powerSample struct {
	system string
	dbfs   float64
}

func (r *recordingPower) RecordIQPowerDbFS(system string, dbfs float64) {
	r.sets = append(r.sets, powerSample{system, dbfs})
}
func (r *recordingPower) ClearIQPowerDbFS(system string) {
	r.cleared = append(r.cleared, system)
}

// TestPumpRecordsIQPowerOnceWindowElapses feeds a known signal level
// through Process and checks the Metrics observer sees the expected
// dBFS once iqPowerWindow worth of samples have been folded in.
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
	d.activeAt = "TestSys"

	// 0.1 amplitude → |c|^2 = 0.02 mean → -16.99 dBFS, but we're
	// computing -16.99 dBFS = 10*log10(0.02). Use a chunk of 0.5
	// amplitude → |c|^2 = 0.5 → -3 dBFS for a value that survives
	// rounding clearly.
	chunk := make([]complex64, 128)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}

	// First pump primes pwWindowAt — no record yet.
	d.pump(chunk)
	if len(pwr.sets) != 0 {
		t.Fatalf("first pump should not record, got %v", pwr.sets)
	}

	// Force the window timer past iqPowerWindow.
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.pump(chunk)

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
	d.activeAt = "TestSys"
	d.clearActive()
	if len(pwr.cleared) != 1 || pwr.cleared[0] != "TestSys" {
		t.Errorf("cleared = %v, want [TestSys]", pwr.cleared)
	}
}
