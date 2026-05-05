package channelizer

import (
	"math"
	"testing"
)

// TestToneInChannel feeds a complex exponential at +k*Fs/M and verifies that
// channel k receives the most power, with neighbors strongly attenuated.
func TestToneInChannel(t *testing.T) {
	const M = 8
	const tapsPer = 16
	ch := New(M, tapsPer, 9)

	const N = 4096
	in := make([]complex64, N)
	const targetCh = 3
	for i := 0; i < N; i++ {
		theta := 2 * math.Pi * float64(targetCh) * float64(i) / float64(M)
		in[i] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	}

	out := ch.Process(nil, in)
	skip := tapsPer * 2 // skip transient
	power := make([]float64, M)
	for c := 0; c < M; c++ {
		for i := skip; i < len(out[c]); i++ {
			s := out[c][i]
			power[c] += float64(real(s))*float64(real(s)) + float64(imag(s))*float64(imag(s))
		}
	}

	maxIdx := 0
	for c, p := range power {
		if p > power[maxIdx] {
			maxIdx = c
		}
	}
	if maxIdx != targetCh {
		t.Fatalf("peak in channel %d, want %d (powers=%v)", maxIdx, targetCh, power)
	}
	// Adjacent-channel rejection should be reasonable (>20 dB).
	for c := 0; c < M; c++ {
		if c == targetCh {
			continue
		}
		ratio := power[targetCh] / (power[c] + 1e-30)
		if ratio < 100 { // 20 dB
			t.Errorf("channel %d leakage: ratio = %.1f, want >= 100", c, ratio)
		}
	}
}

func TestProcessDecimatesByM(t *testing.T) {
	const M = 4
	ch := New(M, 8, 8.6)
	const N = 1024
	in := make([]complex64, N)
	out := ch.Process(nil, in)
	for c := 0; c < M; c++ {
		if got, want := len(out[c]), N/M; got != want {
			t.Errorf("channel %d: len = %d, want %d", c, got, want)
		}
	}
}
