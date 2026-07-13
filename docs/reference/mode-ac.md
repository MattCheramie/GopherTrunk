---
slug: mode-ac
title: Mode A/C
entry_type: protocol
category: aviation-marine
description: "Mode A/C is the legacy pulse-coded aviation transponder scheme in which aircraft reply on 1090 MHz to 1030 MHz radar interrogations with a squawk code or altitude."
keywords: Mode A, Mode C, Mode A/C, transponder, squawk code, SSR, secondary surveillance radar, 1030 MHz, 1090 MHz, altitude reporting, framing pulses
aka: [Mode A/C, Mode A, Mode C, Mode 3/A]
autolink: true
infobox:
  - { label: Type, value: SSR transponder reply }
  - { label: Standards body, value: "ICAO Annex 10 / RTCA" }
  - { label: Interrogation, value: 1030 MHz }
  - { label: Reply, value: 1090 MHz }
  - { label: Modulation, value: Pulse-position (framing pulses) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [mode-s, ads-b, tcas, multilateration, pulse-position-modulation, icao]
cite_urls:
  - https://en.wikipedia.org/wiki/Secondary_surveillance_radar
  - https://en.wikipedia.org/wiki/Transponder_(aeronautics)
---

**Mode A/C** is the legacy [secondary surveillance radar](/reference/multilateration/)
transponder scheme in which an aircraft replies on **1090 MHz** to a ground radar's
**1030 MHz** interrogation with a burst of pulses encoding either a four-digit
**squawk code** (Mode A) or **pressure altitude** (Mode C).[^wiki] It predates the
selective-addressing [Mode S](/reference/mode-s/) protocol that underlies
[ADS-B](/reference/ads-b/), and the two coexist on the same frequency pair because
Mode S was designed to be backward-compatible with Mode A/C interrogators.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A ground radar interrogates on 1030 megahertz and the aircraft transponder replies on 1090 megahertz with framing pulses bracketing twelve information pulses." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="macar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="60" y="30" text-anchor="middle" font-size="8" fill="currentColor">radar</text>
  <path d="M60 34 v12 m-6 0 l6 -8 l6 8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <path d="M80 40 h180" stroke="currentColor" stroke-width="1.2" marker-end="url(#macar)"/>
  <text x="170" y="35" text-anchor="middle" font-size="8" fill="currentColor">1030 MHz interrogation</text>
  <path d="M300 34 l30 8 l-8 -14 m8 14 l14 -10" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="330" y="30" text-anchor="middle" font-size="8" fill="currentColor">aircraft</text>
  <path d="M300 56 h-220" stroke="currentColor" stroke-width="1.2" marker-end="url(#macar)"/>
  <text x="190" y="70" text-anchor="middle" font-size="8" fill="currentColor">1090 MHz reply</text>
  <g stroke="currentColor" stroke-width="1.2" fill="currentColor" fill-opacity="0.4">
    <rect x="40" y="95" width="6" height="20"/><rect x="70" y="95" width="6" height="20"/><rect x="88" y="95" width="6" height="20"/><rect x="112" y="95" width="6" height="20"/><rect x="150" y="95" width="6" height="20"/><rect x="182" y="95" width="6" height="20"/>
    <rect x="206" y="95" width="6" height="20"/>
  </g>
  <text x="43" y="127" font-size="7.5" fill="currentColor">F1</text><text x="205" y="127" font-size="7.5" fill="currentColor">F2</text>
  <text x="130" y="127" text-anchor="middle" font-size="7.5" fill="currentColor">12 info pulses (20.3 µs frame)</text>
</svg>
<figcaption>Mode A/C: a 1030 MHz ground interrogation triggers a 1090 MHz reply whose two framing pulses (F1, F2) bracket up to 12 information pulses.</figcaption>
</figure>

## Overview

Mode A/C is the pulse-only heart of the international Secondary Surveillance Radar
(SSR) system. A rotating ground antenna transmits interrogation pulse pairs on
1030 MHz; the mode is selected by the *spacing* between the pulses (8 µs for Mode A,
21 µs for Mode C). Every transponder in the beam that hears the interrogation replies
on 1090 MHz, so unlike Mode S there is no addressing — the radar sorts replies by the
azimuth of its antenna and the round-trip timing.

## Technical characteristics

| Property | Value |
|----------|-------|
| Interrogation frequency | 1030 MHz |
| Reply frequency | 1090 MHz |
| Mode A trigger | P1–P3 pulses spaced 8 µs |
| Mode C trigger | P1–P3 pulses spaced 21 µs |
| Reply frame | F1 … F2 framing pulses 20.3 µs apart |
| Information pulses | 12 (A, B, C, D groups) |
| Mode A capacity | 4096 codes (octal 0000–7777) |
| Mode C encoding | Gillham/Gray-coded altitude in 100 ft steps |

The reply is a train of up to 12 information pulses bracketed by two framing pulses
(F1 and F2) exactly 20.3 µs apart. In Mode A those 12 bits carry the pilot-set
four-digit octal **squawk** (e.g. 7500 hijack, 7600 radio failure, 7700 emergency).
In Mode C the same 12 positions carry pressure altitude in 100-foot increments using
a [Gray-code](/reference/gray-code/)-derived Gillham encoding. A special position
identification (SPI) pulse can follow F2 when the pilot presses "ident".

## History

Mode A/C descends directly from wartime IFF ("Identification Friend or Foe") and was
standardised by [ICAO](/reference/icao/) in Annex 10. It served as the sole
cooperative surveillance mode for decades. Its weaknesses — no unique address, garbled
overlapping replies ("FRUIT" and synchronous garble) when many aircraft share a beam,
and only 4096 possible codes — drove the development of [Mode S](/reference/mode-s/)
selective interrogation and, later, [ADS-B](/reference/ads-b/).

## Deployment

Mode A/C transponders remain in wide use, especially in general aviation, and every
Mode S transponder still answers legacy Mode A/C interrogations for backward
compatibility. Ground SSR and airborne [TCAS](/reference/tcas/) units both interrogate
Mode A/C. The squawk code is the tag air-traffic control assigns to correlate a radar
return with a flight plan.

## Decoding it with GopherTrunk

**Not decoded.** Mode A/C carries no aircraft identity or absolute position on its own
(a bare squawk plus altitude), and its unaddressed, radar-triggered replies are of
limited value to a passive scanner. GopherTrunk's aviation support targets the
information-rich [Mode S](/reference/mode-s/) extended squitter and
[ADS-B](/reference/ads-b/) instead. Mode A/C replies are, however, the raw material for
ground-based **[multilateration](/reference/multilateration/)** networks, which
time-difference the 1090 MHz bursts across several receivers to locate non-ADS-B
aircraft.

## Sources

[^wiki]: [Secondary surveillance radar](https://en.wikipedia.org/wiki/Secondary_surveillance_radar) — Wikipedia, for the 1030/1090 MHz interrogation-reply scheme, Mode A squawk codes, Mode C altitude encoding, framing pulses, and the relationship to Mode S.
