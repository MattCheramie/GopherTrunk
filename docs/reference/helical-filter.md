---
slug: helical-filter
title: Helical filter
entry_type: hardware
category: rf-front-end
description: "A helical filter uses coiled quarter-wave resonators in shielded cans to make compact, high-Q VHF/UHF band-pass filters, smaller than cavities."
keywords: helical filter, helical resonator, band-pass, VHF, UHF, high-Q, preselector, coiled resonator, shielded can filter
aka: [helical filter, "helical resonator filter"]
autolink: true
infobox:
  - { label: Type, value: "Coiled-resonator band-pass filter" }
  - { label: Resonator, value: "Helical quarter-wave coil in a shielded can" }
  - { label: Key spec, value: "High Q, compact at VHF/UHF" }
  - { label: TX, value: "Low to moderate power" }
  - { label: Typical price, value: "$10–$80 (module)" }
see_also: [cavity-filter, q-factor, rf-filter, resonance, saw-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Helical_resonator
  - https://en.wikipedia.org/wiki/Cavity_filter
---

A **helical filter** is a band-pass filter whose resonators are helices — coils of
wire mounted inside shielded metal cans.[^wiki] Coiling the conductor makes each
resonator electrically a quarter wavelength long while keeping it physically short,
so a helical filter delivers much of a [cavity](/reference/cavity-filter/) filter's
high [Q](/reference/q-factor/) and selectivity in a package small enough to sit on
a receiver board. It occupies the sweet spot between bulky cavities and lossy
lumped-element LC filters across roughly 30 MHz to 1 GHz.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Three shielded cans each holding a coiled helical resonator, coupled through apertures to form a multi-pole band-pass filter." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="35" width="70" height="95" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <rect x="130" y="35" width="70" height="95" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <rect x="220" y="35" width="70" height="95" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <path d="M60 120 q10 -8 20 0 q10 8 20 0 M60 105 q10 -8 20 0 q10 8 20 0 M60 90 q10 -8 20 0 q10 8 20 0 M60 75 q10 -8 20 0 q10 8 20 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M150 120 q10 -8 20 0 q10 8 20 0 M150 105 q10 -8 20 0 q10 8 20 0 M150 90 q10 -8 20 0 q10 8 20 0 M150 75 q10 -8 20 0 q10 8 20 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M240 120 q10 -8 20 0 q10 8 20 0 M240 105 q10 -8 20 0 q10 8 20 0 M240 90 q10 -8 20 0 q10 8 20 0 M240 75 q10 -8 20 0 q10 8 20 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="110" y1="70" x2="130" y2="70" stroke="currentColor" stroke-dasharray="3 2"/>
  <line x1="200" y1="70" x2="220" y2="70" stroke="currentColor" stroke-dasharray="3 2"/>
  <text x="330" y="86" font-size="9" fill="currentColor">coupled cans</text>
  <text x="330" y="100" font-size="9" fill="currentColor">= sharp BPF</text>
</svg>
<figcaption>Coiled quarter-wave resonators in coupled shielded cans form a compact, high-Q band-pass filter.</figcaption>
</figure>

## Overview

A helical resonator is a single-layer coil grounded at one end and open at the
other, enclosed in a square or round shield. The shield both confines the field
(preserving Q) and forms part of the resonant structure. Because most of the
electrical length is folded into turns, a resonator only a few centimetres tall
can resonate at 150 MHz — where an open quarter-wave line would be half a metre.
Unloaded Q typically runs from several hundred up to a couple of thousand: below a
cavity's, but far above any small LC or ceramic part. A tuning slug or capacitor
at the open end trims the frequency.

## Variants

Single-resonator helical filters make simple, cheap preselectors. Cascading two to
four resonators in adjacent shielded cans, coupled through apertures or small
loops, builds a multi-pole [band-pass](/reference/rf-filter/) response with steep
skirts. Helical *notch* configurations also exist. The same geometry doubles as a
[helical antenna](/reference/helical-antenna/) when the coil is left unshielded and
made to radiate — a related structure with an opposite purpose.

## Relevance to SDR

Helical filters are common as the preselector inside VHF/UHF scanners, land-mobile
radios, and receiver front-end modules, and they are a practical add-on for SDR
users who want cavity-grade selectivity without cavity-grade bulk. A helical
band-pass module tuned to the target band, placed ahead of the
[low-noise amplifier](/reference/low-noise-amplifier/), rejects out-of-band
signals that would otherwise cause [intermodulation](/reference/intermodulation/)
in a wideband dongle, improving effective
[dynamic range](/reference/dynamic-range/). Their insertion loss (a couple of dB)
is higher than a cavity's but their size and cost are far lower, making them a
popular middle ground.

GopherTrunk performs no analog filtering — a helical filter lives in the hardware
chain feeding the SDR. Its practical value to a GopherTrunk user is the same as any
good preselector: at a congested [trunking site](/reference/trunking-site/), it
keeps strong nearby transmitters from swamping the receiver so the software can
maintain a clean control-channel lock. It sits alongside SAW and cavity filters in
the broader [RF filter](/reference/rf-filter/) toolkit rather than being something
GopherTrunk implements.

## Sources

[^wiki]: [Helical resonator](https://en.wikipedia.org/wiki/Helical_resonator) — Wikipedia, on coiled quarter-wave resonators and the compact high-Q filters built from them.
