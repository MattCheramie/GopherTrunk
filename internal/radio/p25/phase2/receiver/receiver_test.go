package receiver

import (
	"fmt"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

func TestReceiverConstructsAndProcessesSilence(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(dibits []uint8, baseIdx int) {},
	})
	silence := make([]complex64, 4800)
	for range 4 {
		r.Process(silence)
	}
}

func TestReceiverConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"missing sample rate", Options{DibitSink: func([]uint8, int) {}}},
		{"missing sink", Options{SampleRateHz: 48_000}},
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 10_000, DibitSink: func([]uint8, int) {}}},
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

// makeP2HDQPSKIQ synthesises a P25 Phase 2 H-DQPSK IQ stream by
// walking the phase per-dibit through the standard encoding map
// (with the π/8 rotation offset baked in) and emitting `sps`
// constant complex samples per symbol. The matched filter rounds
// the symbol edges; the naive decimation in the receiver picks
// the centre sample of each symbol.
func makeP2HDQPSKIQ(dibits []uint8) ([]complex64, int) {
	const sampleRate = 48_000.0
	const sps = int(sampleRate / SymbolRate) // 8

	// Encoding map for π/8-rotated H-DQPSK (matches the PiOver4DQPSK
	// helper's decoder when rotation = π/8 is configured): each
	// dibit contributes a phase delta which when reduced by π/8
	// lands in one of the four quadrants.
	deltas := map[uint8]float64{
		0b00: math.Pi/8 + 0,
		0b01: math.Pi/8 + math.Pi/2,
		0b10: math.Pi/8 + math.Pi,
		0b11: math.Pi/8 - math.Pi/2,
	}

	iq := make([]complex64, len(dibits)*sps)
	phase := 0.0
	for d, dibit := range dibits {
		phase += deltas[dibit]
		c := complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
		for k := 0; k < sps; k++ {
			iq[d*sps+k] = c
		}
	}
	return iq, sps
}

// makeP2HDQPSKIQWithOffset is stdModulateHDQPSK plus a constant carrier
// rotation of offsetHz: every emitted sample n is additionally rotated
// by e^{+j·2π·offsetHz·n/fs}, modelling the residual frequency error a
// real (non-TCXO) RTL-SDR tuner carries. A differential decoder removes
// constant carrier *phase* but not this per-symbol *rotation*
// (2π·offsetHz/SymbolRate), so without carrier recovery the dibits land
// a quadrant over and the 20-dibit outbound sync never correlates —
// exactly the issue #813 field symptom (superframes=0). offsetHz=0 is
// byte-identical to stdModulateHDQPSK.
func makeP2HDQPSKIQWithOffset(dibits []uint8, offsetHz, fs float64) ([]complex64, int) {
	sps := int(fs/SymbolRate + 0.5)
	// STANDARD π/4-DQPSK-family modulation (what a real base station transmits),
	// deliberately not GopherTrunk's own (2↔3-transposed) modulator, so the IQ
	// round-trips through the receiver's canonicalising dibit remap (issue #813).
	iq := demod.ModulateHDQPSKSpec(dibits, sps, PulseSpanSymbols, RolloffAlpha)
	if offsetHz != 0 {
		w := 2 * math.Pi * offsetHz / fs
		for n := range iq {
			ph := w * float64(n)
			iq[n] *= complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		}
	}
	return iq, sps
}

// pseudoRandomDibits returns n deterministic pseudo-random dibits (0..3)
// from an xorshift generator. Deterministic so the test is reproducible;
// the varying symbol-to-symbol transitions give the Gardner timing loop
// something to lock to (an all-idle stream is a transition-free tone).
func pseudoRandomDibits(n int, seed uint32) []uint8 {
	out := make([]uint8, n)
	x := seed | 1
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = uint8(x & 0x3)
	}
	return out
}

// bestTailSER aligns the decoded dibit stream against the transmitted one
// (the receiver drops the differential decoder's first symbol and the
// Gardner warmup, so rx lags tx by a small constant) and returns the
// minimum symbol-error rate over the settled tail (rx index >= skip). A
// healthy decode settles to ~0; an uncorrected carrier offset throws ~3/4
// of the symbols a quadrant over.
func bestTailSER(tx, rx []uint8, skip int) float64 {
	if len(rx) <= skip+100 {
		return 1.0
	}
	best := 1.0
	// The combined TX+RX RRC group delay makes the decoded stream lead tx
	// by ~2*span symbols, so the alignment lag is negative; search a window
	// wide enough to cover the group delay in either direction.
	for lag := -64; lag <= 64; lag++ {
		errs, n := 0, 0
		for i := skip; i < len(rx); i++ {
			j := i + lag
			if j < 0 {
				continue
			}
			if j >= len(tx) {
				break
			}
			if rx[i] != tx[j] {
				errs++
			}
			n++
		}
		if n < 100 {
			continue
		}
		if r := float64(errs) / float64(n); r < best {
			best = r
		}
	}
	return best
}

// TestReceiverDecodesUnderCarrierOffset is the issue #813 failing-first
// regression at the dibit level: a real-tuner carrier offset must not stop
// the H-DQPSK receiver from recovering the symbols. Offsets span inside
// (±600 Hz) and outside (±1500 Hz) the fine Costas loop's ±SymbolRate/8 =
// ±750 Hz pull-in, so passing proves the coarse carrier seed — not just
// the fine loop — is doing its job. Without carrier recovery the settled
// tail SER is ~0.75; with it, ~0.
func TestReceiverDecodesUnderCarrierOffset(t *testing.T) {
	const fs = 48_000.0
	tx := pseudoRandomDibits(2400, 0xC0FFEE)
	for _, off := range []float64{600, -600, 1500, -1500} {
		t.Run(fmt.Sprintf("offset=%gHz", off), func(t *testing.T) {
			var rx []uint8
			r := New(Options{
				SampleRateHz: fs,
				ClockMode:    ClockGardner,
				GardnerGain:  0.005, // the value the voice/CC paths use
				DibitSink:    func(d []uint8, _ int) { rx = append(rx, d...) },
			})
			iq, _ := makeP2HDQPSKIQWithOffset(tx, off, fs)
			// Feed in production-sized chunks so the coarse seed must
			// accumulate across calls (a single giant chunk would mask the
			// streaming accumulate-then-fire path).
			for i := 0; i < len(iq); i += 192 {
				end := i + 192
				if end > len(iq) {
					end = len(iq)
				}
				r.Process(iq[i:end])
			}
			// skip past the coarse-seed fire (~512 symbols) + loop lock.
			ser := bestTailSER(tx, rx, 900)
			if ser > 0.02 {
				t.Errorf("offset %g Hz: tail symbol-error rate = %.3f, want <= 0.02 (issue #813: no carrier recovery)", off, ser)
			}
		})
	}
}

// TestReceiverLocksSuperframeUnderOffset is the symptom-level failing-first
// regression: drive a real outbound-sync-bearing superframe stream through
// the receiver into a SuperframeDecoder under a carrier offset and assert at
// least one superframe locks. This reproduces the reporter's superframes=0
// (67/67) directly. Sub-frame bodies are pseudo-random (not idle) so the
// Gardner loop has transitions to track; only the sync lock is asserted, not
// slot classification. The lead-in exceeds the coarse-seed sample budget so
// the seed has fired before the first sync arrives.
func TestReceiverLocksSuperframeUnderOffset(t *testing.T) {
	const fs = 48_000.0
	// Build 4 superframes of random sub-frame bodies; EncodeSuperframe
	// injects the 20-dibit outbound sync at sub-frame 0.
	var subs [phase2.SubframesPerSuperframe][]uint8
	stream := make([]uint8, 0, 800+4*phase2.DibitsPerSuperframe)
	stream = append(stream, pseudoRandomDibits(800, 0xBEEF)...) // lead-in (> seed budget)
	for sfi := 0; sfi < 4; sfi++ {
		for i := range subs {
			subs[i] = pseudoRandomDibits(phase2.DibitsPerSubframe, uint32(0x1000+sfi*16+i))
		}
		stream = append(stream, phase2.EncodeSuperframe(subs)...)
	}

	for _, off := range []float64{0, 600, -600, 1500, -1500} {
		t.Run(fmt.Sprintf("offset=%gHz", off), func(t *testing.T) {
			locked := 0
			sfDec := phase2.NewSuperframeDecoder()
			r := New(Options{
				SampleRateHz: fs,
				ClockMode:    ClockGardner,
				GardnerGain:  0.005, // the value the voice/CC paths use
				DibitSink: func(d []uint8, baseIdx int) {
					locked += len(sfDec.Process(d, baseIdx))
				},
			})
			iq, _ := makeP2HDQPSKIQWithOffset(stream, off, fs)
			for i := 0; i < len(iq); i += 192 {
				end := i + 192
				if end > len(iq) {
					end = len(iq)
				}
				r.Process(iq[i:end])
			}
			if locked == 0 {
				t.Errorf("offset %g Hz: superframes locked = 0, want >= 1 (issue #813)", off)
			}
		})
	}
}

func TestReceiverEmitsDibitsFromHDQPSK(t *testing.T) {
	dibits := []uint8{
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b11, 0b10, 0b01, 0b00, 0b11, 0b10, 0b01, 0b00,
	}
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(d []uint8, baseIdx int) { batches++ },
	})
	iq, _ := makeP2HDQPSKIQ(dibits)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("DibitSink received zero batches; the chain produced no symbols")
	}
}

func TestReceiverDibitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(d []uint8, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(d))
		},
	})

	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11}
	iq, _ := makeP2HDQPSKIQ(dibits)
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

func TestReceiverEmittedDibitsAreValid(t *testing.T) {
	var bad int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(d []uint8, baseIdx int) {
			for _, v := range d {
				if v > 3 {
					bad++
				}
			}
		},
	})
	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11}
	iq, _ := makeP2HDQPSKIQ(dibits)
	r.Process(iq)
	if bad > 0 {
		t.Errorf("%d dibit(s) outside 0..3 range", bad)
	}
}

// TestParseClockMode covers the config-string → ClockMode mapping
// the ccdecoder connector uses to translate the
// `p25_phase2_clock_mode` YAML field into Options.ClockMode.
func TestParseClockMode(t *testing.T) {
	cases := []struct {
		in   string
		want ClockMode
		ok   bool
	}{
		{"", ClockGardner, true},
		{"gardner", ClockGardner, true},
		{"Gardner", ClockGardner, true},
		{"on", ClockGardner, true},
		{"true", ClockGardner, true},
		{"1", ClockGardner, true},
		{" gardner ", ClockGardner, true},
		{"naive", ClockNaive, true},
		{"NAIVE", ClockNaive, true},
		{"off", ClockNaive, true},
		{"false", ClockNaive, true},
		{"0", ClockNaive, true},
		{"nonsense", ClockGardner, false},
	}
	for _, tc := range cases {
		got, ok := ParseClockMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseClockMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestParseSoftDecision pins the config-string contract the ccdecoder /
// widebandt2 pipelines rely on to turn p25_phase2_soft_decision into
// Options.SoftDecision (issue #915): empty / off-family → false (the default
// hard slicer), on-family → true, anything else → (false, ok=false) so the
// caller warns and falls back.
func TestParseSoftDecision(t *testing.T) {
	cases := []struct {
		in   string
		want bool
		ok   bool
	}{
		{"", false, true},
		{"off", false, true},
		{"OFF", false, true},
		{"false", false, true},
		{"0", false, true},
		{" off ", false, true},
		{"on", true, true},
		{"On", true, true},
		{"true", true, true},
		{"1", true, true},
		{" on ", true, true},
		{"nonsense", false, false},
	}
	for _, tc := range cases {
		got, ok := ParseSoftDecision(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseSoftDecision(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
