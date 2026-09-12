package ccdecoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
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
	// preroll is a ring of the most recent post-DDC samples, kept only while
	// the active pipeline asks for one (setPreroll; today DMO only). A voice
	// subscriber that asks for it (subscribe(true) — the same-carrier voice
	// chain) receives the ring's contents, oldest first, as its first chunk,
	// so a chain whose grant structurally trails the start of traffic still
	// decodes the transmission's head. nil when disabled. prerollW is the next
	// write index and prerollN the number of valid samples (≤ len(preroll)).
	preroll  []complex64
	prerollW int
	prerollN int
	// bufDepth is the explicit per-subscriber channel capacity override
	// (recordings.voice_tap_buffer_chunks); 0 auto-sizes each subscription
	// from the pipeline rate at subscribe time (voiceTapBudget).
	bufDepth int
	// rateHz reports the channelised pipeline rate for auto-sizing. May be
	// nil (tests); called OUTSIDE f.mu because the production getter
	// (Decoder.PipelineRateHz) takes the decoder lock.
	rateHz func() float64
}

// defaultVoiceTapBufferChunks is the per-subscriber buffer depth floor.
// Concurrent same-carrier calls share one tap consumer, so the memory cost is
// a single buffer per active carrier — a depth well above the original 64
// buys jitter headroom for a few tens of KB.
const defaultVoiceTapBufferChunks = 128

// Voice-tap auto-sizing. A fixed chunk COUNT is the wrong unit for this buffer
// — the exact defect decodeQueueBudget already fixed for the decode queue
// (decoder.go): SoapyRemote hands a remote SDR ~369-sample datagrams, which
// decimate to ~53-sample post-DDC chunks, so 128 chunks was only ~47 ms at
// 144 kHz. The DMO voice chain runs multi-tens-of-ms work bursts (colour
// recovery Viterbi sweeps) on its drain goroutine, so every burst overflowed
// the tap and starved the call's decode (the operator's dropped_chunks≈240
// per call). Size from wall-clock instead: voiceTapSeconds of IQ at the
// pipeline rate, assuming pessimistically small chunks so the slot count is
// never the limiter.
const (
	voiceTapSeconds             = 1.0
	voiceTapAssumedChunkSamples = 32
	maxVoiceTapChunks           = 16384
)

// voiceTapBudget converts the channelised pipeline rate into a subscriber
// buffer depth covering voiceTapSeconds, clamped to
// [defaultVoiceTapBufferChunks, maxVoiceTapChunks]. An unknown rate (0 —
// subscriber attached before the first pipeline built the DDC) falls back to
// the floor.
func voiceTapBudget(rateHz float64) int {
	if rateHz <= 0 {
		return defaultVoiceTapBufferChunks
	}
	depth := int(math.Ceil(voiceTapSeconds * rateHz / voiceTapAssumedChunkSamples))
	if depth < defaultVoiceTapBufferChunks {
		return defaultVoiceTapBufferChunks
	}
	if depth > maxVoiceTapChunks {
		return maxVoiceTapChunks
	}
	return depth
}

// dmoVoicePrerollSeconds is the pre-roll the DMO pipeline keeps for its
// same-carrier voice chain. DMO grants AFTER traffic has started (the burst
// train is the only evidence there is), and the composer's chain opens a COLD
// receiver on IQ that begins at the grant — so without history it can never
// decode the bursts that preceded the grant, and it loses more while its
// timing / AFC / equalizer loops acquire. Measured on the operator's 13 Sep
// #1003 capture (five PTTs, TestTETRADMOPipelineCaptureReplay phase 2: a cold
// receiver + extractor started at grant − pre-roll, CRC-valid TCH/S bursts
// summed over the five transmissions): 0 s → 320, 0.5 s → 359, 1.0 s → 361,
// 2.0 s → 346, 3.0 s → 318. The head of every transmission (3–8 decodable
// bursts before the slot grid latches) is recovered by ~1 s; longer pre-rolls
// LOSE bursts because the receiver then starts on inter-transmission noise and
// its blind equalizer / normaliser condition on that rather than on the
// signal (one weak transmission went 60 → 20 bursts from 0 s to 3 s). So 1 s:
// the measured optimum, and short enough that the previous transmission
// (≥ dmoGrantRearm = 3 s of silence ago) can never be inside it.
const dmoVoicePrerollSeconds = 1.0

// voicePrerollSamples is the pre-roll ring length the fanout keeps for a
// pipeline of the given protocol at the channelised rate: dmoVoicePrerollSeconds
// of IQ for TETRA DMO, none for anything else (TMO and the trunked protocols
// grant BEFORE their traffic starts, so a chain started at the grant misses
// nothing).
func voicePrerollSamples(protocol trunking.Protocol, rateHz float64) int {
	if protocol != trunking.ProtocolTETRADMO || rateHz <= 0 {
		return 0
	}
	return int(math.Ceil(dmoVoicePrerollSeconds * rateHz))
}

// voiceSub is one subscriber's channel plus its dropped-chunk counter. A drop
// is a gap of missing IQ delivered to the followed call's voice chain, which
// breaks the receiver's symbol timing / lock — the mechanism behind starved,
// short/gappy TETRA recordings (issue #402). Counting them turns that silent
// loss into an actionable log line at call end.
type voiceSub struct {
	ch    chan []complex64
	drops atomic.Uint64
}

func newVoiceFanout(log *slog.Logger, bufDepth int, rateHz func() float64) *voiceFanout {
	if log == nil {
		log = slog.Default()
	}
	if bufDepth < 0 {
		bufDepth = 0
	}
	return &voiceFanout{subs: map[int]*voiceSub{}, log: log, bufDepth: bufDepth, rateHz: rateHz}
}

// setPreroll sizes (or, with samples ≤ 0, disables) the pre-roll ring. Called
// by the decoder when it activates a pipeline (voicePrerollSamples) and when it
// tears one down, so a retune never replays the previous carrier's IQ into a
// chain on the new one. Sizing to the current length just clears it.
func (f *voiceFanout) setPreroll(samples int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if samples <= 0 {
		f.preroll = nil
	} else if len(f.preroll) != samples {
		f.preroll = make([]complex64, samples)
	}
	f.prerollW, f.prerollN = 0, 0
}

// prerollWrite appends chunk to the ring. Caller holds f.mu.
func (f *voiceFanout) prerollWrite(chunk []complex64) {
	n := len(f.preroll)
	if n == 0 {
		return
	}
	if len(chunk) >= n {
		// The chunk alone fills the ring: keep its tail.
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

// prerollSnapshot returns the ring's valid samples in time order as a fresh
// slice, or nil when empty. Caller holds f.mu.
func (f *voiceFanout) prerollSnapshot() []complex64 {
	if f.prerollN == 0 {
		return nil
	}
	out := make([]complex64, f.prerollN)
	n := len(f.preroll)
	if f.prerollN < n {
		copy(out, f.preroll[:f.prerollN])
		return out
	}
	k := copy(out, f.preroll[f.prerollW:])
	copy(out[k:], f.preroll[:f.prerollW])
	return out
}

// subscribe registers a voice subscriber and returns its IQ channel plus an
// unsubscribe func. The unsubscribe func returns the number of chunks that were
// dropped to this subscriber over its lifetime (0 when it kept up), so a consumer
// that wants the count — e.g. a triggered DDC capture reporting whether the grab
// has gaps — can read it; callers that don't care simply ignore the return. With
// withPreroll the subscriber's first chunk is the pre-roll ring's contents (when
// the active pipeline keeps one), delivered atomically with the registration so
// no sample is both in the pre-roll and in a later broadcast, and none is lost
// between them.
func (f *voiceFanout) subscribe(withPreroll bool) (<-chan []complex64, func() uint64) {
	depth := f.bufDepth
	if depth <= 0 {
		// Auto-size from the pipeline rate, resolved OUTSIDE f.mu (the getter
		// takes the decoder lock). By subscribe time the DDC exists, so the
		// rate is real; a 0 rate clamps to the floor.
		var rate float64
		if f.rateHz != nil {
			rate = f.rateHz()
		}
		depth = voiceTapBudget(rate)
	}
	sub := &voiceSub{ch: make(chan []complex64, depth)}
	f.mu.Lock()
	if withPreroll {
		if pre := f.prerollSnapshot(); pre != nil {
			sub.ch <- pre // fresh channel, depth ≥ 1: cannot block
		}
	}
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
	// The pre-roll ring records whether or not anyone is subscribed — that is
	// its whole point (the subscriber arrives after the samples it needs).
	f.prerollWrite(chunk)
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
	return d.voiceFan.subscribe(false)
}

// SubscribeVoiceIQWithPreroll is SubscribeVoiceIQ for a voice chain that must
// decode a transmission already in progress: when the active pipeline keeps a
// pre-roll (TETRA DMO, whose grant structurally trails the start of traffic —
// see dmoVoicePrerollSeconds), the subscriber's first chunk is the most recent
// pre-roll of channelised IQ, followed seamlessly by the live stream. With no
// pre-roll kept it behaves exactly like SubscribeVoiceIQ.
func (d *Decoder) SubscribeVoiceIQWithPreroll() (<-chan []complex64, func() uint64) {
	return d.voiceFan.subscribe(true)
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

// TETRADMOColour reports the DM traffic colour code the active TETRA DMO
// pipeline knows (config override or a confident RecoverDMColourCode), and
// whether one is known at all. Lock-free; safe from any goroutine. A voice
// chain whose grant fired before recovery completed polls this to adopt the
// pipeline's colour instead of brute-forcing its own (see
// composer.runTETRADMOVoiceChain).
func (d *Decoder) TETRADMOColour() (uint32, bool) {
	v := d.dmoColour.Load()
	if v&1 == 0 {
		return 0, false
	}
	return uint32(v >> 1), true
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

// TETRADMOColour delegates to the backing Decoder's recovered DM colour;
// (0, false) while no decoder exists or nothing is known. The composer
// type-asserts for this so a dedicated voice SDR (no CC decoder) simply
// lacks the capability.
func (s *CCVoiceSource) TETRADMOColour() (uint32, bool) {
	if dec := s.get(); dec != nil {
		return dec.TETRADMOColour()
	}
	return 0, false
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

// StreamIQ subscribes to the control carrier's channelised IQ until ctx ends,
// pre-roll first when the active pipeline keeps one (a DMO grant fires after
// the transmission has started; the chain gets its head back this way).
func (s *CCVoiceSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	dec := s.get()
	if dec == nil {
		return nil, errNoControlDecoder
	}
	ch, unsub := dec.SubscribeVoiceIQWithPreroll()
	go func() {
		<-ctx.Done()
		unsub() // closes ch, ending the composer's read loop
	}()
	return ch, nil
}

var errNoControlDecoder = errors.New("ccdecoder: no control decoder for same-carrier voice tap")
