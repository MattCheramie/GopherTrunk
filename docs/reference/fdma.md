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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Frequency-division multiple access (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-division_multiple_access }
---

**FDMA** (**frequency-division multiple access**) is a channel-access method in which
each call occupies its own [frequency](/reference/frequency/) channel.

## How it works

Capacity scales with the number of frequencies available; adding conversations means
adding channels. [P25 Phase 1](/reference/p25-phase-1/), [NXDN](/reference/nxdn/), and
[dPMR](/reference/dpmr/) use FDMA.

## Relevance to SDR

On an FDMA system each [voice channel](/reference/voice-channel/) is a separate
frequency the receiver tunes to; contrast with [TDMA](/reference/tdma/), which shares one
frequency across time slots.
