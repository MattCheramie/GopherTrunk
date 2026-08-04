---
slug: krakensdr
title: KrakenSDR
entry_type: hardware
category: sdr-devices
description: "KrakenSDR is a five-channel phase-coherent RTL-SDR receiver built for radio direction finding and passive radar."
keywords: KrakenSDR, coherent RTL-SDR, five channel SDR, radio direction finding, DOA, MUSIC algorithm, KerberosSDR, passive radar, phase coherent, R820T2
aka: [KrakenSDR, Kraken SDR]
autolink: true
infobox:
  - { label: Type, value: 5-channel coherent USB SDR }
  - { label: Vendor, value: KrakenRF }
  - { label: Tuners, value: 5 × R820T2 (RTL2832U) }
  - { label: ADC, value: 8-bit (×5) }
  - { label: Range, value: "~24 MHz – 1.766 GHz" }
  - { label: TX, value: No (receive only) }
  - { label: Typical price, value: "$450 – $600" }
see_also: [rtl-sdr, music-algorithm, beamforming, multilateration, rtl2832u, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/RTL-SDR
  - https://www.krakenrf.com/
faq:
  - q: "Does GopherTrunk support the KrakenSDR?"
    a: "Yes, in the sense that each of its five channels is an ordinary R820T2/RTL2832U RTL-SDR, which GopherTrunk drives natively over USB with its pure-Go driver — so you can treat a KrakenSDR as up to five plain RTL-SDR IQ sources. GopherTrunk does not use the array's phase-coherence or direction-finding features; there is no reason to buy a KrakenSDR just for decoding."
  - q: "Can GopherTrunk do radio direction finding with a KrakenSDR?"
    a: "No. GopherTrunk is a trunking decoder, not a DF platform — it does not use the coherent array or the MUSIC direction-of-arrival math. Direction finding and passive radar require KrakenRF's own open-source software, which is the reason to own the hardware."
  - q: "Is a KrakenSDR worth it for scanning trunked systems?"
    a: "Not on its own merits. It is five 8-bit RTL-SDRs priced for a DF niche; for decoding, a single good RTL-SDR or an Airspy is cheaper and simpler. Buy a KrakenSDR when you actually want direction finding or passive radar."
  - q: "Where do I buy a KrakenSDR?"
    a: "Direct from KrakenRF (krakenrf.com) and via Crowd Supply — it is not sold on Amazon. Beware third-party listings claiming otherwise."
---

**KrakenSDR** is a **five-channel, phase-coherent** [RTL-SDR](/reference/rtl-sdr/)
receiver from KrakenRF, built so that all five tuners share a single clock and can be
phase-synchronised — the property that makes **radio direction finding** and passive
radar possible from cheap hardware.[^wiki] It is the productised successor to the earlier
four-channel KerberosSDR, and its headline application is estimating the **direction of
arrival** of a signal using array-processing algorithms.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Five antennas feed five coherent RTL-SDR channels sharing one reference clock and a noise source for calibration; their combined phase data yields a direction-of-arrival bearing." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="krar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="30" y="20">5 antennas</text>
    <g stroke="currentColor" stroke-opacity="0.7" fill="none">
      <line x1="30" y1="30" x2="30" y2="118"/>
    </g>
    <rect x="70" y="28" width="150" height="90" rx="5" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="145" y="46">5 × R820T2 / RTL2832U</text>
    <text x="145" y="66" font-size="7">shared reference clock</text>
    <text x="145" y="82" font-size="7">noise source (calibration)</text>
    <text x="145" y="102" font-size="7">phase-coherent channels</text>
    <rect x="260" y="52" width="90" height="42" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="305" y="70">DoA solver</text><text x="305" y="83" font-size="7">MUSIC</text>
    <text x="410" y="76">bearing</text>
  </g>
  <g stroke="currentColor" stroke-opacity="0.6" fill="none">
    <line x1="220" y1="73" x2="258" y2="73" marker-end="url(#krar)"/>
    <line x1="350" y1="73" x2="382" y2="73" marker-end="url(#krar)"/>
  </g>
</svg>
<figcaption>Five coherent channels plus a switched noise source for phase calibration let the array compute a signal's bearing.</figcaption>
</figure>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Five phase-coherent RTL-SDRs built for direction finding.** KrakenSDR shares one reference
clock across five [R820T2](/reference/rtl2832u/) tuners plus a switched noise source, so it
can run the [MUSIC](/reference/music-algorithm/) direction-of-arrival math and passive radar
that independent dongles can't. **With GopherTrunk** each channel is just a plain
[RTL-SDR](/reference/rtl-sdr/) it drives natively over USB — you can use it as up to five IQ
sources — but GopherTrunk uses **none** of the coherent-array or DF features, so there's no
reason to buy one purely for decoding. **Direction finding needs KrakenRF's own software.**
**Sold direct** from KrakenRF / Crowd Supply — **not on Amazon**. Like every RTL-SDR, it's
8-bit and can't decode [AES encryption](/police-scanner-encryption/).
</div>

## Overview

KrakenSDR packages five [RTL2832U](/reference/rtl2832u/) receivers, each with an
**R820T2** tuner, onto one board. The critical engineering is that they run from a
**single shared reference clock** and include a built-in **noise source** that can be
switched into all inputs to calibrate out the relative phase offsets between channels.
Ordinary RTL-SDR dongles cannot do direction finding because their independent clocks
drift apart; coherence is what turns five receivers into a usable **antenna array**. The
device covers roughly **24 MHz – 1.766 GHz** (the R820T2's range) and, like every
RTL-SDR, is **receive-only** with 8-bit converters.

## What it is for

The whole point of a coherent array is to exploit the *phase differences* of one signal
arriving at antennas spaced apart. KrakenSDR's open-source software feeds those phase
measurements to the [MUSIC algorithm](/reference/music-algorithm/)
(MUltiple SIgnal Classification), a super-resolution method that estimates the
**direction of arrival** far more sharply than the array's raw aperture would suggest.
Driving around while logging bearings lets the software triangulate a transmitter's
location — a mobile form of [multilateration](/reference/multilateration/). The same
coherent hardware also supports **passive radar**, where a broadcast tower serves as the
illuminator and reflections off aircraft or vehicles are detected in the cross-channel
correlation.

The array's geometry sets the mode: antennas in a circle (a uniform circular array) give
360° coverage, while a straight line gives higher accuracy over a limited arc. KrakenSDR
does **not** do [beamforming](/reference/beamforming/) in the transmit sense — it has no
transmit path — but the receive-side array math is the same family of techniques used in
phased-array systems.

## In practice

Because it is five RTL-SDRs, KrakenSDR inherits their limits: an **8-bit ADC** per
channel, easy front-end overload near strong transmitters, and modest per-channel
bandwidth. It also draws real power (a powered USB hub or dedicated supply is usual) and
runs warm. At **$450–$600** it is priced for its niche — direction finding, transmitter
hunting, and passive-radar experimentation — rather than as a general scanner, where a
single good dongle is cheaper and simpler.

## Relevance to GopherTrunk

GopherTrunk is a trunking **decoder**, not a direction-finding platform, so it does not
use KrakenSDR's coherent array or DoA features. In principle any one of its five
channels is an ordinary R820T2 RTL-SDR that GopherTrunk could treat as a plain IQ source,
but there is no reason to buy a KrakenSDR for decoding — a standard [RTL-SDR](/reference/rtl-sdr/)
or an [Airspy](/reference/airspy/) is the right tool. KrakenSDR is worth knowing as the
canonical example of how cheap coherent receivers bring array signal processing —
direction finding, [MUSIC](/reference/music-algorithm/), passive radar — within hobbyist
reach.

## Where to buy

The KrakenSDR is sold **direct from [KrakenRF](https://www.krakenrf.com/)**
and through Crowd Supply — it is **not** available on Amazon, so treat any Amazon listing
claiming to be a KrakenSDR with suspicion. With GopherTrunk it works simply as **five
individual [RTL-SDRs](/reference/rtl-sdr/)** (each driven natively over USB), which is all
GopherTrunk needs from it; its **direction-finding and passive-radar features require
KrakenRF's own open-source software**, not GopherTrunk. If you only want to decode trunked
systems, a single [RTL-SDR](/reference/rtl-sdr/) or an [Airspy](/reference/airspy/) is the
cheaper, simpler choice — see the [hardware guide](/hardware.html).

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/RTL-SDR) — Wikipedia, on coherent multi-channel RTL-SDR receivers and direction finding.
[^krf]: [KrakenRF](https://www.krakenrf.com/) — the vendor, for KrakenSDR hardware, coherence, and direction-finding software.
