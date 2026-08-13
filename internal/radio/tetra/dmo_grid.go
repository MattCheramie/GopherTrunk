package tetra

// DMO burst-train qualification — the noise gate the DMO grant path needs.
//
// A DNB is detected by correlating an 11-dibit normal training sequence at
// tolerance 2 (extractDMBursts, dmo.go), under two training sequences × four
// residual rotations = eight matched filters. That is deliberately loose so a
// marginal real burst is not missed, but it means the detector fires freely on
// noise: the number of 11-dibit sequences within Hamming distance 2 of a given
// pattern is
//
//	Σ_{k≤2} C(11,k)·3^k = 1 + 33 + 495 = 529
//
// so p ≈ 529/4^11 ≈ 1.26e-4 per position per filter, ≈ 1.0e-3 across the eight
// filters, and at the 18 000 dibit/s TETRA symbol rate that is ≈ 18 spurious
// DNB "detections" per second off an idle channel. A DMO channel is idle most
// of the time (EN 300 396-2: infrastructure-less, silent between transmissions),
// so a raw DNB count is not evidence of traffic — it is a noise meter. Treating
// it as traffic is what made the daemon grant and open a recording ~230 ms after
// startup, and then latch that grant forever so the operator's real PTT never
// granted at all.
//
// The discriminator is the TDMA slot grid. A real transmission is emitted by one
// radio on one clock, so every one of its bursts lands on the 255-dibit timeslot
// grid and every DNB shares ONE residue mod 255 (both normal training sequences
// sit at the same intra-slot offset, so NTS1 and NTS2 hits agree). Noise hits are
// uniform over all 255 residues. Voting on the residue therefore separates the
// two populations without any decode, at negligible cost, and — importantly —
// without hardcoding a spec-derived DSB→DNB lead offset: the residue is LEARNED
// from the stream, so a wrong constant cannot silently stop DMO from granting.
//
// This is the DMO analogue of the TMO slot anchor (TrafficExtractor.slotOf,
// traffic.go), except TMO only labels bursts with their slot while DMO must gate
// on it.

const (
	// dmSlotDibits is the dibit span of one TETRA timeslot (510 bits). The grid
	// a single transmitter's bursts are laid on. Mirrors ndbSlotDibits.
	dmSlotDibits = 255
	// dmGridTolerance is how far (in dibits) a lead may sit from the latched
	// residue and still qualify, absorbing symbol-clock jitter and the odd
	// slipped symbol from the timing-recovery loop.
	dmGridTolerance = 1
	// dmGridWindow is the residue-vote window in dibits — 1 s at the 18 kdibit/s
	// TETRA symbol rate. The dibit index is used as the clock throughout (it is
	// exact and deterministic, unlike wall time, which makes this testable and
	// immune to a stalled decode goroutine).
	dmGridWindow = 18000
	// dmGridMinTrain is how many in-window leads must agree on a residue before
	// it is latched as a real burst train. Sizing: noise puts ~18 leads/s across
	// 255 residues, so a 1 s window holds ~18 leads and the expected number
	// matching any given residue within ±dmGridTolerance is 18·3/255 ≈ 0.21;
	// requiring 5 further matches is P(Poisson(0.21) ≥ 5) ≈ 1.4e-6 per detection,
	// i.e. a false latch about once per 11 hours. A real transmission puts
	// ~17 leads/s on ONE residue and so latches after ~6 bursts ≈ 340 ms.
	dmGridMinTrain = 6
	// dmGridMaxLeads bounds the ring. A full second of noise is ~18 leads and of
	// traffic ~17, so 64 is comfortable headroom; beyond it the oldest lead is
	// dropped regardless of the window.
	dmGridMaxLeads = 64
	// dmGridTrainGap is the largest slot gap between consecutive qualified bursts
	// that still counts as the same transmission. A transmitting radio emits a burst
	// every frame, so a real train never gaps by more than a few slots; noise that
	// happens to land on the latched residue arrives at only ~18·3/255 ≈ 0.2/s, i.e.
	// a mean gap of ~5 s ≈ 90 000 dibits. Anything past this threshold means the
	// train has ended, so the latch is dropped and the next transmission must re-vote
	// — without it, that noise trickle silently keeps a finished transmission "alive"
	// and the grant edge never re-arms.
	dmGridTrainGap = 16 * dmSlotDibits
	// dmGridDriftAdjust is how many dibits of accumulated signed offset (each
	// qualified lead contributes -1, 0 or +1) shift the latched residue by one.
	// The symbol-clock loop tracks the transmitter, so the recovered index does
	// not drift steadily — but an occasional slipped symbol does step it, and
	// without this the latched residue would eventually sit at the edge of the
	// tolerance band and start dropping real bursts mid-transmission.
	dmGridDriftAdjust = 4
)

// DMSlotGrid votes the recent DNB training-sequence lead indices onto the
// 255-dibit timeslot grid and reports which of them belong to a real burst
// train. Not safe for concurrent use; drive it from the single goroutine that
// drives DMStreamExtractor.
type DMSlotGrid struct {
	// leads is a bounded, time-ordered ring of recent absolute DNB lead indices.
	leads []int

	// residue is the latched traffic residue (mod dmSlotDibits) once
	// dmGridMinTrain leads have agreed on it; latched is false until then.
	// drift integrates the signed offset of qualified leads from it, so a
	// slipped symbol re-centres the band instead of walking off its edge.
	residue int
	latched bool
	drift   int

	// lastQualified is the absolute lead index of the most recent qualified DNB,
	// and qualifiedSeen whether there has been one since the last Reset. Callers
	// use these to measure a traffic drought in the dibit domain.
	lastQualified int
	qualifiedSeen bool
}

// NewDMSlotGrid returns an empty grid with no residue latched.
func NewDMSlotGrid() *DMSlotGrid { return &DMSlotGrid{} }

// Observe records one DNB training-sequence lead (an absolute dibit index, as
// carried in DMBurst.Lead) and reports whether it belongs to a real burst train.
//
// Before a residue is latched, a lead qualifies only once dmGridMinTrain leads
// in the trailing dmGridWindow agree on its residue — at which point that
// residue is latched and this lead (the one that completed the vote) qualifies.
// After latching, a lead qualifies iff it matches the latched residue, so the
// rest of the transmission qualifies immediately and noise on other residues
// does not.
func (g *DMSlotGrid) Observe(lead int) bool {
	g.trim(lead)

	if g.latched {
		// Drop a stale latch rather than trusting it, in two cases: the burst train
		// has ended (gap past dmGridTrainGap — otherwise noise landing on the latched
		// residue keeps a finished transmission alive), or the upstream dibit index
		// ran backwards because the receiver re-anchored (a Reset), which would
		// otherwise reject every burst of the new stream forever.
		if g.qualifiedSeen && (lead-g.lastQualified > dmGridTrainGap || lead < g.lastQualified) {
			g.Reset()
		} else {
			if !residueNear(lead, g.residue) {
				g.push(lead)
				return false
			}
			g.push(lead)
			g.track(lead)
			g.markQualified(lead)
			return true
		}
	}

	// Not latched: count how many in-window leads share this lead's residue.
	r := residueOf(lead)
	agree := 1 // this lead
	sum := 0   // signed offsets of the agreeing leads, for the centred latch
	for _, l := range g.leads {
		if d, ok := residueOffset(l, r); ok {
			agree++
			sum += d
		}
	}
	g.push(lead)
	if agree < dmGridMinTrain {
		return false
	}
	// Latch the CENTRE of the agreeing leads, not this lead's residue: with a
	// ±dmGridTolerance band, latching on an extreme of a jittering train leaves
	// the other extreme outside the band and drops half the real bursts.
	g.residue = wrapResidue(r + roundDiv(sum, agree))
	g.latched = true
	g.drift = 0
	g.markQualified(lead)
	return true
}

// track integrates a qualified lead's signed offset from the latched residue and
// re-centres the band once the accumulated slip reaches dmGridDriftAdjust.
func (g *DMSlotGrid) track(lead int) {
	d, ok := residueOffset(lead, g.residue)
	if !ok {
		return
	}
	g.drift += d
	switch {
	case g.drift >= dmGridDriftAdjust:
		g.residue = wrapResidue(g.residue + 1)
		g.drift = 0
	case g.drift <= -dmGridDriftAdjust:
		g.residue = wrapResidue(g.residue - 1)
		g.drift = 0
	}
}

// Latched reports whether a traffic residue has been identified.
func (g *DMSlotGrid) Latched() bool { return g.latched }

// LastQualifiedLead returns the absolute dibit index of the most recent
// qualified DNB and whether there has been one.
func (g *DMSlotGrid) LastQualifiedLead() (int, bool) {
	return g.lastQualified, g.qualifiedSeen
}

// Reset drops the latched residue and all buffered leads, so the next
// transmission re-learns its own grid. Callers invoke it on a traffic drought
// and on a receiver re-anchor.
func (g *DMSlotGrid) Reset() {
	g.leads = g.leads[:0]
	g.residue = 0
	g.latched = false
	g.drift = 0
	g.lastQualified = 0
	g.qualifiedSeen = false
}

func (g *DMSlotGrid) markQualified(lead int) {
	g.lastQualified = lead
	g.qualifiedSeen = true
}

// push appends a lead to the ring, dropping the oldest when it is full.
func (g *DMSlotGrid) push(lead int) {
	if len(g.leads) >= dmGridMaxLeads {
		copy(g.leads, g.leads[1:])
		g.leads = g.leads[:len(g.leads)-1]
	}
	g.leads = append(g.leads, lead)
}

// trim drops leads that have fallen out of the trailing dmGridWindow relative to
// the newest lead. Leads arrive in ascending order (DMStreamExtractor emits by
// ascending Lead), so this is a prefix drop.
func (g *DMSlotGrid) trim(newest int) {
	cut := 0
	for cut < len(g.leads) && newest-g.leads[cut] > dmGridWindow {
		cut++
	}
	if cut > 0 {
		g.leads = append(g.leads[:0], g.leads[cut:]...)
	}
}

// residueOf is the lead's position within the 255-dibit slot, normalised to
// [0, dmSlotDibits) for negative indices too.
func residueOf(lead int) int { return wrapResidue(lead) }

// wrapResidue normalises any integer into [0, dmSlotDibits).
func wrapResidue(v int) int {
	r := v % dmSlotDibits
	if r < 0 {
		r += dmSlotDibits
	}
	return r
}

// residueOffset returns lead's signed offset from residue r, measured the
// shorter way round the 255-residue ring, and whether that offset is within
// dmGridTolerance.
func residueOffset(lead, r int) (int, bool) {
	d := residueOf(lead) - r
	if d > dmSlotDibits/2 {
		d -= dmSlotDibits
	} else if d < -dmSlotDibits/2 {
		d += dmSlotDibits
	}
	a := d
	if a < 0 {
		a = -a
	}
	return d, a <= dmGridTolerance
}

// residueNear reports whether lead's residue is within dmGridTolerance of r.
func residueNear(lead, r int) bool {
	_, ok := residueOffset(lead, r)
	return ok
}

// roundDiv divides n by d rounding to nearest (ties away from zero).
func roundDiv(n, d int) int {
	if d == 0 {
		return 0
	}
	if n >= 0 {
		return (2*n + d) / (2 * d)
	}
	return -((-2*n + d) / (2 * d))
}
