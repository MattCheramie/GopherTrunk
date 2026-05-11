package ysf

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the YSF state machine. Frequency is always present; future revisions
// will accrete site / call-sign metadata as the FICH path stabilises.
//
// LockState satisfies trunking.LockedPayload so the hunter can consume
// it without importing this package.
type LockState struct {
	FrequencyHz uint32
}

// LockedFrequencyHz / LockedNAC implement trunking.LockedPayload.
// YSF doesn't have a P25-style NAC; LockedNAC returns 0 and the
// hunter treats it as a don't-care.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16          { return 0 }

// Options configure a ControlChannel. Backwards-compatible with the
// old NewControlChannel positional constructor — callers that don't
// care about grant emission can keep using NewControlChannel and the
// state machine will skip the grant path silently.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Now         func() time.Time
	// SyncTolerance forwards to NewSyncDetector. Zero / negative
	// uses the package default (2).
	SyncTolerance int
}

// ControlChannel ingests a stream of YSF dibits, runs the FSW
// detector + (planned) FICH Trellis decoder, and emits cc.locked /
// cc.lost / grant / call.start-via-engine events on the bus.
//
// The FICH Trellis decode + interleave wiring lives behind
// ProcessFICH today: the caller is expected to have decoded the
// 100-dibit FICH region into a parsed FICH struct (via the bit-level
// ParseFICH or the channel-bit DecodeFICHTrellis primitive) and to
// hand it in. A future PR closes the IQ → dibit → FICH gap on the
// hot path; the grant-publication contract here doesn't change.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	det        *SyncDetector
	systemName string
	freqHz     uint32
	now        func() time.Time

	locked     bool
	lastDGID   uint8 // last group ID we published a grant for; suppresses duplicate emission until a Terminator clears it
	hasGrant   bool
}

// New constructs a ControlChannel from Options.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	tol := opts.SyncTolerance
	if tol <= 0 {
		tol = 2
	}
	return &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		det:        NewSyncDetector(tol),
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		now:        now,
	}
}

// NewControlChannel is the legacy positional constructor. Prefer
// New(Options{...}) — it carries the system name through to grant
// payloads, which the engine needs to dispatch them.
func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	return New(Options{Bus: bus, Log: log, FrequencyHz: freqHz})
}

// Process consumes a window of dibits and runs sync detection.
// baseIdx is the absolute dibit index of dibits[0]. Returns the new
// baseIndex (matches the NXDN / P25 detector contracts).
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, next := c.det.Process(nil, dibits, baseIdx)
	for range hits {
		c.maybeLock()
	}
	return next
}

// ProcessFICH forwards a parsed FICH (post-Trellis-decode and
// post-CRC-validation) to the state machine. On Header FICH for a
// Group call the channel publishes a trunking.Grant on the bus; on
// Terminator FICH the dedup state clears so the next transmission
// can fire a fresh CallStart.
//
// FICH must be valid (CRC verified) before reaching here — callers
// that accept partially-corrupted FICHs should drop them rather
// than route them through this path, since a junk FrameType would
// otherwise produce stale CallStart / CallEnd events.
func (c *ControlChannel) ProcessFICH(f FICH) {
	switch f.FrameType {
	case FrameTypeHeader:
		c.publishGrantFor(f)
	case FrameTypeTerminator:
		c.clearGrant()
	}
}

func (c *ControlChannel) publishGrantFor(f FICH) {
	if f.CallType != CallTypeGroup {
		// Private (radio-ID-addressed) calls aren't on the
		// trunking-grant path — they're addressed to a specific
		// subscriber, not a talkgroup. Future work could publish
		// them on a separate event kind, but for now we skip them
		// so the engine doesn't spawn a recorder for a private
		// transmission the operator probably isn't subscribed to.
		return
	}
	// On YSF the FICH carries a Squelch Code (DG-ID, Digital Group
	// ID) that operators use to gate calls to a specific group. In
	// open-squelch mode (SquelchMode == false) every transmission is
	// audible regardless of code, so we use 0 as the talkgroup ID
	// in that case so the engine treats the repeater as a single
	// "all calls" group. With code-squelch active we publish the
	// SquelchCode itself as the group ID, matching what most
	// operator UIs key talkgroup-CSV rows on.
	groupID := uint32(0)
	if f.SquelchMode {
		groupID = uint32(f.SquelchCode)
	}
	if c.hasGrant && c.lastDGID == uint8(groupID) {
		// Same talkgroup as the in-flight call; suppress.
		return
	}
	c.hasGrant = true
	c.lastDGID = uint8(groupID)
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "ysf",
			GroupID:     groupID,
			SourceID:    0, // YSF radio ID lives in the DCH region, not the FICH
			FrequencyHz: c.freqHz,
			At:          c.now(),
		},
	})
	c.log.Debug("ysf: grant",
		"system", c.systemName, "freq_hz", c.freqHz,
		"dgid", groupID, "voip", f.VoIP, "data_type", f.DataType.String())
}

func (c *ControlChannel) clearGrant() {
	c.hasGrant = false
	c.lastDGID = 0
}

func (c *ControlChannel) maybeLock() {
	if c.locked {
		return
	}
	c.locked = true
	state := LockState{FrequencyHz: c.freqHz}
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: state})
	c.log.Info("ysf cc locked", "freq", state.FrequencyHz)
}

// MarkLost publishes cc.lost and resets the locked flag. Wired up by
// the engine's watchdog when the FSW stops correlating.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.clearGrant()
	c.bus.Publish(events.Event{
		Kind:    events.KindCCLost,
		Payload: LockState{FrequencyHz: c.freqHz},
	})
}
