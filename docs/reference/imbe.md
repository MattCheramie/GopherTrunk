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
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**IMBE** (**Improved Multi-Band Excitation**) is the [vocoder](/reference/vocoder/) of
[P25 Phase 1](/reference/p25-phase-1/), part of the
[MBE](/reference/multi-band-excitation/) codec family from [DVSI](/reference/dvsi/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A speech spectrum split into frequency bands, each marked voiced or unvoiced, as in multi-band excitation." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="90" x2="430" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.2"><line x1="90" y1="90" x2="90" y2="35"/><line x1="150" y1="90" x2="150" y2="50"/><line x1="210" y1="90" x2="210" y2="42"/><line x1="270" y1="90" x2="270" y2="60"/><line x1="330" y1="90" x2="330" y2="48"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="90" y="105">V</text><text x="150" y="105">V</text><text x="210" y="105">U</text><text x="270" y="105">V</text><text x="330" y="105">U</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="9" fill="currentColor">each band marked voiced (V) or unvoiced (U)</text>
</svg>
<figcaption>IMBE (used by P25 Phase 1) is a multi-band excitation codec that labels each spectral band voiced or unvoiced.</figcaption>
</figure>

## How it works

IMBE runs at about **7.2 kbps** over the air, of which roughly 4.4 kbps is voice and the
rest is [forward error correction](/reference/forward-error-correction/) protecting the
parameters — important because a corrupted parameter sounds far worse than a corrupted
audio sample.

## Relevance to SDR

GopherTrunk decodes IMBE frames from P25 Phase 1 and synthesises audio. Its successor,
[AMBE+2](/reference/ambe-plus-2/), is used by newer systems.
