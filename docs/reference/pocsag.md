---
slug: pocsag
title: POCSAG
entry_type: protocol
category: protocols
description: POCSAG (CCIR Radiopaging Code No. 1) is the classic asynchronous FSK paging protocol used worldwide for numeric and alphanumeric pager messages at 512, 1200, and 2400 bps.
keywords: POCSAG, paging, CCIR 584, pager, FSK, numeric alphanumeric, DAPNET, fire EMS
aka: [POCSAG]
autolink: true
infobox:
  - { label: Type, value: One-way paging protocol }
  - { label: Standard, value: CCIR Radiopaging Code No. 1 }
  - { label: Modulation, value: 2-level FSK }
  - { label: Bit rates, value: 512 / 1200 / 2400 bps }
  - { label: Error correction, value: BCH(31,21) + parity }
  - { label: GopherTrunk support, value: Decoded }
see_also: [flex, frequency-shift-keying, bch-code, demodulation]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "POCSAG (Wikipedia)", url: https://en.wikipedia.org/wiki/POCSAG }
  - { title: "GopherTrunk POCSAG decoder", url: /pocsag.html }
---

**POCSAG** (the **CCIR Radiopaging Code No. 1**) is the classic one-way **paging**
protocol used worldwide to deliver numeric and alphanumeric messages to pagers. It is
a simple asynchronous **2-level [FSK](/reference/frequency-shift-keying/)** scheme,
still in active use by hospitals, fire/EMS, and industry.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A POCSAG transmission: preamble, sync codeword, then batches of address and message codewords." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="20" y="40" width="70" height="28" fill="none"/><rect x="90" y="40" width="50" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="140" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="220" y="40" width="80" height="28" fill="none"/><rect x="300" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="380" y="40" width="60" height="28" fill="none"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="55" y="58">preamble</text><text x="115" y="58">sync</text><text x="180" y="58">addr</text><text x="260" y="58">msg</text><text x="340" y="58">addr</text><text x="410" y="58">msg</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">2-FSK, one-way · 512/1200/2400 bps</text>
</svg>
<figcaption>POCSAG sends a preamble and sync, then batches of address and message codewords to pagers.</figcaption>
</figure>

## Overview

A POCSAG transmission begins with a preamble and sync codeword, then batches of
address and message codewords. Each pager listens for its address (capcode). Messages
are protected by a [BCH(31,21)](/reference/bch-code/) code plus a parity bit.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 2-FSK (±4.5 kHz typical) |
| Bit rates | 512, 1200, 2400 bps |
| Coding | BCH(31,21) + even parity |
| Content | Numeric or alphanumeric |

## History

Developed by the British Post Office and standardised by the CCIR in the early 1980s;
it became the dominant global paging code.

## Deployment

Hospitals, emergency services, and industrial paging; the amateur DAPNET network also
uses POCSAG.

## Decoding it with GopherTrunk

GopherTrunk demodulates the FSK, recovers codewords, and decodes numeric/alphanumeric
messages. See the [POCSAG decoder](/pocsag.html) page.
