---
slug: superheterodyne-receiver
title: Superheterodyne receiver
entry_type: term
category: sdr-dsp
description: A superheterodyne receiver mixes a desired band down to a fixed intermediate frequency for easier filtering and detection — the basis of most SDR front-ends.
keywords: superheterodyne, superhet, mixer, local oscillator, intermediate frequency, IF, downconversion
aka: [superheterodyne receiver, superhet]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture }
  - { label: Key parts, value: Mixer + local oscillator }
  - { label: Produces, value: Intermediate frequency / baseband }
see_also: [local-oscillator, digital-down-converter, analog-to-digital-converter, software-defined-radio]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Superheterodyne receiver (Wikipedia)", url: https://en.wikipedia.org/wiki/Superheterodyne_receiver }
---

A **superheterodyne receiver** uses a mixer driven by a
[local oscillator](/reference/local-oscillator/) to shift a chosen band **down** to a
fixed intermediate frequency (IF) — or to baseband — where it is easier to filter and
detect.

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

Tuning is just changing the local-oscillator frequency so the wanted band lands at the
fixed IF. SDR quadrature front-ends apply the same idea, mixing to baseband as
[IQ](/reference/iq-data/) before the [ADC](/reference/analog-to-digital-converter/).

## Relevance to SDR

Understanding the mixer/LO explains why "tuning" an SDR is simply setting a number, and
how a [digital down-converter](/reference/digital-down-converter/) does it in software.
