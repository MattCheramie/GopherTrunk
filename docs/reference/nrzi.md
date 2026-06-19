---
slug: nrzi
title: NRZI
entry_type: term
category: modulation
description: NRZI (non-return-to-zero inverted) is a line code that represents a bit by the presence or absence of a transition rather than a fixed level — used by AX.25 and AIS.
keywords: NRZI, non-return-to-zero inverted, line code, AX.25, APRS, AIS, bit stuffing
aka: [NRZI, "non-return-to-zero inverted"]
autolink: true
see_also: [ax25, aprs, ais, frequency-shift-keying, clock-recovery]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Non-return-to-zero#NRZI
---

**NRZI** (**non-return-to-zero inverted**) is a line code in which each bit is conveyed
by **whether the signal level changes**, not by the level itself: conventionally a `0`
causes a transition and a `1` causes none (or vice-versa).[^wiki] It is used by
[AX.25](/reference/ax25/)/[APRS](/reference/aprs/) and [AIS](/reference/ais/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A bit pattern above a NRZI waveform that changes level on each zero and holds level on each one." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle"><text x="60" y="30">0</text><text x="110" y="30">1</text><text x="160" y="30">1</text><text x="210" y="30">0</text><text x="260" y="30">0</text><text x="310" y="30">1</text></g>
  <path d="M35 80 V50 H85 V80 H135 V80 H185 V50 H235 V80 H285 V50 H335 V50 H385" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="230" y="108" text-anchor="middle" font-size="8.5" fill="currentColor">transition = 0 · no transition = 1</text>
</svg>
<figcaption>NRZI encodes bits as the presence or absence of a level transition, which guarantees frequent edges for timing.</figcaption>
</figure>

## Overview

Because NRZI ties bits to transitions, combining it with **bit stuffing** (as
[HDLC](/reference/ax25/) does) guarantees regular edges, which keeps the receiver's
[clock recovery](/reference/clock-recovery/) locked even through long runs of identical
data bits.

## Sources

[^wiki]: [Non-return-to-zero — NRZI](https://en.wikipedia.org/wiki/Non-return-to-zero#NRZI) — Wikipedia, for the transition-based line-code definition.
