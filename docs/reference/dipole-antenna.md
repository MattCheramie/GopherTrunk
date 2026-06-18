---
slug: dipole-antenna
title: Dipole antenna
entry_type: term
category: antennas-propagation
description: A dipole is a simple resonant antenna of two conductors fed at the centre, typically a half wavelength long, with an omnidirectional pattern broadside to its axis.
keywords: dipole, half-wave dipole, resonant antenna, radiation pattern, balun
aka: [dipole antenna, dipole]
autolink: true
infobox:
  - { label: Type, value: Resonant antenna }
  - { label: Length, value: ~λ/2 (half-wave) }
  - { label: Pattern, value: Omnidirectional broadside }
see_also: [antenna, antenna-gain, polarization, wavelength]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
external:
  - { title: "Dipole antenna (Wikipedia)", url: https://en.wikipedia.org/wiki/Dipole_antenna }
---

A **dipole antenna** is a simple resonant [antenna](/reference/antenna/) made of two
conductors fed at the centre, classically a **half [wavelength](/reference/wavelength/)**
long. It is the reference against which other antennas' [gain](/reference/antenna-gain/)
is often compared.

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 150" role="img" aria-label="A centre-fed dipole with two quarter-wave rods and a doughnut-shaped radiation pattern around it." xmlns="http://www.w3.org/2000/svg">
  <line x1="160" y1="20" x2="160" y2="68" stroke="currentColor" stroke-width="3"/>
  <line x1="160" y1="82" x2="160" y2="130" stroke="currentColor" stroke-width="3"/>
  <circle cx="160" cy="75" r="3" fill="currentColor"/>
  <text x="172" y="48" font-size="10" fill="currentColor">λ/4</text><text x="172" y="112" font-size="10" fill="currentColor">λ/4</text>
  <ellipse cx="160" cy="75" rx="110" ry="28" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="4 3"/>
  <text x="250" y="75" font-size="9" fill="currentColor">pattern</text>
</svg>
<figcaption>A half-wave dipole is two quarter-wave rods fed in the middle, most sensitive broadside to its length.</figcaption>
</figure>

## How it works

A half-wave dipole radiates most strongly broadside (perpendicular to its axis) and
least off the ends, giving an omnidirectional doughnut pattern around a vertical
dipole. Its [polarization](/reference/polarization/) follows its orientation.

## Relevance to SDR

A dipole (or its grounded cousin, the quarter-wave whip) cut for the target band is a
cheap, effective receive antenna for scanning.
