---
slug: imbe
title: IMBE
entry_type: technology
category: voice-coding
description: IMBE (Improved Multi-Band Excitation) is the vocoder used by P25 Phase 1, encoding voice at about 7.2 kbps including error correction.
keywords: IMBE, Improved Multi-Band Excitation, P25 Phase 1, vocoder, DVSI, 7200 bps
aka: [IMBE]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder (MBE family) }
  - { label: Bit rate, value: ~7.2 kbps (incl. FEC) }
  - { label: Used by, value: P25 Phase 1 }
  - { label: Licensor, value: DVSI }
see_also: [vocoder, ambe, ambe-plus-2, multi-band-excitation, p25-phase-1, dvsi]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**IMBE** (**Improved Multi-Band Excitation**) is the [vocoder](/reference/vocoder/) of
[P25 Phase 1](/reference/p25-phase-1/), part of the
[MBE](/reference/multi-band-excitation/) codec family from [DVSI](/reference/dvsi/).

## How it works

IMBE runs at about **7.2 kbps** over the air, of which roughly 4.4 kbps is voice and the
rest is [forward error correction](/reference/forward-error-correction/) protecting the
parameters — important because a corrupted parameter sounds far worse than a corrupted
audio sample.

## Relevance to SDR

GopherTrunk decodes IMBE frames from P25 Phase 1 and synthesises audio. Its successor,
[AMBE+2](/reference/ambe-plus-2/), is used by newer systems.
