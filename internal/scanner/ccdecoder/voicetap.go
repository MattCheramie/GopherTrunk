package ccdecoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Same-carrier voice tap. A TETRA Single Carrier Base Station keeps voice calls
// on other TDMA timeslots of the *control* carrier, so a granted call's IQ is
// the control channel's own channelised (post-DDC) stream — already at the
// per-protocol pipeline rate (144 kHz for TETRA), exactly what the voice chain
// wants. Rather than allocate a second SDR or a redundant down-converter, the
// decoder fans that stream to voice subscribers. `CCVoiceSource` adapts one
// Decoder into a voice device the trunking engine binds — but only for grants
// on the control channel's current carrier (a no-retune, same-carrier tap).

// voiceFanout distributes copies of the post-DDC IQ to zero or more voice
// subscribers. Sends are non-blocking (drop on a full buffer) so a slow voice
// consumer can never stall the decode hot path.
type voiceFanout struct {
	mu   sync.Mutex
	subs map[int]*voiceSub
	next int
	log  *slog.Logger
	// bufDepth is the per-subscriber channel capacity: how many post-DDC IQ
	// chunks may queue before a lagging consumer starts dropping (issue #402).
	// Configurable via recordings.voice_tap_buffer_chunks; 0 uses the default.
	bufDepth int
}

// defaultVoiceTapBufferChunks is the per-subscriber buffer depth when none is
// configured. Concurrent same-carrier calls share one tap consumer, so the
// memory cost is a single buffer per active carrier — a depth well above the
// original 64 buys jitter headroom for a few tens of KB.
const defaultVoiceTapBufferChunks = 128

// voiceSub is one subscriber's channel plus its dropped-chunk counter. A drop
// is a gap of missing IQ delivered to the followed call's voice chain, which
// breaks the receiver's symbol timing / lock — the mechanism behind starved,
// short/gappy TETRA recordings (issue #402). Counting them turns that silent
// loss into an actionable log line at call end.
type voiceSub struct {
	ch    chan []complex64
	drops atomic.Uint64
}

func newVoiceFanout(log *slog.Logger, bufDepth int) *voiceFanout {
	if log == nil {
		log = slog.Default()
	}
	if bufDepth <= 0 {
		bufDepth = defaultVoiceTapBufferChunks
	}
	return &voiceFanout{subs: map[int]*voiceSub{}, log: log, bufDepth: bufDepth}
}

// subscribe registers a voice subscriber and returns its IQ channel plus an
// unsubscribe func. The unsubscribe func returns the number of chunks that were
// dropped to this subscriber over its lifetime (0 when it kept up), so a consumer
// that wants the count — e.g. a triggered DDC capture reporting whether the grab
// has gaps — can read it; callers that don't care simply ignore the return.
func (f *voiceFanout) subscribe() (<-chan []complex64, func() uint64) {
	sub := &voiceSub{ch: make(chan []complex64, f.bufDepth)}
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
			// After the delete-under-lock no further broadcast can increment
			// this sub's counter, so the load sees the final total. Surface a
			// non-zero drop count once at unsubscribe (call end) with the
			// remedy, mirroring the --iq-capture drop warning.
			if d := sub.drops.Load(); d > 0 {
				f.log.Warn("ccdecoder: same-carrier voice tap dropped IQ to a lagging voice consumer — the followed call's decode was starved (expect short/gappy recordings); reduce CPU load or lower sdr.sample_rate (issue #402)",
					"dropped_chunks", d)
			}
		})
		// The counter is final once the delete above ran; safe to read on any
		// call (a second unsubscribe is a no-op that still reports the total).
		return sub.drops.Load()
	}
}

// broadcast copies chunk to every subscriber. The caller's slice is reused
// across chunks, so each subscriber receives its own copy. A no-op (and no
// allocation) when nothing is subscribed — the production hot-path default.
// A full subscriber buffer drops the chunk (protecting the decode path) and
// bumps that subscriber's drop counter so the loss is not silent.
func (f *voiceFanout) broadcast(chunk []complex64) {
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
		default: // subscriber lagging; drop to protect the decode path
			s.drops.Add(1)
		}
	}
}

// SubscribeVoiceIQ returns a channel of channelised (post-DDC) IQ at the
// pipeline rate plus an unsubscribe func. The same-carrier voice tap consumes
// it for the life of a followed call. The unsubscribe func returns the number of
// IQ chunks dropped to this subscriber (0 when it kept up); callers that don't
// need the count ignore it.
func (d *Decoder) SubscribeVoiceIQ() (<-chan []complex64, func() uint64) {
	return d.voiceFan.subscribe()
}

// CenterFreqHz reports the frequency the active control pipeline is tuned to,
// or 0 when idle/hunting. Gates the same-carrier voice tap.
func (d *Decoder) CenterFreqHz() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.activeFreqHz
}

// PipelineRateHz reports the channelised stream rate the voice subscriber
// receives (e.g. 144 kHz for TETRA).
func (d *Decoder) PipelineRateHz() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pipelineRateHz
}

// CCVoiceSource adapts a control-channel Decoder into a voice device that
// streams the control carrier's own IQ for a same-carrier grant — no second
// SDR, no retune. It satisfies the voice composer's IQSource, the trunking
// Tuner, and FrequencyChecker, so the engine binds it only for grants on the
// control channel's current carrier.
//
// The Decoder is resolved lazily through an accessor so the source can be
// registered in the voice pool before the decoder exists (the pool is built
// during daemon startup ahead of the control decoder) and keeps working across
// a control-SDR reacquire that swaps the decoder. Nil-safe until then.
type CCVoiceSource struct {
	get    func() *Decoder
	serial string
}

// NewCCVoiceSource wraps the decoder returned by get as a same-carrier voice
// device with the given serial. get may return nil before the control decoder
// is constructed; the source stays inert (never binds) until it is non-nil.
func NewCCVoiceSource(get func() *Decoder, serial string) *CCVoiceSource {
	return &CCVoiceSource{get: get, serial: serial}
}

func (s *CCVoiceSource) Serial() string { return s.serial }

// CarrierKey returns a stable identifier for the control carrier this tap shares
// with the other same-carrier taps. Every cc:same-carrier:N source backed by the
// same control decoder returns the same key, so the voice composer groups them
// under one shared per-carrier TETRA voice demux (one receiver + demux for the
// whole carrier, instead of one per concurrent call). Empty while the control
// decoder is not yet constructed (the source never binds then anyway).
func (s *CCVoiceSource) CarrierKey() string {
	dec := s.get()
	if dec == nil {
		return ""
	}
	return fmt.Sprintf("cc-same-carrier:%p", dec)
}

// SetCenterFreq is a no-op: the carrier is already tuned by the control decoder.
func (s *CCVoiceSource) SetCenterFreq(uint32) error { return nil }

// CanTune reports whether hz is the control channel's current carrier — the only
// frequency this tap can serve. A nil/idle CC never binds, and off-carrier
// grants fall through to a real role:voice SDR.
func (s *CCVoiceSource) CanTune(hz uint32) bool {
	dec := s.get()
	if dec == nil {
		return false
	}
	cc := dec.CenterFreqHz()
	return cc != 0 && hz == cc
}

func (s *CCVoiceSource) SampleRateHz() uint32 {
	if dec := s.get(); dec != nil {
		return uint32(dec.PipelineRateHz() + 0.5)
	}
	return 0
}

func (s *CCVoiceSource) SampleRateExactHz() float64 {
	if dec := s.get(); dec != nil {
		return dec.PipelineRateHz()
	}
	return 0
}

// StreamIQ subscribes to the control carrier's channelised IQ until ctx ends.
func (s *CCVoiceSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	dec := s.get()
	if dec == nil {
		return nil, errNoControlDecoder
	}
	ch, unsub := dec.SubscribeVoiceIQ()
	go func() {
		<-ctx.Done()
		unsub() // closes ch, ending the composer's read loop
	}()
	return ch, nil
}

var errNoControlDecoder = errors.New("ccdecoder: no control decoder for same-carrier voice tap")
