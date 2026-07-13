---
slug: superheterodyne-receiver
title: Superheterodyne receiver
entry_type: term
category: sdr-dsp
description: A superheterodyne receiver mixes a desired band down to a fixed intermediate frequency for easier filtering and detection — the basis of most SDR front-ends.
keywords: superheterodyne, superhet, mixer, local oscillator, intermediate frequency, IF, image frequency, downconversion
aka: [superheterodyne receiver, superhet]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture }
  - { label: Key parts, value: Mixer + local oscillator }
  - { label: Produces, value: Intermediate frequency / baseband }
see_also: [local-oscillator, digital-down-converter, intermediate-frequency, image-frequency, image-rejection, direct-conversion-receiver, analog-to-digital-converter, software-defined-radio]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Superheterodyne_receiver
  - https://en.wikipedia.org/wiki/Edwin_Howard_Armstrong
---

A **superheterodyne receiver** uses a mixer driven by a
[local oscillator](/reference/local-oscillator/) to shift a chosen band **down** to a fixed
[intermediate frequency](/reference/intermediate-frequency/) (IF) — or to baseband — where
it is easier to filter and detect.[^wiki] Invented by
[Edwin Armstrong](/reference/edwin-armstrong/), it has been the dominant receiver
architecture for a century and underlies most SDR front-ends.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 110" role="img" aria-label="Receiver chain: antenna, RF amplifier, mixer fed by a local oscillator, IF filter, then detector." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M28 70 v-24 m-8 0 l8 -11 l8 11" fill="none" stroke="currentColor" stroke-width="1.8"/>
    <rect x="60" y="44" width="56" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="88" y="63">RF amp</text>
    <rect x="140" y="44" width="56" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="168" y="63">mixer</text>
    <circle cx="168" cy="96" r="3" fill="currentColor"/><line x1="168" y1="74" x2="168" y2="93" stroke="currentColor" stroke-width="1.2"/><text x="168" y="108" font-size="8">LO</text>
    <rect x="220" y="44" width="64" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="252" y="60">IF filter</text>
    <rect x="308" y="44" width="64" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="340" y="63">detector</text>
    <rect x="396" y="44" width="64" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="428" y="63">output</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="36" y1="59" x2="59" y2="59"/><line x1="116" y1="59" x2="139" y2="59"/><line x1="196" y1="59" x2="219" y2="59"/><line x1="284" y1="59" x2="307" y2="59"/><line x1="372" y1="59" x2="395" y2="59"/></g>
  </g>
</svg>
<figcaption>A superheterodyne receiver mixes the wanted signal down to a fixed intermediate frequency for easier filtering.</figcaption>
</figure>

## How it works

A mixer multiplies the incoming RF by the local-oscillator tone, producing sum and
difference frequencies. The difference — the IF — is a fixed frequency chosen by the
designer, and **tuning is just changing the LO frequency so the wanted band lands on that
fixed IF**. The genius of the scheme is that the hard part of the receiver — a sharp,
high-gain, well-shaped filter and the detector — can be built once, at one frequency,
instead of being made tunable across the whole spectrum. Fixed-frequency crystal, ceramic,
or SAW filters at the IF give selectivity that a tunable RF filter never could.

The unavoidable cost is the **[image](/reference/image-frequency/)**: two input
frequencies, one above and one below the LO by the IF amount, both mix down to the same IF.
A signal at the image frequency slips through as though it were the wanted station.
Suppressing it needs an RF pre-selector filter or an image-reject mixer topology, so
[image rejection](/reference/image-rejection/) is a headline spec of any superhet.

## Variants

- **Single-conversion** — one mixer/LO/IF stage; simple, adequate for many bands.
- **Double- or triple-conversion** — successive IFs (e.g. a high first IF to push the image
  far away, then a low second IF for tight filtering) used in high-performance communications
  receivers.
- **Quadrature down-conversion** — the SDR form: mix to two IF or baseband channels 90°
  apart to produce [IQ](/reference/iq-data/), which resolves the image mathematically rather
  than by RF filtering. A [zero-IF](/reference/zero-if/)
  [direct-conversion](/reference/direct-conversion-receiver/) receiver is the degenerate
  case where the IF is 0 Hz.

## In practice

Most consumer SDRs are quadrature receivers under the hood: an R820T-based RTL-SDR mixes to
a low IF, while Airspy and many others mix to zero-IF baseband. Either way the classic
superhet blocks — LNA, mixer, LO, IF filter — are present, just followed by an
[ADC](/reference/analog-to-digital-converter/) instead of an analog detector. Everything
after the ADC, including a further software mix in a
[digital down-converter](/reference/digital-down-converter/), is a digital continuation of
the same idea.

## Relevance to SDR

Understanding the mixer/LO explains why "tuning" an SDR is simply setting a number, and why
image rejection and pre-selection still matter even when the detector is software. In
GopherTrunk the RF tuning is done in the radio's superhet front-end; the software then
applies a second, digital down-conversion to pick out individual channels from the digitised
band.

## Sources

[^wiki]: [Superheterodyne receiver](https://en.wikipedia.org/wiki/Superheterodyne_receiver) — Wikipedia, on mixing a band down to a fixed intermediate frequency.
[^armstrong]: [Edwin Howard Armstrong](https://en.wikipedia.org/wiki/Edwin_Howard_Armstrong) — Wikipedia, on the inventor of the superheterodyne principle.
