---
slug: gsm
title: GSM (2G)
entry_type: protocol
category: cellular
description: "GSM is the 2G digital cellular standard using TDMA and GMSK on 200 kHz carriers, the world's most widely deployed mobile-phone system."
keywords: GSM, Global System for Mobile Communications, 2G, TDMA, GMSK, 200 kHz, ETSI, 3GPP, SIM, cellular, GSM-FR, GSM-HR, GSM-EFR
aka: [GSM, 2G, Global System for Mobile Communications, Groupe Spécial Mobile]
autolink: true
infobox:
  - { label: Type, value: Digital cellular (2G) }
  - { label: Standards body, value: "ETSI, later 3GPP" }
  - { label: Introduced, value: "1991" }
  - { label: Access, value: "TDMA/FDMA (8 slots/carrier)" }
  - { label: Channel spacing, value: 200 kHz }
  - { label: Modulation, value: GMSK }
  - { label: Vocoder, value: "GSM FR / HR / EFR" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [gmsk, tdma, gsm-fr-hr-efr, 3gpp, gprs, edge-cellular]
cite_urls:
  - https://en.wikipedia.org/wiki/GSM
  - https://www.etsi.org/technologies/mobile/2g
---

**GSM** (Global System for Mobile Communications) is the second-generation (2G)
digital [cellular](/reference/cellular-handover/) standard that carried mobile voice
and text to billions of users worldwide. It combines frequency-division and
[time-division](/reference/tdma/) multiple access — eight time slots on each 200 kHz
carrier — with [GMSK](/reference/gmsk/) modulation, and it introduced the removable
SIM card that decoupled a subscriber identity from any one handset.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A GSM 200 kHz carrier divided into eight repeating TDMA time slots, each carrying a burst." xmlns="http://www.w3.org/2000/svg">
  <line x1="24" y1="112" x2="440" y2="112" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#gsmar)"/>
  <text x="232" y="136" text-anchor="middle" font-size="9" fill="currentColor">time → · one 200 kHz carrier, 8 slots per TDMA frame (4.615 ms)</text>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="34" y="46" width="48" height="52" fill="currentColor" fill-opacity="0.22"/>
    <rect x="82" y="46" width="48" height="52" fill="none"/>
    <rect x="130" y="46" width="48" height="52" fill="none"/>
    <rect x="178" y="46" width="48" height="52" fill="none"/>
    <rect x="226" y="46" width="48" height="52" fill="none"/>
    <rect x="274" y="46" width="48" height="52" fill="none"/>
    <rect x="322" y="46" width="48" height="52" fill="none"/>
    <rect x="370" y="46" width="48" height="52" fill="none"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="58" y="76">0</text><text x="106" y="76">1</text><text x="154" y="76">2</text><text x="202" y="76">3</text><text x="250" y="76">4</text><text x="298" y="76">5</text><text x="346" y="76">6</text><text x="394" y="76">7</text></g>
  <text x="58" y="38" text-anchor="middle" font-size="8" fill="currentColor">one user burst</text>
  <defs><marker id="gsmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>GSM stacks eight TDMA time slots on each 200 kHz carrier; a phone transmits in one slot per frame.</figcaption>
</figure>

## Overview

GSM replaced the incompatible analogue 1G systems of the 1980s with a single
pan-European — and ultimately global — digital standard. A phone tunes a 200 kHz
carrier and is assigned one of eight repeating [TDMA](/reference/tdma/) slots, so
eight calls share each frequency pair (uplink and downlink are separated by a fixed
duplex spacing). Speech is digitised, compressed by a low-bit-rate
[vocoder](/reference/gsm-fr-hr-efr/), protected with channel coding, and sent as
short bursts. The design also standardised SMS text messaging and the SIM smart card.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 2G |
| Access | TDMA over FDMA, 8 slots per carrier |
| Carrier spacing | 200 kHz |
| Modulation | GMSK (0.3 Gaussian filter, 270.833 kbit/s gross) |
| Bands | 900 / 1800 MHz (Europe), 850 / 1900 MHz (Americas) |
| Vocoder | Full Rate, Half Rate, Enhanced Full Rate |
| Frame | 4.615 ms, eight bursts |
| Security | A5/1 or A5/2 stream cipher, SIM-based authentication |

The constant-envelope GMSK waveform lets handsets use efficient, non-linear power
amplifiers, which helped battery life and kept terminals cheap.

## History

Work began in 1982 under the Groupe Spécial Mobile of CEPT; the standard later moved
to [ETSI](/reference/etsi/) and then to [3GPP](/reference/3gpp/), which maintains it
alongside later generations. The first commercial GSM call was placed in Finland in
1991. Through the 1990s and 2000s GSM became the dominant cellular technology on most
continents, eventually serving well over four billion connections and making
international roaming routine.

## Deployment

GSM spawned a family of enhancements: packet data via [GPRS](/reference/gprs/) and
faster [EDGE](/reference/edge-cellular/), both bolted onto the same carriers. Operators
have since refarmed much GSM spectrum for [LTE](/reference/lte/) and 5G, and several
countries have switched 2G off entirely. Even so, GSM lingers as a fallback layer and
underpins low-power machine-to-machine devices in regions where it remains licensed.

## Decoding it with GopherTrunk

GopherTrunk is a land-mobile and utility-signal trunking scanner; **cellular telephony
such as GSM is out of scope and is not decoded.** GSM carries private, operator-licensed
traffic that is authenticated and usually ciphered, and intercepting it is illegal in
most jurisdictions. GSM appears here for reference and to contrast its
[GMSK](/reference/gmsk/) / [TDMA](/reference/tdma/) air interface with the land-mobile
protocols GopherTrunk does handle.

## Sources

[^wiki]: [GSM](https://en.wikipedia.org/wiki/GSM) — Wikipedia, for the 2G GSM standard, its TDMA/FDMA structure, 200 kHz GMSK carriers, and the SIM card.
