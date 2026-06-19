---
slug: symbol-rate
title: Symbol rate (baud)
entry_type: term
category: modulation
description: Symbol rate, or baud, is the number of modulation symbols sent per second; the bit rate equals symbol rate times bits per symbol.
keywords: symbol rate, baud, bit rate, bits per symbol, 4800 baud, 9600 bps
aka: [symbol rate, baud]
autolink: true
infobox:
  - { label: Symbol, value: Rs }
  - { label: Unit, value: Baud (symbols/second) }
  - { label: Relation, value: "bit rate = baud × bits/symbol" }
see_also: [frequency-shift-keying, phase-shift-keying, quadrature-amplitude-modulation, bandwidth, clock-recovery]
related_lessons:
  - { title: "Symbols, baud & bitrate", url: /learn/rf-sdr/symbols-and-baud/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Symbol_rate
---

**Symbol rate** (**baud**) is the number of modulation symbols transmitted per second.[^wiki]
It differs from bit rate whenever each symbol carries more than one bit.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A row of symbol slots over time, each slot carrying two bits, showing baud versus bit rate." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="40" y="40" width="70" height="34"/><rect x="110" y="40" width="70" height="34"/><rect x="180" y="40" width="70" height="34"/><rect x="250" y="40" width="70" height="34"/><rect x="320" y="40" width="70" height="34"/>
  </g>
  <g font-size="11" fill="currentColor" text-anchor="middle" font-family="monospace"><text x="75" y="62">01</text><text x="145" y="62">11</text><text x="215" y="62">00</text><text x="285" y="62">10</text><text x="355" y="62">01</text></g>
  <text x="215" y="26" text-anchor="middle" font-size="10" fill="currentColor">5 symbols (baud) × 2 bits = 10 bits</text>
  <text x="215" y="96" text-anchor="middle" font-size="10" fill="currentColor">time → · bit rate = baud × bits-per-symbol</text>
</svg>
<figcaption>The symbol rate (baud) counts symbols per second; with 2 bits per symbol the bit rate is twice the baud.</figcaption>
</figure>

## How it works

The relationship is **bit rate = symbol rate × bits-per-symbol**. P25 and DMR run at
4800 baud with 4-level modulation (2 bits/symbol), giving 9600 bps. Higher symbol rates
need more [bandwidth](/reference/bandwidth/).

## Relevance to SDR

The symbol rate sets the rhythm that [clock recovery](/reference/clock-recovery/) must
lock to, and the minimum capture [bandwidth](/reference/bandwidth/) for a clean decode.

## Sources

[^wiki]: [Symbol rate](https://en.wikipedia.org/wiki/Symbol_rate) — Wikipedia, for the baud definition and the bit-rate relationship.
