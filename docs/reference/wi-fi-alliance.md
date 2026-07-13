---
slug: wi-fi-alliance
title: Wi-Fi Alliance
entry_type: organization
category: organizations
description: "The Wi-Fi Alliance is the industry consortium that certifies interoperability of IEEE 802.11 wireless LAN products and owns the Wi-Fi trademark."
keywords: Wi-Fi Alliance, Wi-Fi certification, IEEE 802.11, WLAN, WPA, Wi-Fi 6, Wi-Fi 7, interoperability
aka: [Wi-Fi Alliance, WiFi Alliance, WFA]
autolink: true
infobox:
  - { label: Type, value: Industry consortium }
  - { label: Founded, value: "1999" }
  - { label: Role, value: Certifies 802.11 interoperability }
see_also: [wifi-80211, wi-fi, ieee, ofdm, bluetooth-sig]
cite_urls:
  - https://www.wi-fi.org/
  - https://en.wikipedia.org/wiki/Wi-Fi_Alliance
---

**The Wi-Fi Alliance** is a non-profit industry consortium that owns the "Wi-Fi"
trademark and runs the certification program guaranteeing that wireless-LAN products from
different vendors interoperate.[^home] It does not write the underlying radio standard —
that is [IEEE 802.11](/reference/wifi-80211/) — but it tests and brands products against
selected profiles of that standard, and the familiar "Wi-Fi CERTIFIED" logo signals that a
device has passed.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="IEEE writes the 802.11 standard; the Wi-Fi Alliance certifies products against it and grants the Wi-Fi CERTIFIED mark." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wfa_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="18" y="46" width="96" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="66" y="62">IEEE</text><text x="66" y="72" font-size="7.5">802.11 standard</text>
    <rect x="182" y="42" width="96" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="59">Wi-Fi</text><text x="230" y="70" font-size="7.5">Alliance</text>
    <rect x="346" y="46" width="96" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="394" y="62">Wi-Fi</text><text x="394" y="72" font-size="7.5">CERTIFIED</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="114" y1="60" x2="181" y2="60" marker-end="url(#wfa_ar)"/><line x1="278" y1="60" x2="345" y2="60" marker-end="url(#wfa_ar)"/></g>
    <text x="147" y="40" font-size="7.5">tests against</text>
    <text x="312" y="40" font-size="7.5">grants mark</text>
  </g>
</svg>
<figcaption>The Wi-Fi Alliance certifies 802.11 products for interoperability and licenses the Wi-Fi brand.</figcaption>
</figure>

## Overview

The alliance was founded in 1999 (originally as WECA, the Wireless Ethernet Compatibility
Alliance) precisely because the raw 802.11 standard was large and permissive enough that
two conforming products could still fail to talk to each other. By defining tighter
interoperability profiles and running a compliance-test program in authorized labs, the
alliance gave buyers a single trustworthy mark. It coined the name "Wi-Fi" as a
consumer-friendly brand — the term is a marketing invention, not an abbreviation for
anything.

Over time the alliance took on several roles beyond basic interoperability. It defined the
**WPA**, **WPA2**, and **WPA3** security certifications that turned IEEE's security
amendments into deployable products, and it introduced the simplified generational naming —
**Wi-Fi 4**, **Wi-Fi 5**, **Wi-Fi 6**, and **Wi-Fi 7** — that maps onto the far less
memorable IEEE designations 802.11n, ac, ax, and be. It also certifies feature programs
such as Wi-Fi Direct, Passpoint, and Wi-Fi Easy Connect. Membership spans chipset makers,
device manufacturers, and network operators, and the organization is headquartered in
Austin, Texas.

## Relevance to SDR

For SDR work, the Wi-Fi Alliance sits one layer above the physical radio: the signal an SDR
captures is defined by [IEEE 802.11](/reference/wifi-80211/), while the alliance governs
what "counts" as a certified product. The physical layer is dense —
[OFDM](/reference/ofdm/) subcarriers, wide channels (20 to 320 MHz), and short packet
bursts — which puts most consumer SDRs at the edge of their sample-rate budget for full
capture. Researchers nonetheless use SDRs to study 802.11 preambles, spectrum occupancy,
and coexistence, and the alliance's coexistence work (for example around
[Bluetooth](/reference/bluetooth-sig/) sharing the 2.4 GHz band) is directly relevant to
anyone characterizing that crowded spectrum.

GopherTrunk does not decode Wi-Fi; it is a narrowband trunked-radio scanner aimed at
land-mobile systems, and 802.11's wideband, packet-switched nature is well outside its
scope. The Wi-Fi Alliance is included here as part of the broader map of who governs which
wireless technology — a useful contrast to the land-mobile standards bodies whose systems
GopherTrunk does target.

## Sources

[^home]: [Wi-Fi Alliance](https://www.wi-fi.org/) — the alliance's official site, for its certification programs, generational naming, and the Wi-Fi trademark.
[^wiki]: [Wi-Fi Alliance](https://en.wikipedia.org/wiki/Wi-Fi_Alliance) — Wikipedia, for the organization's history and its relationship to IEEE 802.11.
