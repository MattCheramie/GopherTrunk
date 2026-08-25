package tier3

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// burstLookback is the number of dibits the burst extends BEFORE
// the FSW sync match position. The DMR burst layout puts the
// 24-dibit sync at offsets 54..77 within the 132-dibit burst; the
// match index is the position of the LAST sync dibit, so the burst
// starts 77 dibits behind the match.
const burstLookback = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1 // 77

// burstLookahead is the number of dibits the burst extends AFTER
// the FSW sync match position (5 slot-type + 49 second-half = 54).
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
	// pending matches whose trailing 54 dibits haven't arrived yet.
	pending []dmr.Match
}

// Process consumes a window of raw dibits from the DMR receiver
// (the IQ → C4FM dibit chain in internal/radio/dmr/receiver/) and
// drives the Tier III control-channel state machine.
//
// On every Process call the adapter:
//
//   - Appends the new dibits to its cross-call buffer.
//   - Runs the multi-pattern SyncDetector against all 9 ETSI
//     sync words so any burst type (BS / MS / DM, voice / data /
//     embedded) can be matched.
//   - Queues each sync hit; processes hits whose trailing 54
//     dibits are now in the buffer by extracting the 132-dibit
//     burst, parsing the 20-bit slot-type Hamming(20,8) codeword,
//     and forwarding (Burst, SlotType) to IngestBurst.
//   - Trims the buffer to bufKeep dibits so any future sync match
//     has the full 77-dibit lookback available.
//
// baseIdx is the absolute dibit index of dibits[0]. The adapter
// tracks the buffer's absolute position so cross-call sync matches
// align correctly across chunk boundaries.
//
// Returns baseIdx + len(dibits) to match the YSF / P25 P1 / dPMR /
// NXDN / EDACS / Motorola / LTR / MPT 1327 Process contracts.
//
// On-air voice / data burst payload decoding (the BPTC(196,96) +
// CSBK CRC chain) lives inside IngestBurst; this adapter just gets
// 132 dibits into IngestBurst's hands. The slot-type Hamming(20,8)
// decoder filters out bursts whose slot type doesn't match
// DTCSBK before any BPTC work, so most noise drops here.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det: dmr.NewSyncDetector(nil, 2),
		}
	}
	p := c.proc

	// Track the buffer's absolute start the first time we receive
	// dibits. After trimming, bufStart advances as we drop older
	// dibits.
	if len(p.buf) == 0 {
		p.bufStart = baseIdx
	}
	p.buf = append(p.buf, dibits...)

	// Run sync detection on the freshly-appended dibits.
	matches, _ := p.det.Process(nil, dibits, baseIdx)
	p.pending = append(p.pending, matches...)

	// Process pending hits whose full burst is now in the buffer.
	bufEnd := p.bufStart + len(p.buf)
	keep := p.pending[:0]
	for _, m := range p.pending {
		burstStart := m.Index - burstLookback
		burstEnd := m.Index + burstLookahead + 1 // exclusive
		if burstEnd > bufEnd {
			// Need more trailing dibits.
			keep = append(keep, m)
			continue
		}
		if burstStart < p.bufStart {
			// Lookback already trimmed; drop this match.
			continue
		}
		offset := burstStart - p.bufStart
		// Decode the burst at both discriminator polarities. DMR's sync
		// alphabet is closed under the spectrum-inversion flip (issue
		// #264, RTL-SDR Blog V4 / R828D), so the sync match can't tell a
		// clean burst from an inverted one — we hand IngestBurst a
		// candidate at each polarity and let its slot-type Hamming(20,8)
		// + BPTC + CSBK CRC pick the real one (the wrong polarity is
		// dropped with no state change). Identity (k=0) is tried first
		// so clean R820T2 streams behave exactly as before.
		for _, k := range dmr.CandidatePolarities {
			var b dmr.Burst
			copy(b.Dibits[:], p.buf[offset:offset+dmr.BurstDibits])
			dmr.RotateBurstDibits(&b, k)

			// ParseSlotType returns the slot type + error count; we
			// accept partially-corrected slot types (positive errs) but
			// drop hard failures (slot.DataType won't match DTCSBK
			// anyway).
			slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
			if err != nil {
				continue
			}
			c.IngestBurst(&b, slot)
		}
	}
	p.pending = keep

	// Trim the buffer: keep the last bufKeep dibits.
	if len(p.buf) > bufKeep {
		drop := len(p.buf) - bufKeep
		copy(p.buf, p.buf[drop:])
		p.buf = p.buf[:bufKeep]
		p.bufStart += drop
	}
	return baseIdx + len(dibits)
}

// ResyncReset drops the Process adapter's cross-call dibit buffer and sync
// state so a receiver-side Reset (which restarts the dibit index at 0) can
// reacquire cleanly. Without it the adapter's absolute bufStart stays at the
// pre-reset value while every new sync match arrives at a small index, so
// each is discarded as "lookback already trimmed" for ever. Mirrors the
// TETRA ControlChannel's ResyncReset; call from the pipeline whenever the
// receiver is reset mid-stream.
//
// Precondition: called on the same goroutine as Process (from the pipeline,
// after rx.Process returns), so the proc swap never races a Process call.
func (c *ControlChannel) ResyncReset() {
	c.proc = nil
}
