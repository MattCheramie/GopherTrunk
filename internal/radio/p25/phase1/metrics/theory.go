// Package metrics measures P25 Phase 1 demodulator quality. It turns the
// soft-symbol and corrected-error observations the receiver already exposes
// into the four numbers that localize a weak-decode problem:
//
//   - EVM (error-vector magnitude / constellation dispersion),
//   - SNR (estimated from the symbol residuals),
//   - pre-FEC SER/BER (corrected-error counts over the bit/symbol budget),
//   - sync-correlation margin (how far the FSW match cleared tolerance).
//
// Paired with the closed-form BER-vs-Eb/N0 references in this file, these let
// a benchmark answer the question the field captures cannot: is the demod
// leaving SNR on the table (an implementation gap, measurable as loss versus
// the theoretical curve) or is the signal simply marginal (the demod tracks
// the reference and there is nothing more to recover)?
//
// All functions here are pure: they take observations and return numbers.
// The siglab integration wires them to the live receiver taps; the synthetic
// sweep compares the measured curve to the references below.
package metrics

import "math"

// QFunc is the Gaussian tail probability Q(x) = P(Z > x) for a standard
// normal Z, expressed via the complementary error function:
//
//	Q(x) = ½·erfc(x / √2).
//
// It is the building block of every coherent-detection BER expression below.
func QFunc(x float64) float64 {
	return 0.5 * math.Erfc(x/math.Sqrt2)
}

// dBToLinear converts a dB power ratio to a linear ratio.
func dBToLinear(dB float64) float64 { return math.Pow(10, dB/10) }

// EsN0FromSNR converts the per-sample in-band SNR that the synthetic
// impairment model injects (demod.Impairments.SNRdB — signal power over noise
// power across the full sampled bandwidth Fs) into the per-symbol Es/N0.
//
// Noise power in bandwidth Fs is N0·Fs, so with Rsym symbols/s and
// sps = Fs/Rsym samples/symbol:
//
//	Es/N0 = (S/Rsym)/N0 = (S / (N0·Fs)) · (Fs/Rsym) = SNR · sps.
//
// In dB: Es/N0(dB) = SNRdB + 10·log10(sps).
func EsN0FromSNR(snrDB, sps float64) float64 {
	if sps <= 0 {
		return snrDB
	}
	return snrDB + 10*math.Log10(sps)
}

// EbN0FromEsN0 converts Es/N0 to Eb/N0 for a modulation carrying
// bitsPerSymbol bits per symbol: Eb/N0 = Es/N0 / log2(M), i.e.
// Eb/N0(dB) = Es/N0(dB) − 10·log10(bitsPerSymbol).
func EbN0FromEsN0(esN0dB, bitsPerSymbol float64) float64 {
	if bitsPerSymbol <= 0 {
		return esN0dB
	}
	return esN0dB - 10*math.Log10(bitsPerSymbol)
}

// BERCoherentQPSK is the bit-error probability of ideal coherent,
// Gray-coded QPSK (and BPSK — they share the curve) in AWGN:
//
//	Pb = Q(√(2·Eb/N0)).
//
// This is the matched-filter lower bound for the linear CQPSK path. The
// production receiver detects π/4-DQPSK *differentially*, which costs a
// fixed penalty over this bound (see DQPSKDifferentialPenaltyDB); use this
// as the absolute reference and account for the penalty in the loss budget.
func BERCoherentQPSK(ebN0dB float64) float64 {
	g := dBToLinear(ebN0dB)
	return QFunc(math.Sqrt(2 * g))
}

// DQPSKDifferentialPenaltyDB is the approximate Eb/N0 penalty differentially
// coherent π/4-DQPSK pays over ideal coherent QPSK at the BER values of
// interest (~10⁻²–10⁻³): roughly 2.3 dB (Proakis, differential vs coherent
// QPSK). The CQPSK loss budget is taken relative to BERCoherentQPSK shifted
// right by this amount, so the gate measures *implementation* loss on top of
// the unavoidable differential-detection loss rather than charging the demod
// for a penalty inherent to the chosen scheme.
const DQPSKDifferentialPenaltyDB = 2.3

// BER4PAM is the bit-error probability of ideal coherent, Gray-coded 4-level
// PAM in AWGN — the absolute reference for the C4FM path's 4-level eye.
//
// Symbol-error probability of M-PAM is
//
//	Ps = 2(M−1)/M · Q(√(6/(M²−1) · Es/N0)),
//
// and with Gray coding adjacent symbols differ by one bit, so at the SNRs of
// interest Pb ≈ Ps/log2(M). For M=4, log2(M)=2 and Es/N0 = 2·Eb/N0:
//
//	Pb ≈ (3/4)·Q(√(0.8·Eb/N0)).
//
// C4FM is detected through an FM discriminator, not a coherent 4-PAM matched
// filter, so the real path sits well to the right of this bound; it is the
// theoretical floor the loss budget is measured against (see C4FMReferenceLossDB).
func BER4PAM(ebN0dB float64) float64 {
	g := dBToLinear(ebN0dB)
	return 0.75 * QFunc(math.Sqrt(0.8*g))
}

// SERCoherentQPSK is the symbol-error probability of ideal coherent QPSK in
// AWGN, the symbol-domain companion to BERCoherentQPSK:
//
//	Ps = 2·Q(√(Es/N0)) − Q(√(Es/N0))²  ≈  2·Q(√(Es/N0)),
//
// expressed in Es/N0 (= 2·Eb/N0). The exact two-term form is used so the
// reference stays accurate at low Es/N0 where the union-bound approximation
// over-counts. This is the symbol-domain reference for the CQPSK sweep, whose
// measured quantity is the recovered dibit (symbol) error rate.
func SERCoherentQPSK(esN0dB float64) float64 {
	q := QFunc(math.Sqrt(dBToLinear(esN0dB)))
	return 2*q - q*q
}

// SER4PAM is the symbol-error probability of ideal coherent 4-PAM in AWGN,
// the symbol-domain companion to BER4PAM:
//
//	Ps = 2(M−1)/M · Q(√(6/(M²−1)·Es/N0)) = 1.5·Q(√(0.4·Es/N0))  for M=4,
//
// in Es/N0. The C4FM sweep measures the recovered dibit (symbol) error rate,
// so this — not the bit-domain BER4PAM — is the curve its loss budget is taken
// against.
func SER4PAM(esN0dB float64) float64 {
	return 1.5 * QFunc(math.Sqrt(0.4*dBToLinear(esN0dB)))
}

// C4FMReferenceLossDB is the nominal Eb/N0 gap a non-coherent FM-discriminator
// detector of C4FM carries over the coherent 4-PAM bound (BER4PAM). It is a
// documentation/centering constant only — the actual CI loss budget is taken
// against BER4PAM and calibrated from a measured baseline (the budget absorbs
// this and the residual modeling gap). Literature places FM-discriminator C4FM
// detection several dB off the coherent bound; ~6 dB centers the expectation.
const C4FMReferenceLossDB = 6.0

// InverseBERForEbN0 solves a monotonic BER(Eb/N0) curve for the Eb/N0 (dB) at
// which it crosses targetBER, by bisection over [loDB, hiDB]. ber must be
// strictly decreasing in Eb/N0 (all curves here are). It is how the sweep gate
// compares a measured curve to a reference: find the Eb/N0 each reaches a given
// BER and take the horizontal gap (the implementation loss, in dB).
func InverseBERForEbN0(ber func(float64) float64, targetBER, loDB, hiDB float64) float64 {
	for i := 0; i < 200; i++ {
		mid := 0.5 * (loDB + hiDB)
		if ber(mid) > targetBER {
			loDB = mid // BER still too high → need more Eb/N0
		} else {
			hiDB = mid
		}
	}
	return 0.5 * (loDB + hiDB)
}
