---
slug: vector-network-analyzer
title: Vector network analyzer (VNA)
entry_type: hardware
category: test-equipment
description: "A vector network analyzer measures the S-parameters of an RF device — the magnitude and phase of its reflection and transmission — after a reference-plane calibration."
keywords: vector network analyzer, VNA, S-parameters, S11, S21, reflection, transmission, SOLT calibration, return loss, insertion loss, RF test equipment, impedance
aka: [VNA, network analyzer, network analyser]
autolink: true
infobox:
  - { label: Type, value: RF measurement instrument }
  - { label: Measures, value: "S-parameters (mag + phase)" }
  - { label: Ports, value: "1, 2, or more" }
  - { label: Key step, value: "SOLT / calibration" }
  - { label: TX, value: "Yes (internal test source)" }
  - { label: Typical price, value: "$50 – $100,000+" }
see_also: [s-parameters, return-loss, nanovna, standing-wave-ratio, reflection-coefficient, impedance]
cite_urls:
  - https://en.wikipedia.org/wiki/Network_analyzer_(electrical)
  - https://www.keysight.com/us/en/assets/7018-06841/application-notes/5965-7917.pdf
---

**A vector network analyzer (VNA)** measures how an RF device or network responds to
signals across frequency, reporting both the **magnitude and phase** of the reflected and
transmitted waves as [S-parameters](/reference/s-parameters/).[^wiki] Because it captures
phase, not just amplitude, a VNA fully characterizes an antenna, filter, cable, amplifier,
or matching network — its input match, insertion loss, delay, and complex
[impedance](/reference/impedance/) — where a scalar instrument would see only power.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A two-port VNA diagram: an internal source drives port one into a device under test, directional couplers sample the incident, reflected, and transmitted waves, and a receiver computes S11 and S21." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="vnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="60" width="70" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="55" y="81" font-size="9" fill="currentColor" text-anchor="middle">Source</text>
  <rect x="180" y="55" width="100" height="44" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
  <text x="230" y="74" font-size="9" fill="currentColor" text-anchor="middle">Device</text>
  <text x="230" y="88" font-size="9" fill="currentColor" text-anchor="middle">under test</text>
  <text x="135" y="52" font-size="8" fill="currentColor" text-anchor="middle">Port 1</text>
  <text x="325" y="52" font-size="8" fill="currentColor" text-anchor="middle">Port 2</text>
  <line x1="90" y1="77" x2="180" y2="77" stroke="currentColor" marker-end="url(#vnar)"/>
  <line x1="280" y1="77" x2="360" y2="77" stroke="currentColor" marker-end="url(#vnar)"/>
  <path d="M150 77 q0 -18 -18 -18" fill="none" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="118" y="55" font-size="8" fill="currentColor" text-anchor="end">S11</text>
  <rect x="360" y="60" width="80" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="400" y="76" font-size="8" fill="currentColor" text-anchor="middle">Receiver</text>
  <text x="400" y="87" font-size="8" fill="currentColor" text-anchor="middle">S21</text>
</svg>
<figcaption>A VNA drives a known signal into the device, samples the incident, reflected, and transmitted waves, and computes S-parameters after calibration removes the fixture's own errors.</figcaption>
</figure>

## Overview

Internally a VNA contains a swept [signal source](/reference/signal-generator/),
[directional couplers](/reference/directional-coupler/) that separate the incident from
the reflected wave, and phase-coherent receivers on each port. At every frequency it forms
ratios: **S11** (input reflection, from which [return loss](/reference/return-loss/),
[SWR](/reference/standing-wave-ratio/), and the complex
[reflection coefficient](/reference/reflection-coefficient/) follow) and, on a two-port
instrument, **S21** (forward transmission — insertion loss or gain and phase/delay). A
one-port VNA measures only reflection; two ports add transmission; larger instruments add
more ports for multi-port and differential devices.

## Calibration

A VNA's accuracy comes almost entirely from **calibration**, which mathematically moves
the measurement reference plane to the ends of the test cables and removes the fixture's
own losses, mismatches, and delays. The classic two-port procedure is **SOLT** — measure
known **Short**, **Open**, **Load**, and **Through** standards — which solves for a
12-term error model that is then applied to every subsequent measurement. Variants such as
TRL and SOLR trade different standards for accuracy at higher frequencies or on
non-coaxial fixtures. **Without calibration a VNA reading is meaningless**: the raw
response is dominated by cable loss and connector reflections, not the device under test.

## In practice

- **Antenna tuning** — sweep S11 to find resonance, minimize
  [SWR](/reference/standing-wave-ratio/), and check the match across a band.
- **Filter and cable characterization** — S21 gives passband loss, stopband rejection, and
  group delay; S11 confirms the input match.
- **Impedance measurement** — the complex reflection coefficient maps directly onto a
  Smith chart, giving the device's [impedance](/reference/impedance/) at each frequency.
- **Amplifier gain and match** — a two-port sweep captures gain, input/output return loss,
  and stability inputs.

## Relevance to SDR

For SDR and scanner work a VNA is the instrument you reach for when building or tuning the
front-end plumbing: verifying that an [antenna](/reference/antenna/) is resonant on the
target band, that a [bandpass or notch filter](/reference/rf-filter/) sits where you need
it, and that [coax](/reference/coaxial-cable/) and connectors are low-loss and
well-matched. Affordable pocket instruments such as the
[NanoVNA](/reference/nanovna/) have put S-parameter measurement within reach of every
hobbyist. GopherTrunk is a receive-only decoder and does not interface with a VNA; the VNA
is a bench aid for getting the antenna and filtering right before the signal ever reaches
the SDR.

## Sources

[^wiki]: [Network analyzer (electrical)](https://en.wikipedia.org/wiki/Network_analyzer_(electrical)) — Wikipedia, on vector network analyzer operation, S-parameters, and calibration.
