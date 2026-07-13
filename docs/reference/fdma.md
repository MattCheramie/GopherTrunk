---
slug: fdma
title: FDMA
entry_type: term
category: trunked-radio
description: FDMA (frequency-division multiple access) gives each call its own frequency channel; P25 Phase 1 and NXDN use FDMA.
keywords: FDMA, frequency division multiple access, channel access, P25 Phase 1, NXDN, dPMR, guard band, channel spacing
aka: [FDMA]
autolink: true
infobox:
  - { label: Type, value: Channel-access method }
  - { label: Principle, value: One call per frequency }
  - { label: Used by, value: P25 Phase 1, NXDN, dPMR }
see_also: [tdma, trunked-radio, p25-phase-1, nxdn, dpmr, voice-channel, guard-band]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-division_multiple_access
  - https://en.wikipedia.org/wiki/Channel_access_method
---

**FDMA** (**frequency-division multiple access**) is a channel-access method in which
each call occupies its own [frequency](/reference/frequency/) channel.[^wiki] It is the
simplest way to let many users share the spectrum: divide the band into non-overlapping
channels and give one conversation to each, separated by small [guard
bands](/reference/guard-band/) so adjacent channels do not interfere.

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

Each active conversation transmits continuously on a dedicated carrier for the duration of
the call. Capacity scales directly with the number of frequencies available: adding
conversations means adding channels. There is no need to synchronise transmitters in time,
which keeps subscriber radios simple and tolerant of propagation delay — one reason FDMA
suited early land-mobile systems. The trade-off is spectral: a channel occupied by a quiet
talker still ties up its whole frequency, and narrowing channels to fit more of them runs
into filter selectivity and adjacent-channel limits.

Narrowbanding has pushed FDMA voice channels progressively tighter — from 25 kHz analog FM
down to the 12.5 kHz of [P25 Phase 1](/reference/p25-phase-1/) [C4FM](/reference/c4fm/) and
the 6.25 kHz of [NXDN](/reference/nxdn/) and [dPMR](/reference/dpmr/) — squeezing more
channels into the same band without changing the one-call-per-frequency principle.

## Variants

- **Analog FDMA** — conventional 12.5/25 kHz FM channels; the original land-mobile model.
- **Digital narrowband FDMA** — P25 Phase 1 (12.5 kHz), NXDN and dPMR (6.25 kHz), which
  achieve 6.25 kHz-equivalent efficiency by modulation alone rather than time division.
- **Hybrid access** — many cellular and trunked systems combine FDMA (channelising the
  band) with [TDMA](/reference/tdma/) (sharing each channel in time).

## In practice

FDMA remains the backbone of public-safety P25 Phase 1 across North America and of NXDN in
utility and commercial fleets. Its per-frequency simplicity makes conventional
narrowbanding straightforward and gives good range, at the cost of the spectral efficiency
that [TDMA](/reference/tdma/) achieves by packing two or more calls onto one carrier.

## Relevance to SDR

On an FDMA system each [voice channel](/reference/voice-channel/) is a separate frequency
the receiver tunes to, and there is no timeslot to track — a granted channel is simply a
carrier to demodulate. GopherTrunk handles this by digitally down-converting the granted
frequency from a wideband capture, in contrast with [TDMA](/reference/tdma/), where it must
also select the correct slot.

## Sources

[^wiki]: [Frequency-division multiple access](https://en.wikipedia.org/wiki/Frequency-division_multiple_access) — Wikipedia, on the one-call-per-frequency access method.
[^cam]: [Channel access method](https://en.wikipedia.org/wiki/Channel_access_method) — Wikipedia, comparing FDMA, TDMA, and other multiplexing schemes.
