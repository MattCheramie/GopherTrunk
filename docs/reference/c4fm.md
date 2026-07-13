---
slug: c4fm
title: C4FM
entry_type: technology
category: modulation
description: C4FM is the four-level continuous-phase FSK modulation used by P25 Phase 1 and Yaesu System Fusion, carrying 2 bits per symbol at 4800 baud.
keywords: C4FM, four-level FSK, 4FSK, P25 Phase 1, System Fusion, 4800 baud, CQPSK, continuous phase, dibit, root raised cosine
aka: [C4FM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (4-level FSK) }
  - { label: Symbol rate, value: 4800 baud (9600 bps) }
  - { label: Used by, value: P25 Phase 1, System Fusion }
see_also: [frequency-shift-keying, cqpsk, project-25, p25-phase-1, system-fusion-ysf, symbol-rate, four-fsk, continuous-phase-modulation, fm-deviation, root-raised-cosine-filter, eye-diagram, intersymbol-interference]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Continuous-phase_frequency-shift_keying
---

**C4FM** (compatible four-level FM) is the four-level
[FSK](/reference/frequency-shift-keying/) modulation used by
[P25 Phase 1](/reference/p25-phase-1/) and [System Fusion](/reference/system-fusion-ysf/).
The carrier sits at one of four frequency deviations per [symbol](/reference/symbol-rate/),
carrying 2 bits each.[^p25] It is a [continuous-phase](/reference/continuous-phase-modulation/),
constant-envelope modulation designed to squeeze digital voice into a 12.5 kHz land-mobile
channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Four horizontal deviation levels labelled with dibits, and a stepped trace moving between them over time." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.35"><line x1="60" y1="30" x2="430" y2="30"/><line x1="60" y1="60" x2="430" y2="60"/><line x1="60" y1="90" x2="430" y2="90"/><line x1="60" y1="120" x2="430" y2="120"/></g>
  <g font-size="9" fill="currentColor" text-anchor="end"><text x="55" y="33">+3 (01)</text><text x="55" y="63">+1 (00)</text><text x="55" y="93">-1 (10)</text><text x="55" y="123">-3 (11)</text></g>
  <polyline points="70,60 110,30 150,90 190,90 230,120 270,30 310,60 350,120 390,90 430,30" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="142" text-anchor="middle" font-size="9" fill="currentColor">time → (one dibit per symbol, 4800 baud)</text>
</svg>
<figcaption>C4FM is four-level FSK: each symbol sits at one of four frequency deviations, carrying two bits.</figcaption>
</figure>

## How it works

C4FM runs at 4800 symbols per second, and because each symbol is a two-bit
[dibit](/reference/dibit/) the gross bit rate is 9600 bps — the distinction between
[symbol rate and bit rate](/reference/bit-rate-vs-baud/) that lets a narrow channel carry
a usable voice payload. The four levels map to nominal frequency deviations of +1800,
+600, −600, and −1800 Hz, standing for dibits 01, 00, 10, and 11. To keep the signal
inside its channel mask, the dibit stream is not switched abruptly between levels; it is
first shaped by a [root-raised-cosine](/reference/root-raised-cosine-filter/)-style
baseband filter and then integrated into a continuous phase trajectory, so the spectrum
stays compact and the envelope stays constant. That constant envelope is the whole point:
it lets P25 portables use efficient, non-linear Class-C amplifiers.

The receiver reverses this: it recovers instantaneous frequency (an
[FM](/reference/frequency-modulation/) discriminator), matched-filters, and slices at the
symbol instants to decide which of the four levels was sent. The four levels appear as
four clusters on a [constellation](/reference/constellation-diagram/) and as three stacked
openings — a three-eye pattern — on an [eye diagram](/reference/eye-diagram/). The width
of those eye openings is a direct readout of noise and
[intersymbol interference](/reference/intersymbol-interference/); good timing recovery
samples at the centre of the eye.

## Variants

C4FM is paired with [CQPSK](/reference/cqpsk/) (compatible QPSK, also called linear
simulcast modulation): the two are deliberately engineered to produce the **same recovered
symbol stream**, so a single P25 receiver detects either transmit path. C4FM is the
constant-envelope, FM-family path favoured by battery-powered subscriber units; CQPSK is
the linear path favoured by [simulcast](/reference/simulcast/) infrastructure because
linear modulation superimposes more gracefully when overlapping transmitters are received
together. [System Fusion](/reference/system-fusion-ysf/) (Yaesu YSF) reuses C4FM at the
same 4800 baud for amateur digital voice, and P25's control channel uses the same
modulation as its voice.

## Relevance to SDR

Recognising healthy C4FM symbols is central to decoding [P25 Phase 1](/reference/p25-phase-1/)
and Fusion, and it is exactly what GopherTrunk does: its Phase 1 decode chain recovers the
4800-baud four-level symbols, then frames, deinterleaves, error-corrects, and hands the
IMBE/AMBE voice bits to the [vocoder](/reference/vocoder/). The scopes — constellation and
eye diagram — reveal SNR, tuning offset, and timing problems at a glance, which is why the
[GopherTrunk DSP notes](/reference/eye-diagram/) lean on demod SNR and
[EVM](/reference/error-vector-magnitude/) to judge whether a channel will lock.

## Sources

[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the C4FM modulation, symbol rate, and its use in P25 Phase 1.
[^cpfsk]: [Continuous-phase frequency-shift keying](https://en.wikipedia.org/wiki/Continuous-phase_frequency-shift_keying) — Wikipedia, for the continuous-phase, constant-envelope basis C4FM is built on.
