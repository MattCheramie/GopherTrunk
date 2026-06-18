---
slug: antenna-gain
title: Antenna gain
entry_type: term
category: antennas-propagation
description: Antenna gain is the degree to which an antenna concentrates radiated or received energy in a preferred direction, expressed in dBi or dBd.
keywords: antenna gain, dBi, dBd, directivity, radiation pattern, Yagi
aka: [antenna gain]
autolink: true
infobox:
  - { label: Type, value: Antenna property }
  - { label: Units, value: dBi (vs isotropic), dBd (vs dipole) }
  - { label: Trade-off, value: More gain = narrower pattern }
see_also: [antenna, dipole-antenna, decibel, radio-propagation]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
external:
  - { title: "Antenna gain (Wikipedia)", url: https://en.wikipedia.org/wiki/Antenna_gain }
---

**Antenna gain** measures how strongly an [antenna](/reference/antenna/) concentrates
energy in a preferred direction compared with a reference. It is given in
[decibels](/reference/decibel/): **dBi** relative to an isotropic radiator, or **dBd**
relative to a [dipole](/reference/dipole-antenna/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An omnidirectional circular pattern on the left and a focused directional lobe on the right." xmlns="http://www.w3.org/2000/svg">
  <circle cx="110" cy="75" r="45" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="110" cy="75" r="2.5" fill="currentColor"/>
  <text x="110" y="140" text-anchor="middle" font-size="9" fill="currentColor">omnidirectional</text>
  <path d="M330 75 C 330 35, 430 45, 440 75 C 430 105, 330 115, 330 75 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="330" cy="75" r="2.5" fill="currentColor"/>
  <text x="360" y="140" text-anchor="middle" font-size="9" fill="currentColor">directional (gain)</text>
</svg>
<figcaption>Antenna gain doesn't create energy — it focuses the pattern, trading all-round coverage for reach.</figcaption>
</figure>

## How it works

Gain does not create energy; it focuses it. A high-gain antenna trades coverage for
reach — an omnidirectional vertical hears all directions, while a directional Yagi adds
gain toward where it points at the expense of the sides.

## Relevance to SDR

For general scanning, an omnidirectional antenna is usually best; a directional,
high-gain antenna helps pull in one specific distant system.
