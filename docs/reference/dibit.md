---
slug: dibit
title: Dibit
entry_type: term
category: modulation
description: A dibit is a pair of bits carried by a single symbol in four-level modulations such as C4FM and 4FSK — the unit each P25 or DMR symbol represents.
keywords: dibit, two bits per symbol, 4FSK, C4FM, P25 symbol, DMR symbol
aka: [dibit]
autolink: true
see_also: [symbol-rate, c4fm, frequency-shift-keying, constellation-diagram]
related_lessons:
  - { title: "Symbols, baud & bitrate", url: /learn/rf-sdr/symbols-and-baud/ }
external:
  - { title: "Dibit (Wikipedia)", url: https://en.wikipedia.org/wiki/Dibit }
---

A **dibit** is a **pair of bits** represented by one transmitted symbol. In the
four-level modulations used by [P25](/reference/p25-phase-1/) ([C4FM](/reference/c4fm/))
and [DMR](/reference/dmr/) ([4FSK](/reference/frequency-shift-keying/)), each of the four
symbol states maps to one of the four dibits `00`, `01`, `10`, `11`.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 130" role="img" aria-label="Four symbol levels each labelled with the two-bit dibit it represents." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.35"><line x1="120" y1="30" x2="340" y2="30"/><line x1="120" y1="60" x2="340" y2="60"/><line x1="120" y1="90" x2="340" y2="90"/><line x1="120" y1="120" x2="340" y2="120"/></g>
  <g font-size="10" fill="currentColor" text-anchor="end" font-family="monospace"><text x="110" y="33">+3 → 01</text><text x="110" y="63">+1 → 00</text><text x="110" y="93">-1 → 10</text><text x="110" y="123">-3 → 11</text></g>
</svg>
<figcaption>Each of the four C4FM/4FSK levels carries one dibit, so the bit rate is twice the symbol rate.</figcaption>
</figure>

## Overview

Because each symbol carries two bits, a four-level signal's **bit rate is twice its
[symbol rate](/reference/symbol-rate/)** — which is why P25 Phase 1's 4800-baud C4FM
runs at 9600 bits per second.
