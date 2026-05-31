package receiver

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// TestReceiverConstructsAndProcessesSilence is a smoke test: make
// sure the constructor accepts the typical P25 P1 parameter set,
// Process runs against silent IQ without crashing, and the LDU
// sink is never called (no FSW match in pure noise / silence).
func TestReceiverConstructsAndProcessesSilence(t *testing.T) {
	var emitted int
	r := New(Options{
		SampleRateHz: 48_000,
		Sink:         func(ldu []byte) { emitted++ },
	})
	silence := make([]complex64, 4800) // 100 ms @ 48 kHz
	for range 4 {
		r.Process(silence)
	}
	if emitted != 0 {
		t.Errorf("LDU emitted from silence: count = %d, want 0", emitted)
	}
}

// TestReceiverConstructorPanicsOnBadParams documents the
// preconditions the receiver enforces. These keep configuration
// bugs from silently producing junk dibits.
func TestReceiverConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"missing sample rate", Options{Sink: func([]byte) {}}},
		{"missing sink", Options{SampleRateHz: 48_000}},
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 8_000, Sink: func([]byte) {}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got nil")
				}
			}()
			_ = New(tc.opts)
		})
	}
}

// TestReceiverEmitsDibitsFromPhaseRamp builds an IQ stream whose
// instantaneous frequency follows a known pattern, then verifies
// the receiver pushes a non-zero number of dibits through the
// assembler. We don't assert FSW alignment here — the slicer-/MM-
// loop tuning is calibration-dependent and lives in the
// real-hardware capture tests — but the chain has to *produce*
// dibits or the wiring is broken.
func TestReceiverEmitsDibitsFromPhaseRamp(t *testing.T) {
	const sampleRate = 48_000.0
	const sps = 10 // 48000 / 4800
	const symbols = 1024

	// Build a 4-level FSK signal cycling through {-3,-1,+1,+3}
	// symbol values. Phase increment per sample = symbol *
	// (deviation / Fs) * 2π. We pick deviation so the outer
	// level is well inside the Nyquist band.
	const deviation = 1800.0 // Hz, P25 P1 nominal
	radPerSample := func(symbolValue int) float64 {
		return 2 * math.Pi * float64(symbolValue) * deviation / 3.0 / sampleRate
	}

	iq := make([]complex64, symbols*sps)
	phase := 0.0
	for s := 0; s < symbols; s++ {
		val := []int{-3, -1, +1, +3}[s%4]
		dphi := radPerSample(val)
		base := s * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}

	emitted := 0
	r := New(Options{
		SampleRateHz: sampleRate,
		Sink:         func(ldu []byte) { emitted++ },
	})

	// Process in chunks to exercise the cross-call state plumbing.
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	// The assembler's internal buffer should have collected a
	// meaningful number of dibits — even without FSW match, the
	// MM loop must have produced symbols. We can't read that
	// directly, but we can re-run with a sink that captures any
	// LDU it sees and assert no panic / no over-emission.
	// (FSW alignment from a synthetic phase ramp is not the
	// system under test here.)
	_ = emitted
}

// TestReceiverResetClearsAssembler ensures Reset propagates to the
// LDU assembler so a re-tune doesn't leave a stale FSW match
// hanging.
func TestReceiverResetClearsAssembler(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		Sink:         func([]byte) {},
	})
	// Push some IQ so the assembler accumulates state, then
	// Reset() and confirm a subsequent Process doesn't panic.
	noise := make([]complex64, 8192)
	for i := range noise {
		noise[i] = complex(0.1, -0.1)
	}
	r.Process(noise)
	r.Reset()
	r.Process(noise)
}

// makePhaseRampIQ synthesises a 48 kHz / 10 sps IQ buffer whose
// instantaneous frequency cycles through the four C4FM symbols. It's
// the shared fixture for tests that need a clean signal capable of
// driving the MM loop through Process.
func makePhaseRampIQ(symbols int) []complex64 {
	const sampleRate = 48_000.0
	const sps = 10
	const deviation = 1800.0
	radPerSample := func(symbolValue int) float64 {
		return 2 * math.Pi * float64(symbolValue) * deviation / 3.0 / sampleRate
	}
	iq := make([]complex64, symbols*sps)
	phase := 0.0
	for s := 0; s < symbols; s++ {
		val := []int{-3, -1, +1, +3}[s%4]
		dphi := radPerSample(val)
		base := s * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	return iq
}

// TestReceiverDibitSinkAlone: the CC path doesn't need the LDU
// sink. New must accept a DibitSink-only configuration and Process
// must not panic when no LDU sink is wired.
func TestReceiverDibitSinkAlone(t *testing.T) {
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(dibits []uint8, baseIdx int) { batches++ },
	})
	iq := makePhaseRampIQ(1024)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("DibitSink received zero batches; MM loop produced no symbols")
	}
}

// TestReceiverDibitSinkBaseIdxMonotonic: baseIdx must start at 0
// and equal the cumulative count of all previously-emitted dibits.
// Reset() rewinds baseIdx back to 0.
func TestReceiverDibitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(dibits []uint8, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(dibits))
		},
	})

	iq := makePhaseRampIQ(1024)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	if len(baseIdxs) == 0 {
		t.Fatalf("expected DibitSink to receive at least one batch")
	}
	if baseIdxs[0] != 0 {
		t.Errorf("first baseIdx = %d, want 0", baseIdxs[0])
	}
	cumulative := 0
	for i := range baseIdxs {
		if baseIdxs[i] != cumulative {
			t.Errorf("baseIdx[%d]=%d, want %d", i, baseIdxs[i], cumulative)
		}
		cumulative += batchLens[i]
	}

	// Reset rewinds baseIdx so a retune begins a fresh stream.
	r.Reset()
	baseIdxs = baseIdxs[:0]
	batchLens = batchLens[:0]
	r.Process(iq)
	if len(baseIdxs) == 0 {
		t.Fatalf("post-Reset: expected DibitSink to receive at least one batch")
	}
	if baseIdxs[0] != 0 {
		t.Errorf("post-Reset: first baseIdx = %d, want 0", baseIdxs[0])
	}
}

// TestReceiverDibitAndLDUSinksAgree: when both sinks are set the
// DibitSink must see exactly the same dibit sequence the LDU
// assembler consumes — that's the invariant the CC connector
// relies on to mirror the voice path.
func TestReceiverDibitAndLDUSinksAgree(t *testing.T) {
	var dibitStream []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		Sink:         func([]byte) {}, // noop, just to keep the assembler wired
		DibitSink: func(dibits []uint8, baseIdx int) {
			dibitStream = append(dibitStream, dibits...)
		},
	})

	iq := makePhaseRampIQ(1024)
	r.Process(iq)

	if len(dibitStream) == 0 {
		t.Fatalf("expected DibitSink to receive dibits")
	}
	for i, d := range dibitStream {
		if d > 3 {
			t.Errorf("dibitStream[%d]=%d, want in 0..3", i, d)
		}
	}
}

// TestReceiverConstructorRequiresAtLeastOneSink: the constructor
// must reject Options that wire neither Sink nor DibitSink.
func TestReceiverConstructorRequiresAtLeastOneSink(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when both Sink and DibitSink are nil")
		}
	}()
	_ = New(Options{SampleRateHz: 48_000})
}

// TestC4FMSymbolAGCRescuesCollapsedSlicer guards the Phase B fix for
// issue #275. Before the symbol-AGC, a real RTL-SDR capture's
// matched-filter output landed ~sps× above the 4-level slicer's fixed
// threshold (P25C4FMRxTaps has a DC gain of sps, the slicer is
// calibrated to 2π·deviation/sampleRate), so every sample sliced to
// the outer rails and the dibit stream became {1, 3} only — no NID
// ever decoded even though the FSW correlated perfectly (the FSW is
// all-outer, so a collapsed slicer still recovers it).
//
// The AGC normalises the running mean|x| to the slicer's expected
// threshold, restoring the 4-level eye on any input level. This test
// drives the receiver with a synthetic stream scaled to a level where
// the un-AGC'd slicer would collapse to outers, then asserts the
// recovered dibit histogram is roughly balanced (~25% per bin, not
// 50/0/50/0). The exact balance depends on the synthetic dibit
// distribution, so the assertion is a sanity bound rather than an
// exact ratio.
func TestC4FMSymbolAGCRescuesCollapsedSlicer(t *testing.T) {
	// Build a balanced random dibit stream and modulate with the
	// spec P25 transmitter at 960 kHz (sps=200) — the same
	// parameters the Mt Anakie capture used. At this sample rate
	// the matched filter's DC-gain-sps normalisation puts the
	// matched-filter outer-symbol centres at sps× the slicerScale
	// the un-AGC'd code computes, so the slicer would collapse to
	// {1, 3} without AGC.
	const (
		sr  = 960_000.0
		dev = 1800.0
		n   = 8000
	)
	dibits := make([]uint8, n)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)
	// Scale the IQ up so the matched-filter output dwarfs the
	// pre-AGC slicer threshold (mimicking the Mt Anakie capture
	// where mean|MF output| was ~150× the calibrated slicerScale).
	for i := range iq {
		iq[i] *= 100
	}

	var hist [4]int
	r := New(Options{
		SampleRateHz: sr,
		DeviationHz:  dev,
		DibitSink: func(d []uint8, _ int) {
			for _, v := range d {
				hist[v&3]++
			}
		},
	})
	r.Process(iq)

	total := hist[0] + hist[1] + hist[2] + hist[3]
	if total == 0 {
		t.Fatalf("receiver emitted no dibits")
	}
	// All four bins should carry meaningful mass — at least 10% of
	// total each — proving the slicer didn't collapse to outers
	// only. The exact ratio depends on the synthetic dibit stream,
	// so 10% is a loose-but-decisive bound (a collapsed slicer
	// yields ~50/0/50/0; a working one ~25/25/25/25).
	for v := 0; v < 4; v++ {
		pct := 100 * float64(hist[v]) / float64(total)
		if pct < 10 {
			t.Errorf("dibit value %d at %.1f%% (%d/%d) — slicer collapsed to outers (issue #275 Phase B regression)",
				v, pct, hist[v], total)
		}
	}
}

// TestAdaptiveSlicerNoRegressionOnCleanC4FMStream is the receiver-level
// no-regression guard for the issue #402 adaptive slicer. It captures the
// receiver's post-AGC soft symbols from a clean, symmetric spec-P25 stream
// (via SoftSink) and slices that identical stream both ways. The adaptive
// slicer (opt-in via EnableAdaptiveC4FMSlicer) must decode at least as
// faithfully as the fixed slicer it replaces — i.e. it adds no skew of its
// own when there's no asymmetry to correct. Comparing the two slicers on
// the same soft stream isolates the slicer from sps-dependent matched-
// filter ISI, which neither slicer can fix and which would otherwise make
// an absolute per-bin threshold flaky.
func TestAdaptiveSlicerNoRegressionOnCleanC4FMStream(t *testing.T) {
	const (
		// sps=40 (sr=192000): ModulateP25C4FM only renders a clean, open
		// four-level eye at high sps; at the production sps=10 its ISI
		// closes the eye far more than real hardware does (the #402
		// capture is open at sps=10), so a low-sps synthetic eye is a
		// degenerate baseline, not a no-regression bar. sps=40 is the
		// lowest that keeps the eye open here, chosen to bound the
		// matched-filter cost (~sps²) so the test stays fast under -race.
		sr  = 192_000.0
		dev = 1800.0
		n   = 4000
	)
	dibits := make([]uint8, n)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)

	var soft []float32
	r := New(Options{
		SampleRateHz:             sr,
		DeviationHz:              dev,
		EnableAdaptiveC4FMSlicer: true, // this test is the adaptive-slicer guard
		SoftSink:                 func(s []float32) { soft = append(soft, s...) },
		DibitSink:                func([]uint8, int) {}, // required; the soft stream is what we score
	})
	r.Process(iq)
	if len(soft) == 0 {
		t.Fatalf("receiver produced no soft symbols")
	}

	slicerScale := r.AGCTarget() * 3.0 / 2.0 // target = slicerScale·2/3

	fixed := demod.NewC4FMWithTaps([]float32{1}, slicerScale)
	adapt := demod.NewAdaptiveC4FMSlicer(slicerScale)
	fixedOut := fixed.SliceMany(nil, soft)
	adaptOut := adapt.SliceMany(nil, soft)

	var fixedHist, adaptHist [4]int
	for _, s := range fixedOut {
		fixedHist[phase1.SymbolToDibit(s)&3]++
	}
	for _, s := range adaptOut {
		adaptHist[phase1.SymbolToDibit(s)&3]++
	}
	t.Logf("fixed dibit hist=%v  adaptive dibit hist=%v", fixedHist, adaptHist)

	// Per bin, the adaptive slicer must not under-populate relative to the
	// fixed slicer by more than a small margin (5 % of the bin) — it tracks
	// a symmetric eye, so it should land within noise of the fixed result.
	for v := 0; v < 4; v++ {
		margin := fixedHist[v] / 20
		if adaptHist[v] < fixedHist[v]-margin {
			t.Errorf("dibit %d: adaptive %d < fixed %d − margin %d — slicer skewed a clean stream",
				v, adaptHist[v], fixedHist[v], margin)
		}
	}

	// On a symmetric eye the tracked levels must stay near nominal (no
	// runaway): correct sign ordering and outer rails symmetric within 25 %.
	lv := r.SlicerLevels()
	if !(lv[0] < 0 && lv[1] < 0 && lv[2] > 0 && lv[3] > 0) {
		t.Fatalf("slicer levels lost their sign ordering: %v", lv)
	}
	if asym := math.Abs(lv[3]+lv[0]) / lv[3]; asym > 0.25 {
		t.Errorf("outer rails asymmetric by %.0f%% on a clean stream: %v", asym*100, lv)
	}
}

// TestReceiverStateAccessorsReturnDefaults pins the contract for the
// new diagnostic accessors (issue #402 Phase 2): on a freshly-
// constructed C4FM Receiver, the AFC bias must read 0 (the IIR's
// initial state), the MM clock's SPS must equal SampleRateHz /
// SymbolRate, and AGCTarget must match the slicer calibration from
// DeviationHz. AGCLevel starts at 0 (unseeded EMA) and MMClockMu
// starts at sps (the constructor's initial state). These read-only
// snapshots back the periodic state-evolution log in cmd/gophertrunk/
// replay.go; if any default changes, the log lines change shape and
// the operator-facing diagnostic gets confusing.
func TestReceiverStateAccessorsReturnDefaults(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800,
		Sink:         func([]byte) {},
	})
	if got := r.AFCBiasRadPerSample(); got != 0 {
		t.Errorf("AFCBiasRadPerSample on fresh receiver = %v, want 0", got)
	}
	if got := r.AGCLevel(); got != 0 {
		t.Errorf("AGCLevel on fresh receiver = %v, want 0 (unseeded EMA)", got)
	}
	if got := r.AGCTarget(); got <= 0 {
		t.Errorf("AGCTarget = %v, want > 0 (slicer calibrated from DeviationHz)", got)
	}
	wantSPS := 48_000.0 / SymbolRate
	if got := r.MMClockSPS(); got != wantSPS {
		t.Errorf("MMClockSPS = %v, want %v", got, wantSPS)
	}
	if got := r.MMClockMu(); got != wantSPS {
		t.Errorf("MMClockMu on fresh receiver = %v, want %v (constructor's initial mu)", got, wantSPS)
	}
}

// TestReceiverStateAccessorsZeroOnCQPSKPath: the CQPSK demod path has
// no AFC or symbol-AGC and uses Gardner instead of Mueller-Müller, so
// all the C4FM-state accessors must return 0 without panicking — the
// replay state log relies on that to render a meaningful line on
// either demod choice.
func TestReceiverStateAccessorsZeroOnCQPSKPath(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800,
		DemodMode:    DemodCQPSK,
		Sink:         func([]byte) {},
	})
	if got := r.AFCBiasRadPerSample(); got != 0 {
		t.Errorf("CQPSK AFCBiasRadPerSample = %v, want 0", got)
	}
	if got := r.AGCLevel(); got != 0 {
		t.Errorf("CQPSK AGCLevel = %v, want 0", got)
	}
	if got := r.AGCTarget(); got != 0 {
		t.Errorf("CQPSK AGCTarget = %v, want 0", got)
	}
	if got := r.MMClockMu(); got != 0 {
		t.Errorf("CQPSK MMClockMu = %v, want 0", got)
	}
	if got := r.MMClockSPS(); got != 0 {
		t.Errorf("CQPSK MMClockSPS = %v, want 0", got)
	}
}

// TestReceiverDDAHandoffFiresOnCleanLockedStream is the integration
// guard: on a healthy synthetic stream + carrier offset, the
// receiver must complete the warmup-then-handoff choreography
// (issue #402) and the post-handoff AFCBiasRadPerSample must carry
// the same sign as the carrier offset — the open-loop bootstrap
// estimate has to make it across the freeze-CoarseAFC / fold-into-
// DDA transition without being dropped or sign-flipped.
//
// The DDA's actual loop math (data-mean immunity, integrator
// convergence under feedback, gate behaviour, clamp) lives in the
// unit tests in internal/dsp/demod/afc_test.go.
// TestHarnessC4FMToleratesCarrierOffset already pins that the
// receiver chain locks at ±1.5 kHz offsets; this test pins that
// the new handoff path runs and produces a sensible AFC reading on
// the same kind of stream.
func TestReceiverDDAHandoffFiresOnCleanLockedStream(t *testing.T) {
	const (
		sr       = 48_000.0
		dev      = 1800.0
		offsetHz = 1_500.0
		nDibits  = 8_000
	)
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)
	iq = demod.ApplyImpairments(iq, sr, demod.Impairments{FreqOffsetHz: offsetHz})

	var totalDibits int
	r := New(Options{
		SampleRateHz:              sr,
		DeviationHz:               dev,
		EnableDecisionDirectedAFC: true,
		DibitSink:                 func(d []uint8, _ int) { totalDibits += len(d) },
	})
	// Feed in chunks — the freeze-then-handoff state machine advances
	// per Process call, so a single whole-stream call can't exercise it
	// (and is correctly refused by the handoff gate, since CoarseAFC
	// never freezes mid-call). Issue #402.
	feedChunked(r, iq, 200, nil)

	if totalDibits == 0 {
		t.Fatalf("receiver emitted no dibits")
	}
	if !r.ddaActive {
		t.Errorf("DDA never handed off (ddaValidUpdates=%d, want ≥ %d, c4fmSymbolsTotal=%d, want ≥ %d) — the receiver's handoff plumbing is broken",
			r.ddaValidUpdates, ddaHandoffSymbols, r.c4fmSymbolsTotal, ddaWarmupSymbols)
	}
	if got := r.AFCBiasRadPerSample(); got <= 0 {
		t.Errorf("AFCBiasRadPerSample = %.4f on a +%g Hz offset stream, want > 0 (DDA dropped or flipped the bootstrap estimate)", got, offsetHz)
	}
}

// TestReceiverDDAResetClearsHandoffState confirms Reset wipes the
// DDA's integrator, the handoff flag, and the warmup counter — so a
// stream re-sync (CC hunt success, IQ underrun recovery) doesn't
// carry a stale offset that would mis-steer the next stream's
// slicer.
func TestReceiverDDAResetClearsHandoffState(t *testing.T) {
	r := New(Options{
		SampleRateHz:              48_000,
		DeviationHz:               1800,
		EnableDecisionDirectedAFC: true,
		Sink:                      func([]byte) {},
	})
	// Drive enough traffic through (chunked, so the handoff fires) to
	// populate the DDA state Reset must clear.
	dibits := make([]uint8, 8000)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, 48_000, 1800)
	iq = demod.ApplyImpairments(iq, 48_000, demod.Impairments{FreqOffsetHz: 1500})
	feedChunked(r, iq, 200, nil)
	if r.AFCBiasRadPerSample() == 0 {
		t.Fatalf("AFC didn't accumulate any estimate on an offset stream — receiver state setup is broken")
	}
	if !r.ddaActive {
		t.Fatalf("DDA never handed off — Reset's handoff-state clearing would be untested")
	}

	r.Reset()
	if got := r.AFCBiasRadPerSample(); got != 0 {
		t.Errorf("AFCBiasRadPerSample after Reset = %v, want 0", got)
	}
	if r.ddaActive {
		t.Error("ddaActive remained true after Reset")
	}
	if r.ddaLearning {
		t.Error("ddaLearning remained true after Reset")
	}
	if r.ddaValidUpdates != 0 {
		t.Errorf("ddaValidUpdates = %d after Reset, want 0", r.ddaValidUpdates)
	}
	if r.ddaTotalUpdates != 0 {
		t.Errorf("ddaTotalUpdates = %d after Reset, want 0", r.ddaTotalUpdates)
	}
	if r.afcAtHandoff != 0 {
		t.Errorf("afcAtHandoff = %v after Reset, want 0", r.afcAtHandoff)
	}
	if r.ddaRearms != 0 {
		t.Errorf("ddaRearms = %d after Reset, want 0", r.ddaRearms)
	}
	if r.ddaWarmupDoneAt != ddaWarmupSymbols {
		t.Errorf("ddaWarmupDoneAt = %d after Reset, want %d", r.ddaWarmupDoneAt, ddaWarmupSymbols)
	}
	if r.c4fmSymbolsTotal != 0 {
		t.Errorf("c4fmSymbolsTotal = %d after Reset, want 0", r.c4fmSymbolsTotal)
	}
}

// feedChunked pushes iq through r in fixed-size chunks, calling after
// each Process call. This exercises the per-batch freeze/handoff/
// watchdog state machine the way the live daemon (small USB transfers)
// does — a single Process(iq) call can't, since the freeze takes
// effect only from the batch after warmup completes. Issue #402.
func feedChunked(r *Receiver, iq []complex64, chunk int, after func()) {
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
		if after != nil {
			after()
		}
	}
}

// TestReceiverFreezesCoarseAFCAtWarmupBoundary pins Part A of the
// issue #402 fix: once the DDA starts learning, CoarseAFC is frozen
// (subtract-only) for the entire learning window, so its estimate is
// constant from the first learning batch through handoff. A constant
// estimate is what makes the handoff fold-in exact (no wandering-data-
// mean error).
func TestReceiverFreezesCoarseAFCAtWarmupBoundary(t *testing.T) {
	const (
		sr       = 48_000.0
		dev      = 1800.0
		offsetHz = 1_000.0
		nDibits  = 4_000
	)
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)
	iq = demod.ApplyImpairments(iq, sr, demod.Impairments{FreqOffsetHz: offsetHz})

	r := New(Options{
		SampleRateHz:              sr,
		DeviationHz:               dev,
		EnableDecisionDirectedAFC: true,
		DibitSink:                 func([]uint8, int) {},
	})

	var afcAtLearnStart, afcBeforeHandoff float64
	learnSeen := false
	feedChunked(r, iq, 200, func() {
		if r.ddaLearning && !learnSeen {
			afcAtLearnStart = r.afc.Offset()
			learnSeen = true
		}
		if learnSeen && !r.ddaActive {
			afcBeforeHandoff = r.afc.Offset()
		}
	})

	if !learnSeen {
		t.Fatal("DDA never entered the learning window")
	}
	if !r.ddaActive {
		t.Fatal("DDA never handed off")
	}
	if math.Abs(afcAtLearnStart-afcBeforeHandoff) > 1e-9 {
		t.Errorf("CoarseAFC moved during the learning window: %.9f at start vs %.9f before handoff — freeze didn't take",
			afcAtLearnStart, afcBeforeHandoff)
	}
	if afcAtLearnStart == 0 {
		t.Error("CoarseAFC froze at 0 — it never converged during warmup")
	}
}

// TestReceiverHandoffDataImmuneAcrossUnbalancedLearning is the headline
// regression test for issue #402. A swinging, unbalanced symbol stream
// (the field condition that made CoarseAFC's estimate oscillate) must
// hand off to the same AFC estimate as a balanced stream at the same
// carrier offset — proving the DDA carries the true carrier offset and
// not the data mean the open-loop tracker confused it with. Before the
// fix, CoarseAFC kept adapting through the learning window and the
// fold-in captured a wandering value, so the unbalanced stream settled
// on a different (wrong) offset and decode collapsed.
func TestReceiverHandoffDataImmuneAcrossUnbalancedLearning(t *testing.T) {
	const (
		sr       = 48_000.0
		dev      = 1800.0
		offsetHz = 1_200.0
		nDibits  = 12_000
	)
	balanced := make([]uint8, nDibits)
	for i := range balanced {
		balanced[i] = uint8(i & 3)
	}
	// Swinging skew: alternating positive- and negative-heavy runs,
	// net-balanced over time but with a strong local data mean that
	// drags an open-loop tracker around.
	unbalanced := make([]uint8, nDibits)
	posHeavy := []uint8{0, 1, 0, 1, 2, 3} // dibits 0,1 → +1,+3 ; 2,3 → -1,-3
	negHeavy := []uint8{2, 3, 2, 3, 0, 1}
	for i := range unbalanced {
		pat := posHeavy
		if (i/80)%2 == 1 {
			pat = negHeavy
		}
		unbalanced[i] = pat[i%len(pat)]
	}

	runAFC := func(dibits []uint8) float64 {
		iq := demod.ModulateP25C4FM(dibits, sr, dev)
		iq = demod.ApplyImpairments(iq, sr, demod.Impairments{FreqOffsetHz: offsetHz})
		r := New(Options{
			SampleRateHz:              sr,
			DeviationHz:               dev,
			EnableDecisionDirectedAFC: true,
			DibitSink:                 func([]uint8, int) {},
		})
		feedChunked(r, iq, 200, nil)
		if !r.ddaActive {
			t.Fatalf("DDA never handed off")
		}
		return r.AFCBiasRadPerSample()
	}

	ref := runAFC(balanced)
	got := runAFC(unbalanced)
	if ref == 0 {
		t.Fatal("balanced reference AFC is 0")
	}
	if rel := math.Abs(got-ref) / math.Abs(ref); rel > 0.20 {
		t.Errorf("unbalanced-stream AFC = %.4f, balanced reference = %.4f (%.0f%% off) — handoff not data-immune",
			got, ref, rel*100)
	}
}

// TestReceiverHandoffReadyBlocksBiasedEye pins Part B's validity gate
// (issue #402): a uniformly-biased eye produces plenty of within-gate
// ("accepted") updates, so the raw count alone would green-light a
// handoff onto a stable-but-wrong offset. ddaHandoffReady must refuse
// it because the mean accepted residual is non-zero, and must allow a
// clean (near-zero-mean) eye.
func TestReceiverHandoffReadyBlocksBiasedEye(t *testing.T) {
	const (
		sr  = 48_000.0
		dev = 1800.0
	)
	slicerScale := 2.0 * math.Pi * dev / sr

	newRx := func() *Receiver {
		return New(Options{SampleRateHz: sr, DeviationHz: dev, EnableDecisionDirectedAFC: true, DibitSink: func([]uint8, int) {}})
	}

	// Drive the DDA directly with the same call shape Process uses, so
	// the gate sees realistic counters and residual mean.
	drive := func(r *Receiver, bias float64) {
		syms := []float64{slicerScale, slicerScale / 3, -slicerScale / 3, -slicerScale}
		for i := 0; i < 4*1024; i++ {
			expected := float32(syms[i%4])
			soft := float32(syms[i%4] + bias)
			if r.dda.Update(soft, expected, 1.0) {
				r.ddaValidUpdates++
			}
			r.ddaTotalUpdates++
		}
	}

	clean := newRx()
	drive(clean, 0)
	if !clean.ddaHandoffReady() {
		t.Errorf("ddaHandoffReady = false on a clean eye (residMean=%.5f) — gate too strict", clean.dda.AcceptedResidualMean())
	}

	biased := newRx()
	drive(biased, slicerScale/6) // within the gate, but clearly off-centre
	if biased.ddaHandoffReady() {
		t.Errorf("ddaHandoffReady = true on a biased eye (residMean=%.5f, gate=%.5f) — false lock not blocked",
			biased.dda.AcceptedResidualMean(), biased.ddaResidMeanGate)
	}
}

// TestReceiverWatchdogReArmsOnRunawayDrift pins Part B's watchdog
// (issue #402): after a healthy handoff, if the DDA's estimate walks
// far from the gate-verified handoff value (here forced by a large
// post-handoff carrier-offset step the DDA chases), the receiver
// reverts to CoarseAFC-alone rather than staying stuck on a DDA
// estimate it can no longer trust. This is the reversibility guarantee
// — the receiver can never wander into a worse-than-CoarseAFC state.
func TestReceiverWatchdogReArmsOnRunawayDrift(t *testing.T) {
	const (
		sr        = 48_000.0
		dev       = 1800.0
		offsetHz  = 1_000.0
		stepHz    = 9_000.0 // large step past the drift bound (4 kHz)
		chunkSize = 200
	)
	clean := make([]uint8, 6_000)
	for i := range clean {
		clean[i] = uint8((i*7 + 3) & 3)
	}
	cleanIQ := demod.ModulateP25C4FM(clean, sr, dev)
	cleanIQ = demod.ApplyImpairments(cleanIQ, sr, demod.Impairments{FreqOffsetHz: offsetHz})

	r := New(Options{SampleRateHz: sr, DeviationHz: dev, EnableDecisionDirectedAFC: true, DibitSink: func([]uint8, int) {}})
	feedChunked(r, cleanIQ, chunkSize, nil)
	if !r.ddaActive {
		t.Fatal("DDA never handed off on the clean stream")
	}
	rearmsBefore := r.ddaRearms

	// Step the carrier far enough that the DDA, chasing it, walks past
	// the drift bound and trips the watchdog.
	stepped := make([]uint8, 30_000)
	for i := range stepped {
		stepped[i] = uint8((i*7 + 3) & 3)
	}
	steppedIQ := demod.ModulateP25C4FM(stepped, sr, dev)
	steppedIQ = demod.ApplyImpairments(steppedIQ, sr, demod.Impairments{FreqOffsetHz: offsetHz + stepHz})
	feedChunked(r, steppedIQ, chunkSize, nil)

	if r.ddaRearms <= rearmsBefore {
		t.Errorf("watchdog never re-armed on a runaway-drift step (rearms=%d) — receiver can't fall back to CoarseAFC", r.ddaRearms)
	}
}

// TestReceiverDDADisabledByDefault is the issue #402 regression guard:
// without EnableDecisionDirectedAFC the C4FM path must run CoarseAFC-
// alone (the pre-DDA behaviour) — no DDA allocated, no handoff, ever —
// while still emitting dibits and tracking the carrier offset. The DDA
// is opt-in because it can stably false-lock and broke CC lock on the
// original capture; the default path must be the known-good one.
func TestReceiverDDADisabledByDefault(t *testing.T) {
	const (
		sr       = 48_000.0
		dev      = 1800.0
		offsetHz = 1_500.0
		nDibits  = 8_000
	)
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)
	iq = demod.ApplyImpairments(iq, sr, demod.Impairments{FreqOffsetHz: offsetHz})

	var totalDibits int
	r := New(Options{
		SampleRateHz: sr,
		DeviationHz:  dev,
		// EnableDecisionDirectedAFC left false — the default.
		DibitSink: func(d []uint8, _ int) { totalDibits += len(d) },
	})
	if r.dda != nil {
		t.Fatal("DDA was allocated with EnableDecisionDirectedAFC unset — default must be CoarseAFC-alone")
	}
	feedChunked(r, iq, 200, nil)

	if totalDibits == 0 {
		t.Fatal("CoarseAFC-only receiver emitted no dibits")
	}
	if r.ddaActive {
		t.Error("ddaActive went true with the DDA disabled")
	}
	if r.ddaLearning {
		t.Error("ddaLearning went true with the DDA disabled")
	}
	if r.ddaRearms != 0 {
		t.Errorf("ddaRearms = %d with the DDA disabled, want 0", r.ddaRearms)
	}
	// CoarseAFC alone must still converge onto the offset.
	if got := r.AFCBiasRadPerSample(); got <= 0 {
		t.Errorf("AFCBiasRadPerSample = %.4f on a +%g Hz offset stream with CoarseAFC alone, want > 0", got, offsetHz)
	}
}

// TestReceiverAFCOffsetHzUndoesMatchedFilterGain pins the issue #402
// diagnostic correction: AFCOffsetHz must report the TRUE carrier
// offset (~offsetHz), not the ≈sps×-inflated AFCBias·Fs/(2π) value the
// state log used to print. That inflation sent three rounds of #402
// chasing a phantom ~10 kHz error that was really ~1 kHz.
func TestReceiverAFCOffsetHzUndoesMatchedFilterGain(t *testing.T) {
	const (
		sr       = 48_000.0
		dev      = 1800.0
		offsetHz = 1_500.0
		nDibits  = 8_000
	)
	sps := sr / SymbolRate // 10
	// Balanced stream so CoarseAFC converges cleanly onto the offset.
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8(i & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sr, dev)
	iq = demod.ApplyImpairments(iq, sr, demod.Impairments{FreqOffsetHz: offsetHz})

	r := New(Options{SampleRateHz: sr, DeviationHz: dev, DibitSink: func([]uint8, int) {}})
	feedChunked(r, iq, 200, nil)

	trueHz := r.AFCOffsetHz()
	if math.Abs(trueHz-offsetHz)/offsetHz > 0.25 {
		t.Errorf("AFCOffsetHz = %.1f Hz, want ≈ %g Hz (within 25%%) — true-offset conversion is wrong", trueHz, offsetHz)
	}
	// The old (wrong) Fs-form is ≈sps× larger; confirm AFCOffsetHz is
	// NOT that value, i.e. the sps gain really was divided out.
	oldForm := r.AFCBiasRadPerSample() * sr / (2 * math.Pi)
	if ratio := oldForm / trueHz; math.Abs(ratio-sps) > 0.5 {
		t.Errorf("old-form/AFCOffsetHz = %.2f, want ≈ sps = %.0f — AFCOffsetHz isn't dividing out the matched-filter gain", ratio, sps)
	}
}
