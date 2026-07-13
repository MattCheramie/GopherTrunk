---
slug: isdb-t
title: ISDB-T
entry_type: protocol
category: broadcast
description: "ISDB-T is the Japanese and Brazilian digital terrestrial television standard, using segmented band OFDM so one channel can serve fixed HDTV and handheld receivers at once."
keywords: ISDB-T, ISDB-Tb, SBTVD, digital terrestrial television, OFDM, BST-OFDM, segmented transmission, one-seg, 13 segments, ARIB, Japan, Brazil
aka: [ISDB-T, ISDB-Tb, SBTVD]
autolink: true
infobox:
  - { label: Type, value: Digital terrestrial television }
  - { label: Standards body, value: "ARIB (Japan); ABNT (Brazil)" }
  - { label: Introduced, value: "2003 (Japan), 2007 (Brazil)" }
  - { label: Access, value: Broadcast (one-to-many) }
  - { label: Channel spacing, value: "6 MHz (13 segments)" }
  - { label: Modulation, value: "BST-OFDM, QPSK–64-QAM subcarriers" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [ofdm, dvb-t, atsc-1]
cite_urls:
  - https://en.wikipedia.org/wiki/ISDB-T
  - https://www.arib.or.jp/english/std_tr/broadcasting/std-b31.html
---

**ISDB-T** (Integrated Services Digital Broadcasting — Terrestrial) is the digital
terrestrial television standard developed in Japan and adopted, in a modified form,
across South America.[^wiki] Like [DVB-T](/reference/dvb-t/) it uses coded
[OFDM](/reference/ofdm/), but its defining feature is **band-segmented transmission**:
the 6 MHz channel is divided into thirteen independently configurable segments, so a
single broadcast can simultaneously serve fixed HDTV receivers and low-power handheld
devices.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A 6 MHz television channel divided into thirteen OFDM segments, with the centre segment highlighted as the one-seg mobile service." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="105" x2="445" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1">
    <rect x="30" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="60" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="90" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="120" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="150" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="180" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="210" y="45" width="30" height="60" fill="currentColor" fill-opacity="0.35"/>
    <rect x="240" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="270" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="300" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="330" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="360" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
    <rect x="390" y="55" width="30" height="50" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <text x="225" y="38" text-anchor="middle" font-size="9" fill="currentColor">centre segment = "one-seg" mobile service</text>
  <text x="225" y="128" text-anchor="middle" font-size="9" fill="currentColor">13 OFDM segments across one 6 MHz channel</text>
</svg>
<figcaption>ISDB-T splits the channel into 13 segments; the central segment can be decoded alone by low-power handhelds ("one-seg").</figcaption>
</figure>

## Overview

Each of the thirteen ISDB-T segments can carry its own OFDM parameters — QPSK, 16-QAM,
or 64-QAM subcarriers, its own code rate and interleaving. Broadcasters typically
group twelve segments into a robust HDTV service and reserve the central segment as a
self-contained, rugged **one-seg** stream that battery-powered phones and car
receivers can decode without tuning the whole channel. A deep time interleaver, longer
than in DVB-T, gives ISDB-T strong resistance to the impulsive noise and fading of
mobile reception.

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | BST-OFDM (13 segments) |
| Modes | 2K / 4K / 8K FFT |
| Subcarrier modulation | DQPSK, QPSK, 16-QAM, 64-QAM (per segment) |
| Inner code | Convolutional |
| Outer code | Reed–Solomon (204,188) |
| Guard interval | 1/4, 1/8, 1/16, 1/32 |
| Payload | MPEG-2 transport stream (MPEG-2 / H.264 video) |
| Mobile service | Central "one-seg" segment |

## History

Japan's ARIB standardised ISDB-T (STD-B31) and began broadcasts in 2003.[^arib]
Brazil adopted a modified variant, ISDB-Tb (also called SBTVD), in 2007, switching the
video codec to H.264 and one-seg to a full-motion mode; this Brazilian profile was
subsequently taken up across most of South America and in the Philippines. The
segmented design gave ISDB-T built-in mobile TV years before other systems bolted it
on.

## Deployment

ISDB-T is the terrestrial DTV system of Japan, Brazil, and much of Central and South
America, plus the Philippines, Botswana, and others — the third major terrestrial
family alongside DVB-T and [ATSC](/reference/atsc-1/). Its one-seg service made mobile
digital television commonplace in Japan and Brazil long before smartphone streaming.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode ISDB-T; segmented-OFDM television is outside its
land-mobile trunking scope. The 6 MHz signal can be captured by a wideband
[software-defined radio](/reference/software-defined-radio/) and processed in
dedicated ISDB-T tools, but not by a narrowband dongle in a single pass. The European
counterpart is documented under [DVB-T](/reference/dvb-t/).

## Sources

[^wiki]: [ISDB-T](https://en.wikipedia.org/wiki/ISDB-T) — Wikipedia, for the ISDB-T system, its band-segmented OFDM, the one-seg service, and regional variants.
[^arib]: [ARIB STD-B31](https://www.arib.or.jp/english/std_tr/broadcasting/std-b31.html) — ARIB, the primary Japanese standard defining ISDB-T transmission.
