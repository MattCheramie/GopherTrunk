---
slug: four-fsk
title: 4FSK
entry_type: technology
category: modulation
description: 4FSK is four-level frequency-shift keying that carries two bits per symbol across four deviations; it is the modulation behind DMR, NXDN, and P25 C4FM and Phase 2.
keywords: 4FSK, four-level FSK, 4-level FSK, C4FM, DMR, NXDN, P25, dibit, deviation, symbol levels, land mobile radio
aka: [4FSK, four-level FSK, 4-level FSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (M-ary FSK) }
  - { label: Carries, value: 2 bits (a dibit) per symbol }
  - { label: Used by, value: DMR, NXDN, P25 C4FM }
see_also: [frequency-shift-keying, m-ary-fsk, c4fm, dmr, nxdn, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
  - https://en.wikipedia.org/wiki/Continuous_phase_modulation
---

**4FSK** is four-level [frequency-shift keying](/reference/frequency-shift-keying/): the
[carrier](/reference/carrier-wave/) is switched among **four** discrete frequency
deviations, so each [symbol](/reference/symbol-rate/) encodes a two-bit **dibit**.[^wiki]
It is the workhorse modulation of digital land-mobile radio — the physical layer of
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/), and the
[C4FM](/reference/c4fm/) form of [P25](/reference/project-25/) — because its constant
envelope and modest bandwidth suit efficient handheld transmitters.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Four horizontal deviation levels labelled plus three, plus one, minus one, minus three with a stepped symbol path visiting them, each mapped to a two-bit dibit." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.3"><line x1="90" y1="30" x2="440" y2="30"/><line x1="90" y1="60" x2="440" y2="60"/><line x1="90" y1="90" x2="440" y2="90"/><line x1="90" y1="120" x2="440" y2="120"/></g>
  <g font-size="9" fill="currentColor" font-family="monospace"><text x="20" y="33">+3 (01)</text><text x="20" y="63">+1 (00)</text><text x="20" y="93">-1 (10)</text><text x="20" y="123">-3 (11)</text></g>
  <path d="M100 60 H160 V30 H220 V120 H280 V90 H340 V30 H400" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="130" cy="60" r="3"/><circle cx="190" cy="30" r="3"/><circle cx="250" cy="120" r="3"/><circle cx="310" cy="90" r="3"/><circle cx="370" cy="30" r="3"/></g>
</svg>
<figcaption>4FSK uses four frequency deviations; each symbol level maps to a dibit, carrying two bits at a time — the basis of DMR, NXDN, and P25 C4FM.</figcaption>
</figure>

## How it works

The modulator maps each incoming dibit to one of four evenly spaced frequency offsets
above and below the carrier — conventionally +3, +1, −1, −3 deviation units. In P25 C4FM
these correspond to ±1.8 kHz and ±0.6 kHz around the channel centre, transmitted at
**4800 symbols per second**, which yields 9600 bits per second gross. The bit-to-symbol
mapping is chosen so adjacent levels differ by a single bit, a [Gray
code](/reference/gray-code/), so the most likely error — slipping to a neighbouring
level — corrupts only one bit.

Crucially, land-mobile 4FSK is **continuous-phase**: the frequency changes are shaped so
the phase never jumps, giving a constant envelope that non-linear amplifiers handle
efficiently. C4FM in particular is filtered so tightly that it is mathematically
equivalent to a [CQPSK](/reference/cqpsk/) phase modulation on the same symbol grid — a
receiver can demodulate the identical signal either as 4-level FSK or as a phase
constellation, which is why C4FM and CQPSK interoperate.

## In practice

A 4FSK demodulator recovers the instantaneous frequency, slices it into four levels, and
reads off dibits. The four levels appear as four rails on a symbol scope and as four
clusters on a [constellation](/reference/constellation-diagram/) or
[eye diagram](/reference/eye-diagram/); a clean signal shows tight, well-separated
groups, while noise, timing error, or deviation mismatch smears them together.
Symbol-clock recovery and deviation normalisation are the two adjustments that most
affect decode reliability.

## Relevance to SDR

4FSK is central to the trunked and conventional digital voice systems that scanner users
care about. [DMR](/reference/dmr/) and [NXDN](/reference/nxdn/) are 4FSK, P25 Phase 1 is
C4FM 4FSK, and P25 Phase 2 uses an H-CPM/H-DQPSK 4-level scheme on a TDMA grid. Paging
and telemetry protocols also use 4FSK for its bit-per-symbol efficiency over plain 2FSK.

This is the core of GopherTrunk's decode chain: it demodulates C4FM/4FSK to recover the
dibit stream for P25, DMR, and NXDN before framing, error correction, and vocoder
decoding. GopherTrunk decodes clear and scrambled traffic; keyed encryption remains
opaque regardless of modulation.

## Sources

[^wiki]: [Frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying) — Wikipedia, for M-ary FSK and the four-level (2 bits/symbol) case used in land-mobile radio.
