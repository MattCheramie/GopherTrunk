---
slug: ambe
title: AMBE
entry_type: technology
category: voice-coding
description: AMBE (Advanced Multi-Band Excitation) is a family of low-bitrate speech vocoders from DVSI used across digital voice radio, including D-STAR and as the basis for AMBE+2.
keywords: AMBE, Advanced Multi-Band Excitation, DVSI, D-STAR, vocoder, low bitrate speech
aka: [AMBE]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder family (MBE) }
  - { label: Developer, value: DVSI }
  - { label: Used by, value: D-STAR, ProVoice; basis of AMBE+2 }
see_also: [vocoder, imbe, ambe-plus-2, multi-band-excitation, d-star, dvsi]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**AMBE** (**Advanced Multi-Band Excitation**) is a family of low-bitrate speech
[vocoders](/reference/vocoder/) from [DVSI](/reference/dvsi/), building on the
[MBE](/reference/multi-band-excitation/) model. It is used by
[D-STAR](/reference/d-star/) and EDACS ProVoice, and is the basis for the more efficient
[AMBE+2](/reference/ambe-plus-2/).

## How it works

Like other MBE vocoders, AMBE separates voiced (pitched) and unvoiced (noisy) bands and
transmits compact spectral parameters, reconstructing speech at the receiver from a few
kbps.

## Relevance to SDR

GopherTrunk implements AMBE-family decoding in pure Go to render digital voice without
proprietary hardware.
