package widebandt2

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
)

// Per-channel IQ fan-out + same-carrier voice source for wideband-hosted
// control channels.
//
// A TETRA SCBS (and a conventional DMR repeater) keeps its voice on the SAME
// carrier the control-channel state machine decodes: other timeslots of the
// control carrier. On the single-tuner path that voice is served by
// ccdecoder.CCVoiceSource, which taps the control decoder's own channelised
// IQ. On the wideband path there was nothing equivalent — a TETRA grant on a
// wideband-hosted CC could only bind a 48 kHz wbvoice.VirtualTuner and run
// the SOLO voice chain, never the shared 4-slot per-carrier demux — which is
// why "monitor two TETRA systems on one X310" (10 Sep report) could lock both
// CCs on a role: wideband device but not follow their voice properly.
// ChannelVoiceSource closes that gap: it exposes one wideband channel's own
// 144 kHz (TETRA) tap stream through the same small interfaces
// ccdecoder.CCVoiceSource implements, so the composer routes it to the same
// per-carrier demux keyed by CarrierKey.

// channelFanout distributes copies of one channel's post-DDC IQ to zero or
// more voice subscribers. Sends are non-blocking (drop on a full buffer) so a
// slow voice consumer can never stall the wideband pump goroutine — which
// feeds EVERY channel on the dongle, not just this one. Mirrors
// ccdecoder.voiceFanout.
type channelFanout struct {
	mu     sync.Mutex
	subs   map[int]*channelSub
	next   int
	log    *slog.Logger
	serial string
	freqHz uint32
	rateHz float64
}

type channelSub struct {
	ch    chan []complex64
	drops atomic.Uint64
}

// Subscriber buffer sizing, mirroring ccdecoder's voiceTapBudget: one second
// of IQ at the tap rate assuming pessimistically small chunks, clamped so
// the slot count is never the limiter and never absurd.
const (
	channelTapSeconds             = 1.0
	channelTapAssumedChunkSamples = 32
	minChannelTapChunks           = 128
	maxChannelTapChunks           = 16384
)

func channelTapBudget(rateHz float64) int {
	if rateHz <= 0 {
		return minChannelTapChunks
	}
	depth := int(math.Ceil(channelTapSeconds * rateHz / channelTapAssumedChunkSamples))
	if depth < minChannelTapChunks {
		return minChannelTapChunks
	}
	if depth > maxChannelTapChunks {
		return maxChannelTapChunks
	}
	return depth
}

func newChannelFanout(log *slog.Logger, serial string, freqHz uint32, rateHz float64) *channelFanout {
	if log == nil {
		log = slog.Default()
	}
	return &channelFanout{subs: map[int]*channelSub{}, log: log, serial: serial, freqHz: freqHz, rateHz: rateHz}
}

// subscribe registers a subscriber and returns its IQ channel plus an
// unsubscribe func that closes the channel and reports the lifetime
// dropped-chunk count (0 when the consumer kept up).
func (f *channelFanout) subscribe() (<-chan []complex64, func() uint64) {
	sub := &channelSub{ch: make(chan []complex64, channelTapBudget(f.rateHz))}
	f.mu.Lock()
	id := f.next
	f.next++
	f.subs[id] = sub
	f.mu.Unlock()
	var once sync.Once
	return sub.ch, func() uint64 {
		once.Do(func() {
			f.mu.Lock()
			if s, ok := f.subs[id]; ok {
				delete(f.subs, id)
				close(s.ch)
			}
			f.mu.Unlock()
			if d := sub.drops.Load(); d > 0 {
				f.log.Warn("widebandt2: same-carrier voice tap dropped IQ to a lagging voice consumer — the followed call's decode was starved (expect short/gappy recordings); reduce CPU load or lower sdr.sample_rate (issue #402)",
					"serial", f.serial, "freq_hz", f.freqHz, "dropped_chunks", d)
			}
		})
		return sub.drops.Load()
	}
}

// broadcast copies chunk to every subscriber. The bank's slice is reused
// across calls, so each subscriber gets its own copy. A no-op (no
// allocation) with zero subscribers — the idle hot-path default. A full
// subscriber buffer drops the chunk and bumps that subscriber's counter.
func (f *channelFanout) broadcast(chunk []complex64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.subs) == 0 {
		return
	}
	for _, s := range f.subs {
		cp := make([]complex64, len(chunk))
		copy(cp, chunk)
		select {
		case s.ch <- cp:
		default:
			s.drops.Add(1)
		}
	}
}

// channel returns the engine channel tuned to freqHz, or nil. e.channels is
// fully built in New and append-only, so the scan is lock-free (the same
// assumption Channels / DMRTier3ControlChannel make).
func (e *Engine) channel(freqHz uint32) *engineChannel {
	for _, c := range e.channels {
		if c.freqHz == freqHz {
			return c
		}
	}
	return nil
}

// SubscribeChannelIQ returns the channelised (post-DDC) IQ of the channel
// tuned to freqHz plus an unsubscribe func (which reports the lifetime
// dropped-chunk count). ok is false when no channel is tuned to freqHz.
func (e *Engine) SubscribeChannelIQ(freqHz uint32) (ch <-chan []complex64, unsub func() uint64, ok bool) {
	c := e.channel(freqHz)
	if c == nil || c.fan == nil {
		return nil, nil, false
	}
	ch, unsub = c.fan.subscribe()
	return ch, unsub, true
}

// ChannelRateHz reports the ACTUAL per-tap output rate of the channel tuned
// to freqHz (the DDC bank's realised rate for a 144 kHz TETRA tap, not the
// nominal), or 0 when unknown. Voice consumers must clock symbol recovery
// from this, never from the nominal rate (issue #550).
func (e *Engine) ChannelRateHz(freqHz uint32) float64 {
	if c := e.channel(freqHz); c != nil {
		return c.rateHz
	}
	return 0
}

// Suspended reports whether the pump is parked (the SDR borrowed by a hunt),
// during which no channel emits IQ.
func (e *Engine) Suspended() bool { return e.suspended.Load() }

// ChannelVoiceSource adapts one wideband control channel into a same-carrier
// voice device: it streams that channel's own channelised IQ, so a grant on
// the wideband-hosted CC carrier binds without a second SDR or a retune. It
// implements trunking.Tuner + trunking.FrequencyChecker for the voice pool and
// composer.IQSource (+ the exact-rate and same-carrier capabilities) for the
// composer — the wideband twin of ccdecoder.CCVoiceSource. The engine is
// resolved lazily: the daemon builds the voice pool before its wideband
// engines exist.
type ChannelVoiceSource struct {
	get    func() *Engine
	freqHz uint32
	serial string
}

// NewChannelVoiceSource returns a source for the channel on freqHz of the
// engine get resolves (nil ⇒ inert: never binds).
func NewChannelVoiceSource(get func() *Engine, freqHz uint32, serial string) *ChannelVoiceSource {
	return &ChannelVoiceSource{get: get, freqHz: freqHz, serial: serial}
}

func (s *ChannelVoiceSource) Serial() string { return s.serial }

// FrequencyHz is the carrier this source serves.
func (s *ChannelVoiceSource) FrequencyHz() uint32 { return s.freqHz }

// CarrierKey groups every same-carrier tap of ONE wideband channel under one
// shared per-carrier voice demux in the composer. It embeds the channel
// frequency, not just the engine: one wideband engine can host several TETRA
// CCs and each needs its own demux. Empty while no engine exists (the source
// never binds then anyway).
func (s *ChannelVoiceSource) CarrierKey() string {
	eng := s.get()
	if eng == nil {
		return ""
	}
	return fmt.Sprintf("wb-same-carrier:%p:%d", eng, s.freqHz)
}

// SetCenterFreq is a no-op: the carrier is tuned by the wideband engine.
func (s *ChannelVoiceSource) SetCenterFreq(uint32) error { return nil }

// CanTune reports whether hz is this channel's own carrier — the only
// frequency the source can serve — on a running, unsuspended engine that
// actually hosts the channel. Off-carrier grants fall through to the next
// voice device.
func (s *ChannelVoiceSource) CanTune(hz uint32) bool {
	eng := s.get()
	if eng == nil || hz != s.freqHz || eng.Suspended() {
		return false
	}
	return eng.ChannelRateHz(s.freqHz) > 0
}

func (s *ChannelVoiceSource) SampleRateHz() uint32 {
	if eng := s.get(); eng != nil {
		return uint32(eng.ChannelRateHz(s.freqHz) + 0.5)
	}
	return 0
}

func (s *ChannelVoiceSource) SampleRateExactHz() float64 {
	if eng := s.get(); eng != nil {
		return eng.ChannelRateHz(s.freqHz)
	}
	return 0
}

// ErrNoWidebandEngine is returned by StreamIQ before the wideband engine
// hosting the channel exists.
var ErrNoWidebandEngine = fmt.Errorf("widebandt2: no wideband engine hosts this channel")

// StreamIQ subscribes to the channel's channelised IQ until ctx ends.
func (s *ChannelVoiceSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	eng := s.get()
	if eng == nil {
		return nil, ErrNoWidebandEngine
	}
	ch, unsub, ok := eng.SubscribeChannelIQ(s.freqHz)
	if !ok {
		return nil, fmt.Errorf("widebandt2: no channel tuned to %d Hz", s.freqHz)
	}
	go func() {
		<-ctx.Done()
		unsub() // closes ch, ending the composer's read loop
	}()
	return ch, nil
}
