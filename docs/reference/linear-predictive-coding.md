---
slug: linear-predictive-coding
title: Linear predictive coding (LPC)
entry_type: algorithm
category: voice-coding
description: LPC is a speech-coding method that models the vocal tract as an all-pole filter driven by an excitation signal, sending a few predictor coefficients at low bit rate; it underlies most digital-voice vocoders.
keywords: LPC, linear predictive coding, all-pole filter, vocal tract model, predictor coefficients, LPC-10, source-filter model, speech coding, vocoder, CELP, MELP, formants, LSP
aka: [LPC, linear prediction]
autolink: true
infobox:
  - { label: Type, value: Speech analysis/synthesis }
  - { label: Model, value: All-pole vocal-tract filter }
  - { label: Underlies, value: CELP, MELP, most vocoders }
see_also: [vocoder, code-excited-linear-prediction, melp, codec2, multi-band-excitation, imbe]
cite_urls:
  - https://en.wikipedia.org/wiki/Linear_predictive_coding
  - https://en.wikipedia.org/wiki/Source%E2%80%93filter_model_of_speech_production
---

**Linear predictive coding (LPC)** is a speech-coding method that models the human vocal
tract as an **all-pole digital filter** driven by an excitation signal, transmitting only
a handful of predictor coefficients instead of the raw waveform.[^wiki] Because those few
parameters capture the resonances of speech, LPC compresses voice to very low bit rates and
forms the analytical core of most digital-voice [vocoders](/reference/vocoder/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An excitation source, either a pitched pulse train for voiced speech or noise for unvoiced speech, drives an all-pole LPC filter that shapes it into synthetic speech." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="24" width="96" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="68" y="41" font-size="8">voiced: pulse train</text>
    <rect x="20" y="70" width="96" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="68" y="87" font-size="8">unvoiced: noise</text>
    <text x="150" y="63" font-size="8">voicing</text><text x="150" y="74" font-size="8">switch</text>
    <line x1="116" y1="37" x2="176" y2="55" stroke="currentColor" marker-end="url(#lpcar)"/>
    <line x1="116" y1="83" x2="176" y2="61" stroke="currentColor" marker-end="url(#lpcar)"/>
    <rect x="212" y="42" width="104" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="264" y="58">all-pole filter</text><text x="264" y="70" font-size="8">1 / A(z)</text>
    <text x="264" y="26" font-size="8">predictor coeffs ↓</text><line x1="264" y1="28" x2="264" y2="42" stroke="currentColor"/>
    <line x1="316" y1="60" x2="360" y2="60" stroke="currentColor" marker-end="url(#lpcar)"/>
    <text x="400" y="63">speech</text>
  </g>
  <defs><marker id="lpcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>LPC synthesises speech by passing an excitation — pitched pulses or noise — through an all-pole filter whose few coefficients describe the vocal tract.</figcaption>
</figure>

## How it works

LPC rests on the **source–filter model** of speech: the vocal cords (or turbulent airflow)
produce an excitation, and the throat, mouth, and nasal cavity shape it into recognisable
sounds. The shaping is modelled as a **linear predictor** — each speech sample is estimated
as a weighted sum of the previous *p* samples:

- **Analysis.** For each short frame (typically 20–30 ms), the encoder solves for the
  predictor coefficients *a₁…aₚ* that minimise the energy of the prediction error, usually
  via the autocorrelation method and the Levinson–Durbin recursion. Ten to sixteen
  coefficients suffice for telephone-band speech; the poles of the resulting filter sit at
  the speech **formants** (vocal-tract resonances).
- **Residual.** What the predictor cannot foresee is the **residual** or excitation signal.
  Classic LPC vocoders reduce it to two decisions — voiced or unvoiced, and a pitch period
  — and send a single gain. This is the aggressive compression that gives 1970s-era
  LPC-10 (2.4 kbps) its characteristic buzzy, synthetic sound.
- **Synthesis.** The decoder rebuilds the excitation (a pulse train at the pitch rate for
  voiced frames, white noise for unvoiced) and runs it through the all-pole filter
  reconstructed from the coefficients, producing audible speech.

Coefficients are rarely sent as raw *aₚ* values, which are sensitive to quantisation.
Instead they are converted to more robust equivalents — reflection coefficients, log-area
ratios, or **line spectral pairs (LSPs)** — before being packed into the frame.

## Variants

Plain LPC's two-state excitation is its weakness: real speech is rarely purely voiced or
purely noisy. Later families keep the LPC vocal-tract filter but model the excitation far
more richly. [Code-excited linear prediction (CELP)](/reference/code-excited-linear-prediction/)
picks the excitation from a codebook by analysis-by-synthesis, and
[MELP](/reference/melp/) uses a mixed voiced/unvoiced excitation per frequency band. Both
sound dramatically better at the same bit rate. Even the
[multi-band-excitation](/reference/multi-band-excitation/) family used in land-mobile radio,
though not strictly LPC, shares the same source–filter philosophy of sending pitch, voicing,
and spectral-envelope parameters rather than the waveform.

## Relevance to SDR

Almost every low-rate digital-voice mode a scanner meets is an LPC descendant. The
[IMBE](/reference/imbe/) and AMBE vocoders of P25 and DMR, the MELPe used in military HF,
and open [Codec 2](/reference/codec2/) used by M17 all build on the same linear-prediction
foundation of compressing speech to a few kbps by modelling the vocal tract. GopherTrunk
does not run a standalone LPC-10 decoder, but understanding LPC explains *why* digital voice
sounds the way it does and why a corrupted spectral-parameter frame degrades far more
audibly than a corrupted PCM sample — the parameters, not the samples, carry the speech.

## Sources

[^wiki]: [Linear predictive coding](https://en.wikipedia.org/wiki/Linear_predictive_coding) — Wikipedia, for the all-pole model, coefficient estimation, and LPC-10 history.
