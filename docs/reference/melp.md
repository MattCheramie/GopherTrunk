---
slug: melp
title: MELP / MELPe
entry_type: algorithm
category: voice-coding
description: MELP is a very-low-rate speech vocoder using mixed voiced/unvoiced excitation over an LPC filter at 2400 or 1200 bps; its enhanced MELPe form is the NATO STANAG 4591 standard for military voice.
keywords: MELP, MELPe, mixed-excitation linear prediction, STANAG 4591, MIL-STD-3005, 2400 bps, 1200 bps, low rate voice, military vocoder, secure voice, HF radio
aka: [MELP, MELPe, mixed-excitation linear prediction]
autolink: true
infobox:
  - { label: Type, value: Low-rate LPC vocoder }
  - { label: Bit rate, value: 2400 / 1200 / 600 bps }
  - { label: Standard, value: NATO STANAG 4591 }
see_also: [linear-predictive-coding, code-excited-linear-prediction, vocoder, codec2, imbe]
cite_urls:
  - https://en.wikipedia.org/wiki/Mixed-excitation_linear_prediction
  - https://en.wikipedia.org/wiki/STANAG
---

**MELP (Mixed-Excitation Linear Prediction)** is a very-low-rate speech
[vocoder](/reference/vocoder/) that improves on classic
[LPC](/reference/linear-predictive-coding/) by driving the vocal-tract filter with a **mixed
voiced/unvoiced excitation** that varies across frequency, delivering intelligible speech at
just 2400 or 1200 bps.[^wiki] Its enhanced revision, **MELPe**, is standardised as **NATO
STANAG 4591** and is the primary low-rate voice codec for military and secure HF/UHF
communications.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="MELP splits the excitation spectrum into frequency bands, mixing a pitched pulse component and a noise component per band before the LPC synthesis filter, so each band can be partly voiced and partly noisy." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="18" y="24" width="86" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="61" y="40" font-size="8">pitch pulses</text>
    <rect x="18" y="76" width="86" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="61" y="92" font-size="8">noise</text>
    <text x="150" y="52" font-size="8">per-band</text><text x="150" y="63" font-size="8">mix</text>
    <line x1="104" y1="36" x2="182" y2="56" stroke="currentColor" marker-end="url(#melar)"/>
    <line x1="104" y1="88" x2="182" y2="62" stroke="currentColor" marker-end="url(#melar)"/>
    <rect x="184" y="44" width="72" height="32" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="220" y="58" font-size="8">band split</text><text x="220" y="69" font-size="8">V/U weights</text>
    <line x1="256" y1="60" x2="288" y2="60" stroke="currentColor" marker-end="url(#melar)"/>
    <rect x="290" y="44" width="72" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="326" y="58" font-size="8">LPC synth</text><text x="326" y="69" font-size="8">1/A(z)</text>
    <line x1="362" y1="60" x2="396" y2="60" stroke="currentColor" marker-end="url(#melar)"/>
    <text x="424" y="63" font-size="8">speech</text>
  </g>
  <defs><marker id="melar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>MELP mixes pulse and noise excitation independently in each frequency band, so a frame can be voiced at low frequencies and noisy at high — cured of LPC's buzzy either/or.</figcaption>
</figure>

## How it works

Classic LPC-10 forces every frame to be either fully voiced (a pitch pulse train) or fully
unvoiced (noise), which produces its notorious buzzy, mechanical timbre. MELP fixes this by
enriching the excitation while keeping the same low-rate LPC spectral model:

- **Band-wise mixed excitation.** The excitation spectrum is divided into frequency bands
  (five in the standard), and each is assigned a **voicing strength** blending a periodic
  pulse component with noise. Voiced speech with breathy or fricative high frequencies is
  represented naturally instead of being forced into one category.
- **Aperiodic pulses.** A jitter flag lets the pulse train be slightly irregular, capturing
  the roughness of transitional and creaky voicing.
- **Adaptive spectral enhancement** and a **pulse-dispersion filter** sharpen formants and
  smooth the synthetic buzz.
- **LPC envelope.** As in all LPC coders, line-spectral-pair coefficients carry the
  vocal-tract shape; the encoder also transmits pitch, gain, and the band voicing flags —
  all packed into a 54-bit frame every 22.5 ms at 2400 bps.

The result is markedly more natural and robust in noise than LPC-10 at the same or lower
rate — the reason it displaced older federal-standard vocoders.

## Variants

The original **MELP** is US Federal Standard MIL-STD-3005 at 2400 bps. **MELPe** (enhanced
MELP, STANAG 4591) adds a noise pre-processor, better analysis, and two extra rates —
**1200 bps** and **600 bps** — while remaining interoperable with the 2400 bps mode. MELPe
won the NATO selection over competitors and is embedded in tactical radios worldwide.

## Relevance to SDR

MELP/MELPe is the vocoder of **military and government HF/tactical voice**, usually wrapped in
strong encryption, rather than of the civilian trunked systems GopherTrunk targets — those use
the [multi-band-excitation](/reference/multi-band-excitation/) family such as
[IMBE](/reference/imbe/) and AMBE+2. GopherTrunk does not decode MELPe (and where it appears
the audio is typically encrypted anyway). It belongs on the map as the extreme low-rate,
mixed-excitation cousin of the [CELP](/reference/code-excited-linear-prediction/) and MBE
families, showing how far speech can be compressed while staying intelligible.

## Sources

[^wiki]: [Mixed-excitation linear prediction](https://en.wikipedia.org/wiki/Mixed-excitation_linear_prediction) — Wikipedia, for band-wise mixed excitation, bit rates, and STANAG 4591.
