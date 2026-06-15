package survey

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// This file is the issue #648 ground-truth harness. It synthesises narrowband
// DIGITAL FM (DMR Tier II / P25 C4FM at 4800 baud) and analog FM/AM voice at a
// sweep of SNRs to (a) guard the regression the maintainer warned about — analog
// voice must never be promoted to a digital class — and (b) characterise the SNR
// at which a real digital carrier's cyclostationary 4800-baud line collapses to
// the analog range, so any future tuning is done against measured ground truth
// rather than blind.
//
// Measured finding (see the t.Log table from the characterisation test): a clean
// C4FM carrier has a strong baud line (prominence ~50-65, well over the digital
// gate of 15), but under AWGN the prominence falls into the 3-5 band — the same
// band analog FM voice occupies — long before the carrier is otherwise unusable.
// There is no feature in the current vector (BaudProminence, IFStd, EnvelopeCV,
// IFKurtosis) that separates a degraded C4FM carrier from analog voice, so the
// blind classifier cannot recover it without false-positiving analog. The
// authoritative recovery path is siglab identify (the survey's deep route), not
// a classifier threshold change.

const snrHarnessSamples = 24_000 // 0.5 s at 48 kHz

// digitalSNRSweep is the set of AWGN SNRs (dB) the harness exercises, from clean
// down to the marginal regime where the baud line is buried.
var digitalSNRSweep = []float64{40, 30, 25, 20, 16, 12, 10, 8}

// cleanP25 / cleanDMR build pristine (no-impairment) 4800-baud C4FM carriers.
func cleanP25(seed int64) []complex64 {
	return demod.ModulateP25C4FM(randDibits(snrHarnessSamples/10, seed), testRateHz, 1800)
}

func cleanDMR(seed int64) []complex64 {
	return demod.ModulateC4FM(randDibits(snrHarnessSamples/10, seed), 10, 8, 0.2, testRateHz, 1944)
}

// noisyAnalogFM / noisyAM build analog voice carriers at the given AWGN SNR.
func noisyAnalogFM(snrDB float64, seed int64) []complex64 {
	clean := fmModulate(lowpassNoise(snrHarnessSamples, seed), testRateHz, 3000)
	return demod.ApplyImpairments(clean, testRateHz, demod.Impairments{SNRdB: snrDB, Seed: seed})
}

func noisyAM(snrDB float64, seed int64) []complex64 {
	clean := amModulate(lowpassNoise(snrHarnessSamples, seed), 0.95)
	return demod.ApplyImpairments(clean, testRateHz, demod.Impairments{SNRdB: snrDB, Seed: seed})
}

// withAWGN adds AWGN at the given SNR to a clean carrier.
func withAWGN(clean []complex64, snrDB float64, seed int64) []complex64 {
	return demod.ApplyImpairments(clean, testRateHz, demod.Impairments{SNRdB: snrDB, Seed: seed})
}

// TestCleanDigitalReadsDigital is the synthesis sanity check: a pristine 4800-baud
// C4FM carrier must classify digital (its baud line is strong at the source). If
// this regresses, the harness fixtures or the modulators are broken.
func TestCleanDigitalReadsDigital(t *testing.T) {
	for _, tc := range []struct {
		name string
		iq   []complex64
	}{
		{"p25-c4fm", cleanP25(7)},
		{"dmr-c4fm", cleanDMR(7)},
	} {
		cls := Classify(tc.iq, testRateHz)
		if !IsDigital(cls.Class) {
			t.Errorf("clean %s read %q, want a digital class\n  features: %+v",
				tc.name, cls.Class, cls.Features)
		}
	}
}

// TestAnalogNeverReadsDigitalAcrossSNR is the load-bearing regression guard for
// issue #648: an analog FM or AM voice carrier must never be promoted to a
// digital class, at any SNR. This is the failure mode a careless lowering of the
// digital prominence gate would cause — the harness pins it so such a change
// cannot land silently.
func TestAnalogNeverReadsDigitalAcrossSNR(t *testing.T) {
	analog := []struct {
		name  string
		build func(float64, int64) []complex64
	}{
		{"analog-fm", noisyAnalogFM},
		{"am-voice", noisyAM},
	}
	for _, a := range analog {
		for i, snr := range digitalSNRSweep {
			cls := Classify(a.build(snr, int64(200+i)), testRateHz)
			if IsDigital(cls.Class) {
				t.Errorf("%s at %.0f dB read digital %q (false-positive regression)\n  features: %+v",
					a.name, snr, cls.Class, cls.Features)
			}
		}
	}
}

// TestDigitalBaudCollapseCharacterization logs the baud-line measurement for the
// digital carriers from clean down through the SNR sweep, documenting where the
// cyclostationary line falls into the analog prominence band. It is a
// characterisation test (logs, no behavioural assertion past the clean sanity
// check) — the data is what any future, capture-backed tuning is measured
// against. Run `go test -run TestDigitalBaudCollapseCharacterization -v` to view.
func TestDigitalBaudCollapseCharacterization(t *testing.T) {
	gate := DefaultClassifyConfig().DigitalProminence
	for _, d := range []struct {
		name  string
		clean []complex64
	}{
		{"p25", cleanP25(7)},
		{"dmr", cleanDMR(7)},
	} {
		c := Classify(d.clean, testRateHz)
		t.Logf("%s CLEAN    → baud=%6.0f prom=%5.1f (gate %.0f) ifstd=%.3f modality=%d class=%s",
			d.name, c.Features.BaudHz, c.Features.BaudProminence, gate,
			c.Features.IFStd, c.Features.IFModality, c.Class)
		for _, snr := range digitalSNRSweep {
			cc := Classify(withAWGN(d.clean, snr, 7), testRateHz)
			f := cc.Features
			t.Logf("%s SNR=%3.0f → baud=%6.0f prom=%5.1f (gate %.0f) ifstd=%.3f modality=%d class=%s",
				d.name, snr, f.BaudHz, f.BaudProminence, gate, f.IFStd, f.IFModality, cc.Class)
		}
	}
}
