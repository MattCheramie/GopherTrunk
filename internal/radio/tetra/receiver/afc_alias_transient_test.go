package receiver

import (
	"math"
	"testing"
)

// emaHz reads the smoothed EMA in Hz — the value Process derotates by and
// OffsetHz reports after the next block (track itself never touches a.omega).
func emaHz(a *carrierAFC) float64 { return a.omegaEMA * SymbolRate / (2 * math.Pi) }

// establishTrack primes a carrierAFC at trueHz and feeds it enough accepted
// per-block estimates (small realistic jitter) to corroborate the track past
// omegaEstablishedBlocks — the state a control channel reaches within ~1 s of
// locking.
func establishTrack(a *carrierAFC, trueHz float64) {
	a.track(omegaFromHz(trueHz))
	for i := 0; i <= omegaEstablishedBlocks; i++ {
		jitterHz := float64((i%3)-1) * 90 // −90/0/+90 Hz block-to-block jitter
		a.track(omegaFromHz(trueHz + jitterHz))
	}
}

// TestAFCEstablishedTrackHoldsThroughAliasTransient pins the field failure a
// reporter filmed on a locked, centred TETRA CC: "sometimes AFC spikes to huge
// numbers like -5 kHz". The per-block raw estimate is coarse+fine, and any
// transient coarse-stage bias > f_sym/8 (a neighbour's skirts through the
// channel-filter edge while the wanted carrier fades) lands the raw estimate
// EXACTLY one ±f_sym/4 = ±4500 Hz alias bucket off (the logged spike was
// 5004 ≈ 504 + 4500 Hz). Three such consecutive blocks (~170 ms) used to trip
// the 19 Aug re-prime escape hatch and slam a long-established EMA into the
// wrong bucket — a wrong-bucket derotation that fails every SCH burst until
// the correct estimates re-prime it back. An established track must HOLD
// through an alias-class transient several times the old streak length, and
// keep accepting correct estimates afterwards.
func TestAFCEstablishedTrackHoldsThroughAliasTransient(t *testing.T) {
	const trueHz = 450.0
	a := newCarrierAFC(8)
	establishTrack(a, trueHz)

	// ~340 ms of coherent alias-bucket estimates — twice the fresh-track
	// escape streak, the shape of a sub-second coarse-bias transient.
	for i := 0; i < 2*omegaReprimeStreak; i++ {
		jitterHz := float64((i%3)-1) * 70
		a.track(omegaFromHz(trueHz + 4500 + jitterHz))
		if got := emaHz(a); math.Abs(got-trueHz) > 300 {
			t.Fatalf("established EMA jumped to %.0f Hz after %d alias blocks, want to hold ~%.0f", got, i+1, trueHz)
		}
	}

	// The transient passes; correct estimates must be accepted (not treated
	// as a new outlier cluster) and keep tracking.
	for i := 0; i < 5; i++ {
		a.track(omegaFromHz(trueHz + 60))
	}
	if got := emaHz(a); math.Abs(got-trueHz) > 300 {
		t.Fatalf("EMA did not resume tracking after the transient: %.0f Hz, want ~%.0f", got, trueHz)
	}
}

// TestAFCEstablishedTrackEscapesSustainedAliasShift pins the bound on the
// hold: the re-prime escape hatch must still exist on an established track —
// a carrier genuinely sitting one alias bucket away for over a second
// (adjacent-site takeover, oscillator step) re-primes rather than being
// rejected forever, which would recreate the 19 Aug "locked but deaf" latch.
func TestAFCEstablishedTrackEscapesSustainedAliasShift(t *testing.T) {
	const trueHz = 450.0
	a := newCarrierAFC(8)
	establishTrack(a, trueHz)

	shifted := trueHz + 4500
	for i := 0; i < 2*omegaReprimeStreakEstablished; i++ {
		jitterHz := float64((i%3)-1) * 70
		a.track(omegaFromHz(shifted + jitterHz))
	}
	if got := emaHz(a); math.Abs(got-shifted) > 300 {
		t.Fatalf("EMA never re-primed onto a sustained alias-bucket shift: %.0f Hz, want ~%.0f", got, shifted)
	}
}

// TestAFCEstablishedTrackFastReprimeOnNonAliasJump pins that the slow gate is
// alias-class only: a rejected cluster that is NOT an alias of the tracked
// value (here +3000 Hz — 1500 Hz from the nearest π/2 bucket step, well past
// omegaRejectClusterTol) is a genuine carrier move, and keeps the fast
// omegaReprimeStreak escape even on an established track.
func TestAFCEstablishedTrackFastReprimeOnNonAliasJump(t *testing.T) {
	const trueHz = 450.0
	a := newCarrierAFC(8)
	establishTrack(a, trueHz)

	jumped := trueHz + 3000
	for i := 0; i < omegaReprimeStreak; i++ {
		a.track(omegaFromHz(jumped))
	}
	if got := emaHz(a); math.Abs(got-jumped) > 300 {
		t.Fatalf("non-alias jump did not fast re-prime: %.0f Hz, want ~%.0f", got, jumped)
	}
}
