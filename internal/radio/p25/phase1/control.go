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

// Process consumes a window of dibits and runs detection/parsing. baseIdx
// is the absolute dibit index of dibits[0]. Returns the new baseIndex.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, next := c.det.Process(nil, dibits, baseIdx)
	if len(hits) == 0 && len(dibits) > 0 && !c.locked {
		now := c.now()
		if now.Sub(c.lastNoHitsAt) >= noHitsThrottle {
			c.log.Debug("p25/phase1: no FSW hits in chunk",
				"system", c.systemName, "freq_hz", c.freqHz, "dibits", len(dibits))
			c.lastNoHitsAt = now
		}
	}
	for _, h := range hits {
		// FSW ends at index h (relative to the absolute dibit stream). The
		// 32-dibit NID immediately follows.
		startInWindow := h - baseIdx + 1
		if startInWindow+32 > len(dibits) {
			// Crosses the buffer edge; future calls will have the rest.
			continue
		}
		nidDibits := dibits[startInWindow : startInWindow+32]
		nid, errs, err := NIDFromDibits(nidDibits)
		if err != nil {
			c.log.Debug("nid parse failed", "err", err, "errs", errs)
			c.bus.Publish(events.Event{
				Kind:    events.KindDecodeError,
				Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNIDBCH},
			})
			continue
		}
		if errs > 0 {
			c.log.Debug("nid corrected", "errs", errs, "nac", nid.NAC)
		}
		if nid.DUID != DUIDTrunkingSignaling {
			// Some non-control DUID — record but don't lock.
			c.log.Debug("non-control DUID", "duid", nid.DUID, "nac", nid.NAC)
			continue
		}
		if !c.locked || c.lastNAC != nid.NAC {
			c.locked = true
			c.lastNAC = nid.NAC
			c.bus.Publish(events.Event{
				Kind:    events.KindCCLocked,
				Payload: LockState{FrequencyHz: c.freqHz, NAC: nid.NAC, DUID: nid.DUID},
			})
			c.log.Info("control channel locked", "nac", nid.NAC, "freq", c.freqHz)
		}

		// Try to extract the next TSBK that follows the NID. The
		// channel TSBK occupies 98 dibits; if the buffer is short we
		// defer to a later call (the buffer-edge case).
		tsbkStart := startInWindow + 32
		if tsbkStart+98 > len(dibits) {
			continue
		}
		tsbk, metric, err := DecodeTSBKChannel(dibits[tsbkStart : tsbkStart+98])
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
			continue
		}
		c.dispatchTSBK(tsbk, nid.NAC, metric)
	}
	return next
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

// MarkLost publishes a CCLost event for the current frequency and resets
// the locked flag. Loss tracking is intentionally simple — the engine /
// hunter pair owns the watchdog logic and calls this when the control
// channel goes silent.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{
		Kind:    events.KindCCLost,
		Payload: LockState{FrequencyHz: c.freqHz, NAC: c.lastNAC},
	})
}
