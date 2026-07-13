---
slug: dab
title: Digital Audio Broadcasting (DAB / DAB+)
entry_type: protocol
category: broadcast
description: "DAB is the Eureka-147 digital radio standard using Band III OFDM with DQPSK to multiplex many stations into one ensemble; DAB+ adds AAC audio."
keywords: DAB, DAB+, Digital Audio Broadcasting, Eureka 147, Band III, OFDM, DQPSK, HE-AAC v2, MP2 audio, ensemble, multiplex, ETSI EN 300 401
aka: [DAB, DAB+, Digital Audio Broadcasting, Eureka 147]
autolink: true
infobox:
  - { label: Type, value: Terrestrial digital radio }
  - { label: Standards body, value: ETSI (WorldDAB) }
  - { label: Introduced, value: "1995" }
  - { label: Access, value: OFDM multiplex (COFDM ensemble) }
  - { label: Channel spacing, value: "~1.537 MHz block (Band III / L-band)" }
  - { label: Modulation, value: "π/4-DQPSK on OFDM subcarriers" }
  - { label: Vocoder, value: "MP2 (DAB) / HE-AAC v2 (DAB+)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, pi-4-dqpsk, subcarrier, frequency-modulation, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_audio_broadcasting
  - https://www.etsi.org/deliver/etsi_en/300400_300499/300401/
---

## Overview

**Digital Audio Broadcasting** (**DAB**) is a terrestrial digital radio standard,
developed under the European Eureka-147 project, that replaces one FM station per channel
with a multiplexed *ensemble* of many stations sharing a single ~1.5 MHz block in VHF
Band III.[^wiki] It transmits with coded [OFDM](/reference/ofdm/) using
[π/4-DQPSK](/reference/pi-4-dqpsk/) on each subcarrier, making it robust against the
multipath that plagues wideband signals in mobile reception. The upgraded **DAB+**
profile keeps the same air interface but swaps the original MP2 audio codec for the far
more efficient HE-AAC v2, roughly tripling the number of stations a multiplex can hold.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DAB ensemble shown as many closely spaced OFDM subcarriers within one frequency block, feeding a multiplex that carries several independent audio services." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="daba" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1">
    <line x1="30" y1="70" x2="30" y2="40"/><line x1="38" y1="70" x2="38" y2="40"/><line x1="46" y1="70" x2="46" y2="40"/><line x1="54" y1="70" x2="54" y2="40"/><line x1="62" y1="70" x2="62" y2="40"/><line x1="70" y1="70" x2="70" y2="40"/><line x1="78" y1="70" x2="78" y2="40"/><line x1="86" y1="70" x2="86" y2="40"/><line x1="94" y1="70" x2="94" y2="40"/><line x1="102" y1="70" x2="102" y2="40"/><line x1="110" y1="70" x2="110" y2="40"/><line x1="118" y1="70" x2="118" y2="40"/><line x1="126" y1="70" x2="126" y2="40"/><line x1="134" y1="70" x2="134" y2="40"/><line x1="142" y1="70" x2="142" y2="40"/><line x1="150" y1="70" x2="150" y2="40"/>
  </g>
  <line x1="24" y1="70" x2="160" y2="70" stroke="currentColor"/>
  <text x="92" y="86" text-anchor="middle" font-size="8" fill="currentColor">~1.5 MHz of OFDM subcarriers</text>
  <line x1="165" y1="55" x2="210" y2="55" stroke="currentColor" marker-end="url(#daba)"/>
  <rect x="215" y="35" width="90" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/>
  <text x="260" y="59" text-anchor="middle" font-size="8" fill="currentColor">ensemble mux</text>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="320" y="30" width="60" height="14" fill="none" stroke="currentColor"/><text x="350" y="40">Station 1</text>
    <rect x="320" y="48" width="60" height="14" fill="none" stroke="currentColor"/><text x="350" y="58">Station 2</text>
    <rect x="320" y="66" width="60" height="14" fill="none" stroke="currentColor"/><text x="350" y="76">Station 3…</text>
  </g>
</svg>
<figcaption>One DAB block packs many OFDM subcarriers into a single multiplex that carries several independent audio services.</figcaption>
</figure>

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | Coded OFDM (COFDM), multiple transmission modes |
| Subcarrier modulation | π/4-DQPSK (differential, no channel equaliser needed) |
| Block width | ~1.537 MHz (Mode I: 1536 subcarriers) |
| Bands | VHF Band III (~174–240 MHz); historically L-band |
| Audio | MP2 (DAB) or HE-AAC v2 (DAB+) |
| Error control | Convolutional coding; Reed–Solomon added in DAB+ |
| Network | Supports single-frequency networks (SFN) |

## History

Eureka-147 research began in the late 1980s, and DAB launched commercially from 1995,
with early adoption strongest in the UK, Germany, and the Nordic countries. The original
MP2 codec proved spectrally wasteful, so ETSI standardised **DAB+** in 2007 with HE-AAC v2
and stronger error protection. Most new deployments and receivers are DAB+, though many
multiplexes still carry legacy MP2 services for backward compatibility.

## Deployment

DAB/DAB+ is widely deployed across Europe and in parts of Asia-Pacific, and is mandated in
some markets for new car radios. Single-frequency networks let every transmitter in a
region share one frequency, improving coverage efficiency versus FM. Adoption is uneven
globally — North America chose the in-band [HD Radio](/reference/hd-radio/) approach
instead — so DAB coexists with FM rather than having replaced it.

## Decoding it with GopherTrunk

**GopherTrunk** does not decode DAB. It is a trunked land-mobile scanner (P25, DMR, NXDN,
TETRA and similar), and DAB is a broadcast multiplex outside that scope. DAB is,
however, a clean real-world example of the coded-[OFDM](/reference/ofdm/) and
[π/4-DQPSK](/reference/pi-4-dqpsk/) techniques GopherTrunk readers meet in other digital
systems, and general-purpose SDR tools (such as welle.io) can receive and decode a full
ensemble from a wideband dongle.

## Sources

[^wiki]: [Digital audio broadcasting](https://en.wikipedia.org/wiki/Digital_audio_broadcasting) — Wikipedia, for the Eureka-147 origin, Band III COFDM with DQPSK subcarriers, the ensemble/multiplex structure, and the MP2-to-HE-AAC change in DAB+.
