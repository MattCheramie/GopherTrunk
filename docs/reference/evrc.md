---
slug: evrc
title: EVRC
entry_type: technology
category: voice-coding
description: "EVRC is the Enhanced Variable Rate Codec, a variable-rate ACELP speech coder used for voice on CDMA2000 cellular networks."
keywords: EVRC, Enhanced Variable Rate Codec, CDMA2000, CDMA, ACELP, variable rate speech, 3GPP2, IS-127, cellular voice
aka: [EVRC]
autolink: true
infobox:
  - { label: Type, value: Variable-rate ACELP speech codec }
  - { label: Used by, value: CDMA2000 / 3GPP2 networks }
  - { label: Rates, value: Full / half / eighth rate }
see_also: [acelp, cdma2000, code-excited-linear-prediction, cdma, linear-predictive-coding, vocoder]
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Variable_Rate_Codec
  - https://en.wikipedia.org/wiki/CDMA2000
---

**EVRC** (**Enhanced Variable Rate Codec**) is a variable-rate speech coder based
on [ACELP](/reference/acelp/) that carries voice on
[CDMA2000](/reference/cdma2000/) and earlier IS-95 cellular
networks.[^wiki] Standardised by the TIA as IS-127 and adopted by 3GPP2, it
replaced the older QCELP coder and was tuned to exploit CDMA's soft-capacity
property: by dropping to a lower rate during pauses and quieter speech, it frees
channel capacity that other users' signals can occupy, increasing the number of
simultaneous callers a cell can hold.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="EVRC picks one of three coding rates per 20 millisecond frame based on voice activity, sending full rate for active speech, half rate for transitions and eighth rate for background silence." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="30" y="40" width="70" height="44"/><rect x="108" y="52" width="70" height="32"/><rect x="186" y="70" width="70" height="14"/><rect x="264" y="40" width="70" height="44"/><rect x="342" y="70" width="70" height="14"/>
  </g>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="65" y="98">full</text><text x="143" y="98">half</text><text x="221" y="98">1/8</text><text x="299" y="98">full</text><text x="377" y="98">1/8</text>
  </g>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="65" y="34">speech</text><text x="143" y="34">onset</text><text x="221" y="60">silence</text><text x="299" y="34">speech</text><text x="377" y="60">silence</text>
  </g>
  <text x="230" y="120" text-anchor="middle" font-size="8" fill="currentColor">rate chosen per 20 ms frame &#8594; lower rate returns capacity to the CDMA cell</text>
</svg>
<figcaption>EVRC varies its coding rate frame by frame with voice activity, saving CDMA cell capacity during silence.</figcaption>
</figure>

## How it works

EVRC is a linear-prediction, analysis-by-synthesis coder in the
[CELP](/reference/code-excited-linear-prediction/) family, specifically the
algebraic-codebook variant [ACELP](/reference/acelp/). Each 20 ms frame it fits a
short-term [LPC](/reference/linear-predictive-coding/) filter for the vocal-tract
spectrum, an adaptive codebook for the pitch (long-term) prediction, and searches a
sparse algebraic fixed codebook for the excitation that best reconstructs the frame
through those filters. Only the LPC coefficients, pitch lag and codebook indices are
sent.

What distinguishes EVRC is its rate-selection front end:

- A **rate-determination algorithm** classifies each frame by voice activity and
  chooses **full rate** (~8.5 kbps) for active speech, **half rate** (~4 kbps) for
  transitions and lower-energy voiced sound, or **eighth rate** (~0.8 kbps) for
  background noise and silence.
- Full and half rate use the full ACELP model; eighth rate sends only a coarse
  spectral and gain description to synthesise comfort noise, so silence costs almost
  nothing.
- The coder includes noise suppression and echo-handling preprocessing suited to the
  handset environment.

Later members of the family widened its reach: EVRC-B added more rate options and
better quality, and EVRC-WB extended it to wideband (16 kHz) audio for higher
fidelity, all keeping the same variable-rate structure.

## Relevance to SDR

EVRC is a cellular telephony vocoder, tightly bound to the CDMA physical layer
rather than to land-mobile trunking. It matters to SDR as the voice codec that rode
on top of [CDMA2000](/reference/cdma2000/)/[CDMA](/reference/cdma/) spreading — the
reason those networks could pack many voice users into one carrier was the interplay
between CDMA's soft capacity and EVRC's silence-driven rate reduction. It is not part
of the P25/DMR/NXDN/TETRA world that GopherTrunk decodes; those systems use the MBE
vocoder family, not ACELP. Recovering EVRC voice would also mean demodulating and
de-spreading an encrypted commercial cellular link, which is outside GopherTrunk's
scope. GopherTrunk does not implement EVRC. It is included here to place the
variable-rate ACELP branch of the vocoder family tree alongside the radio vocoders GT
actually handles.

## Sources

[^wiki]: [Enhanced Variable Rate Codec](https://en.wikipedia.org/wiki/Enhanced_Variable_Rate_Codec) — Wikipedia, on the variable-rate ACELP speech coder used across CDMA/CDMA2000 networks.
