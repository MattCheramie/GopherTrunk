---
slug: cavity-filter
title: Cavity filter
entry_type: hardware
category: rf-front-end
description: "A cavity filter uses metal resonant chambers as high-Q resonators to build very sharp, low-loss band-pass or notch filters and duplexers for VHF/UHF radio."
keywords: cavity filter, cavity resonator, high-Q filter, duplexer, band-pass, notch cavity, coaxial resonator, VHF, UHF, repeater, preselector
aka: [cavity filter, "cavity resonator filter", "coaxial cavity filter"]
autolink: true
infobox:
  - { label: Type, value: "Resonant-chamber band-pass/notch filter" }
  - { label: Resonator, value: "Quarter-wave coaxial cavity" }
  - { label: Key spec, value: "Very high Q, low insertion loss" }
  - { label: TX, value: "Yes (high power)" }
  - { label: Typical price, value: "$60–$600 per cavity" }
see_also: [q-factor, duplexer, helical-filter, resonance, rf-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Cavity_filter
  - https://en.wikipedia.org/wiki/Resonator
---

A **cavity filter** is a band-pass or band-stop filter built from one or more
hollow metal chambers that act as high-[Q](/reference/q-factor/) resonators.[^wiki]
Each cavity behaves like a shielded quarter-wave [resonant](/reference/resonance/)
line: at its tuned frequency it stores energy with very little loss, producing an
extremely sharp response with insertion loss often below 1 dB. Cavities are the
tool of choice at VHF and UHF whenever you need steep skirts and high power
handling that lumped-element or [helical](/reference/helical-filter/) filters
cannot match.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Cross-section of a coaxial cavity: an outer metal can with a central conductor and a tuning screw, next to a very narrow high-Q resonance peak." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="30" width="90" height="110" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <rect x="80" y="55" width="10" height="85" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <line x1="85" y1="30" x2="85" y2="20" stroke="currentColor" stroke-width="1.8"/>
  <text x="85" y="15" text-anchor="middle" font-size="8" fill="currentColor">tuning screw</text>
  <text x="85" y="152" text-anchor="middle" font-size="8" fill="currentColor">center conductor</text>
  <line x1="200" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M200 138 L300 138 C316 138 316 45 322 45 L326 45 C332 45 332 138 348 138 L440 138" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.8"/>
  <text x="324" y="38" text-anchor="middle" font-size="8" fill="currentColor">high-Q peak</text>
</svg>
<figcaption>A coaxial cavity resonator and the narrow, low-loss response its high Q produces.</figcaption>
</figure>

## Overview

The heart of a cavity is a conductive rod or tube mounted inside a shielded can,
forming a coaxial line roughly a quarter wavelength long at the target frequency.
Because the fields are contained in air (or vacuum) inside plated metal, resistive
and dielectric losses are tiny, so loaded Q values of several thousand are
achievable — far higher than any small LC or ceramic resonator. A **tuning screw**
that changes the effective length lets the resonant frequency be trimmed over a
useful range. Coupling loops or probes feed energy in and out; how tightly they
couple sets the bandwidth and the loaded Q.

## Variants

- **Band-pass cavity** — coupling arranged so the cavity passes its resonant
  frequency and rejects everything else; cascading two to six cavities steepens
  the skirts.
- **Notch (band-reject) cavity** — the cavity is coupled to reject its resonant
  frequency, creating a deep null used to strip out a single strong carrier.
- **Band-pass/band-reject (pass-notch)** — a cavity that passes the wanted
  frequency while placing a sharp notch a fixed offset away, the workhorse of
  repeater [duplexers](/reference/duplexer/).
- **Combline and interdigital** — multi-rod planar arrangements that pack several
  coupled resonators into one housing for compact multi-pole band-pass filters.

## Relevance to SDR

The best-known cavity application is the repeater **duplexer**: a bank of pass-
notch cavities that lets a transmitter and receiver share one antenna by giving
each ~60–90 dB of isolation at frequencies only a fraction of a percent apart.
Cavities also serve as high-performance preselectors and transmitter harmonic
filters at base-station sites.

For an SDR listener, a tuned cavity band-pass filter is one of the most effective
front-end upgrades in dense RF environments — a mountaintop or urban rooftop where
strong pagers, LTE, and broadcast signals would otherwise drive a wideband dongle
into [intermodulation](/reference/intermodulation/). Placed ahead of the
[low-noise amplifier](/reference/low-noise-amplifier/), a cavity's sub-1-dB loss
barely raises the [noise figure](/reference/noise-figure/) while its steep skirts
protect the receiver's [dynamic range](/reference/dynamic-range/). The trade-offs
are size, weight, and the need to tune each cavity precisely to frequency.

GopherTrunk is a software decoder and contains no physical filtering; a cavity
belongs to the analog chain feeding the SDR. But cavities directly affect what
GopherTrunk can do: at a busy [trunking site](/reference/trunking-site/), removing
front-end overload with a cavity preselector is often the difference between a
clean control-channel lock and a noise floor smeared with spurs. Cavity filtering
is a companion to the general [RF filter](/reference/rf-filter/) family rather than
anything GopherTrunk implements in code.

## Sources

[^wiki]: [Cavity filter](https://en.wikipedia.org/wiki/Cavity_filter) — Wikipedia, on resonant-chamber filters, duplexers, and their high-Q behaviour.
