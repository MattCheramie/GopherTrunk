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
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
---

**AMBE** (**Advanced Multi-Band Excitation**) is a family of low-bitrate speech
[vocoders](/reference/vocoder/) from [DVSI](/reference/dvsi/), building on the
[MBE](/reference/multi-band-excitation/) model.[^wiki] It is used by
[D-STAR](/reference/d-star/) and EDACS ProVoice, and is the basis for the more efficient
[AMBE+2](/reference/ambe-plus-2/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A stream of small AMBE voice frames each carrying voice bits plus error-correction bits." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none"><rect x="40" y="40" width="90" height="32"/><rect x="140" y="40" width="90" height="32"/><rect x="240" y="40" width="90" height="32"/><rect x="340" y="40" width="90" height="32"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="85" y="60">voice+FEC</text><text x="185" y="60">voice+FEC</text><text x="285" y="60">voice+FEC</text><text x="385" y="60">voice+FEC</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="9" fill="currentColor">compact frames, ~20 ms each</text>
</svg>
<figcaption>AMBE is the multi-band excitation vocoder family behind many digital-voice systems.</figcaption>
</figure>

## How it works

Like other MBE vocoders, AMBE separates voiced (pitched) and unvoiced (noisy) bands and
transmits compact spectral parameters, reconstructing speech at the receiver from a few
kbps.

## Relevance to SDR

GopherTrunk implements AMBE-family decoding in pure Go to render digital voice without
proprietary hardware.

## Sources

[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE vocoder family that includes AMBE.
