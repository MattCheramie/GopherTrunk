---
slug: sdrplay-rsp1a
title: SDRplay RSP1A
entry_type: hardware
category: sdr-devices
description: SDRplay RSP1A is a 14-bit, 1 kHz–2 GHz receive-only software-defined radio with continuous coverage, an on-board preselector, and up to 10 MHz of visible bandwidth.
keywords: SDRplay RSP1A, RSP1A, MSi001, MSi2500, 14-bit SDR receiver, wideband receiver, 1 kHz 2 GHz, SoapySDR
aka: [RSP1A, SDRplay RSP1A]
autolink: true
infobox:
  - { label: Type, value: Receive-only SDR }
  - { label: Vendor/Chip, value: "SDRplay, MSi001 + MSi2500" }
  - { label: ADC, value: 14-bit }
  - { label: Range, value: 1 kHz – 2 GHz }
  - { label: Bandwidth, value: up to ~10 MHz }
  - { label: TX, value: No }
  - { label: Typical price, value: ~US$120 }
see_also: [software-defined-radio, soapysdr, msi001-tuner, rtl-sdr, airspy]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/rsp1a/
---

**SDRplay RSP1A** is a low-cost, receive-only
[software-defined radio](/reference/software-defined-radio/) that tunes continuously from
**1 kHz to 2 GHz** with a **14-bit** [ADC](/reference/analog-to-digital-converter/) and up to
about 10 MHz of visible [bandwidth](/reference/bandwidth/).[^wiki] Built around the Mirics
[MSi001](/reference/msi001-tuner/) tuner and MSi2500 sampling chip, it offers markedly more
dynamic range and preselection than an [RTL-SDR](/reference/rtl-sdr/) while staying in a
similar price class.[^sdrplay]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the SDRplay RSP1A spanning roughly 1 kilohertz to 2 gigahertz on an axis from 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="133" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">RSP1A (1 kHz–2 GHz) continuous coverage</text>
</svg>
<figcaption>The RSP1A covers everything from VLF through UHF with one antenna port and a 14-bit converter.</figcaption>
</figure>

## Overview

The RSP1A is SDRplay's entry-level model and the reference design the rest of the RSP line
builds on. Its appeal for scanning and general monitoring is the combination of an
uninterrupted tuning range — no LF/HF/VHF gap — with real front-end filtering: a bank of
switchable [preselection](/reference/rf-filter/) filters, an MW/broadcast-FM notch, and a
software-controlled [bias tee](/reference/bias-tee/) for powering an active antenna or
[LNA](/reference/low-noise-amplifier/). Sampling at 14 bits gives it far better
[dynamic range](/reference/dynamic-range/) and resistance to
[intermodulation](/reference/intermodulation/) than the 8-bit RTL-SDR, so strong nearby
signals are less likely to desensitise the receiver or spray spurs across the band.

## What it is

Internally the RSP1A is a Mirics chipset: the MSi001 is a broadband
[zero-IF](/reference/zero-if/) tuner, and the MSi2500 handles the ADC and USB interface.
The device streams raw [IQ](/reference/iq-data/) samples over USB 2.0; all channelisation,
[demodulation](/reference/demodulation/), and decoding happen on the host. The maximum
displayed spectrum span is around 10 MHz, though the ADC actually samples faster and
decimates internally. Unlike the RTL-SDR, the RSP family is **not** libusb-generic —
SDRplay ships a closed **API/service** (the RSP driver daemon) that applications talk to,
and a [SoapySDR](/reference/soapysdr/) module (`SoapySDRPlay`) that exposes the RSP through
the common Soapy interface used by [GNU Radio](/reference/gnuradio/),
[SDRangel](/reference/sdrangel/), [CubicSDR](/reference/cubicsdr/), and others.

## Variants

The RSP1A sits at the base of a family that trades up in front-end sophistication:

- **RSP1 / RSP1B** — the original RSP1 was simpler (fewer filters, no bias tee); the RSP1B
  refreshes the RSP1A with improved LF/MW performance and remains single-tuner.
- **[RSPdx](/reference/sdrplay-rspdx/)** — adds more preselection filters, three antenna
  ports, and a high-dynamic-range (HDR) mode optimised for LF/MW/HF.
- **[RSPduo](/reference/sdrplay-rspduo/)** — a dual-tuner design that can run two 2 MHz
  streams at once for diversity or simultaneous monitoring of separated bands.

All share the same underlying Mirics silicon and the same SDRplay API, so software support
carries across the range.

## Relevance to SDR

The RSP1A is a popular choice for HF through UHF listening: shortwave and amateur bands,
[ADS-B](/reference/ads-b/) at 1090 MHz, VHF/UHF land-mobile, and broadcast monitoring. Its
14-bit converter and preselector make it a step up for crowded RF environments where an
RTL-SDR would overload.

For trunking specifically, the ~10 MHz span lets it cover a system's control and voice
channels from a single capture, much like an [Airspy](/reference/airspy/). The practical
caveat is the driver model: GopherTrunk drives RTL-SDR, HackRF, and Airspy hardware
directly, whereas the RSP line depends on SDRplay's proprietary API/service. Using an
RSP1A with GopherTrunk therefore hinges on a SoapySDR/RSP bridge rather than a native
backend — check the project status before assuming turn-key support. As a receiver the
hardware is well suited to the task; the integration path, not the RF, is the limiting
factor.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on receiver-class SDRs including the SDRplay RSP line.
[^sdrplay]: [RSP1A](https://www.sdrplay.com/rsp1a/) — SDRplay, official product page with tuning range, ADC resolution, filtering, and bias-tee specifications.
