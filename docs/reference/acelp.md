---
slug: acelp
title: ACELP
entry_type: algorithm
category: voice-coding
description: "ACELP (Algebraic Code-Excited Linear Prediction) is a CELP speech coder that excites an LPC filter from a sparse algebraic codebook; it underpins AMR, G.729, and TETRA voice."
keywords: ACELP, Algebraic Code-Excited Linear Prediction, CELP, algebraic codebook, LPC, speech coding, vocoder, AMR, G.729, TETRA, AMR-WB, EVS, fixed codebook
aka: [ACELP, Algebraic CELP]
autolink: true
infobox:
  - { label: Type, value: Speech-coding algorithm (CELP) }
  - { label: Excitation, value: Sparse algebraic fixed codebook }
  - { label: Used by, value: "AMR, G.729, TETRA, EVS" }
see_also: [code-excited-linear-prediction, linear-predictive-coding, tetra, amr, g729, vocoder]
cite_urls:
  - https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction
  - https://en.wikipedia.org/wiki/Code-excited_linear_prediction
---

**ACELP** (**Algebraic Code-Excited Linear Prediction**) is a variant of
[code-excited linear prediction](/reference/code-excited-linear-prediction/) in which the
fixed excitation codebook is not a stored table but an *algebraic* structure: a small number
of unit pulses whose positions and signs are chosen on the fly.[^wiki] Because the codebook
is defined by a rule rather than memory, the encoder can search a very large excitation space
cheaply, which is why ACELP became the dominant low-bitrate speech algorithm in cellular and
land-mobile radio — it is the engine inside [AMR](/reference/amr/),
[G.729](/reference/g729/), and the [TETRA](/reference/tetra/) voice codec.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An algebraic codebook of sparse signed pulses is added to an adaptive pitch codebook, scaled by gains, and passed through an LPC synthesis filter compared against the input speech in a closed analysis-by-synthesis loop." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="acar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.2" fill="none" font-size="8">
    <rect x="16" y="20" width="86" height="34"/><rect x="16" y="78" width="86" height="34"/>
    <rect x="180" y="49" width="60" height="34"/>
    <rect x="300" y="49" width="70" height="34"/>
    <rect x="300" y="100" width="140" height="26"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="59" y="34">algebraic</text><text x="59" y="45">codebook</text>
    <text x="59" y="92">adaptive</text><text x="59" y="103">(pitch)</text>
    <text x="210" y="63">Σ · gains</text>
    <text x="335" y="63">LPC</text><text x="335" y="74">synth</text>
    <text x="370" y="116">minimise weighted error</text>
    <text x="415" y="44">≈ speech</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="102" y1="37" x2="180" y2="55" marker-end="url(#acar)"/>
    <line x1="102" y1="95" x2="180" y2="77" marker-end="url(#acar)"/>
    <line x1="240" y1="66" x2="300" y2="66" marker-end="url(#acar)"/>
    <line x1="370" y1="66" x2="410" y2="66" marker-end="url(#acar)"/>
    <line x1="335" y1="83" x2="335" y2="100" marker-end="url(#acar)"/>
    <line x1="300" y1="113" x2="200" y2="113"/><line x1="200" y1="113" x2="200" y2="83" marker-end="url(#acar)"/>
  </g>
</svg>
<figcaption>ACELP excites an LPC synthesis filter from a sparse algebraic pulse codebook plus an adaptive pitch codebook, choosing pulses by analysis-by-synthesis.</figcaption>
</figure>

## How it works

ACELP inherits the CELP frame: [linear predictive coding](/reference/linear-predictive-coding/)
models the vocal-tract spectrum as a short-term synthesis filter, an *adaptive* (pitch)
codebook supplies the periodic component of the excitation, and a *fixed* codebook supplies
the remaining noise-like innovation. What makes it algebraic is the form of that fixed
codebook. Each excitation subframe is divided into interleaved *tracks*, and each track may
hold exactly one non-zero pulse of amplitude +1 or −1 at one of a few allowed positions. A
codeword is therefore just a list of pulse positions and signs — no vectors are stored at all.

The encoder runs **analysis-by-synthesis**: it tries candidate excitations, synthesises speech
through the LPC filter, and keeps the one that minimises a *perceptually weighted* error
(weighting deemphasises error near the formant peaks where it is masked). A brute-force search
over every pulse combination would be huge, so ACELP's key trick is that the sparse, signed-unit
structure lets the search be reformulated as maximising a simple correlation term, pruned with
nested loops and depth-first pulse placement. That is the whole reason the algorithm exists: it
delivers near-optimal excitation quality at a search cost cheap enough for a 1990s DSP.

## Variants

- **Bitrate families** — G.729 uses 8 kbps narrowband ACELP; AMR spans 4.75–12.2 kbps by
  changing the number of pulses and gain resolution; AMR-WB and 3GPP
  [EVS](/reference/evs-codec/) extend ACELP to 16 kHz wideband and beyond.
- **Algebraic vs stored codebooks** — earlier CELP coders (e.g. the original DoD CELP) searched
  stored Gaussian codebooks; ACELP replaced them with the pulse-track structure, cutting both
  memory and search complexity.
- **Hybrid coders** — modern super-wideband codecs switch between an ACELP core for speech and a
  transform/MDCT core for music, choosing per frame.

## In practice

ACELP is heavily patented (originally by Université de Sherbrooke / VoiceAge), which shaped the
whole ecosystem: standards bodies licensed it for cellular and TETRA, while royalty-free projects
deliberately avoided it. Its analysis-by-synthesis loop makes the *encoder* far more expensive
than the decoder, so a scanner only ever needs the lightweight synthesis side.

## Relevance to SDR

ACELP is not a modulation but the speech-compression core carried inside several digital-radio
payloads a receiver may meet. [TETRA](/reference/tetra/) voice frames carry an ACELP codec at
4.567 kbps (plus [AMR](/reference/amr/)-style channel coding), and GSM cellular voice moved to
AMR's ACELP modes. GopherTrunk's decode focus is land-mobile trunking whose vocoders are the
MBE family rather than ACELP, so ACELP appears here as background on how competing digital-voice
systems compress speech, not as a codec GopherTrunk itself renders.

## Sources

[^wiki]: [Algebraic code-excited linear prediction](https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction) — Wikipedia, on the algebraic fixed codebook, pulse-track structure, and standards that use ACELP.
