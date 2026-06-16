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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A 12.5 kHz channel split into two TDMA slots, each carrying a half-rate AMBE+2 voice stream." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="380" height="40" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <line x1="230" y1="40" x2="230" y2="80" stroke="currentColor" stroke-width="1.2"/>
  <text x="135" y="64" text-anchor="middle" font-size="9" fill="currentColor">slot 1 · AMBE+2</text>
  <text x="325" y="64" text-anchor="middle" font-size="9" fill="currentColor">slot 2 · AMBE+2</text>
  <text x="230" y="28" text-anchor="middle" font-size="9" fill="currentColor">one 12.5 kHz channel</text>
  <text x="230" y="100" text-anchor="middle" font-size="9" fill="currentColor">half-rate coding fits two voices where one used to go</text>
</svg>
<figcaption>AMBE+2 is a more efficient successor used by P25 Phase 2, DMR, and NXDN, supporting half-rate streams.</figcaption>
</figure>

## How it works

Half-rate AMBE+2 is part of how [DMR](/reference/dmr/) fits two voice
[timeslots](/reference/tdma/) in one channel and how P25 Phase 2 doubles capacity. At a
given bitrate it generally sounds cleaner than IMBE.

## Relevance to SDR

GopherTrunk runs AMBE+2 decoding to produce audio from the most common modern digital
voice systems.
