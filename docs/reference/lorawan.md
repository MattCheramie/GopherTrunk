---
slug: lorawan
title: LoRaWAN
entry_type: protocol
category: wireless-data-iot
description: "LoRaWAN is an open MAC-layer protocol on the LoRa chirp PHY, defining device classes, star-of-stars gateways, and AES security for long-range low-power IoT."
keywords: LoRaWAN, LoRa Alliance, LPWAN, IoT, Class A B C, gateway, network server, ADR, spreading factor, ISM band, OTAA, ABP
aka: [LoRaWAN, "LoRa WAN"]
autolink: true
infobox:
  - { label: Type, value: Low-power wide-area network (LPWAN) }
  - { label: Standards body, value: LoRa Alliance }
  - { label: Introduced, value: "2015" }
  - { label: Access, value: ALOHA-style, per-device spreading factor }
  - { label: PHY, value: LoRa chirp spread spectrum }
  - { label: Bands, value: "Sub-GHz ISM (EU868, US915, AS923, …)" }
  - { label: Security, value: AES-128 (network + application keys) }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [lora, lora-alliance, meshtastic, sigfox, internet-of-things]
cite_urls:
  - https://en.wikipedia.org/wiki/LoRa#LoRaWAN
  - https://lora-alliance.org/about-lorawan/
---

**LoRaWAN** is the open **MAC-layer and network protocol** that rides on top of the
[LoRa](/reference/lora/) chirp physical layer, turning long-range radio links into a
managed [IoT](/reference/internet-of-things/) network with addressing, security, and
gateways.[^wiki] Where LoRa defines *how a symbol is sent*, LoRaWAN defines *how a fleet
of battery devices, gateways, and a server cooperate* — it is maintained by the
[LoRa Alliance](/reference/lora-alliance/) rather than by the chip vendor. Note that its
spread-spectrum PHY uses frequency **chirps**, not
[direct-sequence](/reference/direct-sequence-spread-spectrum/) code spreading.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="LoRaWAN star-of-stars topology: end devices send LoRa uplinks to multiple gateways, which backhaul to a central network server and on to applications." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lw_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="40" cy="35" r="9" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="40" y="58">device</text>
    <circle cx="40" cy="90" r="9" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="40" y="113">device</text>
    <circle cx="40" cy="140" r="9" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="40" y="163">device</text>
    <rect x="170" y="45" width="46" height="26" fill="none" stroke="currentColor"/><text x="193" y="86">gateway</text>
    <rect x="170" y="100" width="46" height="26" fill="none" stroke="currentColor"/><text x="193" y="141">gateway</text>
    <rect x="300" y="72" width="54" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/><text x="327" y="118">net server</text>
    <rect x="405" y="72" width="46" height="30" fill="none" stroke="currentColor"/><text x="428" y="118">app</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-opacity="0.8">
    <line x1="52" y1="37" x2="168" y2="55" marker-end="url(#lw_ar)"/>
    <line x1="52" y1="90" x2="168" y2="58" marker-end="url(#lw_ar)"/>
    <line x1="52" y1="90" x2="168" y2="110" marker-end="url(#lw_ar)"/>
    <line x1="52" y1="138" x2="168" y2="118" marker-end="url(#lw_ar)"/>
    <line x1="218" y1="58" x2="298" y2="82" marker-end="url(#lw_ar)"/>
    <line x1="218" y1="113" x2="298" y2="92" marker-end="url(#lw_ar)"/>
    <line x1="356" y1="87" x2="403" y2="87" marker-end="url(#lw_ar)"/>
  </g>
</svg>
<figcaption>LoRaWAN is a star-of-stars network: any gateway that hears a device forwards its packet to one network server, which deduplicates and routes to the application.</figcaption>
</figure>

## Overview

A LoRaWAN device does not associate with a single gateway. It simply broadcasts an
uplink; **every** gateway in range forwards the frame to a central network server, which
discards duplicates, checks the message integrity code, and passes the payload to the
application server. This "network the sky" design means coverage improves just by adding
gateways, and devices stay dumb, cheap, and long-lived on a coin cell.

## Technical characteristics

| Property | Value |
|----------|-------|
| PHY | LoRa chirp spread spectrum (SF7–SF12) |
| Bands | Sub-GHz ISM: EU868, US915, AS923, IN865, … |
| Data rate | ~0.3 kbps (SF12) to ~50 kbps (SF7 / FSK) |
| Payload | ~11–242 bytes depending on data rate |
| Access | Asynchronous ALOHA; adaptive data rate (ADR) |
| Security | AES-128 CMAC/CTR; separate network + application keys |
| Activation | OTAA (join procedure) or ABP (pre-provisioned) |

Higher spreading factors trade throughput for range and link budget: SF12 decodes far
below the [noise floor](/reference/noise-floor/) but occupies the air far longer, so ADR
pushes each device to the fastest rate its link can sustain.

## Device classes

- **Class A** — the mandatory baseline. Each uplink is followed by two short downlink
  receive windows, then the radio sleeps. Lowest power; downlinks only ride on an uplink.
- **Class B** — adds scheduled receive slots synchronized to gateway beacons, so the
  server can reach a device at predictable times without waiting for it to talk first.
- **Class C** — receives continuously except while transmitting. Lowest latency but
  highest power, so it suits mains-powered actuators rather than battery sensors.

## History

The LoRaWAN specification was first published by the LoRa Alliance in 2015, standardizing
the network layer above Semtech's LoRa PHY.[^all] Regional parameter documents and the
1.0.x and 1.1 specification lines followed, adding roaming, class refinements, and
tightened key handling.

## Deployment

LoRaWAN backs public and private IoT networks worldwide — utility metering, agriculture,
asset tracking, and building sensors — including community networks such as The Things
Network. It competes with cellular LPWANs like [NB-IoT](/reference/nb-iot/) and
[Sigfox](/reference/sigfox/), trading carrier SLAs for unlicensed-band autonomy.

## Decoding it with GopherTrunk

LoRaWAN is outside GopherTrunk's scope: GopherTrunk is a trunked land-mobile *voice*
scanner (P25, DMR, NXDN, TETRA, …), not an IoT gateway. LoRaWAN frames are readily
*visible* on an SDR [waterfall](/reference/waterfall-display/) as the diagonal chirps of
LoRa, and general-purpose tools plus a gateway concentrator can receive them, but decoding
the payload also requires the device's AES keys. GopherTrunk neither implements the MAC
layer nor manages keys.

## Sources

[^wiki]: [LoRa — LoRaWAN](https://en.wikipedia.org/wiki/LoRa#LoRaWAN) — Wikipedia, for the MAC-layer role above the LoRa PHY, device classes, and star-of-stars topology.
[^all]: [About LoRaWAN](https://lora-alliance.org/about-lorawan/) — LoRa Alliance, for the specification's stewardship, security model, and network architecture.
