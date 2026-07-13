---
slug: ocxo
title: OCXO (oven-controlled crystal oscillator)
entry_type: hardware
category: rf-front-end
description: "An OCXO holds its quartz crystal in a temperature-stabilized oven, reaching sub-ppb stability — the high-accuracy frequency reference above a TCXO but below a GPSDO."
keywords: OCXO, oven-controlled crystal oscillator, ppb, frequency stability, frequency reference, crystal oven, holdover, precision oscillator, 10 MHz reference
aka: [OCXO, "oven-controlled crystal oscillator"]
autolink: true
infobox:
  - { label: Type, value: "Oven-stabilized crystal oscillator" }
  - { label: Stability, value: "±1 to ±100 ppb typical" }
  - { label: Warm-up, value: "Minutes to reach spec" }
  - { label: Power, value: "High (oven heater)" }
  - { label: TX, value: "N/A (reference)" }
see_also: [tcxo, gpsdo, frequency-stability, ppm-frequency-correction, local-oscillator]
cite_urls:
  - https://en.wikipedia.org/wiki/Crystal_oven
---

An **OCXO** (oven-controlled crystal oscillator) holds its quartz crystal inside a small,
electrically heated **oven** kept at a constant temperature, so the crystal never sees the
swings that cause drift.[^wiki] By eliminating temperature as a variable rather than merely
compensating for it, an OCXO reaches **parts-per-billion (ppb)** stability — one to three
orders of magnitude better than a [TCXO](/reference/tcxo/). It sits in the middle of the
reference hierarchy: more accurate than a TCXO, less absolute than a GPS-disciplined
[GPSDO](/reference/gpsdo/), and it is the classic choice for a bench "10 MHz reference."

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A crystal enclosed in an insulated oven with a heater and thermostat holding constant temperature, feeding an oscillator that outputs a stable reference." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="35" width="200" height="95" rx="6" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.4"/>
  <rect x="70" y="60" width="60" height="45" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="100" y="86" font-size="9" fill="currentColor" text-anchor="middle">crystal</text>
  <g stroke="currentColor" stroke-width="1.4" fill="none"><path d="M150 55 h40 M150 63 h40 M150 71 h40 M150 79 h40"/></g>
  <text x="200" y="100" font-size="8" fill="currentColor" text-anchor="middle">heater</text>
  <circle cx="55" cy="48" r="5" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="55" y="30" font-size="8" fill="currentColor" text-anchor="middle">thermostat</text>
  <defs><marker id="ocxoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="240" y1="82" x2="300" y2="82" stroke="currentColor" stroke-width="1.6" marker-end="url(#ocxoar)"/>
  <rect x="300" y="62" width="90" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="345" y="86" font-size="9" fill="currentColor" text-anchor="middle">stable 10 MHz</text>
  <text x="140" y="145" font-size="9" fill="currentColor" text-anchor="middle">insulated oven at constant T</text>
</svg>
<figcaption>An OCXO keeps the crystal in a thermostat-controlled oven, removing temperature drift at the source and reaching ppb stability.</figcaption>
</figure>

## Overview

A [TCXO](/reference/tcxo/) accepts the crystal's temperature drift and cancels it
electronically; an OCXO instead prevents the drift by holding the crystal at a fixed
"turnover" temperature — typically a few degrees above the highest ambient it will meet — so
the crystal always operates at the flat top of its frequency-versus-temperature curve. This
brute-force approach yields dramatically better [frequency
stability](/reference/frequency-stability/), at the cost of an oven that draws significant
power and needs several minutes to warm up before it meets spec.

## How it works

- The crystal and its oscillator circuit sit inside an insulated enclosure with a resistive
  heater and a thermostat.
- The control loop drives the heater to keep the enclosure at a constant setpoint regardless
  of the outside temperature, so the crystal's own temperature — and thus its frequency —
  barely moves.
- Operated at the turnover point, the crystal's sensitivity to residual temperature error is
  near zero, giving ±1 to ±100 ppb typical stability versus the TCXO's ±0.5–2 ppm.

## In practice

The trade-offs are warm-up and power. An OCXO must reach oven temperature before it settles,
so it is left powered continuously in a rack; a cold start shows a large initial drift that
tapers over minutes. Its **holdover** — the ability to keep good frequency with no external
correction — is excellent, which is why an OCXO is the flywheel inside most
[GPSDOs](/reference/gpsdo/): GPS disciplines the OCXO's slow aging while the OCXO rides
through short GPS outages. Aging (a slow, roughly logarithmic frequency creep over months)
remains and is why long-term references are still trimmed or GPS-locked.

## Relevance to SDR

For an SDR, an OCXO is the reference you reach for when a [TCXO](/reference/tcxo/) is not
stable enough — narrowband work, phase-coherent multi-receiver setups, or a fixed monitoring
station that must stay dead-on frequency for hours. Many higher-end SDRs accept an **external
10 MHz** input so a shared OCXO can clock several radios at once. GopherTrunk decodes the
samples the front end produces and never drives the oscillator, but a more stable reference
means the control channel it is tracking stays put, reducing residual drift the software's
automatic frequency correction has to chase. When absolute, long-term accuracy matters most,
a GPS-disciplined [GPSDO](/reference/gpsdo/) — an OCXO steered by GPS — is the next step up.

## Sources

[^wiki]: [Crystal oven](https://en.wikipedia.org/wiki/Crystal_oven) — Wikipedia, on oven-controlled oscillators, turnover temperature, ppb stability, warm-up, and holdover.
