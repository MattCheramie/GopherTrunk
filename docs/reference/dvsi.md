---
slug: dvsi
title: Digital Voice Systems (DVSI)
entry_type: organization
category: organizations
description: Digital Voice Systems, Inc. (DVSI) is the company that develops and licenses the IMBE and AMBE vocoder families used by P25, DMR, NXDN, and D-STAR.
keywords: DVSI, Digital Voice Systems, IMBE, AMBE, AMBE+2, vocoder licensing, patents, multi-band excitation, half-rate vocoder
aka: [DVSI, Digital Voice Systems]
autolink: true
infobox:
  - { label: Type, value: Company }
  - { label: Products, value: IMBE, AMBE, AMBE+2 vocoders }
  - { label: Note, value: Patented/licensed codecs }
see_also: [vocoder, imbe, ambe, ambe-plus-2, multi-band-excitation, codec2]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
cite_urls:
  - https://www.dvsinc.com/
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
---

**Digital Voice Systems, Inc.** (**DVSI**) is the company that develops and licenses the
[IMBE](/reference/imbe/) and [AMBE](/reference/ambe/) / [AMBE+2](/reference/ambe-plus-2/)
[vocoder](/reference/vocoder/) families based on
[multi-band excitation](/reference/multi-band-excitation/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="DVSI develops and licenses the patented AMBE and IMBE vocoders adopted by P25, DMR, NXDN, and D-STAR." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">DVSI</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="55">IMBE / AMBE+2</text><text x="225" y="67" font-size="7.5">licensed vocoders</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">P25 · DMR</text><text x="385" y="67" font-size="7.5">NXDN · D-STAR</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_dvsi)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_dvsi)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">patented / royalty-bearing → open Codec 2 exists as alternative</text>
  </g>
  <defs><marker id="rel_dvsi" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DVSI develops and licenses the patented AMBE and IMBE vocoders that most digital land-mobile systems adopted.</figcaption>
</figure>

## Overview

Digital Voice Systems, Inc. is a US company, based in Massachusetts, founded around 1990 to
commercialise the Multi-Band Excitation (MBE) speech-coding research pioneered at MIT. Its
central idea is a low-bit-rate [vocoder](/reference/vocoder/): rather than transmitting the
speech waveform, it analyses each short frame of voice into a compact set of parameters — pitch,
spectral envelope, and a per-band voiced/unvoiced decision — and the receiver resynthesises
intelligible speech from those parameters. This lets usable voice fit into the few thousand
bits per second a narrowband digital radio channel can carry, which is precisely what
land-mobile radio needed.

DVSI's product line evolved through several generations. **[IMBE](/reference/imbe/)** (Improved
Multi-Band Excitation) was adopted for [P25 Phase 1](/reference/p25-phase-1/) at 7.2 kbit/s
including [FEC](/reference/forward-error-correction/). **[AMBE](/reference/ambe/)** (Advanced
MBE) and then **[AMBE+2](/reference/ambe-plus-2/)** improved quality at lower rates and became
the workhorse of the digital land-mobile world: AMBE+2 is used by
[P25 Phase 2](/reference/p25-phase-2/), [DMR](/reference/dmr/), [NXDN](/reference/nxdn/), and,
in a related IMBE/AMBE variant, amateur [D-STAR](/reference/d-star/). The technology is
protected by a portfolio of patents, and DVSI licenses it both as software and as hardware chips
(the AMBE-3000 series), often with the vocoder implementation locked inside the chip to protect
the intellectual property. That commercial, royalty-bearing model is deliberate — and it is the
direct reason open, patent-free alternatives such as [Codec 2](/reference/codec2/) were created
for projects like [M17](/reference/m17/) that wanted to avoid licensing entirely.

## Relevance to SDR

The vocoder is the last mile of decoding digital voice, and DVSI's codecs sit at that last mile
for nearly every system GopherTrunk targets. You can demodulate a [P25](/reference/project-25/)
or [DMR](/reference/dmr/) signal perfectly, recover a clean bitstream, and still hear nothing
intelligible until the AMBE/IMBE frames are turned back into audio — that final step is DVSI's
technology. The patent situation is why this step has historically been awkward for open-source
receivers: some tools shell out to a hardware AMBE dongle, and the legality of software
reimplementations is a genuinely contested question that has shaped how the whole ecosystem is
built.

GopherTrunk implements the relevant multi-band-excitation vocoders in pure Go so it can turn
decoded digital-voice frames into audio without external hardware, while being mindful of the
patent and licensing landscape DVSI created. It is worth being clear about scope: the vocoder
decodes *clear or scrambled* voice payloads; it has nothing to do with keyed
[encryption](/reference/encryption/), which GopherTrunk cannot break. For the bigger picture of
how these codecs squeeze speech into a radio channel, see the
[vocoders — IMBE & AMBE+2](/learn/rf-sdr/vocoders/) lesson.

## Sources

[^home]: [Digital Voice Systems, Inc.](https://www.dvsinc.com/) — DVSI's official site, for its IMBE, AMBE, and AMBE+2 vocoder products and licensing.
[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on DVSI's IMBE/AMBE multi-band-excitation vocoders and their licensing.
