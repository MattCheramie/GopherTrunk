package tier2

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
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
			det: dmr.NewSyncDetector(c.syncPatterns, 2),
		}
		// Late entry (see ConventionalChannel.voice): the embedded-LC
		// assembler runs beside the burst slicer on the same dibits. A
		// base-station carrier interleaves two timeslots, so the interleaved
		// decoder is used whenever the system decodes interleaved voice;
		// direct mode (Tier I) is genuinely single-slot.
		if c.interleavedVoice {
			c.voice = dmrvoice.NewInterleavedDecoder()
		} else {
			c.voice = dmrvoice.NewDecoder()
		}
	}
	p := c.proc

	if len(p.buf) == 0 {
		p.bufStart = baseIdx
	}
	p.buf = append(p.buf, dibits...)

	matches, _ := p.det.Process(nil, dibits, baseIdx)
	c.cnt.syncHits.Add(uint64(len(matches)))
	p.pending = append(p.pending, matches...)

	// Late entry: assemble voice superframes and hand every CRC-valid
	// embedded LC to the state machine. Runs BEFORE the burst pass so the
	// closing superframes of a transmission (emitted a span behind the
	// slicer) refresh their still-tracked call rather than trailing its
	// terminator; the endedAtDibit gate in ingestVoiceSuperframe covers the
	// chunkings where they still trail it.
	voiceIn := dibits
	if c.polarity == int(dmr.PolarityFlip) {
		// Spectrum-inverted front end: undo the flip for the voice assembler
		// (its voice syncs and AMBE bits are canonical-polarity).
		voiceIn = make([]uint8, len(dibits))
		for i, d := range dibits {
			voiceIn[i] = (d + dmr.PolarityFlip) & 3
		}
	}
	for _, sf := range c.voice.Process(voiceIn, baseIdx) {
		c.ingestVoiceSuperframe(sf)
	}

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
		// Decode the burst at both discriminator polarities to recover
		// spectrum-inverted reception (issue #264, RTL-SDR Blog V4 /
		// R828D); IngestBurst's FEC drops the wrong polarity with no
		// state change. Identity (k=0) is tried first. Same as the Tier
		// III adapter — except that once a FEC-valid burst has fixed the
		// stream's polarity, only that polarity is tried.
		//
		// Only DATA-sync bursts carry a slot type. A voice burst A has AMBE
		// bits in the slot-type positions, and Golay(20,8) decodes ~1/3 of
		// arbitrary 20-bit words to SOME codeword — so one voice burst A in
		// ~12 used to parse as a Terminator-with-LC and, through the
		// single-call fallback, end a live call mid-transmission (the 9 Sep
		// "false call ended" report; measured on the operator's captures: a
		// terminator 6 bursts before the next voice superframe of the same
		// over). Voice bursts are the late-entry assembler's business
		// (above), never the slot-type path's. Because data and voice syncs
		// are each other's flip image the check is per polarity; the
		// polarity lock makes it exact.
		for _, k := range dmr.CandidatePolarities {
			if c.polarity >= 0 && uint8(c.polarity) != k {
				continue
			}
			if !dmr.SyncIsDataAtPolarity(m.Pattern, k) {
				continue
			}
			var b dmr.Burst
			copy(b.Dibits[:], p.buf[offset:offset+dmr.BurstDibits])
			dmr.RotateBurstDibits(&b, k)

			slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
			if err != nil {
				continue
			}
			c.ingestDibit = burstStart
			c.burstValid = false
			c.IngestBurst(&b, slot)
			c.ingestDibit = -1
			if c.burstValid && c.polarity < 0 {
				c.polarity = int(k)
				c.log.Debug("dmr/tier2: discriminator polarity fixed by first FEC-valid burst",
					"polarity", k, "sync", m.Pattern.Name)
			}
		}
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
