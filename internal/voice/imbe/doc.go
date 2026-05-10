// Package imbe is the in-progress pure-Go IMBE 4400 bps voice
// decoder used by P25 Phase 1 LDU1 / LDU2 frames. The intent is to
// remove the CGO dependency on libmbe for the most-common digital
// voice scanner setup; the build-tagged internal/voice/mbelib path
// continues to exist for operators who prefer the C reference
// implementation or want AMBE+2 (P25 Phase 2 / DMR / NXDN).
//
// Roadmap (each item lands as its own self-contained PR so review
// stays tractable):
//
//  1. Skeleton + Vocoder interface integration. ← THIS PR.
//     Decoder satisfies voice.Vocoder, registers as "imbe-go" in
//     voice.DefaultRegistry, and emits silence per frame so the
//     full call pipeline can wire to it now and start receiving
//     audio for free as the later pieces land.
//
//  2. Channel coding inverse. 144 transmitted channel bits → 88
//     information bits via Golay(23,12) + Hamming(15,11) + the
//     IMBE-specific bit interleaver + pseudo-random scrambler.
//     framing/golay + framing/hamming are already in tree.
//
//  3. Parameter unpacking. 88 information bits → IMBE model
//     parameters (b_0..b_7+ vectors): fundamental frequency,
//     voiced/unvoiced flags, gain, PRBA + HOC spectral coefficients.
//     Per TIA-102.BABA Section 5.
//
//  4. Speech synthesis. Voiced harmonic sum + unvoiced random
//     excitation + spectral-amplitude shaping → 160 PCM samples /
//     20 ms / 8 kHz mono per frame. Per TIA-102.BABA Section 6.
//
//  5. Quality polish: enhancement filter, frame-repeat on bad-frame
//     indicator, gain smoothing across frames.
//
// Patent + licensing context lives in docs/vocoders.md. The core US
// IMBE patents have expired; this implementation is built from the
// publicly-available TIA-102.BABA specification, with structural
// reference to the open-source mbelib.
package imbe
