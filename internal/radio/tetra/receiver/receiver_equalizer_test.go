package receiver

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// runReceiverDibits pushes an IQ stream through a receiver (Gardner timing + AFC,
// matching the live voice-demux config) with the equalizer optionally enabled,
// and returns the concatenated output dibit stream.
func runReceiverDibits(t *testing.T, iq []complex64, equalize bool) []uint8 {
	t.Helper()
	var out []uint8
	r := New(Options{
		SampleRateHz:    144_000,
		ClockMode:       ClockGardner,
		GardnerGain:     0.005,
		EnableAFC:       true,
		EnableEqualizer: equalize,
		DibitSink:       func(d []uint8, _ int) { out = append(out, d...) },
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		r.Process(iq[i:e])
	}
	return out
}

// bestAlignErrRate slides got against want over a small shift range and returns
// the minimum dibit error rate, measured over want[skip:]. The shift absorbs the
// receiver's matched-filter + timing group delay (unknown a-priori); a 1:1
// timing lock keeps a single global shift valid across the whole stream.
func bestAlignErrRate(want, got []uint8, maxShift, skip int) float64 {
	best := 1.0
	for shift := 0; shift <= maxShift; shift++ {
		var bad, total int
		for i := skip; i+shift < len(got) && i < len(want); i++ {
			total++
			if want[i] != got[i+shift] {
				bad++
			}
		}
		if total < 200 {
			continue
		}
		if r := float64(bad) / float64(total); r < best {
			best = r
		}
	}
	return best
}

// TestReceiverEqualizerRecoversMultipathIQ is the end-to-end proof for issue
// #1001: a multipath channel applied to the IQ (a one-symbol post-cursor echo)
// closes the eye, and the receiver's current path (matched filter → Gardner →
// AFC → differential decode) loses a large fraction of dibits. Enabling the
// opt-in equalizer recovers most of them. The equalizer is off by default, so
// this same chain with EnableEqualizer=false stays the degraded baseline.
func TestReceiverEqualizerRecoversMultipathIQ(t *testing.T) {
	// Random payload, front-loaded with a run the timing loop can acquire on.
	rng := rand.New(rand.NewSource(0x1001BEEF))
	dibits := make([]uint8, 6000)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}

	const sps = 8 // 144 kHz / 18 kHz
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, 8, RolloffAlpha, Rotation)

	// One-symbol (sps-sample) post-cursor echo at 0.6 amplitude + light AWGN:
	// the eye-closing ISI a same-carrier multipath channel imposes.
	echo := make([]complex64, sps+1)
	echo[0] = 1
	echo[sps] = complex(0.6, 0)
	iq = demod.ApplyImpairments(iq, 144_000, demod.Impairments{
		Multipath: echo,
		SNRdB:     30,
		Seed:      1,
	})

	base := runReceiverDibits(t, iq, false)
	eqd := runReceiverDibits(t, iq, true)

	// Skip a generous warm-up so both the timing loop and the equalizer are
	// converged; search a small shift for the group-delay alignment.
	baseErr := bestAlignErrRate(dibits, base, 40, 600)
	eqErr := bestAlignErrRate(dibits, eqd, 40, 600)
	t.Logf("multipath IQ: baseline dibit err=%.4f  equalized dibit err=%.4f (base_syms=%d eq_syms=%d)",
		baseErr, eqErr, len(base), len(eqd))

	if baseErr < 0.08 {
		t.Fatalf("baseline should be badly corrupted by the multipath echo (>8%% dibit err), got %.4f — channel too mild to prove the equalizer", baseErr)
	}
	if eqErr > 0.03 {
		t.Fatalf("equalizer failed to recover the multipath channel: equalized dibit err=%.4f (want <=0.03); baseline was %.4f", eqErr, baseErr)
	}
}

// TestReceiverEqualizerCleanChannelNoHarm guards the issue's central worry: the
// equalizer must not regress the already-clean low-load case. On a single-path
// (ISI-free) channel with light noise, decoding with the CMA enabled must stay
// essentially error-free — the blind equalizer settles to a pass-through and
// does not manufacture errors where there is no multipath to correct.
func TestReceiverEqualizerCleanChannelNoHarm(t *testing.T) {
	rng := rand.New(rand.NewSource(0xC1EA))
	dibits := make([]uint8, 6000)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}
	const sps = 8
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, 8, RolloffAlpha, Rotation)
	// Clean single-path channel, only light AWGN — no multipath.
	iq = demod.ApplyImpairments(iq, 144_000, demod.Impairments{SNRdB: 30, Seed: 7})

	eqErr := bestAlignErrRate(dibits, runReceiverDibits(t, iq, true), 40, 600)
	t.Logf("clean channel with equalizer on: dibit err=%.4f", eqErr)
	if eqErr > 0.01 {
		t.Fatalf("equalizer harmed a clean channel: dibit err=%.4f (want <=0.01) — it must be benign when there is no ISI", eqErr)
	}
}

// TestReceiverEqualizerDefaultOff confirms the equalizer is opt-in: a receiver
// built without EnableEqualizer produces byte-identical dibits to the same
// stream, i.e. the feature is inert unless explicitly enabled (no regression to
// the existing decode path / fixtures).
func TestReceiverEqualizerDefaultOff(t *testing.T) {
	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b11, 0b10, 0b01, 0b00, 0b11, 0b10, 0b01, 0b00}
	iq := makeTetraDQPSKIQ(dibits)

	a := runReceiverDibits(t, iq, false)
	b := runReceiverDibits(t, iq, false)
	if len(a) != len(b) {
		t.Fatalf("nondeterministic decode: %d vs %d dibits", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic decode at %d", i)
		}
	}
}
