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

// diversityMode is the configured Spec.Diversity behaviour.
type diversityMode int

const (
	// diversityOff is an ordinary single-channel stream.
	diversityOff diversityMode = iota
	// diversityMRCTracking is 2-channel phase-coherent MRC whose branch gain is
	// re-estimated from the stream and smoothed. This is what "mrc" selects.
	diversityMRCTracking
	// diversityMRCStatic is the same combine with a one-shot frozen gain — the
	// escape hatch for comparing against the pre-tracking behaviour on a
	// shared-LO front-end, where the gain really is a constant.
	diversityMRCStatic
)

func (m diversityMode) enabled() bool { return m != diversityOff }

func (m diversityMode) String() string {
	switch m {
	case diversityMRCTracking:
		return "tracking"
	case diversityMRCStatic:
		return "static"
	default:
		return "off"
	}
}

// parseDiversity validates the Spec.Diversity mode. "" / "none" / "off" disable
// diversity; "mrc" enables tracking MRC; "mrc-static" enables the one-shot
// variant. Any other value is rejected at open time.
func parseDiversity(mode string) (diversityMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none", "off":
		return diversityOff, nil
	case "mrc":
		return diversityMRCTracking, nil
	case "mrc-static", "mrc_static":
		return diversityMRCStatic, nil
	default:
		return diversityOff, fmt.Errorf("soapyremote: diversity %q not supported (want \"mrc\", \"mrc-static\" or empty)", mode)
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

// Calibration gates.
//
// These are COHERENCE thresholds, not power thresholds, and that distinction is
// the point. The predecessor gated calibration on the reference branch clearing
// -40 dBFS of absolute mean power, which is gain-dependent: an operator running
// an X310 + TwinRX had to raise front-end gain from 65.0 dB to 82.0 dB — not
// because the radio needed it, but to push a number past a software constant
// (they landed at -39.8 dBFS, clearing it by 0.2 dB). Normalised cross-
// correlation asks the question that actually matters — are these two receivers
// seeing the same signal through a constant complex gain? — and is invariant to
// how much gain is in front of them. See diversity.CrossStats.
//
// And they are thresholds on the ESTIMATE'S QUALITY, not on |rho| itself,
// because a fixed |rho| constant turned out to be a staging trap of its own.
// |rho| measured on the wideband stream is diluted by every hertz of
// noise-only bandwidth around the coherent carrier (rho_wb ~
// rho_ch*sqrt(f0*f1) for in-channel power fractions f_k), so a fixed 0.5 bar
// made calibration depend on the configured capture bandwidth and on each
// branch's own noise floor. An X310 operator's 18 Aug 2026 A/B proved it: at
// 200/250 kHz and 70 dB gain the wideband |rho| sat at ~0.16 — while the
// control channel decoded at 1425 CRC-clean BSCH and the branch phase was
// measurable to ~4 degrees — and MRC never calibrated until they raised RF
// gain 5 dB purely to push the number past the constant (their weak branch's
// floor did not scale with gain, so the +5 dB lifted its in-channel fraction
// and with it the diluted rho). The gates below bound the projected PHASE
// ERROR of the window's least-squares estimate instead, which makes the
// minimum rho fall as 1/sqrt(window) while staying ~8x/~5x above the
// noise-only floor sqrt(pi/4N) — noise still cannot clear them on luck (the
// odds are exp(-N*rho^2), about e^-50 for lock). See
// diversity.TrackingOptions.
const (
	// mrcLockPhaseSigmaRad bounds the phase error of the FIRST estimate
	// (~5.7 deg). A one-shot calibration lives with its first estimate forever,
	// so the lock bar is stricter than the tracking bar.
	mrcLockPhaseSigmaRad = diversity.DefaultLockPhaseSigmaRad
	// mrcTrackPhaseSigmaRad bounds each subsequent update (~9.2 deg), whose
	// error averages away over ~1/alpha windows and is slew-clamped besides.
	mrcTrackPhaseSigmaRad = diversity.DefaultTrackPhaseSigmaRad
	// mrcBranchFloorPower is a linear AC-power floor that rejects a DIGITALLY
	// DEAD branch — all zeros, or a receiver that never digitised — where the
	// correlation is 0/0. -100 dBFS; one CS16 LSB is -90.3 dBFS, so a
	// single-LSB-dithered receiver still counts as alive. This is NOT a
	// signal-presence test: that job belongs to the coherence gate.
	mrcBranchFloorPower = 1e-10
)

// mrcCalWindowMs is the calibration/tracking accumulation window, in
// milliseconds of stream time. One datagram is nowhere near enough for a
// least-squares phase estimate — 184 samples at the default MTU, which at
// |rho| = 0.35 pins the phase only to about 9.5 degrees. A window in units of
// TIME rather than datagrams keeps the estimator's quality independent of the
// configured MTU and sample rate.
const mrcCalWindowMs = 2.0

// Window clamps for when the sample rate is unknown or extreme.
const (
	mrcCalWindowMinSamples     = 4096
	mrcCalWindowMaxSamples     = 262144
	mrcCalWindowDefaultSamples = 8192
)

// mrcTrackTauMs is the tracking loop's 1/e time constant. 200 ms is roughly 14
// TETRA bursts: far slower than one burst, so the phase a burst sees is
// effectively constant, and far faster than the relative drift of two
// independently-locked front-end PLLs.
const mrcTrackTauMs = 200.0

// mrcTrackAlpha is the one-pole coefficient implied by the window and the time
// constant. It must be derived from the ACTUAL window duration — window
// samples over the stream rate — not the nominal mrcCalWindowMs: at low
// stream rates the min-samples clamp stretches the window (250 kHz → 4096
// samples = 16.4 ms), and an alpha still assuming 2 ms would make the loop's
// 1/e time ~1.6 s instead of the documented 200 ms — 8× too slow, exactly on
// the narrow-capture rigs whose independently-locked PLLs need the tracking.
func mrcTrackAlpha(mode diversityMode, window int, rateHz float64) float64 {
	if mode == diversityMRCStatic {
		return 0 // one-shot: freeze the first accepted window
	}
	windowMs := mrcCalWindowMs
	if rateHz > 0 && window > 0 {
		windowMs = float64(window) / rateHz * 1000
	}
	alpha := windowMs / mrcTrackTauMs
	if alpha > 1 {
		alpha = 1
	}
	return alpha
}

// mrcCalOptions builds the calibrator options for a mode and window. rateHz is
// the stream rate the window was sized for (0 = unknown, nominal alpha).
func mrcCalOptions(mode diversityMode, window int, rateHz float64) diversity.TrackingOptions {
	return diversity.TrackingOptions{
		WindowSamples:      window,
		Alpha:              mrcTrackAlpha(mode, window, rateHz),
		LockPhaseSigmaRad:  mrcLockPhaseSigmaRad,
		TrackPhaseSigmaRad: mrcTrackPhaseSigmaRad,
		MinBranchPower:     mrcBranchFloorPower,
	}
}

// mrcCalWindowSamples converts the window from stream time to samples for a
// given rate, clamped. A zero/unknown rate takes the default.
func mrcCalWindowSamples(rateHz float64) int {
	if rateHz <= 0 {
		return mrcCalWindowDefaultSamples
	}
	n := int(rateHz * mrcCalWindowMs / 1000)
	if n < mrcCalWindowMinSamples {
		return mrcCalWindowMinSamples
	}
	if n > mrcCalWindowMaxSamples {
		return mrcCalWindowMaxSamples
	}
	return n
}

// mrcBranchDeadMarginDb is how far below the loudest branch a diversity branch
// may sit before it is reported as dead. A branch this far down
// contributes essentially nothing to a maximal-ratio combine (MRC weights by
// |h|²), so it is almost always a disconnected antenna, an unset per-channel
// gain, or a front-end that never digitised the second receiver — all operator-
// fixable, and all previously invisible from GopherTrunk's own logs. Issue
// #1062: the reporter had to read SoapySDRServer's UHD debug output to discover
// their B210's RX2B was dark.
const mrcBranchDeadMarginDb = 20.0

// mrcCombiner turns a two-branch (RX0/RX1) SoapyRemote stream into one
// phase-coherently combined []complex64 via diversity.TrackingCalibrator.
//
// A per-branch amplitude difference (e.g. RX1 at a different gain) is absorbed
// by the estimate along with the phase, since the calibrator estimates a complex
// gain. Until it calibrates, Combine returns a copy of the reference branch, so
// the combined stream is always at least the primary receiver verbatim — never a
// blind sum that could cancel.
//
// Calibration trigger. The driver has no decoder-sync feedback, so it gates on
// the normalised cross-correlation between the branches (see the gate constants
// above). Issue #1062's stated ideal — calibrate on the first decoded protocol
// sync burst — is not just unnecessary here but worse: coherence is available
// every window, needs no protocol knowledge, and works before lock.
//
// Static vs tracking. A frozen constant is right for a shared-LO, single-chip
// front-end (AD9361/B210), where the branch phase offset is fixed trace skew
// plus a power-up divider phase. It is wrong for two branches on separate
// daughterboards with independent PLLs — an X310 with two TwinRX cards — which
// are frequency-locked to a common reference but whose relative phase is random
// at each lock and walks afterwards. "mrc" therefore tracks; "mrc-static" keeps
// the one-shot behaviour for comparison.
//
// Reference selection. The calibrator anchors the phase on branch 0, so the
// first cut hardwired RX0 as the reference — which made the whole stream only as
// good as RX0. If RX0 was dead (antenna off, or its gain never applied) no
// window ever calibrated and the combiner sat in passthrough emitting RX0's
// noise while a perfectly good RX1 was ignored: exactly the "pull the RX2A
// antenna and everything drops to the noise floor" report in #1062. The
// reference is now picked from the first window that carries any live branch and
// then held (see mrcRefDeadDatagrams), the branches being handed to the calibrator
// permuted so the chosen reference is its index 0.
//
// KNOWN LIMITATION. The combine happens on the WIDEBAND stream, so one complex
// scalar optimises whole-band SNR, not the target channel's. That is exact only
// if the two branches differ by a frequency-flat constant. With two physically
// separated antennas each carrier in the span arrives with its own phase
// difference set by geometry and direction of arrival, so a scalar aligns
// whichever is loudest and partially cancels the rest. Co-located, co-polarised
// antennas are approximately fine; metres apart are not, and the symptom is a
// wideband coherence stuck around 0.3-0.5 that no amount of tracking improves.
// The fix for that regime is combining after the per-channel DDC, one gain per
// narrowband channel — a much larger change. The coherence figure in the health
// line is what makes this visible instead of silent.
//
// Concurrency. combine() runs only on the single stream goroutine.
// requestRecalibrate() may be called from any goroutine (e.g. SetCenterFreq on
// the control socket); it just arms an atomic flag that the next combine()
// consumes, keeping all calibrator access on the stream goroutine with no lock
// around the DSP. health() is likewise stream-goroutine only.
type mrcCombiner struct {
	cal      *diversity.TrackingCalibrator
	mode     diversityMode
	format   sampleFormat
	channels int
	window   int // samples per branch per estimation window
	// lockGate is the minimum window |rho| the calibrator will lock on at this
	// window length, kept for the health line so the operator sees the actual
	// bar their stream is held to rather than a constant from a doc.
	lockGate float64
	rearm    atomic.Bool

	// rec, when non-nil, dumps the pre-combine per-branch IQ. nil is the
	// ordinary case; a nil *branchRecorder's write is a no-op.
	rec *branchRecorder

	// refIdx is the branch anchoring the phase, -1 until the first live window.
	// ordered is the scratch permutation handed to the calibrator
	// (ordered[0] == branches[refIdx]).
	refIdx  int
	ordered [][]complex64

	// refDeadDatagrams counts consecutive datagrams in which the incumbent
	// reference looked dead relative to a challenger; see mrcRefDeadDatagrams.
	refDeadDatagrams int

	// Diagnostics, refreshed every combine(): per-branch mean power and how the
	// last payload split. See health().
	powDbFS   []float64
	gotBranch int
}

// mrcRefDeadDatagrams is how many consecutive datagrams the reference branch
// must look dead before the anchor moves. Moving the anchor is a phase discontinuity
// by construction — a different receiver down a different LO path — so it is
// reserved for a genuinely dead reference and never for a marginal power
// difference. The predecessor recomputed argmax on EVERY datagram while
// uncalibrated, so an ordinary ~1 dB crossover between two healthy branches
// flipped the anchor and handed the demodulator an arbitrary phase step.
const mrcRefDeadDatagrams = 4

func newMRCCombiner(format sampleFormat, mode diversityMode, rateHz float64) *mrcCombiner {
	window := mrcCalWindowSamples(rateHz)
	opts := mrcCalOptions(mode, window, rateHz)
	return &mrcCombiner{
		window:   window,
		lockGate: opts.LockGate(window),
		cal:      diversity.NewTrackingCalibrator(diversityChannels, opts),
		mode:     mode,
		format:   format,
		channels: diversityChannels,
		refIdx:   -1,
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

	// coherence is |rho| of the last completed estimation window: the operator-
	// facing answer to "is the second receiver contributing anything?", and the
	// number that replaces "raise the gain until calibrated goes true".
	coherence float64
	// gainDb / phaseDeg describe the applied branch-1 gain relative to the
	// reference. On a shared-LO front-end phaseDeg is a constant; on separate
	// daughterboards it visibly walks, which is how an operator can tell the two
	// apart from the log alone.
	gainDb   float64
	phaseDeg float64
	mode     string
	updates  uint64
	holds    uint64
	// lockGate is the effective minimum window |rho| for a first lock at the
	// stream's window length (see mrcCombiner.lockGate).
	lockGate float64
}

// health snapshots the last combine() for logging. Stream goroutine only.
func (m *mrcCombiner) health() mrcHealth {
	updates, holds := m.cal.Counters()
	h := mrcHealth{
		powDbFS:    append([]float64(nil), m.powDbFS...),
		refIdx:     m.refIdx,
		calibrated: m.cal.Calibrated(),
		branches:   m.gotBranch,
		want:       m.channels,
		deadBranch: -1,
		degenerate: m.gotBranch != m.channels,
		coherence:  m.cal.Coherence(),
		mode:       m.mode.String(),
		updates:    updates,
		holds:      holds,
		lockGate:   m.lockGate,
	}
	if g := m.cal.Gains(); len(g) > 1 {
		mag := math.Hypot(float64(real(g[1])), float64(imag(g[1])))
		if mag > 0 {
			h.gainDb = 20 * math.Log10(mag)
		} else {
			h.gainDb = math.Inf(-1)
		}
		h.phaseDeg = math.Atan2(float64(imag(g[1])), float64(real(g[1]))) * 180 / math.Pi
	}
	if m.gotBranch != m.channels || m.refIdx < 0 || m.refIdx >= len(m.powDbFS) {
		return h
	}
	// A branch is dead if it sits mrcBranchDeadMarginDb below the loudest one.
	//
	// This deliberately compares against the LOUDEST branch, not the reference.
	// The reference used to be the argmax by construction, so it could never be
	// the dead one; now that it is latched and held (see selectReference), the
	// reference itself can go dark mid-session — an antenna falling off the
	// primary receiver — and comparing everything to the reference would make
	// exactly that case invisible, which is the one an operator most needs told.
	loudest := argmaxFloat(m.powDbFS)
	top := m.powDbFS[loudest]
	for i, p := range m.powDbFS {
		if i == loudest {
			continue
		}
		if top-p > mrcBranchDeadMarginDb {
			h.deadBranch = i
			break
		}
	}
	return h
}

// setSampleRate re-sizes the estimation window for a known stream rate. The
// window is specified in stream time (mrcCalWindowMs), so it must not inherit
// whatever a datagram happens to hold. Rebuilding drops any estimate, which is
// harmless here: the only caller is StreamIQ, which re-arms calibration anyway.
// Stream-setup path only — not safe to call while combine() is running.
func (m *mrcCombiner) setSampleRate(rateHz float64) {
	want := mrcCalWindowSamples(rateHz)
	if want == m.window {
		return
	}
	m.window = want
	opts := mrcCalOptions(m.mode, want, rateHz)
	m.lockGate = opts.LockGate(want)
	m.cal = diversity.NewTrackingCalibrator(diversityChannels, opts)
	m.refIdx = -1
	m.refDeadDatagrams = 0
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
		m.refIdx = -1
		m.refDeadDatagrams = 0
	}
	branches := m.format.deinterleave(payload, m.channels, elems)
	// Tap the branches BEFORE anything is done to them, so a capture is exactly
	// what the combiner saw and can be replayed through a different one.
	m.rec.write(branches)
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
	m.selectReference()
	if m.refIdx < 0 {
		// No branch is alive yet; nothing to anchor on.
		return branches[0]
	}
	ordered := m.order(branches)
	m.cal.Observe(ordered)
	out, err := m.cal.Combine(ordered)
	if err != nil {
		// Shape guard (should not happen — branches are equal-length): fall
		// back to the reference branch verbatim rather than drop the chunk.
		return branches[m.refIdx]
	}
	return out
}

// selectReference latches the phase anchor on the first window carrying a live
// branch, then holds it. Once the calibrator has locked the anchor is frozen
// outright: moving it would invalidate the gain estimate anchored to it.
func (m *mrcCombiner) selectReference() {
	if m.cal.Calibrated() {
		return
	}
	best := argmaxFloat(m.powDbFS)
	if m.powDbFS[best] <= mrcDeadBranchDbFS {
		m.refIdx = -1 // nothing is receiving
		return
	}
	if m.refIdx < 0 {
		m.refIdx = best
		m.refDeadDatagrams = 0
		return
	}
	// Only a persistently dead incumbent moves the anchor.
	if best == m.refIdx || m.powDbFS[best]-m.powDbFS[m.refIdx] <= mrcBranchDeadMarginDb {
		m.refDeadDatagrams = 0
		return
	}
	m.refDeadDatagrams++
	if m.refDeadDatagrams >= mrcRefDeadDatagrams {
		m.refIdx = best
		m.refDeadDatagrams = 0
		m.cal.Reset() // the anchor changed; the old estimate means nothing
	}
}

// mrcDeadBranchDbFS is mrcBranchFloorPower expressed in dBFS (-100), used
// against the per-branch power readings the health line reports.
const mrcDeadBranchDbFS = -100.0

// round2 trims a float to 2 decimals for a log attribute. -Inf renders as-is.
func round2(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return v
	}
	return math.Round(v*100) / 100
}

// order returns branches permuted so the reference branch is index 0, which is
// the branch the calibrator anchors its phase estimate on. The scratch slice is
// reused across chunks; it only holds slice headers, not sample copies.
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
	// lastUpdates / staleIntervals detect a calibration that locked once and
	// has been held ever since: the combine keeps running on a gain estimated
	// minutes ago, which is a real degradation that the plain INFO line
	// reported as healthy. An operator's hour-long capture showed exactly this
	// — updates frozen at 6682 while holds climbed by ~6000 every interval —
	// and nothing said so.
	lastUpdates    uint64
	staleIntervals int
}

// mrcStaleUpdateIntervals is how many consecutive health intervals may pass
// with no accepted calibration window before it is worth a WARN. Three
// intervals is 90 s: long enough that a quiet moment or a brief fade does not
// trip it, short enough to catch a combiner that has actually given up.
const mrcStaleUpdateIntervals = 3

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
	if h.updates == r.lastUpdates {
		r.staleIntervals++
	} else {
		r.staleIntervals = 0
	}
	r.lastUpdates = h.updates
	attrs := []any{
		"addr", r.addr,
		"branch_dbfs", formatBranchPowers(h.powDbFS),
		"reference_branch", h.refIdx,
		"calibrated", h.calibrated,
		"coherence", round2(h.coherence),
		"branch_gain_db", round2(h.gainDb),
		"branch_phase_deg", round2(h.phaseDeg),
		"mode", h.mode,
		"updates", h.updates,
		"holds", h.holds,
	}
	switch {
	case h.degenerate:
		r.log.Warn("soapyremote: MRC diversity got "+strconv.Itoa(h.branches)+" of "+strconv.Itoa(h.want)+" channels in the stream — the remote is not delivering the second receiver, so there is no diversity gain. Check that the device has a second RX channel and that the server accepted the 2-channel stream request.",
			append(attrs, "channels_delivered", h.branches, "channels_requested", h.want)...)
	case h.deadBranch >= 0:
		r.log.Warn("soapyremote: MRC diversity branch is dead — it sits more than "+strconv.Itoa(int(mrcBranchDeadMarginDb))+" dB below the reference receiver and contributes nothing to the combine. Check that antenna's connection, and that this RX channel is gained (sdr.soapy_remote[].antennas selects its port).",
			append(attrs, "dead_branch", h.deadBranch)...)
	case !h.calibrated && h.updates+h.holds > 0:
		// Both branches are alive but their measured coherence is down at the
		// estimation noise floor, so there is nothing for a single complex gain
		// to align and the combiner is passing the reference receiver through.
		// The gate itself scales with the window (lock_gate names the actual
		// bar), so this no longer fires just because the capture bandwidth is
		// wide — when it fires, the branches genuinely share (almost) no signal.
		//
		// Gated on a window having actually completed: the first datagram of a
		// stream always reports, and "not coherent" before anything has been
		// measured would be a lie that sends the operator after the wrong thing.
		r.log.Warn("soapyremote: MRC diversity branches are not coherent — the two receivers are not seeing the same signal through a constant complex gain, so there is no diversity gain and the reference branch is being passed through. Check both antennas are on the same band and polarisation and are co-located (widely separated antennas give each carrier its own phase, which one wideband complex gain cannot align), and that the front-end is frequency-locked (clock_source/time_source). If one branch_dbfs sits far below the other, that branch may be buried under its own front-end noise floor — fix ITS gain staging (a per-branch gain or attenuator), not the overall level.",
			append(attrs, "lock_gate", round2(h.lockGate))...)
	case h.calibrated && r.staleIntervals >= mrcStaleUpdateIntervals:
		// Locked once, then every window since has failed the coherence gate.
		// The combine is still running — a rejected window HOLDS rather than
		// dropping to passthrough, deliberately, because falling back mid-stream
		// is itself a phase step — but it is applying a gain measured minutes
		// ago to branches that no longer agree.
		r.log.Warn("soapyremote: MRC diversity has not accepted a calibration window in "+
			strconv.Itoa(int(float64(r.staleIntervals)*r.interval.Seconds()))+
			"s — the combine is running on a stale gain because every window since has "+
			"failed the coherence gate. Two receivers whose coherence stays low across a "+
			"wide span is the wideband-combine limitation: one complex gain cannot align "+
			"every carrier at once when the antennas are metres apart. Co-locate them, or "+
			"try diversity: mrc-static to freeze the estimate deliberately.",
			append(attrs, "stale_intervals", r.staleIntervals)...)
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
