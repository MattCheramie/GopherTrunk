package phase1

import "time"

// pendingGrantTTL bounds how long a grant whose channel ID lacked an
// IdentifierUpdate at arrival is held in the deferred queue. P25 voice
// grants address very short live windows — a queued grant older than
// this describes a call that's already underway on a voice channel
// without us, so resolving it after the fact buys no operational
// benefit. The bound also caps memory exposure to a runaway site.
const pendingGrantTTL = 5 * time.Second

// pendingGrantSlotCap is the per-channel-ID ring capacity. Four covers
// the legitimate "burst of grants arrived a few hundred ms before the
// matching IDEN_UP" shape without growing unboundedly when a site
// genuinely never broadcasts an IDEN_UP for that ID.
const pendingGrantSlotCap = 4

// pendingGrant is one queued voice grant awaiting its IdentifierUpdate.
type pendingGrant struct {
	g   voiceGrant
	nac uint16
	at  time.Time
}

// pendingGrants holds voice grants whose channel ID had no
// IdentifierUpdate when the grant was dispatched, one bounded ring per
// channel ID. The control channel calls add() on a BandPlan miss and
// drain() after a successful BandPlan.Apply for that ID; entries older
// than pendingGrantTTL are dropped on drain. Not safe for concurrent
// use — the control channel reads/writes from a single goroutine, the
// same constraint BandPlan already documents.
type pendingGrants struct {
	slots [16][]pendingGrant
}

// add appends g to channelID's ring. When the ring is full the oldest
// entry is dropped to make room — a stuck channel ID can't grow memory
// unbounded. channelID outside [0,15] is silently ignored to match
// BandPlan.Apply's contract.
func (p *pendingGrants) add(channelID uint8, g voiceGrant, nac uint16, now time.Time) {
	if int(channelID) >= len(p.slots) {
		return
	}
	entry := pendingGrant{g: g, nac: nac, at: now}
	ring := p.slots[channelID]
	if len(ring) >= pendingGrantSlotCap {
		ring = append(ring[:0], ring[1:]...)
	}
	p.slots[channelID] = append(ring, entry)
}

// drain returns and clears every entry for channelID whose age is
// within pendingGrantTTL. Expired entries are dropped silently. The
// returned slice is fresh per call so the caller may safely retain it.
func (p *pendingGrants) drain(channelID uint8, now time.Time) []pendingGrant {
	if int(channelID) >= len(p.slots) {
		return nil
	}
	ring := p.slots[channelID]
	p.slots[channelID] = nil
	if len(ring) == 0 {
		return nil
	}
	out := make([]pendingGrant, 0, len(ring))
	for _, e := range ring {
		if now.Sub(e.at) <= pendingGrantTTL {
			out = append(out, e)
		}
	}
	return out
}

// unresolvedGrantWarnEvery rate-limits the unresolved-channel-ID WARN per
// channel ID: a site that never announces an ID grants on it every call,
// and one line per ten minutes is enough to see it without flooding.
const unresolvedGrantWarnEvery = 10 * time.Minute

// unresolvedChannelID is the per-channel-ID state behind
// noteUnresolvedGrant. Reset when an IdentifierUpdate for the ID applies.
type unresolvedChannelID struct {
	first    time.Time // first unresolved grant since the last reset
	lastWarn time.Time
	grants   int
}

// noteUnresolvedGrant counts a grant whose channel ID has no band-plan
// entry and WARNs once the ID has stayed unresolved for longer than
// pendingGrantTTL, i.e. once grants are being lost rather than briefly
// deferred. A grant that only ever names unannounced IDs used to leave no
// trace above DEBUG, so a site that never follows a call looked identical
// to a site with no traffic (issue #1273). The line names the ID, the
// channel number and the IDs the site HAS announced, which is what an
// operator needs to write a p25_band_plan entry.
func (c *ControlChannel) noteUnresolvedGrant(g voiceGrant, nac uint16) {
	id := g.channelID
	if int(id) >= len(c.unresolved) {
		return
	}
	u := &c.unresolved[id]
	now := c.now()
	if u.first.IsZero() {
		u.first = now
	}
	u.grants++
	if now.Sub(u.first) <= pendingGrantTTL {
		return
	}
	if !u.lastWarn.IsZero() && now.Sub(u.lastWarn) < unresolvedGrantWarnEvery {
		return
	}
	u.lastWarn = now
	known := make([]int, 0, 16)
	for _, e := range c.bandPlan.Snapshot() {
		known = append(known, int(e.ChannelID))
	}
	c.log.Warn("p25: voice grants name a channel ID the site has not announced (no IDEN_UP); their calls are not followed. If it never resolves, add the ID under the system's p25_band_plan",
		"system", c.systemName, "nac", nac, "id", id, "num", g.channelNumber,
		"tg", g.groupID, "grants", u.grants,
		"unresolved_for", now.Sub(u.first).Round(time.Second),
		"announced_ids", known)
}
