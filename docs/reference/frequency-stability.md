---
slug: frequency-stability
title: Frequency Stability
entry_type: term
category: rf-fundamentals
description: Frequency stability is how closely an oscillator holds its nominal frequency over temperature, time, and aging — quantified in ppm and, for short intervals, Allan deviation.
keywords: frequency stability, ppm, parts per million, aging, temperature stability, Allan deviation, oscillator drift, frequency accuracy, TCXO, OCXO, GPSDO
aka: [frequency stability, oscillator stability, frequency accuracy]
autolink: true
infobox:
  - { label: Type, value: Long-term frequency constancy }
  - { label: Unit, value: "ppm (parts per million); Allan deviation" }
  - { label: Degraded by, value: Temperature, aging, supply, load }
see_also: [tcxo, ocxo, gpsdo, ppm-frequency-correction, local-oscillator, phase-noise]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_drift
  - https://en.wikipedia.org/wiki/Allan_variance
---

**Frequency stability** is how well an oscillator keeps its output at the intended
frequency as temperature, supply voltage, load, and time change.[^wiki] It is usually
specified in **parts per million (ppm)** — a ±2 ppm source at 450 MHz may wander up to
±900 Hz — and, for the shortest timescales, as **Allan deviation**. Stability underpins
everything a [local oscillator](/reference/local-oscillator/) is used for, because a
receiver that drifts off frequency loses lock on the signals it is trying to decode.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A plot of frequency error versus temperature: a nearly flat line labelled stable oscillator staying inside a narrow ppm tolerance band, and a steeply sloping curve labelled drifting oscillator that leaves the band at temperature extremes." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" font-size="10">
    <line x1="45" y1="20" x2="45" y2="140" marker-end="url(#fsar)"/>
    <line x1="45" y1="80" x2="440" y2="80" marker-end="url(#fsar)"/>
    <rect x="45" y="65" width="390" height="30" fill="currentColor" fill-opacity="0.12" stroke="none"/>
    <line x1="55" y1="78" x2="430" y2="82" stroke-width="1.7"/>
    <path d="M55 130 Q160 120 240 80 Q330 35 430 26" stroke-width="1.7" stroke-dasharray="5 3"/>
  </g>
  <g fill="currentColor" font-size="10" stroke="none">
    <text x="6" y="30">Δf</text>
    <text x="405" y="74">temp</text>
    <text x="300" y="60">ppm tolerance</text>
    <text x="60" y="98">stable</text>
    <text x="330" y="24">drifting</text>
  </g>
</svg>
<figcaption>A stable oscillator holds its frequency inside a narrow ppm band across temperature; a poorly compensated one drifts out of tolerance at the extremes.</figcaption>
</figure>

## How it works

The dominant error in a quartz oscillator is **temperature**: the crystal's resonant
frequency changes as it warms and cools, tracing a characteristic curve. **Aging** adds a
slow, roughly logarithmic drift over months and years as the crystal and its mount settle.
Supply-voltage and load changes contribute smaller pulling effects. A datasheet folds
these into an overall ppm figure over a stated temperature and time window.

Technologies are layered to fight each error source:

- A plain crystal oscillator drifts tens of ppm over temperature.
- A **[TCXO](/reference/tcxo/)** (temperature-compensated) applies a correction versus
  temperature, reaching roughly ±0.1–2 ppm.
- An **[OCXO](/reference/ocxo/)** (oven-controlled) holds the crystal at a constant
  elevated temperature, reaching parts per billion.
- A **[GPSDO](/reference/gpsdo/)** disciplines a local oscillator to the atomic-clock
  reference in GPS, giving near-perfect long-term accuracy.

Stability is a *long-term* property and is distinct from
[phase noise](/reference/phase-noise/), the *short-term* random jitter around the
carrier. The two are measured differently — ppm and Allan deviation for stability, dBc/Hz
for phase noise — and a source can excel at one while being mediocre at the other.

## In practice

Where an oscillator is stable but simply offset, the fix is calibration: measure the
error against a known reference and apply a
[ppm frequency correction](/reference/ppm-frequency-correction/) so software retunes to
compensate. **Allan deviation** is the tool for characterising stability over a chosen
averaging interval, revealing where a source is best (often around one second) and where
drift or noise takes over. Systems that must interoperate over the air — cellular base
stations, trunked simulcast sites — carry disciplined references precisely so every
transmitter agrees on frequency and timing.

## Relevance to SDR

Frequency stability is a practical daily concern in SDR reception. Inexpensive RTL-SDR
dongles ship with basic crystals specified around ±20–30 ppm and drift as they warm up,
which at UHF is enough to walk a narrowband signal out of the decoder's capture range.
The standard workflow is to measure the offset against a known transmitter and enter a
[ppm correction](/reference/ppm-frequency-correction/); TCXO-equipped dongles reduce the
drift at the source. For [trunked systems](/reference/trunked-radio/) the tolerance is
tight because the decoder must stay locked to a narrow
[control channel](/reference/control-channel/) for long periods.

**GopherTrunk** relies on the front-end's stability but also compensates for it in
software: it applies a configurable frequency correction and its carrier-tracking loops
pull in residual offset, so a modest, slowly drifting SDR can still decode reliably. What
software cannot cure is a source so unstable that it drifts faster than the loops can
follow — which is why a TCXO or better matters for demanding wideband, multi-channel
monitoring.

## Sources

[^wiki]: [Frequency drift](https://en.wikipedia.org/wiki/Frequency_drift) — Wikipedia, causes of oscillator frequency change over temperature, time, and aging.
[^allan]: [Allan variance](https://en.wikipedia.org/wiki/Allan_variance) — Wikipedia, the standard measure of oscillator stability over an averaging interval.
