package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// replayVoiceSerial is the single voice device serial the replay rig exposes.
const replayVoiceSerial = "replay-voice:1"

// replayVoiceSource is the same-carrier voice device the replay rig binds every
// grant to. It streams the control carrier's own channelized IQ (fed from
// siglab's OnChannelIQ tap) to each following voice chain — the offline analogue
// of ccdecoder.CCVoiceSource, which the daemon uses for the same job. Because a
// conventional capture decodes ONE carrier, every grant belongs to it, so the
// source tunes anywhere (CanTune always true).
//
// Backpressure: push blocks on each subscriber's channel, so a fast file replay
// is paced to the (real-time-cost) voice chain instead of overrunning it.
type replayVoiceSource struct {
	rateBits atomic.Uint64 // float64 DDC rate, learned from the first push

	mu   sync.Mutex
	next int
	subs map[int]*replayVoiceSub
}

type replayVoiceSub struct {
	ch   chan []complex64
	done chan struct{}
}

func newReplayVoiceSource(rateHz float64) *replayVoiceSource {
	s := &replayVoiceSource{subs: make(map[int]*replayVoiceSub)}
	s.rateBits.Store(math.Float64bits(rateHz))
	return s
}

func (s *replayVoiceSource) Serial() string { return replayVoiceSerial }

// SetCenterFreq is a no-op: the carrier is already tuned by the replayed capture.
func (s *replayVoiceSource) SetCenterFreq(uint32) error { return nil }

// CanTune accepts any grant — a replay decodes a single carrier, so every grant
// it produces belongs to that carrier.
func (s *replayVoiceSource) CanTune(uint32) bool { return true }

func (s *replayVoiceSource) rate() float64 { return math.Float64frombits(s.rateBits.Load()) }

func (s *replayVoiceSource) SampleRateHz() uint32       { return uint32(s.rate() + 0.5) }
func (s *replayVoiceSource) SampleRateExactHz() float64 { return s.rate() }

// StreamIQ registers a subscriber for the carrier IQ until ctx ends (the composer
// cancels it at call end).
func (s *replayVoiceSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	sub := &replayVoiceSub{ch: make(chan []complex64, 16), done: make(chan struct{})}
	s.mu.Lock()
	id := s.next
	s.next++
	s.subs[id] = sub
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		delete(s.subs, id)
		s.mu.Unlock()
		close(sub.done) // unblocks any in-flight push targeting this sub
	}()
	return sub.ch, nil
}

// push fans one channelized IQ chunk to every following voice chain, copying it
// (siglab reuses the backing array) and blocking until each consumer takes it
// (backpressure — see the type comment). A sub cancelled mid-send is skipped via
// its done channel rather than deadlocking.
func (s *replayVoiceSource) push(iq []complex64, rateHz float64) {
	if rateHz > 0 {
		s.rateBits.Store(math.Float64bits(rateHz))
	}
	s.mu.Lock()
	if len(s.subs) == 0 {
		s.mu.Unlock()
		return
	}
	subs := make([]*replayVoiceSub, 0, len(s.subs))
	for _, sub := range s.subs {
		subs = append(subs, sub)
	}
	s.mu.Unlock()
	cp := make([]complex64, len(iq))
	copy(cp, iq)
	for _, sub := range subs {
		select {
		case sub.ch <- cp:
		case <-sub.done:
		}
	}
}

// replayVoiceDevices resolves the composer's device lookups to the single replay
// voice source.
type replayVoiceDevices struct{ src *replayVoiceSource }

func (d replayVoiceDevices) FindBySerial(serial string) composer.IQSource {
	if serial == d.src.Serial() {
		return d.src
	}
	return nil
}

// replayVoiceRig wires a trunking engine + voice composer + recorder onto a
// siglab decode's grant bus and channelized-IQ tap, so `replay -record-voice`
// turns a capture's grants into .wav/.raw/.json recordings using the exact
// production voice path (VoicePool.Bind → composer voice chains → recorder).
type replayVoiceRig struct {
	bus      *events.Bus
	src      *replayVoiceSource
	engine   *trunking.Engine
	composer *composer.Composer
	recorder *voice.Recorder
	cancel   context.CancelFunc
	hangtime time.Duration
	wg       sync.WaitGroup
}

// setupReplayVoice builds and starts the engine/composer/recorder rig writing to
// outDir. rateHz is the expected channelized (DDC output) rate; the source also
// refines it from the first IQ chunk.
//
// An empty outDir runs the recorder in decode-only mode (no files are written)
// — the `-audio-out`-without-`-record-voice` shape, where the caller only wants
// the live PCM stream. audio, when non-nil, receives every decoded call's PCM
// as a continuous raw s16le stream (issue #314): it is fanned into the
// composer's main sink (analog FM chains write PCM there) AND wired as the
// recorder's decoded-PCM tap (digital protocols emit raw vocoder frames that
// only the recorder decodes — without the tap, digital audio never reaches a
// WritePCM-only sink; the same wiring the daemon uses for live browser audio).
func setupReplayVoice(outDir string, rateHz float64, hangtime time.Duration, audio *pcmStreamWriter, log *slog.Logger) (*replayVoiceRig, error) {
	bus := events.NewBus(1024)
	src := newReplayVoiceSource(rateHz)

	pool := trunking.NewVoicePool([]*trunking.VoiceDevice{{Tuner: src, Serial: src.Serial()}})
	engine, err := trunking.NewEngine(trunking.EngineOptions{
		Bus:        bus,
		Log:        log,
		VoicePool:  pool,
		Talkgroups: trunking.NewTalkgroupDB(),
		ScanMode:   trunking.ScanModeAll,
	})
	if err != nil {
		bus.Close()
		return nil, fmt.Errorf("replay voice: engine: %w", err)
	}
	// An empty outDir puts the recorder in its decode-only mode: every call
	// still builds its per-protocol vocoder (digital voice still decodes and
	// reaches the decoded-PCM tap below), it just never writes WAV/raw/json.
	rec, err := voice.NewRecorder(voice.RecorderOptions{
		Bus:                bus,
		Log:                log,
		OutDir:             outDir,
		SampleRate:         8000,
		WriteRaw:           outDir != "",
		WriteCallJSON:      outDir != "",
		VocoderForProtocol: voice.DefaultVocoderForProtocol(),
	})
	if err != nil {
		bus.Close()
		return nil, fmt.Errorf("replay voice: recorder: %w", err)
	}
	// The composer type-asserts its sink for the raw-frame / drain-coordination
	// extensions, so the multi-sink shape must be the daemon's fanoutSink (which
	// forwards them to the recorder) — a naive tee would silently drop every
	// IMBE/AMBE frame (the issue #356 failure class).
	var sink composer.PCMSink = rec
	if audio != nil {
		sink = fanoutSink{rec, audio}
		rec.SetDecodedPCMSink(audio)
	}
	comp, err := composer.New(composer.Options{
		Bus:           bus,
		Devices:       replayVoiceDevices{src: src},
		Sink:          sink,
		Engine:        engine,
		Log:           log,
		IQSampleRate:  uint32(rateHz + 0.5),
		PCMSampleRate: 8000,
		VoiceHangtime: hangtime,
	})
	if err != nil {
		bus.Close()
		return nil, fmt.Errorf("replay voice: composer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	rig := &replayVoiceRig{
		bus: bus, src: src, engine: engine, composer: comp, recorder: rec,
		cancel: cancel, hangtime: hangtime,
	}
	for _, r := range []func(context.Context) error{rec.Run, comp.Run, engine.Run} {
		run := r
		rig.wg.Add(1)
		go func() { defer rig.wg.Done(); _ = run(ctx) }()
	}
	return rig, nil
}

// onChannelIQ is the siglab tap: it forwards the channelized IQ to the voice
// source (blocking, so decode is paced to the voice chain).
func (r *replayVoiceRig) onChannelIQ(iq []complex64, rateHz float64) { r.src.push(iq, rateHz) }

// finalize lets any in-flight call reach its hangtime end (so its recording is
// finalized), then tears the rig down and flushes.
func (r *replayVoiceRig) finalize() {
	// No more IQ arrives after decode EOF; the composer's boundary tracker ends
	// the active call one hangtime later (wall clock), which publishes CallEnd and
	// finalizes the recording. Wait that out with margin before tearing down.
	time.Sleep(r.hangtime + 2*time.Second)
	_ = r.composer.Close() // cancels chains, draining tails (drain coordination)
	r.cancel()             // stop engine/composer/recorder Run loops
	r.bus.Close()
	_ = r.recorder.Close() // flushPendingFinalize writes any still-pending recording
	r.wg.Wait()
}
