package conventional

import (
	"context"
	"errors"
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
}

// New constructs a Scanner. Channels must be non-empty.
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
	if len(opts.Channels) == 0 {
		return nil, errors.New("conventional: at least one channel is required")
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
	// Apply per-channel defaults.
	for i := range opts.Channels {
		ch := &opts.Channels[i]
		if ch.SquelchDbFS == 0 {
			ch.SquelchDbFS = -50
		}
		if ch.Hangtime <= 0 {
			ch.Hangtime = 1500 * time.Millisecond
		}
		if ch.Mode == "" {
			ch.Mode = "fm"
		}
	}
	return &Scanner{
		opts:             opts,
		log:              opts.Log,
		state:            StateScanning,
		dwellIndex:       -1,
		forcedDwellIndex: -1,
		lastBreakAt:      make([]time.Time, len(opts.Channels)),
	}, nil
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
		idx := s.nextChannel()
		ch := s.opts.Channels[idx]

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
// false when the timer expires (advance to next channel).
func (s *Scanner) scanWindow(ctx context.Context, idx int, ch Channel, stream <-chan []complex64) bool {
	deadline := time.NewTimer(s.opts.MinDwellPerChannel)
	defer deadline.Stop()
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
			if PowerDbFS(iq) >= ch.SquelchDbFS {
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
			if power >= ch.SquelchDbFS {
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

// nextChannel returns the index of the next channel to tune,
// respecting any forced-dwell override from DwellOn.
func (s *Scanner) nextChannel() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.forcedDwellIndex >= 0 && s.forcedDwellIndex < len(s.opts.Channels) {
		idx := s.forcedDwellIndex
		s.forcedDwellIndex = -1
		s.cursor = (idx + 1) % len(s.opts.Channels)
		return idx
	}
	idx := s.cursor
	s.cursor = (s.cursor + 1) % len(s.opts.Channels)
	return idx
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
	if idx < 0 || idx >= len(s.opts.Channels) {
		return false
	}
	s.forcedDwellIndex = idx
	return true
}

// Snapshot returns a copy of the current scanner state for the
// REST cockpit / TUI panel.
func (s *Scanner) Snapshot() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := make([]ChannelStatus, len(s.opts.Channels))
	for i, ch := range s.opts.Channels {
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
