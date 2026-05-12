package ltr

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the LTR per-repeater state machine. LTR has no central control
// channel, so "locked" here means "we're receiving valid status
// words from the repeater on this frequency".
type LockState struct {
	FrequencyHz uint32
	Area        uint8
	Repeater    uint8 // home repeater number from the first valid status word
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises LTR lock events alongside the protocol-neutral P25 /
// DMR / NXDN / TETRA payloads. LTR doesn't have a P25-style NAC;
// (Area, Repeater) is the closest per-site identifier — packed
// into the NAC slot as (Area << 8) | Repeater. Without these
// methods, the supervisor's type-assertion on cc.locked silently
// drops the event and /api/v1/scanner never surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return uint16(s.Area)<<8 | uint16(s.Repeater) }

// ControlChannel ingests Status words from a single LTR repeater,
// emits cc.locked the first time a valid status arrives on a
// freshly-tuned device, and republishes any active call as a
// `trunking.Grant` event with `Protocol = "ltr"`.
//
// The state machine is deliberately minimal: each ingested status
// either confirms the lock, signals an active call (group flag set
// + non-zero group ID), or is silently dropped (idle frame, or a
// status word for a different area than we last locked to).
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time
	// expectedArea: when set, ignore status words for any other area
	// (lets a single physical channel host multiple LTR systems).
	expectedArea uint8
	areaSet      bool

	// proc is the cross-call bit buffer the Process adapter uses
	// (see process.go). Lazily constructed on the first Process call.
	proc *processState

	// manchesterMode controls Manchester decoding of the input
	// bit stream — set via SetManchesterMode. Default
	// ManchesterOff treats the stream as raw NRZ.
	manchesterMode ManchesterDecodeMode

	mu     sync.Mutex
	locked bool
	last   LockState
	// activeGroup tracks the group currently announced as active so
	// repeated grant frames during one call don't republish a new
	// grant on every status word.
	activeGroup      uint16
	strictValidation bool
	fcsMode          FCSMode
}

// FCSMode selects whether the Ingest path verifies the LTR
// CRC-7 trailer per DSheirer/sdrtrunk's CRCLTR.java.
//
//   - FCSOff (default): the Ingest path doesn't run any
//     checksum verification. The Status.FCS field is treated as
//     opaque metadata. Existing behaviour pre-PR #40.
//
//   - FCSOn: the Ingest path computes the CRC-7 over a 24-bit
//     message built from the Status fields per sdrtrunk's layout
//     and compares it to the low 7 bits of Status.FCS. Frames
//     whose CRC doesn't match are silently dropped. Useful when
//     the upstream framing layer has populated Status.FCS from
//     the on-air bits and the caller wants to filter out
//     corrupted frames.
//
// Per sdrtrunk's layout the 24 message bits cover the four LTR
// fields the CRC protects: Area (1 bit, mapped from gophertrunk's
// 1-bit Group / F-bit), Channel (5 bits), Home (5 bits), Group
// (8 bits, mapped from gophertrunk's GroupID), and Free (5 bits).
// Note that sdrtrunk's "Area" field is 1 bit, not the 5-bit
// gophertrunk `Status.Area` — the gophertrunk Group F-bit is what
// matches sdrtrunk's Area at this layer. Status.Area continues to
// drive the multi-system filter; the FCS check is independent of
// it.
type FCSMode uint8

const (
	FCSOff FCSMode = iota
	FCSOn
)

// SetFCSMode toggles the CRC-7 FCS verification on the Ingest
// path. See FCSMode for the trade-offs.
func (c *ControlChannel) SetFCSMode(mode FCSMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fcsMode = mode
}

// FCSMode returns the current FCSMode. Mirrors the Set* family so
// callers (and tests) can introspect the configured mode without
// poking at unexported state.
func (c *ControlChannel) FCSMode() FCSMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fcsMode
}

// ManchesterMode returns the configured Manchester decode mode.
func (c *ControlChannel) ManchesterMode() ManchesterDecodeMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.manchesterMode
}

// ParseFCSMode maps a config / user-facing string into an FCSMode.
// Recognised values (case-insensitive): "" / "off" → FCSOff (the
// pre-PR #40 behaviour, no CRC check), "on" / "true" → FCSOn.
// Unknown strings return FCSOff with `ok = false` so callers can
// surface the misconfiguration.
func ParseFCSMode(s string) (FCSMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return FCSOff, true
	case "on", "true", "1":
		return FCSOn, true
	default:
		return FCSOff, false
	}
}

// ParseManchesterMode maps a config / user-facing string into a
// ManchesterDecodeMode. Recognised values (case-insensitive):
// "" / "off" / "nrz" → ManchesterOff (raw NRZ), "strict" →
// ManchesterStrict (drop transition-less pairs), "soft" / "on" →
// ManchesterSoft (majority-decode + tolerate noise bursts).
// Unknown strings return ManchesterOff with `ok = false`.
func ParseManchesterMode(s string) (ManchesterDecodeMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "nrz":
		return ManchesterOff, true
	case "strict":
		return ManchesterStrict, true
	case "soft", "on":
		return ManchesterSoft, true
	default:
		return ManchesterOff, false
	}
}

// SetStrictValidation toggles the strict frame-validity filter on the
// Ingest path. When enabled, Status words whose fixed-range fields
// (Channel 1..20, Home 1..20) fall outside the documented set are
// silently dropped at Ingest time. The Process adapter already
// filters at the framing layer; strict-mode tightens it further so
// frames from a misaligned-but-passing sync window still drop out.
func (c *ControlChannel) SetStrictValidation(strict bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictValidation = strict
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	// Area, when non-zero, restricts the state machine to status
	// words whose Area field matches. Zero accepts every area.
	Area uint8
	Now  func() time.Time
}

// New constructs a ControlChannel.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ControlChannel{
		bus:          opts.Bus,
		log:          log,
		systemName:   opts.SystemName,
		freqHz:       opts.FrequencyHz,
		resolver:     opts.Resolver,
		now:          now,
		expectedArea: opts.Area,
		areaSet:      opts.Area != 0,
	}
}

// Ingest hands one Status word to the state machine. Real captures
// arrive from an upstream sub-audible 300-baud demod; tests publish
// status words directly.
func (c *ControlChannel) Ingest(s Status) {
	c.mu.Lock()
	strict := c.strictValidation
	fcsMode := c.fcsMode
	c.mu.Unlock()
	if strict && !s.IsWellFormed() {
		return
	}
	if fcsMode == FCSOn && !verifyStatusFCS(s) {
		// CRC-7 mismatch — drop. Almost certainly a frame with
		// bit errors or a misaligned codeword.
		return
	}
	if !s.Sync {
		// Malformed frame; drop.
		return
	}
	if c.areaSet && s.Area != c.expectedArea {
		// A different LTR system is sharing the air; ignore.
		return
	}

	c.maybeLock(LockState{
		FrequencyHz: c.freqHz,
		Area:        s.Area,
		Repeater:    s.Home,
	})

	if !s.IsActive() {
		c.mu.Lock()
		c.activeGroup = 0
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	if c.activeGroup == s.GroupID {
		c.mu.Unlock()
		return
	}
	c.activeGroup = s.GroupID
	c.mu.Unlock()
	c.publishGrant(s)
}

func (c *ControlChannel) publishGrant(s Status) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(s.Channel); err == nil {
			freq = hz
		} else {
			c.log.Debug("ltr: band-plan resolution failed",
				"channel", s.Channel, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "ltr",
			GroupID:     uint32(s.GroupID),
			FrequencyHz: freq,
			ChannelNum:  uint16(s.Channel),
			At:          c.now(),
		},
	})
	c.log.Debug("ltr: grant",
		"system", c.systemName, "tg", s.GroupID,
		"channel", s.Channel, "home", s.Home, "freq_hz", freq)
}

func (c *ControlChannel) maybeLock(st LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == st {
		return
	}
	c.locked = true
	c.last = st
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: st})
	c.log.Info("ltr cc locked",
		"freq", st.FrequencyHz, "area", st.Area, "repeater", st.Repeater,
		"system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when status words stop arriving.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.activeGroup = 0
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
