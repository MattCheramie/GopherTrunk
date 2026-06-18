---
slug: fdma
title: FDMA
entry_type: term
category: trunked-radio
description: FDMA (frequency-division multiple access) gives each call its own frequency channel; P25 Phase 1 and NXDN use FDMA.
keywords: FDMA, frequency division multiple access, channel access, P25 Phase 1, NXDN
aka: [FDMA]
autolink: true
infobox:
  - { label: Type, value: Channel-access method }
  - { label: Principle, value: One call per frequency }
  - { label: Used by, value: P25 Phase 1, NXDN, dPMR }
see_also: [tdma, trunked-radio, p25-phase-1, nxdn]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
external:
  - { title: "Frequency-division multiple access (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-division_multiple_access }
---

**FDMA** (**frequency-division multiple access**) is a channel-access method in which
each call occupies its own [frequency](/reference/frequency/) channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 200" role="img" aria-label="A frequency axis split into separate stacked channels, each carrying one continuous call." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="170" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fdar)"/>
  <text x="20" y="95" font-size="9" fill="currentColor" transform="rotate(-90 20 95)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="30" width="220" height="28"/><rect x="50" y="66" width="220" height="28"/><rect x="50" y="102" width="220" height="28"/><rect x="50" y="138" width="220" height="28"/>
  </g>
  <g font-size="9" fill="currentColor"><text x="160" y="49" text-anchor="middle">call A</text><text x="160" y="85" text-anchor="middle">call B</text><text x="160" y="121" text-anchor="middle">call C</text><text x="160" y="157" text-anchor="middle">call D</text></g>
  <defs><marker id="fdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>FDMA gives each call its own frequency channel — the access method of P25 Phase 1.</figcaption>
</figure>

## How it works

Capacity scales with the number of frequencies available; adding conversations means
adding channels. [P25 Phase 1](/reference/p25-phase-1/), [NXDN](/reference/nxdn/), and
[dPMR](/reference/dpmr/) use FDMA.

## Relevance to SDR

On an FDMA system each [voice channel](/reference/voice-channel/) is a separate
frequency the receiver tunes to; contrast with [TDMA](/reference/tdma/), which shares one
frequency across time slots.
