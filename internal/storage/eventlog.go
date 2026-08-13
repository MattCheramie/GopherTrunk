package storage

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// eventLog is the shared subscribe → drain → close lifecycle behind every
// per-domain log writer (APRS, DSC, MDC1200, pager, AIS, location, LoRa, M17,
// …). Each of those used to carry a byte-identical copy of the bus
// subscription, the Run drain loop that filters one events.Kind and forwards
// the typed payload to an insert, and the Close that drains Run — differing
// only in the Kind, the payload type, and the insert/Recent SQL. This
// generic factors the identical lifecycle; each writer keeps only its own
// insert and Recent (the parts that are genuinely per-table).
//
// A writer embeds *eventLog[T] to promote Run and Close, then supplies its
// insert func to newEventLog. The subscription is taken at construction so
// events published before Run begins are not lost.
type eventLog[T any] struct {
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
	kind      events.Kind
	name      string // log prefix, e.g. "aprslog"
	insert    func(T) error
}

// newEventLog wires a drain for events of the given Kind whose payload is a T,
// forwarding each to insert. name is the writer's log prefix. bus is required;
// a nil logger falls back to slog.Default().
func newEventLog[T any](bus *events.Bus, logger *slog.Logger, kind events.Kind, name string, insert func(T) error) (*eventLog[T], error) {
	if bus == nil {
		return nil, errors.New("storage/" + name + ": events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	e := &eventLog[T]{
		bus:     bus,
		log:     logger,
		kind:    kind,
		name:    name,
		insert:  insert,
		runDone: make(chan struct{}),
	}
	e.sub = bus.Subscribe()
	return e, nil
}

// Run drains events of the writer's Kind until ctx cancels or the bus closes.
// Payloads that don't type-assert to T are skipped; an insert error is logged
// and draining continues.
func (e *eventLog[T]) Run(ctx context.Context) error {
	defer close(e.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-e.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != e.kind {
				continue
			}
			p, ok := ev.Payload.(T)
			if !ok {
				continue
			}
			if err := e.insert(p); err != nil {
				e.log.Error(e.name+": insert failed", "err", err)
			}
		}
	}
}

// Close releases the bus subscription and waits (bounded) for Run to drain.
func (e *eventLog[T]) Close() error {
	e.closeOnce.Do(func() {
		e.sub.Close()
		select {
		case <-e.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
