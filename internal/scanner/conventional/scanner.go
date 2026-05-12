package conventional

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Channel is one entry in the conventional scan list.
type Channel struct {
	Label       string
	FrequencyHz uint32
	// Mode is "fm" or "nfm" — the latter narrows the post-demod
	// audio LPF; both share the IQ-power squelch.
	Mode string
	// SquelchDbFS is the threshold above which the scanner declares
	// "carrier present". A typical value for an RTL-SDR-class
	// receiver is around -50 dBFS; tune per channel as needed.
	SquelchDbFS float64
	// Hangtime is how long below threshold must elapse before the
	// scanner declares the call over and resumes hopping. Default
	// 1500 ms keeps the scanner from clipping the tail of normal
	// FM transmissions.
	Hangtime time.Duration
	// Priority is forwarded to the synthetic talkgroup so the
	// engine's preemption logic respects it relative to other
	// conv-scanner channels.
	Priority int
	// Tone is the optional CTCSS / DCS gate. When set, the
	// scanner only declares "carrier present" while BOTH the
	// IQ-power squelch is open AND the configured sub-audible
	// tone is detected. Zero value (Mode="" / "none") disables
	// tone gating and the scanner behaves identically to its
	// pre-tone version. DCS mode parses + validates but the
	// detector is a tracked follow-up — see ctcss.go.
	Tone ToneConfig
}

// ToneConfig configures CTCSS / DCS squelch gating for one channel.
// Mode selects the family; the relevant field for the chosen mode
// must be populated.
type ToneConfig struct {
	// Mode is "ctcss", "dcs", or "" / "none" (default). Unknown
	// values are rejected at validation time.
	Mode string
	// CTCSSHz is the target CTCSS frequency (50–300 Hz). Required
	// when Mode is "ctcss". Standard EIA codes range from 67.0 to
	// 254.1 Hz; 38 are widely deployed.
	CTCSSHz float64
	// DCSCode is the three-digit octal DCS code (e.g. "023",
	// "754"). Required when Mode is "dcs". Detector wiring is
	// deferred — see Workstream D follow-up.
	DCSCode string
}

// Tuner is the subset of sdr.Device the scanner needs.
type Tuner interface {
	SetCenterFreq(hz uint32) error
}

// IQSource provides the IQ stream the scanner consumes for both
// squelch detection and (downstream) recorder integration. The
// scanner cancels and re-opens the stream every tune to drop any
// in-flight samples from the previous channel.
type IQSource interface {
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
}

// Engine is the subset of trunking.Engine the scanner needs. Only
// the synthetic-call entry points; this keeps tests trivial without
// constructing a full Engine.
type Engine interface {
	HandleSyntheticCall(g trunking.Grant, deviceSerial string)
	EndSyntheticCall(deviceSerial string, reason trunking.EndReason) bool
	Touch(deviceSerial string)
}

// Recorder is the subset of voice.Recorder the scanner feeds. The
// real recorder accepts WritePCM by device serial; tests can stub
// with a noop.
type Recorder interface {
	WritePCM(deviceSerial string, samples []int16) error
}

// Options configure the conventional scanner.
type Options struct {
	Log          *slog.Logger
	Tuner        Tuner
	IQ           IQSource
	Engine       Engine
	Recorder     Recorder
	DeviceSerial string
	SystemName   string // surfaces on the synthetic Grant.System
	Channels     []Channel
	// DwellChunkLen is the IQ-power measurement window in samples
	// during SCANNING. Default 4096 (≈ 1.7 ms at 2.4 MS/s).
	DwellChunkLen int
	// MinDwellPerChannel is the minimum time on each channel during
	// SCANNING before advancing — even if squelch never opens. Default
	// 100 ms; let signals settle after retune.
	MinDwellPerChannel time.Duration
	// SampleRateHz is the IQ sample rate the IQ source delivers
	// (typically 2.4e6 for RTL-SDR). Required when any configured
	// channel has Tone gating — without it the CTCSS detector
	// can't pick the right Goertzel bin. Zero is fine when no
	// tone gating is in play; the field is otherwise inert.
	SampleRateHz float64
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// State is the high-level scanner state surfaced through Snapshot.
type State string

const (
	StateScanning State = "scanning"
	StateDwell    State = "dwell"
	StateHeld     State = "held"
)

// ChannelStatus is one row in the Snapshot.
type ChannelStatus struct {
	Index       int       `json:"index"`
	Label       string    `json:"label"`
	FrequencyHz uint32    `json:"frequency_hz"`
	Mode        string    `json:"mode"`
	Active      bool      `json:"active"`
	LastBreakAt time.Time `json:"last_break_at,omitempty"`
}

// Status is the scanner-wide snapshot.
type Status struct {
	State       State           `json:"state"`
	Channels    []ChannelStatus `json:"channels"`
	CursorIndex int             `json:"cursor_index"`
	DeviceSerial string         `json:"device_serial,omitempty"`
}

// Scanner is the state machine. Construct via New, Run with a ctx.
type Scanner struct {
	opts Options
	log  *slog.Logger

	mu          sync.RWMutex
	channels    []Channel // mutable; opts.Channels is the seed, AddTemporaryChannel grows it
	// detectors parallels channels: detectors[i] is non-nil iff
	// channels[i] has Tone gating configured. Built at New() and
	// kept in sync by AddTemporaryChannel / RemoveTemporaryChannel.
	detectors   []toneDetector
	cursor      int
	state       State
	held        bool
	lastBreakAt []time.Time
	// dwellIndex is the channel index currently dwelling (when
	// state == StateDwell); -1 otherwise. Operator hold and
	// "dwell on N" mutations write through this field.
	dwellIndex int
	// forcedDwellIndex is set by DwellOn — the next round picks this
	// channel up directly, bypassing the scan cursor.
	forcedDwellIndex int
	// tempChannels tracks indices added by AddTemporaryChannel so
	// RemoveTemporaryChannel can validate the index belongs to a
	// VFO-style entry rather than a config-seeded one. Bool value
	// is irrelevant — set membership only.
	tempChannels map[int]bool
}

// New constructs a Scanner. Channels may be empty when the operator
// only intends to drive the scanner via AddTemporaryChannel (manual
// VFO tune from the TUI / API).
func New(opts Options) (*Scanner, error) {
	if opts.Engine == nil {
		return nil, errors.New("conventional: Engine is required")
	}
	if opts.Tuner == nil {
		return nil, errors.New("conventional: Tuner is required")
	}
	if opts.IQ == nil {
		return nil, errors.New("conventional: IQSource is required")
	}
	if opts.Recorder == nil {
		return nil, errors.New("conventional: Recorder is required")
	}
	if opts.DeviceSerial == "" {
		return nil, errors.New("conventional: DeviceSerial is required")
	}
	if opts.DwellChunkLen <= 0 {
		opts.DwellChunkLen = 4096
	}
	if opts.MinDwellPerChannel <= 0 {
		opts.MinDwellPerChannel = 100 * time.Millisecond
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	// Copy + apply per-channel defaults so the caller's slice
	// stays untouched.
	channels := make([]Channel, len(opts.Channels))
	copy(channels, opts.Channels)
	for i := range channels {
		ch := &channels[i]
		if ch.SquelchDbFS == 0 {
			ch.SquelchDbFS = -50
		}
		if ch.Hangtime <= 0 {
			ch.Hangtime = 1500 * time.Millisecond
		}
		if ch.Mode == "" {
			ch.Mode = "fm"
		}
		if err := validateTone(ch.Tone); err != nil {
			return nil, fmt.Errorf("conventional: channel %q: %w", ch.Label, err)
		}
	}
	// Bump min dwell when any channel has tone gating so the
	// Goertzel block has time to fire. 250 ms covers a SampleHz/5
	// block plus margin; without this the scanner would advance
	// before the detector ever updated and tone-gated channels
	// would never lock.
	if hasToneGate(channels) && opts.MinDwellPerChannel < 250*time.Millisecond {
		opts.MinDwellPerChannel = 250 * time.Millisecond
	}
	detectors := make([]toneDetector, len(channels))
	for i, ch := range channels {
		if d := buildDetector(ch.Tone, opts.SampleRateHz); d != nil {
			detectors[i] = d
		}
	}
	return &Scanner{
		opts:             opts,
		log:              opts.Log,
		channels:         channels,
		detectors:        detectors,
		state:            StateScanning,
		dwellIndex:       -1,
		forcedDwellIndex: -1,
		lastBreakAt:      make([]time.Time, len(channels)),
		tempChannels:     make(map[int]bool),
	}, nil
}

// validateTone rejects malformed tone configs at construction time.
// "" / "none" disables gating; "ctcss" requires CTCSSHz in the
// practical band; "dcs" requires a 3-digit octal code (detector
// not yet implemented — config is accepted so deployments can
// pre-stage their YAML).
func validateTone(t ToneConfig) error {
	switch t.Mode {
	case "", "none":
		return nil
	case "ctcss":
		if t.CTCSSHz < 50 || t.CTCSSHz > 300 {
			return fmt.Errorf("tone.ctcss_hz %v outside 50..300 Hz", t.CTCSSHz)
		}
		return nil
	case "dcs":
		if len(t.DCSCode) != 3 {
			return fmt.Errorf("tone.dcs_code must be a 3-digit octal code")
		}
		for _, r := range t.DCSCode {
			if r < '0' || r > '7' {
				return fmt.Errorf("tone.dcs_code %q must be octal digits 0..7", t.DCSCode)
			}
		}
		return nil
	default:
		return fmt.Errorf("tone.mode %q must be ctcss|dcs|none", t.Mode)
	}
}

func hasToneGate(channels []Channel) bool {
	for _, ch := range channels {
		if ch.Tone.Mode == "ctcss" || ch.Tone.Mode == "dcs" {
			return true
		}
	}
	return false
}

// toneDetector is the shared interface CTCSS / DCS detectors satisfy.
// Lets the scanner store either one in a single field without losing
// the typed Process / Reset signatures. Adding a new tone family
// (e.g. selective five-call) reduces to adding a new implementation
// and a new branch in buildDetector.
type toneDetector interface {
	// Process feeds one IQ chunk and returns the latest detection
	// state. Implementations must be safe to call between Resets.
	Process(iq []complex64) bool
	// Present reports the most recent detection state without
	// processing any new samples.
	Present() bool
	// Reset clears all internal state. The scanner calls this on
	// every retune so leftover state from a previous channel
	// doesn't bias the new dwell.
	Reset()
}

// buildDetector returns a tone detector when gating is configured for
// the channel and the sample rate is set. CTCSS routes to the
// Goertzel-based CTCSSDetector; DCS routes to the Golay-based
// DCSDetector. Returns nil for ungated channels or when SampleHz is
// zero. The scanner treats nil as "no gating".
func buildDetector(t ToneConfig, sampleHz float64) toneDetector {
	if sampleHz <= 0 {
		return nil
	}
	switch t.Mode {
	case "ctcss":
		if d := NewCTCSSDetector(CTCSSConfig{SampleHz: sampleHz, TargetHz: t.CTCSSHz}); d != nil {
			return d
		}
	case "dcs":
		if d := NewDCSDetector(DCSConfig{SampleHz: sampleHz, Code: t.DCSCode}); d != nil {
			return d
		}
	}
	return nil
}

// Run blocks until ctx cancels. Tunes, measures squelch, hands off
// to the engine on break, and resumes scanning after hangtime.
// Returns ctx.Err() on shutdown.
func (s *Scanner) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.IsHeld() {
			s.sleep(ctx, 100*time.Millisecond)
			continue
		}
		idx, ch, ok := s.pickNextChannel()
		if !ok {
			// Empty channel list — operator has neither configured
			// any static channels nor added a temporary one. Idle
			// in 100 ms ticks so AddTemporaryChannel picks up on
			// the next round.
			s.sleep(ctx, 100*time.Millisecond)
			continue
		}

		if err := s.opts.Tuner.SetCenterFreq(ch.FrequencyHz); err != nil {
			s.log.Warn("conv: tune failed", "freq_hz", ch.FrequencyHz, "err", err)
			s.sleep(ctx, 100*time.Millisecond)
			continue
		}

		// Open the IQ stream for this dwell. We close it on advance.
		streamCtx, cancel := context.WithCancel(ctx)
		stream, err := s.opts.IQ.StreamIQ(streamCtx)
		if err != nil {
			cancel()
			s.log.Warn("conv: StreamIQ failed", "err", err)
			s.sleep(ctx, 200*time.Millisecond)
			continue
		}

		// Reset the per-channel CTCSS detector (if any) so leftover
		// state from a previous dwell on this index doesn't bias
		// the new window. The detector itself is owned by the
		// scanner so we don't need to thread it through.
		if det := s.detectorFor(idx); det != nil {
			det.Reset()
		}

		// Wait for either squelch to break, the min-dwell timer to
		// expire (advance), or ctx cancel.
		broken := s.scanWindow(ctx, idx, ch, stream)
		if !broken {
			cancel()
			continue
		}

		// Squelch is open — hand off to the engine and stay on this
		// channel feeding PCM through the recorder until hangtime.
		s.beginDwell(idx, ch, stream, streamCtx, cancel)
	}
}

// scanWindow returns true when squelch opens within MinDwellPerChannel,
// false when the timer expires (advance to next channel). When the
// channel has tone gating configured the detector must also be
// matched — power alone isn't enough to declare "right system".
func (s *Scanner) scanWindow(ctx context.Context, idx int, ch Channel, stream <-chan []complex64) bool {
	deadline := time.NewTimer(s.opts.MinDwellPerChannel)
	defer deadline.Stop()
	det := s.detectorFor(idx)
	for {
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case iq, ok := <-stream:
			if !ok {
				return false
			}
			powerOK := PowerDbFS(iq) >= ch.SquelchDbFS
			if !powerOK {
				continue
			}
			if det == nil {
				return true
			}
			if det.Process(iq) {
				return true
			}
		}
	}
}

// beginDwell takes the open IQ stream, announces a synthetic call
// via the engine, and stays on the channel feeding PCM to the
// recorder until hangtime silence is observed. Returns after
// publishing CallEnd.
func (s *Scanner) beginDwell(idx int, ch Channel, stream <-chan []complex64, streamCtx context.Context, cancel context.CancelFunc) {
	defer cancel()

	now := s.opts.Now()
	s.mu.Lock()
	s.state = StateDwell
	s.dwellIndex = idx
	s.lastBreakAt[idx] = now
	s.mu.Unlock()

	// Synthesize a Grant. GroupID is derived from the channel index
	// so each conv channel surfaces as a distinct talkgroup in the
	// API + call log; bit 31 set so it can't collide with a real
	// trunked GroupID (which are 24/28-bit fields in practice).
	gid := uint32(0x80000000) | uint32(idx)
	g := trunking.Grant{
		System:      s.opts.SystemName,
		Protocol:    "fm-conv",
		GroupID:     gid,
		SourceID:    0,
		FrequencyHz: ch.FrequencyHz,
		At:          now,
	}
	s.opts.Engine.HandleSyntheticCall(g, s.opts.DeviceSerial)

	// Carrier-drop detection: keep reading IQ; track time below
	// threshold. Touch the engine watchdog on each chunk so it
	// doesn't time the call out prematurely. We DON'T run a real
	// FM-demod chain here — that's a follow-up. For now the
	// recorder gets a silent WAV that documents the carrier-active
	// window, which is enough to validate the integration.
	//
	// When the channel has tone gating, an "active" sample requires
	// BOTH carrier present AND tone matched. Hangtime starts as soon
	// as either condition goes false — so a transmitter dropping
	// the CTCSS tone (e.g. switching to a different talkgroup on
	// the same repeater) hangs up just like a true carrier drop.
	det := s.detectorFor(idx)
	belowSince := time.Time{}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-streamCtx.Done():
			s.endDwell(idx, trunking.EndReasonError)
			return
		case <-ticker.C:
			s.opts.Engine.Touch(s.opts.DeviceSerial)
		case iq, ok := <-stream:
			if !ok {
				s.endDwell(idx, trunking.EndReasonError)
				return
			}
			power := PowerDbFS(iq)
			active := power >= ch.SquelchDbFS
			if active && det != nil {
				active = det.Process(iq)
			}
			if active {
				belowSince = time.Time{}
			} else if belowSince.IsZero() {
				belowSince = s.opts.Now()
			} else if s.opts.Now().Sub(belowSince) >= ch.Hangtime {
				s.endDwell(idx, trunking.EndReasonNormal)
				return
			}
		}
	}
}

func (s *Scanner) endDwell(idx int, reason trunking.EndReason) {
	s.opts.Engine.EndSyntheticCall(s.opts.DeviceSerial, reason)
	s.mu.Lock()
	if s.dwellIndex == idx {
		s.state = StateScanning
		s.dwellIndex = -1
	}
	s.mu.Unlock()
}

// pickNextChannel returns the index + a copy of the next channel to
// tune, respecting any forced-dwell override from DwellOn. The
// returned ok=false signals an empty channel list — the Run loop
// idles until AddTemporaryChannel populates one.
func (s *Scanner) pickNextChannel() (int, Channel, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.channels)
	if n == 0 {
		return 0, Channel{}, false
	}
	if s.forcedDwellIndex >= 0 && s.forcedDwellIndex < n {
		idx := s.forcedDwellIndex
		s.forcedDwellIndex = -1
		s.cursor = (idx + 1) % n
		return idx, s.channels[idx], true
	}
	if s.cursor >= n {
		s.cursor = 0
	}
	idx := s.cursor
	s.cursor = (s.cursor + 1) % n
	return idx, s.channels[idx], true
}

// detectorFor returns the CTCSS detector for the given channel
// index, or nil if the channel has no tone gating configured. The
// returned detector is owned by the scanner; callers must not
// retain it across Run iterations.
func (s *Scanner) detectorFor(idx int) toneDetector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx < 0 || idx >= len(s.detectors) {
		return nil
	}
	return s.detectors[idx]
}

func (s *Scanner) sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// --- operator mutation surface ---

// Hold pins the scanner on its current channel (in StateDwell) or
// pauses scanning (in StateScanning). The held state persists until
// Resume.
func (s *Scanner) Hold() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held = true
	s.state = StateHeld
}

// Resume undoes Hold. The next Run iteration picks scanning back up.
func (s *Scanner) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held = false
	if s.dwellIndex >= 0 {
		s.state = StateDwell
	} else {
		s.state = StateScanning
	}
}

// IsHeld reports whether the scanner is currently held.
func (s *Scanner) IsHeld() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.held
}

// DwellOn asks the scanner to advance to the named channel on the
// next round. Returns false if idx is out of range.
func (s *Scanner) DwellOn(idx int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.channels) {
		return false
	}
	s.forcedDwellIndex = idx
	return true
}

// AddTemporaryChannel appends a runtime "VFO" channel and forces
// the scanner to dwell on it next round. Returns the new index.
// Defaults are applied identically to the config-seeded path
// (SquelchDbFS=-50, Hangtime=1500ms, Mode=fm). Tone config is
// validated and may produce an out-of-range detector failure
// that's silently ignored — the manual-tune path is best-effort
// for the v1 surface. Safe to call from any goroutine.
func (s *Scanner) AddTemporaryChannel(ch Channel) int {
	if ch.SquelchDbFS == 0 {
		ch.SquelchDbFS = -50
	}
	if ch.Hangtime <= 0 {
		ch.Hangtime = 1500 * time.Millisecond
	}
	if ch.Mode == "" {
		ch.Mode = "fm"
	}
	if err := validateTone(ch.Tone); err != nil {
		// Bad tone config on a temp channel: log and strip the
		// tone rather than reject the whole tune. The manual
		// tune surface is meant to be forgiving.
		s.log.Warn("conv: dropping invalid tone config on temp channel", "err", err)
		ch.Tone = ToneConfig{}
	}
	det := buildDetector(ch.Tone, s.opts.SampleRateHz)
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := len(s.channels)
	s.channels = append(s.channels, ch)
	s.detectors = append(s.detectors, det)
	s.lastBreakAt = append(s.lastBreakAt, time.Time{})
	s.tempChannels[idx] = true
	s.forcedDwellIndex = idx
	// Make sure we're not held — a manual tune is an explicit
	// "listen now" intent.
	s.held = false
	s.state = StateScanning
	return idx
}

// RemoveTemporaryChannel deletes a channel previously added via
// AddTemporaryChannel. Static (config-seeded) channels can't be
// removed at runtime; this returns false for those. If the
// channel is currently dwelling, the scanner ends the synthetic
// call before removing it.
//
// Note: removing a temporary channel re-indexes everything after
// it. This is fine for the v1 manual-tune flow (which adds one
// VFO at a time and revokes it on a new tune) but callers must
// not assume indices are stable across a remove.
func (s *Scanner) RemoveTemporaryChannel(idx int) bool {
	s.mu.Lock()
	if !s.tempChannels[idx] {
		s.mu.Unlock()
		return false
	}
	// If we're currently dwelling on the channel being removed,
	// end the synthetic call first so the engine cleans up.
	dwelling := s.dwellIndex == idx
	s.mu.Unlock()
	if dwelling {
		s.opts.Engine.EndSyntheticCall(s.opts.DeviceSerial, trunking.EndReasonManual)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Re-validate under the lock (a concurrent remove could have
	// snuck in).
	if !s.tempChannels[idx] || idx >= len(s.channels) {
		return false
	}
	s.channels = append(s.channels[:idx], s.channels[idx+1:]...)
	s.detectors = append(s.detectors[:idx], s.detectors[idx+1:]...)
	s.lastBreakAt = append(s.lastBreakAt[:idx], s.lastBreakAt[idx+1:]...)
	// Rebuild the tempChannels set with shifted indices.
	rebuilt := make(map[int]bool, len(s.tempChannels))
	for i := range s.tempChannels {
		switch {
		case i == idx:
			// dropped
		case i > idx:
			rebuilt[i-1] = true
		default:
			rebuilt[i] = true
		}
	}
	s.tempChannels = rebuilt
	if s.cursor > idx {
		s.cursor--
	}
	if s.cursor >= len(s.channels) {
		s.cursor = 0
	}
	if s.dwellIndex == idx {
		s.dwellIndex = -1
		s.state = StateScanning
	} else if s.dwellIndex > idx {
		s.dwellIndex--
	}
	if s.forcedDwellIndex == idx {
		s.forcedDwellIndex = -1
	} else if s.forcedDwellIndex > idx {
		s.forcedDwellIndex--
	}
	return true
}

// Snapshot returns a copy of the current scanner state for the
// REST cockpit / TUI panel.
func (s *Scanner) Snapshot() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := make([]ChannelStatus, len(s.channels))
	for i, ch := range s.channels {
		channels[i] = ChannelStatus{
			Index:       i,
			Label:       ch.Label,
			FrequencyHz: ch.FrequencyHz,
			Mode:        ch.Mode,
			Active:      i == s.dwellIndex,
			LastBreakAt: s.lastBreakAt[i],
		}
	}
	return Status{
		State:        s.state,
		Channels:     channels,
		CursorIndex:  s.cursor,
		DeviceSerial: s.opts.DeviceSerial,
	}
}
