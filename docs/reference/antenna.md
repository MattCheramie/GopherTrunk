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

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 150" role="img" aria-label="A vertical antenna element with concentric arcs radiating outward to represent transmitted or received waves." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="120" x2="150" y2="40" stroke="currentColor" stroke-width="2.5"/>
  <line x1="120" y1="120" x2="180" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <g fill="none" stroke="currentColor" stroke-opacity="0.5"><path d="M150 80 A 40 40 0 0 1 190 80"/><path d="M150 80 A 70 70 0 0 1 220 80"/><path d="M150 80 A 40 40 0 0 0 110 80"/><path d="M150 80 A 70 70 0 0 0 80 80"/></g>
  <text x="150" y="140" text-anchor="middle" font-size="9" fill="currentColor">converts between waves and current</text>
</svg>
<figcaption>An antenna couples radio waves to and from the receiver; its size follows the wavelength it works at.</figcaption>
</figure>

## How it works

An antenna works best when its dimensions are a fraction of the signal's
[wavelength](/reference/wavelength/) (a quarter-wave whip is λ/4). Its key properties
are resonance/[bandwidth](/reference/bandwidth/), [gain](/reference/antenna-gain/),
[polarization](/reference/polarization/), and impedance match
([SWR](/reference/standing-wave-ratio/)).

## Relevance to SDR

Choosing an antenna cut for the target [band](/reference/frequency-bands/) and placing
it high with a clear path usually improves reception more than any change at the radio.
