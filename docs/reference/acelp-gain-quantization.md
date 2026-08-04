---
slug: acelp-gain-quantization
title: ACELP gain quantization
entry_type: term
category: voice-coding
description: "ACELP gain quantization recovers the TETRA subframe pitch and code gains from a 6-bit energy index: an MA-predicted log-energy plus a 64-entry quantizer correction, converted back to a linear gain by pow2."
keywords: ACELP gain quantization, energy quantization, MA prediction, pitch gain, code gain, log2 pow2, Dec_Ener, TETRA vocoder, ETSI EN 300 395-2, quantization
aka: [gain dequantization, "energy dequantization", "gain codebook"]
autolink: true
infobox:
  - { label: Role, value: Recover pitch & code gains }
  - { label: Index, value: "6-bit energy VQ (64 entries)" }
  - { label: Domain, value: "log2 energy, MA-predicted" }
  - { label: Spec, value: ETSI EN 300 395-2 }
see_also: [acelp, quantization, acelp-codebooks, acelp-lsp-codebooks, etsi-g191-basic-operators, fixed-point-vs-floating-point, tetra]
cite_urls:
  - https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction
  - https://en.wikipedia.org/wiki/Vector_quantization
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
---

**ACELP gain quantization** is the stage that recovers, for each subframe, the two scalars that
tell the [TETRA](/reference/tetra/) [ACELP](/reference/acelp/) decoder how loudly to mix its two
excitation contributions: the **pitch gain** on the [adaptive codebook](/reference/acelp-codebooks/)
vector and the **code gain** on the algebraic innovation.[^acelp] Rather than quantizing the two
gains directly, TETRA quantizes them in the **log-energy domain** and predicts most of their value
from the previous subframe, so only a small *correction* travels over the air — a 6-bit index into
a 64-entry energy [quantizer](/reference/quantization/) table.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A moving-average predictor estimates each subframe's pitch and code log-energy from the previous subframe, a six-bit index selects a correction from a sixty-four entry energy table, the two are summed, and a pow2 conversion turns the corrected log-energy into a linear gain." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="16" y="20" width="96" height="30"/><rect x="16" y="90" width="96" height="30"/><rect x="230" y="55" width="52" height="30"/><rect x="330" y="55" width="60" height="30"/></g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="64" y="32">MA predictor</text><text x="64" y="43">from last subfr</text>
    <text x="64" y="102">enerQua[idx]</text><text x="64" y="113">6-bit · 64 rows</text>
    <text x="256" y="73">Σ (Q8)</text>
    <text x="360" y="70">pow2</text><text x="360" y="80">→ gain</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="112" y1="35" x2="230" y2="63" marker-end="url(#gar)"/>
    <line x1="112" y1="105" x2="230" y2="78" marker-end="url(#gar)"/>
    <line x1="282" y1="70" x2="330" y2="70" marker-end="url(#gar)"/>
    <line x1="390" y1="70" x2="452" y2="70" marker-end="url(#gar)"/>
  </g>
  <text x="440" y="66" font-size="7.5" fill="currentColor" text-anchor="end">gainPit / gainCod</text>
</svg>
<figcaption>The transmitted index corrects an MA-predicted log-energy; pow2 converts the result to a linear pitch or code gain.</figcaption>
</figure>

## The log-energy domain

Gains span a wide dynamic range, and the perceptually important quantity is *energy*, so TETRA
works in `log2` energy rather than linear amplitude. For each subframe GopherTrunk's `decEner` (a
port of the reference `Dec_Ener`) first measures three reference energies in the log domain: the
energy of the `1/A(z)` impulse response, the energy of the adaptive-codebook vector, and the
energy of the innovation code. These normalise the gain to the actual excitation the decoder has
in hand, so the transmitted index only has to encode the *residual* the encoder could not predict.
All of the arithmetic runs on the
[G.191 fixed-point basic operators](/reference/etsi-g191-basic-operators/) — `log2fp`, `pow2fp`,
and the shift-accumulate helpers — so the recovered gains are bit-exact with the reference codec.

## MA prediction and the energy table

Speech energy is strongly correlated frame to frame, so most of each subframe's log-energy is
**predicted** from the previous subframe by a fixed moving-average rule. The predicted pitch
energy is `0.5·last_pit + 0.25·last_cod − 3.0` and the predicted code energy is
`0.5·last_cod + 0.25·last_pit − 3.0` (all in Q8, floored at zero). The transmitted 6-bit index
then selects a `(pitch-energy, code-energy)` correction pair from the 64-entry table `enerQua` in
`internal/voice/acelp/ener_tables.go` (stored as 128 Q8 int16 values), and the correction is added
to the prediction. The updated energies are clamped — pitch energy to 27 and code energy to 25 in
Q8 — and stored as the predictor state for the next subframe, which is what closes the MA loop.

## Reconstructing the gain

With the corrected log-energy in hand, the decoder converts back to a linear gain:

- **Pitch gain** = `pow2( 0.5·(last_ener_pit − ener_plt) )`, in Q12, clamped to a maximum of 1.2
  (the constant 4915) so a runaway pitch gain cannot make the synthesis filter ring.
- **Code gain** = `pow2( 0.5·(last_ener_cod − ener_c) )`, in Q0.

Both use the `pow2fp` table interpolation, the inverse of the `log2fp` used on the way in. The
subtraction of the measured reference energy (`ener_plt`, `ener_c`) is what turns a stored *energy*
into the correct *scale factor* for the specific codebook vector the decoder just built.

## Bad-frame handling

Because the predictor is recursive, an erased frame cannot simply be skipped — its missing energy
would poison every subframe that follows. On a flagged bad frame `decEner` decays both stored
energies by 0.5 (128 in Q8) toward zero instead of applying a fresh correction, so a burst of lost
frames fades the gain out gracefully rather than freezing or exploding it. This mirrors the
frame-repeat concealment the rest of the ACELP decoder runs on erasures.

## Relevance to SDR

Gain dequantization is decode-only arithmetic: the encoder's search for the best index is not
needed, only the reconstruction. Its correctness matters out of proportion to its size, though —
because it sets absolute level, a scaling slip here makes an entire call decode too quietly or too
loudly while every spectral detail still looks right. GopherTrunk pins the gain path against the
ETSI reference codec as part of the ACELP decoder's end-to-end conformance, so recovered levels
match the reference sample for sample.

## Sources

[^acelp]: [Algebraic code-excited linear prediction](https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction) — Wikipedia, on the pitch and fixed-codebook gains of an ACELP coder.
[^vq]: [Vector quantization](https://en.wikipedia.org/wiki/Vector_quantization) — Wikipedia, on codebook quantization of coder parameters such as gains.
