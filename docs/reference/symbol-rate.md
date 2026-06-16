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
  - { title: "Symbols, baud & bitrate", url: /learn/symbols-and-baud/ }
external:
  - { title: "Symbol rate (Wikipedia)", url: https://en.wikipedia.org/wiki/Symbol_rate }
---

**Symbol rate** (**baud**) is the number of modulation symbols transmitted per second.
It differs from bit rate whenever each symbol carries more than one bit.

## How it works

The relationship is **bit rate = symbol rate × bits-per-symbol**. P25 and DMR run at
4800 baud with 4-level modulation (2 bits/symbol), giving 9600 bps. Higher symbol rates
need more [bandwidth](/reference/bandwidth/).

## Relevance to SDR

The symbol rate sets the rhythm that [clock recovery](/reference/clock-recovery/) must
lock to, and the minimum capture [bandwidth](/reference/bandwidth/) for a clean decode.
