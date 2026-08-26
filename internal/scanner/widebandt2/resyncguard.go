package widebandt2

import (
	"log/slog"
	"time"
)

// Per-protocol decode-drought windows for the wideband taps, mirroring the
// ccdecoder pipelines' resyncGuard constants (see that package's
// resyncguard.go for the sizing discipline: ≥ 1.5× the protocol's slowest
// always-on signalling cadence). TETRA has its own dedicated
// tetraChannelReceiver (tetra.go) with the additional payload-drought +
// lock-stale watchdogs; DMR Tier II is conventional — inter-transmission
// silence is normal there, so it gets no drought guard.
const (
	p25Phase1ResyncWindow = 2 * time.Second
	p25Phase2ResyncWindow = 2 * time.Second
	dmrTier3ResyncWindow  = 3 * time.Second
	nxdnResyncWindow      = 2 * time.Second
)

// droughtGuardReceiver wraps a protocol receiver with the signal-time
// decode-drought watchdog the ccdecoder pipelines run (their resyncGuard):
// once a full window-worth of narrowband signal has been PROCESSED with no
// CRC-clean control-channel decode (the activity heartbeat holding), the
// reacquire callback fires — receiver loops reset to centre plus whatever
// cross-call sync state the protocol's state machine keys on absolute dibit
// indices. Signal-time budgeting makes the destructive reset immune to CPU
// starvation (a descheduled pump processes no samples, so the budget never
// advances); a decode clears the budget, and each fire re-arms it, so at
// most one reset occurs per window of real signal. No-op until the first
// decode ever lands. Runs on the single wideband pump goroutine.
type droughtGuardReceiver struct {
	rx        narrowbandReceiver
	activity  func() int64 // the CC's LastActivityNano heartbeat
	reacquire func()       // full mid-stream reset (receiver + CC sync state)
	log       *slog.Logger
	system    string
	label     string // protocol tag for the debug line
	rateHz    float64
	window    time.Duration

	samplesSinceDecode int64
	lastSeenActivity   int64
}

func (g *droughtGuardReceiver) Process(iq []complex64) {
	g.rx.Process(iq)
	act := g.activity()
	if act == 0 {
		return
	}
	if act != g.lastSeenActivity {
		g.lastSeenActivity = act
		g.samplesSinceDecode = 0
		return
	}
	if g.rateHz <= 0 || g.window <= 0 {
		return
	}
	g.samplesSinceDecode += int64(len(iq))
	if g.samplesSinceDecode < int64(g.window.Seconds()*g.rateHz) {
		return
	}
	g.samplesSinceDecode = 0
	g.reacquire()
	if g.log != nil {
		g.log.Debug("widebandt2: dsp resync (signal-time decode drought; reacquiring from centre)",
			"system", g.system, "proto", g.label)
	}
}
