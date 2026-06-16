---
slug: radio-propagation
title: Radio propagation
entry_type: term
category: antennas-propagation
description: Radio propagation is the behaviour of radio waves as they travel from transmitter to receiver — line-of-sight, reflection, diffraction, and atmospheric effects.
keywords: radio propagation, line of sight, reflection, diffraction, fading, coverage
aka: [radio propagation, propagation]
autolink: true
infobox:
  - { label: Type, value: Wave-travel behaviour }
  - { label: VHF/UHF, value: Mostly line-of-sight }
  - { label: HF, value: Ionospheric skip possible }
see_also: [multipath-propagation, radio-horizon, ionospheric-propagation, path-loss, frequency-bands]
related_lessons:
  - { title: "How signals travel", url: /learn/propagation/ }
external:
  - { title: "Radio propagation (Wikipedia)", url: https://en.wikipedia.org/wiki/Radio_propagation }
---

**Radio propagation** describes how [radio waves](/reference/radio-wave/) travel from
transmitter to receiver, including line-of-sight travel, reflection, diffraction, and
atmospheric effects.

## How it works

At VHF and UHF, propagation is essentially line-of-sight, bounded by the
[radio horizon](/reference/radio-horizon/); reflections cause
[multipath](/reference/multipath-propagation/). At HF, the
[ionosphere](/reference/ionospheric-propagation/) can refract signals over great
distances. Loss along the way is [path loss](/reference/path-loss/).

## Relevance to SDR

Understanding propagation explains why antenna height and a clear path often matter
more than the radio, and why a distant hilltop system can beat a closer obstructed one.
