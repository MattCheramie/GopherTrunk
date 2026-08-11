---
slug: sdrplay-rsp1b
title: SDRplay RSP1B
entry_type: hardware
category: sdr-devices
description: "SDRplay RSP1B is a 14-bit, 1 kHz–2 GHz receive-only software-defined radio — a refresh of the RSP1A with improved LF/MW/HF performance, a bias tee, and up to 10 MHz of visible bandwidth."
keywords: SDRplay RSP1B, RSP1B, 14-bit SDR receiver, wideband receiver, 1 kHz 2 GHz, MSi001, SoapySDR, RSP1A successor, SDRplay
aka: [RSP1B, SDRplay RSP1B]
autolink: true
affiliate: true
product:
  name: "SDRplay RSP1B"
  brand: SDRplay
  category: Software-defined radio
  lowPrice: "120"
  highPrice: "150"
  url: https://www.amazon.com/s?k=SDRplay+RSP1B&tag=gophertrunk-20
infobox:
  - { label: Type, value: Receive-only SDR }
  - { label: Vendor/Chip, value: "SDRplay, MSi001 + MSi2500" }
  - { label: ADC, value: 14-bit }
  - { label: Range, value: 1 kHz – 2 GHz }
  - { label: Bandwidth, value: up to ~10 MHz }
  - { label: TX, value: No }
  - { label: With GopherTrunk, value: Network only (SoapySDR/rtl_tcp bridge) }
  - { label: Typical price, value: ~US$130 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=SDRplay+RSP1B&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [sdrplay-rsp1a, sdrplay-rspdx, sdrplay-rspduo, software-defined-radio, soapysdr, msi001-tuner, rtl-sdr, airspy]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/rsp1b/
faq:
  - q: "Does GopherTrunk support the SDRplay RSP1B?"
    a: "Yes, but network only — exactly like the RSP1A. GopherTrunk's pure-Go USB drivers cover RTL-SDR, HackRF, and Airspy, not the RSP's closed API/service. You run SDRplay's service and a SoapySDR server (or an rtl_tcp-style bridge) on the machine the RSP1B is plugged into, then mount it over TCP. See the hardware guide."
  - q: "What's the difference between the RSP1B and the RSP1A?"
    a: "They're closely related single-tuner 14-bit receivers. The RSP1B refreshes the RSP1A with improved LF/MW/HF performance (better handling of strong broadcast signals in those bands) and stays otherwise similar — same 1 kHz–2 GHz range, same ~10 MHz span, same SDRplay API. Either is a fine 14-bit step up from an RTL-SDR."
  - q: "Is the RSP1B a good SDR for scanning trunked systems?"
    a: "As RF hardware, yes — its 14-bit ADC, continuous 1 kHz–2 GHz coverage, and front-end preselection give it far more dynamic range than an 8-bit RTL-SDR, and its ~10 MHz span can cover a system's control and voice channels in one capture. The caveat is the integration path: it runs over a network bridge, not as a plug-and-play USB device."
  - q: "RSP1B or an RTL-SDR / Airspy for GopherTrunk?"
    a: "If you want the simplest setup, a directly-supported RTL-SDR or Airspy plugs in and works with no bridge. Choose the RSP1B when you specifically want its 14-bit dynamic range and continuous HF-through-UHF coverage and don't mind running a SoapySDR server for it. And no SDR of any resolution decodes AES encryption."
---

**SDRplay RSP1B** is a low-cost, receive-only
[software-defined radio](/reference/software-defined-radio/) that tunes continuously from
**1 kHz to 2 GHz** with a **14-bit** [ADC](/reference/analog-to-digital-converter/) and up to
about 10 MHz of visible [bandwidth](/reference/bandwidth/).[^wiki] It is SDRplay's refresh of
the popular [RSP1A](/reference/sdrplay-rsp1a/), adding improved LF/MW/HF performance while
keeping the same Mirics [MSi001](/reference/msi001-tuner/) tuner and MSi2500 sampling chip —
so it offers markedly more dynamic range and preselection than an
[RTL-SDR](/reference/rtl-sdr/) in a similar price class.[^sdrplay]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the SDRplay RSP1B spanning roughly 1 kilohertz to 2 gigahertz on an axis from 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="133" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">RSP1B (1 kHz–2 GHz) continuous coverage</text>
</svg>
<figcaption>The RSP1B covers everything from VLF through UHF with one antenna port and a 14-bit converter.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=SDRplay+RSP1B&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**14-bit, continuous 1 kHz–2 GHz, receive-only — the [RSP1A](/reference/sdrplay-rsp1a/)
refresh.** The RSP1B gives you far more [dynamic range](/reference/dynamic-range/) and real
front-end [preselection](/reference/rf-filter/) than an 8-bit
[RTL-SDR](/reference/rtl-sdr/), with a ~10 MHz span that can capture a trunked system's
control and voice channels at once, plus improved LF/MW/HF handling.
**GopherTrunk support is network only:** it drives the RSP1B over a
[SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge, not as a direct USB
device — SDRplay's closed API/service rules out the pure-Go USB path used for RTL-SDR,
HackRF, and Airspy. **Around $130.** Like every receiver, it can't decode
[AES encryption](/police-scanner-encryption/).
</div>

> **GopherTrunk support: network only.** GopherTrunk drives the RSP1B over the network via
> SoapySDRServer/SoapyRemote (or an `rtl_tcp`-style bridge), not as a direct USB device —
> you run SDRplay's API service and a [SoapySDR](/reference/soapysdr/) server on the machine
> it's plugged into and mount it over TCP. GopherTrunk's pure-Go USB drivers cover only
> RTL-SDR, HackRF, and Airspy. See the [hardware guide](/hardware.html) and
> [rtl_tcp](/reference/rtl-tcp/) notes.

## Overview

The RSP1B is SDRplay's current entry-level model, replacing the
[RSP1A](/reference/sdrplay-rsp1a/) at the base of the RSP line. Its appeal for scanning and
general monitoring is the same combination that made the RSP1A popular: an uninterrupted
tuning range — no LF/HF/VHF gap — with real front-end filtering, a bank of switchable
[preselection](/reference/rf-filter/) filters, an MW/broadcast-FM notch, and a
software-controlled [bias tee](/reference/bias-tee/) for powering an active antenna or
[LNA](/reference/low-noise-amplifier/). Sampling at 14 bits gives it far better
[dynamic range](/reference/dynamic-range/) and resistance to
[intermodulation](/reference/intermodulation/) than the 8-bit RTL-SDR, and the RSP1B's
refreshed design tightens up performance at the low end of the spectrum specifically.

## What it is

Internally the RSP1B is a Mirics chipset: the MSi001 is a broadband
[zero-IF](/reference/zero-if/) tuner, and the MSi2500 handles the ADC and USB interface. The
device streams raw [IQ](/reference/iq-data/) samples over USB 2.0; all channelisation,
[demodulation](/reference/demodulation/), and decoding happen on the host. The maximum
displayed spectrum span is around 10 MHz, though the ADC actually samples faster and
decimates internally. Like the rest of the RSP family it is **not** libusb-generic — SDRplay
ships a closed **API/service** that applications talk to, and a
[SoapySDR](/reference/soapysdr/) module (`SoapySDRPlay`) that exposes the RSP through the
common Soapy interface used by [GNU Radio](/reference/gnuradio/),
[SDRangel](/reference/sdrangel/), [CubicSDR](/reference/cubicsdr/), and others.

## Variants

The RSP1B sits at the base of a family that trades up in front-end sophistication:

- **[RSP1A](/reference/sdrplay-rsp1a/)** — the model the RSP1B refreshes; still widely sold
  and functionally very close.
- **[RSPdx](/reference/sdrplay-rspdx/)** — adds more preselection filters, three antenna
  ports, and a high-dynamic-range (HDR) mode optimised for LF/MW/HF.
- **[RSPduo](/reference/sdrplay-rspduo/)** — a dual-tuner design that can run two 2 MHz
  streams at once for diversity or simultaneous monitoring of separated bands.

All share the same underlying Mirics silicon and the same SDRplay API, so software support
carries across the range.

## Relevance to SDR

The RSP1B is a strong general-purpose receiver for HF through UHF: shortwave and amateur
bands, [ADS-B](/reference/ads-b/) at 1090 MHz, VHF/UHF land-mobile, and broadcast monitoring.
Its 14-bit converter and preselector make it a real step up for crowded RF environments where
an RTL-SDR would overload.

For trunking specifically, the ~10 MHz span lets it cover a system's control and voice
channels from a single capture, much like an [Airspy](/reference/airspy/). The practical
caveat is the driver model: GopherTrunk drives RTL-SDR, HackRF, and Airspy hardware directly,
whereas the RSP line depends on SDRplay's proprietary API/service. Using an RSP1B with
GopherTrunk therefore hinges on a SoapySDR/RSP bridge rather than a native backend — check the
project status before assuming turn-key support. As a receiver the hardware is well suited to
the task; the integration path, not the RF, is the limiting factor.

## Where to buy

The RSP1B is sold by SDRplay and its distributors, and listed on Amazon — though stock is
often third-party, so the button is a tagged search that resolves to current listings.
Before you buy it for GopherTrunk, remember the integration path: it is supported **over the
network only**, through a [SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/)
bridge rather than GopherTrunk's direct USB drivers — see the
[hardware guide](/hardware.html). If you want a plug-and-play device, a
[RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) is the simpler pick; if you
already run an [RSP1A](/reference/sdrplay-rsp1a/), the RSP1B is a marginal upgrade rather than
a must-have.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=SDRplay+RSP1B&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on receiver-class SDRs including the SDRplay RSP line.
[^sdrplay]: [RSP1B](https://www.sdrplay.com/rsp1b/) — SDRplay, official product page with tuning range, 14-bit ADC, filtering, bias-tee, and the RSP1A-to-RSP1B improvements.
