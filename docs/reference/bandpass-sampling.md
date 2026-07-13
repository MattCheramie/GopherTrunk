---
slug: bandpass-sampling
title: Bandpass sampling (undersampling)
entry_type: term
category: sdr-dsp
description: "Bandpass sampling digitises a signal at less than its carrier frequency, using controlled aliasing to fold a higher Nyquist zone down to baseband."
keywords: bandpass sampling, undersampling, sub-Nyquist sampling, Nyquist zones, intentional aliasing, IF sampling, direct RF sampling, harmonic sampling
aka: [undersampling, sub-Nyquist sampling, harmonic sampling, IF sampling]
autolink: true
infobox:
  - { label: Type, value: Sampling technique }
  - { label: Rule, value: "Rate ≥ 2 × bandwidth (not 2 × frequency)" }
  - { label: Uses, value: Aliasing on purpose }
see_also: [nyquist-theorem, aliasing, direct-sampling, sample-rate, analog-to-digital-converter, intermediate-frequency]
cite_urls:
  - https://en.wikipedia.org/wiki/Undersampling
  - https://en.wikipedia.org/wiki/Nyquist_rate
---

**Bandpass sampling** — also called **undersampling** — digitises a band-limited signal
at a [sample rate](/reference/sample-rate/) **below** its carrier frequency, deliberately
using [aliasing](/reference/aliasing/) to fold a high-frequency band down onto
baseband.[^wiki] The usual reading of the [Nyquist theorem](/reference/nyquist-theorem/)
— sample at twice the *highest* frequency — is a special case; for a signal confined to a
narrow band, what actually matters is twice the **bandwidth**, not twice the top
frequency. That relaxation lets a modest ADC capture a signal sitting far above its own
sample rate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A frequency axis divided into successive Nyquist zones by multiples of half the sample rate, with a narrow signal band located in a higher zone folding down into the first zone at baseband." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="105" x2="445" y2="105" stroke="currentColor"/>
  <g font-size="8" fill="currentColor">
    <line x1="20" y1="100" x2="20" y2="110" stroke="currentColor"/><text x="10" y="124">0</text>
    <line x1="120" y1="100" x2="120" y2="110" stroke="currentColor"/><text x="108" y="124">fs/2</text>
    <line x1="220" y1="100" x2="220" y2="110" stroke="currentColor"/><text x="212" y="124">fs</text>
    <line x1="320" y1="100" x2="320" y2="110" stroke="currentColor"/><text x="305" y="124">3fs/2</text>
    <line x1="420" y1="100" x2="420" y2="110" stroke="currentColor"/><text x="405" y="124">2fs</text>
  </g>
  <text x="45" y="96" font-size="8" fill="currentColor">zone 1</text>
  <text x="145" y="96" font-size="8" fill="currentColor">zone 2</text>
  <text x="245" y="96" font-size="8" fill="currentColor">zone 3</text>
  <path d="M340 105 Q360 78 380 105 Z" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
  <text x="330" y="70" font-size="8" fill="currentColor">signal in zone 4</text>
  <path d="M40 105 Q60 78 80 105 Z" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-dasharray="2 2"/>
  <path d="M355 74 C 250 20, 150 20, 62 74" fill="none" stroke="currentColor" stroke-opacity="0.6" stroke-dasharray="3 3"/>
  <text x="150" y="30" font-size="8" fill="currentColor">folds down to baseband</text>
</svg>
<figcaption>Half-sample-rate boundaries carve the spectrum into Nyquist zones; a signal in a higher zone aliases into zone 1, appearing at baseband.</figcaption>
</figure>

## How it works

Sampling at rate *f*<sub>s</sub> divides the frequency axis into **Nyquist zones**, each
*f*<sub>s</sub>/2 wide. Any signal, wherever it truly lives, appears after sampling at a
position folded into the first zone (0 to *f*<sub>s</sub>/2). In ordinary baseband
sampling we keep the signal in zone 1 and treat aliasing as an enemy. Bandpass sampling
instead **places** a narrowband signal in a chosen higher zone and lets it alias down on
purpose, landing cleanly at baseband.

The conditions are strict:

- The sample rate must be at least twice the signal's **bandwidth** — the Nyquist rate for
  a bandpass signal.
- The signal must fit entirely inside one Nyquist zone; if its band straddles a
  *k*·*f*<sub>s</sub>/2 boundary, it folds back on itself and is destroyed.
- A sharp analog **bandpass filter** must precede the ADC so that only the wanted zone
  reaches it — otherwise noise and signals from every other zone alias down on top of the
  wanted one, and the [ADC](/reference/analog-to-digital-converter/) cannot tell them
  apart afterward.

Choosing *f*<sub>s</sub> so the band lands neatly in a zone (and knowing whether the zone
is odd or even, which spectrally inverts the result) is the design work of bandpass
sampling.

## In practice

The technique's appeal is that a signal at a high
[intermediate frequency](/reference/intermediate-frequency/) — or even at RF — can be
captured by an ADC clocked far slower than twice that frequency, saving cost and power.
The limits are practical: the converter's **analog input bandwidth** must actually reach
the signal frequency (many ADCs sample a much wider analog band than their clock rate),
and its aperture jitter must be small, because timing noise at a high input frequency
translates into far more amplitude error than at baseband.

## Relevance to SDR

Bandpass sampling underpins [direct-sampling](/reference/direct-sampling/) SDRs and
IF-sampling designs, where a fast ADC grabs a whole band and software picks out channels.
It is also the theory behind the RTL-SDR "direct sampling" HF hack: with the tuner
bypassed, HF signals are undersampled by the [RTL2832U](/reference/rtl2832u/) and appear
aliased into the first Nyquist zone. Understanding zones explains the spurious mirror
signals such setups produce when the input filtering is weak.

GopherTrunk works on the IQ stream after the ADC and does not choose the sampling scheme,
but the Nyquist-zone picture is the same one that governs
[aliasing](/reference/aliasing/) artefacts in any capture it decodes.

## Sources

[^wiki]: [Undersampling](https://en.wikipedia.org/wiki/Undersampling) — Wikipedia, on sampling below the carrier and folding a Nyquist zone to baseband via controlled aliasing.
