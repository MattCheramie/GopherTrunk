package voice

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TerminatorDetector finds the "Terminator with LC" burst that ends a DMR
// transmission in a followed traffic-channel dibit stream. The voice
// Decoder locks on voice sync and so never sees the terminator (a data
// burst); this detector runs alongside it and lets the voice chain end the
// call at once instead of waiting out the composer's hangtime — the
// traffic-channel analogue of the Tier II conventional decoder's
// KindCallRelease (issue-parity with TETRA D-RELEASE).
//
// It reuses the exact, trusted decode chain the Tier II control path uses:
// multi-pattern sync detection over the data sync words, 132-dibit burst
// slicing at both discriminator polarities, slot-type Hamming(20,8) decode,
// BPTC(196,96), and an RS(12,9) parity check with the Terminator-with-LC
// seed. A burst is only reported when it clears every one of those layers,
// so a mid-call data burst cannot be miscorrected into a false end-of-call.
//
// The recovered FLC's destination names the call being torn down; the
// caller ends its call only when that destination matches, so a terminator
// on the other timeslot of an interleaved carrier never ends this call.

// terminatorSyncs are the data-burst sync words that frame a Terminator
// with LC (BS-sourced and MS/DM direct). Voice syncs are intentionally
// excluded — the voice Decoder already handles those.
var terminatorSyncs = []dmr.SyncPattern{dmr.BSData, dmr.MSData, dmr.DMData1, dmr.DMData2}

const (
	// termBurstLookback / termBurstLookahead mirror the Tier II Process
	// adapter: the sync match index is the last sync dibit, so the burst
	// starts 77 dibits behind it and ends 54 ahead.
	termBurstLookback  = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1
	termBurstLookahead = dmr.SlotTypeDibits + dmr.HalfPayloadDibits
	termBufKeep        = termBurstLookback + termBurstLookahead + 32
)

// TerminatorDetector is stateful and not safe for concurrent use;
// construct one per voice-decode chain.
type TerminatorDetector struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int
	pending  []dmr.Match
}

// NewTerminatorDetector returns a ready detector.
func NewTerminatorDetector() *TerminatorDetector {
	return &TerminatorDetector{det: dmr.NewSyncDetector(terminatorSyncs, 2)}
}

// Process consumes a window of dibits and returns the recovered Full Link
// Control of every valid Terminator-with-LC burst that completed within it.
// baseIdx is the absolute dibit index of dibits[0]; it must be
// monotonically non-decreasing across calls.
func (d *TerminatorDetector) Process(dibits []uint8, baseIdx int) []dmr.FLC {
	if len(d.buf) == 0 {
		d.bufStart = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	matches, _ := d.det.Process(nil, dibits, baseIdx)
	d.pending = append(d.pending, matches...)

	var out []dmr.FLC
	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, m := range d.pending {
		burstStart := m.Index - termBurstLookback
		burstEnd := m.Index + termBurstLookahead + 1
		if burstEnd > bufEnd {
			keep = append(keep, m) // trailing burst not fully buffered yet
			continue
		}
		if burstStart < d.bufStart {
			continue // anchor fell off the front of the buffer
		}
		offset := burstStart - d.bufStart
		if flc, ok := decodeTerminator(d.buf[offset : offset+dmr.BurstDibits]); ok {
			out = append(out, flc)
		}
	}
	d.pending = keep

	if len(d.buf) > termBufKeep {
		drop := len(d.buf) - termBufKeep
		copy(d.buf, d.buf[drop:])
		d.buf = d.buf[:termBufKeep]
		d.bufStart += drop
	}
	return out
}

// decodeTerminator decodes a candidate 132-dibit burst at both
// discriminator polarities and returns the FLC when it is a genuine
// Terminator with LC — slot-type Hamming, BPTC, and RS(12,9) with the
// terminator seed all clearing. The wrong polarity is dropped by the FEC
// with no side effect, so spectrum-inverted reception (issue #264) is
// handled the same way the Tier II adapter handles it.
func decodeTerminator(burstDibits []uint8) (dmr.FLC, bool) {
	for _, k := range dmr.CandidatePolarities {
		var b dmr.Burst
		copy(b.Dibits[:], burstDibits)
		dmr.RotateBurstDibits(&b, k)

		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil || slot.DataType != dmr.DTTerminatorWithLC {
			continue
		}
		bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
		if errs < 0 {
			continue
		}
		info := infoBitsToBytes96(bits)
		if !framing.VerifyRS12_9(info, framing.RS129SeedTerminatorLC) {
			continue
		}
		if flc, err := dmr.ParseFLC(info); err == nil {
			return flc, true
		}
	}
	return dmr.FLC{}, false
}

// infoBitsToBytes96 packs a 96-bit slice (each entry 0/1, MSB-first) into
// 12 bytes — the 9 FLC octets plus the 3 RS(12,9) parity octets.
func infoBitsToBytes96(bits []byte) []byte {
	out := make([]byte, 12)
	for i := 0; i < 96 && i < len(bits); i++ {
		if bits[i]&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
