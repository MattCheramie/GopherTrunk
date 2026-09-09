package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"math/cmplx"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/diversity"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// diversityCaptureMeta mirrors the sidecar the driver's branch recorder writes
// (internal/sdr/soapyremote/branchcapture.go). Duplicated rather than exported
// because it is a capture-provenance record read by exactly one harness.
type diversityCaptureMeta struct {
	Addr             string   `json:"addr"`
	SampleRateHz     float64  `json:"sample_rate_hz"`
	CenterFreqHz     uint32   `json:"center_freq_hz"`
	Format           string   `json:"format"`
	DiversityMode    string   `json:"diversity_mode"`
	Branches         int      `json:"branches"`
	BranchFiles      []string `json:"branch_files"`
	SamplesPerBranch int64    `json:"samples_per_branch"`
	DroppedDatagrams int64    `json:"dropped_datagrams"`
}

// TestDiversityCombinerReplay is the offline A/B for MRC diversity, and the
// verification gate for making tracking the default.
//
// Skip unless GT_DIVERSITY_CAPTURE points at a <prefix>.diversity.json written
// by `sdr.soapy_remote[].diversity_capture` — the ONLY tap that sees the
// per-branch IQ before it is combined. Every other capture in GopherTrunk is
// post-combine and cannot answer this question.
//
// It reports two things.
//
//  1. A WINDOWED TRACE of coherence, branch gain and branch PHASE. This is the
//     primary deliverable: it says, quantitatively, whether the relative phase
//     of the two receivers is constant or walking, which is the whole basis for
//     preferring a tracking combiner over a frozen constant. Flat phase means
//     the frozen constant was fine on this hardware and mrc-static is correct;
//     walking phase gives the drift rate in degrees per second.
//
//  2. EIGHT DECODE ARMS through identical downstream wiring — each branch
//     alone, wideband static / tracking / blind-IRC combines, per-channel
//     narrowband static / tracking combines, and a narrowband single-branch
//     baseline — scored by CRC-clean BSCH count. Decode yield is the verdict,
//     never EVM: a combiner can improve a constellation's look while decoding
//     nothing, which this repo has measured before. The tracking arms pass the
//     driver-derived alpha EXPLICITLY: TrackingOptions{Alpha: 0} means
//     one-shot, so an omitted alpha silently measures static (which is what
//     the 17/18 Aug 2026 A/B "tracking" verdicts actually did).
//
// GT_DIVERSITY_TUNE_HZ offsets the down-conversion onto the control channel.
// Assertions are deliberately weak — this is a measurement instrument, and the
// operator's own numbers are what decide whether the default is justified.
func TestDiversityCombinerReplay(t *testing.T) {
	br0, br1, meta := loadDiversityCapture(t)
	t.Logf("capture: %s  rate=%.0f Hz  samples/branch=%d  dropped_datagrams=%d  mode=%s",
		meta.Addr, meta.SampleRateHz, len(br0), meta.DroppedDatagrams, meta.DiversityMode)

	if meta.DroppedDatagrams > 0 {
		t.Logf("NOTE: %d datagrams were dropped from BOTH branches to keep them aligned; "+
			"the branches remain sample-synchronous", meta.DroppedDatagrams)
	}

	// ---- 1. Windowed coherence / gain / phase trace -------------------------
	const windowMs = 100.0
	win := int(meta.SampleRateHz * windowMs / 1000)
	if win < 4096 {
		win = 4096
	}
	var trace []divWindow
	for pos := 0; pos+win <= len(br0); pos += win {
		var s diversity.CrossStats
		s.Accumulate(br0[pos:pos+win], br1[pos:pos+win])
		h, ok := s.Gain()
		if !ok {
			continue
		}
		mag := cmplx.Abs(complex128(h))
		gdb := math.Inf(-1)
		if mag > 0 {
			gdb = 20 * math.Log10(mag)
		}
		trace = append(trace, divWindow{
			rho:      s.Coherence(),
			gainDb:   gdb,
			phaseDeg: cmplx.Phase(complex128(h)) * 180 / math.Pi,
			tSec:     float64(pos) / meta.SampleRateHz,
		})
	}
	if len(trace) == 0 {
		t.Fatal("capture too short for even one analysis window")
	}

	rhos := make([]float64, len(trace))
	for i, s := range trace {
		rhos[i] = s.rho
	}
	t.Logf("coherence over %d x %.0f ms windows: min=%.3f median=%.3f max=%.3f",
		len(trace), windowMs, minOf(rhos), medianOf(rhos), maxOf(rhos))

	// Unwrap the phase so a drift reads as a monotone walk rather than wrapping.
	unwrapped := make([]float64, len(trace))
	unwrapped[0] = trace[0].phaseDeg
	for i := 1; i < len(trace); i++ {
		d := trace[i].phaseDeg - trace[i-1].phaseDeg
		for d > 180 {
			d -= 360
		}
		for d < -180 {
			d += 360
		}
		unwrapped[i] = unwrapped[i-1] + d
	}
	span := unwrapped[len(unwrapped)-1] - unwrapped[0]
	dur := trace[len(trace)-1].tSec - trace[0].tSec
	rate := 0.0
	if dur > 0 {
		rate = span / dur
	}
	t.Logf("branch phase: start=%.1f deg  end=%.1f deg  span=%.1f deg over %.2f s  (%.2f deg/s)",
		unwrapped[0], unwrapped[len(unwrapped)-1], span, dur, rate)
	t.Logf("branch gain: min=%.2f dB median=%.2f dB max=%.2f dB",
		minOf(gainsOf(trace)), medianOf(gainsOf(trace)), maxOf(gainsOf(trace)))
	if math.Abs(span) < 5 {
		t.Logf("VERDICT: branch phase is essentially CONSTANT on this capture — a frozen " +
			"calibration is sufficient here and diversity: mrc-static should measure the same.")
	} else {
		t.Logf("VERDICT: branch phase WALKS by %.1f deg — a frozen calibration decays on this "+
			"hardware and tracking is doing real work.", span)
	}

	tuneHz := 0.0
	if v := os.Getenv("GT_DIVERSITY_TUNE_HZ"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_DIVERSITY_TUNE_HZ: %v", err)
		}
		tuneHz = f
	}

	// ---- 1b. The same trace AFTER the per-channel DDC ------------------------
	//
	// The combiner in the driver runs on the WIDEBAND stream, so one complex
	// scalar has to fit every carrier in the span at once. That is exact only
	// if the branches differ by a frequency-flat constant; two antennas metres
	// apart give each carrier its own phase difference set by geometry and
	// direction of arrival (mrc.go's KNOWN LIMITATION). This is the measurement
	// that says whether that is what is happening on THIS hardware: if
	// narrowband coherence is high where wideband coherence is low, the
	// wideband scalar cannot be rescued by any amount of tracking and the fix
	// is combining after the DDC, one gain per channel.
	nb0 := downconvert(br0, meta.SampleRateHz, tuneHz)
	nb1 := downconvert(br1, meta.SampleRateHz, tuneHz)
	nbRate := 144_000.0
	nbWin := int(nbRate * windowMs / 1000)
	if nbWin < 1024 {
		nbWin = 1024
	}
	var nbRhos []float64
	for pos := 0; pos+nbWin <= len(nb0); pos += nbWin {
		var s diversity.CrossStats
		s.Accumulate(nb0[pos:pos+nbWin], nb1[pos:pos+nbWin])
		nbRhos = append(nbRhos, s.Coherence())
	}
	if len(nbRhos) > 0 {
		wbMed := medianOf(rhos)
		nbMed := medianOf(nbRhos)
		t.Logf("narrowband (post-DDC, %.0f kHz) coherence over %d x %.0f ms windows: "+
			"min=%.3f median=%.3f max=%.3f", nbRate/1000, len(nbRhos), windowMs,
			minOf(nbRhos), nbMed, maxOf(nbRhos))
		switch {
		case nbMed > wbMed+0.2:
			t.Logf("VERDICT: narrowband coherence (%.3f) is well above wideband (%.3f) — "+
				"the branches DO agree on the target channel and the wideband scalar is "+
				"the limitation. Per-channel (post-DDC) combining is the fix; compare the "+
				"nb-* arms below.", nbMed, wbMed)
		case nbMed < 0.2:
			t.Logf("VERDICT: narrowband coherence is %.3f — the two receivers barely agree "+
				"even on the target channel, so no combiner will help. Check antennas, "+
				"gain staging and that both branches are actually receiving.", nbMed)
		default:
			t.Logf("VERDICT: narrowband coherence (%.3f) is close to wideband (%.3f) — "+
				"the wideband scalar is not the bottleneck on this capture.", nbMed, wbMed)
		}
	}

	// ---- 2. Decode arms -----------------------------------------------------

	window := int(meta.SampleRateHz * 2 / 1000) // the driver's 2 ms window
	if window < 4096 {
		window = 4096
	}
	// A narrowband window has to be longer in SAMPLES to hold the same estimate
	// quality: the driver's 2 ms is 288 samples at 144 kHz, far too few for a
	// trustworthy phase estimate. 20 ms of channel-rate stream is 2880.
	nbWindow := int(nbRate * 20 / 1000)

	// Mirror the driver's reference selection: the live combiner anchors on the
	// loudest branch, and the anchor decides both the pre-lock passthrough and
	// the least-squares gain's noise bias. The arms used to anchor on branch 0
	// by file order — on the 18 Aug 2026 captures, whose branch 0 was the weak
	// receiver, every combined arm degenerated to the WEAK branch's passthrough
	// whenever the calibrator held, and the arm scores said nothing about what
	// the driver would actually do.
	wbRef, wbOther, refName := br0, br1, "branch0"
	if meanPowerOf(br1) > meanPowerOf(br0) {
		wbRef, wbOther, refName = br1, br0, "branch1"
	}
	nbRef, nbOther := nb0, nb1
	if meanPowerOf(nb1) > meanPowerOf(nb0) {
		nbRef, nbOther = nb1, nb0
	}
	t.Logf("combined arms anchor on %s (the louder branch), mirroring the driver's reference selection", refName)
	t.Logf("driver-equivalent coherence gates at this rate (%d-sample windows): lock >= %.3f, track >= %.3f",
		window, diversity.TrackingOptions{}.LockGate(window), diversity.TrackingOptions{}.TrackGate(window))

	// The tracking arms MUST pass the driver-derived alpha explicitly: a
	// TrackingOptions with Alpha left at zero is ONE-SHOT (withDefaults keeps 0
	// as the mrc-static freeze), so an "Alpha omitted" tracking arm silently
	// measures static — which is exactly what the 17/18 Aug A/B verdicts did.
	// Mirror of soapyremote's mrcTrackAlpha: actual window duration over the
	// 200 ms tau.
	const trackTauMs = 200.0
	wbAlpha := float64(window) / meta.SampleRateHz * 1000 / trackTauMs
	nbAlpha := float64(nbWindow) / nbRate * 1000 / trackTauMs
	if wbAlpha <= 0 || nbAlpha <= 0 {
		t.Fatalf("derived tracking alphas wb=%g nb=%g — zero would silently freeze the tracking arms into static", wbAlpha, nbAlpha)
	}
	if wbAlpha > 1 {
		wbAlpha = 1
	}
	if nbAlpha > 1 {
		nbAlpha = 1
	}
	t.Logf("tracking-arm alphas (tau %.0f ms): wb=%.4f nb=%.4f", trackTauMs, wbAlpha, nbAlpha)

	// ---- Inter-branch timing skew ------------------------------------------
	// A scalar MRC weight assumes the branches are TIME-ALIGNED. A constant
	// inter-branch delay (start-of-stream skew, differing DDC group delay)
	// makes the sum a comb filter: band-average power still looks fine, the
	// per-frequency coherence is high, but the broadband |rho| is diluted and
	// the combined SYMBOLS are smeared — on the 19 Aug 2026 X310 capture a
	// 2.65-sample skew cost every combined arm ~22% of its CRC-clean BSCH
	// versus the best branch alone. Measure it, report it, and run an aligned
	// arm so the recoverable gain is a number instead of a suspicion.
	lag, lagRho := interBranchLagSamples(wbRef, wbOther)
	frac := interBranchFractionalDelay(wbRef, wbOther, lag, meta.SampleRateHz)
	t.Logf("inter-branch delay: other lags ref by %d%+.2f samples (peak |rho|=%.3f at the integer lag) — "+
		"a scalar combiner cannot represent a delay; wb-aligned-static shows what alignment recovers",
		lag, frac, lagRho)
	wbOtherAligned := shiftBranchFractional(wbOther, lag, frac)

	// Wideband arms are combined then down-converted; narrowband arms are
	// down-converted per branch and combined at the channel rate. Both are
	// scored by the identical decoder so the combiner is the only variable.
	arms := []struct {
		name string
		iq   []complex64
		rate float64
		tune float64
	}{
		{"branch0-only", br0, meta.SampleRateHz, tuneHz},
		{"branch1-only", br1, meta.SampleRateHz, tuneHz},
		{"wb-static", combineWith(t, wbRef, wbOther,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: 0})),
			meta.SampleRateHz, tuneHz},
		{"wb-tracking", combineWith(t, wbRef, wbOther,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: wbAlpha})),
			meta.SampleRateHz, tuneHz},
		// Blind IRC on the wideband stream. Expected to measure the same as
		// wb-tracking: without a training sequence the channel estimate is a
		// power-weighted blend of every signal in the span, so the null is
		// steered at a mixture (see internal/dsp/diversity/irc.go). Carried as
		// an arm so that claim is checked against real air rather than assumed.
		{"wb-irc-blind", combineWith(t, wbRef, wbOther,
			diversity.NewIRCCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: wbAlpha})),
			meta.SampleRateHz, tuneHz},
		// Static combine with the measured inter-branch delay removed first.
		// The one new variable versus wb-static is the alignment, so the score
		// difference IS the cost of the skew.
		{"wb-aligned-static", combineWith(t, wbRef, wbOtherAligned,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: 0})),
			meta.SampleRateHz, tuneHz},
		// The arms that matter for the wideband-limitation question: one gain
		// per narrowband channel instead of one for the whole span.
		{"nb-static", combineWith(t, nbRef, nbOther,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: nbWindow, Alpha: 0})),
			nbRate, 0},
		{"nb-tracking", combineWith(t, nbRef, nbOther,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: nbWindow, Alpha: nbAlpha})),
			nbRate, 0},
		{"nb-branch0-only", nb0, nbRate, 0},
	}

	type result struct {
		name    string
		ok, bad int64
	}
	var results []result
	for _, arm := range arms {
		ok, bad := decodeTETRABSCH(t, arm.iq, arm.rate, arm.tune)
		results = append(results, result{arm.name, ok, bad})
		t.Logf("%-18s BSCH ok=%d fail=%d", arm.name, ok, bad)
	}

	best := results[0]
	for _, r := range results[1:] {
		if r.ok > best.ok {
			best = r
		}
	}
	t.Logf("best arm: %s (BSCH ok=%d)", best.name, best.ok)
	t.Log("CRC-clean BSCH count is the verdict. Do NOT conclude anything from EVM or " +
		"constellation appearance: a combiner can tidy a constellation while decoding nothing.")

	// Weak assertion only: this is an instrument, not a gate.
	if best.ok == 0 {
		t.Log("no arm decoded any BSCH — check GT_DIVERSITY_TUNE_HZ points at the control channel")
	}
}

// combineWith runs the two branches through a calibrator exactly as the driver
// does — Observe then Combine, in windows — and returns the combined stream.
// windowedCalibrator is the shape every combiner in this package shares, so an
// arm can be swapped without the harness caring which one it is.
type windowedCalibrator interface {
	Observe(branches [][]complex64) diversity.ObserveResult
	Combine(branches [][]complex64) ([]complex64, error)
}

// interBranchLagSamples scans integer lags for the delay of `other` relative
// to `ref` (positive = other lags), returning the |rho|-maximising lag. Uses
// the first ~5 s so a 30 s capture stays cheap; a start-of-stream or DDC group
// delay skew is constant, so any span measures it.
func interBranchLagSamples(ref, other []complex64) (bestLag int, bestRho float64) {
	n := len(ref)
	if len(other) < n {
		n = len(other)
	}
	if n > 1<<20 {
		n = 1 << 20
	}
	r, o := ref[:n], other[:n]
	var pr, po float64
	for i := 0; i < n; i++ {
		pr += float64(real(r[i]))*float64(real(r[i])) + float64(imag(r[i]))*float64(imag(r[i]))
		po += float64(real(o[i]))*float64(real(o[i])) + float64(imag(o[i]))*float64(imag(o[i]))
	}
	norm := math.Sqrt(pr * po)
	if norm == 0 {
		return 0, 0
	}
	const maxLag = 16
	for lag := -maxLag; lag <= maxLag; lag++ {
		var cr, ci float64
		for i := 0; i < n-maxLag; i++ {
			var a, b complex64
			if lag >= 0 {
				a, b = r[i], o[i+lag]
			} else {
				a, b = r[i-lag], o[i]
			}
			// conj(a)·b
			cr += float64(real(a))*float64(real(b)) + float64(imag(a))*float64(imag(b))
			ci += float64(real(a))*float64(imag(b)) - float64(imag(a))*float64(real(b))
		}
		if rho := math.Hypot(cr, ci) / norm; rho > bestRho {
			bestRho, bestLag = rho, lag
		}
	}
	return bestLag, bestRho
}

// interBranchFractionalDelay refines the integer lag with the residual
// fractional delay, estimated from the cross-correlation magnitudes at the
// neighbouring lags via parabolic interpolation. rateHz is unused beyond
// documentation; the return is in samples, in (-1, 1).
func interBranchFractionalDelay(ref, other []complex64, lag int, _ float64) float64 {
	rhoAt := func(l int) float64 {
		n := len(ref)
		if len(other) < n {
			n = len(other)
		}
		if n > 1<<20 {
			n = 1 << 20
		}
		var cr, ci float64
		const guard = 20
		for i := 0; i < n-guard; i++ {
			var a, b complex64
			if l >= 0 {
				a, b = ref[i], other[i+l]
			} else {
				a, b = ref[i-l], other[i]
			}
			cr += float64(real(a))*float64(real(b)) + float64(imag(a))*float64(imag(b))
			ci += float64(real(a))*float64(imag(b)) - float64(imag(a))*float64(real(b))
		}
		return math.Hypot(cr, ci)
	}
	ym, y0, yp := rhoAt(lag-1), rhoAt(lag), rhoAt(lag+1)
	den := ym - 2*y0 + yp
	if den == 0 {
		return 0
	}
	f := 0.5 * (ym - yp) / den
	if f > 0.99 || f < -0.99 || math.IsNaN(f) {
		return 0
	}
	return f
}

// shiftBranchFractional advances `x` by lag+frac samples (positive = x lagged
// and is pulled earlier), using linear interpolation for the fractional part —
// adequate here because the channel occupies ≲6% of the stream rate. The
// result keeps len(x); the tail is zero-padded.
func shiftBranchFractional(x []complex64, lag int, frac float64) []complex64 {
	out := make([]complex64, len(x))
	f := float32(frac)
	for i := range out {
		j := i + lag
		if f >= 0 {
			if j >= 0 && j+1 < len(x) {
				out[i] = x[j]*(1-complex(f, 0)) + x[j+1]*complex(f, 0)
			}
		} else {
			if j-1 >= 0 && j < len(x) {
				out[i] = x[j]*(1+complex(f, 0)) - x[j-1]*complex(f, 0)
			}
		}
	}
	return out
}

func combineWith(t *testing.T, br0, br1 []complex64, cal windowedCalibrator) []complex64 {
	t.Helper()
	const chunk = 4096
	out := make([]complex64, 0, len(br0))
	for pos := 0; pos < len(br0); pos += chunk {
		end := min(pos+chunk, len(br0))
		branches := [][]complex64{br0[pos:end], br1[pos:end]}
		cal.Observe(branches)
		y, err := cal.Combine(branches)
		if err != nil {
			t.Fatalf("Combine at %d: %v", pos, err)
		}
		out = append(out, y...)
	}
	return out
}

// decodeTETRABSCH down-converts to the 144 kHz TETRA channel rate and runs the
// shared control-channel receiver, returning CRC-clean / failed BSCH counts. The
// wiring mirrors newTETRAPipeline so every arm is scored identically and the
// combiner is the only variable.
func decodeTETRABSCH(t *testing.T, iq []complex64, rateHz, tuneHz float64) (ok, bad int64) {
	t.Helper()
	// A stream already at the channel rate (the nb-* arms) needs no second
	// down-conversion; running one anyway would filter it twice and make those
	// arms incomparable with the wideband ones.
	if rateHz == 144_000 && tuneHz == 0 {
		return decodeTETRABSCHAtChannelRate(t, iq)
	}
	ddc := ccdecoder.NewDownconverterWithOffset(rateHz, 144_000, tuneHz)

	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	cc := tetra.New(tetra.Options{SystemName: "divreplay", Bus: bus})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	ch, _ := tetra.ParseChannelType("")
	cc.SetExpectedChannel(ch)
	cc.SetColourCode(0) // auto-acquire from the BSCH sync burst

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        ddc.OutRateHz(),
		DibitSink:           func(d []uint8, base int) { cc.Process(d, base) },
		SoftSink:            func(diffs []complex64, base int) { cc.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true,
	})

	const chunk = 65536
	var scratch []complex64
	for pos := 0; pos < len(iq); pos += chunk {
		end := min(pos+chunk, len(iq))
		dec := ddc.Process(scratch, iq[pos:end])
		if len(dec) > 0 {
			rx.Process(dec)
		}
		scratch = dec[:0]
	}
	return cc.BSCHCounts()
}

// downconvert runs one branch through the same per-channel DDC the decoder
// uses, so the nb-* arms combine exactly what a per-channel combiner would see.
func downconvert(iq []complex64, rateHz, tuneHz float64) []complex64 {
	ddc := ccdecoder.NewDownconverterWithOffset(rateHz, 144_000, tuneHz)
	const chunk = 65536
	var out []complex64
	var scratch []complex64
	for pos := 0; pos < len(iq); pos += chunk {
		end := min(pos+chunk, len(iq))
		dec := ddc.Process(scratch, iq[pos:end])
		out = append(out, dec...)
		scratch = dec[:0]
	}
	return out
}

// decodeTETRABSCHAtChannelRate scores a stream that is already at the 144 kHz
// channel rate, feeding the identical receiver + control decoder the
// wideband arms get.
func decodeTETRABSCHAtChannelRate(t *testing.T, iq []complex64) (ok, bad int64) {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	cc := tetra.New(tetra.Options{SystemName: "divreplay", Bus: bus})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	ch, _ := tetra.ParseChannelType("")
	cc.SetExpectedChannel(ch)
	cc.SetColourCode(0)

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        144_000,
		DibitSink:           func(d []uint8, base int) { cc.Process(d, base) },
		SoftSink:            func(diffs []complex64, base int) { cc.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true,
	})
	const chunk = 65536
	for pos := 0; pos < len(iq); pos += chunk {
		end := min(pos+chunk, len(iq))
		rx.Process(iq[pos:end])
	}
	return cc.BSCHCounts()
}

// loadDiversityCapture reads the sidecar named by GT_DIVERSITY_CAPTURE and both
// branch files beside it. Skips the calling test when unset.
func loadDiversityCapture(t *testing.T) (br0, br1 []complex64, meta diversityCaptureMeta) {
	t.Helper()
	path := os.Getenv("GT_DIVERSITY_CAPTURE")
	if path == "" {
		t.Skip("set GT_DIVERSITY_CAPTURE to a <prefix>.diversity.json written by " +
			"sdr.soapy_remote[].diversity_capture to run the diversity A/B")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("parse sidecar: %v", err)
	}
	if meta.Branches != 2 || len(meta.BranchFiles) != 2 {
		t.Fatalf("capture has %d branches; this harness compares exactly 2", meta.Branches)
	}
	if meta.SampleRateHz <= 0 {
		t.Fatal("sidecar carries no sample rate")
	}
	dir := path[:strings.LastIndex(path, "/")+1]
	br0 = loadBranchFile(t, dir+meta.BranchFiles[0])
	br1 = loadBranchFile(t, dir+meta.BranchFiles[1])
	if len(br0) != len(br1) {
		t.Fatalf("branch files are not the same length (%d vs %d) — they are not "+
			"sample-aligned and no comparison from them is meaningful", len(br0), len(br1))
	}
	return br0, br1, meta
}

// loadBranchFile reads one diversity branch capture in either container,
// sniffing the fLaC marker from the file content (a flac branch decodes to the
// identical int16 samples its cs16 twin would carry, so the arms downstream
// cannot tell which container fed them).
func loadBranchFile(t *testing.T, path string) []complex64 {
	t.Helper()
	if baseband.IsFLACIQFile(path) {
		samples, _, err := baseband.ReadIQFLACSamples(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return samples
	}
	return loadCS16File(t, path)
}

func loadCS16File(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := make([]complex64, len(raw)/4)
	for i := range out {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		out[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return out
}

// meanPowerOf is the mean sample power of one branch, the driver's reference-
// selection metric (refPowerDbFS without the dB).
func meanPowerOf(x []complex64) float64 {
	if len(x) == 0 {
		return 0
	}
	var acc float64
	for _, z := range x {
		r, i := float64(real(z)), float64(imag(z))
		acc += r*r + i*i
	}
	return acc / float64(len(x))
}

// divWindow is one analysis window's summary of the branch relationship.
type divWindow struct{ rho, gainDb, phaseDeg, tSec float64 }

func gainsOf(trace []divWindow) []float64 {
	out := make([]float64, len(trace))
	for i, s := range trace {
		out[i] = s.gainDb
	}
	return out
}

func minOf(v []float64) float64 {
	m := v[0]
	for _, x := range v[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func maxOf(v []float64) float64 {
	m := v[0]
	for _, x := range v[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

func medianOf(v []float64) float64 {
	s := append([]float64(nil), v...)
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s[len(s)/2]
}

// TestDiversityCaptureLoaderReadsTheDriverFormat pins the other half of the
// sidecar contract asserted in internal/sdr/soapyremote/branchcapture_test.go:
// the key names are written here as literals, matching what the driver emits, so
// a rename on either side fails a test instead of silently loading a capture
// whose fields are all zero.
//
// It also pins the analysis the harness is for: two branches with a KNOWN
// relative phase must read back as that phase.
func TestDiversityCaptureLoaderReadsTheDriverFormat(t *testing.T) {
	dir := t.TempDir()
	const n = 16384
	br0 := make([]complex64, n)
	br1 := make([]complex64, n)
	const theta = 0.7 // rad
	for i := range br0 {
		ph := float64(i) * 0.37
		br0[i] = complex(float32(0.4*math.Cos(ph)), float32(0.4*math.Sin(ph)))
		br1[i] = complex(
			float32(0.4*math.Cos(ph+theta)),
			float32(0.4*math.Sin(ph+theta)),
		)
	}
	writeCS16File(t, dir+"/cap.br0.cs16", br0)
	writeCS16File(t, dir+"/cap.br1.cs16", br1)
	sidecar := dir + "/cap.diversity.json"
	if err := os.WriteFile(sidecar, []byte(`{
  "addr": "10.0.0.1:23313",
  "sample_rate_hz": 1000000,
  "format": "cs16",
  "diversity_mode": "tracking",
  "branches": 2,
  "branch_files": ["cap.br0.cs16", "cap.br1.cs16"],
  "samples_per_branch": 16384,
  "datagrams": 16,
  "dropped_datagrams": 0
}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	t.Setenv("GT_DIVERSITY_CAPTURE", sidecar)
	gotA, gotB, meta := loadDiversityCapture(t)
	if meta.SampleRateHz != 1e6 || meta.Branches != 2 || meta.Addr == "" {
		t.Fatalf("sidecar decoded as %+v — the field names do not match the driver's", meta)
	}
	if len(gotA) != n || len(gotB) != n {
		t.Fatalf("loaded %d/%d samples, want %d each", len(gotA), len(gotB), n)
	}

	var s diversity.CrossStats
	s.Accumulate(gotA, gotB)
	h, ok := s.Gain()
	if !ok {
		t.Fatal("no gain estimate from a clean fixture")
	}
	if got := cmplx.Phase(complex128(h)); math.Abs(got-theta) > 0.01 {
		t.Errorf("recovered branch phase %.4f rad, want %.4f", got, theta)
	}
	if rho := s.Coherence(); rho < 0.99 {
		t.Errorf("coherence %.4f on a noiseless fixture, want ~1", rho)
	}
}

func writeCS16File(t *testing.T, path string, iq []complex64) {
	t.Helper()
	buf := make([]byte, len(iq)*4)
	for i, z := range iq {
		binary.LittleEndian.PutUint16(buf[4*i:], uint16(int16(real(z)*32768)))
		binary.LittleEndian.PutUint16(buf[4*i+2:], uint16(int16(imag(z)*32768)))
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
