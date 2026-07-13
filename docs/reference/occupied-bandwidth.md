---
slug: occupied-bandwidth
title: Occupied bandwidth
entry_type: term
category: rf-fundamentals
description: Occupied bandwidth is the spectral width containing a defined fraction — conventionally 99% — of a signal's total transmitted power, and underpins emission designators and channel planning.
keywords: occupied bandwidth, 99% power bandwidth, necessary bandwidth, emission designator, OBW, channel spacing, spectrum mask, ITU emission designation
aka: [occupied bandwidth, OBW, 99% bandwidth]
autolink: true
infobox:
  - { label: Type, value: Spectral-width measure }
  - { label: Definition, value: "Band holding 99% of total power" }
  - { label: Related, value: "Necessary bandwidth; emission designator" }
see_also: [bandwidth, guard-band, spectral-efficiency, power-spectral-density, spurious-emissions, pulse-shaping]
cite_urls:
  - https://en.wikipedia.org/wiki/Bandwidth_(signal_processing)
  - https://en.wikipedia.org/wiki/Necessary_bandwidth
---

**Occupied bandwidth (OBW)** is the width of the frequency band that contains a specified fraction of
a signal's total transmitted power — by long-standing ITU convention, **99%**, with **0.5% left over
above and 0.5% below**.[^wiki] Unlike the vague "bandwidth" of a signal, OBW is a precise, measurable
number, which is what makes it useful for channel planning, [guard-band](/reference/guard-band/)
sizing, and the emission designators regulators assign to every licensed transmission.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A power spectral density curve whose area is shaded; vertical lines mark the frequencies below which 0.5% and above which 0.5% of the power lies, and the span between them, holding 99% of the power, is the occupied bandwidth." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" stroke="none">
    <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-opacity="0.4"/>
    <path d="M40 115 C 120 115 150 35 235 35 C 320 35 350 115 430 115 Z" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
    <line x1="120" y1="30" x2="120" y2="120" stroke="currentColor" stroke-dasharray="3 2"/>
    <line x1="350" y1="30" x2="350" y2="120" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="60" y="130" fill-opacity="0.7">0.5%</text>
    <text x="380" y="130" fill-opacity="0.7">0.5%</text>
    <line x1="120" y1="24" x2="350" y2="24" stroke="currentColor" stroke-opacity="0.6"/>
    <text x="175" y="18">occupied bandwidth (99%)</text>
    <text x="205" y="70">99% of P</text>
  </g>
</svg>
<figcaption>Occupied bandwidth is the span holding 99% of the total power, with 0.5% of the power excluded on each edge of the spectrum.</figcaption>
</figure>

## How it works

To find OBW you integrate the signal's [power spectral density](/reference/power-spectral-density/)
across frequency to get the total power, then find the two frequencies below which 0.5% and above
which 0.5% of that total lies. The distance between them is the occupied bandwidth — the band that,
by definition, carries the central 99%. Because the measure is defined on the actual power
distribution, it captures the real spectral footprint including modulation sidebands and
[pulse-shaping](/reference/pulse-shaping/) skirts, not just an idealized main lobe.

A closely related regulatory term is **necessary bandwidth**: the *minimum* width sufficient to carry
the information at the required rate and quality for a given emission class. OBW is what a transmitter
actually occupies; necessary bandwidth is what the class of emission theoretically needs. Both feed
the **ITU emission designator** — a code such as `16K0F3E` (16.0 kHz, F = frequency modulation,
3 = analog telephony, E = telephony) whose leading field is the bandwidth. That designator is how a
license precisely specifies the width and nature of an emission.

## In practice

Occupied bandwidth is the number that sets **channel spacing**. A regulator chooses spacing so that
each emission's OBW fits inside its channel with a [guard band](/reference/guard-band/) to spare, and
a spectrum mask bounds how far the skirts may extend beyond OBW before they count as
[spurious emissions](/reference/spurious-emissions/). Tightening OBW — through sharper filtering, a
smaller [roll-off factor](/reference/roll-off-factor/), or more efficient modulation — is what lets a
band be re-planned at narrower spacing (the VHF/UHF move from 25 kHz to 12.5 kHz "narrowbanding" is
exactly this), improving [spectral efficiency](/reference/spectral-efficiency/).

## Relevance to SDR

For an [SDR](/reference/software-defined-radio/), a signal's occupied bandwidth dictates how much
[bandwidth](/reference/bandwidth/) the receive chain must pass and how wide the channel filter should
be: too narrow clips the modulation sidebands and degrades the [demodulator](/reference/demodulation/);
too wide admits extra [noise](/reference/noise-floor/) and adjacent-channel energy. Measuring OBW off
a [spectrogram](/reference/spectrogram/) or [PSD](/reference/power-spectral-density/) estimate is a
quick way to identify an unknown emission and to set the right decimation and filter width before
decoding.

**GopherTrunk** sizes its per-channel filters and decimation from each protocol's known channel rate
and occupied bandwidth — for example, the ~12.5 kHz footprint of a P25 C4FM channel — so that the
channelizer passes the full modulation without dragging in neighbours. The concept is therefore
directly baked into how GopherTrunk carves channels out of a wideband capture.

## Sources

[^wiki]: [Bandwidth (signal processing)](https://en.wikipedia.org/wiki/Bandwidth_(signal_processing)) — Wikipedia, the 99%-power occupied-bandwidth convention.
[^nb]: [Necessary bandwidth](https://en.wikipedia.org/wiki/Necessary_bandwidth) — Wikipedia, necessary bandwidth and the ITU emission-designator scheme.
