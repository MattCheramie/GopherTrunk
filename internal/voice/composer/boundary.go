package composer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// boundaryTracker is the protocol-agnostic recording-boundary controller
// shared by every voice chain (FM, DMR, P25 Phase 1 / 2). It centralises
// the universal call-boundary behaviour so hangtime + split/conversation
// grouping work identically across protocols:
//
//   - Hangtime end-of-call: once voice has been decoding, the call is
//     ended VoiceHangtime after the last decoded voice frame, instead of
//     waiting out the engine's much longer call-timeout watchdog. This
//     keeps recordings tightly bounded to the actual transmission.
//   - Talkgroup gating (digital protocols that decode an in-band TG):
//     audio whose talkgroup differs from the granted talkgroup is not
//     written, and a sustained foreign talkgroup ends the call — fixing
//     the shared-frequency case where two virtual tuners decode the same
//     IQ and one call would otherwise append the other talkgroup's audio.
//   - Per-transmission splitting: at an end-of-transmission boundary
//     (a P25 terminator, a DMR voice terminator, an FM squelch gap, …)
//     the recorder rolls to a fresh file when split mode is on.
//
// A chain feeds the tracker two facts — onVoice(tg) per decoded voice
// frame and onTransmissionEnd() at an over boundary — and runs run(ctx)
// in a goroutine. Everything else (Touch throttling, EndCall, segment
// publishing) is handled here so the chains stay thin and uniform.
//
// Concurrency: onVoice / onTransmissionEnd are called from the single
// chain goroutine; run executes in its own goroutine and only reads the
// atomics. The mutable match/segment bookkeeping is touched only by the
// chain goroutine.
type boundaryTracker struct {
	c        *Composer
	serial   string
	grantTG  uint32
	patches  map[uint32]bool
	hangtime time.Duration

	lastVoiceNano atomic.Int64 // unixnano of the last MATCHING voice frame
	sawVoice      atomic.Bool

	// Chain-goroutine-only state.
	lastMatch      bool // most recent talkgroup-match decision (LDU2 etc. inherit it)
	foreignRun     int  // consecutive frames with a known foreign talkgroup
	voiceSinceRoll bool // wrote audio since the last segment roll / start
	lastTouchNano  int64
	endOnce        sync.Once
}

// foreignRunToEnd is how many consecutive frames carrying a known
// foreign talkgroup end the call. A couple of frames debounces a single
// mis-decoded Link Control word without letting a real foreign
// transmission append more than ~one LDU of audio (which is gated out
// anyway).
const foreignRunToEnd = 2

func (c *Composer) newBoundaryTracker(serial string, grantTG uint32, patched []uint32) *boundaryTracker {
	var patches map[uint32]bool
	if len(patched) > 0 {
		patches = make(map[uint32]bool, len(patched))
		for _, p := range patched {
			patches[p] = true
		}
	}
	return &boundaryTracker{
		c:         c,
		serial:    serial,
		grantTG:   grantTG,
		patches:   patches,
		hangtime:  c.hangtime,
		lastMatch: true, // write optimistically until a talkgroup proves foreign
	}
}

// onVoice records one decoded voice frame and returns whether its audio
// should be written. tg is the in-band talkgroup the frame belongs to,
// or 0 when this frame carries no talkgroup (P25 LDU2, FM, protocols
// without per-frame TG) — in which case the previous match decision is
// inherited. When the grant has no talkgroup (grantTG == 0) gating is
// disabled and everything matches.
func (bt *boundaryTracker) onVoice(tg uint32) bool {
	if bt.grantTG == 0 {
		bt.lastMatch = true
	} else if tg != 0 {
		bt.lastMatch = tg == bt.grantTG || bt.patches[tg]
		if bt.lastMatch {
			bt.foreignRun = 0
		} else {
			bt.foreignRun++
			if bt.foreignRun >= foreignRunToEnd {
				// A different talkgroup has taken this (shared) frequency;
				// our call is over. The engine will start the other
				// talkgroup's call on its own tuner.
				bt.end(trunking.EndReasonNormal)
			}
		}
	}
	if bt.lastMatch {
		bt.lastVoiceNano.Store(time.Now().UnixNano())
		bt.sawVoice.Store(true)
		bt.voiceSinceRoll = true
	}
	return bt.lastMatch
}

// onTransmissionEnd marks an end-of-transmission boundary. In split
// (per-transmission) mode it rolls the recording to a fresh file via a
// KindCallSegment event — but only when audio was written since the last
// roll, so a run of terminators / idle frames doesn't spawn empty files.
func (bt *boundaryTracker) onTransmissionEnd() {
	if !bt.c.splitTx || !bt.voiceSinceRoll {
		return
	}
	bt.voiceSinceRoll = false
	bt.c.bus.Publish(events.Event{
		Kind: events.KindCallSegment,
		Payload: trunking.CallSegment{
			DeviceSerial: bt.serial,
			At:           time.Now(),
		},
	})
}

// run drives the hangtime timer and throttled Touch heartbeat until ctx
// is cancelled (which the composer does when the call ends). It ends the
// call once no matching voice has arrived for hangtime.
func (bt *boundaryTracker) run(ctx context.Context) {
	// Poll fast enough that the hangtime end + Touch heartbeat are
	// responsive, but coarser than a frame interval so we don't spin.
	tick := 200 * time.Millisecond
	if bt.c.touchEvery > 0 && bt.c.touchEvery < tick {
		tick = bt.c.touchEvery
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !bt.sawVoice.Load() {
				continue
			}
			last := bt.lastVoiceNano.Load()
			// Keep the engine's LastHeardAt fresh: Touch once per new
			// activity (gated on progress — when voice stops, last stops
			// advancing and we stop touching, so the engine watchdog
			// backstop still sees the call go idle).
			if bt.c.engine != nil && last != bt.lastTouchNano {
				bt.c.engine.Touch(bt.serial)
				bt.lastTouchNano = last
			}
			if time.Since(time.Unix(0, last)) > bt.hangtime {
				bt.end(trunking.EndReasonNormal)
				return
			}
		}
	}
}

// end ends the call exactly once via the engine, which publishes
// CallEnd; the composer then cancels this chain's context.
func (bt *boundaryTracker) end(reason trunking.EndReason) {
	bt.endOnce.Do(func() {
		if bt.c.engine != nil {
			bt.c.engine.EndCall(bt.serial, reason)
		}
	})
}
