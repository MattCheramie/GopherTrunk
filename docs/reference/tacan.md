---
slug: tacan
title: Tactical Air Navigation (TACAN)
entry_type: technology
category: aviation-marine
description: TACAN (Tactical Air Navigation) is a military navaid that gives an aircraft both bearing and distance from a single L-band ground or shipborne beacon, reusing the DME distance function.
keywords: TACAN, Tactical Air Navigation, military navaid, bearing and distance, L-band, VORTAC, DME, 15 Hz 135 Hz, rotating pattern, shipborne beacon
aka: [TACAN]
autolink: true
infobox:
  - { label: Type, value: Military bearing + distance navaid }
  - { label: Idea, value: DME distance plus amplitude-modulated bearing }
  - { label: Band, value: L-band 960–1215 MHz }
see_also: [dme, vor, ils, pulse-position-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Tactical_air_navigation_system
  - https://www.icao.int/
---

**TACAN** (**Tactical Air Navigation**) is a military radio navaid that provides an
aircraft with **both bearing and distance from a single beacon** in the L-band
(960–1215 MHz). Its distance function is identical to civil
[DME](/reference/dme/) — the same coded pulse-pair round trip — while its bearing function
is built into the modulation of those same pulses, so one compact, often transportable or
shipborne, ground unit delivers a full fix.[^wiki] TACAN is the military counterpart to
the civil [VOR](/reference/vor/)/DME pairing, and the two are deliberately made to
interoperate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A TACAN beacon whose rotating antenna pattern amplitude-modulates the pulse stream to encode bearing, while pulse-pair timing gives distance." xmlns="http://www.w3.org/2000/svg">
  <circle cx="120" cy="85" r="7" fill="currentColor"/>
  <circle cx="120" cy="85" r="60" fill="none" stroke="currentColor" stroke-opacity="0.2"/>
  <path d="M120 85 C170 45 200 60 180 100 C165 130 130 125 120 85 Z" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="120" y="160" text-anchor="middle" font-size="8" fill="currentColor">rotating pattern → 15 Hz + 135 Hz bearing modulation</text>
  <g font-size="8" fill="currentColor">
    <text x="300" y="60">distance: DME pulse-pair timing</text>
    <text x="300" y="85">bearing: AM of the pulse envelope</text>
    <text x="300" y="110">one beacon = full position fix</text>
  </g>
</svg>
<figcaption>TACAN reuses the DME pulse-pair ranging and adds bearing by amplitude-modulating the pulse stream with a rotating antenna pattern.</figcaption>
</figure>

## How it works

TACAN's ranging half is DME: the aircraft interrogates with pulse pairs and the beacon
replies after the standard fixed delay, and round-trip time gives slant range. The
bearing half comes from the beacon's antenna radiating a **rotating pattern** that
amplitude-modulates the pulse stream at 15 Hz (a coarse indication) and, from a finer
nine-lobe pattern, at 135 Hz. Reference bursts mark the pattern's orientation, and the
airborne set compares the phase of the received 15 Hz and 135 Hz envelope modulation
against those references to resolve bearing to about a degree — more precisely than a
conventional VOR, because the 135 Hz component sharpens the reading.

Where a civil VOR and a TACAN are co-located, the facility is called a **VORTAC**: civil
aircraft take VOR bearing plus TACAN's DME distance, and military aircraft take full
TACAN. TACAN is compact enough to mount on ships and vehicles, which is central to its
tactical role.

## Relevance to SDR

TACAN shares the pulsed L-band environment of DME and [Mode S](/reference/mode-s/), so it
is not a beginner [SDR](/reference/software-defined-radio/) target, but it illustrates how
one signal can multiplex distance (pulse timing) and bearing (envelope amplitude
modulation) at once. **GopherTrunk** does not decode TACAN; it is a land-mobile trunking
scanner, and TACAN is included as reference context alongside DME, VOR, and
[ILS](/reference/ils/).

## Sources

[^wiki]: [Tactical air navigation system](https://en.wikipedia.org/wiki/Tactical_air_navigation_system) — Wikipedia, for TACAN's combined bearing/distance operation, its shared DME ranging, the 15 Hz/135 Hz bearing modulation, and the VORTAC arrangement.
