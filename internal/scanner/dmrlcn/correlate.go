package dmrlcn

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Default correlation window: a carrier keys up shortly after the grant
// CSBK is decoded on the control channel. A small negative floor absorbs
// clock skew between the grant timestamp and the onset detector's frame
// clock.
const (
	defaultMinDelay = -100 * time.Millisecond
	defaultMaxDelay = 700 * time.Millisecond
	defaultGrantTTL = 2 * time.Second
)

// Candidate is an unconfirmed (LCN, frequency) pairing the correlator
// produced from a grant/onset coincidence. It still needs decode
// confirmation before it is trusted as a fit Pair.
type Candidate struct {
	LCN    uint16
	FreqHz uint32
	Grant  events.DMRGrantObserved
}

// pendingGrant is a recently observed grant awaiting an onset.
type pendingGrant struct {
	obs     events.DMRGrantObserved
	matched bool
}

// correlator pairs granted LCNs with the RF carriers that subsequently
// key up. It only commits a pairing when the onset is unambiguous: the
// window must contain grants for exactly one LCN. When several different
// LCNs were granted near the same instant the onset is dropped rather
// than guessed — repeated observations eventually yield an unambiguous
// match for each LCN. It owns no goroutine and is not safe for concurrent
// use; the learner drives it from a single loop.
type correlator struct {
	minDelay time.Duration
	maxDelay time.Duration
	ttl      time.Duration
	grants   []pendingGrant
}

func newCorrelator(minDelay, maxDelay, ttl time.Duration) *correlator {
	if minDelay == 0 {
		minDelay = defaultMinDelay
	}
	if maxDelay == 0 {
		maxDelay = defaultMaxDelay
	}
	if ttl == 0 {
		ttl = defaultGrantTTL
	}
	return &correlator{minDelay: minDelay, maxDelay: maxDelay, ttl: ttl}
}

// addGrant records an observed grant for later onset matching.
func (c *correlator) addGrant(obs events.DMRGrantObserved) {
	c.grants = append(c.grants, pendingGrant{obs: obs})
}

// matchOnset returns the candidate pairing for an onset, or ok=false when
// no grant matches or the match is ambiguous (grants for more than one
// LCN fall in the window). Matched grants for the chosen LCN are consumed
// so they don't re-match a later onset.
func (c *correlator) matchOnset(o CarrierOnset) (Candidate, bool) {
	var (
		lcn      uint16
		grant    events.DMRGrantObserved
		idxs     []int
		haveLCN  bool
		multiLCN bool
	)
	for i := range c.grants {
		g := &c.grants[i]
		if g.matched {
			continue
		}
		d := o.At.Sub(g.obs.At)
		if d < c.minDelay || d > c.maxDelay {
			continue
		}
		switch {
		case !haveLCN:
			lcn, grant, haveLCN = g.obs.LCN, g.obs, true
			idxs = append(idxs, i)
		case g.obs.LCN == lcn:
			idxs = append(idxs, i)
		default:
			multiLCN = true
		}
	}
	if !haveLCN || multiLCN {
		return Candidate{}, false
	}
	for _, i := range idxs {
		c.grants[i].matched = true
	}
	return Candidate{LCN: lcn, FreqHz: o.FreqHz, Grant: grant}, true
}

// expire drops grants older than the TTL relative to now. Called
// periodically so the pending slice doesn't grow without bound.
func (c *correlator) expire(now time.Time) {
	if len(c.grants) == 0 {
		return
	}
	kept := c.grants[:0]
	for _, g := range c.grants {
		if g.matched {
			continue
		}
		if now.Sub(g.obs.At) > c.ttl {
			continue
		}
		kept = append(kept, g)
	}
	c.grants = kept
}

// pendingCount reports how many unexpired, unmatched grants are queued
// (test helper).
func (c *correlator) pendingCount() int {
	n := 0
	for _, g := range c.grants {
		if !g.matched {
			n++
		}
	}
	return n
}
