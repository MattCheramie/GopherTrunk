---
slug: meshtastic
title: Meshtastic
entry_type: technology
category: wireless-data-iot
description: "Meshtastic is an open-source, off-grid mesh messaging system that relays short text and GPS over the long-range LoRa radio using managed flooding, no cellular or LoRaWAN needed."
keywords: Meshtastic, LoRa mesh, off-grid messaging, ESP32, nRF52, managed flooding, AES, text messaging, GPS, ISM band
aka: [Meshtastic]
autolink: true
infobox:
  - { label: Type, value: Open-source LoRa mesh network }
  - { label: Idea, value: Off-grid text + position relayed device-to-device }
  - { label: Radio, value: LoRa PHY on sub-GHz ISM (433/868/915 MHz) }
  - { label: Examples, value: Trail comms, disaster/off-grid, hobbyist mesh }
see_also: [lora, lorawan, internet-of-things, frequency-bands, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Meshtastic
  - https://meshtastic.org/
---

**Meshtastic** is an open-source project that turns cheap [LoRa](/reference/lora/) radio
modules into an **off-grid mesh** for short text messages and GPS positions, with no
carrier, gateway, or internet required.[^wiki] Each low-cost node (typically an ESP32 or
nRF52 board with a LoRa transceiver) both originates messages and relays its neighbors',
so a handful of devices extends communication for kilometres across terrain where phones
have no signal — hiking, sailing, events, and emergency scenarios.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A Meshtastic mesh: several nodes connected by peer-to-peer LoRa links so a message from one node hops through intermediate nodes to reach a distant one, with no central gateway." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="50" cy="75" r="12" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="50" y="100">A</text>
    <circle cx="160" cy="40" r="12" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="160" y="28">B</text>
    <circle cx="180" cy="110" r="12" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="180" y="134">C</text>
    <circle cx="300" cy="70" r="12" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="300" y="95">D</text>
    <circle cx="410" cy="55" r="12" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="410" y="43">E</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-opacity="0.7">
    <line x1="62" y1="70" x2="148" y2="45"/>
    <line x1="60" y1="82" x2="168" y2="105"/>
    <line x1="170" y1="48" x2="292" y2="64"/>
    <line x1="190" y1="104" x2="290" y2="76"/>
    <line x1="312" y1="66" x2="398" y2="58"/>
  </g>
  <text x="230" y="147" text-anchor="middle" font-size="8.5" fill="currentColor">a message hops node-to-node — no gateway, no cell tower</text>
</svg>
<figcaption>Meshtastic relays messages hop-by-hop across peer nodes over LoRa; there is no central gateway, unlike LoRaWAN.</figcaption>
</figure>

## How it works

Meshtastic sits directly on the LoRa physical layer but is **not**
[LoRaWAN](/reference/lorawan/): it needs no gateway or network server. Instead it uses
**managed flooding** — when a node receives a packet it has not seen, it rebroadcasts it,
with hop limits and rebroadcast suppression to keep the flood from exploding. Nodes share a
channel defined by a name and a pre-shared AES key, so members of the same channel see each
other's traffic while it stays private to outsiders.

Because it rides on LoRa's chirp modulation, Meshtastic inherits long range at very low
power and low data rate. Devices expose a phone app over Bluetooth or Wi-Fi for composing
messages, viewing the map of nodes, and configuring channels, while the radio itself keeps
running on a small battery or solar for days.

Key design points:

- **Decentralized** — every node is a router; there is no infrastructure to deploy.
- **Preset "modem" profiles** trade range against speed by selecting LoRa spreading factor
  and bandwidth (e.g. "Long Fast" vs "Short Fast").
- **Encrypted channels** — AES protects payloads on a shared-key basis, not per-device
  cellular-style authentication.

## Relevance to SDR

Meshtastic traffic is ordinary LoRa on sub-GHz ISM bands, so on an SDR
[waterfall](/reference/waterfall-display/) it looks like LoRa's characteristic diagonal
chirps in the 433/868/915 MHz [frequency bands](/reference/frequency-bands/). A general
[software-defined radio](/reference/software-defined-radio/) plus a LoRa demodulator can
detect and, with the channel key, decode packets — but this is squarely outside
**GopherTrunk**, which decodes trunked land-mobile *voice* (P25, DMR, NXDN, TETRA, …) and
implements neither the LoRa PHY nor the Meshtastic mesh protocol. Meshtastic is included
here as context for the sub-GHz IoT signals a scanner operator will encounter on the band.

## Sources

[^wiki]: [Meshtastic](https://en.wikipedia.org/wiki/Meshtastic) — Wikipedia, for the open-source LoRa mesh design, managed flooding, and off-grid use.
[^home]: [Meshtastic](https://meshtastic.org/) — project site, for device support, channel/encryption model, and modem presets.
