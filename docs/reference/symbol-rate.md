---
slug: symbol-rate
title: Symbol rate (baud)
entry_type: term
category: modulation
description: Symbol rate, or baud, is the number of modulation symbols sent per second; the bit rate equals symbol rate times bits per symbol.
keywords: symbol rate, baud, bit rate, bits per symbol, 4800 baud, 9600 bps, modulation rate, signalling rate, Nyquist bandwidth
aka: [symbol rate, baud]
autolink: true
infobox:
  - { label: Symbol, value: Rs }
  - { label: Unit, value: Baud (symbols/second) }
  - { label: Relation, value: "bit rate = baud × bits/symbol" }
see_also: [bit-rate-vs-baud, frequency-shift-keying, phase-shift-keying, quadrature-amplitude-modulation, bandwidth, clock-recovery]
related_lessons:
  - { title: "Symbols, baud & bitrate", url: /learn/rf-sdr/symbols-and-baud/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Symbol_rate
  - https://en.wikipedia.org/wiki/Baud
---

**Symbol rate** (**baud**) is the number of modulation symbols transmitted per second.[^wiki]
A *symbol* is one distinct state of the carrier — a phase, a frequency, an amplitude, or a
combination — held for one signalling interval, and the symbol rate counts how many of those
intervals fit into a second. It differs from the [bit rate](/reference/bit-rate-vs-baud/)
whenever each symbol carries more than one bit, so conflating the two is one of the most common
mistakes when describing a digital radio.

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

The governing identity is **bit rate = symbol rate × bits-per-symbol**, where bits-per-symbol
is log₂ of the number of distinct symbol states (an *alphabet* of M symbols carries log₂M bits
each). A two-level scheme carries one bit per symbol, so its bit rate and baud are equal; a
four-level scheme carries two bits per symbol (a [dibit](/reference/dibit/)), doubling the bit
rate for the same baud. P25 Phase 1 and DMR both run at **4800 baud** with four-level modulation
([C4FM](/reference/c4fm/) / [4FSK](/reference/frequency-shift-keying/)), giving **9600 bps**.
[TETRA](/reference/tetra/) runs at 18 000 baud with [π/4-DQPSK](/reference/pi-4-dqpsk/) — two
bits per symbol — for 36 kbit/s gross.

The symbol rate, not the bit rate, sets the signal's occupied [bandwidth](/reference/bandwidth/).
The Nyquist relation says a symbol rate of *Rs* baud needs at minimum *Rs/2* Hz of baseband
bandwidth (about *Rs* Hz after passband double-sidebanding), before any [pulse
shaping](/reference/pulse-shaping/) [roll-off](/reference/roll-off-factor/) widens it. Packing
more bits into each symbol — going from four levels to sixteen with
[QAM](/reference/quadrature-amplitude-modulation/) — raises the bit rate without widening the
channel, at the cost of shrinking the spacing between symbol states and so demanding a higher
signal-to-noise ratio to keep them apart.

## In practice

Because bandwidth follows baud, spectrum regulators effectively cap the symbol rate a channel
may use, and system designers then choose the modulation order to hit their target throughput.
Once a radio is on the air the symbol rate is fixed and known, which the receiver exploits: it is
the exact rhythm the [clock-recovery](/reference/clock-recovery/) loop must track, the spacing
the [eye diagram](/reference/eye-diagram/) opens at, and the interval the matched filter is
built around. A small error between the transmitter's and receiver's notion of the symbol rate
accumulates as timing drift and eventually slides the sampling instant off the eye centre.

## Relevance to SDR

Knowing a protocol's symbol rate is the starting point for decoding it: it sets the minimum
capture [bandwidth](/reference/bandwidth/) for a clean copy, the resampling ratio from the
SDR's raw sample rate down to an integer number of samples per symbol, and the frequency the
symbol-timing recovery must lock to. GopherTrunk resamples each protocol's channel to a fixed
samples-per-symbol rate (for example the 4800-baud C4FM family is handled at 48 kHz, ten
samples per symbol) so the timing recovery and slicer see a consistent geometry regardless of
the front-end sample rate.

## Sources

[^wiki]: [Symbol rate](https://en.wikipedia.org/wiki/Symbol_rate) — Wikipedia, for the baud definition and the bit-rate relationship.
[^baud]: [Baud](https://en.wikipedia.org/wiki/Baud) — Wikipedia, for the unit's history and the distinction from bits per second.
