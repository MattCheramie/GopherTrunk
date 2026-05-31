package main

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// buildFSWStream plants `hits` canonical frame-sync words (rotation 0) into a
// dibit stream with random filler between them, and an index-aligned soft
// stream where each FSW outer symbol is drawn from its nominal level (+3 →
// +0.37, −3 → −0.24) plus Gaussian noise. Post-transition FSW positions (where
// the symbol differs from the previous) use transNoise; steady positions use
// baseNoise — so the diagnostic can be checked against a known mechanism.
func buildFSWStream(rng *rand.Rand, hits int, baseNoise, transNoise float64) (d iqDiag, positions []int) {
	for h := 0; h < hits; h++ {
		positions = append(positions, len(d.dibits))
		for kk := 0; kk < 24; kk++ {
			d.dibits = append(d.dibits, p25phase1.FrameSyncWord[kk])
			level := 0.37
			if p25phase1.FrameSyncWord[kk] == 3 { // dibit 3 → −3
				level = -0.24
			}
			noise := baseNoise
			if kk > 0 && p25phase1.FrameSyncWord[kk] != p25phase1.FrameSyncWord[kk-1] {
				noise = transNoise
			}
			d.soft = append(d.soft, float32(level+rng.NormFloat64()*noise))
		}
		for f := 0; f < 50; f++ { // filler
			d.dibits = append(d.dibits, uint8(rng.Intn(4)))
			d.soft = append(d.soft, float32(rng.NormFloat64()*0.1))
		}
	}
	return d, positions
}

func stdOf(v []float64) float64 { _, s, _, _, _ := distStats(v); return s }

func TestTrueSymbolEyeFlagsTransitionDrivenSpread(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// ISI / timing signature: only the post-transition positions are noisy.
	d, positions := buildFSWStream(rng, 60, 0.03, 0.22)

	_, steady, trans, ok := d.trueSymbolEye(0, positions)
	if !ok {
		t.Fatal("trueSymbolEye returned ok=false on a well-formed stream")
	}
	for b, label := range []string{"+3", "-3"} {
		ss, ts := stdOf(steady[b]), stdOf(trans[b])
		t.Logf("%s: steady std=%.4f  post-transition std=%.4f  ratio=%.2f", label, ss, ts, ratioOrInf(ts, ss))
		if ratioOrInf(ts, ss) < 1.5 {
			t.Errorf("%s: post-transition std=%.4f not ≥1.5× steady std=%.4f — transition signature missed", label, ts, ss)
		}
	}

	var buf bytes.Buffer
	d.printTrueSymbolEye(&buf, 0, positions)
	if !strings.Contains(buf.String(), "ISI / symbol-timing") {
		t.Errorf("verdict did not flag ISI/timing; got:\n%s", buf.String())
	}
}

func TestTrueSymbolEyeFlagsAmplitudeIntrinsicSpread(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	// Amplitude-intrinsic signature: uniform noise on all outer symbols,
	// symmetric (no skew), independent of transitions.
	d, positions := buildFSWStream(rng, 60, 0.15, 0.15)

	_, steady, trans, ok := d.trueSymbolEye(0, positions)
	if !ok {
		t.Fatal("trueSymbolEye returned ok=false")
	}
	for b, label := range []string{"+3", "-3"} {
		ss, ts := stdOf(steady[b]), stdOf(trans[b])
		t.Logf("%s: steady std=%.4f  post-transition std=%.4f  ratio=%.2f", label, ss, ts, ratioOrInf(ts, ss))
		if r := ratioOrInf(ts, ss); r > 1.4 {
			t.Errorf("%s: transition ratio=%.2f too high for uniform-noise eye (should be ~1)", label, r)
		}
	}

	var buf bytes.Buffer
	d.printTrueSymbolEye(&buf, 0, positions)
	if !strings.Contains(buf.String(), "amplitude-intrinsic") {
		t.Errorf("verdict did not flag amplitude-intrinsic; got:\n%s", buf.String())
	}
}

func TestTrueSymbolEyeRecoversKnownLevels(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	d, positions := buildFSWStream(rng, 60, 0.05, 0.05)
	all, _, _, ok := d.trueSymbolEye(0, positions)
	if !ok {
		t.Fatal("ok=false")
	}
	mP, _, _, _, _ := distStats(all[0]) // +3
	mN, _, _, _, _ := distStats(all[1]) // −3
	if mP < 0.34 || mP > 0.40 {
		t.Errorf("true +3 mean=%.4f, want ≈0.37", mP)
	}
	if mN < -0.27 || mN > -0.21 {
		t.Errorf("true −3 mean=%.4f, want ≈-0.24", mN)
	}
}
