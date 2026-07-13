---
slug: dibit
title: Dibit
entry_type: term
category: modulation
description: "A dibit is a pair of bits carried by a single symbol in four-level modulations such as C4FM and 4FSK — the unit each P25 or DMR symbol represents."
keywords: dibit, two bits per symbol, 4FSK, C4FM, P25 symbol, DMR symbol, Gray code, tribit, quadbit, symbol mapping
aka: [dibit]
autolink: true
see_also: [symbol-rate, bit-rate-vs-baud, c4fm, frequency-shift-keying, constellation-diagram]
related_lessons:
  - { title: "Symbols, baud & bitrate", url: /learn/rf-sdr/symbols-and-baud/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Dibit
  - https://en.wikipedia.org/wiki/Gray_code
---

A **dibit** is a **pair of bits** represented by one transmitted symbol.[^wiki] In the
four-level modulations used by [P25](/reference/p25-phase-1/) ([C4FM](/reference/c4fm/))
and [DMR](/reference/dmr/) ([4FSK](/reference/frequency-shift-keying/)), each of the four
symbol states maps to one of the four dibits `00`, `01`, `10`, `11`. The word is simply the
two-bit generalisation of "bit," the way "byte" names eight — a compact name for the unit of
data one four-level symbol conveys.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 130" role="img" aria-label="Four symbol levels each labelled with the two-bit dibit it represents." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.35"><line x1="120" y1="30" x2="340" y2="30"/><line x1="120" y1="60" x2="340" y2="60"/><line x1="120" y1="90" x2="340" y2="90"/><line x1="120" y1="120" x2="340" y2="120"/></g>
  <g font-size="10" fill="currentColor" text-anchor="end" font-family="monospace"><text x="110" y="33">+3 → 01</text><text x="110" y="63">+1 → 00</text><text x="110" y="93">-1 → 10</text><text x="110" y="123">-3 → 11</text></g>
</svg>
<figcaption>Each of the four C4FM/4FSK levels carries one dibit, so the bit rate is twice the symbol rate.</figcaption>
</figure>

## How it works

Four distinct symbol states can be told apart at the receiver, and log₂4 = 2, so each state
labels a two-bit pattern. Which dibit maps to which level is not arbitrary: the standards use a
**Gray-coded** mapping, in which the dibits assigned to physically adjacent levels differ in only
one bit. In P25 C4FM the four frequency deviations +1800, +600, −600, −1800 Hz carry the dibits
`01`, `00`, `10`, `11` respectively — read down the levels and only a single bit changes between
neighbours. The payoff is error resilience: the most likely symbol error is a slip to an adjacent
level, and with Gray coding that slip corrupts just one of the two bits instead of both, roughly
halving the resulting bit-error rate.

The direct consequence for throughput is that a four-level signal's **bit rate is twice its
[symbol rate](/reference/symbol-rate/)** — the general [bits-per-symbol
relationship](/reference/bit-rate-vs-baud/) with two bits per symbol. That is why P25 Phase 1's
4800-baud C4FM runs at 9600 bits per second, and DMR's 4800-baud 4FSK likewise.

## In practice

Grouping the bitstream two at a time is exactly what a four-level modulator does on transmit and
what the demodulator undoes on receive: after symbol-timing recovery the slicer classifies each
sample into one of the four levels, then a lookup turns that level back into its dibit, and the
dibits are concatenated to rebuild the bitstream for framing and error correction. The two-bit
alphabet is why P25/DMR frame structures, sync words, and interleaver spans are often described
and counted in symbols (dibits) rather than raw bits. The same idea scales: a **tribit** is three
bits per symbol (eight levels, as in 8-FSK or [8PSK](/reference/8psk/)) and a **quadbit** four
bits (sixteen states, as in 16-[QAM](/reference/quadrature-amplitude-modulation/)).

## Relevance to SDR

For a scanner, "dibit" is the natural unit at the boundary between DSP and protocol logic: the
demodulator's job ends when it has classified each symbol into a dibit, and the framer's job
begins there. A [constellation](/reference/constellation-diagram/) or
[eye diagram](/reference/eye-diagram/) of a healthy four-level signal shows the four cleanly
separated levels the slicer relies on. GopherTrunk's C4FM/4FSK decode chain produces exactly this
dibit stream from the demodulated 4800-baud symbols before handing it to P25 and DMR framing,
where sync detection and forward error correction operate.

## Sources

[^wiki]: [Dibit](https://en.wikipedia.org/wiki/Dibit) — Wikipedia, for the two-bits-per-symbol definition.
[^gray]: [Gray code](https://en.wikipedia.org/wiki/Gray_code) — Wikipedia, for why adjacent symbol levels are assigned dibits differing in a single bit.
