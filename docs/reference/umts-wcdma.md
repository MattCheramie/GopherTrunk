---
slug: umts-wcdma
title: UMTS / W-CDMA (3G)
entry_type: protocol
category: cellular
description: "UMTS is the 3G cellular standard whose W-CDMA air interface spreads users across a shared 5 MHz carrier using code-division multiple access."
keywords: UMTS, W-CDMA, WCDMA, 3G, CDMA, 5 MHz, 3GPP, UTRAN, spreading code, HSPA, cellular
aka: [UMTS, W-CDMA, WCDMA, Universal Mobile Telecommunications System, 3G]
autolink: true
infobox:
  - { label: Type, value: Digital cellular (3G) }
  - { label: Standards body, value: 3GPP }
  - { label: Introduced, value: "2001" }
  - { label: Access, value: "CDMA (W-CDMA)" }
  - { label: Channel spacing, value: 5 MHz }
  - { label: Modulation, value: "QPSK (HSPA adds 16/64-QAM)" }
  - { label: Vocoder, value: "AMR / AMR-WB" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [cdma, 3gpp, hspa, gsm, qpsk]
cite_urls:
  - https://en.wikipedia.org/wiki/UMTS
  - https://www.3gpp.org/technologies/umts
---

**UMTS** (Universal Mobile Telecommunications System) is the third-generation (3G)
cellular standard from [3GPP](/reference/3gpp/), and **W-CDMA** (Wideband
[CDMA](/reference/cdma/)) is its primary air interface. Rather than assigning each user
a slot or a narrow channel, W-CDMA spreads every user across a shared 5 MHz carrier
with a unique spreading code, so many calls occupy the same spectrum at once and are
separated by code correlation.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Several users, each with a distinct spreading code, overlaid on one shared 5 MHz W-CDMA carrier and separated at the receiver by code correlation." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="300" height="60" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="190" y="30" text-anchor="middle" font-size="9" fill="currentColor">one shared 5 MHz carrier</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="190" y="60">user A · code Cₐ</text>
    <text x="190" y="76">user B · code C_b</text>
    <text x="190" y="92">user C · code C_c</text>
  </g>
  <path d="M340 70 h60" stroke="currentColor" stroke-width="1.2" fill="none" marker-end="url(#umtsar)"/>
  <text x="410" y="60" text-anchor="middle" font-size="8" fill="currentColor">de-spread</text>
  <text x="410" y="84" text-anchor="middle" font-size="8" fill="currentColor">by code</text>
  <text x="190" y="120" text-anchor="middle" font-size="9" fill="currentColor">codes are orthogonal / near-orthogonal → users coexist</text>
  <defs><marker id="umtsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>W-CDMA overlays many users on one 5 MHz carrier, each tagged by a spreading code that the receiver correlates against to recover that user.</figcaption>
</figure>

## Overview

UMTS was the European-led branch of the ITU's IMT-2000 3G family. Its W-CDMA radio
access network (UTRAN) uses direct-sequence [CDMA](/reference/cdma/): each user's data
is multiplied by a fast spreading code (chip rate 3.84 Mchip/s) that widens the signal
across the 5 MHz carrier. All users share that carrier simultaneously, and tight power
control keeps every signal at the base station near the same level so no one user
drowns out the others. The design brought far higher data rates and true simultaneous
voice-and-data to mainstream phones.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 3G |
| Access | Direct-sequence CDMA (W-CDMA) |
| Carrier spacing | 5 MHz |
| Chip rate | 3.84 Mchip/s |
| Duplex | FDD (paired) and TDD variants |
| Modulation | QPSK; HSPA adds 16-QAM and 64-QAM |
| Vocoder | AMR / AMR-WB (adaptive multi-rate) |
| Core network | Evolved from the GSM/GPRS packet core |

Fast closed-loop power control (1500 times per second) is essential to CDMA and is one
of the features that distinguishes W-CDMA from the TDMA of GSM.

## History

[3GPP](/reference/3gpp/) specified UMTS in its Release 99, and the first commercial
W-CDMA networks launched in 2001–2003. Operators bid heavily for 3G spectrum around the
turn of the millennium. Later releases added the [HSPA](/reference/hspa/) family
(HSDPA and HSUPA), sharply raising throughput and turning 3G into a practical mobile
broadband platform.

## Deployment

UMTS/W-CDMA became the dominant 3G technology in Europe, much of Asia, and beyond,
often deployed by the same operators who ran [GSM](/reference/gsm/) so that handsets
could fall back to 2G outside 3G coverage. It has since been superseded by
[LTE](/reference/lte/) and 5G, and many carriers have shut down their 3G networks to
reclaim the 5 MHz carriers for newer technology.

## Decoding it with GopherTrunk

GopherTrunk is a trunked-radio and utility-signal scanner; **cellular telephony such as
UMTS/W-CDMA is out of scope and is not decoded.** Its wideband CDMA carriers hold
private, authenticated, ciphered subscriber traffic on licensed spectrum. UMTS appears
here to explain how 3G's code-division approach differs from the time- and
frequency-division schemes used by land-mobile systems.

## Sources

[^wiki]: [UMTS](https://en.wikipedia.org/wiki/UMTS) — Wikipedia, for the 3G UMTS standard, its 5 MHz W-CDMA air interface, chip rate, and evolution into HSPA.
