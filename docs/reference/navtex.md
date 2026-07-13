---
slug: navtex
title: NAVTEX
entry_type: protocol
category: aviation-marine
description: NAVTEX is an international maritime safety broadcast that sends navigational and weather warnings as forward-error-corrected FSK teleprinter text, primarily on 518 kHz.
keywords: NAVTEX, maritime safety information, 518 kHz, 490 kHz, 4209.5 kHz, SITOR-B, FEC, forward error correction, teleprinter, RTTY, GMDSS, MSI
aka: [NAVTEX]
autolink: true
infobox:
  - { label: Type, value: Maritime safety text broadcast }
  - { label: Standards body, value: IMO / ITU (GMDSS) }
  - { label: Introduced, value: 1970s–1980s }
  - { label: Frequencies, value: 518 kHz (int'l), 490 & 4209.5 kHz }
  - { label: Modulation, value: FSK, 100 baud, FEC teleprinter }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [rtty, forward-error-correction, frequency-shift-keying, dsc, itu]
cite_urls:
  - https://en.wikipedia.org/wiki/NAVTEX
  - https://www.itu.int/
---

**NAVTEX** is an international system for broadcasting **maritime safety information** —
navigational warnings, weather forecasts and gale warnings, and search-and-rescue
notices — as printed text to ships automatically. It sends narrow-band
[FSK](/reference/frequency-shift-keying/) teleprinter data with
[forward error correction](/reference/forward-error-correction/), chiefly on **518 kHz**
for English-language international traffic, so a shipboard receiver can print bulletins
around the clock without an operator.[^wiki] NAVTEX is a core component of the Global
Maritime Distress and Safety System (GMDSS) and complements voice and
[DSC](/reference/dsc/) alerting.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A coast station broadcasting a NAVTEX message with a station and message identifier followed by forward-error-corrected safety text to ships." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="navtexar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="45" width="70" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="55" y="64" text-anchor="middle" font-size="8" fill="currentColor">coast stn</text>
  <line x1="90" y1="60" x2="180" y2="60" stroke="currentColor" marker-end="url(#navtexar)"/>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="185" y="46" width="55" height="28" fill="currentColor" fill-opacity="0.2"/>
    <rect x="240" y="46" width="150" height="28" fill="none"/>
  </g>
  <text x="212" y="63" text-anchor="middle" font-size="7.5" fill="currentColor">B1B2B3B4</text>
  <text x="315" y="63" text-anchor="middle" font-size="8" fill="currentColor">FEC safety text</text>
  <text x="230" y="100" text-anchor="middle" font-size="8" fill="currentColor">518 kHz · 100 baud FSK · SITOR-B forward error correction</text>
</svg>
<figcaption>Each NAVTEX message opens with a four-character station/subject/serial identifier, then the forward-error-corrected safety text.</figcaption>
</figure>

## Overview

Every NAVTEX message begins with a four-character technical header — a transmitter
identity letter, a subject-indicator letter (navigational warning, meteorological
warning, ice report, and so on), and a two-digit serial number. Receivers use the header
to **reject messages** from out-of-range stations, on unwanted subjects, or already
printed, so the crew sees only relevant, un-duplicated bulletins. Transmissions are time-
scheduled so that many stations in a region share one frequency without colliding.

## Technical characteristics

| Property | Value |
|----------|-------|
| Frequencies | 518 kHz (international), 490 kHz (national language), 4209.5 kHz (tropical) |
| Modulation | FSK, 170 Hz shift, 100 baud |
| Coding | SITOR collective (FEC) mode with character repetition |
| Header | Station + subject + serial (B1B2B3B4) |
| Access | Scheduled time-sharing by station |

## History

NAVTEX was developed in the 1970s and adopted by the IMO and [ITU](/reference/itu/) as
part of GMDSS, replacing manual receipt of safety traffic. The international 518 kHz
service uses English; 490 kHz was later added for broadcasts in national languages, and
4209.5 kHz serves some tropical regions.

## Deployment

Coast and coastguard stations worldwide broadcast NAVTEX; SOLAS vessels carry dedicated
NAVTEX receivers that print continuously. Its forward-error-corrected mode is the same
SITOR-B / [RTTY](/reference/rtty/)-family teleprinter coding used elsewhere in maritime HF.

## Decoding it with GopherTrunk

GopherTrunk is a land-mobile trunking scanner and does **not** decode NAVTEX. The signal
sits well below GT's target VHF/UHF trunking bands, and its teleprinter FEC decoding is
outside GT's scope. Enthusiasts typically receive NAVTEX with a general-coverage HF
[SDR](/reference/software-defined-radio/) and a dedicated SITOR-B/NAVTEX decoder.

## Sources

[^wiki]: [NAVTEX](https://en.wikipedia.org/wiki/NAVTEX) — Wikipedia, for the NAVTEX safety-broadcast concept, 518/490/4209.5 kHz frequencies, FSK SITOR-B FEC teleprinter coding, and the four-character message header.
