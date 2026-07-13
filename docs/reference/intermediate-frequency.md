---
slug: intermediate-frequency
title: Intermediate frequency (IF)
entry_type: term
category: sdr-dsp
description: The intermediate frequency (IF) is the fixed lower frequency a superheterodyne receiver mixes the incoming signal down to, where filtering and digitising are easier.
keywords: intermediate frequency, IF, superheterodyne, mixing, low-IF, zero-IF, downconversion, image frequency, IF filter
aka: [IF, "intermediate frequency"]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture stage }
  - { label: Role, value: Fixed frequency for filtering & digitising }
  - { label: Variants, value: High-IF, low-IF, zero-IF }
see_also: [superheterodyne-receiver, local-oscillator, baseband, analog-to-digital-converter, image-frequency, direct-conversion-receiver, low-if]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Intermediate_frequency
  - https://en.wikipedia.org/wiki/Superheterodyne_receiver
---

The **intermediate frequency** (**IF**) is the fixed, lower frequency that a
[superheterodyne receiver](/reference/superheterodyne-receiver/) shifts the wanted
signal down to before filtering and digitising.[^wiki] Mixing the variable input frequency to a
**constant IF** lets the receiver use one well-designed filter regardless of the tuned
channel, which is the central insight that made selective, tunable radios practical.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A high radio frequency mixed with a local oscillator to produce a fixed lower intermediate frequency." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="430" y2="80" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="360" y1="80" x2="360" y2="40" stroke="currentColor" stroke-width="2"/><text x="360" y="32" text-anchor="middle" font-size="8" fill="currentColor">RF</text>
  <line x1="110" y1="80" x2="110" y2="50" stroke="currentColor" stroke-width="2"/><text x="110" y="42" text-anchor="middle" font-size="8" fill="currentColor">IF (fixed)</text>
  <path d="M360 78 q-40 -30 -120 -20 q-80 10 -130 20" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#ifar)"/>
  <text x="240" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">mixer + local oscillator shift RF → IF</text>
  <defs><marker id="ifar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Mixing the tuned signal to a constant intermediate frequency lets one filter handle any channel.</figcaption>
</figure>

## How it works

A [mixer](/reference/mixer-rf/) multiplies the incoming RF by a
[local oscillator](/reference/local-oscillator/) (LO), producing sum and difference frequencies.
The receiver keeps the difference, |f_RF − f_LO|, and tunes the LO so this difference always
lands on the same IF. To change channels you retune the LO, not the IF filtering. That fixed IF
is where the receiver puts its sharpest, most selective filter and most of its gain — a filter
that would be extremely hard to build as a tunable stage but is straightforward at one fixed
frequency.

The catch is the **[image frequency](/reference/image-frequency/)**: because the mixer responds
to |f_RF − f_LO|, a second input on the far side of the LO, offset by twice the IF, also falls on
the IF and interferes. A higher IF pushes this image further from the wanted signal so a
front-end filter can reject it more easily — the classic tension in IF selection is image
rejection (favouring a high IF) versus cheaper, sharper filtering (favouring a low IF).

## Variants

- **High-IF (single or multiple conversion).** Traditional communications receivers use one or
  more high IFs (e.g. 10.7 MHz, 45 MHz, 21.4 MHz) for strong image rejection and selectivity.
- **[Low-IF](/reference/low-if/).** The signal is placed at a small non-zero IF, close enough to
  baseband to digitise directly yet offset enough to avoid the DC artefacts of zero-IF.
- **Zero-IF (direct conversion).** The IF is 0 Hz — the signal is mixed straight to
  [baseband](/reference/baseband/). Simple and cheap, but it introduces a DC-offset spike and
  [IQ imbalance](/reference/iq-imbalance/) to manage; see the
  [direct-conversion receiver](/reference/direct-conversion-receiver/).

## In practice

Many SDR front ends use a low-IF or zero-IF architecture because it feeds the
[ADC](/reference/analog-to-digital-converter/) conveniently and pushes the remaining selectivity
into software filters. Others, such as some SDRPlay and Airspy designs, use a genuine IF stage
before digitising. Once samples reach the software domain, GopherTrunk performs a *digital*
down-conversion — its own numerically controlled oscillator and filters act as an ideal,
drift-free IF stage, translating each channel to baseband without the image and filter-tolerance
compromises of analog IF hardware.

## Sources

[^wiki]: [Intermediate frequency](https://en.wikipedia.org/wiki/Intermediate_frequency) — Wikipedia, on superheterodyne downconversion to a fixed IF.
[^het]: [Superheterodyne receiver](https://en.wikipedia.org/wiki/Superheterodyne_receiver) — Wikipedia, on the mixing architecture and the image-frequency trade-off that IF choice governs.
