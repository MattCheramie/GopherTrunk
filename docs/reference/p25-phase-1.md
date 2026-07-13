---
slug: p25-phase-1
title: P25 Phase 1
entry_type: protocol
category: land-mobile-trunking
description: P25 Phase 1 is the FDMA air interface of Project 25, using C4FM modulation at 4800 baud and the IMBE vocoder in 12.5 kHz channels for North American public-safety radio.
keywords: P25 Phase 1, C4FM, CQPSK, IMBE, FDMA, 9600 bps, public safety, trunking, TSBK, NAC, trellis coded modulation, TIA-102
aka: [P25 Phase 1, P25 Phase I, Phase 1 P25]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: Project 25 }
  - { label: Access, value: FDMA }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: C4FM (4-level FSK) / CQPSK }
  - { label: Symbol rate, value: 4800 baud (9600 bps) }
  - { label: Vocoder, value: IMBE }
  - { label: GopherTrunk support, value: Decoded }
see_also: [project-25, p25-phase-2, p25-cai, c4fm, cqpsk, four-fsk, imbe, trellis-coded-modulation, network-access-code, tsbk, fdma, control-channel]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://www.cisa.gov/safecom/p25
---

**P25 Phase 1** is the first-generation air interface of [Project 25](/reference/project-25/),
using **[FDMA](/reference/fdma/)** — one conversation per 12.5 kHz channel — with
[C4FM](/reference/c4fm/) modulation and the [IMBE](/reference/imbe/) vocoder.[^wiki]
It is the [Common Air Interface](/reference/p25-cai/) baseline that every P25 system
supports, and the foundation that [Phase 2](/reference/p25-phase-2/) extends.

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Stacked 12.5 kHz FDMA channels each carrying one P25 Phase 1 call." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_p25-phase-1)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (12.5 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_p25-phase-1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>P25 Phase 1 is FDMA: each call occupies its own 12.5 kHz channel.</figcaption>
</figure>

## Overview

In Phase 1 each call occupies its own frequency. A trunked Phase 1 system uses a
dedicated [control channel](/reference/control-channel/) that broadcasts
[TSBK](/reference/tsbk/) messages to assign callers to voice channels; conventional
Phase 1 simply transmits on a fixed frequency without any trunking. It is the most
widely deployed P25 variant and the baseline that Phase 2 builds on. Because Phase 1
keeps one call per 12.5 kHz channel, its receiver is comparatively simple — a
4-level FSK demodulator running at 4800 symbols per second — which is why it remained
the standard control-channel format even on systems whose voice traffic later moved
to Phase 2.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | FDMA |
| Channel | 12.5 kHz |
| Modulation | [C4FM](/reference/c4fm/) (4-level FSK); [CQPSK](/reference/cqpsk/) on the linear path |
| Symbol rate | 4800 [baud](/reference/symbol-rate/) → 9600 bps |
| Vocoder | [IMBE](/reference/imbe/) (7.2 kbps incl. FEC) |
| Error correction | Golay, Hamming, Reed–Solomon, [trellis](/reference/trellis-coded-modulation/) (by field) |
| Squelch / addressing | [NAC](/reference/network-access-code/), talkgroup, source radio ID |

[C4FM](/reference/c4fm/) and [CQPSK](/reference/cqpsk/) are two ways of generating the
same on-air symbol constellation: C4FM shifts a carrier to four
[frequency](/reference/frequency-shift-keying/) deviations (±600 Hz, ±1800 Hz), while
CQPSK varies phase and amplitude to place energy in the same 6.25 kHz-shaped
spectrum. They are deliberately designed to be detected by the same receiver, so one
demodulator handles both transmit paths. Voice frames are protected by a layered FEC
stack — Golay and Hamming on the more sensitive bits, [Reed–Solomon](/reference/reed-solomon-code/)
and rate-1/2 [trellis-coded modulation](/reference/trellis-coded-modulation/) across
data blocks — so speech survives a fading mobile channel.

## History

Phase 1 documents were published by the [TIA](/reference/tia/) as part of the TIA-102
suite starting in the mid-1990s and saw broad public-safety adoption through the
2000s as agencies migrated from analog and proprietary systems under interoperability
and narrowbanding pressure.[^wiki][^cisa] Its selection of a spectrally efficient
4-level FSK and the IMBE vocoder let systems keep existing 12.5 kHz channel plans
while going digital.

## Deployment

Phase 1 underpins many statewide and municipal public-safety networks across North
America, often alongside Phase 2 voice channels on the same trunked system. It is
also common in conventional (non-trunked) form on interoperability and mutual-aid
channels, where a fixed NAC and frequency are all a radio needs to join. Encryption
(AES-256 or legacy DES) is widely layered on top of the Phase 1 air interface.

## Decoding it with GopherTrunk

GopherTrunk demodulates the C4FM symbols, reads the [NAC](/reference/network-access-code/)
and — on trunked systems — the [TSBK](/reference/tsbk/) control messages, recovers the
[IMBE](/reference/imbe/) frames through the FEC stack, and synthesises audio. The
[constellation](/reference/constellation-diagram/) and
[eye diagram](/reference/eye-diagram/) views help confirm a clean lock before
decoding. Clear and known-key traffic decode to voice; keyed-encrypted calls surface
only as metadata. See [Status](/status.html) for details.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the P25 Phase 1 FDMA air interface, C4FM/CQPSK modulation, the IMBE vocoder, the FEC stack, and TIA standardisation.
[^cisa]: [P25 (Project 25)](https://www.cisa.gov/safecom/p25) — CISA SAFECOM, for the TIA-102 standards suite and the public-safety interoperability and narrowbanding drivers behind Phase 1.
