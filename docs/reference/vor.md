---
slug: vor
title: VHF Omnidirectional Range (VOR)
entry_type: technology
category: aviation-marine
description: VOR (VHF Omnidirectional Range) is an aviation navaid that lets aircraft read their bearing from a ground station by comparing the phase of two 30 Hz signals radiated in the VHF band.
keywords: VOR, VHF Omnidirectional Range, radio navigation, navaid, radial, bearing, phase comparison, 30 Hz reference, variable phase, 108-118 MHz, DVOR
aka: [VOR]
autolink: true
infobox:
  - { label: Type, value: Bearing navaid (radio navigation) }
  - { label: Idea, value: Phase difference between two 30 Hz signals = radial }
  - { label: Band, value: VHF 108–117.95 MHz }
see_also: [phase, amplitude-modulation, frequency-modulation, dme, ils]
cite_urls:
  - https://en.wikipedia.org/wiki/VHF_omnidirectional_range
  - https://www.icao.int/
---

**VOR** (**VHF Omnidirectional Range**) is a short-range radio navigation aid that lets
an aircraft determine its **magnetic bearing to (or from) a ground station** — its
"radial" — without any inertial reference. The station radiates two 30 Hz signals whose
relative [phase](/reference/phase/) encodes direction: one is a fixed reference, the other
varies with the compass angle at which the receiver sits. Measuring the phase difference
between them yields the bearing directly.[^wiki] Operating in the 108–117.95 MHz band,
VOR has been a backbone of airway navigation since the 1950s.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A ground station radiating radials in all directions, with an aircraft on the 060 radial reading its bearing from the phase difference of two 30 hertz signals." xmlns="http://www.w3.org/2000/svg">
  <circle cx="150" cy="90" r="8" fill="currentColor"/>
  <g stroke="currentColor" stroke-opacity="0.35">
    <line x1="150" y1="90" x2="150" y2="20"/><line x1="150" y1="90" x2="220" y2="90"/>
    <line x1="150" y1="90" x2="150" y2="160"/><line x1="150" y1="90" x2="80" y2="90"/>
    <line x1="150" y1="90" x2="200" y2="40"/><line x1="150" y1="90" x2="200" y2="140"/>
    <line x1="150" y1="90" x2="100" y2="140"/><line x1="150" y1="90" x2="100" y2="40"/>
  </g>
  <circle cx="150" cy="90" r="70" fill="none" stroke="currentColor" stroke-opacity="0.2"/>
  <path d="M200 40 l6 -3 l-1 7 z" fill="currentColor"/>
  <text x="205" y="34" font-size="8" fill="currentColor">aircraft on 060 radial</text>
  <text x="150" y="15" text-anchor="middle" font-size="8" fill="currentColor">N (000)</text>
  <g font-size="8" fill="currentColor">
    <text x="300" y="70">reference 30 Hz (FM subcarrier)</text>
    <text x="300" y="95">variable 30 Hz (rotating AM)</text>
    <text x="300" y="120">phase difference = radial</text>
  </g>
</svg>
<figcaption>A VOR beacon radiates a reference and a rotating signal; their phase difference tells the aircraft which radial it is on.</figcaption>
</figure>

## How it works

A conventional VOR combines two 30 Hz components. The **reference** phase is carried as
frequency modulation on a 9960 Hz subcarrier, so it is identical in every direction. The
**variable** phase is produced by a rotating antenna pattern (or its electronic
equivalent) that amplitude-modulates the carrier; because the lobe sweeps around the
compass, the phase a receiver sees depends on its bearing. At magnetic north the two are
in phase; at due east they differ by 90°. The receiver demodulates both, measures the
phase offset, and displays the radial. The signal uses conventional
[amplitude modulation](/reference/amplitude-modulation/) for the carrier and identity
tone, with the reference riding as an FM subcarrier.

The widely deployed **Doppler VOR (DVOR)** inverts which signal is AM and which is FM to
reduce siting errors from reflections, but the phase-comparison principle is identical.
Each station also transmits a Morse identifier so pilots can confirm they have tuned the
right beacon.

## Relevance to SDR

VOR is a classic teaching signal for software-defined radio: an
[SDR](/reference/software-defined-radio/) tuned to a local VOR can capture the AM
envelope, recover the 30 Hz variable tone, demodulate the 9960 Hz FM subcarrier for the
reference tone, and compute a bearing — a compact exercise in AM/FM demodulation and
phase estimation. VOR often shares a site and frequency pairing with
[DME](/reference/dme/) for distance, and it complements the
[ILS](/reference/ils/) localizer on approach. **GopherTrunk** is a land-mobile trunking
scanner and does not implement VOR bearing recovery; VOR is included here as RF context,
not as a GT decoder feature.

## Sources

[^wiki]: [VHF omnidirectional range](https://en.wikipedia.org/wiki/VHF_omnidirectional_range) — Wikipedia, for the VOR phase-comparison principle, the reference/variable 30 Hz signals, the 108–118 MHz band, and the Doppler VOR variant.
