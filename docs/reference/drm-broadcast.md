---
slug: drm-broadcast
title: Digital Radio Mondiale (DRM)
entry_type: protocol
category: broadcast
description: "DRM is the open digital broadcasting standard for the AM bands, using COFDM in HF/MW/LW channels to deliver near-FM audio in a shortwave slot."
keywords: DRM, Digital Radio Mondiale, DRM30, DRM+, COFDM, OFDM, HF broadcasting, shortwave digital, MW digital, xHE-AAC, ETSI ES 201 980
aka: [DRM, Digital Radio Mondiale, DRM30, DRM+]
autolink: true
infobox:
  - { label: Type, value: Digital broadcasting for AM/VHF bands }
  - { label: Standards body, value: "ETSI (DRM Consortium)" }
  - { label: Introduced, value: "2003" }
  - { label: Access, value: COFDM single-service or small multiplex }
  - { label: Channel spacing, value: "4.5 / 5 / 9 / 10 / 18 / 20 kHz (DRM30)" }
  - { label: Modulation, value: "COFDM with 4/16/64-QAM subcarriers" }
  - { label: Vocoder, value: "xHE-AAC (formerly HE-AAC / CELP / HVXC)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, shortwave-broadcast, amplitude-modulation, sky-wave, subcarrier, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_Radio_Mondiale
  - https://www.etsi.org/deliver/etsi_es/201900_201999/201980/
---

## Overview

**Digital Radio Mondiale** (**DRM**) is an open ETSI standard for digital broadcasting in
the same long-, medium-, and short-wave bands historically used by
[AM](/reference/amplitude-modulation/), designed to deliver clear, near-FM-quality audio
within a conventional ~10 kHz AM channel.[^wiki] It replaces the analog AM envelope with a
coded [OFDM](/reference/ofdm/) waveform (COFDM), so the same
[shortwave](/reference/shortwave-broadcast/) slot that once carried noisy AM can carry a
robust digital stream — plus text and station data — that either locks cleanly or drops
out, with little of AM's gradual fade. The **DRM30** profile covers the AM bands below
30 MHz; the later **DRM+** profile extends the system into VHF up to about 174 MHz.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DRM signal replacing an analog AM carrier and sidebands within the same ten-kilohertz channel by filling it with coded OFDM subcarriers carrying digital audio." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="drma" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="50" y1="40" x2="50" y2="95" stroke="currentColor" stroke-width="1.6"/>
  <path d="M20 95 Q50 80 80 95" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="50" y="112" text-anchor="middle" font-size="8" fill="currentColor">analog AM</text>
  <line x1="120" y1="65" x2="165" y2="65" stroke="currentColor" marker-end="url(#drma)"/>
  <text x="142" y="58" text-anchor="middle" font-size="7" fill="currentColor">DRM</text>
  <g stroke="currentColor" stroke-width="1">
    <line x1="185" y1="95" x2="185" y2="50"/><line x1="193" y1="95" x2="193" y2="55"/><line x1="201" y1="95" x2="201" y2="48"/><line x1="209" y1="95" x2="209" y2="58"/><line x1="217" y1="95" x2="217" y2="46"/><line x1="225" y1="95" x2="225" y2="52"/><line x1="233" y1="95" x2="233" y2="49"/><line x1="241" y1="95" x2="241" y2="57"/><line x1="249" y1="95" x2="249" y2="47"/><line x1="257" y1="95" x2="257" y2="54"/><line x1="265" y1="95" x2="265" y2="50"/><line x1="273" y1="95" x2="273" y2="56"/><line x1="281" y1="95" x2="281" y2="48"/><line x1="289" y1="95" x2="289" y2="53"/><line x1="297" y1="95" x2="297" y2="51"/><line x1="305" y1="95" x2="305" y2="55"/>
  </g>
  <line x1="180" y1="95" x2="312" y2="95" stroke="currentColor"/>
  <text x="246" y="112" text-anchor="middle" font-size="8" fill="currentColor">COFDM in the same channel</text>
</svg>
<figcaption>DRM fills a conventional AM channel with coded OFDM subcarriers, carrying digital audio where analog AM once sat.</figcaption>
</figure>

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | COFDM, four robustness modes (A–D) for worsening channels |
| Subcarrier modulation | 4-QAM, 16-QAM, or 64-QAM per service needs |
| Channel width | 4.5–20 kHz (DRM30); ~100 kHz for DRM+ in VHF |
| Bands | LW/MW/SW below 30 MHz (DRM30); VHF to ~174 MHz (DRM+) |
| Audio codec | xHE-AAC today (originally HE-AAC, CELP, HVXC) |
| Error control | Multi-level coding with interleaving |
| Data | Text messages, station labels, and a small data service |

## History

DRM was developed by the DRM Consortium — broadcasters, manufacturers, and research
bodies — and first standardised by ETSI as ES 201 980 in 2003, with regular international
transmissions beginning that year. DRM+ (Mode E) added VHF operation later in the decade,
and the audio codec was modernised to xHE-AAC to improve quality at the very low bit rates
the HF channel allows. Adoption has been gradual, concentrated in a handful of national
broadcasters and notably in India's medium-wave rollout.

## Deployment

DRM sees real but limited use, most prominently in India, with intermittent international
[shortwave](/reference/shortwave-broadcast/) services elsewhere. Its promise — full AM-band
coverage with digital clarity and lower transmitter power than analog AM — competes against
the reality of a huge analog receiver base and modest consumer hardware availability, so it
supplements rather than replaces AM on most of the globe.

## Decoding it with GopherTrunk

**GopherTrunk** does not decode DRM; it is a VHF/UHF trunked land-mobile scanner and DRM is
an HF/MW/VHF broadcast mode outside its scope. DRM is nonetheless a good illustration for GT
readers of coded [OFDM](/reference/ofdm/) surviving a hostile fading channel, and
open-source software (such as Dream) can decode a DRM broadcast from the baseband of an
HF-capable SDR.

## Sources

[^wiki]: [Digital Radio Mondiale](https://en.wikipedia.org/wiki/Digital_Radio_Mondiale) — Wikipedia, for the AM-band digital standard, COFDM waveform with QAM subcarriers, the DRM30/DRM+ profiles and channel widths, and the xHE-AAC audio codec.
