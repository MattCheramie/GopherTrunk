package composer

import (
	"log/slog"
	"math"
)

// p25p1TxCounters is a snapshot of a P25 Phase 1 voice chain's cumulative
// decode counters (see runP25Phase1VoiceChain).
type p25p1TxCounters struct {
	ldus, uncorrectable, corrErrBits, lcRS, essRS uint64
	shortLDUs, missingSymbols                     int
}

// p25p1TxQuality splits a voice chain's decode counters per transmission
// (one over, one PTT) and logs one line for each, named by the radio that
// keyed it.
//
// The chain-wide "composer: p25p1 decode quality" line is cumulative over
// the whole chain, and with voice_call_grouping: conversation one chain
// carries every over on the talkgroup, from every radio (#1242: a 34 s
// "call" whose corrected bit errors climbed from ~3 to ~100 per LDU read as
// the call degrading at its end, but the counters could not say whether one
// distant radio was bad from its first frame, or the receiver drifted). A
// per-transmission line with the source unit, the error rate per LDU and the
// carrier offset answers that from a log alone.
//
// A transmission ends at a terminator (TDU / TDULC), when an LDU1's link
// control names a different source unit (the terminator between two overs
// can be lost on a weak signal), or when the chain ends. Not safe for
// concurrent use; the chain calls it from its receiver goroutine only.
type p25p1TxQuality struct {
	log    *slog.Logger
	serial string
	// src is the source unit of the current transmission (0 until an
	// LDU1 / TDULC link control names one).
	src   uint32
	start p25p1TxCounters
	n     int // transmissions logged so far
}

// sourceSeen records the source unit an LDU1 (or TDULC) link control named.
// pre is the chain's counters from BEFORE the LDU that carried it, so when
// the source changes, that LDU is counted toward the new transmission.
// afcHz is the receiver's current carrier offset estimate.
func (q *p25p1TxQuality) sourceSeen(src uint32, pre p25p1TxCounters, afcHz float64) {
	if src == 0 {
		return
	}
	if q.src != 0 && src != q.src {
		q.flush(pre, afcHz)
	}
	q.src = src
}

// end closes the current transmission at now (the chain's counters after
// its last LDU) and starts the next one there.
func (q *p25p1TxQuality) end(now p25p1TxCounters, afcHz float64) {
	q.flush(now, afcHz)
	q.src = 0
}

func (q *p25p1TxQuality) flush(now p25p1TxCounters, afcHz float64) {
	d := p25p1TxCounters{
		ldus:           now.ldus - q.start.ldus,
		uncorrectable:  now.uncorrectable - q.start.uncorrectable,
		corrErrBits:    now.corrErrBits - q.start.corrErrBits,
		lcRS:           now.lcRS - q.start.lcRS,
		essRS:          now.essRS - q.start.essRS,
		shortLDUs:      now.shortLDUs - q.start.shortLDUs,
		missingSymbols: now.missingSymbols - q.start.missingSymbols,
	}
	q.start = now
	if d.ldus == 0 {
		return
	}
	q.n++
	q.log.Info("composer: p25p1 transmission quality",
		"serial", q.serial, "tx", q.n, "src", q.src,
		"ldus", d.ldus,
		"corrected_bit_errs", d.corrErrBits,
		"errs_per_ldu", math.Round(float64(d.corrErrBits)/float64(d.ldus)*10)/10,
		"uncorrectable_ldus", d.uncorrectable,
		"lc_rs_uncorrectable", d.lcRS,
		"ess_rs_uncorrectable", d.essRS,
		"short_ldus", d.shortLDUs,
		"carrier_offset_hz", math.Round(afcHz))
}
