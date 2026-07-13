---
slug: 8psk
title: 8PSK
entry_type: technology
category: modulation
description: 8PSK (eight phase-shift keying) carries three bits per symbol across eight carrier phases; it raises data rate over QPSK and appears in EDGE, DVB-S2, and ProVoice.
keywords: 8PSK, eight phase shift keying, 8-PSK, three bits per symbol, eight phases, EDGE, DVB-S2, ProVoice, higher order modulation, spectral efficiency
aka: [8PSK, 8-PSK, eight phase-shift keying]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (PSK) }
  - { label: Carries, value: 3 bits per symbol (eight phases) }
  - { label: Used by, value: EDGE, DVB-S2, ProVoice }
see_also: [phase-shift-keying, qpsk, quadrature-amplitude-modulation, constellation-diagram, spectral-efficiency, gray-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-shift_keying
  - https://en.wikipedia.org/wiki/Amplitude_and_phase-shift_keying
---

**8PSK** (eight phase-shift keying) is [phase-shift
keying](/reference/phase-shift-keying/) with **eight** carrier phases spaced 45° apart,
so each [symbol](/reference/symbol-rate/) carries **three bits**.[^wiki] It lifts the data
rate 50% above [QPSK](/reference/qpsk/) in the same bandwidth, at the price of packing the
constellation points closer together — which demands a cleaner channel to keep them apart.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 230" role="img" aria-label="An 8PSK constellation with eight points evenly spaced around a circle at 45-degree intervals on the IQ plane, each carrying a three-bit tribit." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="115" x2="270" y2="115" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="25" x2="150" y2="205" stroke="currentColor" stroke-opacity="0.4"/>
  <circle cx="150" cy="115" r="80" fill="none" stroke="currentColor" stroke-opacity="0.25" stroke-dasharray="3 3"/>
  <text x="262" y="129" font-size="10" fill="currentColor">I</text><text x="136" y="35" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor"><circle cx="230" cy="115" r="4"/><circle cx="207" cy="58" r="4"/><circle cx="150" cy="35" r="4"/><circle cx="93" cy="58" r="4"/><circle cx="70" cy="115" r="4"/><circle cx="93" cy="172" r="4"/><circle cx="150" cy="195" r="4"/><circle cx="207" cy="172" r="4"/></g>
  <text x="40" y="222" font-size="9" fill="currentColor">eight phases, 45&#176; apart, three bits (a tribit) each</text>
</svg>
<figcaption>8PSK spaces eight phase points evenly around the circle; each encodes three bits, raising throughput over QPSK but shrinking the noise margin between points.</figcaption>
</figure>

## How it works

An 8PSK modulator maps each three-bit group (a *tribit*) to one of eight phases at 0°,
45°, 90°, … 315°, all at the same amplitude. Because the points lie on a circle, the
angular gap between neighbours is only 45° — half QPSK's 90° — so for equal transmit
power the minimum distance between symbols is smaller, and 8PSK needs a few dB more
signal-to-noise ratio to match QPSK's error rate. A [Gray code](/reference/gray-code/)
around the circle again ensures adjacent-symbol slips flip just one bit.

This is the general pattern of higher-order PSK: every extra bit per symbol crowds the
constellation and costs noise margin. Beyond 8PSK the points get so close that pure phase
modulation becomes inefficient, and systems switch to
[quadrature amplitude modulation](/reference/quadrature-amplitude-modulation/), which
uses amplitude *and* phase to spread points across a two-dimensional grid rather than a
single ring — buying more bits per symbol for less penalty. 8PSK sits right at the
sweet spot where phase-only modulation is still worthwhile.

## Relevance to SDR

8PSK shows up wherever a link wants more throughput than QPSK but still values the
constant envelope of phase-only modulation. **EDGE** (Enhanced Data rates for GSM
Evolution) added 8PSK to GSM to roughly triple its data rate; **DVB-S2** satellite uses
8PSK as a step above QPSK when the link budget allows; and Motorola's **ProVoice**
digital land-mobile mode uses an 8-level scheme. On a constellation display 8PSK appears
as eight clusters on a circle, and its tighter spacing makes it a good visual test of a
receiver's phase-noise and SNR performance.

GopherTrunk's decode targets — P25, DMR, NXDN, TETRA — use 4FSK/C4FM and π/4-DQPSK rather
than 8PSK, so 8PSK is not in its decode chain. It is documented here to round out the PSK
family and to mark the crossover point where engineers move from higher-order PSK to QAM.

## Sources

[^wiki]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, for the 8PSK constellation, three-bits-per-symbol mapping, and the SNR penalty of higher-order PSK.
