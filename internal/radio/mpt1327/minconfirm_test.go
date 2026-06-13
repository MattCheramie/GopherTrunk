package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// countCCLocked drains the subscription and returns how many cc.locked events
// were published.
func countCCLocked(sub *events.Subscription) int {
	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				got++
			}
		default:
			return got
		}
	}
}

// TestMinConfirmSuppressesSingleCodewordLock: with the production confirmation
// threshold a lone recognised codeword (a cross-protocol false parse) must not
// declare a lock; the second confirms it.
func TestMinConfirmSuppressesSingleCodewordLock(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetMinConfirm(2)

	cc.Ingest(alohaCodeword(0x3))
	if got := countCCLocked(sub); got != 0 {
		t.Fatalf("after 1 codeword with minConfirm=2: got %d CCLocked, want 0", got)
	}

	cc.Ingest(alohaCodeword(0x3))
	if got := countCCLocked(sub); got != 1 {
		t.Fatalf("after 2 codewords with minConfirm=2: got %d CCLocked, want 1", got)
	}
}

// TestMinConfirmDefaultLocksOnFirst: the zero-value (in-package fixture) default
// keeps the legacy lock-on-first behaviour so existing fixtures still lock.
func TestMinConfirmDefaultLocksOnFirst(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	cc.Ingest(alohaCodeword(0x3))
	if got := countCCLocked(sub); got != 1 {
		t.Fatalf("default minConfirm: got %d CCLocked on first codeword, want 1", got)
	}
}
