---
slug: tdma
title: TDMA
entry_type: term
category: trunked-radio
description: TDMA (time-division multiple access) splits one frequency into repeating timeslots so several calls share it; P25 Phase 2, DMR, and TETRA use TDMA.
keywords: TDMA, time division multiple access, timeslot, two-slot, P25 Phase 2, DMR, TETRA, frame, burst, guard time
aka: [TDMA]
autolink: true
infobox:
  - { label: Type, value: Channel-access method }
  - { label: Principle, value: Calls share a frequency in time slots }
  - { label: Used by, value: P25 Phase 2 (2), DMR (2), TETRA (4) }
see_also: [fdma, trunked-radio, p25-phase-2, dmr, tetra, voice-channel, frame-synchronization]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
  - https://en.wikipedia.org/wiki/Channel_access_method
---

**TDMA** (**time-division multiple access**) splits one [frequency](/reference/frequency/)
into rapid, repeating **timeslots**, so two or more calls share the channel by taking
turns.[^wiki] Each call is present only during its own slots but recurs often enough — many
times per second — that the conversation sounds continuous, so one RF channel does the work
of several.

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

Transmitters and receivers agree on a repeating frame divided into equal slots. A radio
bursts its data in its assigned slot and stays silent in the others, so several radios time
share one carrier without colliding. This demands accurate timing: each burst is
surrounded by a small *guard time* to absorb propagation delay, and radios lock onto a
periodic [frame-synchronisation](/reference/frame-synchronization/) pattern to know where
the slot boundaries fall. In exchange for that complexity, TDMA doubles or quadruples the
number of simultaneous calls per frequency and can turn a subscriber's transmitter off
between bursts, extending battery life.

[P25 Phase 2](/reference/p25-phase-2/) and [DMR](/reference/dmr/) use two slots, doubling
capacity per channel; [TETRA](/reference/tetra/) uses four. On a trunked TDMA system a
[grant](/reference/channel-grant/) therefore names both a frequency and a slot, and a
receiver must follow the correct slot as well as the frequency.

## Variants

- **Two-slot TDMA** — DMR and P25 Phase 2 (the P25 Phase 2 air interface is called H-DQPSK
  / TDMA, giving 6.25 kHz-equivalent efficiency in a 12.5 kHz channel).
- **Four-slot TDMA** — TETRA, packing four calls into a 25 kHz carrier.
- **FDMA/TDMA hybrid** — most cellular and trunked networks channelise with FDMA and then
  time-share each channel, combining the two access methods.

## In practice

TDMA is the efficiency play for capacity-constrained systems: a two-slot DMR repeater
carries two independent conversations on one frequency pair, which is why DMR became
popular for commercial and amateur repeaters. The cost is stricter timing and more complex
radios and receivers, and the fact that a single carrier now carries multiple unrelated
calls that a monitor must separate by slot.

## Relevance to SDR

On a TDMA system a single granted [voice channel](/reference/voice-channel/) can carry two
(or four) simultaneous calls, so GopherTrunk decodes the relevant slot of the assigned
channel rather than the whole carrier. Recovering the slot requires frame synchronisation
and burst timing, which GopherTrunk performs as part of its P25 Phase 2, DMR, and TETRA
decode paths — contrast [FDMA](/reference/fdma/), where the whole frequency is one call.

## Sources

[^wiki]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on sharing one frequency via repeating timeslots.
[^cam]: [Channel access method](https://en.wikipedia.org/wiki/Channel_access_method) — Wikipedia, comparing TDMA with FDMA and other multiplexing schemes.
