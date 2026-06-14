package siglab

import (
	"bytes"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// placeCarrier upsamples a baseband capture by U and shifts it so the signal
// (originally at DC) lands at placeHz in the wider band, scaled by amp. It is
// the test rig for building a wideband multi-carrier capture out of the
// narrowband synthesizer's output.
func placeCarrier(iq []complex64, fromRate float64, u int, placeHz, amp float64) []complex64 {
	rs := dsp.NewResampler(u, 1, 16, 8.0)
	wide := rs.Process(nil, iq)
	wideRate := fromRate * float64(u)
	// NCO.Mix shifts +offset → DC, so an offset of -placeHz moves DC → placeHz.
	nco := dsp.NewNCO(-placeHz, wideRate)
	wide = nco.Mix(wide, wide)
	a := complex(float32(amp), 0)
	for i := range wide {
		wide[i] *= a
	}
	return wide
}

// fmInterferer is an FM-modulated tone centered at centerHz — a stand-in for an
// analog voice carrier: spectrally broad (so its peak-bin power is comparable
// to a digital carrier, not a needle like a CW tone) and impossible to lock as
// a trunked protocol. Used as the LOUD, dominant carrier a single-dominant
// auto-tune would wrongly latch onto.
func fmInterferer(n int, wideRate, centerHz, devHz, modHz, amp float64) []complex64 {
	out := make([]complex64, n)
	phase := 0.0
	for i := range out {
		t := float64(i) / wideRate
		f := centerHz + devHz*math.Sin(2*math.Pi*modHz*t)
		phase += 2 * math.Pi * f / wideRate
		out[i] = complex(float32(amp*math.Cos(phase)), float32(amp*math.Sin(phase)))
	}
	return out
}

// wideband2Carrier builds a wideband capture with a clean P25 control channel
// at p25Hz (amp 1.0) and a louder FM interferer at fmHz, returning the IQ and
// its sample rate. The interferer is the dominant carrier.
func wideband2Carrier(t *testing.T, p25Hz, fmHz float64) ([]complex64, float64) {
	t.Helper()
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	const u = 4
	wideRate := meta.SampleRateHz * u
	p25 := placeCarrier(iq, meta.SampleRateHz, u, p25Hz, 1.0)
	fm := fmInterferer(len(p25), wideRate, fmHz, 5000, 300, 4.0)
	out := make([]complex64, len(p25))
	for i := range out {
		out[i] = p25[i] + fm[i]
	}
	return out, wideRate
}

// TestRunReaderAutoTuneMultiPicksNonDominantCarrier proves the multi-candidate
// auto-tune locks an off-centre P25 control channel that is NOT the loudest
// carrier — and that the single dominant-carrier auto-tune misses it (latching
// the louder FM interferer instead). This is the synthetic of the real-capture
// case where the control channel sat at -625 kHz under a louder neighbour.
func TestRunReaderAutoTuneMultiPicksNonDominantCarrier(t *testing.T) {
	const p25Hz, fmHz = -30_000.0, 40_000.0
	iq, wideRate := wideband2Carrier(t, p25Hz, fmHz)
	data := EncodeCapture(iq, FormatF32)
	cfg := Config{
		Protocol:      trunking.ProtocolP25,
		SystemName:    "autotune-multi-test",
		SampleRateHz:  wideRate,
		Format:        FormatF32,
		CollectIQDiag: true,
	}

	// Single dominant-carrier auto-tune latches the louder FM carrier → no lock.
	single, err := RunReader(bytes.NewReader(data), "two-carrier.cfile", withAutoTune(cfg))
	if err != nil {
		t.Fatalf("single RunReader: %v", err)
	}
	if single.Locked {
		t.Errorf("single dominant auto-tune unexpectedly locked at tune=%.0f Hz (should miss the non-dominant CC)", single.TuneHz)
	}

	// Multi-candidate auto-tune tries the ranked carriers and locks the P25 CC.
	multi, err := RunReaderAutoTuneMulti(bytes.NewReader(data), "two-carrier.cfile", cfg, 0)
	if err != nil {
		t.Fatalf("RunReaderAutoTuneMulti: %v", err)
	}
	if !multi.Locked {
		t.Fatalf("multi-candidate auto-tune failed to lock the off-centre P25 CC (tune=%.0f Hz)", multi.TuneHz)
	}
	if math.Abs(multi.TuneHz-p25Hz) > 2000 {
		t.Errorf("locked at tune=%.0f Hz, want ≈ %.0f Hz (the P25 carrier)", multi.TuneHz, p25Hz)
	}
}

// TestIdentifyAutoTunePicksNonDominantCarrier proves identify, under auto-tune,
// names P25 at the off-centre control-channel carrier rather than stopping at
// the louder FM carrier (which identifies as nothing trunked).
func TestIdentifyAutoTunePicksNonDominantCarrier(t *testing.T) {
	const p25Hz, fmHz = -30_000.0, 40_000.0
	iq, wideRate := wideband2Carrier(t, p25Hz, fmHz)
	data := EncodeCapture(iq, FormatF32)

	idr, err := IdentifyReader(bytes.NewReader(data), "two-carrier.cfile", IdentifyConfig{
		SampleRateHz: wideRate,
		Format:       FormatF32,
		AutoTune:     true,
	})
	if err != nil {
		t.Fatalf("IdentifyReader: %v", err)
	}
	if idr.Winner != "p25" {
		t.Errorf("winner = %q (conf %.2f), want p25", idr.Winner, idr.Confidence)
	}
	if idr.Inconclusive {
		t.Errorf("identification flagged inconclusive (conf %.2f)", idr.Confidence)
	}
	if math.Abs(idr.TuneHz-p25Hz) > 2000 {
		t.Errorf("identify TuneHz = %.0f Hz, want ≈ %.0f Hz", idr.TuneHz, p25Hz)
	}
}

// withAutoTune returns a copy of cfg with single dominant-carrier auto-tune on.
func withAutoTune(cfg Config) Config {
	cfg.AutoTune = true
	return cfg
}
