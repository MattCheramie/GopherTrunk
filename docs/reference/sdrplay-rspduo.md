---
slug: sdrplay-rspduo
title: SDRplay RSPduo
entry_type: hardware
category: sdr-devices
description: SDRplay RSPduo is a 14-bit, 1 kHz–2 GHz dual-tuner receive-only SDR that can run two independent 2 MHz streams for antenna diversity or simultaneous multi-band monitoring.
keywords: SDRplay RSPduo, RSPduo, dual tuner SDR, diversity reception, coherent receiver, 14-bit receiver, two independent streams, SoapySDR
aka: [RSPduo, SDRplay RSPduo]
autolink: true
affiliate: true
product:
  name: "SDRplay RSPduo"
  brand: SDRplay
  category: Software-defined radio
  lowPrice: "260"
  highPrice: "300"
  url: https://www.amazon.com/s?k=SDRplay+RSPduo&tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-tuner receive-only SDR }
  - { label: Vendor/Chip, value: "SDRplay, dual Mirics tuners" }
  - { label: ADC, value: 14-bit (per tuner) }
  - { label: Range, value: 1 kHz – 2 GHz }
  - { label: Bandwidth, value: 2 × ~2 MHz (dual) }
  - { label: TX, value: No }
  - { label: With GopherTrunk, value: Network only (SoapySDR/rtl_tcp bridge) }
  - { label: Typical price, value: ~US$280 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=SDRplay+RSPduo&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [sdrplay-rsp1a, sdrplay-rsp1b, sdrplay-rspdx, antenna-diversity, software-defined-radio, soapysdr, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/rspduo/
faq:
  - q: "Does GopherTrunk support the SDRplay RSPduo?"
    a: "Yes, but network only — like the rest of the RSP line. GopherTrunk's pure-Go USB drivers cover RTL-SDR, HackRF, and Airspy, not SDRplay's closed API/service. You run SDRplay's service and a SoapySDR server (or an rtl_tcp bridge) on the machine the RSPduo is plugged into and mount it over TCP. See the hardware guide."
  - q: "What can the RSPduo's two tuners do for scanning?"
    a: "Two independent ~2 MHz receivers at once: one could sit on a control channel while the other tracks a separated band, or the pair can run coherently for antenna diversity against multipath and fading. The catch is the ~2 MHz per-tuner cap in dual mode — narrower than an Airspy's single wide capture — plus the SoapySDR bridge requirement."
  - q: "RSPduo, RSP1B, or an Airspy for GopherTrunk?"
    a: "For plug-and-play trunking, a directly-supported RTL-SDR or Airspy is simpler. Choose an RSP1B if you want a single 14-bit receiver over a bridge; choose the RSPduo only if you specifically need two simultaneous tuners or diversity and don't mind running a SoapySDR server."
  - q: "Does the RSPduo decode encrypted channels?"
    a: "No. Two tuners don't change the encryption wall — like every SDR, the RSPduo cannot decode AES-encrypted traffic. It only receives clear signals for the host to demodulate."
---

**SDRplay RSPduo** is a **dual-tuner** receive-only
[software-defined radio](/reference/software-defined-radio/) covering **1 kHz to 2 GHz**
with **14-bit** sampling on each tuner.[^wiki] Its distinguishing feature is two
independent receiver chains that can operate at once — as two separate ~2 MHz streams on
different frequencies, or as a coherent pair for
[antenna diversity](/reference/antenna-diversity/).[^sdrplay]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Block diagram of the SDRplay RSPduo showing two antenna inputs each feeding its own 14-bit tuner, both streamed over one USB connection to the host." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="duoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="40" y="35">Ant A</text><text x="40" y="95">Ant B</text>
    <rect x="90" y="18" width="90" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="135" y="35">Tuner 1 (14-bit)</text>
    <rect x="90" y="82" width="90" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="135" y="99">Tuner 2 (14-bit)</text>
    <rect x="250" y="50" width="80" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="290" y="67">USB</text>
    <rect x="380" y="50" width="60" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="410" y="67">Host</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="55" y1="31" x2="88" y2="31" marker-end="url(#duoar)"/>
    <line x1="55" y1="95" x2="88" y2="95" marker-end="url(#duoar)"/>
    <line x1="180" y1="31" x2="248" y2="58" marker-end="url(#duoar)"/>
    <line x1="180" y1="95" x2="248" y2="68" marker-end="url(#duoar)"/>
    <line x1="330" y1="63" x2="378" y2="63" marker-end="url(#duoar)"/>
  </g>
</svg>
<figcaption>Two independent 14-bit tuners share one USB link — usable as two receivers or as a coherent diversity pair.</figcaption>
</figure>

## Overview

The RSPduo effectively packs two RSP-class receivers into one box, sharing a single USB
connection and the SDRplay API. That opens uses no single-tuner SDR can manage: watch two
widely separated bands simultaneously — say an HF net and a VHF
[control channel](/reference/control-channel/) — or point two antennas at the same signal
and combine them. It retains the family's continuous 1 kHz–2 GHz tuning and 14-bit
converters, but each tuner's instantaneous [bandwidth](/reference/bandwidth/) is capped
around 2 MHz when both run together (a single tuner alone can go wider).

## What it is

Each chain is a Mirics tuner and ADC; both stream raw [IQ](/reference/iq-data/) to the host
over USB, where all [demodulation](/reference/demodulation/) and decoding happen. Two modes
matter:

- **Dual independent** — the tuners sit on different centre frequencies, giving two
  unrelated ~2 MHz windows for parallel monitoring.
- **Diversity / coherent** — both tuners share a common clock and can be phase-referenced,
  enabling [antenna diversity](/reference/antenna-diversity/) to fight
  [multipath](/reference/multipath-propagation/) and fading, or experiments in coherent
  processing and direction finding.

The shared reference clock is what makes the coherent mode meaningful: without a common
timebase, two receivers cannot be combined sample-for-sample.

## Variants

The RSPduo is the only dual-tuner model in the RSP line; the single-tuner alternatives
trade its second receiver for other strengths:

- **[RSP1A](/reference/sdrplay-rsp1a/)** — the low-cost single-tuner baseline.
- **[RSPdx](/reference/sdrplay-rspdx/)** — a single tuner with the richest front-end
  filtering and an HDR mode for LF/MW/HF.

All three use SDRplay's proprietary API/service and the shared
[SoapySDR](/reference/soapysdr/) module.

## Relevance to SDR

The RSPduo's niche is anything that benefits from two simultaneous, frequency-agile
receivers: diversity reception, spectrum comparison across bands, and
[multilateration](/reference/multilateration/) or direction-finding experiments that need
phase-coherent captures. For trunking, the dual tuners could in principle follow a control
channel on one receiver while a voice channel is tracked on the other, though the ~2 MHz
per-tuner limit is narrower than an [Airspy](/reference/airspy/)'s single wide capture.

As with every RSP, GopherTrunk has no native RSPduo driver — the hardware needs SDRplay's
closed API/service, reachable only through a SoapySDR bridge rather than GopherTrunk's
direct USB backends for RTL-SDR, HackRF, and Airspy. The receiver is capable; integration
is the open question, so confirm current support before designing around it.

## Where to buy

The RSPduo is sold by SDRplay and its distributors, and listed on Amazon — stock is often
third-party, so the button is a tagged search that resolves to current listings. Before you
buy it for GopherTrunk, weigh two things: it is supported **over the network only**, through a
[SoapySDR](/reference/soapysdr/)/[rtl_tcp](/reference/rtl-tcp/) bridge rather than
GopherTrunk's direct USB drivers (see the [hardware guide](/hardware.html)); and its second
tuner only earns its price if you actually need two simultaneous receivers or
[antenna diversity](/reference/antenna-diversity/). If you want a single 14-bit receiver, the
[RSP1B](/reference/sdrplay-rsp1b/) or [RSP1A](/reference/sdrplay-rsp1a/) is cheaper; for
plug-and-play trunking, an [RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) is
simpler still. Compare radios in [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/s?k=SDRplay+RSPduo&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on receiver-class and diversity-capable SDRs.
[^sdrplay]: [RSPduo](https://www.sdrplay.com/rspduo/) — SDRplay, official product page describing the dual 14-bit tuners, shared clock, and diversity operation.
