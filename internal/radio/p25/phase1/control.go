package phase1

import (
	"errors"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ControlChannel consumes a stream of P25 Phase 1 dibits (already
// symbol-time-recovered and mapped via SymbolToDibit) and emits
// trunking events onto an events.Bus.
//
// Pipeline: dibit window → FSW detect → NID parse (BCH(63,16,11) +
// even-parity check) → if DUID is TSDU and the buffer holds enough
// dibits, deinterleave + Viterbi-decode the next 98-dibit TSBK block,
// validate the CRC trailer, and dispatch on the parsed opcode.
//
//   - OpIdentifierUpdate (0x3D) populates the band-plan slot for its
//     Channel ID.
//   - OpGroupVoiceChannelGrant (0x00) parses the channel/group/source
//     payload, looks up the frequency in the band plan, and publishes
//     a trunking.Grant with Protocol="p25" on the bus.
//
// CCLocked / CCLost events fan out on the first corrected NID with a
// TSDU DUID. Uncorrectable NIDs and TSBK CRC failures publish
// KindDecodeError; a grant whose Channel ID has no IdentifierUpdate
// yet publishes KindDecodeError with stage="no-bandplan" so the
// metric counter surfaces the gap.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	det        *SyncDetector
	systemName string
	freqHz     uint32
	bandPlan   *BandPlan
	now        func() time.Time
	locked     bool
	lastNAC    uint16
	// lastNoHitsAt throttles the "no FSW hits" debug log so the chunk-rate
	// emission doesn't flood at debug level. See Process for the rationale.
	lastNoHitsAt time.Time

	// buf accumulates dibits across Process calls so a frame whose
	// FSW + NID + TSBK straddles IQ-chunk boundaries is still
	// assembled; bufBase is the absolute dibit index of buf[0].
	// pending holds FSW hits whose NID + TSBK has not been fully
	// buffered yet. See Process — this is the fix for issue #275,
	// where a live SDR's small IQ chunks delivered far fewer than a
	// frame's worth of dibits per call.
	buf     []uint8
	bufBase int
	pending []pendingHit
}

// pendingHit is an FSW match awaiting enough buffered dibits to decode
// its NID + TSBK. end is the absolute dibit index of the FSW's last
// dibit; rot is the cyclic rotation the sync detector matched under.
type pendingHit struct {
	end int
	rot uint8
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	BandPlan    *BandPlan // optional; a new empty BandPlan is used if nil
	Now         func() time.Time
}

// New constructs a ControlChannel from Options. SystemName ends up on
// every trunking.Grant the channel publishes; the daemon passes it
// through from config.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	bp := opts.BandPlan
	if bp == nil {
		bp = &BandPlan{}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		det:        NewSyncDetector(4),
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		bandPlan:   bp,
		now:        now,
	}
}

// NewControlChannel keeps the legacy positional constructor working —
// it's used by the existing FEC/decode tests that don't care about
// grant publication. New callers should use New(Options{...}).
func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	return New(Options{Bus: bus, Log: log, FrequencyHz: freqHz})
}

// LockState is the payload of CCLocked / CCLost events. It satisfies
// trunking.LockedPayload so the hunter can consume it without
// importing this package (which would create an import cycle now that
// phase1 publishes trunking.Grant events).
type LockState struct {
	FrequencyHz uint32
	NAC         uint16
	DUID        DUID
}

// LockedFrequencyHz / LockedNAC implement trunking.LockedPayload.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.NAC }

// noHitsThrottle bounds how often Process emits its "no FSW hits" debug
// log when the sync detector is finding nothing in successive chunks.
// Issue #275 surfaced because that state produced zero logs at all —
// operators couldn't tell whether the IQ pipeline was alive but
// unsynchronized or wholly silent. Throttling at 2 s keeps the signal
// visible without flooding.
const noHitsThrottle = 2 * time.Second

// frameLookahead is the number of dibits that must follow the FSW for
// a full frame to decode: the 32-dibit NID plus the 98-dibit TSBK
// channel block. Process defers an FSW hit until this many dibits have
// accumulated, so frame assembly no longer depends on the IQ chunking.
const frameLookahead = 32 + 98

// Process consumes a window of dibits and runs detection/parsing.
// baseIdx is the absolute dibit index of dibits[0]. Returns the
// absolute index one past the last consumed dibit.
//
// Dibits are accumulated into an internal buffer that spans calls, so
// a frame whose FSW + NID + TSBK straddles several Process calls is
// still assembled. This matters on live hardware: a 16 KiB RTL-SDR USB
// transfer carries only ~19 P25 symbols — far short of the 154-dibit
// frame — so without cross-call buffering every FSW hit was discarded
// and the control channel never locked (issue #275).
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, rots, next := c.det.ProcessWithRotation(nil, nil, dibits, baseIdx)
	if len(hits) == 0 && len(dibits) > 0 && !c.locked {
		now := c.now()
		if now.Sub(c.lastNoHitsAt) >= noHitsThrottle {
			c.log.Debug("p25/phase1: no FSW hits in chunk",
				"system", c.systemName, "freq_hz", c.freqHz, "dibits", len(dibits))
			c.lastNoHitsAt = now
		}
	}

	// Accumulate the new dibits. The receiver hands them over in
	// contiguous, in-order batches, so buf stays a faithful copy of
	// the dibit stream from bufBase onward.
	if len(c.buf) == 0 {
		c.bufBase = baseIdx
	}
	c.buf = append(c.buf, dibits...)
	for i, h := range hits {
		c.pending = append(c.pending, pendingHit{end: h, rot: rots[i]})
	}

	// Parse every pending FSW hit whose full frame has now been
	// buffered; keep the rest for a later call once more dibits land.
	kept := c.pending[:0]
	for _, ph := range c.pending {
		// FSW ends at absolute index ph.end; the 32-dibit NID
		// immediately follows.
		nidStart := ph.end + 1 - c.bufBase
		if nidStart < 0 {
			continue // buffer already trimmed past this hit — drop it
		}
		if nidStart+frameLookahead > len(c.buf) {
			kept = append(kept, ph) // not enough buffered yet
			continue
		}
		c.parseFrame(c.buf, nidStart, ph.rot)
	}
	c.pending = kept
	c.trimBuffer()
	return next
}

// parseFrame decodes the NID + TSBK of one FSW hit. buf[nidStart:]
// must hold at least frameLookahead dibits — the caller guarantees it.
// rot is the FSW-search rotation: the sync detector matched after
// adding rot mod 4 to each input dibit, so the canonical dibit values
// the BCH / trellis decoders expect are recovered by subtracting rot
// (rotateDibits) before parsing.
func (c *ControlChannel) parseFrame(buf []uint8, nidStart int, rot uint8) {
	nidDibits := rotateDibits(buf[nidStart:nidStart+32], rot)
	nid, errs, err := NIDFromDibits(nidDibits)
	if err != nil {
		c.log.Debug("nid parse failed", "err", err, "errs", errs, "rot", rot)
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNIDBCH},
		})
		return
	}
	if errs > 0 {
		c.log.Debug("nid corrected", "errs", errs, "nac", nid.NAC, "rot", rot)
	}
	if nid.DUID != DUIDTrunkingSignaling {
		// Some non-control DUID — record but don't lock.
		c.log.Debug("non-control DUID", "duid", nid.DUID, "nac", nid.NAC)
		return
	}
	if !c.locked || c.lastNAC != nid.NAC {
		c.locked = true
		c.lastNAC = nid.NAC
		c.bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: LockState{FrequencyHz: c.freqHz, NAC: nid.NAC, DUID: nid.DUID},
		})
		c.log.Info("control channel locked", "nac", nid.NAC, "freq", c.freqHz, "rot", rot)
	}

	// The channel TSBK occupies the 98 dibits after the NID.
	tsbkStart := nidStart + 32
	tsbkDibits := rotateDibits(buf[tsbkStart:tsbkStart+98], rot)
	tsbk, metric, err := DecodeTSBKChannel(tsbkDibits)
	if err != nil {
		c.log.Debug("tsbk decode failed", "err", err, "metric", metric, "nac", nid.NAC)
		stage := events.StageTSBKTrellis
		if errors.Is(err, CRCError) {
			stage = events.StageTSBKCRC
		}
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: stage},
		})
		return
	}
	c.dispatchTSBK(tsbk, nid.NAC, metric)
}

// trimBuffer drops dibits no pending hit needs any more. With no
// pending hits the whole buffer is released; otherwise everything
// before the earliest pending hit's NID is dropped — keeping buf
// bounded to roughly one frame.
func (c *ControlChannel) trimBuffer() {
	keep := len(c.buf)
	for _, ph := range c.pending {
		if s := ph.end + 1 - c.bufBase; s >= 0 && s < keep {
			keep = s
		}
	}
	if keep > 0 {
		c.buf = append(c.buf[:0], c.buf[keep:]...)
		c.bufBase += keep
	}
}

// rotateDibits returns a copy of src with the inverse FSW-search
// rotation applied: each dibit value has `rot` subtracted mod 4.
// rot=0 short-circuits to avoid the copy.
func rotateDibits(src []uint8, rot uint8) []uint8 {
	if rot == 0 {
		return src
	}
	inv := (4 - rot) & 3
	out := make([]uint8, len(src))
	for i, d := range src {
		out[i] = (d + inv) & 3
	}
	return out
}

// dispatchTSBK routes a successfully-CRC'd TSBK to the right opcode
// handler. Unknown opcodes are still useful for diagnostics — they're
// logged at debug but not republished, since they're the bulk of what
// a busy site emits and would drown signal in noise.
func (c *ControlChannel) dispatchTSBK(t TSBK, nac uint16, metric int) {
	switch t.Opcode {
	case OpIdentifierUpdate:
		u := ParseIdentifierUpdate(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz)
	case OpGroupVoiceChannelGrant:
		c.publishGroupGrant(ParseGroupVoiceChannelGrant(t.Payload), nac)
	case OpGroupAffiliationResponse:
		c.publishAffiliation(ParseGroupAffiliationResponse(t.Payload), nac)
	case OpUnitRegistrationResponse:
		c.publishUnitRegistration(ParseUnitRegistrationResponse(t.Payload), nac)
	default:
		c.log.Debug("tsbk decoded",
			"opcode", t.Opcode, "lb", t.LB, "metric", metric, "nac", nac)
	}
}

// publishGroupGrant resolves the grant's channel through the band
// plan and publishes a trunking.Grant on the bus. If the channel ID
// hasn't been seen yet, a `decode.error` with stage="no-bandplan"
// is published instead — the engine can't do anything with a
// zero-frequency grant, and surfacing this lets operators see when
// a site's IdentifierUpdate cadence is too slow for their capture.
func (c *ControlChannel) publishGroupGrant(g GroupVoiceChannelGrant, nac uint16) {
	freq, err := c.bandPlan.Frequency(g.ChannelID, g.ChannelNumber)
	if err != nil {
		c.log.Debug("p25: grant before identifier update",
			"nac", nac, "id", g.ChannelID, "num", g.ChannelNumber, "err", err)
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNoBandPlan},
		})
		return
	}
	// SVC_OPTIONS bit layout per TIA-102.AABF: bit 7 = Emergency,
	// bit 6 = Protected (encryption indicator).
	emergency := g.ServiceOptions&0x80 != 0
	encrypted := g.ServiceOptions&0x40 != 0
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "p25",
			GroupID:     uint32(g.GroupAddress),
			SourceID:    g.SourceID,
			FrequencyHz: freq,
			ChannelID:   g.ChannelID,
			ChannelNum:  g.ChannelNumber,
			Encrypted:   encrypted,
			Emergency:   emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("p25: grant",
		"system", c.systemName, "nac", nac,
		"tg", g.GroupAddress, "src", g.SourceID,
		"id", g.ChannelID, "num", g.ChannelNumber, "freq_hz", freq,
		"enc", encrypted, "emer", emergency)
}

// publishAffiliation publishes a trunking.Affiliation on the bus when
// the site issues a Group Affiliation Response (opcode 0x28). No
// band-plan resolution is needed — affiliations don't carry channel
// info.
func (c *ControlChannel) publishAffiliation(g GroupAffiliationResponse, nac uint16) {
	c.bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            c.systemName,
			Protocol:          "p25",
			SourceID:          g.TargetID,
			GroupID:           uint32(g.GroupAddress),
			AnnouncementGroup: uint32(g.AnnouncementGroup),
			Response:          trunking.AffiliationResponse(g.Response),
			At:                c.now(),
		},
	})
	c.log.Debug("p25: affiliation",
		"system", c.systemName, "nac", nac,
		"src", g.TargetID, "tg", g.GroupAddress,
		"ann", g.AnnouncementGroup, "rsp", g.Response)
}

// publishUnitRegistration publishes a trunking.UnitRegistration on the
// bus when the site issues a Unit Registration Response (opcode 0x2C).
func (c *ControlChannel) publishUnitRegistration(u UnitRegistrationResponse, nac uint16) {
	c.bus.Publish(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   c.systemName,
			Protocol: "p25",
			SourceID: u.SourceID,
			WACN:     u.WACN,
			SystemID: u.SystemID,
			Response: trunking.RegistrationResponse(u.Response),
			At:       c.now(),
		},
	})
	c.log.Debug("p25: registration",
		"system", c.systemName, "nac", nac,
		"src", u.SourceID, "wacn", u.WACN, "sysid", u.SystemID,
		"rsp", u.Response)
}

// MarkLost publishes a CCLost event for the current frequency and resets
// the locked flag. Loss tracking is intentionally simple — the engine /
// hunter pair owns the watchdog logic and calls this when the control
// channel goes silent.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	// Drop accumulated dibits + pending hits so a re-acquisition
	// starts from a clean slate.
	c.buf = c.buf[:0]
	c.pending = c.pending[:0]
	c.bus.Publish(events.Event{
		Kind:    events.KindCCLost,
		Payload: LockState{FrequencyHz: c.freqHz, NAC: c.lastNAC},
	})
}
