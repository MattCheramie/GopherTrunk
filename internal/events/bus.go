// Package events implements an in-process pub/sub bus used by the engine to
// publish trunking events. A separate API surface (gRPC, WebSocket) subscribes
// without coupling to the engine.
package events

import (
	"sync"
	"sync/atomic"
	"time"
)

type Kind string

const (
	KindSDRAttached Kind = "sdr.attached"
	KindSDRDetached Kind = "sdr.detached"
	KindCCLocked    Kind = "cc.locked"
	KindCCLost      Kind = "cc.lost"
	KindCallStart   Kind = "call.start"
	KindCallEnd     Kind = "call.end"
	KindGrant       Kind = "grant"
	KindToneAlert   Kind = "tone.alert"
	KindError       Kind = "error"
)

type Event struct {
	Kind      Kind
	Timestamp time.Time
	Payload   any
}

type Bus struct {
	mu     sync.RWMutex
	subs   map[uint64]chan Event
	nextID atomic.Uint64
	buffer int
	closed bool
}

func NewBus(buffer int) *Bus {
	if buffer <= 0 {
		buffer = 64
	}
	return &Bus{subs: make(map[uint64]chan Event), buffer: buffer}
}

type Subscription struct {
	id uint64
	C  <-chan Event
	b  *Bus
}

func (b *Bus) Subscribe() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		ch := make(chan Event)
		close(ch)
		return &Subscription{C: ch}
	}
	id := b.nextID.Add(1)
	ch := make(chan Event, b.buffer)
	b.subs[id] = ch
	return &Subscription{id: id, C: ch, b: b}
}

func (s *Subscription) Close() {
	if s.b == nil {
		return
	}
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	if ch, ok := s.b.subs[s.id]; ok {
		delete(s.b.subs, s.id)
		close(ch)
	}
	s.b = nil
}

// Publish delivers e to every subscriber. Slow subscribers drop the event
// rather than blocking the publisher; we count drops via the returned int.
func (b *Bus) Publish(e Event) int {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	dropped := 0
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			dropped++
		}
	}
	return dropped
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, ch := range b.subs {
		close(ch)
		delete(b.subs, id)
	}
}
