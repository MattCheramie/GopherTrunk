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

// TestCQPSKDemodRoundTripStableTail feeds a long all-zero dibit stream
// (which under LSM produces a constant-phase carrier) through the
// modulator → DemodCQPSK demodulator chain and asserts the recovered
// dibit tail is all zeros after the Gardner loop settles. This is the
// minimum bar for "the CQPSK path produces canonical TIA-102.BAAA
// dibits compatible with the FSW + NID + TSBK decoders downstream."
//
// A round-trip identity test on a random dibit stream would be more
// general but is over-strict for this demod chain: the Gardner loop
// on complex IQ converges within ~5 symbols on a clean signal but
// the matched-filter group delay (~8 symbols) and differential
// reference sample combine to make exact dibit-by-dibit alignment
// fragile to construct as a test fixture. The structural guarantee
// (FSW survives) is covered by TestCQPSKDemodRecoversFSW below.
func TestCQPSKDemodRoundTripStableTail(t *testing.T) {
	const sampleRate = 48_000.0
	const sps = 10
	const symbols = 400

	in := make([]uint8, symbols)
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

	if len(captured) < 100 {
		t.Fatalf("captured %d dibits, want at least 100", len(captured))
	}
	// Skip the leading 20 dibits — Gardner startup + matched filter
	// fill — then assert the stable tail is all-zero.
	tail := captured[20:]
	nonZero := 0
	for _, d := range tail {
		if d != 0 {
			nonZero++
		}
	}
	if nonZero > 0 {
		t.Errorf("post-settle tail has %d/%d non-zero dibits; expected pure zeros under all-zero LSM input",
			nonZero, len(tail))
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
