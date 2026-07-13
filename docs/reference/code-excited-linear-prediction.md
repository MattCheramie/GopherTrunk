---
slug: code-excited-linear-prediction
title: Code-excited linear prediction (CELP)
entry_type: algorithm
category: voice-coding
description: CELP is a speech-coding method that pairs an LPC vocal-tract filter with an excitation chosen from a codebook by analysis-by-synthesis; it is the basis of most cellular and VoIP voice codecs.
keywords: CELP, code-excited linear prediction, analysis-by-synthesis, LPC, codebook excitation, ACELP, adaptive codebook, AMR, G.729, EVRC, Speex, cellular voice codec, VoIP codec
aka: [CELP, code-excited linear prediction]
autolink: true
infobox:
  - { label: Type, value: Speech codec (LPC + codebook) }
  - { label: Method, value: Analysis-by-synthesis }
  - { label: Used by, value: GSM, 3G, VoIP (G.729, AMR) }
see_also: [linear-predictive-coding, vocoder, melp, codec2, multi-band-excitation, imbe]
cite_urls:
  - https://en.wikipedia.org/wiki/Code-excited_linear_prediction
  - https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction
---

**Code-excited linear prediction (CELP)** is a speech-coding method that combines an
[LPC](/reference/linear-predictive-coding/) vocal-tract filter with an excitation signal
selected from a **codebook** by **analysis-by-synthesis** — the encoder searches the
codebook for the entry that, once filtered, best reproduces the original speech.[^wiki] This
closed-loop search is what lets CELP sound natural at 4–16 kbps, and it underpins nearly
every cellular and [VoIP](/reference/voice-channel/) voice codec.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An encoder tries codebook entries through an LPC synthesis filter, compares each output against the input speech, and transmits the index of the codeword giving the smallest perceptually weighted error." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="70" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="57" font-size="8">codebook</text><text x="55" y="69" font-size="8">of vectors</text>
    <line x1="90" y1="60" x2="126" y2="60" stroke="currentColor" marker-end="url(#celar)"/>
    <rect x="128" y="42" width="84" height="36" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="170" y="58">LPC synth</text><text x="170" y="70" font-size="8">filter 1/A(z)</text>
    <line x1="212" y1="60" x2="248" y2="60" stroke="currentColor" marker-end="url(#celar)"/>
    <circle cx="266" cy="60" r="14" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="266" y="64" font-size="12">−</text>
    <text x="266" y="24" font-size="8">input speech</text><line x1="266" y1="26" x2="266" y2="46" stroke="currentColor" marker-end="url(#celar)"/>
    <line x1="280" y1="60" x2="316" y2="60" stroke="currentColor" marker-end="url(#celar)"/>
    <rect x="318" y="42" width="84" height="36" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="360" y="58" font-size="8">minimise</text><text x="360" y="70" font-size="8">weighted error</text>
    <path d="M360 78 L360 118 L55 118 L55 80" fill="none" stroke="currentColor" stroke-dasharray="3 2" marker-end="url(#celar)"/>
    <text x="200" y="132" font-size="8">choose best codeword index → transmit</text>
  </g>
  <defs><marker id="celar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>CELP filters each candidate codebook vector through the LPC model, compares it to the input, and sends only the index of the best match — analysis-by-synthesis.</figcaption>
</figure>

## How it works

CELP keeps LPC's insight that speech is an excitation passed through a vocal-tract filter,
but replaces LPC's crude voiced/unvoiced pulse-or-noise excitation with a much better one:

- **Short-term (LPC) filter.** Predictor coefficients modelling the spectral envelope
  (formants) are found per frame, exactly as in plain LPC.
- **Long-term / adaptive codebook.** A pitch predictor supplies the periodicity of voiced
  speech from recently synthesised excitation.
- **Fixed codebook.** A stochastic (or, in ACELP, algebraic) codebook supplies the residual
  detail. The encoder tries entries, gains, and pitch lags, filters each candidate through
  the LPC synthesis filter, and keeps the combination that minimises a **perceptually
  weighted** error — error energy is shaped to hide where the ear masks it.
- **Transmit indices, not waveforms.** Only the LPC coefficients, pitch lag, codebook
  indices, and gains are sent. The decoder is a simple copy of the encoder's synthesis path,
  so it needs no search.

The genius is that both encoder and decoder share the same synthesis filter, so the encoder
can *hear its own output* and optimise against it — the closed loop is why CELP outperforms
open-loop LPC at the same rate.

## Variants

The dominant flavour is **ACELP** (algebraic CELP), whose sparse algebraic codebook makes the
search cheap and powers G.729, AMR (GSM/3G), AMR-WB, and EVS. Other members include
Qualcomm's EVRC/QCELP (CDMA cellular), LD-CELP (G.728, low delay), and the open Speex codec.

## Relevance to SDR

CELP is the codec family of the *phone network* rather than trunked land-mobile radio, so
it is worth contrasting with what a scanner actually decodes. GSM, UMTS, LTE VoLTE, and most
VoIP run CELP variants. Public-safety radio instead uses the
[multi-band-excitation](/reference/multi-band-excitation/) family —
[IMBE](/reference/imbe/) for P25 Phase 1 and AMBE+2 for DMR and P25 Phase 2 — which model
the speech spectrum band-by-band rather than by codebook search. Open
[Codec 2](/reference/codec2/) sits closer to the MBE philosophy than to CELP. GopherTrunk
does not implement a CELP decoder, because the systems it targets do not use CELP; knowing the
distinction clarifies why land-mobile vocoders sound different from a mobile-phone call.

## Sources

[^wiki]: [Code-excited linear prediction](https://en.wikipedia.org/wiki/Code-excited_linear_prediction) — Wikipedia, for analysis-by-synthesis, codebooks, and the codec lineage.
