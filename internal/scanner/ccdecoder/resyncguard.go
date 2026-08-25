package ccdecoder

import "time"

// resyncGuard is the signal-time decode-drought watchdog generalised from the
// TETRA pipeline's checkResync (see tetraResyncTimeout for the full design
// rationale): once the receiver has PROCESSED a full window-worth of post-DDC
// signal without a single CRC-clean control-channel decode, the pipeline
// forces a fast DSP re-acquire (Reset) instead of waiting for the slow
// steady-state timing loop to drift back after a noise burst knocks it
// off-lock.
//
// The trigger is PROCESSED-SIGNAL time, not wall clock: check counts the
// samples actually fed to the receiver since the last decode and compares
// them against a sample budget (window × rateHz). That makes the destructive
// reset immune to CPU starvation — a descheduled goroutine (whose IQ is
// meanwhile dropped upstream) feeds no samples, so the budget never advances
// and a still-good lock is never discarded. A decode since the previous check
// (heartbeat advanced) clears the budget; a fire resets the budget again, so
// exactly one reset can occur per window-worth of real signal — an inherent
// throttle and reacquire window.
//
// The guard is a no-op until the first decode ever lands (activity == 0):
// with nothing to reacquire toward, resetting a still-hunting channel would
// just churn. It applies only to control channels that transmit continuously
// (P25 TSBKs, DMR Tier III CSBKs, NXDN CACs, P25 Phase 2 MAC PDUs, TETRA
// BSCH/SCH) — on a camped conventional channel (DMR Tier I/II, TETRA DMO)
// inter-transmission silence is normal, a drought proves nothing, and the
// guard must not be wired.
type resyncGuard struct {
	rateHz float64       // post-DDC channel rate; converts the window into a sample budget
	window time.Duration // signal-time decode-drought budget

	samplesSinceDecode int64
	lastSeenActivity   int64
}

// check accumulates n processed samples against the drought budget and
// reports true when a full window of real signal has been processed with no
// decode — the caller then performs its (destructive) DSP re-acquire.
// activity is the control channel's decode heartbeat (LastActivityNano):
// any change since the previous call means a decode landed and clears the
// budget. Call after rx.Process so a decode from the current chunk is
// credited before the budget grows.
func (g *resyncGuard) check(n int, activity int64) bool {
	if activity == 0 {
		// No decode has ever landed — nothing to reacquire toward.
		return false
	}
	if activity != g.lastSeenActivity {
		g.lastSeenActivity = activity
		g.samplesSinceDecode = 0
		return false
	}
	if g.rateHz <= 0 || g.window <= 0 {
		return false
	}
	g.samplesSinceDecode += int64(n)
	if g.samplesSinceDecode < int64(g.window.Seconds()*g.rateHz) {
		return false
	}
	g.samplesSinceDecode = 0
	return true
}

// reset clears the accumulated budget and heartbeat tracking; call from the
// pipeline's Reset so a retune restarts the drought accounting from scratch.
func (g *resyncGuard) reset() {
	g.samplesSinceDecode = 0
	g.lastSeenActivity = 0
}

// Per-protocol decode-drought windows. Each is sized ≥ 1.5× the protocol's
// slowest always-on signalling cadence so a healthy lock (which decodes far
// more often) never trips it, while a genuine off-lock reacquires within a
// few seconds — the same sizing discipline as tetraResyncTimeout (1.5 s
// against the ~1 s BSCH cadence).
const (
	// P25 Phase 1: the control channel transmits TSBKs continuously
	// (~10+/s on an idle site), so 2 s of TSBK-free signal is deeply
	// pathological.
	p25Phase1ResyncWindow = 2 * time.Second
	// P25 Phase 2: the 360 ms TDMA superframe carries MAC PDUs on every
	// superframe of a control channel; 2 s ≈ 5+ missed superframes.
	p25Phase2ResyncWindow = 2 * time.Second
	// DMR Tier III: the TSCC beacons Aloha CSBKs continuously (several
	// per second); 3 s is generous against MBC-heavy stretches.
	dmrTier3ResyncWindow = 3 * time.Second
	// NXDN: the RCCH carries a CAC in every 80 ms frame; 2 s ≈ 25 missed
	// frames.
	nxdnResyncWindow = 2 * time.Second
)
