---
slug: antenna
title: Antenna
entry_type: term
category: antennas-propagation
description: An antenna is a conductor that converts electrical signals into radio waves and back; its dimensions, sized to the wavelength, set the reception of an SDR system.
keywords: antenna, aerial, radiator, resonance, reception
aka: [antenna, aerial]
autolink: true
infobox:
  - { label: Type, value: Transducer (RF ↔ current) }
  - { label: Sized to, value: Wavelength (e.g. λ/4, λ/2) }
  - { label: Key specs, value: Resonance, gain, polarization, SWR }
see_also: [dipole-antenna, antenna-gain, polarization, standing-wave-ratio, wavelength]
related_lessons:
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Antenna (radio) (Wikipedia)", url: https://en.wikipedia.org/wiki/Antenna_(radio) }
---

An **antenna** is a conductor that converts electrical signals into
[radio waves](/reference/radio-wave/) and, on receive, converts passing radio waves
back into a tiny current. It sets the ceiling on everything downstream — no receiver
can recover a signal the antenna never captured.

## How it works

An antenna works best when its dimensions are a fraction of the signal's
[wavelength](/reference/wavelength/) (a quarter-wave whip is λ/4). Its key properties
are resonance/[bandwidth](/reference/bandwidth/), [gain](/reference/antenna-gain/),
[polarization](/reference/polarization/), and impedance match
([SWR](/reference/standing-wave-ratio/)).

## Relevance to SDR

Choosing an antenna cut for the target [band](/reference/frequency-bands/) and placing
it high with a clear path usually improves reception more than any change at the radio.
