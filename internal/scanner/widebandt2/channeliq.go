package widebandt2

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
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

	// total counts every sample broadcast on this channel, subscribed or
	// not: the stream position a counted subscriber and an onset mark are
	// measured in.
	total uint64
	// lastNano is when the channel last broadcast a chunk (unix ns, 0 =
	// never) and dropped the cumulative chunks any subscriber missed on a
	// full buffer — the channel's IQ-flow health (issue #1288 §12), read
	// lock-free by the status / metrics paths.
	lastNano atomic.Int64
	dropped  atomic.Uint64
	// preroll is a ring of the most recent samples (nil unless
	// enablePreroll; conventional channels only, issue #1288). prerollW is
	// the next write index, prerollN the number of valid samples.
	preroll  []complex64
	prerollW int
	prerollN int
	// armed is a transmission onset (a stream position) waiting for the
	// next preroll subscriber, set by arm when a conventional scanner
	// declares a call; armedAt is total when it was set, so a mark nobody
	// collects goes stale instead of replaying old IQ into a later call.
	armed   bool
	onset   uint64
	armedAt uint64
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

// convPrerollSeconds is the ring a conventional channel keeps for the voice
// chain of the call its scanner declares (issue #1288). A tone-gated
// channel's call starts only once the CTCSS / DCS gate confirms, up to about
// half a second after the carrier came up (a 250 ms Goertzel block that the
// onset can land in the middle of, then a full one; DCS needs ~0.4 s), and
// the chain used to subscribe only then, so the first words of every
// transmission were lost. The ring holds that span, with margin; the scanner
// says where the transmission began (arm), so an untoned channel's call gets
// only the few milliseconds before its squelch broke, not a second of hiss.
// 1 s at the 48 kHz tap rate is 384 KiB per channel.
const convPrerollSeconds = 1.0

// enablePreroll sizes the pre-roll ring to samples (≤ 0 disables it).
func (f *channelFanout) enablePreroll(samples int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if samples <= 0 {
		f.preroll = nil
	} else {
		f.preroll = make([]complex64, samples)
	}
	f.prerollW, f.prerollN = 0, 0
}

// prerollWrite appends chunk to the ring. Caller holds f.mu.
func (f *channelFanout) prerollWrite(chunk []complex64) {
	n := len(f.preroll)
	if n == 0 {
		return
	}
	if len(chunk) >= n {
		copy(f.preroll, chunk[len(chunk)-n:])
		f.prerollW, f.prerollN = 0, n
		return
	}
	k := copy(f.preroll[f.prerollW:], chunk)
	if k < len(chunk) {
		copy(f.preroll, chunk[k:])
	}
	f.prerollW = (f.prerollW + len(chunk)) % n
	if f.prerollN += len(chunk); f.prerollN > n {
		f.prerollN = n
	}
}

// prerollSince returns, as a fresh slice in time order, the ring's samples
// from stream position from to the latest broadcast (as much of that span
// as the ring still holds), or nil. Caller holds f.mu.
func (f *channelFanout) prerollSince(from uint64) []complex64 {
	if f.prerollN == 0 || from >= f.total {
		return nil
	}
	want := f.total - from
	if want > uint64(f.prerollN) {
		want = uint64(f.prerollN)
	}
	out := make([]complex64, want)
	n := len(f.preroll)
	start := (f.prerollW - int(want) + n) % n // index of the oldest wanted sample
	k := copy(out, f.preroll[start:min(start+int(want), n)])
	copy(out[k:], f.preroll[:int(want)-k])
	return out
}

// arm marks stream position onset as the start of the transmission whose
// call was just declared; the next preroll subscriber receives the IQ from
// there on (see convPrerollSeconds).
func (f *channelFanout) arm(onset uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed, f.onset, f.armedAt = true, onset, f.total
}

// subscribe registers a subscriber and returns its IQ channel plus an
// unsubscribe func that closes the channel and reports the lifetime
// dropped-chunk count (0 when the consumer kept up).
func (f *channelFanout) subscribe() (<-chan []complex64, func() uint64) {
	ch, _, unsub := f.register(false)
	return ch, unsub
}

// register adds a subscriber. It returns the stream position of the first
// live sample the subscriber will receive. With preroll, an onset mark that
// is still fresh is consumed: the subscriber's first chunk is the IQ from
// the onset up to that position, delivered atomically with the registration
// so no sample is both in it and in a later broadcast, and none is lost
// between them.
func (f *channelFanout) register(preroll bool) (<-chan []complex64, uint64, func() uint64) {
	ch, start, unsub, _ := f.registerWithDrops(preroll)
	return ch, start, unsub
}

// registerWithDrops is register plus a live read of the subscriber's
// dropped-chunk count, for a consumer that must report the loss while the
// subscription is still open (a recording's partial marker, issue #1288).
func (f *channelFanout) registerWithDrops(preroll bool) (<-chan []complex64, uint64, func() uint64, func() uint64) {
	sub := &channelSub{ch: make(chan []complex64, channelTapBudget(f.rateHz))}
	f.mu.Lock()
	start := f.total
	if preroll && f.armed {
		f.armed = false
		if f.total-f.armedAt <= uint64(len(f.preroll)) {
			if pre := f.prerollSince(f.onset); pre != nil {
				sub.ch <- pre // fresh channel, depth ≥ 1: cannot block
			}
		}
	}
	id := f.next
	f.next++
	f.subs[id] = sub
	f.mu.Unlock()
	var once sync.Once
	unsub := func() uint64 {
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
	return sub.ch, start, unsub, sub.drops.Load
}

// broadcast copies chunk to every subscriber. The bank's slice is reused
// across calls, so each subscriber gets its own copy. A no-op (no
// allocation) with zero subscribers — the idle hot-path default. A full
// subscriber buffer drops the chunk and bumps that subscriber's counter.
func (f *channelFanout) broadcast(chunk []complex64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.total += uint64(len(chunk))
	f.lastNano.Store(time.Now().UnixNano())
	f.prerollWrite(chunk)
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
			f.dropped.Add(1)
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

// SubscribeChannelIQCounted is SubscribeChannelIQ that also returns the
// stream position of the first sample the subscriber receives, so it can
// name a sample it has seen to MarkChannelOnset (issue #1288).
func (e *Engine) SubscribeChannelIQCounted(freqHz uint32) (ch <-chan []complex64, start uint64, unsub func() uint64, ok bool) {
	c := e.channel(freqHz)
	if c == nil || c.fan == nil {
		return nil, 0, nil, false
	}
	ch, start, unsub = c.fan.register(false)
	return ch, start, unsub, true
}

// SubscribeChannelIQWithPreroll is SubscribeChannelIQ for the voice chain of
// a conventional channel's call: when the channel's scanner has marked the
// call's onset (MarkChannelOnset), the first chunk is the IQ from that onset
// on, so the chain hears the start of the transmission it was not yet
// subscribed for. Without a mark it is plain SubscribeChannelIQ.
func (e *Engine) SubscribeChannelIQWithPreroll(freqHz uint32) (ch <-chan []complex64, unsub func() uint64, ok bool) {
	c := e.channel(freqHz)
	if c == nil || c.fan == nil {
		return nil, nil, false
	}
	ch, _, unsub = c.fan.register(true)
	return ch, unsub, true
}

// SubscribeChannelIQWithPrerollLoss is SubscribeChannelIQWithPreroll plus a
// live read of the chunks this subscription has dropped so far.
func (e *Engine) SubscribeChannelIQWithPrerollLoss(freqHz uint32) (ch <-chan []complex64, unsub func() uint64, dropped func() uint64, ok bool) {
	c := e.channel(freqHz)
	if c == nil || c.fan == nil {
		return nil, nil, nil, false
	}
	ch, _, unsub, dropped = c.fan.registerWithDrops(true)
	return ch, unsub, dropped, true
}

// MarkChannelOnset records stream position onset (as counted from a
// SubscribeChannelIQCounted start) as the beginning of the transmission whose
// call was just declared on the channel tuned to freqHz; the next
// SubscribeChannelIQWithPreroll collects it.
func (e *Engine) MarkChannelOnset(freqHz uint32, onset uint64) {
	if c := e.channel(freqHz); c != nil && c.fan != nil {
		c.fan.arm(onset)
	}
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

// ChannelIQHealth is the IQ-flow health of one engine channel (issue #1288
// §12): when it last emitted a chunk (zero = never), how many chunks its
// subscribers have missed on full buffers, and whether the pump is parked
// (the SDR lent to a hunt), which is why a channel may legitimately be
// silent. A quiet channel still advances — the bank emits noise IQ whether
// or not anyone transmits — so a stale LastIQ means the stream, not the RF.
type ChannelIQHealth struct {
	LastIQ    time.Time
	Drops     uint64
	Suspended bool
}

// ChannelIQHealth reports the channel tuned to freqHz; ok is false when no
// channel is tuned there.
func (e *Engine) ChannelIQHealth(freqHz uint32) (h ChannelIQHealth, ok bool) {
	c := e.channel(freqHz)
	if c == nil || c.fan == nil {
		return h, false
	}
	if n := c.fan.lastNano.Load(); n != 0 {
		h.LastIQ = time.Unix(0, n)
	}
	h.Drops = c.fan.dropped.Load()
	h.Suspended = e.Suspended()
	return h, true
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

// StreamIQ subscribes to the channel's channelised IQ until ctx ends. It is
// what the composer calls for a call's voice chain, so on a conventional
// channel it collects the call's onset mark (MarkCallOnset): the chain starts
// at the transmission's first samples, not at the call start.
func (s *ChannelVoiceSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	eng := s.get()
	if eng == nil {
		return nil, ErrNoWidebandEngine
	}
	ch, unsub, ok := eng.SubscribeChannelIQWithPreroll(s.freqHz)
	if !ok {
		return nil, fmt.Errorf("widebandt2: no channel tuned to %d Hz", s.freqHz)
	}
	go func() {
		<-ctx.Done()
		unsub() // closes ch, ending the composer's read loop
	}()
	return ch, nil
}

// StreamIQWithLoss is StreamIQ plus a func reporting the IQ chunks this
// call's stream has lost so far: chunks the channel fan-out dropped to it,
// plus chunks the physical SDR's driver dropped upstream of the whole
// wideband stream since the call began (those hole every channel at once).
// The composer reads it when the call ends to mark a recording partial
// (issue #1288).
func (s *ChannelVoiceSource) StreamIQWithLoss(ctx context.Context) (<-chan []complex64, func() uint64, error) {
	eng := s.get()
	if eng == nil {
		return nil, nil, ErrNoWidebandEngine
	}
	physical := eng.Serial()
	base := sdr.IQDropCount(physical)
	ch, unsub, dropped, ok := eng.SubscribeChannelIQWithPrerollLoss(s.freqHz)
	if !ok {
		return nil, nil, fmt.Errorf("widebandt2: no channel tuned to %d Hz", s.freqHz)
	}
	go func() {
		<-ctx.Done()
		unsub()
	}()
	return ch, func() uint64 { return dropped() + (sdr.IQDropCount(physical) - base) }, nil
}

// StreamIQCounted is StreamIQ for the conventional scanner watching the
// channel: it never collects an onset mark, and it returns the stream
// position of its first sample, so the scanner can mark where a
// transmission began (conventional.OnsetIQSource, issue #1288).
func (s *ChannelVoiceSource) StreamIQCounted(ctx context.Context) (<-chan []complex64, uint64, error) {
	eng := s.get()
	if eng == nil {
		return nil, 0, ErrNoWidebandEngine
	}
	ch, start, unsub, ok := eng.SubscribeChannelIQCounted(s.freqHz)
	if !ok {
		return nil, 0, fmt.Errorf("widebandt2: no channel tuned to %d Hz", s.freqHz)
	}
	go func() {
		<-ctx.Done()
		unsub()
	}()
	return ch, start, nil
}

// IQHealth is the source channel's ChannelIQHealth; ok is false before the
// engine exists or when it has no channel at this frequency.
func (s *ChannelVoiceSource) IQHealth() (ChannelIQHealth, bool) {
	eng := s.get()
	if eng == nil {
		return ChannelIQHealth{}, false
	}
	return eng.ChannelIQHealth(s.freqHz)
}

// MarkCallOnset records stream position onset as the start of the
// transmission whose call the scanner is about to declare; the call's voice
// chain (StreamIQ) starts there.
func (s *ChannelVoiceSource) MarkCallOnset(onset uint64) {
	if eng := s.get(); eng != nil {
		eng.MarkChannelOnset(s.freqHz, onset)
	}
}
