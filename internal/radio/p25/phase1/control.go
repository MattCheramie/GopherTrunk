package phase1

import (
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// ControlChannel consumes a stream of P25 Phase 1 dibits (already symbol-
// time-recovered and mapped via SymbolToDibit) and emits trunking events
// onto an events.Bus.
//
// This is the minimum scaffold: dibit window → FSW detect → NID parse →
// (placeholder) TSBK extraction. Trellis decoding and TSBK CRC validation
// for the live stream land alongside the interleaver work in a follow-up
// — for now the state machine emits CCLocked / CCLost events when a NID
// with a TSDU DUID is observed, which is enough to drive the CC hunter
// and gives downstream layers a stable surface to subscribe to.
type ControlChannel struct {
	bus     *events.Bus
	log     *slog.Logger
	det     *SyncDetector
	freqHz  uint32
	locked  bool
	lastNAC uint16
}

func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	if log == nil {
		log = slog.Default()
	}
	return &ControlChannel{
		bus:    bus,
		log:    log,
		det:    NewSyncDetector(4),
		freqHz: freqHz,
	}
}

// LockState is the payload of CCLocked / CCLost events.
type LockState struct {
	FrequencyHz uint32
	NAC         uint16
	DUID        DUID
}

// Process consumes a window of dibits and runs detection/parsing. baseIdx
// is the absolute dibit index of dibits[0]. Returns the new baseIndex.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, next := c.det.Process(nil, dibits, baseIdx)
	for _, h := range hits {
		// FSW ends at index h (relative to the absolute dibit stream). The
		// 32-dibit NID immediately follows.
		startInWindow := h - baseIdx + 1
		if startInWindow+32 > len(dibits) {
			// Crosses the buffer edge; future calls will have the rest.
			continue
		}
		nidDibits := dibits[startInWindow : startInWindow+32]
		nid, err := NIDFromDibits(nidDibits)
		if err != nil {
			c.log.Debug("nid parse failed", "err", err)
			continue
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
	}
	return next
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
