package api

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// AudioPublisher is the runtime fan-out point between the per-call
// composer (which produces PCM) and any number of gRPC StreamAudio
// subscribers (which consume frames over the wire). It satisfies the
// same WritePCM contract the recorder + player + tone-out detector
// implement, so the daemon drops it straight into the existing
// composer.PCMSink fan-out.
//
// The publisher subscribes to the events bus at construction time to
// keep a per-device-serial Grant map alive — that's how the published
// AudioFrame can carry talkgroup / system context the subscriber
// filters against. Slow subscribers don't block fast ones (or the
// composer): each subscriber has a bounded channel and we drop on
// full, counting the loss for visibility.
//
// Lifecycle: NewAudioPublisher → Run (subscribes + drains bus until
// ctx cancels) → Close (releases bus subscription). The daemon
// spawns Run on a goroutine like every other long-lived component.
type AudioPublisher struct {
	log *slog.Logger

	bus    *events.Bus
	busSub *events.Subscription

	mu     sync.RWMutex
	subs   map[*audioSubscriber]struct{}
	grants map[string]trunking.Grant // by device serial

	dropped atomic.Uint64

	runDone   chan struct{}
	closeOnce sync.Once
}

// AudioSubFilter is what callers pass to Subscribe to scope the
// frames they receive. Empty fields match everything.
type AudioSubFilter struct {
	DeviceSerials []string
	TalkgroupIDs  []uint32
	// IncludeRaw mirrors the proto flag. Until WriteRawFrame is
	// wired into the publisher (digital-voice raw frames are a
	// follow-up), this just selects whether to surface PCM frames
	// at all — false is the safe default that's never going to
	// break a caller that didn't ask for audio.
	IncludeRaw bool
}

// audioSubscriber is one wire-side stream. The publisher writes
// frames into the bounded channel; StreamAudio reads them out and
// writes to the gRPC stream. cancel signals the publisher's hot
// loop to drop this subscriber if the gRPC stream closes mid-flight.
type audioSubscriber struct {
	filter AudioSubFilter
	ch     chan *apiv1.AudioFrame
	// dropped counts samples lost on this subscriber alone (per-
	// channel insight that lets clients tell whether they fell
	// behind vs. the publisher being slow).
	dropped atomic.Uint64
}

// NewAudioPublisher constructs a publisher backed by the supplied
// bus. The bus subscription happens at construction time so callers
// can publish CallStart events before Run begins without losing them.
func NewAudioPublisher(bus *events.Bus, log *slog.Logger) (*AudioPublisher, error) {
	if bus == nil {
		return nil, errors.New("audio: events.Bus is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &AudioPublisher{
		log:     log,
		bus:     bus,
		busSub:  bus.Subscribe(),
		subs:    make(map[*audioSubscriber]struct{}),
		grants:  make(map[string]trunking.Grant),
		runDone: make(chan struct{}),
	}, nil
}

// Run drains bus events until ctx cancels, maintaining the per-
// device-serial Grant map that WritePCM consults. Returns ctx.Err()
// on shutdown.
func (p *AudioPublisher) Run(ctx context.Context) error {
	defer close(p.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-p.busSub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				cs, ok := ev.Payload.(trunking.CallStart)
				if !ok {
					continue
				}
				p.mu.Lock()
				p.grants[cs.DeviceSerial] = cs.Grant
				p.mu.Unlock()
			case events.KindCallEnd:
				ce, ok := ev.Payload.(trunking.CallEnd)
				if !ok {
					continue
				}
				p.mu.Lock()
				delete(p.grants, ce.DeviceSerial)
				p.mu.Unlock()
			}
		}
	}
}

// Close releases the bus subscription. Safe to call multiple times.
func (p *AudioPublisher) Close() error {
	p.closeOnce.Do(func() {
		if p.busSub != nil {
			p.busSub.Close()
		}
	})
	return nil
}

// Stats reports cumulative publisher counters. Useful for the
// /metrics surface and for diagnosing slow consumers.
type AudioPublisherStats struct {
	Subscribers    int
	DroppedTotal   uint64
	TrackedGrants  int
}

func (p *AudioPublisher) Stats() AudioPublisherStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return AudioPublisherStats{
		Subscribers:   len(p.subs),
		DroppedTotal:  p.dropped.Load(),
		TrackedGrants: len(p.grants),
	}
}

// WritePCM satisfies composer.PCMSink. Builds one AudioFrame per
// call and fans it to every subscriber whose filter matches. A
// missing Grant (composer wrote PCM before CallStart landed) drops
// the frame silently — the publisher only emits frames that carry
// full talkgroup context.
func (p *AudioPublisher) WritePCM(deviceSerial string, samples []int16) error {
	if p == nil || len(samples) == 0 {
		return nil
	}
	p.mu.RLock()
	grant, ok := p.grants[deviceSerial]
	if !ok || len(p.subs) == 0 {
		p.mu.RUnlock()
		return nil
	}
	frame := buildPCMFrame(grant, deviceSerial, samples)
	for sub := range p.subs {
		if !sub.filter.matches(deviceSerial, grant.GroupID) {
			continue
		}
		select {
		case sub.ch <- frame:
		default:
			sub.dropped.Add(uint64(len(samples)))
			p.dropped.Add(uint64(len(samples)))
		}
	}
	p.mu.RUnlock()
	return nil
}

// Subscribe registers a new subscriber and returns its frame
// channel. Caller MUST call Unsubscribe(ret) before letting the
// channel go out of scope — leaked subscribers keep the publisher
// fanning frames into them forever. Channel capacity defaults to
// 64 frames (≈ 1 second of audio at typical chunk sizes).
func (p *AudioPublisher) Subscribe(filter AudioSubFilter) *audioSubscriber {
	sub := &audioSubscriber{
		filter: filter,
		ch:     make(chan *apiv1.AudioFrame, 64),
	}
	p.mu.Lock()
	p.subs[sub] = struct{}{}
	p.mu.Unlock()
	return sub
}

// Unsubscribe removes the subscriber. Idempotent. After
// unsubscribing the channel is closed so any reader sees io.EOF /
// channel-closed.
func (p *AudioPublisher) Unsubscribe(sub *audioSubscriber) {
	if sub == nil {
		return
	}
	p.mu.Lock()
	if _, ok := p.subs[sub]; ok {
		delete(p.subs, sub)
		close(sub.ch)
	}
	p.mu.Unlock()
}

// matches reports whether this subscriber wants a frame for the
// given device serial + talkgroup ID. Empty filter slices match
// everything; non-empty slices act as allow-lists.
func (f AudioSubFilter) matches(serial string, groupID uint32) bool {
	if len(f.DeviceSerials) > 0 {
		hit := false
		for _, s := range f.DeviceSerials {
			if s == serial {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if len(f.TalkgroupIDs) > 0 {
		hit := false
		for _, id := range f.TalkgroupIDs {
			if id == groupID {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// buildPCMFrame builds the wire-side AudioFrame for a chunk of
// samples. We copy the int16 buffer into a fresh byte slice so the
// caller can reuse theirs without mutating in-flight frames.
func buildPCMFrame(grant trunking.Grant, serial string, samples []int16) *apiv1.AudioFrame {
	body := make([]byte, len(samples)*2)
	for i, s := range samples {
		u := uint16(s)
		body[i*2] = byte(u)
		body[i*2+1] = byte(u >> 8)
	}
	return &apiv1.AudioFrame{
		Grant:        grantToPB(grant),
		DeviceSerial: serial,
		Body: &apiv1.AudioFrame_Pcm{
			Pcm: &apiv1.PCMSamples{
				SampleRate: 8000, // recorder writes at 8 kHz today; future: pull from composer
				Samples:    body,
			},
		},
	}
}
