---
slug: noaa-apt
title: NOAA APT
entry_type: protocol
category: satellite-gnss
description: "NOAA APT (Automatic Picture Transmission) is an analog 137 MHz weather-satellite image mode that sends a line-by-line raster on an AM subcarrier, decodable with a basic SDR and QFH antenna."
keywords: NOAA APT, Automatic Picture Transmission, weather satellite, 137 MHz, AM subcarrier, POES, NOAA-15, NOAA-18, NOAA-19, weather fax, QFH antenna
aka: [APT, Automatic Picture Transmission]
autolink: true
infobox:
  - { label: Type, value: Analog weather-image downlink }
  - { label: Standards body, value: NOAA / NASA (POES) }
  - { label: Introduced, value: "1960s (TIROS/NOAA)" }
  - { label: Access, value: Continuous downlink, single carrier }
  - { label: Channel spacing, value: "~137 MHz VHF" }
  - { label: Modulation, value: "2400 Hz AM subcarrier, FM on carrier" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [amplitude-modulation, helical-antenna, meteor-lrpt, goes-hrit, frequency-modulation, subcarrier, sstv]
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_picture_transmission
  - https://en.wikipedia.org/wiki/Polar_Operational_Environmental_Satellites
---

**NOAA APT** (**Automatic Picture Transmission**) is the long-running analog weather-image
mode broadcast by NOAA's polar-orbiting (POES) satellites near 137 MHz. It sends a
continuous **line-by-line raster**: each video line amplitude-modulates a 2400 Hz
[subcarrier](/reference/subcarrier/), and that subcarrier in turn frequency-modulates the
VHF carrier — so the picture rides as [AM](/reference/amplitude-modulation/) inside an
[FM](/reference/frequency-modulation/) downlink.[^apt] The result is a slow-scan grey-scale
strip of the Earth beneath the satellite's path.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An APT line: a 2400 Hz subcarrier whose amplitude tracks the image brightness, carrying a sync burst then two image channels per line." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="20" font-size="9" fill="currentColor">one APT line (2 lines/second) →</text>
  <rect x="20" y="40" width="30" height="40" fill="currentColor" fill-opacity="0.35" stroke="currentColor"/>
  <rect x="50" y="40" width="170" height="40" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
  <rect x="220" y="40" width="30" height="40" fill="currentColor" fill-opacity="0.35" stroke="currentColor"/>
  <rect x="250" y="40" width="170" height="40" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="35" y="63">sync A</text><text x="135" y="63">image A (visible)</text><text x="235" y="63">sync B</text><text x="335" y="63">image B (IR)</text></g>
  <path d="M20 120 q10 -20 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
  <text x="230" y="150" text-anchor="middle" font-size="9" fill="currentColor">brightness = amplitude of the 2400 Hz subcarrier</text>
</svg>
<figcaption>Each APT line carries a sync burst and image data on a 2400 Hz AM subcarrier; two channels (visible and infrared) are interleaved per line.</figcaption>
</figure>

## Overview

APT descends from the earliest TIROS weather satellites of the 1960s and has flown on
every NOAA polar orbiter since, chosen precisely because it can be received with the
simplest possible ground station. Two 137 MHz APT-capable satellites remained active into
the 2020s — NOAA-15, NOAA-18, and NOAA-19 (NOAA-18 was decommissioned in 2025) — each
passing overhead a few times a day. The signal is unencrypted and low rate by design, so
that anyone from a school to a fishing boat could pull down a fresh cloud image.[^poes]

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | ~137 MHz VHF, ~34 kHz wide |
| Carrier modulation | FM |
| Image modulation | AM on a 2400 Hz subcarrier |
| Line rate | 2 lines/second (120 lines/minute) |
| Word rate | 4160 words/second |
| Content | Two channels/line: visible + infrared, plus sync and telemetry |

An APT frame interleaves two AVHRR imager channels (typically one visible and one
infrared) with sync bars, minute markers, and a telemetry wedge used to calibrate the
grey scale. Because the picture is amplitude-carried, receiver AGC and a clean FM demod
matter for image quality.

## History

The mode has been remarkably stable across six decades, outliving the analog technology of
its origin because of its accessibility. Its digital successor,
[Meteor-M LRPT](/reference/meteor-lrpt/), and the geostationary
[GOES HRIT](/reference/goes-hrit/) service deliver sharper, calibrated imagery, but APT's
simplicity kept it valuable to hobbyists and educators to the end of the legacy POES
fleet.[^apt]

## Deployment

APT is a receive-only public service — there is no user uplink. A ground station needs a
right-hand-circularly-polarised antenna (typically a
**[quadrifilar helix (QFH)](/reference/helical-antenna/)** or a turnstile) to match the
satellite's polarisation, a 137 MHz [SDR](/reference/software-defined-radio/), and
free software to turn the audio into an image and apply map overlays.

## Decoding it with GopherTrunk

GopherTrunk does not decode APT: it is a land-mobile trunking scanner, and APT is an analog
image raster, not a voice or trunking protocol. That said, APT is one of the friendliest
possible first satellite-reception projects — a $30 SDR dongle, a home-made QFH, and a pass
prediction are enough to receive it — which is why it earns an entry in this guide even
though it sits outside GopherTrunk's decode chain. It is closely related to the amateur
[SSTV](/reference/sstv/) slow-scan modes in spirit, and to its digital replacement
[Meteor-M LRPT](/reference/meteor-lrpt/).

## Sources

[^apt]: [Automatic picture transmission](https://en.wikipedia.org/wiki/Automatic_picture_transmission) — Wikipedia, for the APT format, the 2400 Hz AM subcarrier, and the per-line channel structure.
[^poes]: [Polar Operational Environmental Satellites](https://en.wikipedia.org/wiki/Polar_Operational_Environmental_Satellites) — Wikipedia, for the NOAA POES fleet that carries APT.
