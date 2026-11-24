package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestSDRGroupSeamReplay is the phase-2 feasibility harness for coherent
// SDR-group stitching (docs/design/coherent-sdr-stitching.md). Point it at
// the plan file a group baseband recording writes:
//
//	GT_SDR_GROUP_PLAN=<dir>/<group>_plan_<UTC>.json \
//	  go test ./cmd/gophertrunk -run TestSDRGroupSeamReplay -v
//
// It loads each member's recording from the same directory (the file whose
// timestamp is nearest the plan's), cuts every seam's overlap band out of the
// two neighbours, and reports their relative delay, phase and coherence over
// time with a verdict: independent clocks (stitching impossible), locked
// clocks with a walking phase, or fixed alignment (stitching feasible).
// GT_SDR_GROUP_SECONDS limits how much of each file is read (default 30).
// The overlap needs a common signal; a quiet seam says so.
func TestSDRGroupSeamReplay(t *testing.T) {
	planPath := os.Getenv("GT_SDR_GROUP_PLAN")
	if planPath == "" {
		t.Skip("set GT_SDR_GROUP_PLAN to a group recording's <group>_plan_<UTC>.json")
	}
	body, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan sdrGroupPlanFile
	if err := json.Unmarshal(body, &plan); err != nil {
		t.Fatalf("plan: %v", err)
	}
	seconds := 30.0
	if v, err := strconv.ParseFloat(os.Getenv("GT_SDR_GROUP_SECONDS"), 64); err == nil && v > 0 {
		seconds = v
	}
	dir := filepath.Dir(planPath)
	iq := make([][]complex64, len(plan.Members))
	var rate uint32
	for i, m := range plan.Members {
		path := memberRecording(t, dir, m.Serial, plan.WrittenAt)
		samples, r, err := siglab.DecodeContainerFile(path)
		if err != nil {
			t.Fatalf("member %s: %v", m.Serial, err)
		}
		if rate != 0 && r != rate {
			t.Fatalf("member %s recorded at %d S/s, others at %d", m.Serial, r, rate)
		}
		rate = r
		if limit := int(seconds * float64(r)); len(samples) > limit {
			samples = samples[:limit]
		}
		iq[i] = samples
		t.Logf("member %d %s: %s, %.1f s at %d S/s, centre %.4f MHz", i, m.Serial, filepath.Base(path),
			float64(len(samples))/float64(r), r, float64(m.CenterFreqHz)/1e6)
	}
	for i := 0; i+1 < len(plan.Members); i++ {
		lo, hi := plan.Members[i], plan.Members[i+1]
		if lo.HighHz <= hi.LowHz {
			t.Logf("seam %s|%s: the windows do not overlap, nothing to measure", lo.Serial, hi.Serial)
			continue
		}
		f0 := (float64(hi.LowHz) + float64(lo.HighHz)) / 2
		rep := analyzeSeam(iq[i], iq[i+1], float64(rate), float64(lo.CenterFreqHz), float64(hi.CenterFreqHz), f0)
		rep.Lower, rep.Upper = lo.Serial, hi.Serial
		t.Logf("seam %s|%s at %.4f MHz (%.0f S/s): %d windows, median coherence %.2f, clock %.3f ppm, phase %.2f °/s",
			rep.Lower, rep.Upper, f0/1e6, rep.RateHz, len(rep.Windows), rep.MedianCoherence, rep.LagDriftPPM, rep.PhaseDriftDegPerS)
		step := max(1, len(rep.Windows)/20)
		for k := 0; k < len(rep.Windows); k += step {
			w := rep.Windows[k]
			t.Logf("  t=%6.2fs lag=%10.3f phase=%7.1f° coherence=%.2f", w.T, w.LagSamples, w.PhaseDeg, w.Coherence)
		}
		t.Logf("VERDICT seam %s|%s: %s", rep.Lower, rep.Upper, rep.Verdict)
	}
}

// memberRecording finds the member's recording in dir: <serial>_<UTC>.wav or
// .flac, the one whose timestamp is nearest the plan's.
func memberRecording(t *testing.T, dir, serial string, planAt time.Time) string {
	t.Helper()
	var cands []string
	for _, ext := range []string{"wav", "flac"} {
		m, _ := filepath.Glob(filepath.Join(dir, recordingSerialPrefix(serial)+"_*."+ext))
		cands = append(cands, m...)
	}
	if len(cands) == 0 {
		t.Fatalf("no recording for member %s in %s", serial, dir)
	}
	stamp := func(p string) time.Time {
		base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		ts, err := time.Parse("20060102T150405Z", base[strings.LastIndex(base, "_")+1:])
		if err != nil {
			return time.Time{}
		}
		return ts
	}
	sort.Slice(cands, func(a, b int) bool {
		return math.Abs(stamp(cands[a]).Sub(planAt).Seconds()) < math.Abs(stamp(cands[b]).Sub(planAt).Seconds())
	})
	return cands[0]
}

// recordingSerialPrefix mirrors the baseband recorder's file-name sanitizer.
func recordingSerialPrefix(serial string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, strings.TrimSpace(serial))
}
