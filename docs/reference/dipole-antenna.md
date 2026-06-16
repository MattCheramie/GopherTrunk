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
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Dipole antenna (Wikipedia)", url: https://en.wikipedia.org/wiki/Dipole_antenna }
---

A **dipole antenna** is a simple resonant [antenna](/reference/antenna/) made of two
conductors fed at the centre, classically a **half [wavelength](/reference/wavelength/)**
long. It is the reference against which other antennas' [gain](/reference/antenna-gain/)
is often compared.

## How it works

A half-wave dipole radiates most strongly broadside (perpendicular to its axis) and
least off the ends, giving an omnidirectional doughnut pattern around a vertical
dipole. Its [polarization](/reference/polarization/) follows its orientation.

## Relevance to SDR

A dipole (or its grounded cousin, the quarter-wave whip) cut for the target band is a
cheap, effective receive antenna for scanning.
