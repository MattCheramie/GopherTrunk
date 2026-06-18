---
slug: polarization
title: Polarization
entry_type: term
category: antennas-propagation
description: Polarization is the orientation of a radio wave's electric field, set by the transmitting antenna; matching receive and transmit polarization avoids signal loss.
keywords: polarization, vertical, horizontal, circular, cross-polarization, electric field
aka: [polarization, polarisation]
autolink: true
infobox:
  - { label: Type, value: Wave/antenna property }
  - { label: Common kinds, value: Vertical, horizontal, circular }
  - { label: Mismatch loss, value: Up to ~20 dB }
see_also: [antenna, dipole-antenna, radio-propagation, antenna-gain]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
external:
  - { title: "Polarization (waves) (Wikipedia)", url: https://en.wikipedia.org/wiki/Polarization_(waves) }
---

**Polarization** is the orientation of a [radio wave](/reference/radio-wave/)'s
electric field, determined by how the transmitting [antenna](/reference/antenna/) is
mounted — vertical, horizontal, or circular.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A vertically oriented wave on the left and a horizontally oriented wave on the right, showing polarization." xmlns="http://www.w3.org/2000/svg">
  <line x1="110" y1="20" x2="110" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M110 20 q -28 25 0 50 t0 50" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="110" y="135" text-anchor="middle" font-size="9" fill="currentColor">vertical</text>
  <line x1="300" y1="70" x2="420" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M300 70 q 25 -28 50 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="125" text-anchor="middle" font-size="9" fill="currentColor">horizontal</text>
  <text x="230" y="70" text-anchor="middle" font-size="9" fill="currentColor">match the</text><text x="230" y="84" text-anchor="middle" font-size="9" fill="currentColor">transmitter</text>
</svg>
<figcaption>Polarization is the orientation of the wave's electric field; match it to the transmitter to avoid loss.</figcaption>
</figure>

## How it works

A receive antenna should match the transmitter's polarization; a full mismatch can
cost on the order of 20 dB. Most land-mobile and public-safety radio is vertically
polarized, while FM broadcast is often horizontal or circular.

## Relevance to SDR

A vertical antenna is the safe default for scanning land-mobile and trunked systems,
matching their vertical polarization.
