package trunking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// Tuner is the subset of sdr.Device the hunter needs. Decoupling from the
// full Device interface keeps the hunter testable without an IQ source.
type Tuner interface {
	SetCenterFreq(hz uint32) error
}

// Hunter scans a System's candidate control channels and parks on the
// first frequency that produces a matching cc.locked event within the
// per-frequency dwell timeout.
//
// The hunter is intentionally protocol-agnostic at the wiring level: it
// retunes the SDR and watches the events.Bus. The downstream demod
// pipeline (channelizer + C4FM/H-DQPSK demod + protocol decoder)
// publishes cc.locked events; the hunter parks on the first match.
type Hunter struct {
	bus     *events.Bus
	log     *slog.Logger
	cache   *Cache
	tuner   Tuner
	system  System
	dwell   time.Duration
}

// HunterOptions configure a Hunter at construction.
type HunterOptions struct {
	System System
	Tuner  Tuner
	Bus    *events.Bus
	Cache  *Cache
	Log    *slog.Logger
	// Dwell is how long to wait on each candidate before giving up.
	// Defaults to 3 seconds.
	Dwell time.Duration
}

func NewHunter(o HunterOptions) (*Hunter, error) {
	if err := o.System.Validate(); err != nil {
		return nil, err
	}
	if o.Bus == nil {
		return nil, errors.New("trunking/hunter: events.Bus is required")
	}
	if o.Tuner == nil {
		return nil, errors.New("trunking/hunter: Tuner is required")
	}
	log := o.Log
	if log == nil {
		log = slog.Default()
	}
	dwell := o.Dwell
	if dwell <= 0 {
		dwell = 3 * time.Second
	}
	return &Hunter{
		bus:    o.Bus,
		log:    log,
		cache:  o.Cache,
		tuner:  o.Tuner,
		system: o.System,
		dwell:  dwell,
	}, nil
}

// Hunt scans the candidate frequencies until either a CC locks (success)
// or ctx cancels (returns ctx.Err()) or the candidate list is exhausted
// (returns ErrNoControlChannel).
//
// On success the locked frequency and NAC are persisted to the cache and
// returned to the caller.
func (h *Hunter) Hunt(ctx context.Context) (LockResult, error) {
	var lastKnown uint32
	if h.cache != nil {
		if e, ok := h.cache.Get(h.system.Name); ok {
			lastKnown = e.LastFrequencyHz
		}
	}
	candidates := h.system.HuntOrder(lastKnown)

	sub := h.bus.Subscribe()
	defer sub.Close()

	for _, freq := range candidates {
		if err := ctx.Err(); err != nil {
			return LockResult{}, err
		}
		h.log.Info("cc-hunt: trying", "system", h.system.Name, "freq_hz", freq)
		if err := h.tuner.SetCenterFreq(freq); err != nil {
			h.log.Warn("cc-hunt: tune failed", "freq_hz", freq, "err", err)
			continue
		}
		// Drain any stale events buffered before this candidate.
		drainSubscription(sub)

		deadline := time.NewTimer(h.dwell)
		locked, ok := waitForLock(ctx, sub, deadline.C, freq)
		deadline.Stop()
		if ok {
			result := LockResult{
				System:    h.system.Name,
				Frequency: freq,
				NAC:       locked.NAC,
				At:        time.Now().UTC(),
			}
			if h.cache != nil {
				if err := h.cache.Set(h.system.Name, CachedSystem{
					LastFrequencyHz: freq,
					LastLockAt:      result.At,
					NAC:             locked.NAC,
				}); err != nil {
					h.log.Warn("cc-hunt: cache write failed", "err", err)
				}
			}
			h.log.Info("cc-hunt: locked", "system", h.system.Name, "freq_hz", freq, "nac", locked.NAC)
			return result, nil
		}
		if err := ctx.Err(); err != nil {
			return LockResult{}, err
		}
	}
	return LockResult{}, ErrNoControlChannel
}

// LockResult is returned by a successful Hunt.
type LockResult struct {
	System    string
	Frequency uint32
	NAC       uint16
	At        time.Time
}

// ErrNoControlChannel is returned when every candidate frequency exhausts
// its dwell window without locking.
var ErrNoControlChannel = errors.New("trunking/hunter: no control channel found")

func drainSubscription(sub *events.Subscription) {
	for {
		select {
		case _, ok := <-sub.C:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func waitForLock(ctx context.Context, sub *events.Subscription, timeout <-chan time.Time, freq uint32) (phase1.LockState, bool) {
	for {
		select {
		case ev, ok := <-sub.C:
			if !ok {
				return phase1.LockState{}, false
			}
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(phase1.LockState)
			if !ok {
				continue
			}
			if ls.FrequencyHz != freq {
				continue
			}
			return ls, true
		case <-timeout:
			return phase1.LockState{}, false
		case <-ctx.Done():
			return phase1.LockState{}, false
		}
	}
}

// String renders a one-line summary of a LockResult for logs.
func (r LockResult) String() string {
	return fmt.Sprintf("%s @ %d Hz (NAC=%X) at %s", r.System, r.Frequency, r.NAC, r.At.Format(time.RFC3339))
}

// Compile-time check: sdr.Device satisfies Tuner.
var _ Tuner = (sdr.Device)(nil)
