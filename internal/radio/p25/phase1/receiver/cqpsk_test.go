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
		if locked < 6 {
			t.Errorf("offset=%.0f Hz snr=%.0f dB: recovered FSW from only %d/%d starting phases; want >= 6 (issue #492 carrier recovery)",
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
