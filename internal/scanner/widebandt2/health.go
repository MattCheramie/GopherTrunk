package widebandt2

import (
	"sync"
	"time"
)

// Decode-health reporting for the web console's "decode:" chip.
//
// On a single-tuner rig the cchunt supervisor reports each system's lock,
// decode quality and level (GET /api/v1/scanner → systems[]). A wideband
// dongle has no supervisor: every control channel on it decodes in parallel,
// and before this the scanner endpoint listed none of them, so the Dashboard,
// Scanner and Plots read "decode: —" beside a control channel decoding
// thousands of clean frames. ChannelHealth is that missing report, one entry
// per wideband control channel, refreshed once per diagnostics window from
// the counters the engine already reads.

// healthSpanWindows is how many diagnostics windows the decode-quality rate is
// measured over (~30 s at iqpower.Window = 1 s). The single-tuner decoder rates
// a lock from its first frame, because a hunt rebuilds its pipeline; a wideband
// tap runs for days, so a lifetime rate would never move. A rolling span lets
// the chip follow the channel as it degrades and recovers.
const healthSpanWindows = 30

// healthSignalFresh is how old a channel's level reading may be before it is
// no longer reported: the pump stops refreshing it while the engine is
// suspended or the stream has stopped, and a stale level is worse than none.
const healthSignalFresh = 5 * time.Second

// Decode-quality buckets and the minimum frame attempts behind one. These are
// ccdecoder's (ccQualityCleanMaxPct / ccQualityMarginalMaxPct /
// ccQualityMinAttempts), so a system reads the same whether a dedicated
// control SDR or a wideband tap decodes it.
const (
	healthCleanMaxPct    = 1.0
	healthMarginalMaxPct = 5.0
	healthMinAttempts    = 20
)

// ChannelHealth is one wideband control channel's live decode health.
type ChannelHealth struct {
	// System and Protocol name the trunked system this channel decodes;
	// Protocol is trunking.Protocol.String() ("p25", "tetra", ...), the same
	// spelling the cchunt supervisor reports.
	System   string
	Protocol string
	FreqHz   uint32
	// Serial is the wideband dongle the channel is tapped from.
	Serial string
	// Group is the sdr.groups entry that dongle is a member of ("" when it
	// is a standalone wideband dongle).
	Group string
	// CampsWhenIdle marks a conventional / direct-mode protocol whose channel
	// is legitimately silent between transmissions (trunking.Protocol).
	CampsWhenIdle bool
	// Decoding is true while the channel has decoded a frame within
	// lowPowerDecodeGrace: the wideband equivalent of a held lock.
	// DecodingSince is when the current decoding stretch began.
	Decoding      bool
	DecodingSince time.Time
	LastDecodeAt  time.Time
	// RecentDecodes is the frames decoded over the last health span; the
	// daemon uses it to pick a system's busiest channel when it has several.
	RecentDecodes int64
	// DecodeQuality buckets the frame-error rate over the last health span
	// (clean / marginal / poor). Only P25 Phase 1 (TSBKs) and TETRA (BSCH)
	// count their failures, as on the single-tuner decoder; HasDecodeHealth
	// is false for other protocols, before enough frames have been attempted,
	// and while the channel is not decoding.
	DecodeQuality   string
	HasDecodeHealth bool
	// SignalDbFS is the channel's mean narrowband power over the last
	// diagnostics window. HasSignal is false before the first window and once
	// the reading is older than healthSignalFresh.
	SignalDbFS float64
	HasSignal  bool
}

// frameTally is a cumulative (decoded, failed) frame count at one window.
type frameTally struct{ ok, fail int64 }

// channelHealthState is the engine's per-channel health record. The pump
// writes it under Engine.healthMu; ChannelHealth reads it.
type channelHealthState struct {
	static       ChannelHealth // System / Protocol / FreqHz / Serial / CampsWhenIdle
	lastDecodeAt time.Time
	since        time.Time
	recent       int64
	quality      string
	hasQuality   bool
	dbfs         float64
	measuredAt   time.Time
}

// healthTracker holds the engine's per-channel health records and the
// pump-owned history behind them.
type healthTracker struct {
	mu     sync.Mutex
	states []channelHealthState // indexed like Engine.channels
}

// staticHealth returns the fields of a channel's health record that never
// change. ok is false for a conventional (tap-only analog) channel, which has
// no decoder to report on.
func staticHealth(ec *engineChannel, serial, group string) (ChannelHealth, bool) {
	if ec.conventional {
		return ChannelHealth{}, false
	}
	return ChannelHealth{
		System:        ec.sysName,
		Protocol:      ec.sysProto.String(),
		FreqHz:        ec.freqHz,
		Serial:        serial,
		Group:         group,
		CampsWhenIdle: ec.sysProto.CampsWhenIdle(),
	}, true
}

// pushTally appends one window's cumulative counts to the channel's ring and
// returns the deltas over the span it now covers (oldest to newest).
func (ec *engineChannel) pushTally(t frameTally) (dOK, dFail int64) {
	n := len(ec.healthRing)
	ec.healthRing[ec.healthNext] = t
	ec.healthNext = (ec.healthNext + 1) % n
	if ec.healthCount < n {
		ec.healthCount++
	}
	oldest := ec.healthRing[0]
	if ec.healthCount == n {
		oldest = ec.healthRing[ec.healthNext]
	}
	return t.ok - oldest.ok, t.fail - oldest.fail
}

// bucketFrameErrors maps a frame-error rate (percent) to clean / marginal /
// poor, with ccdecoder's thresholds.
func bucketFrameErrors(ratePct float64) string {
	switch {
	case ratePct <= healthCleanMaxPct:
		return "clean"
	case ratePct <= healthMarginalMaxPct:
		return "marginal"
	default:
		return "poor"
	}
}

// updateHealth refreshes one channel's health record from the window that
// just closed. i indexes e.channels; prevDecodeAt is the channel's
// lastDecodeAt before this window's decode delta was folded in, so a new
// decoding stretch can be told from a continuing one. Pump goroutine only.
func (e *Engine) updateHealth(i int, ec *engineChannel, now time.Time, dbfs float64, prevDecodeAt time.Time) {
	static, ok := staticHealth(ec, e.serial, e.group)
	if !ok {
		return
	}
	decodedNow := ec.lastDecodeAt.Equal(now)
	if decodedNow && (prevDecodeAt.IsZero() || now.Sub(prevDecodeAt) > lowPowerDecodeGrace) {
		ec.decodingSince = now
	}

	var tally frameTally
	if ec.frameCounts != nil {
		tally.ok, tally.fail = ec.frameCounts()
	} else if ec.decoded != nil {
		tally.ok = int64(ec.decoded())
	}
	dOK, dFail := ec.pushTally(tally)
	quality, hasQuality := "", false
	if ec.frameCounts != nil {
		if attempts := dOK + dFail; attempts >= healthMinAttempts {
			quality = bucketFrameErrors(float64(dFail) / float64(attempts) * 100)
			hasQuality = true
		}
	}

	e.health.mu.Lock()
	defer e.health.mu.Unlock()
	if len(e.health.states) != len(e.channels) {
		e.health.states = make([]channelHealthState, len(e.channels))
	}
	st := &e.health.states[i]
	st.static = static
	st.lastDecodeAt = ec.lastDecodeAt
	st.since = ec.decodingSince
	st.recent = dOK
	st.quality, st.hasQuality = quality, hasQuality
	st.dbfs, st.measuredAt = dbfs, now
}

// ChannelHealth reports every control channel on this engine: its system,
// whether it is decoding, its frame-error bucket and its level. Conventional
// (tap-only analog) channels are left out. A channel that has had no IQ yet
// is still listed, not decoding, so a configured system never disappears.
// Safe to call from any goroutine.
func (e *Engine) ChannelHealth() []ChannelHealth {
	now := e.now()
	e.health.mu.Lock()
	defer e.health.mu.Unlock()
	out := make([]ChannelHealth, 0, len(e.channels))
	for i, ec := range e.channels {
		h, ok := staticHealth(ec, e.serial, e.group)
		if !ok {
			continue
		}
		if i >= len(e.health.states) {
			out = append(out, h)
			continue
		}
		st := e.health.states[i]
		if st.measuredAt.IsZero() {
			out = append(out, h)
			continue
		}
		h.LastDecodeAt = st.lastDecodeAt
		h.Decoding = !st.lastDecodeAt.IsZero() && now.Sub(st.lastDecodeAt) <= lowPowerDecodeGrace
		if h.Decoding {
			h.DecodingSince = st.since
			h.RecentDecodes = st.recent
			h.DecodeQuality, h.HasDecodeHealth = st.quality, st.hasQuality
		}
		if now.Sub(st.measuredAt) <= healthSignalFresh {
			h.SignalDbFS, h.HasSignal = st.dbfs, true
		}
		out = append(out, h)
	}
	return out
}
