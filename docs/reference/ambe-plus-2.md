---
slug: ambe-plus-2
title: AMBE+2
entry_type: technology
category: voice-coding
description: AMBE+2 is an efficient successor to IMBE/AMBE from DVSI, used by P25 Phase 2, DMR, and NXDN, and supporting half-rate operation for higher capacity.
keywords: AMBE+2, AMBE plus 2, DVSI, DMR, P25 Phase 2, NXDN, half-rate vocoder
aka: [AMBE+2]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder (MBE family) }
  - { label: Developer, value: DVSI }
  - { label: Used by, value: P25 Phase 2, DMR, NXDN }
  - { label: Feature, value: Half-rate operation }
see_also: [vocoder, imbe, ambe, dmr, p25-phase-2, nxdn, dvsi]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**AMBE+2** is the efficient successor to [IMBE](/reference/imbe/) and
[AMBE](/reference/ambe/) from [DVSI](/reference/dvsi/). It powers
[P25 Phase 2](/reference/p25-phase-2/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/), and supports **half-rate** operation.

## How it works

Half-rate AMBE+2 is part of how [DMR](/reference/dmr/) fits two voice
[timeslots](/reference/tdma/) in one channel and how P25 Phase 2 doubles capacity. At a
given bitrate it generally sounds cleaner than IMBE.

## Relevance to SDR

GopherTrunk runs AMBE+2 decoding to produce audio from the most common modern digital
voice systems.
