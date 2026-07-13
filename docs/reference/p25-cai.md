---
slug: p25-cai
title: P25 Common Air Interface (P25 CAI)
entry_type: protocol
category: land-mobile-trunking
description: The P25 Common Air Interface is the over-the-air standard for Project 25 radios, defining C4FM/CQPSK modulation, the IMBE vocoder, and the frame formats they share.
keywords: P25 CAI, Common Air Interface, Project 25, APCO-25, C4FM, CQPSK, IMBE, 9600 baud, TIA-102, public safety
aka: [P25 CAI, "Common Air Interface", "APCO-25 CAI"]
autolink: true
infobox:
  - { label: Type, value: Digital PMR air-interface standard }
  - { label: Standards body, value: "TIA (TIA-102), APCO" }
  - { label: Introduced, value: "1990s" }
  - { label: Access, value: "FDMA (Phase 1)" }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: "C4FM / CQPSK (4800 sym/s)" }
  - { label: Vocoder, value: "IMBE (Phase 1)" }
  - { label: GopherTrunk support, value: Decoded }
see_also: [project-25, c4fm, cqpsk, imbe, p25-phase-1, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Continuous_4_level_FM
---

The **P25 Common Air Interface** (**CAI**) is the over-the-air portion of
[Project 25](/reference/project-25/) — the suite of standards, defined by the
[TIA](/reference/tia/) as TIA-102, that specifies exactly how P25 radios modulate,
frame, and code their transmissions so equipment from different vendors interoperates.
In Phase 1 the CAI uses **[C4FM](/reference/c4fm/)** (or the spectrally identical
**[CQPSK](/reference/cqpsk/)**) at 4800 symbols per second carrying the
**[IMBE](/reference/imbe/)** vocoder.[^wiki][^c4fm]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="C4FM maps two-bit dibits to four frequency-deviation levels, and CQPSK maps the same dibits to four phase points, producing the same 9600 bps." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="30" y1="80" x2="200" y2="80" stroke-opacity="0.4"/>
    <line x1="30" y1="35" x2="60" y2="35"/><line x1="70" y1="52" x2="100" y2="52"/><line x1="110" y1="108" x2="140" y2="108"/><line x1="150" y1="125" x2="180" y2="125"/>
  </g>
  <g font-size="7.5" fill="currentColor"><text x="62" y="33">+1050</text><text x="102" y="50">+525</text><text x="142" y="106">-525</text><text x="182" y="123">-1050 Hz</text></g>
  <text x="115" y="145" text-anchor="middle" font-size="8.5" fill="currentColor">C4FM: 4 deviation levels (dibit)</text>
  <g stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4"><line x1="270" y1="75" x2="420" y2="75"/><line x1="345" y1="15" x2="345" y2="135"/></g>
  <g fill="currentColor"><circle cx="385" cy="45" r="3"/><circle cx="315" cy="45" r="3"/><circle cx="315" cy="105" r="3"/><circle cx="385" cy="105" r="3"/></g>
  <g font-size="7" fill="currentColor"><text x="390" y="42">01</text><text x="300" y="42">00</text><text x="300" y="118">10</text><text x="390" y="118">11</text></g>
  <text x="345" y="145" text-anchor="middle" font-size="8.5" fill="currentColor">CQPSK: 4 phases, same dibits</text>
</svg>
<figcaption>The Phase 1 CAI carries the same 9600 bps whether transmitted as C4FM's four frequency levels or CQPSK's four phase points; a C4FM receiver detects both.</figcaption>
</figure>

## Overview

The Common Air Interface is what makes P25 an open standard rather than a single vendor's
product: it fixes the symbol rate, deviation levels, framing, forward error correction,
and vocoder so that a subscriber unit built by one manufacturer works on another's
infrastructure. The CAI covers both conventional and trunked operation, and both direct
(unit-to-unit) and repeated modes. A [control channel](/reference/control-channel/)
carries the trunking signalling, while traffic channels carry digitised voice.

## Technical characteristics

| Property | Value |
|----------|-------|
| Symbol rate | 4800 symbols/s (9600 bps) |
| Modulation | C4FM (4-level FM) or CQPSK |
| Channel | 12.5 kHz (Phase 1) |
| Vocoder | IMBE (Phase 1), AMBE+2 half-rate (Phase 2) |
| FEC | Golay, Hamming, Reed–Solomon, trellis (by field) |
| Frame | Fixed sync + Network ID, then data/voice |

C4FM and CQPSK are designed to be **compatible**: they produce the same spectrum and
symbol timing, so a standard C4FM discriminator receiver recovers a CQPSK transmission.
This let agencies mix modulation types during migration. [Phase 1](/reference/p25-phase-1/)
is FDMA; Phase 2 adds two-slot TDMA and the AMBE+2 half-rate vocoder for double capacity.

## History

Project 25 began in the late 1980s under APCO to give US public safety an open digital
replacement for analog FM, and the CAI was among its earliest and most important
deliverables. Published through the TIA-102 series, the air interface was refined over
the 1990s and 2000s, with Phase 2 TDMA added later to improve spectral efficiency.

## Deployment

The P25 CAI underpins statewide and metropolitan public-safety radio across the United
States and in many other countries. Because it is an open, published interface, it is
also the P25 layer that scanners and software decoders target directly.

## Decoding it with GopherTrunk

The P25 CAI is squarely **in scope** for GopherTrunk. It demodulates the 4800-symbol/s
C4FM waveform (which also recovers CQPSK), synchronises on the CAI frame, decodes the
signalling and IMBE voice, and follows P25 trunking by tracking the control channel and
its grants. Encrypted (keyed) P25 voice is not decoded — GopherTrunk handles clear and
known-key traffic only. See [Status](/status.html) for the current state of P25 support.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the TIA-102 Common Air Interface, its C4FM/CQPSK modulation, IMBE vocoder, and Phase 1/Phase 2 structure.
[^c4fm]: [Continuous 4 level FM](https://en.wikipedia.org/wiki/Continuous_4_level_FM) — Wikipedia, for C4FM's four deviation levels at 4800 symbols/s and its spectral equivalence to CQPSK.
