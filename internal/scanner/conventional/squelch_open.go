package conventional

import (
	"math"
	"sync"
	"time"
)

// Squelch onset delay and live channel level (issue #1321 follow-up).
//
// scanWindow used to open on the FIRST chunk above the squelch threshold;
// ActivityDebounce only de-bounces the trailing edge of a call that is
// already open. On a channel with impulsive noise every click above
// squelch_dbfs therefore became a hangtime-length "call", and the only cure
// was a squelch high enough to lose weak real signals too.
// Channel.SquelchOpenDelay asks for the threshold to be cleared
// continuously for that long first. 0 (default) keeps the old
// first-chunk behaviour exactly.

// MaxSquelchOpenDelay bounds Channel.SquelchOpenDelay. Longer than this is
// no longer click rejection: it eats the start of every transmission.
const MaxSquelchOpenDelay = time.Second

// openDelaySlack is added to a scan-window extension granted to a
// carrier run still short of SquelchOpenDelay, to cover chunk
// granularity and delivery jitter.
const openDelaySlack = 50 * time.Millisecond

// onsetRun measures how long the channel has been continuously above its
// open threshold. In stream time (samples / rate) when the scanner knows
// its sample rate — exact, and independent of how chunks are delivered —
// else in wall time.
type onsetRun struct {
	delay   time.Duration
	rateHz  float64
	now     func() time.Time
	samples int
	start   time.Time
}

// observe records one chunk of n samples and reports whether the run, this
// chunk included, has lasted at least the delay. A chunk below threshold
// ends the run.
func (r *onsetRun) observe(n int, above bool) bool {
	if r.delay <= 0 {
		return above
	}
	if !above {
		r.samples, r.start = 0, time.Time{}
		return false
	}
	return r.extend(n) >= r.delay
}

func (r *onsetRun) extend(n int) time.Duration {
	if r.rateHz > 0 {
		r.samples += n
		return time.Duration(float64(r.samples) / r.rateHz * float64(time.Second))
	}
	now := r.now()
	if r.start.IsZero() {
		r.start = now
	}
	return now.Sub(r.start)
}

// remaining is how much longer the current run must last to open, or 0
// when no run is in progress (a below-threshold channel gets no extension).
func (r *onsetRun) remaining() time.Duration {
	if r.delay <= 0 {
		return 0
	}
	var have time.Duration
	switch {
	case r.rateHz > 0 && r.samples > 0:
		have = time.Duration(float64(r.samples) / r.rateHz * float64(time.Second))
	case r.rateHz <= 0 && !r.start.IsZero():
		have = r.now().Sub(r.start)
	default:
		return 0
	}
	if have >= r.delay {
		return 0
	}
	return r.delay - have
}

// levelTau is the time constant of the reported channel level. Long enough
// that a REST poll does not read one random chunk, short enough to follow a
// carrier keying up within a poll interval.
const levelTau = 200 * time.Millisecond

// levelFloorDb clamps the reported level: far below any real receiver's
// noise floor, it only ever shows for digitally silent IQ.
const levelFloorDb = -150.0

// levelGauge holds a channel's live squelch level for Snapshot: in-channel
// power in dBFS on an FM/digital channel, carrier-to-noise in dB on an AM
// one — the same number the squelch compares, smoothed. Written by the Run
// goroutine every chunk, read by Snapshot.
type levelGauge struct {
	mu   sync.Mutex
	db   float64
	at   time.Time
	set  bool
	seed bool // the next sample replaces db instead of smoothing into it
}

// startVisit makes the next sample seed the gauge: a channel's previous
// visit says nothing about its level now.
func (g *levelGauge) startVisit() {
	g.mu.Lock()
	g.seed = true
	g.mu.Unlock()
}

// observe smooths one chunk's level (n samples at rateHz; rateHz <= 0 seeds
// every chunk) into the gauge.
func (g *levelGauge) observe(db float64, n int, rateHz float64, now time.Time) {
	if math.IsNaN(db) {
		return
	}
	// Exact digital silence reads -Inf, which JSON cannot carry and which
	// would pin the average there for good.
	db = max(min(db, -levelFloorDb), levelFloorDb)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seed || !g.set || rateHz <= 0 {
		g.db = db
	} else {
		a := 1 - math.Exp(-float64(n)/(rateHz*levelTau.Seconds()))
		g.db += a * (db - g.db)
	}
	g.at, g.set, g.seed = now, true, false
}

// read returns the smoothed level and when it was last updated; ok=false
// when the channel has not been measured yet.
func (g *levelGauge) read() (db float64, at time.Time, ok bool) {
	if g == nil {
		return 0, time.Time{}, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.db, g.at, g.set
}

// levelFor returns channel idx's gauge, or nil. Same ownership rule as
// detectorFor.
func (s *Scanner) levelFor(idx int) *levelGauge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx < 0 || idx >= len(s.levels) {
		return nil
	}
	return s.levels[idx]
}

// observeChunk is observe for a possibly nil gauge (a Scanner built without
// New, as some tests do), stamped with the scanner clock.
func (g *levelGauge) observeChunk(db float64, n int, rateHz float64, now func() time.Time) {
	if g == nil {
		return
	}
	g.observe(db, n, rateHz, now())
}
