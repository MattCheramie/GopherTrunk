---
slug: sdr-sharp
title: SDR# (SDRSharp)
entry_type: technology
category: sdr-software
description: "SDR# (SDRSharp) is a popular Windows software-defined-radio receiver application known for its plugin ecosystem and smooth waterfall display."
keywords: SDR#, SDRSharp, Airspy software, Windows SDR receiver, waterfall, plugin, RTL-SDR receiver, .NET SDR
aka: [SDR#, SDRSharp, SDR Sharp]
autolink: true
infobox:
  - { label: Type, value: Desktop SDR receiver app }
  - { label: Platform, value: "Windows (.NET)" }
  - { label: Examples, value: "RTL-SDR, Airspy, HackRF front ends" }
see_also: [software-defined-radio, rtl-sdr, gqrx, sdrangel, waterfall-display, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/SDR_Sharp
  - https://airspy.com/download/
---

**SDR#** (pronounced "SDR Sharp", stylized *SDRSharp*) is a widely used Windows
[software-defined radio](/reference/software-defined-radio/) receiver application, written in
.NET and maintained by the Airspy team.[^proj] It presents a live spectrum and
[waterfall display](/reference/waterfall-display/), a tuning dial, and demodulators for the
common analog modes, and it is a common first application for newcomers pairing a cheap
[RTL-SDR](/reference/rtl-sdr/) dongle with a PC. Its clean interface and rich plugin
ecosystem made it one of the most recognizable SDR programs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An SDR# style receiver window: a spectrum plot above a scrolling waterfall, with a tuning marker, driven by an SDR front end." xmlns="http://www.w3.org/2000/svg">
  <rect x="12" y="10" width="436" height="120" rx="6" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M22 60 Q80 58 120 56 T190 30 Q210 22 230 30 T300 56 Q360 58 438 60" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <line x1="210" y1="18" x2="210" y2="126" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="210" y="16" text-anchor="middle" font-size="8" fill="currentColor">tuned signal</text>
  <g fill="currentColor" fill-opacity="0.14"><rect x="22" y="72" width="416" height="54"/></g>
  <g stroke="currentColor" stroke-width="0.8" opacity="0.5">
    <line x1="180" y1="72" x2="188" y2="126"/><line x1="210" y1="72" x2="210" y2="126"/><line x1="240" y1="72" x2="232" y2="126"/>
  </g>
  <text x="70" y="88" font-size="8" fill="currentColor">spectrum</text>
  <text x="70" y="112" font-size="8" fill="currentColor">waterfall</text>
</svg>
<figcaption>SDR# shows a real-time spectrum over a scrolling waterfall; a movable marker selects the tuned signal for demodulation.</figcaption>
</figure>

## How it works

SDR# reads a stream of [IQ](/reference/iq-data/) samples from a front end, runs a fast
Fourier transform for the spectrum and [waterfall](/reference/waterfall-display/), and applies
a selected demodulator to the slice of spectrum under the tuning marker. Users pick the
mode (WFM, NFM, AM, [single sideband](/reference/single-sideband/), CW, raw), set the
bandwidth of the receive filter, and adjust gain; the audio is played out the sound card or
piped to another program via a virtual audio cable.

Front-end support is provided through a **source plugin** for each radio family — RTL-SDR,
[Airspy](/reference/airspy/), [HackRF](/reference/hackrf/), and others — that hands raw
samples to the core. The rest of SDR#'s reach comes from its **plugin API**: third-party
add-ons implement digital-mode decoders, frequency scanners, noise reduction, frequency-manager
databases, and more, docking into the main window. This plugin model let a single receiver
front end become a platform for many niche tools without those tools reimplementing the
tuning, FFT, and audio plumbing.

The application is closed-source but free to download, and it has long been the flagship
software for Airspy hardware, tuned to get the best out of those receivers.

## Relevance to SDR

For general-purpose listening — broadcast FM, air band, ham bands, utility signals — SDR# is
one of the most-used receivers in the hobby, especially on Windows where alternatives like
[SDRangel](/reference/sdrangel/) and [Gqrx](/reference/gqrx/) (the latter more common on
Linux) compete. Its accessibility made it a standard on-ramp: plug in a dongle, run the
Zadig driver installer, open SDR#, and see signals within minutes. Plugins extend it toward
digital decoding, but SDR# itself is fundamentally an analog-mode receiver and spectrum
browser rather than a trunking decoder.

**GopherTrunk** is unrelated software and shares no code with SDR#. GopherTrunk is a
headless, cross-platform, pure-Go trunking scanner that ingests IQ directly and decodes
digital voice/control protocols (P25, DMR, NXDN, TETRA, and more) on its own; SDR# is a
Windows GUI receiver aimed at manual tuning and listening. They can, however, use the same
hardware — an RTL-SDR or Airspy that works in SDR# is the same class of front end GopherTrunk
consumes. Someone might use SDR# to eyeball a control channel's frequency on the waterfall,
then point GopherTrunk at that frequency to actually follow the trunked system.

## Sources

[^proj]: [SDR# and Airspy downloads](https://airspy.com/download/) — the official distribution page from the Airspy team, and the [Wikipedia article on SDR Sharp](https://en.wikipedia.org/wiki/SDR_Sharp), covering the application, its plugin ecosystem, and supported hardware.
