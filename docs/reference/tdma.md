---
slug: tdma
title: TDMA
entry_type: term
category: trunked-radio
description: TDMA (time-division multiple access) splits one frequency into repeating timeslots so several calls share it; P25 Phase 2, DMR, and TETRA use TDMA.
keywords: TDMA, time division multiple access, timeslot, two-slot, P25 Phase 2, DMR, TETRA
aka: [TDMA]
autolink: true
infobox:
  - { label: Type, value: Channel-access method }
  - { label: Principle, value: Calls share a frequency in time slots }
  - { label: Used by, value: P25 Phase 2 (2), DMR (2), TETRA (4) }
see_also: [fdma, trunked-radio, p25-phase-2, dmr, tetra]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Time-division multiple access (Wikipedia)", url: https://en.wikipedia.org/wiki/Time-division_multiple_access }
---

**TDMA** (**time-division multiple access**) splits one [frequency](/reference/frequency/)
into rapid, repeating **timeslots**, so two or more calls share the channel by taking
turns.

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="A single frequency channel divided along time into repeating slots, alternating between two calls." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="110" x2="340" y2="110" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#tdar)"/>
  <text x="185" y="135" text-anchor="middle" font-size="9" fill="currentColor">time →</text>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="40" y="40" width="50" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="90" y="40" width="50" height="50" fill="none"/>
    <rect x="140" y="40" width="50" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="190" y="40" width="50" height="50" fill="none"/>
    <rect x="240" y="40" width="50" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="290" y="40" width="50" height="50" fill="none"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="65" y="69">1</text><text x="115" y="69">2</text><text x="165" y="69">1</text><text x="215" y="69">2</text><text x="265" y="69">1</text><text x="315" y="69">2</text></g>
  <text x="185" y="28" text-anchor="middle" font-size="9" fill="currentColor">one frequency, two calls share the slots</text>
  <defs><marker id="tdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>TDMA splits one frequency into time slots so several calls share it — used by P25 Phase 2 and DMR.</figcaption>
</figure>

## How it works

[P25 Phase 2](/reference/p25-phase-2/) and [DMR](/reference/dmr/) use two slots,
doubling capacity per channel; [TETRA](/reference/tetra/) uses four. A receiver must
follow the correct slot as well as the frequency.

## Relevance to SDR

On a TDMA system a single granted [voice channel](/reference/voice-channel/) can carry
two simultaneous calls, so GopherTrunk decodes the relevant slot of the assigned channel.
