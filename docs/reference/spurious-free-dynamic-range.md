---
slug: spurious-free-dynamic-range
title: Spurious-free dynamic range (SFDR)
entry_type: term
category: rf-metrics
description: SFDR is the range in decibels between a receiver's noise floor and the level where intermodulation spurs first emerge; it is set by the third-order intercept.
keywords: SFDR, spurious-free dynamic range, intermodulation dynamic range, IMD, third-order intermodulation, IP3, two-tone test, ADC SFDR, spurs
aka: [SFDR, intermodulation-free dynamic range, IMD dynamic range]
autolink: true
infobox:
  - { label: Symbol, value: "SFDR" }
  - { label: Unit, value: Decibels (dB or dBc) }
  - { label: Formula, value: "SFDR = ⅔·(IIP3 − noise floor)" }
see_also: [third-order-intercept, intermodulation, dynamic-range, blocking-dynamic-range, noise-floor, analog-to-digital-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/Spurious-free_dynamic_range
  - https://en.wikipedia.org/wiki/Intermodulation
---

**Spurious-free dynamic range** (**SFDR**) is the span, in
[decibels](/reference/decibel/), between a receiver's
[noise floor](/reference/noise-floor/) and the input level at which
[intermodulation](/reference/intermodulation/) products first rise above that
floor.[^wiki] It answers the question that matters in a crowded band: how strong can
interfering signals get before the receiver manufactures its own in-band spurs that
masquerade as real signals? SFDR is usually the tightest constraint on a receiver's
[dynamic range](/reference/dynamic-range/), and it is governed almost entirely by the
front end's [third-order intercept](/reference/third-order-intercept/) (IP3).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 185" role="img" aria-label="A log-log power plot with a linear wanted-tone line rising at slope one and a third-order intermodulation product rising at slope three, the two meeting at the third-order intercept, and SFDR marked from the noise floor to where the spur crosses it." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="55" y1="20" x2="55" y2="160" stroke="currentColor" stroke-width="1.3"/>
  <line x1="55" y1="160" x2="440" y2="160" stroke="currentColor" stroke-width="1.3"/>
  <text x="48" y="18" text-anchor="end" font-size="9" fill="currentColor">P_out</text>
  <text x="438" y="175" text-anchor="end" font-size="9" fill="currentColor">P_in</text>
  <line x1="55" y1="150" x2="330" y2="40" stroke="currentColor" stroke-width="1.6"/>
  <text x="300" y="52" font-size="9" fill="currentColor">wanted (slope 1)</text>
  <line x1="110" y1="160" x2="330" y2="40" stroke="currentColor" stroke-width="1.4" stroke-dasharray="4 3"/>
  <text x="150" y="150" font-size="9" fill="currentColor">3rd-order (slope 3)</text>
  <circle cx="330" cy="40" r="3.5" fill="currentColor"/>
  <text x="336" y="36" font-size="9.5" fill="currentColor">IP3</text>
  <line x1="55" y1="128" x2="440" y2="128" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 2"/>
  <text x="360" y="124" font-size="9" fill="currentColor">noise floor</text>
  <line x1="205" y1="128" x2="205" y2="95" stroke="currentColor" stroke-width="1.4" marker-end="url(#sfar)"/>
  <line x1="205" y1="95" x2="205" y2="128" stroke="currentColor" stroke-width="1.4" marker-end="url(#sfar)"/>
  <text x="212" y="112" font-size="10" fill="currentColor">SFDR</text>
</svg>
<figcaption>The wanted tone rises 1 dB per dB while third-order spurs rise 3 dB per dB; SFDR is the vertical window from the noise floor up to where the spur just clears it.</figcaption>
</figure>

## How it works

Drive a non-linear front end with two equal tones at f₁ and f₂. Its cubic
non-linearity generates third-order intermodulation products at 2f₁ − f₂ and
2f₂ − f₁, which fall right next to the originals and cannot be filtered out. The
geometry is what makes SFDR predictable:

- The wanted output grows **1 dB for every 1 dB** of input.
- The third-order product grows **3 dB for every 1 dB** of input.

Extrapolate the two lines and they meet at the **third-order intercept point**
(IP3). Because the spur closes on the wanted signal at 2 dB per dB, a little algebra
gives the input level at which the spur just reaches the noise floor, and hence:

**SFDR = ⅔·(IIP3 − noise floor)**

with everything in dB (IIP3 is the input-referred intercept). Every 3 dB of extra
IP3 buys 2 dB of SFDR; every 3 dB the noise floor drops buys 2 dB back. The result is
often quoted in **dB** relative to the floor, or in **dBc** relative to a full-scale
carrier for a converter.

## Variants

- **Analog front-end SFDR** is set by the LNA and mixer linearity via IP3, as above.
- **ADC/converter SFDR** is defined differently: for an
  [analog-to-digital converter](/reference/analog-to-digital-converter/) or DAC it is
  the ratio in dBc between a full-scale fundamental and the largest spurious tone
  (often a harmonic or a clock/quantization artifact) anywhere in the Nyquist band.
  A converter datasheet's SFDR figure describes how clean its spectrum is, closely
  related to but distinct from [ENOB](/reference/enob/).

## Relevance to SDR

SFDR is the number that decides whether a scanner survives a busy band. Trunking
systems, paging transmitters, and broadcast signals often sit within a few hundred
kHz of a weak control channel; if the front end's IP3 is low, two strong neighbours
breed a third-order spur that can land squarely on the wanted channel and either mask
it or fool the decoder with a phantom carrier. Cheap SDR dongles have modest IP3 and
low converter SFDR, which is why an out-of-band bandpass or cavity filter — removing
the strong signals before they reach the non-linear stages — so often rescues a
marginal setup. Higher-linearity front ends (Airspy R2/Discovery, SDRplay, USRP) push
IP3 and converter SFDR up and widen the spur-free window.

GopherTrunk operates on whatever the ADC delivered: if intermod spurs are already
baked into the samples, no DSP can distinguish them from genuine signals. When a
control channel decodes alone but breaks when strong local carriers are present,
SFDR — not the decoder — is the limit, and the fix is front-end filtering or lower
gain to keep the strong signals inside the linear region.

## Sources

[^wiki]: [Spurious-free dynamic range](https://en.wikipedia.org/wiki/Spurious-free_dynamic_range) — Wikipedia, SFDR definition for receivers and converters and its link to IP3.
