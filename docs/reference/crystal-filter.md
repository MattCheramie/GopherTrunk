---
slug: crystal-filter
title: Crystal filter
entry_type: hardware
category: rf-front-end
description: "A crystal filter uses quartz resonators to build very narrow, high-Q band-pass filters, most often as the IF filter that sets a superheterodyne receiver's selectivity."
keywords: crystal filter, quartz crystal filter, IF filter, ladder filter, narrow band-pass, high-Q, selectivity, SSB filter, CW filter, monolithic crystal filter
aka: [crystal filter, "quartz crystal filter", "XTAL filter"]
autolink: true
infobox:
  - { label: Type, value: "Quartz-resonator band-pass filter" }
  - { label: Resonator, value: "Piezoelectric quartz crystal" }
  - { label: Key spec, value: "Very high Q, very narrow bandwidth" }
  - { label: TX, value: "No (low-level IF stage)" }
  - { label: Typical price, value: "$5–$100" }
see_also: [intermediate-frequency, saw-filter, rf-filter, superheterodyne-receiver, q-factor]
cite_urls:
  - https://en.wikipedia.org/wiki/Crystal_filter
  - https://en.wikipedia.org/wiki/Crystal_oscillator
---

A **crystal filter** is a very narrow, high-[Q](/reference/q-factor/) band-pass
filter built from one or more piezoelectric quartz resonators.[^wiki] A quartz
crystal behaves like an extraordinarily sharp series-resonant circuit with an
effective Q in the tens of thousands — orders of magnitude beyond any LC network —
so crystals can define passbands only a few hundred hertz to a few kilohertz wide.
Their classic home is the [intermediate-frequency](/reference/intermediate-frequency/)
stage of a [superheterodyne receiver](/reference/superheterodyne-receiver/), where
the crystal filter sets the receiver's adjacent-channel selectivity.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A ladder of quartz crystals between two ground capacitors, producing a very narrow band-pass response at the intermediate frequency." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="55" x2="60" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <rect x="60" y="47" width="26" height="16" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="86" y1="55" x2="120" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <rect x="120" y="47" width="26" height="16" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="146" y1="55" x2="176" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <rect x="176" y="47" width="26" height="16" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="202" y1="55" x2="220" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <line x1="103" y1="55" x2="103" y2="80" stroke="currentColor" stroke-width="1.4"/>
  <line x1="159" y1="55" x2="159" y2="80" stroke="currentColor" stroke-width="1.4"/>
  <text x="125" y="30" text-anchor="middle" font-size="9" fill="currentColor">quartz ladder</text>
  <line x1="270" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M270 118 L340 118 C352 118 352 40 358 40 L362 40 C368 40 368 118 380 118 L440 118" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="33" text-anchor="middle" font-size="8" fill="currentColor">narrow IF passband</text>
</svg>
<figcaption>A crystal ladder filter and the very narrow IF response its high-Q resonators produce.</figcaption>
</figure>

## Overview

Quartz is piezoelectric: an applied voltage strains the crystal and mechanical
vibration generates a voltage, so a thin quartz plate behaves as an electrical
resonator whose frequency is set by its physical dimensions. That mechanical
resonance is extremely lossless, which is where the enormous Q comes from. A single
crystal has both a series-resonant frequency and, a little higher, a parallel
(anti-resonant) frequency; combining several crystals with coupling capacitors
places their responses side by side to synthesise a flat-topped passband with
steep skirts.

## Variants

- **Ladder filter** — several crystals of the *same* nominal frequency in series
  with shunt capacitors to ground; simple and popular in amateur SSB/CW rigs.
- **Lattice filter** — crystals arranged in a bridge, using two frequencies for a
  symmetric response.
- **Monolithic crystal filter (MCF)** — two or more resonators fabricated on one
  quartz plate with shared electrodes, giving a compact two-pole section.
- **Roofing filter** — a moderately wide crystal filter placed early in the IF
  chain to keep strong nearby signals out of later stages and preserve
  [dynamic range](/reference/dynamic-range/).

Bandwidth is chosen for the mode: ~250–500 Hz for CW, ~2.4 kHz for SSB, and wider
for AM or data. [SAW](/reference/saw-filter/) filters serve the same "sharp
band-pass" role but at much higher frequencies and wider bandwidths, so the two
technologies are complementary rather than competing.

## Relevance to SDR

Crystal filters define selectivity in the analog IF of virtually every classic
superheterodyne communications receiver, from HF transceivers to land-mobile
radios and the front ends of many scanners. In a software-defined receiver, much of
that job moves into DSP: a [digital down-converter](/reference/digital-down-converter/)
and [FIR filter](/reference/fir-filter/) can synthesise an arbitrarily sharp,
reconfigurable channel filter that no fixed crystal could match. That is precisely
what GopherTrunk does — its channelisation and matched filtering are numerical, so
it needs no physical crystal filter. Even so, SDRs with an analog IF (such as
tuner-plus-IF architectures) still rely on a crystal or SAW roofing filter to
protect the ADC from strong out-of-band signals, and the concept of a narrow,
high-Q IF filter remains the reference point for what the digital filters in
GopherTrunk emulate in code.

## Sources

[^wiki]: [Crystal filter](https://en.wikipedia.org/wiki/Crystal_filter) — Wikipedia, on quartz-resonator band-pass filters and their use in IF selectivity.
