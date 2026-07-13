---
slug: goes-hrit
title: GOES HRIT
entry_type: protocol
category: satellite-gnss
description: "GOES HRIT is the ~1.7 GHz L-band BPSK downlink from NOAA's geostationary GOES weather satellites, delivering full-disk imagery with Viterbi and Reed–Solomon error correction."
keywords: GOES HRIT, GOES, geostationary weather satellite, 1694 MHz, L-band, BPSK, Reed-Solomon, Viterbi, LRIT, EMWIN, DCS, NOAA, full disk imagery
aka: [HRIT, GOES-R HRIT]
autolink: true
infobox:
  - { label: Type, value: Digital weather-image downlink (GEO) }
  - { label: Standards body, value: NOAA / NASA (CCSDS) }
  - { label: Introduced, value: "2017 (GOES-R series)" }
  - { label: Access, value: Continuous downlink, single carrier }
  - { label: Channel spacing, value: "~1694.1 MHz L-band" }
  - { label: Modulation, value: "BPSK, ~927 ksym/s" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [bpsk, reed-solomon-code, viterbi-algorithm, meteor-lrpt, noaa-apt, forward-error-correction, parabolic-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/GOES-R_series
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

**GOES HRIT** (**High Rate Information Transmission**) is the digital image-and-data
downlink from NOAA's geostationary GOES weather satellites, broadcast near 1694.1 MHz in
L-band. Unlike the polar 137 MHz systems, a GOES satellite hovers over a fixed point on
the equator, so its [BPSK](/reference/bpsk/) downlink is always present and a ground
station can use a small fixed dish. The bitstream is heavily protected — a
[convolutional code](/reference/convolutional-code/) with [Viterbi](/reference/viterbi-algorithm/)
decoding on the inside and a [Reed–Solomon code](/reference/reed-solomon-code/) on the
outside — so the imagery arrives essentially error-free.[^goes]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A geostationary GOES satellite continuously broadcasts a BPSK HRIT downlink to a small fixed dish, protected by Viterbi and Reed-Solomon coding." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ghrit" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="230" cy="35" r="10" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">GOES at GEO (~35 786 km), fixed over equator</text>
  <path d="M215 45 L120 130 M225 45 L150 132 M235 45 L180 133" stroke="currentColor" stroke-opacity="0.7"/>
  <line x1="230" y1="45" x2="150" y2="128" stroke="currentColor" marker-end="url(#ghrit)"/>
  <path d="M120 150 a40 22 0 0 1 60 0" fill="none" stroke="currentColor"/>
  <text x="150" y="167" text-anchor="middle" font-size="9" fill="currentColor">small fixed dish + LNA</text>
  <g font-size="8" fill="currentColor"><text x="300" y="95">1694 MHz BPSK</text><text x="300" y="110">Viterbi + Reed–Solomon</text></g>
</svg>
<figcaption>GOES HRIT is an always-on 1.7 GHz BPSK downlink from a fixed geostationary satellite, receivable with a small dish and heavily error-corrected.</figcaption>
</figure>

## Overview

The current HRIT service flies on the GOES-R generation (GOES-16 through GOES-19), which
became operational from 2017 onward and replaced the older, slower LRIT service. From
geostationary orbit a single GOES satellite images an entire hemisphere, and HRIT relays a
rebroadcast of that imagery along with the EMWIN text/weather-product stream and the DCS
data-collection reports from remote ground sensors.[^goes]

## Technical characteristics

| Property | Value |
|----------|-------|
| Orbit | Geostationary (~35 786 km) |
| Downlink | ~1694.1 MHz L-band |
| Modulation | [BPSK](/reference/bpsk/) |
| Symbol rate | ~927 ksym/s (GOES-R HRIT) |
| Error correction | Inner Viterbi (r=1/2) + outer Reed–Solomon, with interleaving |
| Framing | CCSDS virtual channels |
| Content | Full-disk/ABI imagery, EMWIN, DCS |

The concatenated coding is the classic deep-space recipe: the inner
[Viterbi](/reference/viterbi-algorithm/) decoder cleans up most random bit errors, and the
outer [Reed–Solomon](/reference/reed-solomon-code/) block code mops up the residual bursts,
so even a marginal dish delivers a clean picture.[^rs]

## History

GOES has carried a low-rate image rebroadcast for decades (WEFAX, then LRIT); the GOES-R
series upgraded it to the faster, higher-resolution HRIT service and modern CCSDS framing.
Together with the polar [Meteor-M LRPT](/reference/meteor-lrpt/) and the legacy analog
[NOAA APT](/reference/noaa-apt/), it rounds out the set of directly receivable civilian
weather-satellite downlinks.

## Deployment

HRIT is a receive-only broadcast. Because the satellite is fixed in the sky, a modest
prime-focus [parabolic dish](/reference/parabolic-antenna/) (or a specialised grid/loop
feed) with a 1.7 GHz [low-noise amplifier](/reference/low-noise-amplifier/) and an
[SDR](/reference/software-defined-radio/) can lock the signal continuously, and open-source
software handles the BPSK demod, Viterbi/Reed–Solomon decode, and image assembly.

## Decoding it with GopherTrunk

GopherTrunk does not decode GOES HRIT. It is a VHF/UHF land-mobile trunking scanner, while
HRIT is an L-band satellite imagery protocol with CCSDS framing and a Viterbi/Reed–Solomon
chain unrelated to the land-mobile voice systems GopherTrunk targets. It is listed here as
the geostationary counterpart to [Meteor-M LRPT](/reference/meteor-lrpt/) and
[NOAA APT](/reference/noaa-apt/), and as a well-known SDR weather-satellite project.

## Sources

[^goes]: [GOES-R series](https://en.wikipedia.org/wiki/GOES-R_series) — Wikipedia, for the GOES-R satellites and the HRIT downlink, EMWIN, and DCS services.
[^rs]: [Reed–Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, for the outer block code used in the concatenated HRIT coding scheme.
