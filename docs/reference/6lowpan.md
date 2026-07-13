---
slug: 6lowpan
title: 6LoWPAN
entry_type: protocol
category: wireless-data-iot
description: "6LoWPAN is an IETF adaptation layer that carries IPv6 over low-power IEEE 802.15.4 radios by compressing headers and fragmenting packets to fit tiny frames."
keywords: 6LoWPAN, IPv6 over Low-Power Wireless Personal Area Networks, IETF, IEEE 802.15.4, header compression, fragmentation, RFC 4944, RFC 6282, Thread, IoT
aka: [6LoWPAN, IPv6 over 802.15.4]
autolink: true
infobox:
  - { label: Type, value: IPv6 adaptation layer }
  - { label: Standards body, value: "IETF (RFC 4944, 6282)" }
  - { label: Introduced, value: "2007" }
  - { label: Runs on, value: IEEE 802.15.4 (and others) }
  - { label: Key function, value: IPv6 header compression + fragmentation }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [zigbee-802154, thread-protocol, internet-of-things, ip-address, matter-protocol, connectivity-standards-alliance]
cite_urls:
  - https://en.wikipedia.org/wiki/6LoWPAN
  - https://datatracker.ietf.org/doc/html/rfc4944
---

**6LoWPAN** (IPv6 over Low-Power Wireless Personal Area Networks) is an IETF adaptation
layer that lets full IPv6 packets travel over the tiny, low-power frames of
[IEEE 802.15.4](/reference/zigbee-802154/) radios.[^wiki] It sits between the 802.15.4
link and the IP stack, compressing headers and fragmenting packets so that internet
addressing reaches even the smallest battery-powered
[IoT](/reference/internet-of-things/) devices — the foundation on which
[Thread](/reference/thread-protocol/) is built.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="6LoWPAN compresses a large 40-byte IPv6 header down to a few bytes and, when a packet is too big, fragments it across several 127-byte 802.15.4 frames." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lowar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="currentColor">
    <rect x="30" y="30" width="120" height="24" fill="currentColor" fill-opacity="0.25"/><text x="90" y="45" stroke="none">IPv6 header (40 B)</text>
    <line x1="155" y1="42" x2="200" y2="42" stroke-width="1" marker-end="url(#lowar)"/>
    <text x="177" y="34" font-size="7" stroke="none">compress</text>
    <rect x="205" y="32" width="34" height="20" fill="currentColor" fill-opacity="0.5"/><text x="222" y="46" stroke="none" font-size="7">~4 B</text>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="currentColor">
    <text x="90" y="82" stroke="none">large payload</text>
    <line x1="150" y1="95" x2="195" y2="95" stroke-width="1" marker-end="url(#lowar)"/>
    <text x="172" y="87" font-size="7" stroke="none">fragment</text>
    <rect x="205" y="85" width="60" height="20" fill="none"/><text x="235" y="99" stroke="none" font-size="7">frame 1</text>
    <rect x="270" y="85" width="60" height="20" fill="none"/><text x="300" y="99" stroke="none" font-size="7">frame 2</text>
    <rect x="335" y="85" width="60" height="20" fill="none"/><text x="365" y="99" stroke="none" font-size="7">frame 3</text>
  </g>
  <text x="230" y="130" text-anchor="middle" font-size="9" fill="currentColor">fit IPv6 into 127-byte 802.15.4 frames</text>
</svg>
<figcaption>6LoWPAN shrinks IPv6 headers and splits oversized packets so they fit inside the 127-byte frames of an 802.15.4 radio.</figcaption>
</figure>

## Overview

An 802.15.4 frame is at most 127 bytes, yet a bare IPv6 header alone is 40 bytes and IPv6
requires links to carry packets of 1280 bytes. 6LoWPAN bridges that gap with two core
mechanisms. **Header compression** exploits shared context — the link already knows the
addresses and many fields are predictable — to squeeze a 40-byte IPv6 header (plus UDP)
down to a handful of bytes. **Fragmentation and reassembly** splits a larger IP packet
across several link frames and rebuilds it at the far end. A mesh-forwarding sublayer and
stateless address autoconfiguration round out the design, letting devices derive IPv6
addresses from their 802.15.4 identifiers.

## Technical characteristics

| Property | Value |
|----------|-------|
| Defined by | IETF RFC 4944, updated by RFC 6282/6775 |
| Carried protocol | IPv6 (with UDP compression) |
| Underlying link | IEEE 802.15.4 (also BLE, sub-GHz) |
| Header compression | 40 B → as few as ~4 B |
| Fragmentation | IPv6 ≥1280 B over 127 B frames |
| Addressing | Stateless autoconfiguration from EUI-64 |

Although born for 802.15.4, the same adaptation ideas were later applied over other
constrained links, including Bluetooth LE.

## History

The IETF 6LoWPAN working group published the base specification, RFC 4944, in 2007, with
improved compression (RFC 6282) and neighbour discovery (RFC 6775) following. Its design
directly shaped Thread and other IP-based IoT stacks.

## Deployment

6LoWPAN is rarely a consumer-facing brand; it lives inside stacks like Thread and some
industrial and metering systems. Its significance is architectural: it made "IP all the
way to the sensor" practical on power- and bandwidth-starved radios.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode 6LoWPAN. It is a networking adaptation layer above an
802.15.4 (or similar) radio that GopherTrunk cannot demodulate in the first place, and
its traffic is typically encrypted within Thread. It has no bearing on GopherTrunk's
land-mobile and aeronautical decode chain.

## Sources

[^wiki]: [6LoWPAN](https://en.wikipedia.org/wiki/6LoWPAN) — Wikipedia, on the IPv6-over-802.15.4 adaptation layer, header compression, fragmentation, and its role under Thread.
