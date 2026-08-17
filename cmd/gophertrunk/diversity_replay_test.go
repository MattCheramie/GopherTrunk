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
//  2. FOUR DECODE ARMS through identical downstream wiring — branch 0 alone,
//     branch 1 alone, static combine, tracking combine — scored by CRC-clean
//     BSCH count. Decode yield is the verdict, never EVM: a combiner can improve
//     a constellation's look while decoding nothing, which this repo has
//     measured before.
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

	// ---- 2. Four decode arms ------------------------------------------------
	tuneHz := 0.0
	if v := os.Getenv("GT_DIVERSITY_TUNE_HZ"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_DIVERSITY_TUNE_HZ: %v", err)
		}
		tuneHz = f
	}

	window := int(meta.SampleRateHz * 2 / 1000) // the driver's 2 ms window
	if window < 4096 {
		window = 4096
	}
	arms := []struct {
		name string
		iq   []complex64
	}{
		{"branch0-only", br0},
		{"branch1-only", br1},
		{"static-combine", combineWith(t, br0, br1,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: 0}))},
		{"tracking-combine", combineWith(t, br0, br1,
			diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window}))},
	}

	type result struct {
		name    string
		ok, bad int64
	}
	var results []result
	for _, arm := range arms {
		ok, bad := decodeTETRABSCH(t, arm.iq, meta.SampleRateHz, tuneHz)
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
	if results[3].ok == 0 && results[0].ok == 0 && results[1].ok == 0 {
		t.Log("no arm decoded any BSCH — check GT_DIVERSITY_TUNE_HZ points at the control channel")
	}
}

// combineWith runs the two branches through a calibrator exactly as the driver
// does — Observe then Combine, in windows — and returns the combined stream.
func combineWith(t *testing.T, br0, br1 []complex64, cal *diversity.TrackingCalibrator) []complex64 {
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
	br0 = loadCS16File(t, dir+meta.BranchFiles[0])
	br1 = loadCS16File(t, dir+meta.BranchFiles[1])
	if len(br0) != len(br1) {
		t.Fatalf("branch files are not the same length (%d vs %d) — they are not "+
			"sample-aligned and no comparison from them is meaningful", len(br0), len(br1))
	}
	return br0, br1, meta
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
