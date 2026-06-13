package hunt

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fmNoise builds a constant-envelope analog-FM carrier modulated by deterministic
// voice-like (low-pass noise) audio — a stand-in for a conventional FM channel.
func fmNoise(n int, rateHz, deviationHz float64, seed int64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	raw := make([]float64, n)
	for i := range raw {
		raw[i] = r.NormFloat64()
	}
	const win = 12
	msg := make([]float64, n)
	var acc float64
	for i := 0; i < n; i++ {
		acc += raw[i]
		if i >= win {
			acc -= raw[i-win]
		}
		msg[i] = acc / win
	}
	out := make([]complex64, n)
	var phase float64
	k := 2 * math.Pi * deviationHz / rateHz
	for i, m := range msg {
		phase += k * m
		out[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return out
}

// randDibits builds a deterministic 0..3 dibit stream for C4FM synthesis.
func randDibits(n int, seed int64) []uint8 {
	r := rand.New(rand.NewSource(seed))
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(r.Intn(4))
	}
	return out
}

// dmrCarrierAt synthesises a DMR-like 4800-baud C4FM carrier (deviation 1944 Hz)
// at sampleRate and frequency-shifts it so its centre lands at offsetHz.
func dmrCarrierAt(nSyms int, sampleRate, offsetHz float64, seed int64) []complex64 {
	iq := demod.ModulateP25C4FM(randDibits(nSyms, seed), sampleRate, 1944)
	if offsetHz != 0 {
		// NewNCO(offsetHz) shifts +offsetHz down to DC; negating shifts a DC
		// carrier up to +offsetHz.
		iq = dsp.NewNCO(-offsetHz, sampleRate).Mix(nil, iq)
	}
	return iq
}

// TestClassifyDenseDMRBand is the regression guard for issue #648: a DMR carrier
// surrounded by neighbours on a 12.5 kHz grid must classify as digital (C4FM),
// not be smeared by its neighbours into am/wfm with an absurd occupied bandwidth.
// It exercises the multi-adjacent-carrier wideband path that the survey package's
// isolated-carrier tests never reach.
func TestClassifyDenseDMRBand(t *testing.T) {
	const (
		rate   = 480_000 // integer multiple of the 48 kHz survey channel
		nSyms  = 960     // 960 symbols → 0.2 s at 4800 baud
		grid   = 12_500  // DMR channel spacing
		center = 440_000_000
	)
	// Five carriers on a 12.5 kHz grid; the centre one (DC) is under test.
	offsets := []float64{-2 * grid, -grid, 0, grid, 2 * grid}
	var sum []complex64
	for i, off := range offsets {
		c := dmrCarrierAt(nSyms, rate, off, int64(i+1))
		if sum == nil {
			sum = make([]complex64, len(c))
		}
		for j := range sum {
			sum[j] += c[j]
		}
	}
	// Light AWGN so the fixture isn't unrealistically clean.
	full := demod.ApplyImpairments(sum, rate, demod.Impairments{SNRdB: 25, Seed: 42})

	ds, _ := classifyAndRoute(
		&DiscoveredSystem{}, full, rate,
		Candidate{FreqHz: center}, grid,
		LiveHuntOptions{ClassifyOnly: true}, slog.New(slog.DiscardHandler),
	)

	// The crux of #648: the centre DMR carrier must read as digital, not be
	// smeared by its neighbours into analog am/wfm (which would then be routed to
	// the analog-FM analyser and emit bogus "active / CTCSS").
	if ds.Class == survey.ClassAM || ds.Class == survey.ClassWideFM {
		t.Errorf("dense DMR centre carrier mis-classified as %q (the #648 bug)\n  features: %+v",
			ds.Class, ds.Features)
	}
	if !survey.IsDigital(ds.Class) {
		t.Errorf("dense DMR centre carrier class = %q, want a digital class\n  features: %+v",
			ds.Class, ds.Features)
	}
	if ds.OccupiedBwHz > 25_000 {
		t.Errorf("occupied bandwidth = %d Hz, want ≤ 25 kHz (neighbours bridged)", ds.OccupiedBwHz)
	}
}

// TestClassifyLoneWidebandNotTruncated is the control for the #648 fix: a
// genuinely isolated wideband carrier (no neighbour → neighborHz 0) must keep its
// full measured bandwidth, proving the neighbour-aware bound never truncates a
// real wide signal.
func TestClassifyLoneWidebandNotTruncated(t *testing.T) {
	const rate = 480_000
	full := fmNoise(96_000, rate, 75_000, 11) // wide deviation → broadband FM

	ds, _ := classifyAndRoute(
		&DiscoveredSystem{}, full, rate,
		Candidate{FreqHz: 100_000_000}, 0, // lone candidate: unbounded
		LiveHuntOptions{ClassifyOnly: true}, slog.New(slog.DiscardHandler),
	)

	if ds.OccupiedBwHz < 60_000 {
		t.Errorf("lone wideband occupied bandwidth = %d Hz, want wide (not truncated)\n  features: %+v",
			ds.OccupiedBwHz, ds.Features)
	}

	// The bound is opt-in via the neighbour distance: on this same continuously-
	// occupied wideband signal, a small maxBwHz caps the measured width while the
	// unbounded walk (maxBwHz 0) spans the whole signal. This is the mechanism
	// that stops a dense band's bridged spectrum from reporting 1000+ kHz.
	cfg := survey.DefaultClassifyConfig()
	unbounded, _ := survey.OccupiedBandwidth(full, rate, 0, cfg)
	capped, _ := survey.OccupiedBandwidth(full, rate, 20_000, cfg)
	if capped > 22_000 {
		t.Errorf("capped occupied bandwidth = %d Hz, want ≤ ~20 kHz", capped)
	}
	if unbounded <= capped {
		t.Errorf("expected unbounded width %d Hz to exceed the capped width %d Hz", unbounded, capped)
	}
}

// TestSurveyDeepRoutesAnalogToSiglab is the #648 follow-up guard: an opt-in deep
// survey hands a narrowband carrier the blind classifier called analog to the
// authoritative siglab identify (so a trunked CC the classifier missed is still
// found), while the default survey leaves analog carriers untouched. A genuine
// analog carrier that siglab can't lock still gets its analog tone scan.
func TestSurveyDeepRoutesAnalogToSiglab(t *testing.T) {
	const rate = 48_000
	full := fmNoise(96_000, rate, 3000, 21) // a narrowband carrier siglab won't lock
	ch, chRate := channelize(full, rate, surveyChannelRateHz)
	mkInputs := func(opts LiveHuntOptions) routeInputs {
		return routeInputs{
			fullIQ: full, fullRate: rate, chIQ: ch, chRate: chRate,
			cand: Candidate{FreqHz: 451_000_000}, opts: opts, log: slog.New(slog.DiscardHandler),
		}
	}
	// A carrier the blind classifier put in an analog family, narrowband.
	mkDS := func() *DetectedSignal {
		return &DetectedSignal{FreqHz: 451_000_000, Class: survey.ClassNBFM, OccupiedBwHz: 12_000}
	}

	// Default survey: an analog carrier is NOT handed to siglab (no report), but
	// still gets the analog tone scan.
	base := mkDS()
	if rep := routeSignal(&DiscoveredSystem{}, base, mkInputs(LiveHuntOptions{})); rep != nil {
		t.Errorf("default survey must not hand an analog carrier to siglab; got report %+v", rep)
	}
	if base.Analog == nil {
		t.Errorf("default survey should still run the analog tone scan")
	}

	// Deep survey: the same carrier IS handed to siglab identify (a report comes
	// back). Noise won't lock, so it stays analog (lock-only, never promoted to
	// trunk-voice on a weak guess) and still gets the tone scan.
	deep := mkDS()
	deepRep := routeSignal(&DiscoveredSystem{}, deep, mkInputs(LiveHuntOptions{SurveyDeep: true}))
	if deepRep == nil {
		t.Errorf("deep survey must hand an analog narrowband carrier to siglab identify")
	}
	if deep.Class != survey.ClassNBFM {
		t.Errorf("non-locking deep identify must keep the analog label, got %q", deep.Class)
	}
	if deep.Analog == nil {
		t.Errorf("deep survey carrier that didn't lock should still get the analog tone scan")
	}

	// And a wideband analog carrier is left out of the deep identify entirely.
	wide := &DetectedSignal{FreqHz: 99_500_000, Class: survey.ClassWideFM, OccupiedBwHz: 180_000}
	if rep := routeSignal(&DiscoveredSystem{}, wide, mkInputs(LiveHuntOptions{SurveyDeep: true})); rep != nil {
		t.Errorf("deep survey must not identify a wideband carrier; got report %+v", rep)
	}
}

// TestRunLiveSurvey_MixedTraffic probes two carriers — a synthesized P25 control
// channel and an analog FM voice channel — and asserts the survey classifies and
// routes each correctly: the P25 carrier folds into the discovered system, the
// analog carrier is reported as active NBFM.
func TestRunLiveSurvey_MixedTraffic(t *testing.T) {
	p25, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := uint32(meta.SampleRateHz)

	const p25Hz = 851_000_000
	const fmHz = 154_000_000
	analog := fmNoise(len(p25), float64(rate), 3000, 7)

	src := NewMappedIQSource(map[uint32][]complex64{
		p25Hz: p25,
		fmHz:  analog,
	}, rate)

	sv, reports, err := RunLiveSurvey(context.Background(), LiveHuntOptions{
		Source:        src,
		Candidates:    []uint32{p25Hz, fmHz},
		DwellSeconds:  float64(len(p25)) / float64(rate),
		MinConfidence: 0.3,
		Name:          "Survey Test",
	})
	if err != nil {
		t.Fatalf("RunLiveSurvey: %v", err)
	}
	if len(sv.Signals) != 2 {
		t.Fatalf("len(Signals) = %d, want 2", len(sv.Signals))
	}

	byFreq := map[uint32]DetectedSignal{}
	for _, s := range sv.Signals {
		byFreq[s.FreqHz] = s
	}

	// P25 carrier → trunk control, folded into the system.
	p25Sig := byFreq[p25Hz]
	if p25Sig.Class != survey.ClassTrunkControl {
		t.Errorf("p25 carrier class = %q, want trunk-control\n  features: %+v", p25Sig.Class, p25Sig.Features)
	}
	if p25Sig.Trunking == nil || !p25Sig.Trunking.Locked {
		t.Errorf("p25 carrier: expected a locked trunking ref, got %+v", p25Sig.Trunking)
	}
	if sv.System == nil || len(sv.System.Sites) == 0 {
		t.Fatalf("expected a discovered system with at least one site, got %+v", sv.System)
	}

	// Analog carrier → active NBFM.
	fmSig := byFreq[fmHz]
	if fmSig.Class != survey.ClassNBFM {
		t.Errorf("fm carrier class = %q, want nbfm\n  features: %+v", fmSig.Class, fmSig.Features)
	}
	if fmSig.Analog == nil || !fmSig.Analog.Active {
		t.Errorf("fm carrier: expected an active analog report, got %+v", fmSig.Analog)
	}

	// The trunking branch reports through the shared export tail.
	if len(reports) == 0 {
		t.Errorf("expected at least one trunking CaptureReport")
	}
}
