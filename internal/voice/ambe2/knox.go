package ambe2

import (
	"fmt"
	"sync"
)

// Knox / call-alert AMBE+2 dual-tone overrides.
//
// AMBE+2 tone frames with b1 ∈ [144, 163] are vendor-specific
// "knox" or "call-alert" pairs — Motorola Trbo, Hytera, and
// generic-AMBE+2 implementations all use slightly different
// (frequency_a, frequency_b) pairings for this range, and the
// public AMBE+2 spec doesn't document them. Without an override,
// the decoder routes these indices through the §6.4 silence path.
//
// Operators (or in-tree tests) that have a per-vendor reference
// frequency table can SetKnoxTone to register the pairs they care
// about. Decode then synthesises the same summed-sinewave dual-tone
// it produces for DTMF (b1 ∈ [128, 143]), with phase continuity
// across consecutive tone frames.

// KnoxIndexLow / KnoxIndexHigh are the inclusive bounds of the
// knox / call-alert b1 range. Indices outside [144, 163] are
// either DTMF (already routed via ambeDualToneTable) or invalid.
const (
	KnoxIndexLow  = 144
	KnoxIndexHigh = 163
)

// knoxToneTable is the runtime extension layer for vendor-specific
// AMBE+2 knox / call-alert dual-tone frequencies. Indices map
// b1 - KnoxIndexLow → (freqA, freqB) in Hz. Zero entries route
// through silence.
//
// Guarded by knoxMu (RWMutex): SetKnoxTone takes the write lock,
// the decoder's tone-frame branch takes the read lock once per
// matching frame. Read latency is ~50 ns vs. ~5 µs for the
// synthesis itself, so the lock cost is in the noise.
var (
	knoxMu       sync.RWMutex
	knoxToneTable [KnoxIndexHigh - KnoxIndexLow + 1][2]float64
)

// SetKnoxTone registers a vendor-specific dual-tone pair for the
// supplied knox b1 index. b1 must be in [KnoxIndexLow, KnoxIndexHigh];
// values outside that range return an error so a typo doesn't
// silently shadow a DTMF pair. (freqA, freqB) of (0, 0) clears the
// override and routes b1 through silence again.
//
// Safe to call concurrently from any goroutine; the decoder's
// tone-frame branch reads under a read lock. Typically called from
// package init() in a per-vendor sub-package, or from a test that
// wants to assert dual-tone synthesis behaviour for a specific
// b1 index.
func SetKnoxTone(b1 int, freqA, freqB float64) error {
	if b1 < KnoxIndexLow || b1 > KnoxIndexHigh {
		return fmt.Errorf("ambe2: knox b1 index %d outside range [%d, %d]",
			b1, KnoxIndexLow, KnoxIndexHigh)
	}
	knoxMu.Lock()
	defer knoxMu.Unlock()
	knoxToneTable[b1-KnoxIndexLow] = [2]float64{freqA, freqB}
	return nil
}

// KnoxTone returns the registered (freqA, freqB) pair for the
// supplied knox b1 index. The second return is false when b1 is
// outside the knox range OR no override has been set (the pair is
// (0, 0)). Decoder.Decode uses this in its tone-frame branch to
// decide whether to synthesise a dual-tone or route to silence.
func KnoxTone(b1 int) (float64, float64, bool) {
	if b1 < KnoxIndexLow || b1 > KnoxIndexHigh {
		return 0, 0, false
	}
	knoxMu.RLock()
	pair := knoxToneTable[b1-KnoxIndexLow]
	knoxMu.RUnlock()
	if pair[0] == 0 && pair[1] == 0 {
		return 0, 0, false
	}
	return pair[0], pair[1], true
}

// ClearKnoxTones removes every registered knox override. Useful in
// tests that need a clean baseline; production code should set
// overrides once at startup and leave them.
func ClearKnoxTones() {
	knoxMu.Lock()
	defer knoxMu.Unlock()
	for i := range knoxToneTable {
		knoxToneTable[i] = [2]float64{}
	}
}
