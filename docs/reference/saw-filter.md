---
slug: saw-filter
title: SAW filter
entry_type: hardware
category: rf-front-end
description: A SAW (surface acoustic wave) filter is a compact, sharp band-pass filter used in SDR front ends to pass one band (e.g. 1090 MHz ADS-B) and reject out-of-band signals.
keywords: SAW filter, surface acoustic wave, band-pass, front-end filter, 1090 MHz, ADS-B, preselector
aka: [SAW filter, "surface acoustic wave filter"]
autolink: true
see_also: [low-noise-amplifier, ads-b, bandwidth, attenuation]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Surface_acoustic_wave
---

A **SAW** (**surface acoustic wave**) filter is a compact, sharp **band-pass** filter
built on a piezoelectric substrate.[^wiki] In an SDR front end it acts as a preselector —
passing only the wanted band and strongly rejecting out-of-band signals that would
otherwise overload the receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sharp band-pass response curve passing a narrow band and rejecting everything outside it." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="95" x2="430" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M40 88 L200 88 C 215 88 215 30 235 30 L 245 30 C 265 30 265 88 280 88 L 430 88" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="24" text-anchor="middle" font-size="9" fill="currentColor">passband (e.g. 1090 MHz)</text>
  <text x="110" y="80" font-size="8" fill="currentColor">rejected</text><text x="360" y="80" font-size="8" fill="currentColor">rejected</text>
</svg>
<figcaption>A SAW filter passes one narrow band with steep skirts, protecting the receiver from strong out-of-band signals.</figcaption>
</figure>

## Overview

ADS-B receive chains commonly add a 1090 MHz SAW filter (often with a
[low-noise amplifier](/reference/low-noise-amplifier/)) so nearby cellular and broadcast
transmitters don't desensitise the [front end](/reference/superheterodyne-receiver/).

## Sources

[^wiki]: [Surface acoustic wave](https://en.wikipedia.org/wiki/Surface_acoustic_wave) — Wikipedia, on the surface-acoustic-wave devices used to build compact sharp band-pass filters.
