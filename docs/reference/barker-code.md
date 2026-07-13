---
slug: barker-code
title: Barker code
entry_type: algorithm
category: spread-spectrum
description: A Barker code is a short binary sequence whose autocorrelation sidelobes are at most 1, giving a sharp, unambiguous correlation peak ideal for frame sync, spreading, and radar pulse compression; used in 802.11b and radar.
keywords: Barker code, Barker sequence, autocorrelation, sidelobe, preamble, sync word, 802.11b, pulse compression, radar, spreading code, correlation
aka: [Barker code, Barker sequence]
autolink: true
infobox:
  - { label: Type, value: Short low-sidelobe binary code }
  - { label: Property, value: Autocorrelation sidelobe ≤ 1 }
  - { label: Used by, value: 802.11b, sync words, radar }
see_also: [maximal-length-sequence, gold-code, direct-sequence-spread-spectrum, matched-filter, wi-fi, pulse-shaping]
cite_urls:
  - https://en.wikipedia.org/wiki/Barker_code
  - https://en.wikipedia.org/wiki/Pulse_compression
---

**A Barker code** is a short binary sequence with a near-perfect **autocorrelation**: every
off-peak (sidelobe) value has magnitude at most 1, while the on-peak value equals the code
length N.[^wiki] That gives the sharpest, least ambiguous correlation spike any binary
sequence of that length can produce, which is why Barker codes are the classic choice for
frame **synchronization**, short-code spreading, and radar **pulse compression**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="The length-7 Barker sequence plus-plus-plus-minus-minus-plus-minus, and its autocorrelation showing a peak of seven with all sidelobes at plus or minus one." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" fill="currentColor">
    <rect x="30" y="30" width="16" height="16"/><rect x="48" y="30" width="16" height="16"/><rect x="66" y="30" width="16" height="16"/><rect x="84" y="46" width="16" height="16" fill="none"/><rect x="102" y="46" width="16" height="16" fill="none"/><rect x="120" y="30" width="16" height="16"/><rect x="138" y="46" width="16" height="16" fill="none"/>
  </g>
  <text x="92" y="80" text-anchor="middle" font-size="9" fill="currentColor">Barker-7: + + + − − + −</text>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <path d="M240 140 H440"/>
    <path d="M340 140 V50"/>
    <path d="M260 140 v-4 M280 140 v4 M300 140 v-4 M320 140 v4 M360 140 v4 M380 140 v-4 M400 140 v4 M420 140 v-4"/>
  </g>
  <text x="340" y="45" text-anchor="middle" font-size="9" fill="currentColor">peak = 7</text>
  <text x="340" y="155" text-anchor="middle" font-size="8" fill="currentColor">sidelobes ≤ ±1</text>
</svg>
<figcaption>The length-7 Barker code correlates to a peak of 7 with every sidelobe bounded to ±1 — the flat floor that makes the peak unmistakable.</figcaption>
</figure>

## How it works

Slide a copy of the sequence past itself and sum the products (a
[matched-filter](/reference/matched-filter/) correlation, mapping bits to ±1). At perfect
alignment every term is +1, so the sum is N. At any other shift the aligned terms nearly
cancel, and for a Barker code the leftover never exceeds ±1 in magnitude. The result is a tall
central peak sitting on an almost flat floor — the ideal shape for deciding *exactly* where a
known pattern begins in a noisy stream, because no false alignment comes close to the true
peak.

The catch is that Barker codes are **rare and short**. Only a handful exist — lengths 2, 3,
4, 5, 7, 11, and 13 — and no longer binary Barker sequence is known to exist. The length-13
code (peak 13, sidelobes ±1, ~22 dB peak-to-sidelobe) is the longest and a workhorse in
radar. For applications needing long codes, other families take over:
[m-sequences](/reference/maximal-length-sequence/) and [Gold codes](/reference/gold-code/)
give good-but-not-perfect autocorrelation at arbitrary length, and are used where a large
family or a long spreading code matters more than a strictly ±1 floor.

## In practice

Barker codes fill three related roles:

- **Sync / preamble.** A receiver correlates for the code to find frame boundaries and set
  symbol timing.
- **Spreading.** Multiplying each data bit by a Barker code spreads it by the code length —
  the [DSSS](/reference/direct-sequence-spread-spectrum/) mode of early Wi-Fi.
- **Pulse compression.** A radar transmits a long Barker-phase-coded pulse for energy, then
  compresses it on receive to Barker-code time resolution — decoupling range resolution from
  transmit power.

## Relevance to SDR

The best-known RF use is **802.11b Wi-Fi**, whose 1 and 2 Mbit/s DSSS modes spread each
symbol with the length-11 Barker code (higher rates switch to CCK/OFDM). **Radar** systems use
Barker phase coding, especially length 13, for pulse compression. More broadly, short
sync/correlation sequences with sharp autocorrelation appear as preambles throughout digital
[Wi-Fi](/reference/wi-fi/) and other packet radios.

GopherTrunk's land-mobile trunking targets (P25, DMR, NXDN, TETRA) do not use Barker spreading;
they use their own frame sync words, and GopherTrunk correlates for *those* to find frame
boundaries — the same peak-finding idea a Barker code embodies. So while the scanner ships no
Barker-code decoder, the autocorrelation principle documented here is exactly what its sync
detection relies on.

## Sources

[^wiki]: [Barker code](https://en.wikipedia.org/wiki/Barker_code) — Wikipedia, for the sidelobe-≤1 property, the complete list of known Barker lengths, and 802.11b/radar uses.
