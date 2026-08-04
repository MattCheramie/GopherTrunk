---
slug: sdrplay-rsp1a
title: SDRplay RSP1A
entry_type: hardware
category: sdr-devices
description: SDRplay RSP1A is a 14-bit, 1 kHz–2 GHz receive-only software-defined radio with continuous coverage, an on-board preselector, and up to 10 MHz of visible bandwidth.
keywords: SDRplay RSP1A, RSP1A, MSi001, MSi2500, 14-bit SDR receiver, wideband receiver, 1 kHz 2 GHz, SoapySDR
aka: [RSP1A, SDRplay RSP1A]
autolink: true
affiliate: true
product:
  name: "SDRplay RSP1A"
  brand: SDRplay
  category: Software-defined radio
  lowPrice: "110"
  highPrice: "130"
  url: https://www.amazon.com/s?k=SDRplay+RSP1A&tag=gophertrunk-20
infobox:
  - { label: Type, value: Receive-only SDR }
  - { label: Vendor/Chip, value: "SDRplay, MSi001 + MSi2500" }
  - { label: ADC, value: 14-bit }
  - { label: Range, value: 1 kHz – 2 GHz }
  - { label: Bandwidth, value: up to ~10 MHz }
  - { label: TX, value: No }
  - { label: With GopherTrunk, value: Network only (SoapySDR/rtl_tcp bridge) }
  - { label: Typical price, value: ~US$120 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=SDRplay+RSP1A&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [software-defined-radio, soapysdr, msi001-tuner, rtl-sdr, airspy]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/rsp1a/
faq:
  - q: "Does GopherTrunk support the SDRplay RSP1A?"
    a: "Yes, but network only. GopherTrunk's pure-Go USB drivers cover RTL-SDR, HackRF, and Airspy — not the RSP's closed API/service. You drive the RSP1A over the network via a SoapySDRServer/SoapyRemote (or rtl_tcp) bridge: run SDRplay's service and the Soapy server on the machine the RSP1A is plugged into, then mount it over TCP. See the hardware guide."
  - q: "Is the RSP1A a good SDR for scanning trunked systems?"
    a: "As RF hardware, yes — its 14-bit ADC, continuous 1 kHz–2 GHz coverage, and front-end preselection give it far more dynamic range than an 8-bit RTL-SDR, and its ~10 MHz span can cover a system's control and voice channels in one capture. The only caveat is the integration path: it runs over a network bridge, not as a plug-and-play USB device."
  - q: "Why can't GopherTrunk open the RSP1A directly over USB?"
    a: "The RSP line is not libusb-generic. SDRplay ships a closed API/service that applications must talk to, so there is no pure-Go USB path the way there is for RTL-SDR, HackRF, and Airspy. The supported route is the vendor service plus a SoapySDR module, mounted over the network."
  - q: "RSP1A or an RTL-SDR / Airspy for GopherTrunk?"
    a: "If you want the simplest setup, a directly-supported RTL-SDR or Airspy plugs in and works with no bridge. Choose the RSP1A when you specifically want its 14-bit dynamic range and continuous HF-through-UHF coverage and don't mind running a SoapySDR server for it."
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

<a class="btn btn--buy" href="https://www.amazon.com/s?k=SDRplay+RSP1A&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**14-bit, continuous 1 kHz–2 GHz, receive-only.** The RSP1A gives you far more
[dynamic range](/reference/dynamic-range/) and real front-end
[preselection](/reference/rf-filter/) than an 8-bit [RTL-SDR](/reference/rtl-sdr/), with a
~10 MHz span that can capture a trunked system's control and voice channels at once.
**GopherTrunk support is network only:** it drives the RSP1A over a
[SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge, not as a direct USB
device — SDRplay's closed API/service rules out the pure-Go USB path used for RTL-SDR,
HackRF, and Airspy. **Around $120.** Like every receiver, it can't decode
[AES encryption](/police-scanner-encryption/).
</div>

> **GopherTrunk support: network only.** GopherTrunk drives the RSP1A over the network via
> SoapySDRServer/SoapyRemote (or an `rtl_tcp`-style bridge), not as a direct USB device —
> you run SDRplay's API service and a [SoapySDR](/reference/soapysdr/) server on the machine
> it's plugged into and mount it over TCP. GopherTrunk's pure-Go USB drivers cover only
> RTL-SDR, HackRF, and Airspy. See the [hardware guide](/hardware.html) and
> [rtl_tcp](/reference/rtl-tcp/) notes.

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

## Where to buy

The RSP1A is sold by SDRplay and its distributors, and listed on Amazon. Before you buy it
for GopherTrunk, remember the integration path: it is supported **over the network only**,
through a [SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge rather than
GopherTrunk's direct USB drivers — see the [hardware guide](/hardware.html). If you want a
plug-and-play device, a [RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) is
the simpler pick.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=SDRplay+RSP1A&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on receiver-class SDRs including the SDRplay RSP line.
[^sdrplay]: [RSP1A](https://www.sdrplay.com/rsp1a/) — SDRplay, official product page with tuning range, ADC resolution, filtering, and bias-tee specifications.
