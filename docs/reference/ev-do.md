---
slug: ev-do
title: EV-DO (3G data)
entry_type: protocol
category: cellular
description: "EV-DO is the CDMA2000 high-speed data standard that dedicates a 1.25 MHz carrier to packet data with a fast rate-adaptive downlink."
keywords: EV-DO, EVDO, 1xEV-DO, Evolution-Data Optimized, CDMA2000, IS-856, 3G data, CDMA, 3GPP2, 1.25 MHz, cellular
aka: [EV-DO, EVDO, 1xEV-DO, Evolution-Data Optimized, IS-856]
autolink: true
infobox:
  - { label: Type, value: "Cellular packet data (3G)" }
  - { label: Standards body, value: 3GPP2 }
  - { label: Introduced, value: "2002" }
  - { label: Access, value: "CDMA + time-scheduled downlink" }
  - { label: Channel spacing, value: 1.25 MHz }
  - { label: Modulation, value: "QPSK, 8PSK, 16-QAM (adaptive)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [cdma2000, cdma, umts-wcdma, hspa]
cite_urls:
  - https://en.wikipedia.org/wiki/EV-DO
  - https://en.wikipedia.org/wiki/CDMA2000
---

**EV-DO** (Evolution-Data Optimized, also 1xEV-DO or IS-856) is the high-speed data
member of the [CDMA2000](/reference/cdma2000/) family. It dedicates an entire 1.25 MHz
[CDMA](/reference/cdma/) carrier to packet data and, rather than sharing the downlink by
code alone, time-multiplexes it: the base station transmits at full power to one
best-placed user at a time, choosing the data rate from that user's reported channel
quality.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An EV-DO base station transmitting at full power to one user per 1.67 ms slot, picking the fastest rate each user's channel can support." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="46" width="66" height="44" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="63" y="72" text-anchor="middle" font-size="8.5" fill="currentColor">base stn</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="150" y="30" width="40" height="22" fill="currentColor" fill-opacity="0.28"/>
    <rect x="190" y="30" width="40" height="22"/>
    <rect x="230" y="30" width="40" height="22" fill="currentColor" fill-opacity="0.28"/>
    <rect x="270" y="30" width="40" height="22"/>
    <rect x="310" y="30" width="40" height="22" fill="currentColor" fill-opacity="0.28"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="170" y="45">A</text><text x="210" y="45">C</text><text x="250" y="45">A</text><text x="290" y="45">B</text><text x="330" y="45">A</text></g>
  <path d="M96 60 L146 42" stroke="currentColor" stroke-width="1.1" marker-end="url(#evdoar)"/>
  <text x="250" y="72" text-anchor="middle" font-size="8" fill="currentColor">1.67 ms slots · full power to one user each slot</text>
  <text x="250" y="112" text-anchor="middle" font-size="9" fill="currentColor">rate chosen per user from reported channel quality (DRC)</text>
  <defs><marker id="evdoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>EV-DO serves the downlink one user per 1.67 ms slot at full power, adapting the rate to each user's reported channel quality.</figcaption>
</figure>

## Overview

[CDMA2000](/reference/cdma2000/) 1xRTT carried voice and data together, which capped
data performance. EV-DO ("DO" for data-optimized) splits data onto its own carrier and
optimises purely for throughput. The downlink is time-division multiplexed in 1.67 ms
slots; each terminal continuously reports the best rate its channel can sustain (a data
rate control signal), and a proportional-fair scheduler picks who to serve. Adaptive
modulation from QPSK up to 16-QAM and hybrid ARQ push peak rates well above 1xRTT. This
is conceptually parallel to [HSPA](/reference/hspa/) on the [W-CDMA](/reference/umts-wcdma/)
side.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 3G data |
| Family | CDMA2000 (IS-856) |
| Carrier spacing | 1.25 MHz (dedicated to data) |
| Downlink | Time-multiplexed, full-power to one user per slot |
| Slot length | 1.67 ms |
| Modulation | QPSK, 8PSK, 16-QAM (adaptive) |
| Peak (Rev. A) | ≈3.1 Mbit/s down, ≈1.8 Mbit/s up |
| Rate control | Per-user DRC feedback, proportional-fair scheduling |

Giving the whole carrier to one user at a time, at full power, is what lets EV-DO reach
high peak rates despite the narrow 1.25 MHz channel.

## History

EV-DO was standardised by 3GPP2 as IS-856 and reached networks from about 2002
(Release 0), with Revisions A and B raising uplink rates and adding multi-carrier
bonding. It gave CDMA2000 operators a mobile-broadband answer to
[UMTS](/reference/umts-wcdma/) HSPA during the 3G era.

## Deployment

EV-DO powered mobile broadband for CDMA carriers in the US, South Korea, and elsewhere,
delivered through phones and USB modems. As those operators moved to
[LTE](/reference/lte/) for 4G, EV-DO and the rest of [CDMA2000](/reference/cdma2000/)
were phased out and their carriers refarmed.

## Decoding it with GopherTrunk

GopherTrunk scans trunked land-mobile and utility signals; **cellular data such as EV-DO
is out of scope and is not decoded.** It carries private, authenticated, ciphered
subscriber traffic on licensed [CDMA2000](/reference/cdma2000/) spectrum. EV-DO is
documented here only for reference within the 3G family.

## Sources

[^wiki]: [EV-DO](https://en.wikipedia.org/wiki/EV-DO) — Wikipedia, for the CDMA2000 EV-DO data standard, its dedicated 1.25 MHz carrier, time-multiplexed downlink, and rate-adaptive scheduling.
