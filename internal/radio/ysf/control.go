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
func (s LockState) LockedNAC() uint16         { return 0 }

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
// detector and the on-air FICH decoder (DecodeAirFICH, via AirFramer), and
// emits cc.locked / cc.lost / grant events on the bus.
//
// The lock follows the frame sync alone; grants come only from FICHs that
// decode (Golay + CRC-16) — a header or communications frame opens a call
// on its DG-ID, a terminator ends it (issue #1261: the earlier FICH model
// was fabricated, matched no real frame, and nothing on the live path fed
// it, so a YSF control channel locked but never granted).
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	det        *SyncDetector
	framer     *AirFramer
	systemName string
	freqHz     uint32
	now        func() time.Time

	locked   bool
	lastDGID uint8 // last group ID we published a grant for; suppresses duplicate emission until a Terminator clears it
	hasGrant bool
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
		framer:     NewAirFramer(tol),
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

// Process consumes a window of dibits: sync detection drives the lock,
// and every frame whose FICH decodes is handed to ProcessAirFICH.
// baseIdx is the absolute dibit index of dibits[0]. Returns the new
// baseIndex (matches the NXDN / P25 detector contracts).
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, next := c.det.Process(nil, dibits, baseIdx)
	for range hits {
		c.maybeLock()
	}
	for _, fr := range c.framer.Process(dibits, baseIdx) {
		if fr.FICHOK {
			c.ProcessAirFICH(fr.FICH)
		}
	}
	return next
}

// ProcessAirFICH forwards one decoded on-air FICH to the state machine.
// A header or communications frame of a group call publishes a
// trunking.Grant on its DG-ID (a communications frame is late entry when
// the header was lost; the dedup holds it to one grant per call); a
// terminator clears the dedup so the next transmission grants afresh.
//
// The FICH must have decoded (DecodeAirFICH ok): a junk frame indicator
// would otherwise produce stale CallStart / CallEnd events.
func (c *ControlChannel) ProcessAirFICH(f AirFICH) {
	switch f.FI {
	case AirFIHeader, AirFIComms:
		c.publishGrantFor(f)
	case AirFITerminator:
		c.clearGrant()
	}
}

// airCMIndividual is the FICH call mode of a radio-to-radio call.
const airCMIndividual = 3

func (c *ControlChannel) publishGrantFor(f AirFICH) {
	if f.CM == airCMIndividual {
		// Private (radio-ID-addressed) calls aren't on the
		// trunking-grant path — they're addressed to a specific
		// subscriber, not a talkgroup.
		return
	}
	if f.DT == AirDTDataFR {
		// Data full-rate frames carry no voice for a recorder.
		return
	}
	// The FICH's DG-ID (Digital Group ID) is what operators gate calls
	// on; 0 is the open "all calls" group. Publishing it as the group ID
	// matches what operator UIs key talkgroup-CSV rows on.
	groupID := uint32(f.DGID)
	if c.hasGrant && c.lastDGID == f.DGID {
		// Same talkgroup as the in-flight call; suppress.
		return
	}
	c.hasGrant = true
	c.lastDGID = f.DGID
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "ysf",
			GroupID:     groupID,
			SourceID:    0, // YSF carries a callsign in the DCH, not a numeric ID
			FrequencyHz: c.freqHz,
			At:          c.now(),
		},
	})
	c.log.Debug("ysf: grant",
		"system", c.systemName, "freq_hz", c.freqHz,
		"dgid", groupID, "fi", f.FI, "data_type", f.DT)
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
