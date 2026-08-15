package soapyremote

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
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

// mrcBranchDeadMarginDb is how far below the reference branch a diversity
// branch may sit before it is reported as dead. A branch this far down
// contributes essentially nothing to a maximal-ratio combine (MRC weights by
// |h|²), so it is almost always a disconnected antenna, an unset per-channel
// gain, or a front-end that never digitised the second receiver — all operator-
// fixable, and all previously invisible from GopherTrunk's own logs. Issue
// #1062: the reporter had to read SoapySDRServer's UHD debug output to discover
// their B210's RX2B was dark.
const mrcBranchDeadMarginDb = 20.0

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
// Reference selection. StaticCalibrator anchors the phase on branch 0, so the
// first cut hardwired RX0 as the reference — which made the whole stream only as
// good as RX0. If RX0 was dead (antenna off, or its gain never applied) no
// window ever cleared the floor, calibration never fired, and the combiner sat
// in passthrough emitting RX0's noise while a perfectly good RX1 was ignored:
// exactly the "pull the RX2A antenna and everything drops to the noise floor"
// report in #1062. The reference is now the STRONGEST branch of the window,
// re-evaluated while uncalibrated and frozen once the constant locks (the
// branches are handed to StaticCalibrator permuted so the chosen reference is
// its index 0). Which branch anchors the phase is arbitrary — the combine is
// symmetric — so this costs nothing and makes a single live receiver sufficient.
//
// Concurrency. combine() runs only on the single stream goroutine.
// requestRecalibrate() may be called from any goroutine (e.g. SetCenterFreq on
// the control socket); it just arms an atomic flag that the next combine()
// consumes, keeping all StaticCalibrator access on the stream goroutine with no
// lock around the DSP. health() is likewise stream-goroutine only.
type mrcCombiner struct {
	cal      *diversity.StaticCalibrator
	format   sampleFormat
	channels int
	rearm    atomic.Bool

	// refIdx is the branch anchoring the phase: argmax power while uncalibrated,
	// frozen at calibration. ordered is the scratch permutation handed to the
	// calibrator (ordered[0] == branches[refIdx]).
	refIdx  int
	ordered [][]complex64

	// Diagnostics, refreshed every combine(): per-branch mean power and how the
	// last payload split. See health().
	powDbFS   []float64
	gotBranch int
	shortSpan bool
}

func newMRCCombiner(format sampleFormat) *mrcCombiner {
	return &mrcCombiner{
		cal:      diversity.NewStaticCalibrator(diversityChannels),
		format:   format,
		channels: diversityChannels,
		ordered:  make([][]complex64, diversityChannels),
		powDbFS:  make([]float64, diversityChannels),
	}
}

// mrcHealth is a snapshot of the combiner's per-branch state for the operator-
// facing diagnostic line.
type mrcHealth struct {
	powDbFS    []float64 // per-branch mean power; math.Inf(-1) when absent
	refIdx     int       // branch anchoring the phase
	calibrated bool
	branches   int  // branches recovered from the last payload
	want       int  // branches requested
	deadBranch int  // first branch >mrcBranchDeadMarginDb below the reference, else -1
	degenerate bool // the payload did not carry every requested channel
}

// health snapshots the last combine() for logging. Stream goroutine only.
func (m *mrcCombiner) health() mrcHealth {
	h := mrcHealth{
		powDbFS:    append([]float64(nil), m.powDbFS...),
		refIdx:     m.refIdx,
		calibrated: m.cal.Calibrated(),
		branches:   m.gotBranch,
		want:       m.channels,
		deadBranch: -1,
		degenerate: m.gotBranch != m.channels,
	}
	if m.gotBranch != m.channels || m.refIdx >= len(m.powDbFS) {
		return h
	}
	ref := m.powDbFS[m.refIdx]
	for i, p := range m.powDbFS {
		if i == m.refIdx {
			continue
		}
		if ref-p > mrcBranchDeadMarginDb {
			h.deadBranch = i
			break
		}
	}
	return h
}

// requestRecalibrate arms a calibration reset to be applied by the next
// combine(). Call on every retune: a new LO lock re-randomizes the front-end's
// divider phase, invalidating the previous constant.
func (m *mrcCombiner) requestRecalibrate() { m.rearm.Store(true) }

// combine de-interleaves one multi-channel payload (elems = the header's valid
// sample count per channel), takes the one-time phase calibration once a branch
// carries signal, and returns the combined single-branch stream. A payload that
// does not carry every requested channel is passed through as-is rather than
// force-split into fake branches.
func (m *mrcCombiner) combine(payload []byte, elems int) []complex64 {
	if m.rearm.Swap(false) {
		m.cal.Reset()
		m.refIdx = 0
	}
	branches := m.format.deinterleave(payload, m.channels, elems)
	m.gotBranch = len(branches)
	for i := range m.powDbFS {
		m.powDbFS[i] = math.Inf(-1)
	}
	for i, b := range branches {
		if i < len(m.powDbFS) {
			m.powDbFS[i] = refPowerDbFS(b)
		}
	}
	if len(branches) != m.channels {
		// The server did not deliver every channel (see deinterleave). Emitting
		// the payload verbatim keeps the stream a valid single-receiver time
		// series; the health line reports the shortfall.
		if len(branches) == 0 {
			return nil
		}
		m.refIdx = 0
		return branches[0]
	}
	if !m.cal.Calibrated() {
		// Anchor on whichever branch is actually receiving, so one live
		// receiver is enough (issue #1062).
		m.refIdx = argmaxFloat(m.powDbFS)
		if m.powDbFS[m.refIdx] >= mrcCalFloorDbFS {
			// A silent/short reference is rejected inside Calibrate; ignore the
			// error and retry on the next window (stay in passthrough).
			_ = m.cal.Calibrate(m.order(branches))
		}
	}
	out, err := m.cal.Combine(m.order(branches))
	if err != nil {
		// Shape guard (should not happen — branches are equal-length): fall
		// back to the reference branch verbatim rather than drop the chunk.
		return branches[m.refIdx]
	}
	return out
}

// order returns branches permuted so the reference branch is index 0, which is
// the branch StaticCalibrator anchors its phase estimate on. The scratch slice
// is reused across chunks; it only holds slice headers, not sample copies.
func (m *mrcCombiner) order(branches [][]complex64) [][]complex64 {
	m.ordered[0] = branches[m.refIdx]
	n := 1
	for i, b := range branches {
		if i == m.refIdx {
			continue
		}
		m.ordered[n] = b
		n++
	}
	return m.ordered
}

// argmaxFloat returns the index of the largest value (0 for an empty slice).
func argmaxFloat(v []float64) int {
	best := 0
	for i := 1; i < len(v); i++ {
		if v[i] > v[best] {
			best = i
		}
	}
	return best
}

// mrcHealthInterval throttles the per-branch diversity report. The first
// datagram of a stream always reports, so an operator enabling `diversity: mrc`
// sees both branches' levels immediately rather than 30 s later.
const mrcHealthInterval = 30 * time.Second

// diversityReporter surfaces the MRC combiner's per-branch state to the
// operator, rate-limited. Before this, a diversity branch that never digitised
// (disconnected antenna, ungained receiver, a server that honoured only one
// channel of the request) was invisible from GopherTrunk's side: the combined
// stream simply looked like a weak single receiver. Issue #1062.
//
// Not safe for concurrent use; streamLoop owns one.
type diversityReporter struct {
	log      *slog.Logger
	addr     string
	interval time.Duration
	now      func() time.Time

	last time.Time
}

func newDiversityReporter(log *slog.Logger, addr string) *diversityReporter {
	return &diversityReporter{log: log, addr: addr, interval: mrcHealthInterval, now: time.Now}
}

// observe logs one line per interval describing both branches. A branch that is
// missing from the payload or is mrcBranchDeadMarginDb below the reference is a
// WARN naming the fix; a healthy pair is INFO.
func (r *diversityReporter) observe(m *mrcCombiner) {
	now := r.now()
	if !r.last.IsZero() && now.Sub(r.last) < r.interval {
		return
	}
	r.last = now
	h := m.health()
	attrs := []any{
		"addr", r.addr,
		"branch_dbfs", formatBranchPowers(h.powDbFS),
		"reference_branch", h.refIdx,
		"calibrated", h.calibrated,
	}
	switch {
	case h.degenerate:
		r.log.Warn("soapyremote: MRC diversity got "+strconv.Itoa(h.branches)+" of "+strconv.Itoa(h.want)+" channels in the stream — the remote is not delivering the second receiver, so there is no diversity gain. Check that the device has a second RX channel and that the server accepted the 2-channel stream request.",
			append(attrs, "channels_delivered", h.branches, "channels_requested", h.want)...)
	case h.deadBranch >= 0:
		r.log.Warn("soapyremote: MRC diversity branch is dead — it sits more than "+strconv.Itoa(int(mrcBranchDeadMarginDb))+" dB below the reference receiver and contributes nothing to the combine. Check that antenna's connection, and that this RX channel is gained (sdr.soapy_remote[].antennas selects its port).",
			append(attrs, "dead_branch", h.deadBranch)...)
	default:
		r.log.Info("soapyremote: MRC diversity branches", attrs...)
	}
}

// formatBranchPowers renders per-branch mean power for a log line, e.g.
// "ch0=-31.2 ch1=-33.8" (a branch with no samples renders as "silent").
func formatBranchPowers(pow []float64) string {
	var b strings.Builder
	for i, p := range pow {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("ch")
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('=')
		if math.IsInf(p, -1) {
			b.WriteString("silent")
			continue
		}
		b.WriteString(strconv.FormatFloat(p, 'f', 1, 64))
	}
	return b.String()
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
