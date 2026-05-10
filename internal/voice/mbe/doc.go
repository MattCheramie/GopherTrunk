// Package mbe is the shared Multi-Band Excitation synthesis core
// used by GopherTrunk's IMBE 4400 (P25 Phase 1) and AMBE+2 2400
// (P25 Phase 2 / DMR / NXDN) decoders. The two vocoders differ in
// their bit-level parameter unpacking (different bit budgets,
// quantization tables, and codebook structures), but they both
// drive the same MBE-family synthesis pipeline:
//
//   - cross-frame log2(Ml) prediction recovering harmonic
//     log-amplitudes from per-frame Tl residuals (PredictLog2Ml,
//     UpdateLog2Ml — TIA-102.BABA §6.1);
//   - log2(Ml) → linear Ml exponentiation + spectral moments
//     (AmplitudesFromLog2Ml, FrameEnergy, SpectralCosineSum —
//     §6.2 entry path);
//   - per-harmonic spectral-amplitude enhancement
//     (EnhanceAmplitudes — §6.2);
//   - voiced harmonic generator with cross-frame phase + amplitude
//     continuity (SynthVoiced, UpdateVoicedState — §6.3);
//   - unvoiced FFT excitation with §6.4 overlap-add synthesis
//     window (SynthUnvoicedOverlapAdd, ShapeUnvoicedSpectrum —
//     §6.4);
//   - per-frame fast-attack / slow-release peak AGC (AGC,
//     AGCConfig).
//
// Both decoders construct an mbe.Params from their bit-level unpack
// (Header { W0, L, Silent } + Vl[1..L] + Tl[1..L]) and feed it
// through the shared pipeline. Vocoder-specific intermediates
// (IMBE's K voicing-decision count + Gm PRBA blocks + Cik DCT
// coefficients; AMBE+2's two-stage VQ codebook indices) live on
// the per-decoder Params types.
//
// Patent + licensing context lives in docs/vocoders.md. The IMBE
// patents are expired; AMBE+2 carries active patents in some
// jurisdictions. Re-implementing the algorithm in Go does not
// change the patent posture.
package mbe
