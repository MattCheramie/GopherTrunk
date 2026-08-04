---
slug: acelp-lsp-codebooks
title: ACELP LSP codebooks
entry_type: term
category: voice-coding
description: "The ACELP LSP codebooks are the split-vector quantizer tables (dico1 256×3, dico2 512×3, dico3 512×4, Q15) that reconstruct the ten line spectral pairs describing each TETRA speech frame's short-term spectral envelope."
keywords: ACELP LSP codebook, line spectral pairs, LSP, split vector quantizer, split-VQ, TETRA vocoder, LPC envelope, dico1 dico2 dico3, ETSI EN 300 395-2, quantization
aka: [LSP codebook, "LSP split-VQ", "line spectral pair codebook"]
autolink: true
infobox:
  - { label: Role, value: LSP (spectral envelope) dequantizer }
  - { label: Structure, value: "Split-3 VQ (3, 3, 4 LSPs)" }
  - { label: Tables, value: "dico1 256×3, dico2 512×3, dico3 512×4" }
  - { label: Spec, value: ETSI EN 300 395-2 }
see_also: [acelp, linear-predictive-coding, quantization, code-excited-linear-prediction, acelp-codebooks, acelp-gain-quantization, tetra, vocoder]
cite_urls:
  - https://en.wikipedia.org/wiki/Line_spectral_pairs
  - https://en.wikipedia.org/wiki/Vector_quantization
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
---

The **ACELP LSP codebooks** are the split-vector-quantizer tables that the
[TETRA](/reference/tetra/) [ACELP](/reference/acelp/) decoder uses to turn a handful of
transmitted indices back into the ten **line spectral pairs** (LSPs) that describe a speech
frame's short-term spectral envelope.[^lsp] LSPs are simply an alternate, quantization-friendly
representation of the [linear-predictive-coding](/reference/linear-predictive-coding/) filter
coefficients — living in the cosine domain on the open interval (−1, 1), where small errors
stay bounded and an ordering property guarantees the reconstructed filter is stable.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A 26-bit index is split into three parts that select entries from three codebooks, dico1 for the first three line spectral pairs, dico2 for the next three, and dico3 for the last four, which concatenate into a ten-element line-spectral-pair vector describing the spectral envelope." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lspar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none" font-size="8">
    <rect x="16" y="22" width="70" height="24"/><rect x="16" y="56" width="70" height="24"/><rect x="16" y="90" width="70" height="24"/>
    <rect x="200" y="22" width="80" height="24"/><rect x="200" y="56" width="80" height="24"/><rect x="200" y="90" width="80" height="24"/>
    <rect x="360" y="56" width="92" height="24"/>
  </g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="51" y="37">idx1 · 8 bit</text><text x="51" y="71">idx2 · 9 bit</text><text x="51" y="105">idx3 · 9 bit</text>
    <text x="240" y="34">dico1 256×3</text><text x="240" y="45">LSP 1–3</text>
    <text x="240" y="68">dico2 512×3</text><text x="240" y="79">LSP 4–6</text>
    <text x="240" y="102">dico3 512×4</text><text x="240" y="113">LSP 7–10</text>
    <text x="406" y="65">10 LSPs</text><text x="406" y="75">→ envelope</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="86" y1="34" x2="200" y2="34" marker-end="url(#lspar)"/>
    <line x1="86" y1="68" x2="200" y2="68" marker-end="url(#lspar)"/>
    <line x1="86" y1="102" x2="200" y2="102" marker-end="url(#lspar)"/>
    <line x1="280" y1="34" x2="360" y2="64" marker-end="url(#lspar)"/>
    <line x1="280" y1="68" x2="360" y2="68" marker-end="url(#lspar)"/>
    <line x1="280" y1="102" x2="360" y2="72" marker-end="url(#lspar)"/>
  </g>
</svg>
<figcaption>Three transmitted indices select rows of three codebooks; the selected sub-vectors concatenate into the ten-element LSP vector that fixes the frame's spectral envelope.</figcaption>
</figure>

## Why split the vector

A single joint codebook covering all ten LSPs would need an impractically large table — an
*N*-bit index demands 2^*N* stored vectors, and enough resolution for ten dimensions would be
astronomical. **Split-vector quantization** slices the ten-dimensional LSP vector into smaller
sub-vectors quantized independently, trading a little coding efficiency for a table small
enough to store and search. TETRA's ACELP uses a **split-3** design: the first three LSPs, the
middle three, and the final four are each quantized with their own codebook, for 8 + 9 + 9 = 26
bits of LSP information per frame. Grouping neighbouring LSPs together preserves the local
correlation between adjacent spectral lines that carries most of the perceptual information.

## The three codebooks

GopherTrunk stores the ETSI reference tables as `lspDico1`, `lspDico2`, and `lspDico3` in
`internal/voice/acelp/lsp_tables.go`. Each row is a sub-vector of signed Q15 fixed-point values
in the cosine domain:

| Codebook | Index width | Entries | LSPs per row | Flat size |
| --- | --- | --- | --- | --- |
| `lspDico1` | 8 bits | 256 | 3 (LSP 1–3) | 768 int16 |
| `lspDico2` | 9 bits | 512 | 3 (LSP 4–6) | 1536 int16 |
| `lspDico3` | 9 bits | 512 | 4 (LSP 7–10) | 2048 int16 |

The values are stored in **Q15** here, one detail worth flagging: the ETSI reference keeps the
table in Q14 and doubles each value as it loads, and GopherTrunk folds that doubling directly
into the constants so the decoder can index the table without a scaling step. Getting the
[fixed-point](/reference/fixed-point-vs-floating-point/) scale wrong at this stage would tilt
the whole reconstructed envelope. These are pure [quantization](/reference/quantization/) data —
fixed constants derived from the reference codec, consumed by the decoder's `dLsp334` routine.

## Reconstructing the envelope

For each frame the decoder reads the three indices, looks up the three sub-vectors, and
concatenates them into a ten-element LSP vector. That vector is converted from line spectral
pairs back to LPC coefficients, which define the all-pole synthesis filter `1/A(z)`. Exciting
that filter with the pitch and innovation contributions from the
[adaptive and algebraic codebooks](/reference/acelp-codebooks/) — scaled by the
[dequantized gains](/reference/acelp-gain-quantization/) — produces the reconstructed speech.
The LSP codebooks therefore fix the *shape* of the spectrum (the formant structure that makes a
vowel sound like that vowel), while the excitation codebooks supply its fine structure.

Because LSPs move slowly and smoothly frame to frame, a single mis-decoded index tends to
produce an audible but brief spectral glitch rather than a catastrophic failure — the ordering
property still yields a stable filter, so the synthesizer never diverges. This graceful
behaviour under channel errors is one reason LSP quantization is the standard front end for
[CELP](/reference/code-excited-linear-prediction/)-family coders across cellular and
land-mobile radio.

## Relevance to SDR

For a scanner, the LSP codebooks are decode-only lookup tables: TETRA voice arrives as
already-quantized indices, and reconstructing intelligible audio just requires the correct
table values, scale, and ordering. GopherTrunk's copies are validated as part of the wider
end-to-end conformance of the TETRA ACELP path against the ETSI reference codec, so a decoded
call's spectral envelope matches the reference decoder rather than merely sounding plausible.

## Sources

[^lsp]: [Line spectral pairs](https://en.wikipedia.org/wiki/Line_spectral_pairs) — Wikipedia, on the LSP representation of LPC filters and why it is used for robust quantization.
[^vq]: [Vector quantization](https://en.wikipedia.org/wiki/Vector_quantization) — Wikipedia, on codebook-based quantization and split-vector designs.
