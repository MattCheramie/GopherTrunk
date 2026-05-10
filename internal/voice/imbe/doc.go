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
//  1. Skeleton + Vocoder interface integration. Decoder satisfies
//     voice.Vocoder, registers as "imbe-go" in
//     voice.DefaultRegistry, and emits silence per frame so the
//     full call pipeline can wire to it now and start receiving
//     audio for free as the later pieces land.
//
//  2. Channel coding inverse — per-vector FEC.
//     144 channel bits → 88 information bits via Golay(23,12,7)
//     for u_0..u_3 + Hamming(15,11,3) for u_4..u_6 + a
//     no-FEC u_7 passthrough. See channel.go.
//
//  2b. Channel coding inverse — pseudo-random scrambler. ← THIS PR.
//     XORs a 114-bit u_0-keyed LCG PRBS (TIA-102.BABA §7.4) over
//     the channel bits of u_1..u_6 to whiten the spectrum. u_0
//     stays unscrambled because it carries the seed; u_7 stays
//     unscrambled per the spec. See scrambler.go.
//
//     Note: there is no separate "bit interleaver" inside the IMBE
//     codec — the §7.5 ordering is satisfied by how the upstream
//     P25 LDU1 / LDU2 frame decoder extracts the 144 bits from a
//     voice frame (a P25 phase1 layer concern, not an IMBE one).
//     channel.go's per-vector layout already matches the order the
//     upstream extractor will hand it.
//
//  3. Parameter unpacking — header.
//     88 information bits → IMBE Header { W0, L, K, Silent }. The
//     b_0 fundamental-frequency parameter lives at scattered
//     positions {0..5, 85, 86}; ω₀ + L + K all derive from it
//     per TIA-102.BABA §5.3 / Annex E. See params.go.
//
//  3b. Parameter unpacking — voicing + gain + spectral. ← THIS PR.
//     Re-orders the remaining 79 bits via bo[L9] into the bb[v][p]
//     layout, then extracts Vl[1..L] voicing decisions, the b_2
//     gain index → Gm[1] = B2[b_2], the 5 PRBA gain blocks
//     Gm[2..6] (Annex E eq. 68), and the HOC spectral coefficients
//     Cik via hoba[L9] × quantstep × standdev (§5.4). Two inverse
//     DCT-IIs (over Gm and over Cik) produce Tl[1..L], the
//     pre-prediction log-amplitude residuals. The cross-frame
//     log2Ml prediction (eq. 75-77) needs prev-frame state and
//     lives in the synthesizer (step 4).
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
