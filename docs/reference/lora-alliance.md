---
slug: lora-alliance
title: LoRa Alliance
entry_type: organization
category: organizations
description: "The LoRa Alliance is the non-profit association that standardizes and certifies LoRaWAN, the open low-power wide-area network protocol built on LoRa radio."
keywords: LoRa Alliance, LoRaWAN, LoRa, LPWAN, low-power wide-area network, IoT, certification, open standard
aka: [LoRa Alliance]
autolink: true
infobox:
  - { label: Type, value: Non-profit industry association }
  - { label: Founded, value: "2015" }
  - { label: Standard, value: LoRaWAN }
see_also: [lorawan, lora, internet-of-things, sigfox, nb-iot]
cite_urls:
  - https://lora-alliance.org/
  - https://en.wikipedia.org/wiki/LoRa
---

**The LoRa Alliance** is a non-profit industry association that develops, publishes, and
certifies the [LoRaWAN](/reference/lorawan/) standard — the open networking protocol that
turns the [LoRa](/reference/lora/) radio modulation into a complete low-power
wide-area network (LPWAN) for the Internet of Things.[^home] The alliance owns the LoRaWAN
specification and runs the certification and interoperability program, while the underlying
LoRa physical layer remains proprietary to Semtech.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="LoRa is the proprietary radio layer; the LoRa Alliance standardizes the open LoRaWAN network protocol on top of it." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lora_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="70" width="420" height="24" rx="4" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="230" y="86">LoRa PHY — chirp spread spectrum (Semtech)</text>
    <rect x="20" y="38" width="420" height="24" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="230" y="54">LoRaWAN — MAC &amp; network (LoRa Alliance)</text>
    <rect x="20" y="8" width="420" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="230" y="23">Applications — sensors, meters, trackers</text>
    <g stroke="currentColor" stroke-width="1"><line x1="230" y1="70" x2="230" y2="62" marker-end="url(#lora_ar)"/><line x1="230" y1="38" x2="230" y2="30" marker-end="url(#lora_ar)"/></g>
  </g>
</svg>
<figcaption>The LoRa Alliance standardizes LoRaWAN, the open MAC and network layer built on the LoRa radio.</figcaption>
</figure>

## Overview

The LoRa Alliance was founded in 2015 by a group of technology and telecom companies to
turn LoRa — Semtech's long-range, low-data-rate chirp-spread-spectrum modulation — into an
open, multi-vendor standard rather than a single-supplier product. The key distinction the
alliance draws is between the **radio** and the **network**: LoRa is the patented physical
layer that any licensed chipset implements, whereas **LoRaWAN** is the openly published
media-access and networking specification the alliance controls, defining how end devices,
gateways, network servers, and application servers exchange messages, handle security keys,
and manage data rates.

The alliance publishes the LoRaWAN specification and regional parameters (band plans and
duty-cycle rules differ by country), operates a certification program so that a
"LoRaWAN Certified" device works across compliant networks, and coordinates a large
ecosystem of operators, gateway makers, and sensor vendors. Its design goals are the classic
LPWAN trade-off: very long range and multi-year battery life at the cost of low throughput
and high latency, which suits telemetry — utility meters, agricultural sensors, asset
trackers — rather than anything real-time.

## Relevance to SDR

LoRaWAN operates in sub-GHz [ISM bands](/reference/frequency-bands/) (for example 868 MHz
in Europe and 915 MHz in North America), which sit comfortably within the tuning range and
bandwidth of inexpensive SDRs. Because the LoRaWAN specification is open, SDR-based
receivers and gateways can be built to observe uplink traffic, and the chirp structure of
the LoRa physical layer — up-chirps and down-chirps whose starting frequency encodes the
symbol — is a well-known and visually distinctive target on a
[spectrogram](/reference/spectrogram/). Projects that decode LoRa with GNU Radio rely on
understanding both the proprietary chirp modulation and the LoRaWAN framing the alliance
standardizes.

GopherTrunk does not decode LoRa or LoRaWAN; it is a trunked land-mobile voice scanner and
LPWAN telemetry is outside its scope. The LoRa Alliance is included here to round out the
IoT and LPWAN corner of the standards landscape, alongside competing approaches such as
[Sigfox](/reference/sigfox/) and cellular [NB-IoT](/reference/nb-iot/) — a useful contrast
for readers mapping where each low-power technology fits.

## Sources

[^home]: [LoRa Alliance](https://lora-alliance.org/) — the alliance's official site, for the LoRaWAN specification, regional parameters, and certification program.
[^wiki]: [LoRa](https://en.wikipedia.org/wiki/LoRa) — Wikipedia, for the LoRa modulation, the LoRaWAN protocol, and the alliance's role.
