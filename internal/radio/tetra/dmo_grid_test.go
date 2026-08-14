package tetra

import (
	"math/rand"
	"testing"
)

// TestDMSlotGridLatchesTrafficResidue drives a grid with leads laid on a real
// 255-dibit slot grid and asserts it latches that residue and qualifies the
// train once dmGridMinTrain bursts agree.
func TestDMSlotGridLatchesTrafficResidue(t *testing.T) {
	g := NewDMSlotGrid()
	const start = 100_000 // arbitrary absolute index; residue 100000%255 = 130

	qualified := 0
	for i := 0; i < 20; i++ {
		if g.Observe(start + i*dmSlotDibits) {
			qualified++
		}
	}
	if !g.Latched() {
		t.Fatalf("grid did not latch on a clean %d-dibit burst train", dmSlotDibits)
	}
	if g.residue != residueOf(start) {
		t.Errorf("latched residue = %d, want %d", g.residue, residueOf(start))
	}
	// The first dmGridMinTrain-1 leads vote but do not yet qualify; every lead
	// from the one that completes the vote onwards does.
	want := 20 - (dmGridMinTrain - 1)
	if qualified != want {
		t.Errorf("qualified %d of 20 leads, want %d", qualified, want)
	}
	if last, ok := g.LastQualifiedLead(); !ok || last != start+19*dmSlotDibits {
		t.Errorf("LastQualifiedLead() = %d,%v; want %d,true", last, ok, start+19*dmSlotDibits)
	}
}

// TestDMSlotGridToleratesSymbolJitter pins that a ±dmGridTolerance wobble in the
// recovered symbol clock does not break a train: the grid still latches, and once
// latched it centres the band so it does not drop bursts on the far edge.
//
// A full ±1 dibit wobble on EVERY burst is far harsher than a tracking symbol
// clock produces, so the latch is allowed to take a few extra bursts to find six
// mutually-agreeing leads; what must not happen is bursts being dropped after it.
func TestDMSlotGridToleratesSymbolJitter(t *testing.T) {
	g := NewDMSlotGrid()
	const (
		start  = 4_000
		bursts = 24
	)
	jitter := []int{0, 1, 0, -1, 0, 1, -1, 0, 1, 0, -1, 1}
	latchedAt := -1
	for i := 0; i < bursts; i++ {
		j := jitter[i%len(jitter)]
		lead := start + i*dmSlotDibits + j
		got := g.Observe(lead)
		if latchedAt >= 0 && !got {
			t.Errorf("lead %d (jitter %+d, burst %d) dropped after latch at burst %d",
				lead, j, i, latchedAt)
		}
		if got && latchedAt < 0 {
			latchedAt = i
		}
	}
	if !g.Latched() {
		t.Fatalf("grid did not latch through ±%d dibit jitter", dmGridTolerance)
	}
	if latchedAt > dmGridMinTrain+4 {
		t.Errorf("latched only at burst %d, want by burst %d", latchedAt, dmGridMinTrain+4)
	}
	// The centred latch must sit at the middle of the jitter band, not an edge.
	if want := residueOf(start); g.residue != want {
		t.Errorf("latched residue = %d, want the band centre %d", g.residue, want)
	}
}

// TestDMSlotGridTracksSlippedSymbols covers a symbol-clock slip mid-transmission:
// the recovered dibit index steps by one, so every later burst sits one dibit off
// the original residue. Without drift tracking the train would sit at the edge of
// the tolerance band and a second slip would drop it entirely.
func TestDMSlotGridTracksSlippedSymbols(t *testing.T) {
	g := NewDMSlotGrid()
	const start = 60_000
	lead := start
	for i := 0; i < 12; i++ { // establish the train
		g.Observe(lead)
		lead += dmSlotDibits
	}
	if !g.Latched() {
		t.Fatalf("did not latch")
	}
	base := g.residue

	// Two slips in the same direction, well separated.
	drops := 0
	for slip := 0; slip < 2; slip++ {
		lead++ // the slipped symbol
		for i := 0; i < 12; i++ {
			if !g.Observe(lead) {
				drops++
			}
			lead += dmSlotDibits
		}
	}
	if drops != 0 {
		t.Errorf("dropped %d bursts across two symbol slips", drops)
	}
	if g.residue == base {
		t.Errorf("latched residue never followed the slips (still %d)", base)
	}
}

// TestDMSlotGridRejectsNoise is the core regression: the DNB correlator fires
// ~18 times a second on an idle channel (see dmo_grid.go for the arithmetic),
// and those detections are uniform over the 255 residues. Ten simulated seconds
// of that must not latch a traffic residue or qualify a burst — otherwise the
// daemon grants and opens a recording on a silent DMO channel.
func TestDMSlotGridRejectsNoise(t *testing.T) {
	g := NewDMSlotGrid()
	rng := rand.New(rand.NewSource(20260813))

	const (
		seconds       = 10
		leadsPerSec   = 18 // measured false-alarm rate off an idle channel
		meanSpacing   = dmGridWindow / leadsPerSec
		totalExpected = seconds * leadsPerSec
	)
	lead := 0
	qualified := 0
	for i := 0; i < totalExpected; i++ {
		// Uniformly-random spacing → uniformly-distributed residues, which is
		// what an uncorrelated correlator false alarm looks like.
		lead += 1 + rng.Intn(2*meanSpacing)
		if g.Observe(lead) {
			qualified++
		}
	}
	if qualified != 0 {
		t.Errorf("qualified %d of %d noise detections, want 0 (latched=%v residue=%d)",
			qualified, totalExpected, g.Latched(), g.residue)
	}
	if g.Latched() {
		t.Errorf("latched residue %d on pure noise", g.residue)
	}
}

// TestDMSlotGridQualifiesTrafficThroughNoise checks the discriminator works when
// both populations are present at once — a real train buried in the same ~18/s
// false-alarm rain, which is what the live pipeline actually sees.
func TestDMSlotGridQualifiesTrafficThroughNoise(t *testing.T) {
	g := NewDMSlotGrid()
	rng := rand.New(rand.NewSource(7))

	const (
		base    = 500_000
		bursts  = 40
		noisePB = 1 // interleaved noise detections per real burst
	)
	trafficQualified, noiseQualified := 0, 0
	for i := 0; i < bursts; i++ {
		real := base + i*dmSlotDibits
		// A noise hit somewhere inside this slot, off the traffic residue.
		off := 8 + rng.Intn(dmSlotDibits-16)
		if residueNear(real+off, residueOf(base)) {
			off += dmGridTolerance + 2
		}
		for _, l := range []int{real, real + off} {
			ok := g.Observe(l)
			if l == real {
				if ok {
					trafficQualified++
				}
			} else if ok {
				noiseQualified++
			}
		}
	}
	if noiseQualified != 0 {
		t.Errorf("qualified %d off-grid noise detections, want 0", noiseQualified)
	}
	if want := bursts - (dmGridMinTrain - 1); trafficQualified != want {
		t.Errorf("qualified %d of %d on-grid bursts, want %d", trafficQualified, bursts, want)
	}
}

// TestDMSlotGridResetRelearns pins that after a Reset (the pipeline's traffic
// drought) a second transmission on a DIFFERENT residue latches afresh — the
// next PTT must grant even though it does not share the previous grid.
func TestDMSlotGridResetRelearns(t *testing.T) {
	g := NewDMSlotGrid()
	for i := 0; i < 10; i++ {
		g.Observe(1_000 + i*dmSlotDibits)
	}
	if !g.Latched() {
		t.Fatalf("first transmission did not latch")
	}
	first := g.residue

	g.Reset()
	if g.Latched() {
		t.Fatalf("Reset left the grid latched")
	}
	// Second transmission, deliberately off the first residue.
	base := 900_000 + first + dmSlotDibits/2
	qualified := 0
	for i := 0; i < 10; i++ {
		if g.Observe(base + i*dmSlotDibits) {
			qualified++
		}
	}
	if !g.Latched() {
		t.Fatalf("second transmission did not re-latch after Reset")
	}
	if g.residue == first {
		t.Errorf("re-latched the stale residue %d", first)
	}
	if want := 10 - (dmGridMinTrain - 1); qualified != want {
		t.Errorf("qualified %d of 10 second-transmission bursts, want %d", qualified, want)
	}
}

// TestDMSlotGridReanchorDropsStaleLatch covers the receiver Reset case, where the
// absolute dibit index restarts at 0 while a residue is still latched: the grid
// must re-learn rather than reject every burst of the new stream forever.
func TestDMSlotGridReanchorDropsStaleLatch(t *testing.T) {
	g := NewDMSlotGrid()
	for i := 0; i < 10; i++ {
		g.Observe(5_000_000 + i*dmSlotDibits)
	}
	if !g.Latched() {
		t.Fatalf("did not latch before the re-anchor")
	}
	// Stream restarts near zero on a different residue.
	base := 37
	qualified := 0
	for i := 0; i < 12; i++ {
		if g.Observe(base + i*dmSlotDibits) {
			qualified++
		}
	}
	if !g.Latched() {
		t.Fatalf("did not re-latch after the index re-anchored")
	}
	if qualified == 0 {
		t.Errorf("no burst qualified after the index re-anchored (stale latch stuck)")
	}
}

// TestDMSlotGridBounded pins the ring bound so a long transmission cannot grow
// the lead buffer without limit.
func TestDMSlotGridBounded(t *testing.T) {
	g := NewDMSlotGrid()
	for i := 0; i < 10_000; i++ {
		g.Observe(i * dmSlotDibits)
	}
	if len(g.leads) > dmGridMaxLeads {
		t.Errorf("lead ring grew to %d, want <= %d", len(g.leads), dmGridMaxLeads)
	}
}

// TestDMSlotGridDropsLatchWhenTrainEnds is the subtle half of the noise gate. Once a
// residue is latched, the ~18/s correlator false alarms still land on it about
// 18·3/255 ≈ 0.2 times a second. Without a train-gap check those stray hits qualify
// forever after the transmission has ended, which keeps the caller's traffic drought
// from ever firing — exactly the latch that stopped the operator's second PTT from
// granting. A hit arriving long after the train must NOT qualify.
func TestDMSlotGridDropsLatchWhenTrainEnds(t *testing.T) {
	g := NewDMSlotGrid()
	const base = 200_000
	for i := 0; i < 12; i++ {
		g.Observe(base + i*dmSlotDibits)
	}
	if !g.Latched() {
		t.Fatalf("did not latch")
	}
	last, _ := g.LastQualifiedLead()

	// A lone stray hit on the latched residue, a long silence later.
	stray := last + dmGridTrainGap + 5*dmSlotDibits
	if g.Observe(stray) {
		t.Errorf("stray on-residue noise hit %d qualified after the train ended", stray)
	}
	if g.Latched() {
		t.Errorf("latch survived a %d-dibit gap", stray-last)
	}

	// A gap INSIDE the tolerance is still the same transmission.
	g2 := NewDMSlotGrid()
	for i := 0; i < 12; i++ {
		g2.Observe(base + i*dmSlotDibits)
	}
	l2, _ := g2.LastQualifiedLead()
	if !g2.Observe(l2 + 4*dmSlotDibits) {
		t.Errorf("a 4-slot intra-train gap broke the latch")
	}
}
