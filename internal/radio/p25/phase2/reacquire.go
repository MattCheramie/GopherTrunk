package phase2

// Carrier re-acquisition for the Phase 2 receiver.
//
// The receiver takes its coarse carrier-frequency estimate exactly once, at
// the start of a stream, and hands the residual to a Costas loop whose
// pull-in range is ±SymbolRate/8 = ±750 Hz. Nothing re-takes that estimate.
// So a channel whose offset drifts past the loop — or whose one-shot seed was
// measured over an unlucky window — stays lost for the rest of the call, with
// no path back: the sync detector keeps searching a dibit stream that is
// rotated out from under it.
//
// Measured on a real Phase 2 traffic-channel capture (6.5 s, which SDRtrunk
// decodes end to end): a single continuous pass finds the outbound sync for
// the first 0.6 s and never again, recovering 54 ACCH bursts. The same samples
// fed to a receiver restarted every second recover **301** — the later audio
// is not weaker, the receiver simply cannot see it any more. That 5.6x is the
// largest single factor in Phase 2 yield found so far, and it is a state bug
// rather than a sensitivity limit (issue #915).
//
// A watchdog is the fix rather than a workaround: both reference decoders
// re-acquire continuously, re-seeding carrier recovery from every frame sync
// they detect. This is the minimal form of the same idea — notice that no
// superframe has locked for a while, and let the receiver start over.

// ReacquireIdleSuperframes is how many superframes may pass with no lock
// before the carrier seed is re-taken.
//
// One (0.36 s) measured best on the reference capture, and by a wide margin —
// distinct MAC PDUs recovered went 7 (no watchdog) → 15 (idle 1) → 11 (idle 2)
// → 8 (idle 3). Waiting is pure loss: the receiver is not going to recover on
// its own, so every extra superframe of patience is signalling thrown away.
// The floor is acquisition cost — re-seeding needs roughly a quarter-second of
// samples before it decodes anything, and a capture chopped into 0.25 s pieces
// decodes nothing at all — which is why this does not go below one superframe.
const ReacquireIdleSuperframes = 1

// CarrierWatchdog tracks how long a Phase 2 receiver has gone without locking
// a superframe and says when its carrier recovery should be re-seeded.
//
// The zero value is not usable; construct with NewCarrierWatchdog.
type CarrierWatchdog struct {
	idle  int // dibits seen since the last superframe locked
	limit int
}

// NewCarrierWatchdog returns a watchdog that fires after idleSuperframes
// superframes' worth of dibits with no lock. A non-positive value uses
// ReacquireIdleSuperframes.
func NewCarrierWatchdog(idleSuperframes int) *CarrierWatchdog {
	if idleSuperframes <= 0 {
		idleSuperframes = ReacquireIdleSuperframes
	}
	return &CarrierWatchdog{limit: idleSuperframes * DibitsPerSuperframe}
}

// Observe records a window of dibits in which superframes were locked, and
// reports whether the receiver should now be reset to re-seed its carrier
// estimate. A true return re-arms the watchdog, so the caller resets once and
// then gets a fresh idle period to acquire in rather than being told to reset
// on every subsequent window.
//
// Call it once per DibitSink delivery, before or after draining superframes —
// the order does not matter, since it is the count that carries the signal.
func (w *CarrierWatchdog) Observe(dibits, superframes int) bool {
	if superframes > 0 {
		w.idle = 0
		return false
	}
	w.idle += dibits
	if w.idle < w.limit {
		return false
	}
	w.idle = 0
	return true
}

// Reset re-arms the watchdog without reporting a re-acquisition. Use it when
// the stream itself restarts (a retune, a new call) so a stale idle count from
// the previous channel does not fire immediately.
func (w *CarrierWatchdog) Reset() { w.idle = 0 }
