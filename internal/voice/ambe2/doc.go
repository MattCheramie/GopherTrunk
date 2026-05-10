// Package ambe2 is the in-progress pure-Go AMBE+2 2400 bps voice
// decoder used by P25 Phase 2, DMR (Tier II / III), and NXDN voice
// frames. The intent is to remove the CGO dependency on libmbe so
// every default build has a working AMBE+2 path without a C
// toolchain or system shared library.
//
// AMBE+2 is the same Multi-Band Excitation algorithm family as
// IMBE — both produce 8 kHz / 20 ms / 160 int16 PCM by summing
// voiced harmonics + an FFT-shaped unvoiced excitation. The
// algorithm-shared synthesis primitives live in
// internal/voice/mbe (PredictLog2Ml, SynthVoiced,
// SynthUnvoicedOverlapAdd, EnhanceAmplitudes, AGC, …); this
// package layers the AMBE+2-specific front half on top:
// bit-level parameter unpack from 49 information bits into the
// shared mbe.Params shape.
//
// Patent + licensing context lives in docs/vocoders.md. The
// AMBE+2 algorithm itself is patent-encumbered in some
// jurisdictions; re-implementing it in pure Go does not change
// that posture. Operators in licence-restrictive jurisdictions
// should evaluate before deploying.
//
// Roadmap (each item landed as its own self-contained PR so review
// stayed tractable):
//
//  1. Skeleton + Vocoder interface integration. Decoder satisfies
//     voice.Vocoder, registers as "ambe2" in voice.DefaultRegistry
//     unconditionally on the default build. FrameSize is 7 bytes
//     (49 information bits + 7 padding) matching the libmbe
//     wrapper's contract.
//
//  2. Parameter unpacking — 49 bits → ambe2.Params (mbe.Params +
//     AMBE+2-specific DeltaGamma / Unvc / Tone / B0..B8). Reads
//     b₀ → ω₀ + L from a scattered bit position layout;
//     voicing-pattern table lookup → Vl[1..L]; gain-vector index →
//     log-amplitude offset feeding the gain block; two-stage
//     spectral VQ index → DCT residual coefficients per band;
//     DCT-II → Tl[1..L]. Reference: szechyjs/mbelib's
//     mbe_decodeAmbe2400Parms in ambe3600x2400.c with constants
//     from ambe3600x2400_const.h (ISC-licensed code; algorithm
//     patents are a separate concern — see docs/vocoders.md).
//
//  3. Synthesis wire-up + bad-frame handling. ← THIS PR.
//     Decode() runs the shared mbe pipeline: UnpackParams → gamma
//     fold (DC removal + 0.5·prev_gamma recursion + 0.5·log2(L)
//     offset) → mbe.PredictLog2Ml → mbe.AmplitudesFromLog2Ml →
//     unvoiced amplitude scaling by Unvc → mbe.EnhanceAmplitudes
//     → mbe.SynthVoiced + mbe.SynthUnvoicedOverlapAdd →
//     SynthState.Update… → mbe.AGC.Apply. AMBE+2-specific
//     tone-frame path (b₀ ∈ {0x7E, 0x7F}) routes through the §6.4
//     OA fade-out + state reset; bad-frame replay uses the shared
//     mbe.MaxBadFrames / mbe.BadFrameAttenuation. The cross-frame
//     gamma (gamma_curr = ΔG + 0.5·gamma_prev) lives on the
//     Decoder; the per-frame fold rewrites Tl so the shared
//     mbe.PredictLog2Ml produces AMBE+2-spec output without an
//     AMBE+2-aware variant.
//
//  4. Remaining polish: calibration against a DSD-FME or OP25
//     reference WAV at testdata/ (capture a known DMR voice frame
//     + decode through both, compare RMS + cross-correlation);
//     tune the per-frame gain constant if AGC shows systematic
//     level offset against the reference (AGC defaults are tuned
//     for IMBE and AMBE+2 quantization may produce different
//     per-frame energy); proper tone-frame synthesis (single +
//     dual sinewave from the preserved B1/B2 indices) replacing
//     the current silence-out path.
package ambe2
