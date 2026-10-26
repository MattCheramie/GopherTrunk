package adsb

import (
	"sync"
	"time"
)

// Tracker maintains per-ICAO state so consecutive even+odd CPR
// position messages decode into a globally-unambiguous lat/lon
// pair. ADS-B aircraft alternate between even-encoded (CPRFormat=0)
// and odd-encoded (CPRFormat=1) position reports roughly every
// 0.5 s; the receiver pairs them within ~10 s to recover the
// global position via CPRDecodeGlobal.
//
// One Tracker per receive pipeline (e.g. one per BEAST upstream
// connection, or one per native DSP receiver). Thread-safe;
// callers may invoke Update from multiple goroutines.
//
// Spec: DO-260B §2.2.3.2.3.7 — CPR pair-pairing semantics.
type Tracker struct {
	mu       sync.Mutex
	state    map[uint32]*icaoState
	maxAgeNs int64 // CPR pair max age — spec is 10 s

	// seen holds the time each ICAO was last proven by a CRC-checked
	// frame (DF 11 / 17 / 18). ProcessFrame consults it before
	// accepting an address/parity frame (DF 0 / 4 / 5 / 16 / 20 / 21),
	// whose recovered ICAO is otherwise unverifiable (#1316).
	seen       map[uint32]int64
	seenRecent int64 // how long a proven ICAO stays trusted
}

// ICAORecentWindow is how long an ICAO proven by a CRC-checked frame
// stays trusted for address/parity frames: 60 s, dump1090's figure. A
// transponder answering interrogations also squitters DF 11 at least
// once a second and DF 17 every 0.5 s, so a live aircraft is re-proven
// far more often than this.
const ICAORecentWindow = 60 * time.Second

// icaoState holds the most-recent even and odd CPR halves for one
// aircraft.
type icaoState struct {
	evenLat, evenLon int
	oddLat, oddLon   int
	evenAt, oddAt    int64 // unix nanoseconds
	hasEven, hasOdd  bool
	// mostRecentIsEven tracks which half arrived more recently.
	// CPRDecodeGlobal uses it to pick which half's reference
	// position to return.
	mostRecentIsEven bool
}

// NewTracker returns an empty Tracker with the default 10 s CPR
// pair max age.
func NewTracker() *Tracker {
	return &Tracker{
		state:      make(map[uint32]*icaoState),
		maxAgeNs:   int64(10 * time.Second),
		seen:       make(map[uint32]int64),
		seenRecent: int64(ICAORecentWindow),
	}
}

// NoteICAO records that icao was proven by a CRC-checked frame at now
// (unix nanoseconds; 0 = time.Now()).
func (t *Tracker) NoteICAO(icao uint32, now int64) {
	if icao == 0 {
		return
	}
	if now == 0 {
		now = time.Now().UnixNano()
	}
	t.mu.Lock()
	t.seen[icao] = now
	t.mu.Unlock()
}

// KnownICAO reports whether icao was proven by a CRC-checked frame
// within ICAORecentWindow of now.
func (t *Tracker) KnownICAO(icao uint32, now int64) bool {
	if icao == 0 {
		return false
	}
	if now == 0 {
		now = time.Now().UnixNano()
	}
	t.mu.Lock()
	at, ok := t.seen[icao]
	t.mu.Unlock()
	return ok && now-at <= t.seenRecent
}

// KnownICAOs is the number of ICAOs currently trusted (proven within the
// window, not yet pruned).
func (t *Tracker) KnownICAOs() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.seen)
}

// Update ingests one parsed Mode-S message. If the message
// carries a CPR-encoded position and a complementary
// (even / odd) half is available within the spec's 10 s window,
// Update returns a copy of the message with the Position
// struct's Latitude / Longitude / global-decode flags populated
// via CPRDecodeGlobal. Otherwise the message passes through
// unchanged.
//
// now is the receive timestamp in unix nanoseconds. Passing 0
// uses time.Now() — convenient for live receivers; tests pass
// a fixed clock.
//
// Non-position-bearing messages (identification, velocity,
// status, etc.) pass through unchanged and do not update
// tracker state. ICAO 0 (unparseable / CRC-failed frames) is
// ignored.
func (t *Tracker) Update(m Message, now int64) (Message, bool) {
	if m.Kind != KindAirbornePosition && m.Kind != KindSurfacePosition {
		return m, false
	}
	if m.ICAO == 0 || m.Position == nil {
		return m, false
	}
	if now == 0 {
		now = time.Now().UnixNano()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	st, ok := t.state[m.ICAO]
	if !ok {
		st = &icaoState{}
		t.state[m.ICAO] = st
	}

	if m.Position.CPRFormat == 0 {
		st.evenLat = m.Position.CPRLatEven
		st.evenLon = m.Position.CPRLonEven
		st.evenAt = now
		st.hasEven = true
		st.mostRecentIsEven = true
	} else {
		st.oddLat = m.Position.CPRLatOdd
		st.oddLon = m.Position.CPRLonOdd
		st.oddAt = now
		st.hasOdd = true
		st.mostRecentIsEven = false
	}

	if !st.hasEven || !st.hasOdd {
		return m, false
	}

	// Drop stale halves — spec is 10 s; aircraft typically
	// alternate every 0.5 s so the pair should be fresh.
	age := st.evenAt - st.oddAt
	if age < 0 {
		age = -age
	}
	if age > t.maxAgeNs {
		return m, false
	}

	lat, lon, ok := CPRDecodeGlobal(
		st.evenLat, st.evenLon,
		st.oddLat, st.oddLon,
		st.mostRecentIsEven)
	if !ok {
		return m, false
	}

	// Mutate a copy of the message (the caller's Position is a
	// pointer; preserve the raw CPR fields by cloning the
	// struct rather than overwriting in place).
	out := m
	pos := *m.Position
	pos.Latitude = lat
	pos.Longitude = lon
	pos.HasGlobalPosition = true
	out.Position = &pos
	return out, true
}

// Reset clears all per-ICAO state. Call when the upstream
// connection drops (BEAST disconnect, SDR re-acquisition) so
// stale CPR halves don't pair with brand-new ones across a
// gap.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = make(map[uint32]*icaoState)
	t.seen = make(map[uint32]int64)
}

// Size returns the number of distinct ICAOs the tracker is
// holding state for. Useful for /metrics surfacing and
// debugging.
func (t *Tracker) Size() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.state)
}

// Prune drops state for ICAOs whose most-recent half is older
// than the spec's 10 s pair window. Aircraft that left the
// receiver's range stop transmitting; without pruning the
// tracker grows linearly with all aircraft ever seen. Call
// periodically (typically on each Update or once per second).
func (t *Tracker) Prune(now int64) int {
	if now == 0 {
		now = time.Now().UnixNano()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	dropped := 0
	for icao, st := range t.state {
		mostRecent := st.evenAt
		if st.oddAt > mostRecent {
			mostRecent = st.oddAt
		}
		if now-mostRecent > t.maxAgeNs {
			delete(t.state, icao)
			dropped++
		}
	}
	for icao, at := range t.seen {
		if now-at > t.seenRecent {
			delete(t.seen, icao)
		}
	}
	return dropped
}
