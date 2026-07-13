---
slug: msi001-tuner
title: MSi001 tuner
entry_type: hardware
category: sdr-devices
description: "MSi001 is Mirics' wideband RF tuner chip that, paired with the MSi2500 ADC, forms the front end of SDRplay RSP receivers."
keywords: MSi001, Mirics MSi001, MSi2500, SDRplay tuner, RSP1A tuner, wideband tuner chip, low-IF tuner, zero-IF, RF front end
aka: [MSi001, Mirics MSi001]
autolink: true
infobox:
  - { label: Type, value: RF tuner chip }
  - { label: Vendor, value: Mirics }
  - { label: Pairs with, value: MSi2500 (ADC + USB) }
  - { label: Architecture, value: Zero-IF / Low-IF }
  - { label: Range, value: "~1 kHz – 2 GHz" }
  - { label: Used in, value: SDRplay RSP receivers }
  - { label: TX, value: No (receive tuner) }
see_also: [sdrplay-rsp1a, sdrplay-rspduo, sdrplay-rspdx, analog-to-digital-converter, low-if, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/
---

**MSi001** is a wideband **RF tuner chip** from the British fabless company **Mirics**,
and the front-end half of the two-chip design at the heart of the
[SDRplay RSP](/reference/sdrplay-rsp1a/) family of
[software-defined radio](/reference/software-defined-radio/) receivers.[^wiki] Where a
budget dongle uses a Rafael Micro or Elonics tuner ahead of an RTL2832U, an SDRplay RSP
pairs the MSi001 tuner with the Mirics **MSi2500**, which combines the
[analog-to-digital converter](/reference/analog-to-digital-converter/) and USB interface.
Together the two chips give the RSP line its wide, continuous coverage and better dynamic
range than an 8-bit dongle.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Block diagram: antenna into the MSi001 tuner, into the MSi2500 which holds the analog-to-digital converter and USB interface, out to a computer as IQ samples." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="msar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="32" y="52">antenna</text>
    <rect x="78" y="34" width="96" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="126" y="48">MSi001</text><text x="126" y="60" font-size="7">tuner</text>
    <rect x="206" y="34" width="120" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="266" y="48">MSi2500</text><text x="266" y="60" font-size="7">ADC + USB</text>
    <text x="398" y="52">computer (IQ)</text>
  </g>
  <g stroke="currentColor" stroke-opacity="0.6" fill="none">
    <line x1="58" y1="47" x2="76" y2="47" marker-end="url(#msar)"/>
    <line x1="174" y1="47" x2="204" y2="47" marker-end="url(#msar)"/>
    <line x1="326" y1="47" x2="360" y2="47" marker-end="url(#msar)"/>
  </g>
</svg>
<figcaption>The MSi001 tuner and the MSi2500 ADC/USB chip together form the front end of every SDRplay RSP receiver.</figcaption>
</figure>

## Overview

The MSi001 is a **multi-standard, multi-band tuner** originally designed for consumer
digital-TV and radio reception (DVB-T, DAB, FM). Its appeal for SDR is **continuous
coverage** from roughly **1 kHz up to about 2 GHz** in one chip — no gap and no separate
upconverter — spanning LF, HF, VHF, and most of the UHF range in a single tuning path. It
supports both **zero-IF** and **[low-IF](/reference/low-if/)** modes, letting the
downstream software choose the architecture that best rejects images and DC artefacts for
a given band and bandwidth.

## How the pairing works

On its own the MSi001 only downconverts and filters; it needs a companion to digitise the
result. That companion is the **MSi2500**, which contains the RSP's
[ADC](/reference/analog-to-digital-converter/) and the USB controller that ships
[IQ](/reference/iq-data/) samples to the host. This split is deliberate: the tuner handles
the analog RF and the MSi2500 handles conversion and transport, and SDRplay's own API
manages the two together as one receiver. Compared with the 8-bit RTL2832U path, the
MSi2500's higher-resolution converter gives the RSP line noticeably more dynamic range,
which is why SDRplay receivers cope better with strong nearby signals and offer selectable
capture bandwidths up to several MHz.

## In practice

The MSi001 is not sold as a loose hobbyist part; you encounter it *inside* an SDRplay RSP.
It is the common thread across the family — the single-tuner
[RSP1A](/reference/sdrplay-rsp1a/), the dual-tuner
[RSPduo](/reference/sdrplay-rspduo/) (two MSi001 front ends for diversity or dual
reception), and the higher-end [RSPdx](/reference/sdrplay-rspdx/) with extra filtering and
a wider front end. All are **receive-only**. Because Mirics keeps the tuner's programming
behind the SDRplay API, these receivers are typically used through SDRplay's driver rather
than a generic librtlsdr-style interface.

## Relevance to GopherTrunk

GopherTrunk targets the [RTL-SDR](/reference/rtl-sdr/) and Airspy paths as its tested
front ends, and SDRplay's MSi001-based RSP receivers use a **different driver stack**
(the SDRplay API, typically bridged through SoapySDR) rather than the raw-IQ librtlsdr
interface GopherTrunk speaks natively. Where a SoapySDR bridge is available, an RSP's
wide continuous coverage and better dynamic range make it a capable IQ source for
channelising trunked control channels, but it is not a drop-in the way an RTL-SDR is. The
MSi001 matters here mainly as the silicon that explains *why* an SDRplay behaves
differently from a dongle — a higher-resolution, gap-free front end. See the
[hardware guide](/hardware.html) for GopherTrunk's supported devices.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on tuner-plus-ADC SDR front ends such as the Mirics chipset.
[^sdrplay]: [SDRplay](https://www.sdrplay.com/) — the vendor, for RSP receiver specifications built on the MSi001 and MSi2500.
