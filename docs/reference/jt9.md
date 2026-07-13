---
slug: jt9
title: JT9
entry_type: protocol
category: amateur-digital
description: "JT9 is a narrowband weak-signal amateur-radio digital mode using 9-tone MFSK, designed for LF/MF/HF work with an occupied bandwidth of under 16 Hz per signal."
keywords: JT9, weak signal, 9 tone, MFSK, narrowband, WSJT-X, LF, MF, HF, K1JT, amateur radio digital mode, one minute
aka: [JT9]
autolink: true
infobox:
  - { label: Type, value: Weak-signal amateur digital mode }
  - { label: Developed by, value: Joe Taylor (K1JT) }
  - { label: Introduced, value: 2013 }
  - { label: Modulation, value: 9-tone MFSK, 1.736 baud }
  - { label: Timing, value: ~49 s transmission, 1-min slots }
  - { label: FEC, value: K=32, rate-1/2 convolutional }
  - { label: GopherTrunk support, value: Not decoded (use WSJT-X) }
see_also: [jt65, ft8, m-ary-fsk, convolutional-code, joe-taylor]
cite_urls:
  - https://en.wikipedia.org/wiki/WSJT_(amateur_radio_software)
  - https://wsjt.sourceforge.io/wsjtx.html
---

**JT9** is a **narrowband weak-signal amateur-radio digital mode** designed to pack
sub-noise sensitivity into an extremely small slice of spectrum — an occupied bandwidth of
under 16 Hz, so many signals fit in the space of a single SSB channel. It uses a 9-tone
[MFSK](/reference/m-ary-fsk/) waveform on a one-minute cadence and, like its stablemate
[JT65](/reference/jt65/), exchanges only minimal callsign, grid, and report
messages.[^wiki] JT9 was created chiefly for the low bands, where its narrowness and
sensitivity suit the quiet, slow paths of LF and MF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="JT9 occupies under 16 Hz of bandwidth, so many JT9 signals fit within one 2.5 kHz SSB channel." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="40" width="400" height="40" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="230" y="30" font-size="8.5" fill="currentColor" text-anchor="middle">one ~2.5 kHz SSB passband</text>
  <g stroke="currentColor" stroke-width="2.5">
    <line x1="60" y1="45" x2="60" y2="75"/><line x1="95" y1="45" x2="95" y2="75"/><line x1="130" y1="45" x2="130" y2="75"/><line x1="165" y1="45" x2="165" y2="75"/><line x1="205" y1="45" x2="205" y2="75"/><line x1="250" y1="45" x2="250" y2="75"/><line x1="300" y1="45" x2="300" y2="75"/><line x1="345" y1="45" x2="345" y2="75"/><line x1="395" y1="45" x2="395" y2="75"/>
  </g>
  <text x="230" y="100" font-size="8" fill="currentColor" text-anchor="middle">each bar = one JT9 signal (&lt;16 Hz wide)</text>
</svg>
<figcaption>JT9's sub-16 Hz footprint lets dozens of signals share a single SSB-width passband on the crowded low bands.</figcaption>
</figure>

## Overview

JT9 works like JT65 from the operator's seat: synchronized one-minute transmit/receive
slots, a fixed 72-bit message, and the same station-report script. The difference is under
the hood — nine tones spaced only 1.74 Hz apart and a strong convolutional code yield a
signal a fraction of JT65's width, at comparable sensitivity. That narrowness is the whole
point on LF/MF, where spectrum is scarce and stability is high.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 9-tone MFSK, ~1.74 Hz tone spacing |
| Symbol rate | 1.736 baud (85 symbols) |
| Occupied bandwidth | ~15.6 Hz per signal |
| Slot length | 1 min (~49 s of tones) |
| Message payload | 72 bits |
| FEC | Constraint-length 32, rate-1/2 [convolutional code](/reference/convolutional-code/) |
| Threshold | ≈ −27 dB SNR (2.5 kHz reference) |

## History

JT9 was introduced in 2013 by [Joe Taylor](/reference/joe-taylor/) (K1JT) in WSJT-X,
alongside a refreshed JT65 decoder. It replaced JT65's Reed-Solomon code with the same
constraint-length-32 convolutional code used in WSPR, achieving a slightly better
threshold in far less bandwidth. WSJT-X could decode JT9 and [JT65](/reference/jt65/)
simultaneously in adjacent sub-bands, and both later ceded HF popularity to
[FT8](/reference/ft8/).[^wsjtx]

## Deployment

JT9 is used mainly on the LF, MF, and lower HF amateur bands, where its narrow footprint
and stability are most valuable. Activity is lighter than FT8's but persists among low-band
and QRP operators who prize its efficiency.

## Decoding it with GopherTrunk

GopherTrunk does not decode JT9 — it is a weak-signal HF/LF mode outside the scope of a
trunked land-mobile scanner. JT9 is received with an SSB receiver or SDR feeding clean USB
audio into **WSJT-X**, with the PC clock synchronized to UTC and a stable, well-calibrated
front end.

## Sources

[^wiki]: [WSJT (amateur radio software)](https://en.wikipedia.org/wiki/WSJT_(amateur_radio_software)) — Wikipedia, for the WSJT mode family including JT9's 9-tone MFSK waveform, narrow bandwidth, and timing.
[^wsjtx]: [WSJT-X](https://wsjt.sourceforge.io/wsjtx.html) — the official WSJT-X project page, documenting JT9's modulation, convolutional coding, and low-band design goals.
