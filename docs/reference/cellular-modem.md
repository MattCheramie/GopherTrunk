---
slug: cellular-modem
title: Cellular modem
entry_type: hardware
category: hw-mobile
description: A cellular modem is the radio subsystem that connects a device to mobile networks (4G LTE, 5G), handling the RF, modulation, and protocols needed to carry data and calls over licensed cellular bands.
keywords: cellular modem, baseband, LTE, 5G, 4G, mobile broadband, baseband processor, modem chip, SIM, eSIM
infobox:
  - { label: Type, value: Wireless radio modem }
  - { label: Connects to, value: 4G/5G cellular networks }
  - { label: Also called, value: Baseband processor }
  - { label: Pairs with, value: SIM / eSIM }
see_also: [system-on-a-chip, esim, gps-receiver, smartphone, mobile-operating-system, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Baseband_processor
---

A **cellular modem** is the radio subsystem that connects a device to mobile networks — 4G LTE, 5G, and their predecessors — handling the RF, [modulation](/reference/modulation/), and protocols that carry calls and data over licensed cellular bands.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A cellular modem signal chain. From an antenna, signals pass through an RF front end that filters and amplifies, then a transceiver that converts between radio and baseband, then the baseband processor running the protocol stack, which connects to the main SoC. A SIM or eSIM plugs into the baseband to supply the subscriber identity." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <path d="M40 40 l-8 -16 M40 40 l8 -16 M40 40 v46" stroke-width="1.2"/>
    <g fill-opacity="0" >
      <rect x="66" y="46" width="70" height="40" rx="3"/>
      <rect x="156" y="46" width="70" height="40" rx="3"/>
      <rect x="246" y="46" width="80" height="40" rx="3"/>
      <rect x="346" y="46" width="76" height="40" rx="3"/>
      <rect x="246" y="106" width="80" height="30" rx="3"/>
    </g>
    <line x1="48" y1="66" x2="66" y2="66"/>
    <line x1="136" y1="66" x2="156" y2="66"/>
    <line x1="226" y1="66" x2="246" y2="66"/>
    <line x1="326" y1="66" x2="346" y2="66"/>
    <line x1="286" y1="86" x2="286" y2="106"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="40" y="100">antenna</text>
    <text x="101" y="63">RF front</text>
    <text x="101" y="74">end</text>
    <text x="191" y="63">trans-</text>
    <text x="191" y="74">ceiver</text>
    <text x="286" y="63">baseband</text>
    <text x="286" y="74">processor</text>
    <text x="384" y="63">main</text>
    <text x="384" y="74">SoC</text>
    <text x="286" y="124">SIM / eSIM</text>
  </g>
</svg>
<figcaption>A cellular modem chains an RF front end, a transceiver, and a baseband processor running the protocol stack; the SIM or eSIM supplies the subscriber identity, and the whole block hands data to the main SoC.</figcaption>
</figure>

## Overview

Often called the *baseband processor*, a cellular modem runs its own real-time firmware and manages the complex dance of attaching to a tower, negotiating bandwidth, authenticating, and hopping between cells as the device moves. It is paired with a subscriber identity from a SIM or an [eSIM](/reference/esim/), which the network uses to identify and bill the account.

In a phone the modem is usually a block inside the main [SoC](/reference/system-on-a-chip/) (or a companion chip), wired to its own antennas separate from Wi-Fi and [GPS](/reference/gps-receiver/). Its firmware is a large, security-sensitive body of code — effectively a second computer beside the application processor — precisely because cellular protocols are so intricate and standardized across generations.

## Cellular generations

Each generation raised data rates and reworked the air interface the modem must speak:

| Generation | Era | Peak rate (order) | Note |
|-----------|-----|-------------------|------|
| 2G (GSM) | 1990s | tens of kbit/s | Digital voice, SMS |
| 3G (UMTS) | 2000s | a few Mbit/s | Mobile data arrives |
| 4G (LTE) | 2010s | tens–100s Mbit/s | All-IP, VoLTE |
| 5G (NR) | 2020s | ~Gbit/s | mmWave, low latency |

A modern modem is multi-mode, falling back through these as coverage allows.

## Where it fits

The cellular modem is what makes a [smartphone](/reference/smartphone/) "mobile" in the connectivity sense — always-on data anywhere there is coverage. For a remote GopherTrunk capture node out of Wi-Fi range, a cellular modem (as a USB stick or HAT) is the practical backhaul, letting the node upload decoded calls over the mobile network. The modem speaks the carrier's protocols; it is not a general SDR and does not expose raw RF the way an [RTL-SDR](/reference/rtl-sdr/) does.

## Sources

[^wiki]: [Baseband processor](https://en.wikipedia.org/wiki/Baseband_processor) — Wikipedia, on the modem subsystem in mobile devices.
