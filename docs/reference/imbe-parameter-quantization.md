---
slug: imbe-parameter-quantization
title: IMBE parameter quantization
entry_type: term
category: voice-coding
description: IMBE parameter quantization is the mapping between the 88 recovered voice bits of a P25 Phase 1 frame and the MBE model parameters — the fundamental frequency, the per-band voicing decisions, and the PRBA and HOC spectral amplitudes.
keywords: IMBE parameter quantization, b0 fundamental frequency, voicing decisions, PRBA, HOC, spectral amplitudes, TIA-102.BABA 5.3, Annex E, MBE model parameters
aka: [IMBE parameter unpacking, "b_0 decode", "IMBE quantizer"]
autolink: true
infobox:
  - { label: Input, value: 88 information bits }
  - { label: Fundamental, value: "b0 at positions {0–5, 85, 86}" }
  - { label: Voicing, value: "K = ⌈(L+2)/3⌉ decisions" }
  - { label: Spec, value: "TIA-102.BABA §5.3 / Annex E" }
see_also: [multi-band-excitation, quantization, imbe, imbe-channel-coding, discrete-fourier-transform, mbe-voiced-synthesis, mbe-unvoiced-synthesis]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Quantization_(signal_processing)
---

**IMBE parameter quantization** is the step that gives the 88 error-corrected bits of a
[P25 Phase 1](/reference/imbe/) voice frame their meaning: it unpacks them into the
[multi-band excitation](/reference/multi-band-excitation/) model parameters the synthesizer needs
— a fundamental (pitch) frequency, the number of harmonics, a voiced/unvoiced decision per band,
and the spectral amplitude at every harmonic.[^mbe] It is the inverse of the encoder's
[quantization](/reference/quantization/) (TIA-102.BABA §5.3 / Annex E), and it is where a flat
list of bits becomes a speech spectrum.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 140" role="img" aria-label="The 88-bit information frame is divided into fields: the eight fundamental-frequency bits read from scattered positions zero through five plus eighty-five and eighty-six, followed by the voicing-decision bits, the gain index, and the PRBA and HOC spectral-amplitude blocks that fill the remainder." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="14" y="34" width="60" height="28" fill="currentColor" fill-opacity="0.26"/><text x="44" y="51">b0 (8)</text>
    <rect x="74" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.16"/><text x="109" y="48">voicing</text><text x="109" y="57" font-size="6.5">K bits</text>
    <rect x="144" y="34" width="52" height="28" fill="currentColor" fill-opacity="0.16"/><text x="170" y="48">b2 gain</text><text x="170" y="57" font-size="6.5">6</text>
    <rect x="196" y="34" width="120" height="28" fill="none"/><text x="256" y="51">PRBA gain blocks</text>
    <rect x="316" y="34" width="132" height="28" fill="none"/><text x="382" y="51">HOC spectral blocks</text>
  </g>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="44" y="78">ω₀, L, K</text>
    <text x="256" y="94">inverse DCT → Tl[1..L]</text>
  </g>
  <text x="231" y="122" text-anchor="middle" font-size="8" fill="currentColor">88 bits → fundamental + per-band voicing + harmonic amplitudes</text>
</svg>
<figcaption>The 88-bit frame partitions into the scattered b0 fundamental, the K voicing bits, a 6-bit gain index, and the PRBA and HOC spectral-amplitude blocks that an inverse DCT turns into the per-harmonic residuals.</figcaption>
</figure>

## The fundamental frequency

Everything begins with `b_0`, the fundamental-frequency parameter — and it is not contiguous. Its
8 bits are read from *scattered* positions in the frame: bits 0–5 plus bits 85 and 86, packed
MSB-first. From `b_0` the decoder derives the fundamental frequency in radians per sample,
`ω₀ = 4π / (b_0 + 39.5)`, then the harmonic count `L = ⌊0.9254 · ⌊π/ω₀ + 0.25⌋⌋`, which the model
constrains to the range 9–56. Preserving that inner integer truncation matters at the boundary
cases. Values of `b_0` above 207 are special: the narrow window 216–219 signals a silence frame,
and any other value above 207 marks an invalid frame the decoder repeats rather than voices. A
sustained run of the lowest `b_0` values (0–7, the ~340–405 Hz, L = 9/10 corner) is the signature
of an idle or settling carrier, not speech, and GopherTrunk mutes that buzz.

## Voicing decisions

The number of voicing decisions is `K = ⌈(L+2)/3⌉` for `L < 37`, and 12 otherwise — one binary
voiced/unvoiced flag for each group of three harmonics, which is how the model fits up to 56
harmonic decisions into a handful of bits. To read them, the remaining 79 bits (positions 6–84)
are first *re-ordered* through an L-indexed bit-order table into a `bb[v][p]` layout; the voicing
bits then live in the first re-ordered vector. Each group of three consecutive harmonics shares one
decision, which the synthesizer expands into per-harmonic voiced or unvoiced treatment.

## PRBA and HOC spectral amplitudes

The spectral envelope is the largest part of the payload and the most heavily structured. A 6-bit
gain index `b_2` selects the overall level from a lookup table. Five **PRBA** (prediction-residual
block-amplitude) blocks then supply a coarse gain vector, each block read MSB-first and dequantized
as `step × (value − 2^(bits−1) + 0.5)`. An inverse 6-point DCT-II turns those into per-band DC
terms. The finer detail comes from the **HOC** (higher-order coefficient) blocks, whose per-slot
bit allocations are table-driven and dequantized with a shared quantizer step scaled by a
per-coefficient standard deviation; a zero bit allocation simply means that coefficient is absent.
A second inverse [DCT](/reference/discrete-fourier-transform/) per band expands the coefficients
into the log-amplitude residuals `Tl[1..L]`. These are residuals *before* the cross-frame
log-amplitude prediction, which needs the previous frame's state and lives in the synthesizer, not
in the unpack.

## Relevance to SDR

Parameter quantization is the bridge between GopherTrunk's [channel-coding](/reference/imbe-channel-coding/)
layer and its synthesizer: the FEC hands over 88 clean bits, and this stage turns them into the
`ω₀`, `L`, voicing decisions, and `Tl` residuals the harmonic and noise generators consume. It is
transcribed to match mbelib's reference unpack exactly, because every arithmetic detail — the
scattered `b_0` positions, the integer truncation in `L`, the DCT normalisation — changes the
decoded pitch or envelope audibly. An error here does not crash; it produces the wrong voice, which
is why the unpack is validated against reference vectors rather than only round-tripped.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE model parameters (pitch, voicing, spectral amplitudes) this stage recovers.
