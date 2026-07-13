---
slug: wireless-mbus
title: Wireless M-Bus (wM-Bus)
entry_type: protocol
category: wireless-data-iot
description: "Wireless M-Bus is a European standard (EN 13757) for reading utility meters over short-range FSK radio, mostly at 868 MHz, and is receivable with common SDR tools."
keywords: Wireless M-Bus, wM-Bus, EN 13757, utility meters, AMR, smart metering, 868 MHz, FSK, GFSK, mode S T C N, water gas heat electricity
aka: [Wireless M-Bus, wM-Bus, "wireless Meter-Bus"]
autolink: true
infobox:
  - { label: Type, value: Utility-meter reading (AMR) radio }
  - { label: Standards body, value: "CEN — EN 13757-4" }
  - { label: Access, value: Unslotted, meter-initiated }
  - { label: Bands, value: "868 MHz (also 169 MHz, 433 MHz)" }
  - { label: Modulation, value: FSK / GFSK (mode-dependent) }
  - { label: Modes, value: "S, T, C, N, F (rate/band variants)" }
  - { label: GopherTrunk support, value: Not decoded (receivable with other SDR tools) }
see_also: [frequency-shift-keying, gfsk, internet-of-things, software-defined-radio, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Meter-Bus
  - https://en.wikipedia.org/wiki/Wireless_Meter-Bus
---

**Wireless M-Bus** (**wM-Bus**) is the radio variant of the European Meter-Bus standard,
used to read water, gas, heat, and electricity **utility meters** without a wired
connection.[^wiki] Defined in the EN 13757 family (the radio layer is EN 13757-4), it lets a
meter periodically transmit its reading over short-range sub-GHz radio to a fixed collector
or a walk-by/drive-by reader — the backbone of automatic meter reading (AMR) across much of
Europe. It uses simple [FSK](/reference/frequency-shift-keying/) modulation, and its frames
are readily received with ordinary SDR hardware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Several utility meters each transmitting a short FSK radio burst to a single collector or drive-by reader, illustrating wireless M-Bus automatic meter reading." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wm_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="30" y="25" width="34" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/><text x="47" y="60">water</text>
    <rect x="30" y="75" width="34" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/><text x="47" y="110">gas</text>
    <rect x="30" y="118" width="34" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/><text x="47" y="150">heat</text>
    <rect x="350" y="62" width="60" height="40" fill="none" stroke="currentColor"/><text x="380" y="120">collector</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-opacity="0.8">
    <line x1="66" y1="36" x2="348" y2="72" marker-end="url(#wm_ar)"/>
    <line x1="66" y1="86" x2="348" y2="82" marker-end="url(#wm_ar)"/>
    <line x1="66" y1="129" x2="348" y2="92" marker-end="url(#wm_ar)"/>
  </g>
  <text x="200" y="147" text-anchor="middle" font-size="8.5" fill="currentColor">meters push short 868 MHz FSK bursts — no request needed</text>
</svg>
<figcaption>Wireless M-Bus meters push brief FSK bursts to a collector or drive-by reader, typically at 868 MHz.</figcaption>
</figure>

## Overview

A wM-Bus meter is usually the talker: it wakes on a schedule, sends a short frame
containing its identity and consumption data, and returns to deep sleep to preserve its
battery. The counterpart — a fixed concentrator or a technician's mobile reader — listens
and logs. Frames may be sent in the clear or encrypted (commonly AES-128), depending on the
utility's configuration.

## Technical characteristics

| Property | Value |
|----------|-------|
| Standard | EN 13757-4 (radio) within the M-Bus family |
| Bands | 868 MHz (primary); also 169 MHz and 433 MHz |
| Modulation | FSK / [GFSK](/reference/gfsk/), rate depends on mode |
| Modes | S (stationary), T (frequent transmit), C (compact), N (169 MHz), F (433 MHz) |
| Direction | Meter-to-collector primary; some bidirectional |
| Security | Optional AES-128 encryption of payload |

The various letter modes trade data rate, band, and duty cycle to fit different meter types
and reading strategies, from occasional walk-by to fixed-network collection.

## History

Wired M-Bus (EN 13757-2/-3) came first for cabled meter buses; the wireless layer was added
as EN 13757-4 to serve meters where running cable is impractical.[^wm] It has been widely
deployed as European utilities rolled out smart and automatic metering, and the OMS (Open
Metering System) profile builds on it for interoperability.

## Deployment

wM-Bus is common across European water, gas, heat, and electricity metering. Its short,
infrequent bursts and long battery life make it a natural fit for meters buried in
basements and pits, complementing cellular LPWANs like [NB-IoT](/reference/nb-iot/) that
some newer meters use instead.

## Decoding it with GopherTrunk

GopherTrunk does not decode wireless M-Bus — it is a trunked land-mobile *voice* scanner,
and metering telemetry is outside its scope. However, wM-Bus is very much *receivable with
general SDR tools*: an [RTL-SDR](/reference/rtl-sdr/) or similar
[software-defined radio](/reference/software-defined-radio/) tuned to 868 MHz, paired with
community decoders (for example `rtl_433` or `wmbusmeters`), will pull out meter frames, and
readings decode fully when the frame is unencrypted or the key is known. That workflow just
lives in those tools, not in GopherTrunk.

## Sources

[^wiki]: [Meter-Bus](https://en.wikipedia.org/wiki/Meter-Bus) — Wikipedia, for the M-Bus standard family and its metering role.
[^wm]: [Wireless Meter-Bus](https://en.wikipedia.org/wiki/Wireless_Meter-Bus) — Wikipedia, for the EN 13757-4 radio layer, 868 MHz operation, and mode structure.
