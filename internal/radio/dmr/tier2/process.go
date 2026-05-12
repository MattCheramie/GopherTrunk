package tier2

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// burstLookback is the number of dibits the burst extends BEFORE the
// FSW sync match position. The DMR burst layout puts the 24-dibit
// sync at offsets 54..77 within the 132-dibit burst; the match
// index is the position of the LAST sync dibit, so the burst starts
// 77 dibits behind the match.
const burstLookback = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1 // 77

// burstLookahead is the number of dibits the burst extends AFTER the
// FSW sync match position (5 slot-type + 49 second-half = 54).
const burstLookahead = dmr.SlotTypeDibits + dmr.HalfPayloadDibits // 54

// bufKeep is the minimum dibit retention so any incoming sync match
// can look back the full 77 dibits + a small safety margin.
const bufKeep = burstLookback + burstLookahead + 32 // 163

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on first use.
type processState struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int // absolute dibit index of buf[0]
	pending  []dmr.Match
}

// Process consumes a window of raw dibits from the DMR receiver
// (the IQ → C4FM dibit chain in internal/radio/dmr/receiver/) and
// drives the Tier II per-repeater state machine.
//
// Same shape as the Tier III adapter — buffer dibits, run the
// multi-pattern SyncDetector against all 9 ETSI sync words, slice
// 132-dibit bursts whose trailing 54 dibits are now in the buffer,
// decode the slot-type Hamming(20,8) codeword, and hand
// (Burst, SlotType) to IngestBurst. The state machine then routes
// the burst based on DataType (DTVoiceLCHeader / DTTerminatorWithLC
// drive the grant flow; everything else just updates the lock state
// for cc.locked / cc.lost.)
//
// baseIdx is the absolute dibit index of dibits[0]. The adapter
// tracks the buffer's absolute position so cross-call sync matches
// align correctly across chunk boundaries.
//
// Returns baseIdx + len(dibits) to match the Tier III / NXDN /
// other Process contracts.
func (c *ConventionalChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det: dmr.NewSyncDetector(nil, 2),
		}
	}
	p := c.proc

	if len(p.buf) == 0 {
		p.bufStart = baseIdx
	}
	p.buf = append(p.buf, dibits...)

	matches, _ := p.det.Process(nil, dibits, baseIdx)
	p.pending = append(p.pending, matches...)

	bufEnd := p.bufStart + len(p.buf)
	keep := p.pending[:0]
	for _, m := range p.pending {
		burstStart := m.Index - burstLookback
		burstEnd := m.Index + burstLookahead + 1
		if burstEnd > bufEnd {
			keep = append(keep, m)
			continue
		}
		if burstStart < p.bufStart {
			continue
		}
		offset := burstStart - p.bufStart
		var b dmr.Burst
		copy(b.Dibits[:], p.buf[offset:offset+dmr.BurstDibits])

		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil {
			continue
		}
		c.IngestBurst(&b, slot)
	}
	p.pending = keep

	if len(p.buf) > bufKeep {
		drop := len(p.buf) - bufKeep
		copy(p.buf, p.buf[drop:])
		p.buf = p.buf[:bufKeep]
		p.bufStart += drop
	}
	return baseIdx + len(dibits)
}
