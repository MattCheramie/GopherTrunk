package soapyremote

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/diversity"
)

// parseDiversity validates the Spec.Diversity mode. "" / "none" / "off" disable
// diversity (ordinary single-channel stream); "mrc" enables 2-channel
// phase-coherent MRC. Any other value is rejected at open time.
func parseDiversity(mode string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none", "off":
		return false, nil
	case "mrc":
		return true, nil
	default:
		return false, fmt.Errorf("soapyremote: diversity %q not supported (want \"mrc\" or empty)", mode)
	}
}

// diversityChannels is the RX channel count for phase-coherent MRC diversity: a
// shared-LO front-end's RX0 (reference) + RX1. Only 2-branch MRC is supported.
const diversityChannels = 2

// diversityActivateLead is how far in the future (relative to the remote
// hardware clock) a multi-channel stream start is scheduled. It only needs to
// cover the RPC round-trip plus the radio's command latency; UHD's own
// examples use ~100 ms, and being generous costs one-time startup delay only.
const diversityActivateLead = 200 * time.Millisecond

// mrcCalFloorDbFS is the reference-branch mean-power floor (dBFS, 0 = unit
// amplitude) above which the combiner takes its one-time phase calibration.
// Below it the window is treated as noise and calibration is deferred (Combine
// passes the reference branch through until then). A carrier on a locked trunk
// sits well above this; thermal noise on these front-ends is far below it.
// Heuristic bootstrap — see mrcCombiner. Issue #1062.
const mrcCalFloorDbFS = -40.0

// mrcCombiner turns a two-branch (RX0/RX1) SoapyRemote stream into one
// phase-coherently combined []complex64 via diversity.StaticCalibrator.
//
// The AD9361/B210 RX0↔RX1 phase offset is a constant per LO lock (fixed trace/
// cable skew + power-up divider phase), so a single calibration on the first
// signal-bearing window is sufficient — no per-sample phase tracking. Until it
// calibrates, StaticCalibrator.Combine returns a copy of the reference branch,
// so the combined stream is always at least the primary receiver verbatim (never
// a blind sum that could cancel). A per-branch amplitude difference (e.g. RX1 at
// a different gain) is absorbed by the calibration — StaticCalibrator estimates a
// complex gain, weighting each branch by both phase and amplitude.
//
// Calibration trigger. The driver has no decoder-sync feedback, so this
// bootstraps on the first window whose reference-branch power clears
// mrcCalFloorDbFS. Issue #1062's stated ideal (calibrate on the first decoded
// protocol sync burst) is a follow-up needing a driver↔decoder signal; the
// power-floor bootstrap is a safe stand-in because the phase constant is
// identical whatever high-SNR window anchors it.
//
// Concurrency. combine() runs only on the single stream goroutine.
// requestRecalibrate() may be called from any goroutine (e.g. SetCenterFreq on
// the control socket); it just arms an atomic flag that the next combine()
// consumes, keeping all StaticCalibrator access on the stream goroutine with no
// lock around the DSP.
type mrcCombiner struct {
	cal      *diversity.StaticCalibrator
	format   sampleFormat
	channels int
	rearm    atomic.Bool
}

func newMRCCombiner(format sampleFormat) *mrcCombiner {
	return &mrcCombiner{
		cal:      diversity.NewStaticCalibrator(diversityChannels),
		format:   format,
		channels: diversityChannels,
	}
}

// requestRecalibrate arms a calibration reset to be applied by the next
// combine(). Call on every retune: a new LO lock re-randomizes the front-end's
// divider phase, invalidating the previous constant.
func (m *mrcCombiner) requestRecalibrate() { m.rearm.Store(true) }

// combine de-interleaves one multi-channel payload, takes the one-time phase
// calibration once the reference branch carries signal, and returns the combined
// single-branch stream. An empty/degenerate payload yields the reference branch.
func (m *mrcCombiner) combine(payload []byte) []complex64 {
	if m.rearm.Swap(false) {
		m.cal.Reset()
	}
	branches := m.format.deinterleave(payload, m.channels)
	if !m.cal.Calibrated() && refPowerDbFS(branches[0]) >= mrcCalFloorDbFS {
		// A silent/short reference is rejected inside Calibrate; ignore the
		// error and retry on the next window (stay in reference passthrough).
		_ = m.cal.Calibrate(branches)
	}
	out, err := m.cal.Combine(branches)
	if err != nil {
		// Shape guard (should not happen — branches are equal-length): fall
		// back to the reference branch verbatim rather than drop the chunk.
		return branches[0]
	}
	return out
}

// refPowerDbFS is the mean power of a branch in dBFS (0 dBFS = unit amplitude),
// -Inf for an empty or silent branch.
func refPowerDbFS(x []complex64) float64 {
	if len(x) == 0 {
		return math.Inf(-1)
	}
	var acc float64
	for _, z := range x {
		r, i := float64(real(z)), float64(imag(z))
		acc += r*r + i*i
	}
	mean := acc / float64(len(x))
	if mean <= 0 {
		return math.Inf(-1)
	}
	return 10 * math.Log10(mean)
}
