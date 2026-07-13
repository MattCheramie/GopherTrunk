---
slug: noise-source
title: Noise source
entry_type: hardware
category: test-equipment
description: "A noise source is a calibrated broadband noise generator, rated by its excess noise ratio (ENR), used to measure the noise figure of receivers and amplifiers."
keywords: noise source, excess noise ratio, ENR, noise figure measurement, Y-factor method, avalanche diode noise, hot cold noise, calibrated noise generator, RF test equipment
aka: [noise source, noise diode, ENR source]
autolink: true
infobox:
  - { label: Type, value: Calibrated RF noise generator }
  - { label: Rated by, value: "Excess noise ratio (ENR)" }
  - { label: Element, value: "Avalanche / Zener noise diode" }
  - { label: Used for, value: "Noise-figure (Y-factor) measurement" }
  - { label: TX, value: "Yes (broadband noise output)" }
  - { label: Typical price, value: "$50 – $5,000+" }
see_also: [noise-figure, thermal-noise, noise-temperature, signal-generator, low-noise-amplifier, spectrum-analyzer]
cite_urls:
  - https://en.wikipedia.org/wiki/Noise_generator
  - https://en.wikipedia.org/wiki/Noise_figure
---

**A noise source** is a calibrated broadband noise generator used to measure the
[noise figure](/reference/noise-figure/) of receivers, amplifiers, and other RF
components.[^wiki] Instead of a clean carrier like a
[signal generator](/reference/signal-generator/), it emits flat, noise-like power across a
wide band, characterized by its **excess noise ratio (ENR)** — the amount by which its
"on" noise exceeds the reference [thermal noise](/reference/thermal-noise/) of a resistor
at room temperature.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A noise-figure measurement: a noise source switched between a cold off state and a hot on state feeds the device under test, and the receiver's output power ratio between the two states gives the Y-factor." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="52" width="100" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="70" y="70" font-size="8" fill="currentColor" text-anchor="middle">Noise source</text>
  <text x="70" y="83" font-size="7" fill="currentColor" text-anchor="middle">OFF (cold) /</text>
  <text x="70" y="92" font-size="7" fill="currentColor" text-anchor="middle">ON (hot), ENR</text>
  <rect x="180" y="55" width="90" height="38" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
  <text x="225" y="70" font-size="8" fill="currentColor" text-anchor="middle">Device</text>
  <text x="225" y="82" font-size="8" fill="currentColor" text-anchor="middle">under test</text>
  <rect x="330" y="55" width="100" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="380" y="70" font-size="8" fill="currentColor" text-anchor="middle">Receiver /</text>
  <text x="380" y="82" font-size="8" fill="currentColor" text-anchor="middle">power meter</text>
  <line x1="120" y1="74" x2="180" y2="74" stroke="currentColor" marker-end="url(#nsar)"/>
  <line x1="270" y1="74" x2="330" y2="74" stroke="currentColor" marker-end="url(#nsar)"/>
  <text x="225" y="120" font-size="8" fill="currentColor" text-anchor="middle">Y = P(hot) / P(cold)</text>
</svg>
<figcaption>Noise-figure measurement by the Y-factor method: switch the source between its calibrated hot and cold states and read the ratio of output powers; ENR and Y together give the device's noise figure.</figcaption>
</figure>

## How it works

At the heart of a noise source is a reverse-biased **avalanche (Zener) noise diode** that,
when energized (typically by a 28 V supply), produces broadband shot noise well above the
thermal floor. Switch the bias off and the source presents a matched resistor at ambient
temperature — the "cold" state at roughly 290 K. The two states are the "hot" and "cold"
references the measurement needs. The source's **ENR**, supplied as a calibration table
versus frequency, states in dB how far its hot-state
[noise temperature](/reference/noise-temperature/) exceeds the 290 K reference.

## The Y-factor method

Noise figure is most commonly measured by the **Y-factor** technique:

- Connect the noise source to the device under test and its output to a receiver, noise-
  figure meter, or [spectrum analyzer](/reference/spectrum-analyzer/).
- Record the output power with the source **on** (hot) and **off** (cold).
- The ratio **Y = P_hot / P_cold** combines with the calibrated ENR to solve for the
  device's added noise. A low-noise device barely changes Y as the input toggles; a noisy
  one swamps the difference — which is exactly what noise figure quantifies.

A two-stage "measure the analyzer alone, then the analyzer plus DUT" calibration removes
the meter's own noise contribution, so the result reflects the device under test rather
than the instrument.

## In practice

- **ENR grade matters.** Low-ENR sources (~5–6 dB) suit sensitive, low-noise-figure
  devices; high-ENR sources (~15 dB) suit lossy or high-NF paths.
- **Match and cabling.** Poor [impedance](/reference/impedance/) match between states adds
  error; keep connections short and well-characterized.
- **Frequency limits.** ENR is calibrated per frequency; use the table, not a single
  number, across a wide sweep.

## Relevance to SDR

Noise figure sets a receiver's ultimate sensitivity, and a noise source is the standard
way to measure it — for instance to confirm that adding a
[low-noise amplifier](/reference/low-noise-amplifier/) actually improves the system NF
rather than the [attenuation](/reference/attenuation/) of a long feedline dominating it.
For the SDR scanner enthusiast this is bench-lab territory: quantifying an LNA or
front-end so weak trunking signals clear the [noise floor](/reference/noise-floor/).
GopherTrunk is a receive-only decoder that neither generates nor measures noise figure;
a noise source is a specialized RF-lab aid used upstream to characterize the hardware that
feeds the SDR, not part of the decode chain.

## Sources

[^wiki]: [Noise generator](https://en.wikipedia.org/wiki/Noise_generator) — Wikipedia, on calibrated noise sources, ENR, and their use in noise-figure measurement.
