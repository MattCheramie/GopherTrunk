package receiver

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// dibitsToLSMIQ synthesises an LSM (Linear Simulcast Modulation) IQ
// stream from a canonical-TIA dibit sequence. The dibits are first
// remapped through the inverse of lsmDibitRemap so the PiOver4DQPSK
// modulator (rotation=π/4) emits the spec-correct LSM phase deltas:
//
//	canonical dibit 0 → modulator dibit 0 → phase delta +π/4   (spec)
//	canonical dibit 1 → modulator dibit 1 → phase delta +3π/4  (spec)
//	canonical dibit 2 → modulator dibit 3 → phase delta -π/4   (spec)
//	canonical dibit 3 → modulator dibit 2 → phase delta -3π/4  (spec)
//
// The pre-remap is the inverse of the demod's post-remap so the
// round-trip "dibit → IQ → dibit" identity holds end-to-end.
func dibitsToLSMIQ(t *testing.T, dibits []uint8, sps, span int, alpha float64) []complex64 {
	t.Helper()
	var inv [4]uint8
	for i, m := range lsmDibitRemap {
		inv[m] = uint8(i)
	}
	pre := make([]uint8, len(dibits))
	for i, d := range dibits {
		pre[i] = inv[d&3]
	}
	return demod.ModulatePiOver4DQPSK(pre, sps, span, alpha, math.Pi/4)
}

// TestCQPSKDemodRoundTripIdentity feeds a long pseudo-random dibit stream
// through the modulator → DemodCQPSK demodulator chain and asserts the
// recovered dibits match the input, confirming the path produces canonical
// TIA-102.BAAA dibits compatible with the FSW + NID + TSBK decoders
// downstream.
//
// The fixture uses random (not constant) data on purpose: a constant dibit
// stream is, under LSM, a single constant-phase-step tone — physically
// indistinguishable from an unmodulated carrier at +baud/8, which the
// carrier-recovery stage (issue #492) correctly tunes out. Random data has
// the broad spectrum symmetric about the carrier that a real (scrambled)
// control channel does, so the coarse seed reads ~0 and the round trip is a
// near-identity. The matched-filter group delay and differential reference
// shift the stream by a fixed amount, so we recover the alignment by a
// best-match search rather than assuming dibit 0 lines up with sample 0.
func TestCQPSKDemodRoundTripIdentity(t *testing.T) {
	const sampleRate = 48_000.0
	const sps = 10
	const symbols = 600

	rng := uint32(0x5eed1234)
	in := make([]uint8, symbols)
	for i := range in {
		rng = rng*1664525 + 1013904223
		in[i] = uint8((rng >> 16) & 3)
	}
	iq := dibitsToLSMIQ(t, in, sps, PulseSpanSymbols, RolloffAlpha)

	var captured []uint8
	r := New(Options{
		SampleRateHz: sampleRate,
		DemodMode:    DemodCQPSK,
		DibitSink: func(d []uint8, _ int) {
			captured = append(captured, d...)
		},
	})
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if len(captured) < symbols/2 {
		t.Fatalf("captured %d dibits, want at least %d", len(captured), symbols/2)
	}

	// Recover the fixed alignment offset: slide the input over the captured
	// stream and pick the shift with the most matches over a settled window.
	const settle = 40 // skip loop-acquisition + matched-filter fill
	bestShift, bestMatch := 0, -1
	for shift := 0; shift < 24; shift++ {
		match, n := 0, 0
		for i := settle; i < len(in) && i+shift < len(captured); i++ {
			if in[i] == captured[i+shift] {
				match++
			}
			n++
		}
		if n > 0 && match > bestMatch {
			bestMatch, bestShift = match, shift
		}
	}
	// Count the match ratio at the recovered alignment.
	match, total := 0, 0
	for i := settle; i < len(in) && i+bestShift < len(captured); i++ {
		if in[i] == captured[i+bestShift] {
			match++
		}
		total++
	}
	if total == 0 {
		t.Fatal("no overlap between input and captured dibits")
	}
	ratio := float64(match) / float64(total)
	if ratio < 0.95 {
		t.Errorf("round-trip recovered %d/%d dibits (%.1f%%) at shift %d; want >= 95%%",
			match, total, 100*ratio, bestShift)
	}
}

// TestCQPSKDemodRecoversFSW: synthesise an LSM stream that embeds the
// canonical P25 FSW and confirm the receiver recovers it — across the
// full range of starting symbol phases, not just the perfectly-aligned
// one. This is the issue #492 regression guard.
//
// A real capture's symbol clock has no relationship to sample 0, so the
// demod must lock from an arbitrary sub-symbol phase. The Gardner loop
// applies its timing correction in samples, so the effective per-symbol
// gain is gain/sps; at this path's 10 sps the inherited 0.03 step left
// the loop ~5× over-gained, overshooting the timing null so it only
// "locked" when the input was already aligned at sample phase 0 — the one
// phase every fixture here used to start on, which is why the bug hid
// behind green tests (issue #492). With the gain corrected
// (defaultGardnerGain) the loop pulls in from essentially any phase. A
// couple of phases still land on a residual false lock, so this asserts a
// strong-majority pull-in rather than every phase.
func TestCQPSKDemodRecoversFSW(t *testing.T) {
	const sampleRate = 48_000.0
	const sps = 10

	// Long lead-in (loop convergence) + FSW + trailer.
	in := make([]uint8, 0)
	for i := 0; i < 256; i++ {
		in = append(in, uint8(i&3))
	}
	in = append(in, phase1.FrameSyncWord[:]...)
	for i := 0; i < 64; i++ {
		in = append(in, uint8((i+2)&3))
	}
	base := dibitsToLSMIQ(t, in, sps, PulseSpanSymbols, RolloffAlpha)

	locked := 0
	for k := 0; k < sps; k++ {
		var captured []uint8
		r := New(Options{
			SampleRateHz: sampleRate,
			DemodMode:    DemodCQPSK,
			DibitSink:    func(d []uint8, _ int) { captured = append(captured, d...) },
		})
		// Drop k leading samples to start the stream at sub-symbol phase k,
		// and feed in chunks to exercise the cross-call timing state.
		shifted := base[k:]
		const chunk = 4096
		for i := 0; i < len(shifted); i += chunk {
			end := i + chunk
			if end > len(shifted) {
				end = len(shifted)
			}
			r.Process(shifted[i:end])
		}
		det := phase1.NewSyncDetector(2)
		if hits, _ := det.Process(nil, captured, 0); len(hits) > 0 {
			locked++
		}
	}
	// Pre-fix this was 1/10; the upsampling fix takes it to a strong
	// majority. Require well over half so a regression that narrows the
	// timing pull-in is caught, without pinning the exact (interpolator-
	// dependent) count.
	if locked < 6 {
		t.Fatalf("CQPSK recovered FSW from only %d/%d starting phases; want >= 6 (issue #492 timing pull-in)", locked, sps)
	}
}

// TestCQPSKDemodRecoversFSWWithCarrierOffset is the issue #492 decode-fix
// guard: it injects a residual carrier-frequency offset (the thing a real
// tuner carries and that the prior, zero-offset fixtures never modelled)
// and asserts the path still recovers the FSW. Before the carrier-recovery
// stage the differential decoder spun the whole constellation by
// 2π·Δf/baud per symbol and the FSW never correlated at any nonzero
// offset; the coarse seed + Costas loop must now pull each offset back in.
//
// The sweep covers the realistic post-channelisation residual range (a few
// hundred Hz to a couple kHz, all inside the coarse search half-width) at
// both signs, and one case with additive noise so a too-tight loop that
// can only acquire on a perfectly clean fixture is caught.
func TestCQPSKDemodRecoversFSWWithCarrierOffset(t *testing.T) {
	// Single-buffer path: one chunk larger than the whole fixture.
	runCQPSKCarrierOffsetSweep(t, 4096)
}

// TestCQPSKDemodRecoversFSWWithCarrierOffsetStreaming is the same sweep fed
// in production-realistic small chunks. The live SDR + replay both hand the
// receiver only ~160–200 samples per Process() call (8192 SDR samples ÷ the
// ~42× DDC ratio), far below cqpskSeedMinSamples. The earlier 4096-chunk
// test fired the coarse seed on its first chunk and so never exercised the
// streamed path — where the per-call length gate never tripped, the NCO
// stayed identity, the carrier was never removed, and the post-Gardner
// Costas railed at its ±baud/8 clamp (issue #492 round 2). This guards that
// the accumulated seed fires across many tiny chunks, including the >600 Hz
// offsets the Costas clamp alone cannot reach.
func TestCQPSKDemodRecoversFSWWithCarrierOffsetStreaming(t *testing.T) {
	runCQPSKCarrierOffsetSweep(t, 192)
}

func runCQPSKCarrierOffsetSweep(t *testing.T, chunk int) {
	t.Helper()
	const sampleRate = 48_000.0
	const sps = 10

	// Longer lead-in than the zero-offset test: the carrier loop needs a
	// few hundred symbols to acquire before the FSW arrives. Use a
	// pseudo-random dibit stream (a real P25 control channel is scrambled,
	// so its spectrum is broad and symmetric about the carrier) — a
	// periodic ramp would put the spectral energy on discrete lines and
	// defeat the coarse PSD-peak seed, which is a fixture artifact, not a
	// real-signal property.
	rng := uint32(0x2c1f00d)
	nextDibit := func() uint8 {
		rng = rng*1664525 + 1013904223
		return uint8((rng >> 16) & 3)
	}
	in := make([]uint8, 0)
	for i := 0; i < 512; i++ {
		in = append(in, nextDibit())
	}
	in = append(in, phase1.FrameSyncWord[:]...)
	for i := 0; i < 64; i++ {
		in = append(in, nextDibit())
	}
	clean := dibitsToLSMIQ(t, in, sps, PulseSpanSymbols, RolloffAlpha)

	cases := []struct {
		offsetHz float64
		snrDB    float64 // 0 = noiseless
	}{
		{offsetHz: 200},
		{offsetHz: -350},
		{offsetHz: 800},
		{offsetHz: -1200},
		{offsetHz: 2000},
		{offsetHz: -2500},
		{offsetHz: 1000, snrDB: 18}, // offset + noise
	}

	for _, tc := range cases {
		base := demod.ApplyImpairments(clean, sampleRate, demod.Impairments{
			FreqOffsetHz: tc.offsetHz,
			SNRdB:        tc.snrDB,
			Seed:         1,
		})
		locked := 0
		for k := 0; k < sps; k++ {
			var captured []uint8
			r := New(Options{
				SampleRateHz: sampleRate,
				DemodMode:    DemodCQPSK,
				DibitSink:    func(d []uint8, _ int) { captured = append(captured, d...) },
			})
			shifted := base[k:]
			for i := 0; i < len(shifted); i += chunk {
				end := i + chunk
				if end > len(shifted) {
					end = len(shifted)
				}
				r.Process(shifted[i:end])
			}
			det := phase1.NewSyncDetector(2)
			if hits, _ := det.Process(nil, captured, 0); len(hits) > 0 {
				locked++
			}
		}
		if locked < 6 {
			t.Errorf("chunk=%d offset=%.0f Hz snr=%.0f dB: recovered FSW from only %d/%d starting phases; want >= 6 (issue #492 carrier recovery)",
				chunk, tc.offsetHz, tc.snrDB, locked, sps)
		}
	}
}

// TestCQPSKDemodRecoversFSWWithMultipathAndOffset is the issue #492
// "carrier loop rails" regression guard. The earlier offset-sweep fixtures
// inject only a frequency offset into a perfect RRC round-trip (zero ISI),
// so the blind CMA equalizer never adapts — its taps stay at the identity
// centre spike and it injects no phase. On a real DDC-channelised simulcast
// capture the channel carries multipath ISI, so the CMA *does* adapt; being
// phase-blind (its constant-modulus cost is rotation-invariant) its taps
// random-walk in phase, and the downstream Costas loop reads that drift as a
// carrier frequency and rails at its ±baud/8 clamp — control channel never
// locks, every frame lost, exactly as reported on the live captures.
//
// This drives a simulcast-like multipath channel *and* a residual offset in
// production-realistic 192-sample streaming chunks and asserts both that the
// FSW still correlates and that the reported carrier estimate converges near
// the injected offset instead of railing. It must fail on the pre-fix
// CMA→Costas ordering and pass once carrier recovery runs ahead of the
// equalizer (and the CMA's phase null is pinned).
func TestCQPSKDemodRecoversFSWWithMultipathAndOffset(t *testing.T) {
	// Reproduces the confirmed issue #492 failure mode — a near-spectral-null
	// simulcast channel biases the raw-IQ lag-1 coarse seed into a spurious
	// ~750 Hz "offset" that mis-tunes the NCO and rails the Costas loop, even
	// though the CMA→Costas loop alone (no seed) recovers this same signal.
	// This is a fully synthetic stand-in for the live captures on #492, so the
	// seed-robustness fix can be validated without a private recording.
	const sampleRate = 48_000.0
	const sps = 10

	// Scrambled lead-in + FSW + trailer, as in the offset sweep.
	rng := uint32(0x2c1f00d)
	nextDibit := func() uint8 {
		rng = rng*1664525 + 1013904223
		return uint8((rng >> 16) & 3)
	}
	in := make([]uint8, 0)
	for i := 0; i < 2000; i++ {
		in = append(in, nextDibit())
	}
	in = append(in, phase1.FrameSyncWord[:]...)
	for i := 0; i < 2000; i++ {
		in = append(in, nextDibit())
	}
	clean := dibitsToLSMIQ(t, in, sps, PulseSpanSymbols, RolloffAlpha)

	// A near-spectral-null simulcast channel at 10 sps: main path plus a
	// half-symbol (5-sample) echo at ~0.95 amplitude, rotated 70°. Two
	// near-equal-power overlapping transmitters (the simulcast case LSM
	// exists for) produce exactly this deep fade. The strong echo (a) forces
	// the CMA to adapt hard and (b) badly biases the lag-1 coarse seed, which
	// reads the channel's autocorrelation phase as a spurious carrier offset.
	multipath := []complex64{
		complex(1, 0),
		0, 0, 0, 0,
		complex(float32(0.95*math.Cos(70*math.Pi/180)), float32(0.95*math.Sin(70*math.Pi/180))),
	}

	cases := []struct {
		offsetHz float64
		snrDB    float64
	}{
		{offsetHz: 0},
		{offsetHz: 100}, // the reporter's ~100 Hz drift, post-DDC residual
		{offsetHz: -300, snrDB: 20},
	}

	for _, tc := range cases {
		base := demod.ApplyImpairments(clean, sampleRate, demod.Impairments{
			Multipath:    multipath,
			FreqOffsetHz: tc.offsetHz,
			SNRdB:        tc.snrDB,
			Seed:         1,
		})
		locked := 0
		for k := 0; k < sps; k++ {
			var captured []uint8
			r := New(Options{
				SampleRateHz: sampleRate,
				DemodMode:    DemodCQPSK,
				DibitSink:    func(d []uint8, _ int) { captured = append(captured, d...) },
			})
			shifted := base[k:]
			const chunk = 192
			maxCarrier := 0.0
			for i := 0; i < len(shifted); i += chunk {
				end := i + chunk
				if end > len(shifted) {
					end = len(shifted)
				}
				r.Process(shifted[i:end])
				if c := math.Abs(r.CQPSKCarrierOffsetHz()); c > maxCarrier {
					maxCarrier = c
				}
			}
			det := phase1.NewSyncDetector(2)
			hits, _ := det.Process(nil, captured, 0)
			lastCarrier := r.CQPSKCarrierOffsetHz()
			// The fine loop rails at ±baud/8 (±600 Hz at 4800 baud) on top of
			// the coarse seed; a converged loop settles near the injected
			// offset. Treat a starting phase as locked only when the FSW
			// correlated AND the carrier estimate tracked the true offset
			// rather than running away to the clamp.
			converged := math.Abs(lastCarrier-tc.offsetHz) <= 300 && maxCarrier <= math.Abs(tc.offsetHz)+550
			if len(hits) > 0 && converged {
				locked++
			}
		}
		if locked < 6 {
			t.Errorf("offset=%.0f Hz snr=%.0f dB multipath: recovered+converged from only %d/%d starting phases; want >= 6 (issue #492 carrier rails before equalizer)",
				tc.offsetHz, tc.snrDB, locked, sps)
		}
	}
}

// TestParseDemodMode locks down the YAML-string → DemodMode mapping
// shipped via the ccdecoder connector.
func TestParseDemodMode(t *testing.T) {
	cases := []struct {
		in     string
		wantM  DemodMode
		wantOk bool
	}{
		{"", DemodC4FM, true},
		{"c4fm", DemodC4FM, true},
		{"C4FM", DemodC4FM, true},
		{"fm", DemodC4FM, true},
		{"cqpsk", DemodCQPSK, true},
		{"CQPSK", DemodCQPSK, true},
		{"lsm", DemodCQPSK, true},
		{"linear", DemodCQPSK, true},
		{"bogus", DemodC4FM, false},
	}
	for _, tc := range cases {
		got, ok := ParseDemodMode(tc.in)
		if got != tc.wantM || ok != tc.wantOk {
			t.Errorf("ParseDemodMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.wantM, tc.wantOk)
		}
	}
}

// TestCQPSKSymbolSinkAlignedAndClustered verifies the constellation tap:
// the CQPSK path delivers one complex symbol per emitted dibit (aligned
// for client colour-by-dibit), and on a clean LSM stream those symbols
// settle into the four π/4-DQPSK constellation quadrants rather than a
// blob at the origin.
func TestCQPSKSymbolSinkAlignedAndClustered(t *testing.T) {
	const sampleRate = 48_000.0
	const sps = 10
	const symbols = 1200

	rng := uint32(0x1234abcd)
	in := make([]uint8, symbols)
	for i := range in {
		rng = rng*1664525 + 1013904223
		in[i] = uint8((rng >> 16) & 3)
	}
	iq := dibitsToLSMIQ(t, in, sps, PulseSpanSymbols, RolloffAlpha)

	var dibits int
	var syms []complex64
	r := New(Options{
		SampleRateHz: sampleRate,
		DemodMode:    DemodCQPSK,
		DibitSink:    func(d []uint8, _ int) { dibits += len(d) },
		SymbolSink:   func(s []complex64) { syms = append(syms, s...) },
	})
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	if len(syms) == 0 {
		t.Fatal("SymbolSink never fired on the CQPSK path")
	}
	// One symbol per dibit — the alignment a client relies on to colour
	// each constellation point by its decided dibit.
	if len(syms) != dibits {
		t.Fatalf("symbol/dibit count mismatch: %d symbols, %d dibits", len(syms), dibits)
	}

	// After loop acquisition the points should populate all four
	// quadrants (a real constellation), not collapse to the origin.
	const settle = 200
	var quad [4]int
	var nonTrivial int
	for _, s := range syms[min(settle, len(syms)):] {
		if math.Hypot(float64(real(s)), float64(imag(s))) < 0.2 {
			continue
		}
		nonTrivial++
		q := 0
		if real(s) < 0 {
			q |= 1
		}
		if imag(s) < 0 {
			q |= 2
		}
		quad[q]++
	}
	if nonTrivial < symbols/4 {
		t.Fatalf("too few non-trivial symbol points: %d (constellation collapsed to origin)", nonTrivial)
	}
	for q := 0; q < 4; q++ {
		if quad[q] == 0 {
			t.Errorf("constellation quadrant %d empty — symbols did not form 4 clusters (quads=%v)", q, quad)
		}
	}
}

// TestC4FMSymbolSinkNeverFires confirms the complex symbol tap is a
// CQPSK-only contract: the C4FM path's symbol domain is the real soft
// waveform, so SymbolSink must stay uncalled there.
func TestC4FMSymbolSinkNeverFires(t *testing.T) {
	const sampleRate = 48_000.0
	dibits := make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, sampleRate, 1800.0)
	for i := range iq {
		iq[i] *= 100
	}

	symbolCalls := 0
	dibitCalls := 0
	r := New(Options{
		SampleRateHz: sampleRate,
		DeviationHz:  1800.0,
		DemodMode:    DemodC4FM,
		DibitSink:    func(d []uint8, _ int) { dibitCalls += len(d) },
		SymbolSink:   func(s []complex64) { symbolCalls += len(s) },
	})
	r.Process(iq)

	if dibitCalls == 0 {
		t.Fatal("C4FM path produced no dibits — fixture or path broken")
	}
	if symbolCalls != 0 {
		t.Errorf("C4FM SymbolSink fired %d times, want 0 (C4FM has no complex symbol domain)", symbolCalls)
	}
}
