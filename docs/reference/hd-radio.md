---
slug: hd-radio
title: HD Radio (IBOC)
entry_type: protocol
category: broadcast
description: "HD Radio is the North American in-band on-channel digital radio system that adds OFDM sidebands around an existing FM or AM analog broadcast."
keywords: HD Radio, IBOC, in-band on-channel, NRSC-5, iBiquity, Xperi, OFDM sidebands, hybrid FM, digital sidebands, HDC codec, secondary program service
aka: [HD Radio, IBOC, in-band on-channel, NRSC-5]
autolink: true
infobox:
  - { label: Type, value: In-band on-channel digital radio }
  - { label: Standards body, value: "NRSC (NRSC-5); Xperi (proprietary)" }
  - { label: Introduced, value: "2002" }
  - { label: Access, value: OFDM sidebands beside the analog host }
  - { label: Channel spacing, value: "Within the host 200 kHz FM / 20 kHz AM mask" }
  - { label: Modulation, value: "OFDM, QPSK-mapped digital sidebands" }
  - { label: Vocoder, value: "HDC (HE-AAC-derived, proprietary)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, broadcast-fm, broadcast-am, subcarrier, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/HD_Radio
  - https://en.wikipedia.org/wiki/In-band_on-channel
---

## Overview

**HD Radio** is the trademark name for the North American **in-band on-channel** (**IBOC**)
digital radio system, standardised as NRSC-5 and licensed by Xperi (formerly
iBiquity).[^wiki] Rather than moving digital radio to a new band as
[DAB](/reference/dab/) does, IBOC squeezes [OFDM](/reference/ofdm/) digital
[sidebands](/reference/subcarrier/) into the spectrum immediately around an existing
analog [FM](/reference/broadcast-fm/) or [AM](/reference/broadcast-am/) station, so one
transmitter radiates both the legacy analog signal and a digital version on the same
assigned channel. Receivers blend seamlessly from analog to digital as the digital signal
locks, and can offer additional program streams (HD2, HD3) tucked into the same channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An FM analog carrier in the centre of a channel flanked by two blocks of OFDM digital sidebands, showing the hybrid in-band on-channel arrangement." xmlns="http://www.w3.org/2000/svg">
  <line x1="230" y1="40" x2="230" y2="100" stroke="currentColor" stroke-width="1.8"/>
  <path d="M195 100 Q230 78 265 100" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="230" y="118" text-anchor="middle" font-size="8" fill="currentColor">analog host</text>
  <g stroke="currentColor" stroke-width="1">
    <line x1="70" y1="100" x2="70" y2="70"/><line x1="80" y1="100" x2="80" y2="66"/><line x1="90" y1="100" x2="90" y2="72"/><line x1="100" y1="100" x2="100" y2="64"/><line x1="110" y1="100" x2="110" y2="70"/><line x1="120" y1="100" x2="120" y2="68"/><line x1="130" y1="100" x2="130" y2="72"/><line x1="140" y1="100" x2="140" y2="66"/>
    <line x1="320" y1="100" x2="320" y2="66"/><line x1="330" y1="100" x2="330" y2="72"/><line x1="340" y1="100" x2="340" y2="64"/><line x1="350" y1="100" x2="350" y2="70"/><line x1="360" y1="100" x2="360" y2="68"/><line x1="370" y1="100" x2="370" y2="72"/><line x1="380" y1="100" x2="380" y2="66"/><line x1="390" y1="100" x2="390" y2="70"/>
  </g>
  <line x1="55" y1="100" x2="405" y2="100" stroke="currentColor"/>
  <text x="105" y="118" text-anchor="middle" font-size="8" fill="currentColor">digital OFDM</text>
  <text x="355" y="118" text-anchor="middle" font-size="8" fill="currentColor">digital OFDM</text>
</svg>
<figcaption>Hybrid IBOC: OFDM digital sidebands flank the analog host carrier inside the station's existing channel.</figcaption>
</figure>

## Technical characteristics

| Property | Value |
|----------|-------|
| Arrangement | In-band on-channel; hybrid (analog + digital) or all-digital |
| Waveform | OFDM sidebands, QPSK-mapped subcarriers |
| FM sidebands | Digital energy outside the analog MPX, within the 200 kHz mask |
| AM mode | Narrow OFDM sidebands within the ~20 kHz AM channel |
| Audio codec | HDC, a proprietary HE-AAC derivative |
| Extra services | Secondary programs (HD2/HD3), text, and album art |
| Blend | Automatic analog↔digital transition on the FM host |

## History

iBiquity Digital developed IBOC from earlier competing proposals, and the FCC authorised
hybrid daytime operation in 2002; the NRSC formalised it as the NRSC-5 standard.
All-digital AM (MA3) and denser FM configurations were added over the following years.
iBiquity was acquired by DTS and then Xperi, which continues to license the HD Radio brand
and the proprietary HDC codec that keeps the system closed despite the public NRSC-5 layer
descriptions.

## Deployment

HD Radio is deployed primarily in the United States and, to a lesser degree, Mexico and
Canada, where a large fraction of FM stations and many AM stations run hybrid IBOC. It is
the North American answer to digital radio, chosen over [DAB](/reference/dab/) precisely
because it needs no new spectrum and preserves the analog service. AM HD Radio has seen
limited uptake because its digital sidebands can raise adjacent-channel interference on the
crowded medium-wave band.

## Decoding it with GopherTrunk

**GopherTrunk** does not decode HD Radio; it is a trunked land-mobile scanner (P25, DMR,
NXDN, TETRA and similar) and IBOC is a broadcast system outside its scope. The proprietary
HDC codec also means even general-purpose SDR tools can demodulate the OFDM layer (e.g. the
open-source nrsc5 project) but rely on reverse-engineered audio decoding. For GT readers,
HD Radio is a useful example of layering an [OFDM](/reference/ofdm/) digital signal
alongside an existing analog carrier without a new channel assignment.

## Sources

[^wiki]: [HD Radio](https://en.wikipedia.org/wiki/HD_Radio) — Wikipedia, for the in-band on-channel concept, NRSC-5 standardisation and iBiquity/Xperi licensing, the OFDM digital sidebands around FM/AM hosts, and the proprietary HDC codec.
