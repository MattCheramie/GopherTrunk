package survey

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

const testRateHz = 48_000.0

// --- synthesis helpers (deterministic) ---

func randDibits(n int, seed int64) []uint8 {
	r := rand.New(rand.NewSource(seed))
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(r.Intn(4))
	}
	return out
}

func randBits(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.Intn(2))
	}
	return out
}

// lowpassNoise builds a deterministic voice-like message: white noise smoothed
// by a moving average so its spectrum rolls off like speech (no discrete line).
func lowpassNoise(n int, seed int64) []float64 {
	r := rand.New(rand.NewSource(seed))
	raw := make([]float64, n)
	for i := range raw {
		raw[i] = r.NormFloat64()
	}
	const win = 12 // ~moving average → low-pass
	out := make([]float64, n)
	var acc float64
	for i := 0; i < n; i++ {
		acc += raw[i]
		if i >= win {
			acc -= raw[i-win]
		}
		out[i] = acc / win
	}
	// Normalise to unit peak.
	var peak float64
	for _, v := range out {
		if a := math.Abs(v); a > peak {
			peak = a
		}
	}
	if peak > 0 {
		for i := range out {
			out[i] /= peak
		}
	}
	return out
}

// fmModulate frequency-modulates a real message into constant-envelope IQ.
func fmModulate(msg []float64, rateHz, deviationHz float64) []complex64 {
	out := make([]complex64, len(msg))
	var phase float64
	k := 2 * math.Pi * deviationHz / rateHz
	for i, m := range msg {
		phase += k * m
		out[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return out
}

// amModulate builds a (DC-centred) AM signal: real-valued (1 + depth·msg).
func amModulate(msg []float64, depth float64) []complex64 {
	out := make([]complex64, len(msg))
	for i, m := range msg {
		out[i] = complex(float32(1+depth*m), 0)
	}
	return out
}

func TestClassify(t *testing.T) {
	const nSamples = 24_000 // 0.5 s at 48 kHz

	tests := []struct {
		name    string
		iq      []complex64
		want    SignalClass // exact class, or "" to skip the exact check
		digital bool        // assert IsDigital(class) == digital
		baud    float64     // if > 0, assert detected baud within 8%
	}{
		{
			// 4-level C4FM reads as a digital FM-family carrier; siglab makes
			// the precise "P25 C4FM" call downstream.
			name:    "p25 c4fm",
			iq:      demod.ModulateP25C4FM(randDibits(nSamples/10, 1), testRateHz, 1800),
			digital: true,
		},
		{
			name:    "gfsk 4800 baud",
			iq:      demod.ModulateGFSK(randBits(nSamples/10, 2), 10, 4, 0.5, testRateHz, 2400),
			want:    ClassFSK,
			digital: true,
			baud:    4800,
		},
		{
			// A paging-baud FSK carrier; the router proves "paging" by decoding.
			name:    "gfsk 1200 baud",
			iq:      demod.ModulateGFSK(randBits(nSamples/40, 3), 40, 4, 0.5, testRateHz, 2400),
			want:    ClassFSK,
			digital: true,
			baud:    1200,
		},
		{
			// FSK degraded by AWGN + IQ imbalance still reads as digital — its
			// cyclostationary baud line survives noise, and the IQ-imbalance
			// envelope wobble doesn't trip the AM gate (digital is checked first).
			name: "gfsk impaired",
			iq: demod.ApplyImpairments(
				demod.ModulateGFSK(randBits(nSamples/10, 6), 10, 4, 0.5, testRateHz, 2400),
				testRateHz,
				demod.Impairments{SNRdB: 30, IQGainImbalance: 1.05, IQPhaseSkewRad: 0.03, Seed: 9},
			),
			digital: true,
		},
		{
			// π/4-DQPSK (the dominant PSK in this domain) reads as digital.
			name:    "pi/4-dqpsk",
			iq:      demod.ModulatePiOver4DQPSK(randDibits(nSamples/10, 7), 10, 8, 0.2, math.Pi/4),
			digital: true,
		},
		{
			name: "analog fm voice",
			iq:   fmModulate(lowpassNoise(nSamples, 4), testRateHz, 3000),
			want: ClassNBFM,
		},
		{
			name: "am voice",
			iq:   amModulate(lowpassNoise(nSamples, 5), 0.95),
			want: ClassAM,
		},
		{
			name: "silence → unknown",
			iq:   make([]complex64, nSamples),
			want: ClassUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.iq, testRateHz)
			if tc.want != "" && got.Class != tc.want {
				t.Errorf("Classify() = %q (conf %.2f), want %q\n  features: %+v",
					got.Class, got.Confidence, tc.want, got.Features)
			}
			if IsDigital(got.Class) != tc.digital {
				t.Errorf("IsDigital(%q) = %v, want %v\n  features: %+v",
					got.Class, IsDigital(got.Class), tc.digital, got.Features)
			}
			if tc.baud > 0 {
				if dev := math.Abs(got.Features.BaudHz-tc.baud) / tc.baud; dev > 0.08 {
					t.Errorf("BaudHz = %.1f, want ≈%.0f (%.1f%% off)", got.Features.BaudHz, tc.baud, dev*100)
				}
			}
		})
	}
}

// TestClassifyDegradesGracefully checks that a heavily-impaired C4FM carrier
// (whose fragile 4-level baud line the blind classifier can lose under noise)
// is never confidently mis-fired as AM — it falls back to analog FM or unknown,
// the safe outcome, leaving the authoritative call to siglab.
func TestClassifyDegradesGracefully(t *testing.T) {
	const nSamples = 24_000
	iq := demod.ApplyImpairments(
		demod.ModulateP25C4FM(randDibits(nSamples/10, 11), testRateHz, 1800),
		testRateHz,
		demod.Impairments{SNRdB: 12, IQGainImbalance: 1.08, IQPhaseSkewRad: 0.05, Seed: 3},
	)
	got := Classify(iq, testRateHz)
	if got.Class == ClassAM {
		t.Errorf("heavily-impaired C4FM mis-fired as AM; features: %+v", got.Features)
	}
}

func TestClassifyShortInputIsUnknown(t *testing.T) {
	got := Classify(make([]complex64, 100), testRateHz)
	if got.Class != ClassUnknown {
		t.Errorf("short input: got %q, want unknown", got.Class)
	}
}

func TestIsPagingBaud(t *testing.T) {
	for _, b := range []float64{512, 1200, 1600, 2400, 1180, 2450} {
		if !IsPagingBaud(b) {
			t.Errorf("IsPagingBaud(%g) = false, want true", b)
		}
	}
	for _, b := range []float64{4800, 9600, 800} {
		if IsPagingBaud(b) {
			t.Errorf("IsPagingBaud(%g) = true, want false", b)
		}
	}
}
