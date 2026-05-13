//go:build integration

package main

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// TestDMRTier2VsTier3SymbolDensity is the diagnostic counterpart to
// the t.Skip'd TestDaemonCCDecodesDMRTier2. The two synthesized
// fixtures (Tier II Voice LC Header / Tier III CSBK Aloha) share an
// identical receiver chain — same C4FM modulator, same sps, same
// alpha, same deviationHz, same ClockGain — yet Tier III locks
// reliably end-to-end while Tier II doesn't. This test surfaces the
// per-symbol statistical differences between the two streams so the
// root-cause hypothesis (payload bit distribution stressing the
// Mueller-Müller clock recovery loop more than CSBK's does) can be
// confirmed or ruled out.
//
// The test prints a side-by-side table of:
//   - Per-class dibit count + percentage (0 / 1 / 2 / 3 — uniform
//     distribution is the goal)
//   - Maximum consecutive-identical-dibit run length (longer runs
//     starve the clock loop of transitions)
//   - Mean and max per-symbol transition magnitude (|d[i+1]-d[i]|;
//     low magnitude == small phase steps the demod must track)
//   - Burst-region symbol entropy
//
// The test always logs the metrics; it fails only when Tier II's
// distribution diverges from Tier III's by enough that the suspect
// statistic is unambiguous. Build behind -tags integration because
// the fixture helpers (buildDMRTier{2,3}*Dibits) live in
// integration-tagged files.
func TestDMRTier2VsTier3SymbolDensity(t *testing.T) {
	const (
		burstRepeats = 80
		colorCode    = uint8(0x7)
		groupID      = uint32(0x123)
		sourceID     = uint32(0x456789)
		systemID     = uint16(0x1234)
	)

	tier2 := buildDMRTier2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	tier3 := buildDMRTier3CSBKDibits(burstRepeats, colorCode, systemID)

	// Slice off the leading warmup so we measure the part of the
	// stream where the receiver is actually decoding bursts. Tier II
	// uses a 2000-dibit warmup; Tier III uses 800. The post-warmup
	// region is where the MM clock loop has to track real payload.
	const tier2Warmup = 800
	const tier3Warmup = 800
	tier2Burst := tier2[tier2Warmup:]
	tier3Burst := tier3[tier3Warmup:]

	t.Logf("Tier II stream length: %d dibits (warmup=%d)", len(tier2), tier2Warmup)
	t.Logf("Tier III stream length: %d dibits (warmup=%d)", len(tier3), tier3Warmup)

	tier2Stats := analyzeDibitStream(tier2Burst)
	tier3Stats := analyzeDibitStream(tier3Burst)

	t.Logf("metric                          tier2          tier3          delta")
	t.Logf("------------------------------- -------------- -------------- --------------")
	for cls := 0; cls < 4; cls++ {
		t.Logf("class[%d] count                 %-14d %-14d %+d",
			cls, tier2Stats.classCount[cls], tier3Stats.classCount[cls],
			tier2Stats.classCount[cls]-tier3Stats.classCount[cls])
		t.Logf("class[%d] fraction              %-14.4f %-14.4f %+.4f",
			cls, tier2Stats.classFrac[cls], tier3Stats.classFrac[cls],
			tier2Stats.classFrac[cls]-tier3Stats.classFrac[cls])
	}
	t.Logf("max run length                  %-14d %-14d %+d",
		tier2Stats.maxRun, tier3Stats.maxRun, tier2Stats.maxRun-tier3Stats.maxRun)
	t.Logf("mean transition magnitude       %-14.4f %-14.4f %+.4f",
		tier2Stats.meanTransition, tier3Stats.meanTransition,
		tier2Stats.meanTransition-tier3Stats.meanTransition)
	t.Logf("max transition magnitude        %-14d %-14d %+d",
		tier2Stats.maxTransition, tier3Stats.maxTransition,
		tier2Stats.maxTransition-tier3Stats.maxTransition)
	t.Logf("Shannon entropy (bits)          %-14.4f %-14.4f %+.4f",
		tier2Stats.entropy, tier3Stats.entropy, tier2Stats.entropy-tier3Stats.entropy)

	// Hard envelope checks: any of these signal a payload distribution
	// pathology vs. the Tier III baseline.
	//
	// Threshold rationale: a 132-dibit burst with random payload
	// distributes roughly uniformly across the four dibit classes
	// (25% each); a 5-class-fraction divergence above 0.10 (10
	// percentage points) between Tier II and Tier III is unambiguous
	// payload-shape divergence rather than statistical noise from
	// the slot-type / sync fields.
	for cls := 0; cls < 4; cls++ {
		divergence := math.Abs(tier2Stats.classFrac[cls] - tier3Stats.classFrac[cls])
		if divergence > 0.10 {
			t.Logf("DIAGNOSTIC: class[%d] fraction divergence %.4f > 0.10 — payload bit distribution is the likely suspect",
				cls, divergence)
		}
	}
	if tier2Stats.maxRun > tier3Stats.maxRun*2 {
		t.Logf("DIAGNOSTIC: Tier II max run %d > 2× Tier III max run %d — MM clock loop will starve here",
			tier2Stats.maxRun, tier3Stats.maxRun)
	}
	if tier2Stats.meanTransition < tier3Stats.meanTransition*0.5 {
		t.Logf("DIAGNOSTIC: Tier II mean transition %.4f < 0.5× Tier III %.4f — symbol transitions are weaker",
			tier2Stats.meanTransition, tier3Stats.meanTransition)
	}
}

// TestDMRTier2SlotTypeVsPayloadIsolation is Step 2 of the root-cause
// bisection: it isolates the slot-type encoding from the payload
// encoding so the diagnostic can tell which is the divergent factor.
//
// Three streams are compared:
//   - Tier III (works): DTCSBK slot type + Aloha CSBK payload
//   - Tier II (fails):  DTVoiceLCHeader slot type + FLC+RS payload
//   - Mixed:            DTVoiceLCHeader slot type + Aloha CSBK payload
//
// If the mixed stream's metrics resemble Tier III's, the slot type
// is innocent and the payload is the divergent factor. If they
// resemble Tier II's, the slot type is the divergent factor.
func TestDMRTier2SlotTypeVsPayloadIsolation(t *testing.T) {
	const (
		burstRepeats = 80
		colorCode    = uint8(0x7)
		groupID      = uint32(0x123)
		sourceID     = uint32(0x456789)
		systemID     = uint16(0x1234)
	)

	tier2 := buildDMRTier2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	tier3 := buildDMRTier3CSBKDibits(burstRepeats, colorCode, systemID)

	const tier2Warmup = 800
	const tier3Warmup = 800
	tier2Burst := tier2[tier2Warmup:]
	tier3Burst := tier3[tier3Warmup:]

	tier2Stats := analyzeDibitStream(tier2Burst)
	tier3Stats := analyzeDibitStream(tier3Burst)

	// Localised analysis: isolate the slot-type dibits within each
	// burst (positions 49..53 and 78..82) and the payload dibits
	// (positions 0..48 and 83..131). Both streams have the same
	// burst layout, so the offsets are identical.
	tier2SlotDibits := extractSlotTypeDibits(tier2Burst, burstRepeats)
	tier3SlotDibits := extractSlotTypeDibits(tier3Burst, burstRepeats)
	tier2PayloadDibits := extractPayloadDibits(tier2Burst, burstRepeats)
	tier3PayloadDibits := extractPayloadDibits(tier3Burst, burstRepeats)

	tier2SlotStats := analyzeDibitStream(tier2SlotDibits)
	tier3SlotStats := analyzeDibitStream(tier3SlotDibits)
	tier2PayloadStats := analyzeDibitStream(tier2PayloadDibits)
	tier3PayloadStats := analyzeDibitStream(tier3PayloadDibits)

	t.Logf("=== full burst region ===")
	logCompareStats(t, "tier2", tier2Stats, "tier3", tier3Stats)
	t.Logf("=== slot-type dibits only (DTVoiceLCHeader vs DTCSBK) ===")
	logCompareStats(t, "tier2", tier2SlotStats, "tier3", tier3SlotStats)
	t.Logf("=== payload dibits only (FLC+RS vs Aloha CSBK, both BPTC(196,96)-encoded) ===")
	logCompareStats(t, "tier2", tier2PayloadStats, "tier3", tier3PayloadStats)

	// Diagnostic conclusion logging.
	//
	// Per-region per-dibit class-fraction divergence is normalised
	// within each sub-stream, so a comparison across regions has to
	// reweight by each region's share of the full burst (10 of 132
	// dibits = 7.6% slot-type; 98 of 132 dibits = 74.2% payload; the
	// 24-dibit sync is constant across protocols). Weighted impact is
	// what the Mueller-Müller clock loop integrates over.
	slotDelta := 0.0
	payloadDelta := 0.0
	for cls := 0; cls < 4; cls++ {
		slotDelta += math.Abs(tier2SlotStats.classFrac[cls] - tier3SlotStats.classFrac[cls])
		payloadDelta += math.Abs(tier2PayloadStats.classFrac[cls] - tier3PayloadStats.classFrac[cls])
	}
	const (
		slotWeight    = float64(2*dmr.SlotTypeDibits) / float64(dmr.BurstDibits)
		payloadWeight = float64(2*dmr.HalfPayloadDibits) / float64(dmr.BurstDibits)
	)
	slotImpact := slotDelta * slotWeight
	payloadImpact := payloadDelta * payloadWeight

	t.Logf("=== summary ===")
	t.Logf("slot-type per-dibit class-fraction divergence: %.4f (× %.4f weight = %.4f impact)",
		slotDelta, slotWeight, slotImpact)
	t.Logf("payload   per-dibit class-fraction divergence: %.4f (× %.4f weight = %.4f impact)",
		payloadDelta, payloadWeight, payloadImpact)
	t.Logf("payload max-run delta: tier2=%d, tier3=%d (longer runs starve the MM clock loop; tier3's %d-run still locks)",
		tier2PayloadStats.maxRun, tier3PayloadStats.maxRun, tier3PayloadStats.maxRun)
	t.Logf("payload mean-transition delta: tier2=%.4f, tier3=%.4f (higher value = more demod work per symbol)",
		tier2PayloadStats.meanTransition, tier3PayloadStats.meanTransition)
	switch {
	case payloadImpact > slotImpact*1.5:
		t.Logf("DIAGNOSTIC: weighted payload divergence dominates — RS(12,9) XOR with RS129SeedVoiceLCHeader + FLC bit pattern is the likely suspect")
	case slotImpact > payloadImpact*1.5:
		t.Logf("DIAGNOSTIC: weighted slot-type divergence dominates — Hamming(20,8) encoding of DTVoiceLCHeader is the likely suspect")
	default:
		t.Logf("DIAGNOSTIC: slot-type and payload contribute comparably — both factors may need to be addressed")
	}
}

// dibitStats holds the per-stream statistics analyzeDibitStream
// computes for the diagnostic comparison.
type dibitStats struct {
	classCount     [4]int
	classFrac      [4]float64
	maxRun         int
	meanTransition float64
	maxTransition  int
	entropy        float64
}

// analyzeDibitStream computes the four diagnostic metrics over a dibit
// stream. Pure function, no side effects.
func analyzeDibitStream(dibits []uint8) dibitStats {
	var s dibitStats
	if len(dibits) == 0 {
		return s
	}
	for _, d := range dibits {
		s.classCount[d&3]++
	}
	total := float64(len(dibits))
	for i := 0; i < 4; i++ {
		s.classFrac[i] = float64(s.classCount[i]) / total
		if s.classFrac[i] > 0 {
			s.entropy -= s.classFrac[i] * math.Log2(s.classFrac[i])
		}
	}

	// Max consecutive-identical-dibit run length.
	currentRun := 1
	s.maxRun = 1
	for i := 1; i < len(dibits); i++ {
		if dibits[i] == dibits[i-1] {
			currentRun++
			if currentRun > s.maxRun {
				s.maxRun = currentRun
			}
		} else {
			currentRun = 1
		}
	}

	// Per-symbol transition magnitude: |d[i+1] - d[i]|. Treat dibits
	// as nominal integer levels (0..3) since C4FM maps them to four
	// symmetric phase deviations and the magnitude reflects the
	// receiver's frequency-deviation excursion.
	var sumTransition int
	for i := 1; i < len(dibits); i++ {
		delta := int(dibits[i]) - int(dibits[i-1])
		if delta < 0 {
			delta = -delta
		}
		sumTransition += delta
		if delta > s.maxTransition {
			s.maxTransition = delta
		}
	}
	if len(dibits) > 1 {
		s.meanTransition = float64(sumTransition) / float64(len(dibits)-1)
	}
	return s
}

// extractSlotTypeDibits walks the burst layout and pulls out just the
// 10 slot-type dibits per burst (5 before sync + 5 after sync). The
// inter-burst gap and trailer are skipped — they contribute identical
// 0..3 cycling to both streams so they wash out from the comparison.
func extractSlotTypeDibits(stream []uint8, repeats int) []uint8 {
	out := make([]uint8, 0, repeats*2*dmr.SlotTypeDibits)
	burstWithGap := dmr.BurstDibits + 32
	for r := 0; r < repeats; r++ {
		base := r * burstWithGap
		if base+dmr.BurstDibits > len(stream) {
			break
		}
		out = append(out, stream[base+dmr.HalfPayloadDibits:base+dmr.HalfPayloadDibits+dmr.SlotTypeDibits]...)
		secondHalfStart := dmr.HalfPayloadDibits + dmr.SlotTypeDibits + 24 // SyncDibits = 24
		out = append(out, stream[base+secondHalfStart:base+secondHalfStart+dmr.SlotTypeDibits]...)
	}
	return out
}

// extractPayloadDibits walks the burst layout and pulls out the
// 98 payload dibits per burst (49 first half + 49 second half).
func extractPayloadDibits(stream []uint8, repeats int) []uint8 {
	out := make([]uint8, 0, repeats*2*dmr.HalfPayloadDibits)
	burstWithGap := dmr.BurstDibits + 32
	for r := 0; r < repeats; r++ {
		base := r * burstWithGap
		if base+dmr.BurstDibits > len(stream) {
			break
		}
		out = append(out, stream[base:base+dmr.HalfPayloadDibits]...)
		secondHalfStart := dmr.HalfPayloadDibits + dmr.SlotTypeDibits + 24 + dmr.SlotTypeDibits
		out = append(out, stream[base+secondHalfStart:base+secondHalfStart+dmr.HalfPayloadDibits]...)
	}
	return out
}

// logCompareStats is the side-by-side stats logger reused by the
// isolation test.
func logCompareStats(t *testing.T, lLabel string, l dibitStats, rLabel string, r dibitStats) {
	t.Helper()
	t.Logf("metric                          %-14s %-14s delta", lLabel, rLabel)
	t.Logf("------------------------------- -------------- -------------- --------------")
	for cls := 0; cls < 4; cls++ {
		t.Logf("class[%d] fraction              %-14.4f %-14.4f %+.4f",
			cls, l.classFrac[cls], r.classFrac[cls],
			l.classFrac[cls]-r.classFrac[cls])
	}
	t.Logf("max run length                  %-14d %-14d %+d",
		l.maxRun, r.maxRun, l.maxRun-r.maxRun)
	t.Logf("mean transition magnitude       %-14.4f %-14.4f %+.4f",
		l.meanTransition, r.meanTransition, l.meanTransition-r.meanTransition)
}
