---
slug: d-star
title: D-STAR
entry_type: protocol
category: amateur-digital
description: D-STAR is an amateur digital-voice and data standard developed by the JARL and popularised by Icom, using GMSK modulation and an AMBE-family vocoder.
keywords: D-STAR, amateur digital voice, JARL, Icom, GMSK, AMBE, DV, DD mode, reflectors, G2 routing, DPlus, DExtra, DCS
aka: [D-STAR, DSTAR, DPlus, DExtra]
autolink: true
infobox:
  - { label: Type, value: Amateur digital voice + data }
  - { label: Developer, value: JARL (popularised by Icom) }
  - { label: Access, value: FDMA }
  - { label: Channel spacing, value: 6.25 kHz (DV) }
  - { label: Modulation, value: GMSK (4800 bps) }
  - { label: Vocoder, value: AMBE (DV mode) }
  - { label: GopherTrunk support, value: See Status }
see_also: [system-fusion-ysf, m17, dmr, gmsk, ambe, vocoder]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/D-STAR
  - https://www.jarl.com/d-star/
---

**D-STAR** (**Digital Smart Technologies for Amateur Radio**) is an amateur-radio
digital voice and data standard developed by the Japan Amateur Radio League (JARL)
and widely implemented by **Icom**. It uses [GMSK](/reference/gmsk/) modulation and an
[AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/) for its digital-voice
(DV) mode, and is best known for internet-linked repeaters that route calls by
amateur callsign anywhere in the world.[^wiki] D-STAR was the first of the three
mainstream amateur digital-voice systems — ahead of Yaesu's
[System Fusion](/reference/system-fusion-ysf/) and the open
[M17](/reference/m17/) — and it established the pattern the others followed: a narrow
digital channel plus an internet backbone.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 110" role="img" aria-label="D-STAR digital voice from radio to repeater to internet gateways and reflectors." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="63">radio</text>
    <rect x="150" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="190" y="63">repeater</text>
    <rect x="290" y="44" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="350" y="58">internet</text><text x="350" y="69" font-size="8">reflectors/gateways</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="100" y1="59" x2="149" y2="59" marker-end="url(#am_d-star)"/><line x1="230" y1="59" x2="289" y2="59" marker-end="url(#am_d-star)"/></g>
    <text x="65" y="30" font-size="8">GMSK · AMBE</text>
  </g>
  <defs><marker id="am_d-star" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>D-STAR carries digital voice and data and links repeaters worldwide via internet reflectors and callsign (G2) routing.</figcaption>
</figure>

## Overview

D-STAR has two air interfaces. The everyday one is **DV (Digital Voice)**: a
4800 bps [GMSK](/reference/gmsk/) stream in a 6.25 kHz channel that multiplexes
digital speech with a slow "D-STAR data" side channel of roughly 900–1200 bps —
enough for GPS position beacons, short text, and callsign routing without
interrupting the voice. A second interface, **DD (Digital Data)**, runs 128 kbps in a
150 kHz channel on the 23 cm (1.2 GHz) band and behaves like a slow Ethernet link; it
is rarely deployed compared with DV. The system's headline feature is not the radio
link at all but the **gateway network**: every DV frame carries four callsign fields
(the caller "MY", the destination "UR", and up to two repeater fields "RPT1/RPT2"),
and gateway software uses them to route a call over the internet to a distant
repeater or a **reflector** — a conference server that many repeaters connect to at
once.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Modulation | GMSK, 4800 bps (BT ≈ 0.5) |
| Channel spacing | 6.25 kHz (DV), 150 kHz (DD) |
| Vocoder | AMBE (DVSI AMBE 2020), 2400 bps voice + 1200 bps FEC |
| Voice/data split | ~3600 bps voice+FEC, remainder sync + slow data |
| DD mode | 128 kbps digital data, 23 cm band |
| FEC | Convolutional coding + interleaving on the header |
| Routing | Callsign (G2) routing; DPlus / DExtra / DCS reflectors |

## How the DV frame works

A DV transmission opens with a **header** that carries the flags and the four
callsigns. Because losing the header means losing the routing, it is heavily
protected — scrambled, convolutionally coded, and interleaved — then GMSK-modulated
ahead of the voice. The voice that follows is produced by the DVSI **AMBE 2020**
[vocoder](/reference/vocoder/) at 2400 bps, wrapped with 1200 bps of forward error
correction to survive fading, for roughly 3600 bps of the 4800 bps budget. The
remaining capacity carries frame sync and the slow-data sub-stream. Because the
vocoder is a proprietary DVSI product, D-STAR (like [DMR](/reference/dmr/) and YSF)
cannot be fully re-encoded in open software without a licensed AMBE chip or
library — one of the motivations behind [M17](/reference/m17/)'s royalty-free
[Codec 2](/reference/codec2/) choice.

## History

JARL specified D-STAR around 2001 under a Japanese government-funded research
program to modernise amateur radio, publishing it as an open protocol so any
manufacturer could implement it.[^jarl] In practice **Icom** built essentially all
of the commercial radios and repeaters, giving the standard a single-vendor feel
even though the specification itself is public. The vocoder is the one closed piece.
D-STAR reached the market in the mid-2000s and, being first, seeded the amateur
digital-voice ecosystem — including the low-cost **hotspot** movement (Raspberry-Pi
boards such as DVMega and openSPOT) that later carried DMR and Fusion as well.

## Deployment

D-STAR is used by amateur operators worldwide through DV repeaters, personal
hotspots, and networked reflectors. Reflector systems (DPlus, DExtra, DCS, and the
newer XLX multiprotocol reflectors) act as the internet backbone, and gateways bridge
RF repeaters into it. Coverage is strongest in Japan, North America, and Western
Europe. It competes for the same operators as amateur DMR and System Fusion; many
hotspots and reflectors now bridge all three, so the practical distinction is often
just which handheld a given operator owns.

## Decoding it with GopherTrunk

GopherTrunk is a receiver, and D-STAR's GMSK air interface is within reach of its DSP
front end; the link-layer header (callsigns, flags, and slow-data such as GPS) is
decodable because it is plain FEC-protected framing. The **voice payload is AMBE**,
which is a licensed proprietary vocoder, so GopherTrunk can recover and log the frame
metadata but does not reproduce speech without an external AMBE facility — the same
honest limit that applies to DMR and YSF audio. See
[Status](/status.html) for GopherTrunk's current D-STAR link-layer coverage.

## Sources

[^wiki]: [D-STAR](https://en.wikipedia.org/wiki/D-STAR) — Wikipedia, for the JARL-developed amateur digital-voice standard, its GMSK modulation, AMBE vocoder, DV/DD modes, and reflector networking.
[^jarl]: [JARL D-STAR](https://www.jarl.com/d-star/) — Japan Amateur Radio League, the originating body's overview of the D-STAR system, its open specification, and DV/DD structure.
