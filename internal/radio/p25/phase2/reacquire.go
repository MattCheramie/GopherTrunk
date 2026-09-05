package phase2

// Loss-of-lock re-acquisition for the Phase 2 receiver.
//
// A receiver that stops locking superframes partway through a call does not
// recover on its own. This notices that no superframe has locked for a while
// and lets the caller start the receiver over, which is the minimal form of
// what both reference decoders do continuously — they re-acquire from every
// frame sync they detect.
//
// **On the diagnosis.** This was written as a *carrier* watchdog, on the
// reasoning that the receiver takes its coarse carrier estimate exactly once
// per stream and nothing ever re-takes it, so a drifting channel falls out of
// the Costas loop's ±SymbolRate/8 pull-in and never comes back. That reasoning
// was wrong, and the fix worked for a different reason. Driving the same
// watchdog schedule with each piece of receiver state in turn attributes the
// loss completely: re-seeding carrier recovery alone changes the yield by
// literally nothing (byte-identical to no watchdog at all), while resetting
// the Gardner timing loop alone recovers everything a full reset does. The
// receiver was not losing its carrier; it was losing the eye, because the
// timing loop's feedback sign was inverted and it was settling on the
// transition instant. See internal/dsp/sync/gardner.go.
//
// With that sign corrected this watchdog is no longer load-bearing on the
// reference corpus — it fires once across a 6.5 s capture and changes no
// counts — but it is kept: a genuine loss of lock (fade, retune, underrun)
// still wants a way back, and nothing else provides one.
//
// The measurements the original text quoted are superseded. On the reference
// capture, continuous decode with the timing fix recovers 830 ACCH bursts and
// 31 distinct MAC PDUs — matching SDRtrunk's decode of the same file exactly,
// with nothing fabricated and nothing missed — against 87 bursts and 9 PDUs
// before it (issue #915).

// ReacquireIdleSuperframes is how many superframes may pass with no lock
// before the receiver is restarted.
//
// One (0.36 s) measured best when this was the only thing standing between the
// decoder and a lost channel, and by a wide margin — distinct MAC PDUs went 7
// (no watchdog) → 15 (idle 1) → 11 (idle 2) → 8 (idle 3). Waiting was pure
// loss, because the receiver was never going to recover on its own. Now that
// the underlying timing bug is fixed it rarely fires at all, so the value is
// no longer critical; it is left at one superframe because the floor still
// holds — a restart needs roughly a quarter-second of samples before it
// decodes anything, so anything shorter cannot pay for itself.
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
