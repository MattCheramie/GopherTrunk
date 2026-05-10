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
// Roadmap (each item lands as its own self-contained PR so review
// stays tractable):
//
//  1. Skeleton + Vocoder interface integration. ← THIS PR.
//     Decoder satisfies voice.Vocoder, registers as "ambe2" in
//     voice.DefaultRegistry unconditionally on the default build,
//     and emits silence per frame so the full call pipeline can
//     wire to it now and start receiving audio for free as the
//     later pieces land. FrameSize is 7 bytes (49 information
//     bits + 7 padding) matching the libmbe wrapper's contract.
//
//  2. Parameter unpacking — 49 bits → mbe.Params. The largest
//     single PR of the AMBE+2 work. Reads b₀ → ω₀ + L from a
//     scattered bit position layout; voicing-pattern table
//     lookup → Vl[1..L]; gain-vector index → log-amplitude offset
//     feeding the gain block; two-stage spectral VQ index → DCT
//     residual coefficients per band; DCT-II → Tl[1..L].
//     Reference: szechyjs/mbelib's mbe_decodeAmbe2400Parms in
//     ambe3600x2400.c, with constants from
//     ambe3600x2450_const.h (ISC-licensed code; algorithm
//     patents are a separate concern — see docs/vocoders.md).
//
//  3. Synthesis wire-up + bad-frame handling + calibration.
//     Decode() runs the shared mbe pipeline:
//     UnpackParams → mbe.PredictLog2Ml → mbe.AmplitudesFromLog2Ml
//     → mbe.EnhanceAmplitudes → mbe.SynthVoiced
//     → mbe.SynthUnvoicedOverlapAdd → mbe.SynthState.Update…
//     → mbe.AGC.Apply. AMBE+2-specific silence-frame indicator +
//     frame-repeat-on-bad-frame using shared mbe.MaxBadFrames /
//     BadFrameAttenuation. Calibration loop against a DSD-FME or
//     OP25 reference WAV at testdata/. Tune the per-frame gain
//     constant if AGC shows systematic level offset — AGC
//     defaults are tuned for IMBE and AMBE+2 quantization may
//     produce different per-frame energy.
package ambe2
