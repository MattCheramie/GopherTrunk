---
slug: limesdr
title: LimeSDR
entry_type: hardware
category: sdr-devices
description: LimeSDR is a full-duplex transmit-and-receive software-defined radio built on the Lime Microsystems LMS7002M transceiver, covering 100 kHz–3.8 GHz with 12-bit sampling.
keywords: LimeSDR, LMS7002M, Lime Microsystems, full duplex SDR, MIMO transceiver, 100 kHz 3.8 GHz, 12-bit, SoapySDR, LimeSDR Mini
aka: [LimeSDR, LimeSDR Mini]
autolink: true
affiliate: true
product:
  name: "LimeSDR Mini"
  brand: Lime Microsystems
  category: Software-defined radio
  lowPrice: "160"
  highPrice: "200"
  url: https://www.amazon.com/s?k=LimeSDR+Mini&tag=gophertrunk-20
infobox:
  - { label: Type, value: Full-duplex SDR transceiver }
  - { label: Vendor/Chip, value: "Lime Microsystems, LMS7002M" }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: 100 kHz – 3.8 GHz }
  - { label: Bandwidth, value: up to ~61.44 MHz }
  - { label: TX, value: Yes (full-duplex) }
  - { label: With GopherTrunk, value: Network only (SoapySDR/rtl_tcp bridge) }
  - { label: Typical price, value: "~US$180 (Mini)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=LimeSDR+Mini&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [software-defined-radio, soapysdr, hackrf, mimo, gnuradio]
cite_urls:
  - https://en.wikipedia.org/wiki/LimeSDR
  - https://limesdr.org/
faq:
  - q: "Does GopherTrunk support the LimeSDR?"
    a: "Yes, but network only. GopherTrunk's pure-Go USB drivers cover RTL-SDR, HackRF, and Airspy — not Lime's LimeSuite stack. You drive the LimeSDR (or LimeSDR Mini) over the network via a SoapySDRServer/SoapyRemote (or rtl_tcp) bridge: run the SoapyLMS7 module and Soapy server on the machine it's plugged into, then mount it over TCP. See the hardware guide."
  - q: "LimeSDR or LimeSDR Mini for GopherTrunk?"
    a: "The Mini is the common, cheaper single-channel board and is plenty for feeding IQ to GopherTrunk. The full-size LimeSDR adds a second channel, ~61.44 MHz bandwidth, and full 2×2 MIMO — capabilities aimed at TX and lab work that receive-only scanning doesn't use."
  - q: "Can GopherTrunk transmit with a LimeSDR?"
    a: "No. GopherTrunk is a receive-only decoder and uses the LimeSDR purely as an IQ source over the bridge. Its transmit and MIMO features go unused, and transmitting would require appropriate licensing regardless."
  - q: "Is a LimeSDR worth it for scanning versus a directly-supported SDR?"
    a: "For pure trunk-tracking, a plug-and-play RTL-SDR or Airspy is cheaper, needs no SoapySDR bridge, and has GopherTrunk's native USB backend. The LimeSDR shines at two-way and development work; treat scanning as a secondary role."
---

**LimeSDR** is an open-source, **full-duplex** transmit-and-receive
[software-defined radio](/reference/software-defined-radio/) built around Lime
Microsystems' **LMS7002M** field-programmable RF transceiver, covering **100 kHz to
3.8 GHz** with **12-bit** [ADC](/reference/analog-to-digital-converter/) sampling.[^wiki]
Unlike a receive-only dongle, it can transmit and receive at the same time and offers two
independent channels, making it a [MIMO](/reference/mimo/)-capable platform.[^lime]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the LimeSDR from about 100 kilohertz to 3.8 gigahertz on an axis from 0 to 6 gigahertz, marked as a transmit-and-receive device." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="253" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">LimeSDR (100 kHz–3.8 GHz) TX + RX, 2×2 MIMO</text>
</svg>
<figcaption>LimeSDR transmits and receives full-duplex across HF through low-microwave, with two channels for MIMO.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=LimeSDR+Mini&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Full-duplex 12-bit transceiver for developers.** The LMS7002M-based LimeSDR transmits and
receives at once across 100 kHz–3.8 GHz with two channels for [MIMO](/reference/mimo/) — a
workhorse for cellular and protocol lab work more than a scanner. **GopherTrunk support is
network only:** it drives the LimeSDR over a
[SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge, not as a direct USB
device, because GopherTrunk's pure-Go USB drivers cover only RTL-SDR, HackRF, and Airspy.
For scanning, transmit and MIMO go unused, so a cheaper directly-supported SDR is usually
the better buy. **Mini around $180.** Like every receiver, it can't decode
[AES encryption](/police-scanner-encryption/).
</div>

> **GopherTrunk support: network only.** GopherTrunk drives the LimeSDR over the network via
> SoapySDRServer/SoapyRemote (or an `rtl_tcp`-style bridge), not as a direct USB device —
> you run the LimeSuite/`SoapyLMS7` module and a [SoapySDR](/reference/soapysdr/) server on
> the machine it's plugged into and mount it over TCP. GopherTrunk's pure-Go USB drivers
> cover only RTL-SDR, HackRF, and Airspy. See the [hardware guide](/hardware.html) and
> [rtl_tcp](/reference/rtl-tcp/) notes.

## Overview

The LimeSDR targets developers building and testing wireless systems rather than pure
listeners: cellular ([GSM](/reference/gsm/), [LTE](/reference/lte/)), IoT radios,
and general RF prototyping. Its wide instantaneous
[bandwidth](/reference/bandwidth/) — up to about 61.44 MHz on the full-size board — and
two-channel front end let it host software base stations and duplex links that a
receive-only SDR cannot. The heart of the board is the LMS7002M, a highly configurable
[zero-IF](/reference/zero-if/) transceiver whose gain, filtering, and mixing are all set in
software, with an on-board FPGA handling sample transport over USB 3.0.

## What it is

Each of the LMS7002M's two RX and two TX chains has its own tuner, so the board can run
2×2 MIMO or act as two loosely related radios. Samples move as [IQ](/reference/iq-data/)
over USB 3.0; the FPGA does packing and buffering while
[demodulation](/reference/demodulation/) and decoding stay on the host. Software support
centres on Lime's **LimeSuite** driver and a [SoapySDR](/reference/soapysdr/) module
(`SoapyLMS7`), which plugs the board into [GNU Radio](/reference/gnuradio/),
[SDRangel](/reference/sdrangel/), and the wider Soapy ecosystem. Because transmit is a
first-class function, LimeSDR is subject to the same regulatory care as any transmitter —
it can radiate, and licensing rules apply.

## Variants

The line spans several form factors around the same silicon:

- **LimeSDR (USB)** — the original full-size board: two RX and two TX channels, ~61.44 MHz
  bandwidth, USB 3.0.
- **LimeSDR Mini / Mini 2.0** — a smaller, lower-cost single-channel version with reduced
  bandwidth, the most common entry point.
- **LimeSDR-PCIe** and **LimeNET / LimeSDR-QPCIe** — PCIe and appliance variants aimed at
  fixed infrastructure and multi-radio deployments.

All share the LMS7002M transceiver family and the LimeSuite/SoapySDR software stack, so
code and flowgraphs port across them.

## Relevance to SDR

LimeSDR is a workhorse for two-way and lab work: running an experimental cellular cell,
prototyping a protocol end-to-end, or testing a receiver against a signal you generate
yourself. It overlaps the [HackRF](/reference/hackrf/) in the "wide-range transceiver"
niche but adds full-duplex operation, a second channel, and 12-bit sampling.

For trunking reception it is capable but not the natural pick — transmit and MIMO are
irrelevant to receive-only scanning, and its dynamic range in a single RX channel is
comparable to other 12-bit SDRs rather than exceptional. GopherTrunk provides native USB
backends for RTL-SDR, HackRF, and Airspy, not for the LimeSuite stack, so any LimeSDR use
would go through a SoapySDR bridge; treat it as a general-purpose transceiver whose scanning
role is secondary to its development strengths.

## Where to buy

LimeSDR boards are sold through Crowd Supply, Lime Microsystems' distributors, and Amazon;
the **LimeSDR Mini** is the common, lower-cost entry point. Before buying one for
GopherTrunk, note the integration path: it is supported **over the network only**, via a
[SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge rather than
GopherTrunk's direct USB drivers — see the [hardware guide](/hardware.html). For scanning
alone, a [RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) is cheaper and
plug-and-play; the LimeSDR pays off when you also want its transmit and development
capabilities.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=LimeSDR+Mini&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [LimeSDR](https://en.wikipedia.org/wiki/LimeSDR) — Wikipedia, on the LimeSDR platform, the LMS7002M transceiver, and its full-duplex MIMO capabilities.
[^lime]: [LimeSDR](https://limesdr.org/) — Lime Microsystems / Myriad-RF, official project site with tuning range, channel count, bandwidth, and software details.
