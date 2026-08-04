---
slug: acelp-codebooks
title: ACELP adaptive & algebraic codebooks
entry_type: term
category: voice-coding
description: "The ACELP adaptive (pitch) and algebraic (fixed) codebooks build the TETRA excitation each subframe: the pitch codebook delays past excitation with 32-tap fractional interpolation, the algebraic codebook places four signed pulses."
keywords: ACELP codebook, adaptive codebook, pitch codebook, algebraic codebook, fixed codebook, four-pulse, fractional delay, interpolation filter, TETRA vocoder, ETSI EN 300 395-2
aka: [pitch codebook, "adaptive codebook", "algebraic codebook", "fixed codebook"]
autolink: true
infobox:
  - { label: Role, value: Excitation (innovation) builder }
  - { label: Adaptive, value: Past excitation at pitch lag T0 }
  - { label: Algebraic, value: "4 signed pulses over 60 samples" }
  - { label: Spec, value: ETSI EN 300 395-2 }
see_also: [acelp, code-excited-linear-prediction, linear-predictive-coding, acelp-lsp-codebooks, acelp-gain-quantization, tetra, vocoder]
cite_urls:
  - https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction
  - https://en.wikipedia.org/wiki/Code-excited_linear_prediction
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
---

The **ACELP codebooks** are the two sources of excitation that the
[TETRA](/reference/tetra/) [ACELP](/reference/acelp/) decoder combines each subframe to drive
the LPC synthesis filter: an **adaptive (pitch) codebook** for the periodic component of
voiced speech and an **algebraic (fixed) codebook** for the noise-like innovation.[^acelp] This
two-codebook excitation is the defining structure of every
[CELP](/reference/code-excited-linear-prediction/)-family coder; what makes it *algebraic* is
that the fixed codebook is a rule for placing a few signed pulses rather than a stored table.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="An adaptive codebook produces a periodic contribution by delaying past excitation by the pitch lag, an algebraic codebook produces a four-pulse innovation over sixty samples, and the two sum into the subframe excitation that drives the LPC synthesis filter." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="16" y="20" width="120" height="30"/><rect x="16" y="96" width="120" height="30"/><rect x="250" y="58" width="52" height="30"/><rect x="360" y="58" width="92" height="30"/></g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="76" y="32">adaptive codebook</text><text x="76" y="43">delay T0 + frac/3</text>
    <text x="76" y="108">algebraic codebook</text><text x="76" y="119">4 signed pulses</text>
    <text x="276" y="76">Σ gains</text>
    <text x="406" y="72">1/A(z)</text><text x="406" y="82">synthesis</text>
  </g>
  <g font-size="8" fill="currentColor"><line x1="150" y1="60" x2="200" y2="60"/><line x1="200" y1="60" x2="200" y2="100"/><line x1="150" y1="111" x2="200" y2="111"/></g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="136" y1="35" x2="250" y2="66" marker-end="url(#cbar)"/>
    <line x1="136" y1="111" x2="250" y2="80" marker-end="url(#cbar)"/>
    <line x1="302" y1="73" x2="360" y2="73" marker-end="url(#cbar)"/>
    <line x1="452" y1="73" x2="466" y2="73" marker-end="url(#cbar)"/>
  </g>
</svg>
<figcaption>Each subframe the pitch and algebraic codebooks are scaled by their gains and summed into the excitation that drives the LPC synthesis filter.</figcaption>
</figure>

## The adaptive (pitch) codebook

Voiced speech is quasi-periodic at the pitch, so the cheapest way to model its excitation is to
reuse the recent past. The **adaptive codebook** does exactly that: for each of the 60 samples
in a subframe it copies the excitation from `T0` samples earlier, where `T0` is the transmitted
pitch lag. Because true pitch rarely lands on an integer sample, ACELP resolves the lag to
**±1/3 of a sample**. When the fractional part is non-zero the decoder interpolates with a
**32-tap fractional-delay filter** — GopherTrunk stores the two tap sets, `inter32Coef1_3` and
`inter32CoefM1_3`, for the +1/3 and −1/3 offsets in `internal/voice/acelp/codebook.go`. The
`predLt` routine (a port of the reference `Pred_Lt`) selects the integer-copy path for `frac ==
0` and the interpolating path otherwise, reconstructing the periodic component sample by sample.

## The algebraic (fixed) codebook

Whatever periodicity the pitch codebook cannot explain — the noise-like innovation, transients,
and unvoiced energy — is supplied by the **algebraic codebook**. TETRA's is a **four-pulse**
design over the 60-sample subframe: a 14-bit index encodes four pulse positions on interleaved
tracks, each pulse carrying amplitude +1 or −1. GopherTrunk's `algebraicPulses` unpacks the
index into the four positions, and `dD4i60` (a port of the reference `D_D4i60`) builds the
codeword and convolves it with the perceptual noise-shaping filter `F[]`. The first pulse is
scaled by √2 — the constant `q11GainI0 = 2896`, √2 in Q11 — and a whole-codeword sign flip and a
0/1 sample shift complete the reconstruction. Storing only four positions and signs is what lets
ACELP represent a very large excitation space in a handful of bits, the property that gave the
algorithm its name and its efficiency.

## Combining the two

The subframe excitation is the sum of the two codebook contributions, each multiplied by its own
gain from the [gain dequantizer](/reference/acelp-gain-quantization/). That excitation drives the
all-pole LPC synthesis filter whose coefficients come from the
[LSP codebooks](/reference/acelp-lsp-codebooks/), and the filter output — after the decoder's
post-processing — is the reconstructed speech. Crucially, today's excitation becomes tomorrow's
history: the pitch codebook on the next subframe delays *this* subframe's excitation, so the two
codebooks are coupled through a feedback buffer. That coupling is why a scanner must carry enough
valid excitation history before each subframe (`predLt`'s caller guarantees `pitMax + lInter`
samples of it) and why an erased frame must run concealment rather than simply muting — a gap in
the history would corrupt every subsequent pitch prediction.

## Relevance to SDR

For decoding, the codebooks are deterministic reconstruction rules: given the transmitted pitch
lag, fraction, algebraic index, sign, and shift, they rebuild the exact excitation the encoder
chose. The heavy analysis-by-synthesis *search* that picks those parameters lives only in the
encoder, so GopherTrunk's TETRA voice path implements just the lightweight synthesis side. The
codebook ports are exercised as part of the ACELP decoder's end-to-end conformance against the
ETSI reference codec, so a decoded call's excitation is bit-faithful rather than merely
plausible.

## Sources

[^acelp]: [Algebraic code-excited linear prediction](https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction) — Wikipedia, on the adaptive and algebraic codebook structure of ACELP.
[^celp]: [Code-excited linear prediction](https://en.wikipedia.org/wiki/Code-excited_linear_prediction) — Wikipedia, on the CELP excitation model the codebooks implement.
