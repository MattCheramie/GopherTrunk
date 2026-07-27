package receiver_test

// End-to-end equalizer test for issue #915: modulate real-shaped P25 Phase 2
// superframes carrying a MAC_PTT (ALGID/KID) payload, pass them through a
// multipath ISI channel, and drive them through the production receiver with
// the CMA equalizer off vs on. The blind constant-modulus equalizer must
// recover MAC payloads the un-equalized path loses to inter-symbol
// interference, and must stay byte-identical on a clean (ISI-free) signal.
//
// This is the receiver-level counterpart to the primitive's ISI tests in
// internal/dsp/sync/equalizer_test.go — it exercises the actual demod chain
// (NCO seed → matched filter → AGC → Gardner → Costas → CMA → differential
// decode), so it verifies the wiring, not just the filter.

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
)

// eqMultipath convolves the IQ with a fixed sparse multipath channel — a main
// tap plus a delayed echo a few samples out — the ISI a real RF channel /
// front-end imposes and a linear equalizer inverts. Operates at the sample
// rate (echo delay in samples), so it smears energy across symbol boundaries.
func eqMultipath(iq []complex64, echo complex64, echoDelay int) []complex64 {
	out := make([]complex64, len(iq))
	for n := range iq {
		v := iq[n]
		if n-echoDelay >= 0 {
			v += echo * iq[n-echoDelay]
		}
		out[n] = v
	}
	return out
}

func eqDecode(iq []complex64, equalize bool) int {
	sfDec := phase2.NewSuperframeDecoder()
	var sfs []phase2.Superframe
	opts := rx.Options{
		SampleRateHz: 8 * rx.SymbolRate,
		ClockMode:    rx.ClockGardner,
		GardnerGain:  0.005,
		Equalizer:    equalize,
	}
	opts.DibitSink = func(d []uint8, base int) {
		sfs = append(sfs, sfDec.Process(d, base)...)
	}
	r := rx.New(opts)
	for off := 0; off < len(iq); off += 192 {
		end := off + 192
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[off:end])
	}
	cfg := phase2.MACDecodeConfig{Trellis: phase2.TrellisOn}
	got := 0
	for _, sf := range sfs {
		for _, tagged := range phase2.DecodeSuperframeMACPDUsWithSlot(sf, cfg) {
			if tagged.SlotType != phase2.SlotTypeMACPTT {
				continue
			}
			if es, ok := tagged.PDU.AsPushToTalk(); ok &&
				int(es.AlgorithmID) == sdWantAlg && int(es.KeyID) == sdWantKey {
				got++
			}
		}
	}
	return got
}

// TestParseEqualizer covers the config-string → bool mapping.
func TestParseEqualizer(t *testing.T) {
	cases := []struct {
		in       string
		want, ok bool
	}{
		{"", false, true}, {"off", false, true}, {"false", false, true}, {"0", false, true},
		{"on", true, true}, {"On", true, true}, {"true", true, true}, {"1", true, true},
		{"cma", true, true}, {" cma ", true, true},
		{"yes", false, false}, {"lms", false, false},
	}
	for _, c := range cases {
		got, ok := rx.ParseEqualizer(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseEqualizer(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestEqualizerNoiseless: on a clean ISI-free signal the CMA equalizer is
// transparent — it recovers exactly what the un-equalized path does.
func TestEqualizerNoiseless(t *testing.T) {
	iq := sdStream(t, 6)
	plain := eqDecode(iq, false)
	equal := eqDecode(iq, true)
	if plain == 0 {
		t.Fatalf("un-equalized path recovered 0 payloads on a clean signal")
	}
	if equal != plain {
		t.Fatalf("clean signal: equalized recovered %d, un-equalized %d — CMA must be transparent", equal, plain)
	}
}

// TestEqualizerBeatsPlainOnMultipath: through a multipath ISI channel the CMA
// equalizer recovers strictly more MAC payloads than the un-equalized path,
// and never fewer.
func TestEqualizerBeatsPlainOnMultipath(t *testing.T) {
	base := sdStream(t, 40)
	// A strong echo ~5 samples (~0.6 symbol at 8 sps) after the main tap:
	// enough ISI to knock out a chunk of bursts without the equalizer.
	totalPlain, totalEqual := 0, 0
	for _, ch := range []struct {
		echo  complex64
		delay int
	}{
		{complex(0.45, 0.10), 5},
		{complex(0.55, -0.15), 6},
		{complex(0.38, 0.22), 4},
	} {
		iq := eqMultipath(base, ch.echo, ch.delay)
		p := eqDecode(iq, false)
		e := eqDecode(iq, true)
		t.Logf("echo=%v delay=%d  plain=%d  equalized=%d  (of 40 superframes)", ch.echo, ch.delay, p, e)
		if e < p {
			t.Errorf("echo=%v: equalized %d recovered FEWER than plain %d", ch.echo, e, p)
		}
		totalPlain += p
		totalEqual += e
	}
	t.Logf("TOTAL plain=%d equalized=%d", totalPlain, totalEqual)
	if totalEqual <= totalPlain {
		t.Fatalf("CMA equalizer did not improve recovery through multipath (plain=%d equalized=%d)", totalPlain, totalEqual)
	}
}
