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

## How it works

[P25 Phase 2](/reference/p25-phase-2/) and [DMR](/reference/dmr/) use two slots,
doubling capacity per channel; [TETRA](/reference/tetra/) uses four. A receiver must
follow the correct slot as well as the frequency.

## Relevance to SDR

On a TDMA system a single granted [voice channel](/reference/voice-channel/) can carry
two simultaneous calls, so GopherTrunk decodes the relevant slot of the assigned channel.
