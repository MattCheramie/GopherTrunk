---
slug: funcube-dongle
title: FUNcube Dongle
entry_type: hardware
category: sdr-devices
description: "FUNcube Dongle is an early UK USB software-defined radio receiver created to receive satellite telemetry, predating the RTL-SDR."
keywords: FUNcube Dongle, FUNcube Dongle Pro, FUNcube Dongle Pro Plus, AMSAT-UK, Howard Long, satellite SDR, cubesat telemetry, early USB SDR
aka: [FUNcube Dongle, FCD, FUNcube Dongle Pro Plus]
autolink: true
infobox:
  - { label: Type, value: USB SDR receiver }
  - { label: Origin, value: "AMSAT-UK / Howard Long (UK)" }
  - { label: Introduced, value: "2010 (Pro), 2012 (Pro+)" }
  - { label: Range, value: "150 kHz – 240 MHz, 420 MHz – 1.9 GHz (Pro+)" }
  - { label: Bandwidth, value: ~192 kHz (Pro+) }
  - { label: TX, value: No (receive only) }
  - { label: Typical price, value: "~£125 (historic)" }
see_also: [software-defined-radio, rtl-sdr, airspy, antenna, gnss]
cite_urls:
  - https://en.wikipedia.org/wiki/Funcube_Dongle
  - http://www.funcubedongle.com/
---

**FUNcube Dongle** is an early UK USB
[software-defined radio](/reference/software-defined-radio/) receiver, created by
**Howard Long** under the AMSAT-UK amateur-satellite banner and named after the
**FUNcube** educational cubesat whose telemetry it was built to receive.[^wiki] Launched
in 2010 — before hobbyists discovered that cheap DVB-T dongles could stream raw IQ — it
was one of the first affordable, self-contained SDRs aimed squarely at ordinary computer
users, and it helped seed the SDR hobby that the [RTL-SDR](/reference/rtl-sdr/) would
later make ubiquitous.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the FUNcube Dongle Pro Plus showing two segments, roughly 150 kHz to 240 MHz and 420 MHz to 1.9 GHz, on an axis from 0 to about 2 gigahertz, with a gap between them." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="130" y="86">0.5 GHz</text><text x="230" y="86">1 GHz</text><text x="430" y="86">2 GHz</text></g>
  <rect x="32" y="40" width="48" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <rect x="116" y="40" width="298" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">FUNcube Dongle Pro+ coverage (with a gap)</text>
</svg>
<figcaption>The Pro+ reaches from LF up to 240 MHz and again from 420 MHz to 1.9 GHz, with a gap in between — receive only.</figcaption>
</figure>

## Overview

The FUNcube Dongle was designed for a concrete mission: giving schools and radio
amateurs an easy way to receive **satellite telemetry** from AMSAT-UK's FUNcube
spacecraft (FUNcube-1 / AO-73). Unlike the repurposed TV tuners that came after it, the
FCD was purpose-built as a receiver — a small USB stick presenting itself as a **USB
audio device**, so its demodulated IQ arrived through the operating system's standard
sound interface at up to ~192 kHz, needing no special driver. That plug-and-play
simplicity, at a time when SDR otherwise meant expensive or DIY hardware, was its appeal.

## Variants

- **FUNcube Dongle Pro (2010).** The original. It targeted the VHF/UHF amateur-satellite
  bands, covering roughly **64 MHz – 1.7 GHz**, and established the USB-audio-device
  design.
- **FUNcube Dongle Pro+ (2012).** The improved and better-known model. It widened
  coverage dramatically — about **150 kHz – 240 MHz** and **420 MHz – 1.9 GHz**, with a
  gap between the two segments — added input filtering and a TCXO for better frequency
  stability, and kept the ~192 kHz capture width. The lower-frequency reach let it work
  HF and the [GNSS](/reference/gnss/) and satellite bands the hobby cared about.

Both are **receive-only**, drew their power from the USB port, and used an SMA-style
antenna connector on later revisions.

## In practice

By modern standards the FUNcube Dongle's **~192 kHz** of bandwidth is narrow — an
[RTL-SDR](/reference/rtl-sdr/) captures more than ten times as much, and an
[Airspy](/reference/airspy/) far more — but for its intended job of demodulating a single
satellite's narrowband telemetry that width is ample, and the dedicated receiver design
gave it cleaner performance and better stability than the earliest RTL-SDR dongles. It
was priced around **£125**, well above a later RTL-SDR but reasonable for a
purpose-built receiver of its era. Its main historical significance is as a *bridge*
device: it proved there was a real amateur market for a cheap, simple, computer-connected
SDR just before the RTL-SDR discovery made such a thing nearly free.

## Relevance to GopherTrunk

The FUNcube Dongle predates and is largely superseded by the RTL-SDR generation
GopherTrunk targets, and its narrow ~192 kHz capture is far too small to channelise the
multiple wideband control channels a trunking system spreads across a band — so it is not
a practical front end for GopherTrunk. It matters here as **context**: it is an early
milestone in the story of affordable SDR, the receiver niche the cheaper, wider RTL-SDR
soon filled for scanning and trunk-tracking. For decoding, use a modern
[RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) instead.

## Sources

[^wiki]: [Funcube Dongle](https://en.wikipedia.org/wiki/Funcube_Dongle) — Wikipedia, on the FUNcube Dongle's origin, variants and coverage.
[^home]: [funcubedongle.com](http://www.funcubedongle.com/) — the official project site, for the Pro and Pro+ specifications.
