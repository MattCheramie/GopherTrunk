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
