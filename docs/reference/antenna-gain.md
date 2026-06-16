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
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Antenna gain (Wikipedia)", url: https://en.wikipedia.org/wiki/Antenna_gain }
---

**Antenna gain** measures how strongly an [antenna](/reference/antenna/) concentrates
energy in a preferred direction compared with a reference. It is given in
[decibels](/reference/decibel/): **dBi** relative to an isotropic radiator, or **dBd**
relative to a [dipole](/reference/dipole-antenna/).

## How it works

Gain does not create energy; it focuses it. A high-gain antenna trades coverage for
reach — an omnidirectional vertical hears all directions, while a directional Yagi adds
gain toward where it points at the expense of the sides.

## Relevance to SDR

For general scanning, an omnidirectional antenna is usually best; a directional,
high-gain antenna helps pull in one specific distant system.
