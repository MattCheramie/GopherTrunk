---
slug: meteor-lrpt
title: Meteor-M LRPT
entry_type: protocol
category: satellite-gnss
description: "Meteor-M LRPT is a digital 137 MHz weather-satellite image downlink using QPSK with convolutional coding and Viterbi decoding, the digital successor to analog APT."
keywords: Meteor-M, LRPT, Low Rate Picture Transmission, weather satellite, 137 MHz, QPSK, Viterbi, CCSDS, Meteor-M N2, digital weather image, Roscosmos
aka: [LRPT, Meteor LRPT, Meteor-M]
autolink: true
infobox:
  - { label: Type, value: Digital weather-image downlink }
  - { label: Standards body, value: Roscosmos / Roshydromet (CCSDS) }
  - { label: Introduced, value: "2014 (Meteor-M N2)" }
  - { label: Access, value: Continuous downlink, single carrier }
  - { label: Channel spacing, value: "~137 MHz VHF" }
  - { label: Modulation, value: "QPSK, ~72 kbps" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [qpsk, viterbi-algorithm, noaa-apt, goes-hrit, convolutional-code, forward-error-correction, helical-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Meteor_(satellite)
  - https://en.wikipedia.org/wiki/Low-rate_picture_transmission
---

**Meteor-M LRPT** (**Low Rate Picture Transmission**) is the digital weather-image
downlink carried by Russia's Meteor-M polar-orbiting satellites near 137 MHz. It is the
modern replacement for analog [NOAA APT](/reference/noaa-apt/): instead of an amplitude
raster, LRPT sends compressed image data as a [QPSK](/reference/qpsk/) bitstream protected
by a [convolutional code](/reference/convolutional-code/) that the receiver undoes with a
[Viterbi decoder](/reference/viterbi-algorithm/).[^lrpt] The payoff is sharper, calibrated
imagery from the same simple 137 MHz ground station.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The Meteor LRPT receive chain: QPSK demodulation, then Viterbi decoding of the convolutional code, then CCSDS packet assembly into an image." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mlrpt" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none">
    <rect x="20" y="45" width="90" height="40"/>
    <rect x="140" y="45" width="90" height="40"/>
    <rect x="260" y="45" width="90" height="40"/>
    <rect x="380" y="45" width="60" height="40"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="65" y="62">QPSK</text><text x="65" y="75">demod</text>
    <text x="185" y="62">Viterbi</text><text x="185" y="75">decode</text>
    <text x="305" y="62">CCSDS</text><text x="305" y="75">packets</text>
    <text x="410" y="68">image</text>
  </g>
  <g stroke="currentColor"><line x1="110" y1="65" x2="138" y2="65" marker-end="url(#mlrpt)"/><line x1="230" y1="65" x2="258" y2="65" marker-end="url(#mlrpt)"/><line x1="350" y1="65" x2="378" y2="65" marker-end="url(#mlrpt)"/></g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">137 MHz QPSK downlink → soft bits → error-corrected packets → picture</text>
</svg>
<figcaption>LRPT recovers soft QPSK symbols, Viterbi-decodes the convolutional FEC, and reassembles CCSDS packets into a weather image.</figcaption>
</figure>

## Overview

LRPT flies on the Meteor-M "N2" series operated by Roscosmos and Roshydromet, the first of
which reached orbit in 2014. Like APT it broadcasts continuously on a 137 MHz VHF carrier
that any modest ground station can hear, but everything above the antenna is digital: the
image is delivered as CCSDS space-packets carrying compressed pixels, so a good pass yields
a clean, geometrically corrected picture rather than the noise-streaked grey scale of an
analog signal.[^lrpt]

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | ~137 MHz VHF |
| Modulation | [QPSK](/reference/qpsk/) (some satellites/modes OQPSK) |
| Symbol/bit rate | ~72 kbps (72k or 80k symbol variants) |
| Error correction | Convolutional r=1/2 + Viterbi, with outer coding/interleaving |
| Framing | CCSDS virtual channels / space packets |
| Content | Multiple MSU-MR imager channels, JPEG-like compression |

The [forward error correction](/reference/forward-error-correction/) is what lets LRPT work
at the same weak-signal margins as APT: soft QPSK decisions feed a
[Viterbi](/reference/viterbi-algorithm/) decoder that corrects the bit errors a fading
137 MHz pass inevitably produces, and the CCSDS layer packages the corrected bits into the
imager channels.

## History

Meteor-M LRPT arrived as the digital counterpart to the aging analog fleet, giving amateurs
and forecasters a higher-quality 137 MHz image source. Early Meteor-M satellites suffered
outages and a debris strike, but the series has been sustained, and open-source
demodulators made LRPT a staple of the SDR weather-satellite hobby alongside its western
sibling, the geostationary [GOES HRIT](/reference/goes-hrit/).[^meteor]

## Deployment

LRPT is a receive-only broadcast with no user uplink. The ground station is essentially the
same as for APT — a right-hand-circularly-polarised
**[QFH or turnstile antenna](/reference/helical-antenna/)**, a 137 MHz
[SDR](/reference/software-defined-radio/), and pass-prediction software — with the demod and
Viterbi decode done in software.

## Decoding it with GopherTrunk

GopherTrunk does not decode Meteor-M LRPT. It is a terrestrial land-mobile trunking scanner,
whereas LRPT is a satellite image protocol with CCSDS framing and a QPSK/Viterbi chain that
shares nothing with the land-mobile voice systems GopherTrunk implements. It is documented
here as the digital successor to [NOAA APT](/reference/noaa-apt/) and a rewarding
SDR-decoding project in its own right.

## Sources

[^lrpt]: [Low-rate picture transmission](https://en.wikipedia.org/wiki/Low-rate_picture_transmission) — Wikipedia, for the LRPT QPSK downlink, convolutional coding, and CCSDS image format.
[^meteor]: [Meteor (satellite)](https://en.wikipedia.org/wiki/Meteor_(satellite)) — Wikipedia, for the Meteor-M satellite series that carries LRPT.
