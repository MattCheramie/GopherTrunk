---
slug: software-defined-radio
title: Software-defined radio (SDR)
entry_type: technology
category: sdr-dsp
description: Software-defined radio (SDR) is radio technology in which traditionally hardware functions — tuning, filtering, demodulation — are implemented in software operating on digitised IQ samples.
keywords: software defined radio, SDR, IQ, digital radio, RTL-SDR, flexibility
aka: [software-defined radio, SDR]
autolink: true
infobox:
  - { label: Type, value: Radio architecture }
  - { label: Idea, value: Move tuning/demod into software }
  - { label: Hardware emits, value: IQ samples }
  - { label: Examples, value: RTL-SDR, HackRF, Airspy }
see_also: [iq-data, analog-to-digital-converter, superheterodyne-receiver, rtl-sdr, demodulation]
related_lessons:
  - { title: "What is software-defined radio?", url: /learn/what-is-sdr/ }
external:
  - { title: "Software-defined radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio }
---

**Software-defined radio** (**SDR**) moves the functions that were once fixed hardware —
tuning, filtering, [demodulation](/reference/demodulation/) — into **software** operating
on digitised [IQ samples](/reference/iq-data/). The hardware does only enough to convert a
slice of spectrum into numbers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 110" role="img" aria-label="Antenna into SDR hardware which outputs IQ samples into software which outputs audio and data." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <path d="M40 78 v-28 m-9 0 l9 -12 l9 12" fill="none" stroke="currentColor" stroke-width="2"/><text x="40" y="96">antenna</text>
    <rect x="86" y="44" width="104" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="138" y="58">SDR hardware</text><text x="138" y="71" font-size="8.5">tune · digitise</text>
    <rect x="262" y="44" width="116" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="320" y="58">software</text><text x="320" y="71" font-size="8.5">filter · demod · decode</text>
    <rect x="450" y="44" width="78" height="34" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="489" y="64">audio &amp; data</text>
    <text x="226" y="38" font-size="8.5">IQ samples</text>
    <g stroke="currentColor" stroke-width="1.2"><line x1="52" y1="61" x2="85" y2="61"/><line x1="190" y1="61" x2="261" y2="61"/><line x1="378" y1="61" x2="449" y2="61"/></g>
  </g>
</svg>
<figcaption>An SDR moves tuning, filtering, and demodulation into software; the hardware just delivers IQ samples.</figcaption>
</figure>

## How it works

An SDR front-end amplifies, mixes, and digitises a band into IQ; software then does
everything else. Because the differences between systems live in code, one device can
decode many protocols.

## Relevance to SDR

GopherTrunk is the software half of an SDR, specialised for digital trunked radio. The
hardware (e.g. [RTL-SDR](/reference/rtl-sdr/)) is almost interchangeable.
