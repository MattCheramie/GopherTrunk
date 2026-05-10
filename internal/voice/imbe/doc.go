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
//  3b. Parameter unpacking — voicing + gain + spectral.
//     Re-orders the remaining 79 bits via bo[L9] into the bb[v][p]
//     layout, then extracts Vl[1..L] voicing decisions, the b_2
//     gain index → Gm[1] = B2[b_2], the 5 PRBA gain blocks
//     Gm[2..6] (Annex E eq. 68), and the HOC spectral coefficients
//     Cik via hoba[L9] × quantstep × standdev (§5.4). Two inverse
//     DCT-IIs (over Gm and over Cik) produce Tl[1..L], the
//     pre-prediction log-amplitude residuals. The cross-frame
//     log2Ml prediction (eq. 75-77) needs prev-frame state and
//     lives in the synthesizer (step 4a).
//
//  4a. Speech synthesis — cross-frame log-amplitude recovery.
//     TIA-102.BABA §6.1 eqs. 75-77: predict at curr-frame harmonic
//     positions by interpolating prev-frame log2(Ml) at l ·
//     ω₀_curr/ω₀_prev (γ = 0.65 scale), subtract the prediction's
//     DC bias, then add Tl[l]. Lives on a SynthState that the
//     synthesizer (step 4c) extends with voicing + phase memory.
//     See synth.go.
//
//  4b. Speech synthesis — amplitude prep.
//     log2(Ml) → linear Ml = 2^log2(Ml), the spectral moments
//     R_M0 = Σ Ml² and R_M1 = Σ Ml² · cos(ω₀·l) feeding §6.2
//     enhancement, and a voicing-fraction summary used as a
//     coarse voiced/unvoiced hint by the upcoming synthesis
//     combiner. See amps.go.
//
//  4c. Speech synthesis — voiced harmonic generator.
//     For each harmonic that's voiced this frame OR was voiced
//     last frame, a sinusoid at l · ω₀ with linear amplitude tilt
//     between M_prev[l] and M_curr[l] across 160 samples + a
//     quadratic phase term that integrates the linear ω₀ drift.
//     The dual-frame iteration gives clean fade-in / fade-out on
//     voicing transitions without click artifacts. SynthState
//     gains PrevPhase + PrevMl for the cross-frame continuity.
//     TIA-102.BABA §6.3. See synth_voiced.go.
//
//  4d. Speech synthesis — unvoiced excitation.
//     A 256-point FFT noise spectrum (caller-supplied so unit
//     tests stay deterministic) is shaped by the §6.4 rules: bins
//     under voiced harmonics are zeroed (those go through 4c),
//     bins under unvoiced harmonics are scaled by Ml[l], bins
//     outside [1..L] are zeroed. The conjugate-mirror invariant
//     is preserved by applying the same real scale to (k, N−k)
//     pairs so the IFFT produces a real-valued time-domain
//     contribution. See synth_unvoiced.go.
//
//  4e. Speech synthesis — combine + Decode() wiring. ← THIS PR.
//     Decoder gains a SynthState + math/rand source per call.
//     Decode() runs the full pipeline: bytes → 88 info bits →
//     UnpackParams → PredictLog2Ml → AmplitudesFromLog2Ml →
//     SynthVoiced + SynthUnvoicedFromNoise (additive into one
//     buffer) → state roll-forward → hard-clip × placeholder gain
//     → int16 PCM. b_0 silence-window frames short-circuit to
//     all-zero PCM and reset the state; bad b_0 returns silence
//     without resetting state so the next valid frame can pick up
//     from the last-known-good prediction history. The §6.4
//     overlap-add synthesis window + the §6.2 spectral-amplitude
//     enhancement become quality-polish PRs (see step 5) — the
//     synthesizer produces intelligible voice without them, just
//     with frame-edge click artifacts and an untilted envelope.
//
//  5a. Speech synthesis — §6.4 overlap-add window. ← THIS PR.
//     SynthState gains PrevUnvoicedTail[96]; SynthUnvoicedOverlapAdd
//     wraps the §6.4 IFFT in a 256-sample periodic Hann window,
//     emits the prev-frame's windowed tail into dst[0..95]
//     (the overlap region) and stashes the curr-frame's tail
//     [160..255] for the next call. Eliminates the click artifacts
//     that plain truncated-IFFT excitation produced at frame
//     boundaries. Decoder.Decode runs the OA path on every frame,
//     including silence frames where the prev tail still fades
//     out before the state is cleared. See synth_unvoiced.go.
//
//  5b. §6.2 spectral-amplitude enhancement. Per-harmonic Ml
//     multiplier from the closed-form weight over R_M0 + R_M1 +
//     ω₀ + l (§6.2). Boosts harmonics the model under-represents
//     so the spectral envelope tilts correctly.
//
//  5c. Spec-derived gain calibration. Replaces the placeholder
//     pcmGain = 4096 with the gain that makes mbelib + imbe-go
//     produce comparable output levels for the same input frame
//     stream.
//
//  5d. Robustness polish: enhancement filter, frame-repeat on
//     bad-frame indicator, gain smoothing across frames.
//
// Patent + licensing context lives in docs/vocoders.md. The core US
// IMBE patents have expired; this implementation is built from the
// publicly-available TIA-102.BABA specification, with structural
// reference to the open-source mbelib.
package imbe
