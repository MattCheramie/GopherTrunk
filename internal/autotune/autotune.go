// Package autotune tracks the per-dongle carrier-frequency error of an SDR
// source and turns it into a digital tuning correction, ported in concept from
// trunk-recorder's AutotuneManager (trunk-recorder/autotune.cc).
//
// Every RTL-SDR crystal is a little off, which shifts the received carrier off
// centre. A demodulator's AFC / carrier-recovery loop measures that residual
// offset in Hz once it locks (GopherTrunk reads it from
// internal/radio/p25/phase1/receiver.Receiver.AFCOffsetHz). This package keeps
// a running average of the last MaxHistory measurements and exposes it as an
// offset the caller applies digitally — by shifting the channel's down-
// converter target — so the next recorder/decoder hands its loop a starting
// frequency that is already close to lock. It never rewrites the dongle's
// hardware PPM; it only reports a suggested config value the operator can bake
// in by hand.
//
// A Manager is per physical dongle (keyed by serial in a Registry): a control
// SDR and a voice SDR have independent crystal errors and independent
// corrections.
package autotune

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
)

const (
	// MaxHistory is how many recent error measurements feed the running
	// average. Matches trunk-recorder's MAX_HISTORY.
	MaxHistory = 20
	// PPMThreshold is the correction magnitude (in PPM, relative to the
	// channel centre frequency) above which the source is flagged as
	// mis-configured. Matches trunk-recorder's PPM_THRESHOLD.
	PPMThreshold = 3.5
	// SuggestedErrorRounding rounds the suggested config value to the
	// nearest N Hz. Matches trunk-recorder's SUGGESTED_ERROR_ROUNDING.
	SuggestedErrorRounding = 10
	// WarmupSamples is how many measurements must accumulate before the
	// running average is trusted enough to apply. Until then Ready() is
	// false and callers apply no correction, so a single noisy first
	// measurement can't yank tuning by hundreds of Hz before the average
	// has had a chance to settle.
	WarmupSamples = 3
	// MaxAbsErrorHz bounds a single measurement's plausible magnitude when
	// the channel centre is unknown (the voice path passes centre 0). A
	// real crystal error is a few PPM — single-digit Hz on a disciplined
	// USRP, low hundreds on a loose RTL crystal at VHF/UHF. A per-call AFC
	// residual far beyond that is a measurement glitch (an unconverged loop,
	// a data-dependent bias snapshot at end-of-call), not drift, and folding
	// it in produces the runaway kHz corrections this guard prevents. ~1.5 kHz
	// is ≈3.3 PPM at 450 MHz — comfortably above any correction worth making.
	MaxAbsErrorHz = 1500
)

// Manager accumulates carrier-error measurements for one SDR source and serves
// the running-average correction. It is safe for concurrent use: the control
// decoder and the voice composer both call into the same Manager.
type Manager struct {
	log    *slog.Logger
	serial string

	mu      sync.Mutex
	history []int // most-recent-first; capped at MaxHistory
	avg     int   // cached running average, in Hz
	samples int   // count of accepted measurements (for warm-up gating)
}

// NewManager returns a Manager labelled with the dongle serial for logging. A
// nil logger is tolerated (logging becomes a no-op).
func NewManager(serial string, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.New(slog.NewTextHandler(discard{}, nil))
	}
	return &Manager{log: log, serial: serial}
}

// AddErrorMeasurement folds one observation into the running average. observedHz
// is the loop's freshly-measured residual offset; currentOffsetHz is the
// autotune offset that was already applied while that residual was observed, so
// total = observed + applied is the source's true error. centerFreqHz is the
// channel centre the measurement was taken at, used only for the PPM sanity
// warning. Port of AutotuneManager::add_error_measurement.
func (m *Manager) AddErrorMeasurement(observedHz, currentOffsetHz int, centerFreqHz uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := observedHz + currentOffsetHz

	// Reject implausible measurements before they can poison the average.
	// A real crystal error is bounded by a few PPM of the channel centre;
	// anything beyond that is a measurement glitch, not drift. Drop it (and
	// say so at Debug) rather than averaging in a value that would drag the
	// applied correction into the kHz.
	if bound := m.rejectBoundHz(centerFreqHz); abs(total) > bound {
		m.log.Debug("autotune: measurement rejected (implausible, treated as glitch)",
			"serial", m.serial, "observed_hz", observedHz, "applied_hz", currentOffsetHz,
			"total_hz", total, "bound_hz", bound)
		return
	}

	// Prepend, then trim the oldest past MaxHistory (deque push_front /
	// pop_back).
	m.history = append([]int{total}, m.history...)
	if len(m.history) > MaxHistory {
		m.history = m.history[:MaxHistory]
	}
	m.samples++

	sum := 0
	for _, e := range m.history {
		sum += e
	}
	m.avg = sum / len(m.history)

	m.log.Debug("autotune: measurement", "serial", m.serial,
		"observed_hz", observedHz, "applied_hz", currentOffsetHz,
		"total_hz", total, "avg_hz", m.avg, "samples", len(m.history),
		"warm", m.samples >= WarmupSamples)

	if centerFreqHz != 0 {
		ppm := float64(m.avg) / (float64(centerFreqHz) / 1e6)
		if math.Abs(ppm) > PPMThreshold {
			m.log.Warn("autotune: correction exceeds PPM threshold; verify configured ppm",
				"serial", m.serial, "avg_hz", m.avg, "ppm", ppm,
				"threshold_ppm", PPMThreshold, "center_mhz", float64(centerFreqHz)/1e6)
		}
	}
}

// AverageError returns the cached running-average correction in Hz. Callers
// apply it digitally as targetHz - AverageError(). Port of get_average_error.
// This is the raw average regardless of warm-up; callers that drive tuning
// should gate on Ready() so an unsettled early average is not applied.
func (m *Manager) AverageError() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.avg
}

// Ready reports whether enough measurements have accumulated (>= WarmupSamples)
// for the running average to be trusted as an applied tuning correction.
// Before that, callers should apply no correction.
func (m *Manager) Ready() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.samples >= WarmupSamples
}

// Correction returns the average correction to apply, or 0 until warm-up
// completes. It is the value tuning call sites should use.
func (m *Manager) Correction() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.samples < WarmupSamples {
		return 0
	}
	return m.avg
}

// rejectBoundHz is the largest plausible single-measurement magnitude. With a
// known channel centre it is the PPM-threshold equivalent (a measurement worse
// than a misconfigured-dongle warning is not a fine correction to chase); with
// an unknown centre (the voice path) it falls back to the absolute cap.
func (m *Manager) rejectBoundHz(centerFreqHz uint32) int {
	if centerFreqHz == 0 {
		return MaxAbsErrorHz
	}
	bound := int(math.Round(PPMThreshold * float64(centerFreqHz) / 1e6))
	if bound < MaxAbsErrorHz {
		bound = MaxAbsErrorHz
	}
	return bound
}

// abs returns the absolute value of an int.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// StatusString renders the live correction and a suggested config value, given
// the channel centre frequency and the operator's currently-configured error in
// Hz (derive from the dongle's configured ppm: round(ppm * centerHz/1e6)). Port
// of AutotuneManager::get_status_string, adapted to GopherTrunk's ppm config: it
// also reports the suggested value as a ppm so it can be pasted into the device
// block.
func (m *Manager) StatusString(centerFreqHz uint32, initialErrorHz int) string {
	correction := m.AverageError()
	total := initialErrorHz - correction
	suggested := int(math.Round(float64(total)/SuggestedErrorRounding) * SuggestedErrorRounding)

	if centerFreqHz == 0 {
		return fmt.Sprintf("AutoTune: %+d Hz, suggested error: %+d Hz", correction, suggested)
	}
	suggestedPPM := float64(suggested) / (float64(centerFreqHz) / 1e6)
	return fmt.Sprintf("AutoTune: %+d Hz, suggested error: %+d Hz (ppm %+.1f)",
		correction, suggested, suggestedPPM)
}

// Reset clears the measurement history. Port of AutotuneManager::reset. The
// cached average is intentionally left in place so an in-flight correction is
// not yanked to zero mid-call (matches trunk-recorder, which only clears the
// deque).
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = nil
}

// Registry hands out per-serial Managers. When disabled, Get returns nil so
// every call site collapses to the pre-autotune behaviour at zero cost.
type Registry struct {
	enabled bool
	log     *slog.Logger

	mu  sync.Mutex
	mgr map[string]*Manager
}

// NewRegistry builds a Registry. When enabled is false, Get always returns nil.
func NewRegistry(enabled bool, log *slog.Logger) *Registry {
	return &Registry{enabled: enabled, log: log, mgr: map[string]*Manager{}}
}

// Enabled reports whether autotune is on.
func (r *Registry) Enabled() bool { return r != nil && r.enabled }

// Get returns the shared Manager for a dongle serial, creating it on first use.
// Returns nil when the Registry is nil or disabled, so callers can write
// `if m := reg.Get(serial); m != nil { ... }`.
func (r *Registry) Get(serial string) *Manager {
	if !r.Enabled() {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.mgr[serial]
	if !ok {
		m = NewManager(serial, r.log)
		r.mgr[serial] = m
	}
	return m
}

// discard is an io.Writer that drops everything, used to build a no-op logger.
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
