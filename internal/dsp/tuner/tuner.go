// Package tuner extracts narrow-band baseband IQ for one or more frequency
// offsets from a single wide-band SDR IQ stream. It is the building block
// that lets one physical SDR feed several protocol receivers tuned to
// different repeater carriers within the dongle's IQ bandwidth.
//
// A Bank owns the wide-band-to-narrow-band conversion and dispatches the
// resulting per-offset IQ chunks to registered sinks. Two implementations
// are provided:
//
//   - DDCBank: one independent complex-NCO mixer + rational polyphase
//     resampler per tap. Cost scales linearly with the number of taps but
//     there is no constraint on the tap offsets - they can sit anywhere
//     inside the dongle's usable IQ band. Best for a small number of taps.
//
//   - ChannelizerBank: a single M-channel critically-sampled polyphase
//     channelizer (internal/dsp/channelizer) splits the input into evenly-
//     spaced bins; a small fine-tune DDC on the bin nearest each tap
//     offset cleans up the residual. The wide-band filter cost is shared
//     across all taps, which wins at higher tap counts.
//
// Both implementations decimate to the same narrow-band rate - typically
// 48 kHz, matching what the existing 4800-baud C4FM receivers
// (DMR / P25 / NXDN / dPMR / YSF / D-STAR) are matched-filter-tuned for.
package tuner

import "errors"

// SinkFunc receives narrow-band IQ for one tap each time the bank advances.
// The slice is owned by the bank and is reused across calls - copy what you
// need before returning. Empty slices are passed through unchanged so sinks
// see a faithful sample-rate timeline.
type SinkFunc func(out []complex64)

// Bank is the common interface implemented by DDCBank and ChannelizerBank.
// AddTap is called once per repeater offset before the first Process call;
// dynamic tap add/remove is intentionally out of scope for v1.
type Bank interface {
	// AddTap registers a tap at the given offset from the dongle's center
	// frequency (positive = above center, negative = below). The sink is
	// invoked once per Process call with the narrow-band IQ for this tap.
	AddTap(offsetHz float64, sink SinkFunc) error

	// Process consumes one chunk of wide-band IQ at InputRateHz and
	// invokes every registered sink with the corresponding narrow-band
	// IQ at OutputRateHz. src is not retained.
	Process(src []complex64)

	// InputRateHz returns the wide-band sample rate the bank consumes.
	InputRateHz() float64

	// OutputRateHz returns the narrow-band per-tap sample rate the bank
	// actually emits. It equals the construction target when the rational
	// resampler hits it exactly, but may differ by a fraction of a percent
	// when the exact ratio exceeds the resampler caps (issue #550). Symbol
	// clocks must be built from this value, not the nominal target.
	OutputRateHz() float64

	// Reset clears all internal filter / NCO state. Called when the
	// upstream SDR is restarted or retuned so stale samples don't bleed
	// into the next stream.
	Reset()
}

// ErrOffsetOutOfBand is returned by AddTap when the requested offset falls
// outside the usable portion of the dongle's IQ band (defined as
// ±InputRateHz/2 with an implementation-specific guard band).
var ErrOffsetOutOfBand = errors.New("tuner: offset is outside the usable IQ band")
